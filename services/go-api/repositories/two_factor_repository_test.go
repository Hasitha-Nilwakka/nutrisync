// This file contains unit tests for the TwoFactorRepository methods implemented on GormUserSessionRepository.
package repositories

import (
	"context"
	"testing"
)

// TestGormUserSessionRepositoryTwoFactorMethodsValidationAndDryRunExecution verifies SetPendingTwoFactorSecret,
// EnableTwoFactorAuthentication, DisableTwoFactorAuthentication, and FindUserByID.
func TestGormUserSessionRepositoryTwoFactorMethodsValidationAndDryRunExecution(testingInstance *testing.T) {
	ctx := context.Background()
	dryRunDB := createTransactionalDryRunGormDatabase(testingInstance)
	repository, initErr := NewGormUserSessionRepository(dryRunDB)
	if initErr != nil {
		testingInstance.Fatalf("failed to initialize repository: %v", initErr)
	}

	var twoFactorRepo TwoFactorRepository = repository
	if twoFactorRepo == nil {
		testingInstance.Fatal("expected repository to implement TwoFactorRepository")
	}

	if err := repository.SetPendingTwoFactorSecret(ctx, "", "SECRET123"); err == nil {
		testingInstance.Fatal("expected error when userID is empty in SetPendingTwoFactorSecret")
	}

	if err := repository.SetPendingTwoFactorSecret(ctx, "user-1", ""); err == nil {
		testingInstance.Fatal("expected error when secret is empty in SetPendingTwoFactorSecret")
	}

	if err := repository.SetPendingTwoFactorSecret(ctx, "user-1", "JBSWY3DPEHPK3PXP"); err != nil {
		testingInstance.Fatalf("expected SetPendingTwoFactorSecret to succeed in dry-run, got: %v", err)
	}

	if err := repository.EnableTwoFactorAuthentication(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in EnableTwoFactorAuthentication")
	}

	if err := repository.EnableTwoFactorAuthentication(ctx, "user-1"); err != nil {
		testingInstance.Fatalf("expected EnableTwoFactorAuthentication to succeed in dry-run, got: %v", err)
	}

	if err := repository.DisableTwoFactorAuthentication(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in DisableTwoFactorAuthentication")
	}

	if err := repository.DisableTwoFactorAuthentication(ctx, "user-1"); err != nil {
		testingInstance.Fatalf("expected DisableTwoFactorAuthentication to succeed in dry-run, got: %v", err)
	}

	if _, err := repository.FindUserByID(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in FindUserByID")
	}

	userRecord, findErr := repository.FindUserByID(ctx, "user-1")
	if findErr != nil || userRecord == nil {
		testingInstance.Fatalf("expected dry-run FindUserByID to succeed, got user=%+v err=%v", userRecord, findErr)
	}
}
