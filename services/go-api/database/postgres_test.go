// This file contains unit tests for ConnectToPostgreSQLDatabase and RunDatabaseMigrations in the database package.
package database

import (
	"testing"
)

// TestConnectToPostgreSQLDatabaseReturnsErrorWhenDatabaseURLIsEmpty verifies that ConnectToPostgreSQLDatabase
// rejects an empty database URL before attempting any network connection.
func TestConnectToPostgreSQLDatabaseReturnsErrorWhenDatabaseURLIsEmpty(testingInstance *testing.T) {
	databaseConnection, connectionError := ConnectToPostgreSQLDatabase("")
	if connectionError == nil {
		testingInstance.Fatal("expected error when databaseURL is empty, got nil")
	}

	if databaseConnection != nil {
		testingInstance.Fatalf("expected nil databaseConnection when error occurs, got %+v", databaseConnection)
	}
}

// TestConnectToPostgreSQLDatabaseReturnsErrorForUnreachableDatabase verifies that ConnectToPostgreSQLDatabase
// surfaces a connection error and nil *gorm.DB when the target PostgreSQL endpoint is unreachable.
func TestConnectToPostgreSQLDatabaseReturnsErrorForUnreachableDatabase(testingInstance *testing.T) {
	invalidDatabaseURL := "postgres://invalid_user:invalid_password@127.0.0.1:1/invalid_db?sslmode=disable&connect_timeout=1"
	databaseConnection, connectionError := ConnectToPostgreSQLDatabase(invalidDatabaseURL)
	if connectionError == nil {
		testingInstance.Fatal("expected error for unreachable database, got nil")
	}

	if databaseConnection != nil {
		testingInstance.Fatalf("expected nil databaseConnection for unreachable database, got %+v", databaseConnection)
	}
}

// TestRunDatabaseMigrationsReturnsErrorWhenDatabaseConnectionIsNil verifies that RunDatabaseMigrations
// safely returns an error rather than panicking when passed a nil *gorm.DB pointer.
func TestRunDatabaseMigrationsReturnsErrorWhenDatabaseConnectionIsNil(testingInstance *testing.T) {
	migrationError := RunDatabaseMigrations(nil)
	if migrationError == nil {
		testingInstance.Fatal("expected error when databaseConnection is nil, got nil")
	}
}
