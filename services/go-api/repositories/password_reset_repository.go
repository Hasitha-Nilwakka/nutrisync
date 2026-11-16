// This file defines the PasswordResetRepository interface and its GORM PostgreSQL implementation
// for creating, looking up, and completing password reset tokens and revoking active sessions.
package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"nutrysync/go-api/models"
)

// PasswordResetRepository defines persistence operations required by the password reset flow.
type PasswordResetRepository interface {
	FindUserByEmailForPasswordReset(ctx context.Context, emailAddress string) (*models.User, error)
	ReplaceActivePasswordResetToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error
	FindPasswordResetByTokenHash(ctx context.Context, tokenHash string) (*models.PasswordReset, error)
	CompletePasswordResetAndRevokeSessions(ctx context.Context, resetID string, userID string, newPasswordHash string, completedAt time.Time) error
}

// GormPasswordResetRepository implements PasswordResetRepository using GORM and PostgreSQL.
type GormPasswordResetRepository struct {
	database *gorm.DB
}

// NewGormPasswordResetRepository creates a GormPasswordResetRepository backed by databaseConnection.
func NewGormPasswordResetRepository(databaseConnection *gorm.DB) (*GormPasswordResetRepository, error) {
	if databaseConnection == nil {
		return nil, errors.New("database connection is required")
	}

	return &GormPasswordResetRepository{database: databaseConnection}, nil
}

// FindUserByEmailForPasswordReset looks up a user by normalized lowercase email, returning (nil, nil) if not found.
func (repository *GormPasswordResetRepository) FindUserByEmailForPasswordReset(
	ctx context.Context,
	emailAddress string,
) (*models.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(emailAddress))
	if normalizedEmail == "" {
		return nil, nil
	}

	var matchedUser models.User
	queryError := repository.database.WithContext(ctx).
		Where("LOWER(email) = ?", normalizedEmail).
		First(&matchedUser).Error
	if errors.Is(queryError, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if queryError != nil {
		return nil, queryError
	}

	return &matchedUser, nil
}

// ReplaceActivePasswordResetToken revokes any prior unverified password reset tokens for userID
// and inserts a new PasswordReset record inside a single transaction.
func (repository *GormPasswordResetRepository) ReplaceActivePasswordResetToken(
	ctx context.Context,
	userID string,
	tokenHash string,
	expiresAt time.Time,
) error {
	trimmedUserID := strings.TrimSpace(userID)
	trimmedTokenHash := strings.TrimSpace(tokenHash)
	if trimmedUserID == "" || trimmedTokenHash == "" {
		return errors.New("userID and tokenHash are required")
	}

	nowUTC := time.Now().UTC()
	return repository.database.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		if revokeError := transactionDB.Model(&models.PasswordReset{}).
			Where("user_id = ? AND verified = ? AND revoked_at IS NULL", trimmedUserID, false).
			Update("revoked_at", nowUTC).Error; revokeError != nil {
			return revokeError
		}

		newResetRecord := &models.PasswordReset{
			UserID:                  trimmedUserID,
			VerificationToken:       trimmedTokenHash,
			VerificationTokenExpiry: expiresAt.UTC(),
			Verified:                false,
		}

		return transactionDB.Create(newResetRecord).Error
	})
}

// FindPasswordResetByTokenHash fetches a PasswordReset record by its SHA-256 token hash, returning (nil, nil) if not found.
func (repository *GormPasswordResetRepository) FindPasswordResetByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*models.PasswordReset, error) {
	trimmedTokenHash := strings.TrimSpace(tokenHash)
	if trimmedTokenHash == "" {
		return nil, nil
	}

	var resetRecord models.PasswordReset
	queryError := repository.database.WithContext(ctx).
		Where("verification_token = ?", trimmedTokenHash).
		First(&resetRecord).Error
	if errors.Is(queryError, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if queryError != nil {
		return nil, queryError
	}

	return &resetRecord, nil
}

// CompletePasswordResetAndRevokeSessions atomically updates the user's password hash, marks the reset token verified/revoked,
// and deletes all active refresh tokens and sessions belonging to userID.
func (repository *GormPasswordResetRepository) CompletePasswordResetAndRevokeSessions(
	ctx context.Context,
	resetID string,
	userID string,
	newPasswordHash string,
	completedAt time.Time,
) error {
	trimmedResetID := strings.TrimSpace(resetID)
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedResetID == "" || trimmedUserID == "" || strings.TrimSpace(newPasswordHash) == "" {
		return errors.New("resetID, userID, and newPasswordHash are required")
	}

	completedUTC := completedAt.UTC()
	return repository.database.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		if updateUserError := transactionDB.Model(&models.User{}).
			Where("id = ?", trimmedUserID).
			Update("password", newPasswordHash).Error; updateUserError != nil {
			return updateUserError
		}

		if updateResetError := transactionDB.Model(&models.PasswordReset{}).
			Where("id = ?", trimmedResetID).
			Updates(map[string]any{
				"verified":   true,
				"revoked_at": completedUTC,
			}).Error; updateResetError != nil {
			return updateResetError
		}

		sessionIDSubQuery := transactionDB.Model(&models.Session{}).
			Select("id").
			Where("user_id = ?", trimmedUserID)
		if deleteTokensError := transactionDB.
			Where("session_id IN (?)", sessionIDSubQuery).
			Delete(&models.RefreshToken{}).Error; deleteTokensError != nil {
			return deleteTokensError
		}

		return transactionDB.
			Where("user_id = ?", trimmedUserID).
			Delete(&models.Session{}).Error
	})
}
