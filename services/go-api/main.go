// Package main is the entry point for the Go API authentication and health check HTTP server running on port 8080.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"nutrysync/go-api/auth"
	"nutrysync/go-api/database"
	"nutrysync/go-api/handlers"
	"nutrysync/go-api/repositories"
)

// handleServiceHealthCheck writes the JSON health check status response for GET /health.
func handleServiceHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	healthCheckResponse := handlers.ServiceHealthCheckResponse{
		Status:  "ok",
		Service: "go-api",
	}

	_ = json.NewEncoder(w).Encode(healthCheckResponse)
}

// CreateApplicationMux builds and returns an *http.ServeMux with the health check route
// and optional user registration and session/authentication routes registered.
func CreateApplicationMux(
	userRegistrationHandler *handlers.UserRegistrationHandler,
	userSessionHandler *handlers.UserSessionHandler,
) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleServiceHealthCheck)

	if userRegistrationHandler != nil {
		mux.HandleFunc("POST /api/auth/register", userRegistrationHandler.HandleUserRegistrationRequest)
		mux.HandleFunc("GET /api/auth/verify", userRegistrationHandler.HandleUserEmailVerificationRequest)
	}

	if userSessionHandler != nil {
		mux.HandleFunc("POST /api/auth/login", userSessionHandler.HandleUserLoginRequest)
		mux.HandleFunc("POST /api/auth/refresh", userSessionHandler.HandleRefreshTokenRotationRequest)
		mux.HandleFunc("POST /api/auth/logout", userSessionHandler.HandleUserLogoutRequest)
		mux.HandleFunc("GET /api/auth/session", userSessionHandler.HandleVerifyActiveSessionRequest)
	}

	return mux
}

func main() {
	var userRegistrationHandler *handlers.UserRegistrationHandler
	var userSessionHandler *handlers.UserSessionHandler

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL != "" {
		databaseConnection, connectionError := database.ConnectToPostgreSQLDatabase(databaseURL)
		if connectionError != nil {
			log.Fatalf("Failed to connect to PostgreSQL database: %v", connectionError)
		}

		if migrationError := database.RunDatabaseMigrations(databaseConnection); migrationError != nil {
			log.Fatalf("Failed to run database migrations: %v", migrationError)
		}

		userRegistrationRepository := repositories.NewGormUserRegistrationRepository(databaseConnection)
		verificationEmailSender := auth.NewGmailSMTPVerificationEmailSender()
		userRegistrationHandler = handlers.NewUserRegistrationHandler(userRegistrationRepository, verificationEmailSender)

		userSessionRepository, sessionRepoError := repositories.NewGormUserSessionRepository(databaseConnection)
		if sessionRepoError != nil {
			log.Fatalf("Failed to initialize user session repository: %v", sessionRepoError)
		}

		jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
		if jwtSecret == "" {
			jwtSecret = "development-fallback-jwt-secret"
		}

		isProduction := strings.EqualFold(strings.TrimSpace(os.Getenv("NODE_ENV")), "production") ||
			strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")

		initializedSessionHandler, sessionHandlerError := handlers.NewUserSessionHandler(
			userSessionRepository,
			jwtSecret,
			isProduction,
		)
		if sessionHandlerError != nil {
			log.Fatalf("Failed to initialize user session handler: %v", sessionHandlerError)
		}
		userSessionHandler = initializedSessionHandler
	}

	applicationMux := CreateApplicationMux(userRegistrationHandler, userSessionHandler)

	log.Println("Server is running on http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", applicationMux))
}
