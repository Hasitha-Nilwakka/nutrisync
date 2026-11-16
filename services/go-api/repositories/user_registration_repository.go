// Package repositories provides data access abstractions and GORM-backed persistence
// implementations for user registration and email verification workflows.
package repositories

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"nutrysync/go-api/models"
)

// UserRegistrationRepository defines the persistence operations required for
// registering new users and verifying their email addresses.
type UserRegistrationRepository interface {
	FindUserByEmailAddress(emailAddress string) (*models.User, error)
	CreateUnverifiedUserRecord(user *models.User) error
	RefreshVerificationToken(userID string, verificationToken string, expiresAt time.Time) error
	DeleteUnverifiedUserRecord(userID string) error
	FindUserByVerificationToken(verificationToken string) (*models.User, error)
	MarkUserEmailAsVerified(userID string, verifiedAt time.Time) error
}

// GormUserRegistrationRepository implements UserRegistrationRepository using a GORM *gorm.DB connection.
type GormUserRegistrationRepository struct {
	databaseConnection *gorm.DB
}

// NewGormUserRegistrationRepository constructs a GormUserRegistrationRepository backed by the provided *gorm.DB instance.
func NewGormUserRegistrationRepository(databaseConnection *gorm.DB) *GormUserRegistrationRepository {
	return &GormUserRegistrationRepository{
		databaseConnection: databaseConnection,
	}
}

// FindUserByEmailAddress looks up a single User record matching the provided email address.
// Returns the matching *models.User, or an error (such as gorm.ErrRecordNotFound) if no record is found or the query fails.
func (repository *GormUserRegistrationRepository) FindUserByEmailAddress(emailAddress string) (*models.User, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	if strings.TrimSpace(emailAddress) == "" {
		return nil, errors.New("email address must not be empty")
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(emailAddress))
	var userRecord models.User
	queryResult := repository.databaseConnection.Where("LOWER(email) = ?", normalizedEmail).First(&userRecord)
	if queryResult.Error != nil {
		return nil, queryResult.Error
	}

	return &userRecord, nil
}

// CreateUnverifiedUserRecord inserts a new unverified User record into the database.
// Enforces that Verified is false prior to insertion and returns an error if the insert fails.
func (repository *GormUserRegistrationRepository) CreateUnverifiedUserRecord(user *models.User) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	if user == nil {
		return errors.New("user record must not be nil")
	}

	// Ensure newly registered user accounts start in an unverified and unconsented state before persisting.
	user.Verified = false
	user.Consented = false

	creationResult := repository.databaseConnection.Create(user)
	return creationResult.Error
}

// RefreshVerificationToken updates the verification token and expiration timestamp for an unverified user.
func (repository *GormUserRegistrationRepository) RefreshVerificationToken(
	userID string,
	verificationToken string,
	expiresAt time.Time,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	trimmedToken := strings.TrimSpace(verificationToken)
	if trimmedUserID == "" || trimmedToken == "" {
		return errors.New("user ID and verification token must not be empty")
	}

	return repository.databaseConnection.
		Model(&models.User{}).
		Where("id = ? AND verified = ?", trimmedUserID, false).
		Updates(map[string]any{
			"verification_token":            trimmedToken,
			"verification_token_expired_at": expiresAt.UTC(),
		}).Error
}

// DeleteUnverifiedUserRecord deletes a newly created unverified user record (used for cleanup when verification email delivery fails).
func (repository *GormUserRegistrationRepository) DeleteUnverifiedUserRecord(userID string) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return errors.New("user ID must not be empty")
	}

	return repository.databaseConnection.
		Where("id = ? AND verified = ?", trimmedUserID, false).
		Delete(&models.User{}).Error
}

// FindUserByVerificationToken looks up a single User record associated with the given email verification token.
// Returns the matching *models.User, or an error (such as gorm.ErrRecordNotFound) if no record matches or the query fails.
func (repository *GormUserRegistrationRepository) FindUserByVerificationToken(verificationToken string) (*models.User, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	if strings.TrimSpace(verificationToken) == "" {
		return nil, errors.New("verification token must not be empty")
	}

	var userRecord models.User
	queryResult := repository.databaseConnection.Where("verification_token = ?", verificationToken).First(&userRecord)
	if queryResult.Error != nil {
		return nil, queryResult.Error
	}

	return &userRecord, nil
}

// MarkUserEmailAsVerified updates the user record identified by userID to mark their email as verified,
// sets VerifiedAt to the provided timestamp, and clears VerificationToken and VerificationTokenExpiredAt.
func (repository *GormUserRegistrationRepository) MarkUserEmailAsVerified(userID string, verifiedAt time.Time) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	if strings.TrimSpace(userID) == "" {
		return errors.New("user ID must not be empty")
	}

	// Using a map ensures GORM updates nullable pointer columns to NULL and boolean columns reliably.
	verificationUpdates := map[string]any{
		"verified":                      true,
		"verified_at":                   verifiedAt,
		"verification_token":            nil,
		"verification_token_expired_at": nil,
	}

	updateResult := repository.databaseConnection.Model(&models.User{}).Where("id = ?", userID).Updates(verificationUpdates)
	if updateResult.Error != nil {
		return updateResult.Error
	}

	return nil
}
