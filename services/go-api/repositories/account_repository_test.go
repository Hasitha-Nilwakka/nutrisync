// This file contains unit tests for GormAccountRepository and all AccountRepository interface methods.
package repositories

import (
	"context"
	"testing"

	"nutrysync/go-api/models"
)

// TestNewGormAccountRepositoryValidatesDatabaseConnection verifies constructor validation and interface compliance.
func TestNewGormAccountRepositoryValidatesDatabaseConnection(testingInstance *testing.T) {
	if _, err := NewGormAccountRepository(nil); err == nil {
		testingInstance.Fatal("expected error when initializing NewGormAccountRepository with nil *gorm.DB")
	}

	dryRunDB := createTransactionalDryRunGormDatabase(testingInstance)
	repository, initErr := NewGormAccountRepository(dryRunDB)
	if initErr != nil {
		testingInstance.Fatalf("expected NewGormAccountRepository to succeed, got: %v", initErr)
	}

	var repositoryInterface AccountRepository = repository
	if repositoryInterface == nil {
		testingInstance.Fatal("expected non-nil AccountRepository")
	}
}

// TestGormAccountRepositoryMethodsValidationAndDryRunExecution verifies FindAccountByProviderAndID,
// FindUserByEmail, CreateOAuthUserAndAccount, and LinkAccountToExistingUser.
func TestGormAccountRepositoryMethodsValidationAndDryRunExecution(testingInstance *testing.T) {
	ctx := context.Background()
	dryRunDB := createTransactionalDryRunGormDatabase(testingInstance)
	repository, _ := NewGormAccountRepository(dryRunDB)

	if _, err := repository.FindAccountByProviderAndID(ctx, "", "sub-123"); err == nil {
		testingInstance.Fatal("expected error when provider is empty in FindAccountByProviderAndID")
	}

	if _, err := repository.FindAccountByProviderAndID(ctx, "google", ""); err == nil {
		testingInstance.Fatal("expected error when providerUserID is empty in FindAccountByProviderAndID")
	}

	accountRecord, accountErr := repository.FindAccountByProviderAndID(ctx, "google", "google-sub-123")
	if accountErr != nil || accountRecord == nil {
		testingInstance.Fatalf("expected dry-run FindAccountByProviderAndID to succeed, got account=%+v err=%v", accountRecord, accountErr)
	}

	if _, err := repository.FindUserByEmail(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when email is empty in FindUserByEmail")
	}

	userRecord, userErr := repository.FindUserByEmail(ctx, "oauth@example.com")
	if userErr != nil || userRecord == nil {
		testingInstance.Fatalf("expected dry-run FindUserByEmail to succeed, got user=%+v err=%v", userRecord, userErr)
	}

	if err := repository.CreateOAuthUserAndAccount(ctx, nil, &models.Account{}); err == nil {
		testingInstance.Fatal("expected error when user is nil in CreateOAuthUserAndAccount")
	}

	if err := repository.CreateOAuthUserAndAccount(ctx, &models.User{}, nil); err == nil {
		testingInstance.Fatal("expected error when account is nil in CreateOAuthUserAndAccount")
	}

	newOAuthUser := &models.User{
		ID:        "oauth-user-uuid-1",
		Email:     "new-oauth@example.com",
		Consented: false,
		Verified:  true,
	}
	newOAuthAccount := &models.Account{
		ID:             "oauth-account-uuid-1",
		Provider:       "google",
		ProviderUserID: "google-sub-999",
		ProviderEmail:  "new-oauth@example.com",
	}

	if createErr := repository.CreateOAuthUserAndAccount(ctx, newOAuthUser, newOAuthAccount); createErr != nil {
		testingInstance.Fatalf("expected CreateOAuthUserAndAccount to succeed in dry-run, got: %v", createErr)
	}

	if newOAuthAccount.UserID != newOAuthUser.ID {
		testingInstance.Errorf("expected newOAuthAccount.UserID to be %q, got %q", newOAuthUser.ID, newOAuthAccount.UserID)
	}

	if err := repository.LinkAccountToExistingUser(ctx, nil); err == nil {
		testingInstance.Fatal("expected error when account is nil in LinkAccountToExistingUser")
	}

	if err := repository.LinkAccountToExistingUser(ctx, &models.Account{}); err == nil {
		testingInstance.Fatal("expected error when account.UserID is empty in LinkAccountToExistingUser")
	}

	if linkErr := repository.LinkAccountToExistingUser(ctx, newOAuthAccount); linkErr != nil {
		testingInstance.Fatalf("expected LinkAccountToExistingUser to succeed in dry-run, got: %v", linkErr)
	}
}
