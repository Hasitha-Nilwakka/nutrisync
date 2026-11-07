// This file contains unit tests for GormUserSessionRepository and all UserSessionRepository methods.
package repositories

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"nutrysync/go-api/models"
)

// transactionalDryRunConnectionPool extends dryRunConnectionPool with GORM transaction beginner/committer
// methods so GORM Transaction(...) blocks execute cleanly in DryRun unit tests.
type transactionalDryRunConnectionPool struct {
	dryRunConnectionPool
}

func (pool *transactionalDryRunConnectionPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	return pool, nil
}

func (*transactionalDryRunConnectionPool) Commit() error {
	return nil
}

func (*transactionalDryRunConnectionPool) Rollback() error {
	return nil
}

func createTransactionalDryRunGormDatabase(testingInstance *testing.T) *gorm.DB {
	testingInstance.Helper()

	databaseConnection, openError := gorm.Open(
		postgres.New(postgres.Config{Conn: &transactionalDryRunConnectionPool{}}),
		&gorm.Config{DryRun: true},
	)
	if openError != nil {
		testingInstance.Fatalf("failed to initialize transactional dry-run GORM database: %v", openError)
	}

	return databaseConnection
}

// TestNewGormUserSessionRepositoryValidatesDatabaseConnection verifies constructor validation and interface compliance.
func TestNewGormUserSessionRepositoryValidatesDatabaseConnection(testingInstance *testing.T) {
	if _, err := NewGormUserSessionRepository(nil); err == nil {
		testingInstance.Fatal("expected error when initializing NewGormUserSessionRepository with nil *gorm.DB")
	}

	dryRunDB := createTransactionalDryRunGormDatabase(testingInstance)
	repository, initError := NewGormUserSessionRepository(dryRunDB)
	if initError != nil {
		testingInstance.Fatalf("expected NewGormUserSessionRepository to succeed, got: %v", initError)
	}

	var repositoryInterface UserSessionRepository = repository
	if repositoryInterface == nil {
		testingInstance.Fatal("expected non-nil UserSessionRepository")
	}
}

// TestGormUserSessionRepositoryMethodsValidationAndDryRunExecution verifies FindUserByEmailForLogin,
// CreateSessionWithInitialRefreshToken, FindRefreshTokenWithSessionByID, RotateRefreshTokenInTransaction,
// and RevokeSessionAndAllRefreshTokens.
func TestGormUserSessionRepositoryMethodsValidationAndDryRunExecution(testingInstance *testing.T) {
	ctx := context.Background()
	dryRunDB := createTransactionalDryRunGormDatabase(testingInstance)
	repository, _ := NewGormUserSessionRepository(dryRunDB)

	if _, err := repository.FindUserByEmailForLogin(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when email is empty in FindUserByEmailForLogin")
	}

	userRecord, loginLookupErr := repository.FindUserByEmailForLogin(ctx, "login@example.com")
	if loginLookupErr != nil || userRecord == nil {
		testingInstance.Fatalf("expected dry-run FindUserByEmailForLogin to succeed, got user=%+v err=%v", userRecord, loginLookupErr)
	}

	if err := repository.CreateSessionWithInitialRefreshToken(ctx, nil, &models.RefreshToken{}); err == nil {
		testingInstance.Fatal("expected error when session is nil in CreateSessionWithInitialRefreshToken")
	}

	if err := repository.CreateSessionWithInitialRefreshToken(ctx, &models.Session{}, nil); err == nil {
		testingInstance.Fatal("expected error when initialRefreshToken is nil in CreateSessionWithInitialRefreshToken")
	}

	sessionRecord := &models.Session{
		ID:        "session-uuid-1",
		UserID:    "user-uuid-1",
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	initialTokenRecord := &models.RefreshToken{
		ID:        "token-uuid-1",
		TokenHash: "hash-1",
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}

	if createErr := repository.CreateSessionWithInitialRefreshToken(ctx, sessionRecord, initialTokenRecord); createErr != nil {
		testingInstance.Fatalf("expected CreateSessionWithInitialRefreshToken to succeed in dry-run, got: %v", createErr)
	}

	if initialTokenRecord.SessionID != sessionRecord.ID {
		testingInstance.Errorf("expected initialRefreshToken.SessionID to be %q, got %q", sessionRecord.ID, initialTokenRecord.SessionID)
	}

	if _, err := repository.FindRefreshTokenWithSessionByID(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when refreshTokenID is empty in FindRefreshTokenWithSessionByID")
	}

	foundToken, findTokenErr := repository.FindRefreshTokenWithSessionByID(ctx, "token-uuid-1")
	if findTokenErr != nil || foundToken == nil {
		testingInstance.Fatalf("expected dry-run FindRefreshTokenWithSessionByID to succeed, got token=%+v err=%v", foundToken, findTokenErr)
	}

	if err := repository.RotateRefreshTokenInTransaction(ctx, "", &models.RefreshToken{}); err == nil {
		testingInstance.Fatal("expected error when currentTokenID is empty in RotateRefreshTokenInTransaction")
	}

	if err := repository.RotateRefreshTokenInTransaction(ctx, "token-uuid-1", nil); err == nil {
		testingInstance.Fatal("expected error when newRefreshToken is nil in RotateRefreshTokenInTransaction")
	}

	replacementToken := &models.RefreshToken{
		ID:        "token-uuid-2",
		SessionID: sessionRecord.ID,
		TokenHash: "hash-2",
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}

	if rotateErr := repository.RotateRefreshTokenInTransaction(ctx, "token-uuid-1", replacementToken); rotateErr != nil {
		testingInstance.Fatalf("expected RotateRefreshTokenInTransaction to succeed in dry-run, got: %v", rotateErr)
	}

	if err := repository.RevokeSessionAndAllRefreshTokens(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when sessionID is empty in RevokeSessionAndAllRefreshTokens")
	}

	if revokeErr := repository.RevokeSessionAndAllRefreshTokens(ctx, sessionRecord.ID); revokeErr != nil {
		testingInstance.Fatalf("expected RevokeSessionAndAllRefreshTokens to succeed in dry-run, got: %v", revokeErr)
	}

	if _, err := repository.FindSessionByID(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when sessionID is empty in FindSessionByID")
	}

	foundSession, findSessionErr := repository.FindSessionByID(ctx, sessionRecord.ID)
	if findSessionErr != nil || foundSession == nil {
		testingInstance.Fatalf("expected dry-run FindSessionByID to succeed, got session=%+v err=%v", foundSession, findSessionErr)
	}

	if err := repository.AcceptUserGDPRConsent(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in AcceptUserGDPRConsent")
	}

	if consentErr := repository.AcceptUserGDPRConsent(ctx, "user-uuid-1"); consentErr != nil {
		testingInstance.Fatalf("expected AcceptUserGDPRConsent to succeed in dry-run, got: %v", consentErr)
	}

	if err := repository.DeleteUnconsentedUserAndAccounts(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in DeleteUnconsentedUserAndAccounts")
	}

	if deleteErr := repository.DeleteUnconsentedUserAndAccounts(ctx, "user-uuid-1"); deleteErr != nil {
		testingInstance.Fatalf("expected DeleteUnconsentedUserAndAccounts to succeed in dry-run, got: %v", deleteErr)
	}
}
