// This file defines the UserSessionRepository interface and its GORM-backed implementation
// for user login lookup, transactional session creation, refresh token rotation, and session revocation.
package repositories

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"nutrysync/go-api/models"
)

// ErrRefreshTokenAlreadyRotated indicates a refresh token was already revoked or rotated by a concurrent/prior request.
var ErrRefreshTokenAlreadyRotated = errors.New("refresh token already rotated or revoked")

// UserSessionRepository defines the persistence operations required for user login,
// refresh token rotation, reuse detection, session revocation, and GDPR consent management.
type UserSessionRepository interface {
	FindUserByEmailForLogin(ctx context.Context, email string) (*models.User, error)
	CreateSessionWithInitialRefreshToken(ctx context.Context, session *models.Session, initialRefreshToken *models.RefreshToken) error
	FindRefreshTokenWithSessionByID(ctx context.Context, refreshTokenID string) (*models.RefreshToken, error)
	RotateRefreshTokenInTransaction(ctx context.Context, currentTokenID string, newRefreshToken *models.RefreshToken) error
	RevokeSessionAndAllRefreshTokens(ctx context.Context, sessionID string) error
	FindSessionByID(ctx context.Context, sessionID string) (*models.Session, error)
	AcceptUserGDPRConsent(ctx context.Context, userID string) error
	DeleteUnconsentedUserAndAccounts(ctx context.Context, userID string) error
	DeleteAuthenticatedUserAccountAndAllData(ctx context.Context, userID string) error
}

// TwoFactorRepository defines the persistence operations required for configuring, enabling,
// disabling, and verifying optional TOTP two-factor authentication on a user account.
type TwoFactorRepository interface {
	SetPendingTwoFactorSecret(ctx context.Context, userID, secret string) error
	EnableTwoFactorAuthentication(ctx context.Context, userID string) error
	DisableTwoFactorAuthentication(ctx context.Context, userID string) error
	FindUserByID(ctx context.Context, userID string) (*models.User, error)
}

// GormUserSessionRepository implements UserSessionRepository using a GORM *gorm.DB connection.
type GormUserSessionRepository struct {
	databaseConnection *gorm.DB
}

// NewGormUserSessionRepository constructs a GormUserSessionRepository after verifying databaseConnection is non-nil.
func NewGormUserSessionRepository(databaseConnection *gorm.DB) (*GormUserSessionRepository, error) {
	if databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	return &GormUserSessionRepository{
		databaseConnection: databaseConnection,
	}, nil
}

// FindUserByEmailForLogin retrieves a User record by email address for login authentication.
// Returns (nil, nil) when gorm.ErrRecordNotFound occurs so callers can distinguish unknown emails from database errors.
func (repository *GormUserSessionRepository) FindUserByEmailForLogin(ctx context.Context, email string) (*models.User, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" {
		return nil, errors.New("email must not be empty")
	}

	var userRecord models.User
	queryResult := repository.databaseConnection.WithContext(ctx).Where("email = ?", trimmedEmail).First(&userRecord)
	if queryResult.Error != nil {
		if errors.Is(queryResult.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, queryResult.Error
	}

	return &userRecord, nil
}

// CreateSessionWithInitialRefreshToken inserts a new Session and its initial RefreshToken inside a single GORM transaction,
// assigning initialRefreshToken.SessionID = session.ID once the session record is created.
func (repository *GormUserSessionRepository) CreateSessionWithInitialRefreshToken(
	ctx context.Context,
	session *models.Session,
	initialRefreshToken *models.RefreshToken,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	if session == nil {
		return errors.New("session must not be nil")
	}

	if initialRefreshToken == nil {
		return errors.New("initial refresh token must not be nil")
	}

	return repository.databaseConnection.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		if sessionCreateError := transactionDB.Create(session).Error; sessionCreateError != nil {
			return sessionCreateError
		}

		initialRefreshToken.SessionID = session.ID
		if tokenCreateError := transactionDB.Create(initialRefreshToken).Error; tokenCreateError != nil {
			return tokenCreateError
		}

		return nil
	})
}

