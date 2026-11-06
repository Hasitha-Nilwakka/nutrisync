// This file defines the UserSessionRepository interface and its GORM-backed implementation
// for user login lookup, transactional session creation, refresh token rotation, and session revocation.
package repositories

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"nutrysync/go-api/models"
)

// UserSessionRepository defines the persistence operations required for user login,
// refresh token rotation, reuse detection, and session revocation.
type UserSessionRepository interface {
	FindUserByEmailForLogin(ctx context.Context, email string) (*models.User, error)
	CreateSessionWithInitialRefreshToken(ctx context.Context, session *models.Session, initialRefreshToken *models.RefreshToken) error
	FindRefreshTokenWithSessionByID(ctx context.Context, refreshTokenID string) (*models.RefreshToken, error)
	RotateRefreshTokenInTransaction(ctx context.Context, currentTokenID string, newRefreshToken *models.RefreshToken) error
	RevokeSessionAndAllRefreshTokens(ctx context.Context, sessionID string) error
	FindSessionByID(ctx context.Context, sessionID string) (*models.Session, error)
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
		if createError := transactionDB.Create(newRefreshToken).Error; createError != nil {
			return createError
		}

		rotationUpdates := map[string]any{
			"revoked":              true,
			"replaced_by_token_id": newRefreshToken.ID,
		}

		updateResult := transactionDB.Model(&models.RefreshToken{}).
			Where("id = ?", trimmedCurrentTokenID).
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
