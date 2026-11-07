// This file defines the AccountRepository interface and its GORM-backed implementation
// for federated OAuth identity lookup, new OAuth user + account creation, and linking OAuth accounts to existing users.
package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"nutrysync/go-api/models"
)

// AccountRepository defines the persistence operations required for OAuth authentication and federated account linking.
type AccountRepository interface {
	FindAccountByProviderAndID(ctx context.Context, provider, providerUserID string) (*models.Account, error)
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	CreateOAuthUserAndAccount(ctx context.Context, user *models.User, account *models.Account) error
	LinkAccountToExistingUser(ctx context.Context, account *models.Account) error
}

// GormAccountRepository implements AccountRepository using a GORM *gorm.DB connection.
type GormAccountRepository struct {
	databaseConnection *gorm.DB
}

// NewGormAccountRepository constructs a GormAccountRepository after verifying databaseConnection is non-nil.
func NewGormAccountRepository(databaseConnection *gorm.DB) (*GormAccountRepository, error) {
	if databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	return &GormAccountRepository{
		databaseConnection: databaseConnection,
	}, nil
}

// FindAccountByProviderAndID looks up an Account by provider name and providerUserID, preloading its associated User.
// Returns (nil, nil) when gorm.ErrRecordNotFound occurs.
func (repository *GormAccountRepository) FindAccountByProviderAndID(
	ctx context.Context,
	provider, providerUserID string,
) (*models.Account, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	trimmedProvider := strings.TrimSpace(provider)
	trimmedProviderUserID := strings.TrimSpace(providerUserID)
	if trimmedProvider == "" || trimmedProviderUserID == "" {
		return nil, errors.New("provider and provider user ID must not be empty")
	}

	var accountRecord models.Account
	queryResult := repository.databaseConnection.WithContext(ctx).
		Preload("User").
		Where("provider = ? AND provider_user_id = ?", trimmedProvider, trimmedProviderUserID).
		First(&accountRecord)
	if queryResult.Error != nil {
		if errors.Is(queryResult.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, queryResult.Error
	}

	return &accountRecord, nil
}

// FindUserByEmail retrieves a User record by email address for federated account matching.
// Returns (nil, nil) when gorm.ErrRecordNotFound occurs.
func (repository *GormAccountRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" {
		return nil, errors.New("email must not be empty")
	}

	var userRecord models.User
	queryResult := repository.databaseConnection.WithContext(ctx).
		Where("email = ?", trimmedEmail).
		First(&userRecord)
	if queryResult.Error != nil {
		if errors.Is(queryResult.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, queryResult.Error
	}

	return &userRecord, nil
}

// CreateOAuthUserAndAccount atomically inserts a new OAuth User and its linked Account inside a single GORM transaction,
// assigning account.UserID = user.ID after creating the user row.
func (repository *GormAccountRepository) CreateOAuthUserAndAccount(
	ctx context.Context,
	user *models.User,
	account *models.Account,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	if user == nil {
		return errors.New("user must not be nil")
	}

	if account == nil {
		return errors.New("account must not be nil")
	}

	return repository.databaseConnection.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		if userCreateError := transactionDB.Create(user).Error; userCreateError != nil {
			return userCreateError
		}

		account.UserID = user.ID
		if accountCreateError := transactionDB.Create(account).Error; accountCreateError != nil {
			return accountCreateError
		}

		return nil
	})
}

// LinkAccountToExistingUser atomically inserts a new OAuth Account linked to an existing User
// and marks the user's email as verified if not already verified.
func (repository *GormAccountRepository) LinkAccountToExistingUser(
	ctx context.Context,
	account *models.Account,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	if account == nil || strings.TrimSpace(account.UserID) == "" {
		return errors.New("account and account.UserID must not be empty")
	}

	return repository.databaseConnection.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		if accountCreateError := transactionDB.Create(account).Error; accountCreateError != nil {
			return accountCreateError
		}

		verifiedTimestamp := time.Now().UTC()
		verificationUpdates := map[string]any{
			"verified":                      true,
			"verified_at":                   verifiedTimestamp,
			"verification_token":            nil,
			"verification_token_expired_at": nil,
		}

		return transactionDB.Model(&models.User{}).
			Where("id = ?", account.UserID).
			Updates(verificationUpdates).Error
	})
}