// FindRefreshTokenWithSessionByID retrieves a RefreshToken by its primary key and preloads its parent Session.
// Returns (nil, nil) when gorm.ErrRecordNotFound occurs.
func (repository *GormUserSessionRepository) FindRefreshTokenWithSessionByID(
	ctx context.Context,
	refreshTokenID string,
) (*models.RefreshToken, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	trimmedTokenID := strings.TrimSpace(refreshTokenID)
	if trimmedTokenID == "" {
		return nil, errors.New("refresh token ID must not be empty")
	}

	var refreshTokenRecord models.RefreshToken
	queryResult := repository.databaseConnection.WithContext(ctx).
		Preload("Session").
		Where("id = ?", trimmedTokenID).
		First(&refreshTokenRecord)
	if queryResult.Error != nil {
		if errors.Is(queryResult.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, queryResult.Error
	}

	return &refreshTokenRecord, nil
}

// RotateRefreshTokenInTransaction atomically inserts newRefreshToken and marks currentTokenID
// as revoked with replaced_by_token_id set to newRefreshToken.ID.
func (repository *GormUserSessionRepository) RotateRefreshTokenInTransaction(
	ctx context.Context,
	currentTokenID string,
	newRefreshToken *models.RefreshToken,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	trimmedCurrentTokenID := strings.TrimSpace(currentTokenID)
	if trimmedCurrentTokenID == "" {
		return errors.New("current token ID must not be empty")
	}

	if newRefreshToken == nil {
		return errors.New("new refresh token must not be nil")
	}

	return repository.databaseConnection.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		var lockedToken models.RefreshToken
		if lockError := transactionDB.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", trimmedCurrentTokenID).
			First(&lockedToken).Error; lockError != nil {
			return lockError
		}

		if lockedToken.Revoked || lockedToken.ReplacedByTokenID != nil {
			return ErrRefreshTokenAlreadyRotated
		}

		if createError := transactionDB.Create(newRefreshToken).Error; createError != nil {
			return createError
		}

		rotationUpdates := map[string]any{
			"revoked":              true,
			"replaced_by_token_id": newRefreshToken.ID,
		}

		updateResult := transactionDB.Model(&models.RefreshToken{}).
			Where("id = ? AND revoked = ?", trimmedCurrentTokenID, false).
			Updates(rotationUpdates)
		return updateResult.Error
	})
}

// RevokeSessionAndAllRefreshTokens atomically sets revoked = true on the Session identified by sessionID
// and on all RefreshToken rows belonging to that session.
func (repository *GormUserSessionRepository) RevokeSessionAndAllRefreshTokens(
	ctx context.Context,
	sessionID string,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return errors.New("session ID must not be empty")
	}

	return repository.databaseConnection.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		if sessionUpdateError := transactionDB.Model(&models.Session{}).
			Where("id = ?", trimmedSessionID).
			Update("revoked", true).Error; sessionUpdateError != nil {
			return sessionUpdateError
		}

		if tokensUpdateError := transactionDB.Model(&models.RefreshToken{}).
			Where("session_id = ?", trimmedSessionID).
			Update("revoked", true).Error; tokensUpdateError != nil {
			return tokensUpdateError
		}

		return nil
	})
}

// FindSessionByID retrieves a Session record by its primary key ID.
// Returns (nil, nil) when gorm.ErrRecordNotFound occurs so callers can treat missing sessions as unauthenticated.
func (repository *GormUserSessionRepository) FindSessionByID(
	ctx context.Context,
	sessionID string,
) (*models.Session, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return nil, errors.New("session ID must not be empty")
	}

	var sessionRecord models.Session
	queryResult := repository.databaseConnection.WithContext(ctx).
		Preload("User").
		Preload("User.HealthProfile").
		Where("id = ?", trimmedSessionID).
		First(&sessionRecord)
	if queryResult.Error != nil {
		if errors.Is(queryResult.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, queryResult.Error
	}

	return &sessionRecord, nil
}

// AcceptUserGDPRConsent sets consented = true on the User record identified by userID.
func (repository *GormUserSessionRepository) AcceptUserGDPRConsent(
	ctx context.Context,
	userID string,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return errors.New("user ID must not be empty")
	}

	return repository.databaseConnection.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", trimmedUserID).
		Update("consented", true).Error
}

// DeleteUnconsentedUserAndAccounts verifies inside a GORM transaction that the user currently has consented == false,
// and deletes all associated Account, RefreshToken, Session, and User records for userID.
func (repository *GormUserSessionRepository) DeleteUnconsentedUserAndAccounts(
	ctx context.Context,
	userID string,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return errors.New("user ID must not be empty")
	}

	return repository.databaseConnection.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		var userRecord models.User
		if lookupError := transactionDB.Where("id = ?", trimmedUserID).First(&userRecord).Error; lookupError != nil {
			return lookupError
		}

		if userRecord.Consented {
			return errors.New("cannot delete user via unconsented flow after consent has already been granted")
		}

		if deleteAccountsErr := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.Account{}).Error; deleteAccountsErr != nil {
			return deleteAccountsErr
		}

		sessionIDSubQuery := transactionDB.Model(&models.Session{}).Select("id").Where("user_id = ?", trimmedUserID)
		if deleteTokensErr := transactionDB.Where("session_id IN (?)", sessionIDSubQuery).Delete(&models.RefreshToken{}).Error; deleteTokensErr != nil {
			return deleteTokensErr
		}

		if deleteSessionsErr := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.Session{}).Error; deleteSessionsErr != nil {
			return deleteSessionsErr
		}

		if deletePasswordResetsErr := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.PasswordReset{}).Error; deletePasswordResetsErr != nil {
			return deletePasswordResetsErr
		}

		if deleteUserErr := transactionDB.Where("id = ?", trimmedUserID).Delete(&models.User{}).Error; deleteUserErr != nil {
			return deleteUserErr
		}

		return nil
	})
}

