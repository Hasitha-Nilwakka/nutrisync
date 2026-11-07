// Package database provides PostgreSQL connectivity and GORM schema migration helpers for the Go API service.
package database

import (
	"errors"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"nutrysync/go-api/models"
)

// ConnectToPostgreSQLDatabase opens a PostgreSQL database connection using GORM and the provided DSN/URL.
// Returns the initialized *gorm.DB instance or an error if the URL is blank or the connection attempt fails.
func ConnectToPostgreSQLDatabase(databaseURL string) (*gorm.DB, error) {
	// Reject empty or whitespace-only URLs early before attempting to initialize the PostgreSQL driver.
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("database URL must not be empty")
	}

	databaseConnection, connectionError := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if connectionError != nil {
		return nil, connectionError
	}

	return databaseConnection, nil
}

// RunDatabaseMigrations runs GORM AutoMigrate for the application's domain models
// (models.User, models.Session, models.RefreshToken, and models.Account).
// Expects a non-nil *gorm.DB connection created by ConnectToPostgreSQLDatabase and returns any migration error.
func RunDatabaseMigrations(databaseConnection *gorm.DB) error {
	// Guard against nil pointer dereference if called before a valid database connection is established.
	if databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	return databaseConnection.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.RefreshToken{},
		&models.Account{},
		&models.PasswordReset{},
		&models.HealthProfile{},
		&models.WeightEntry{},
		&models.ActivityEntry{},
	)
}
