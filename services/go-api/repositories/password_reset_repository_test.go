// This file contains unit tests for GormPasswordResetRepository and its PasswordResetRepository methods.
package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestNewGormPasswordResetRepositoryValidatesDatabaseConnection(t *testing.T) {
	if _, err := NewGormPasswordResetRepository(nil); err == nil {
		t.Fatal("expected error when database connection is nil")
	}

	dryRunDatabase := createDryRunGormDatabaseConnection(t)
	repository, err := NewGormPasswordResetRepository(dryRunDatabase)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var _ PasswordResetRepository = repository
}

func TestGormPasswordResetRepositoryMethodsValidationAndDryRunExecution(t *testing.T) {
	dryRunDatabase := createDryRunGormDatabaseConnection(t)
	repository, err := NewGormPasswordResetRepository(dryRunDatabase)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx := context.Background()
	expiry := time.Now().UTC().Add(1 * time.Hour)

	if user, lookupErr := repository.FindUserByEmailForPasswordReset(ctx, "   "); lookupErr != nil || user != nil {
		t.Fatalf("expected (nil, nil) for blank email, got (%v, %v)", user, lookupErr)
	}
	if _, lookupErr := repository.FindUserByEmailForPasswordReset(ctx, "Member@Example.com"); lookupErr != nil {
		t.Fatalf("expected dry-run email lookup to succeed, got %v", lookupErr)
	}

	if replaceErr := repository.ReplaceActivePasswordResetToken(ctx, "", "hash", expiry); replaceErr == nil {
		t.Fatal("expected error when userID is empty")
	}
	if replaceErr := repository.ReplaceActivePasswordResetToken(ctx, "11111111-1111-1111-1111-111111111111", "sha256-hash", expiry); replaceErr != nil && !errors.Is(replaceErr, gorm.ErrInvalidTransaction) {
		t.Fatalf("expected dry-run ReplaceActivePasswordResetToken to reach transaction cleanly, got %v", replaceErr)
	}

	if record, findErr := repository.FindPasswordResetByTokenHash(ctx, "   "); findErr != nil || record != nil {
		t.Fatalf("expected (nil, nil) for blank token hash, got (%v, %v)", record, findErr)
	}
	if _, findErr := repository.FindPasswordResetByTokenHash(ctx, "sha256-hash"); findErr != nil {
		t.Fatalf("expected dry-run FindPasswordResetByTokenHash to succeed, got %v", findErr)
	}

	if completeErr := repository.CompletePasswordResetAndRevokeSessions(ctx, "", "u1", "hash", time.Now().UTC()); completeErr == nil {
		t.Fatal("expected error when resetID is empty")
	}
	if completeErr := repository.CompletePasswordResetAndRevokeSessions(
		ctx,
		"22222222-2222-2222-2222-222222222222",
		"11111111-1111-1111-1111-111111111111",
		"bcrypt-hash",
		time.Now().UTC(),
	); completeErr != nil && !errors.Is(completeErr, gorm.ErrInvalidTransaction) {
		t.Fatalf("expected dry-run CompletePasswordResetAndRevokeSessions to reach transaction cleanly, got %v", completeErr)
	}
}
