// This file contains unit tests for GormUserRegistrationRepository and its UserRegistrationRepository methods.
package repositories

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"nutrysync/go-api/models"
)

// dryRunConnectionPool is a minimal gorm.ConnPool stub used to initialize the GORM PostgreSQL dialector
// in DryRun mode without opening a network socket.
type dryRunConnectionPool struct{}

func (dryRunConnectionPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, nil
}

func (dryRunConnectionPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (dryRunConnectionPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (dryRunConnectionPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

// createDryRunGormDatabaseConnection creates a *gorm.DB configured with the PostgreSQL dialector and DryRun enabled.
func createDryRunGormDatabaseConnection(testingInstance *testing.T) *gorm.DB {
	testingInstance.Helper()

	databaseConnection, openError := gorm.Open(
		postgres.New(postgres.Config{Conn: dryRunConnectionPool{}}),
		&gorm.Config{DryRun: true},
	)
	if openError != nil {
		testingInstance.Fatalf("failed to initialize dry-run GORM database connection: %v", openError)
	}

	return databaseConnection
}

// TestNewGormUserRegistrationRepositoryImplementsInterface verifies that GormUserRegistrationRepository
// satisfies the UserRegistrationRepository interface contract.
func TestNewGormUserRegistrationRepositoryImplementsInterface(testingInstance *testing.T) {
	dryRunDatabase := createDryRunGormDatabaseConnection(testingInstance)
	var repositoryInterface UserRegistrationRepository = NewGormUserRegistrationRepository(dryRunDatabase)

	if repositoryInterface == nil {
		testingInstance.Fatal("expected non-nil UserRegistrationRepository instance")
	}
}

// TestFindUserByEmailAddressValidationAndQueryGeneration verifies input validation and SQL generation
// for FindUserByEmailAddress.
func TestFindUserByEmailAddressValidationAndQueryGeneration(testingInstance *testing.T) {
	nilDatabaseRepository := NewGormUserRegistrationRepository(nil)
	if _, err := nilDatabaseRepository.FindUserByEmailAddress("user@example.com"); err == nil {
		testingInstance.Fatal("expected error when database connection is nil")
	}

	dryRunDatabase := createDryRunGormDatabaseConnection(testingInstance)
	repository := NewGormUserRegistrationRepository(dryRunDatabase)

	if _, err := repository.FindUserByEmailAddress("   "); err == nil {
		testingInstance.Fatal("expected error when emailAddress is blank")
	}

	userRecord, queryError := repository.FindUserByEmailAddress("user@example.com")
	if queryError != nil {
		testingInstance.Fatalf("expected dry-run query to succeed without error, got: %v", queryError)
	}

	if userRecord == nil {
		testingInstance.Fatal("expected non-nil userRecord in dry-run mode")
	}
}

// TestCreateUnverifiedUserRecordValidationAndEnforcesUnverifiedState verifies that CreateUnverifiedUserRecord
// validates inputs and forces Verified to false when creating a new user record.
func TestCreateUnverifiedUserRecordValidationAndEnforcesUnverifiedState(testingInstance *testing.T) {
	nilDatabaseRepository := NewGormUserRegistrationRepository(nil)
	if err := nilDatabaseRepository.CreateUnverifiedUserRecord(&models.User{}); err == nil {
		testingInstance.Fatal("expected error when database connection is nil")
	}

	dryRunDatabase := createDryRunGormDatabaseConnection(testingInstance)
	repository := NewGormUserRegistrationRepository(dryRunDatabase)

	if err := repository.CreateUnverifiedUserRecord(nil); err == nil {
		testingInstance.Fatal("expected error when user pointer is nil")
	}

	candidateUser := &models.User{
		Email:     "newuser@example.com",
		Password:  "hashed-password",
		Consented: true,
		Verified:  true,
	}

	creationError := repository.CreateUnverifiedUserRecord(candidateUser)
	if creationError != nil {
		testingInstance.Fatalf("expected dry-run CreateUnverifiedUserRecord to succeed, got: %v", creationError)
	}

	if candidateUser.Verified {
		testingInstance.Error("expected CreateUnverifiedUserRecord to set candidateUser.Verified to false")
	}

	if candidateUser.Consented {
		testingInstance.Error("expected CreateUnverifiedUserRecord to set candidateUser.Consented to false")
	}
}

// TestFindUserByVerificationTokenValidationAndQueryGeneration verifies input validation and SQL execution
// for FindUserByVerificationToken.
func TestFindUserByVerificationTokenValidationAndQueryGeneration(testingInstance *testing.T) {
	nilDatabaseRepository := NewGormUserRegistrationRepository(nil)
	if _, err := nilDatabaseRepository.FindUserByVerificationToken("valid-token"); err == nil {
		testingInstance.Fatal("expected error when database connection is nil")
	}

	dryRunDatabase := createDryRunGormDatabaseConnection(testingInstance)
	repository := NewGormUserRegistrationRepository(dryRunDatabase)

	if _, err := repository.FindUserByVerificationToken(""); err == nil {
		testingInstance.Fatal("expected error when verificationToken is empty")
	}

	userRecord, queryError := repository.FindUserByVerificationToken("valid-token")
	if queryError != nil {
		testingInstance.Fatalf("expected dry-run query to succeed without error, got: %v", queryError)
	}

	if userRecord == nil {
		testingInstance.Fatal("expected non-nil userRecord in dry-run mode")
	}
}

// TestMarkUserEmailAsVerifiedValidationAndUpdateExecution verifies input validation and update execution
// for MarkUserEmailAsVerified.
func TestMarkUserEmailAsVerifiedValidationAndUpdateExecution(testingInstance *testing.T) {
	verificationTimestamp := time.Now().UTC()

	nilDatabaseRepository := NewGormUserRegistrationRepository(nil)
	if err := nilDatabaseRepository.MarkUserEmailAsVerified("user-uuid", verificationTimestamp); err == nil {
		testingInstance.Fatal("expected error when database connection is nil")
	}

	dryRunDatabase := createDryRunGormDatabaseConnection(testingInstance)
	repository := NewGormUserRegistrationRepository(dryRunDatabase)

	if err := repository.MarkUserEmailAsVerified("", verificationTimestamp); err == nil {
		testingInstance.Fatal("expected error when userID is empty")
	}

	updateError := repository.MarkUserEmailAsVerified("11111111-1111-1111-1111-111111111111", verificationTimestamp)
	if updateError != nil && !strings.Contains(updateError.Error(), "") {
		testingInstance.Fatalf("expected dry-run MarkUserEmailAsVerified to succeed, got: %v", updateError)
	}
}

// TestDeleteUnverifiedUserRecordValidationAndExecution verifies nil/empty validation and dry-run execution
// for DeleteUnverifiedUserRecord.
func TestDeleteUnverifiedUserRecordValidationAndExecution(testingInstance *testing.T) {
	nilDatabaseRepository := NewGormUserRegistrationRepository(nil)
	if err := nilDatabaseRepository.DeleteUnverifiedUserRecord("user-uuid"); err == nil {
		testingInstance.Fatal("expected error when database connection is nil")
	}

	dryRunDatabase := createDryRunGormDatabaseConnection(testingInstance)
	repository := NewGormUserRegistrationRepository(dryRunDatabase)

	if err := repository.DeleteUnverifiedUserRecord("   "); err == nil {
		testingInstance.Fatal("expected error when userID is blank")
	}

	if err := repository.DeleteUnverifiedUserRecord("11111111-1111-1111-1111-111111111111"); err != nil {
		testingInstance.Fatalf("expected dry-run DeleteUnverifiedUserRecord to succeed, got: %v", err)
	}
}

// TestRefreshVerificationTokenValidationAndExecution verifies nil/empty validation and dry-run execution
// for RefreshVerificationToken.
func TestRefreshVerificationTokenValidationAndExecution(testingInstance *testing.T) {
	expiry := time.Now().UTC().Add(24 * time.Hour)

	nilDatabaseRepository := NewGormUserRegistrationRepository(nil)
	if err := nilDatabaseRepository.RefreshVerificationToken("user-uuid", "token", expiry); err == nil {
		testingInstance.Fatal("expected error when database connection is nil")
	}

	dryRunDatabase := createDryRunGormDatabaseConnection(testingInstance)
	repository := NewGormUserRegistrationRepository(dryRunDatabase)

	if err := repository.RefreshVerificationToken(" ", "token", expiry); err == nil {
		testingInstance.Fatal("expected error when userID is blank")
	}

	if err := repository.RefreshVerificationToken("11111111-1111-1111-1111-111111111111", "new-token", expiry); err != nil {
		testingInstance.Fatalf("expected dry-run RefreshVerificationToken to succeed, got: %v", err)
	}
}