// DeleteAuthenticatedUserAccountAndAllData permanently deletes an authenticated user's health records,
// AI insights, OAuth accounts, sessions, refresh tokens, password resets, and user record in a transaction.
func (repository *GormUserSessionRepository) DeleteAuthenticatedUserAccountAndAllData(
	ctx context.Context,
	userID string,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return errors.New("user ID must not be empty")
	}

	return repository.databaseConnection.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		if err := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.AIInsightRecord{}).Error; err != nil {
			return err
		}
		if err := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.WellnessSnapshot{}).Error; err != nil {
			return err
		}
		if err := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.ActivityEntry{}).Error; err != nil {
			return err
		}
		if err := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.WeightEntry{}).Error; err != nil {
			return err
		}
		if err := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.HealthProfile{}).Error; err != nil {
			return err
		}
		if err := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.Account{}).Error; err != nil {
			return err
		}

		sessionIDSubQuery := transactionDB.Model(&models.Session{}).Select("id").Where("user_id = ?", trimmedUserID)
		if err := transactionDB.Where("session_id IN (?)", sessionIDSubQuery).Delete(&models.RefreshToken{}).Error; err != nil {
			return err
		}
		if err := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.Session{}).Error; err != nil {
			return err
		}
		if err := transactionDB.Where("user_id = ?", trimmedUserID).Delete(&models.PasswordReset{}).Error; err != nil {
			return err
		}

		return transactionDB.Where("id = ?", trimmedUserID).Delete(&models.User{}).Error
	})
}

// SetPendingTwoFactorSecret stores a newly generated TOTP secret in two_factor_pending_secret for the given userID.
func (repository *GormUserSessionRepository) SetPendingTwoFactorSecret(
	ctx context.Context,
	userID, secret string,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return errors.New("user ID must not be empty")
	}

	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret == "" {
		return errors.New("two-factor pending secret must not be empty")
	}

	return repository.databaseConnection.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", trimmedUserID).
		Update("two_factor_pending_secret", trimmedSecret).Error
}

// EnableTwoFactorAuthentication promotes two_factor_pending_secret to two_factor_secret, clears two_factor_pending_secret,
// sets two_factor_enabled = true, and sets two_factor_provider = "authenticator" for the given userID.
func (repository *GormUserSessionRepository) EnableTwoFactorAuthentication(
	ctx context.Context,
	userID string,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return errors.New("user ID must not be empty")
	}

	updates := map[string]any{
		"two_factor_secret":         gorm.Expr("two_factor_pending_secret"),
		"two_factor_pending_secret": nil,
		"two_factor_enabled":        true,
		"two_factor_provider":       "authenticator",
	}

	return repository.databaseConnection.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", trimmedUserID).
		Updates(updates).Error
}

// DisableTwoFactorAuthentication disables 2FA and clears two_factor_secret, two_factor_pending_secret,
// and two_factor_provider for the given userID.
func (repository *GormUserSessionRepository) DisableTwoFactorAuthentication(
	ctx context.Context,
	userID string,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return errors.New("user ID must not be empty")
	}

	updates := map[string]any{
		"two_factor_enabled":        false,
		"two_factor_secret":         nil,
		"two_factor_pending_secret": nil,
		"two_factor_provider":       nil,
	}

	return repository.databaseConnection.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", trimmedUserID).
		Updates(updates).Error
}

// FindUserByID retrieves a User record by its primary key ID.
// Returns (nil, nil) when gorm.ErrRecordNotFound occurs.
func (repository *GormUserSessionRepository) FindUserByID(
	ctx context.Context,
	userID string,
) (*models.User, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return nil, errors.New("user ID must not be empty")
	}

	var userRecord models.User
	queryResult := repository.databaseConnection.WithContext(ctx).
		Where("id = ?", trimmedUserID).
		First(&userRecord)
	if queryResult.Error != nil {
		if errors.Is(queryResult.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, queryResult.Error
	}

	return &userRecord, nil
}
