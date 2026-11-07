// Package main is the entry point for the Go API authentication and health check HTTP server running on port 8080.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
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
// and optional user registration, session/authentication, and Google OAuth routes registered.
func CreateApplicationMux(
	userRegistrationHandler *handlers.UserRegistrationHandler,
	userSessionHandler *handlers.UserSessionHandler,
	googleOAuthHandlers ...*handlers.GoogleOAuthHandler,
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

	if len(googleOAuthHandlers) > 0 && googleOAuthHandlers[0] != nil {
		googleOAuthHandler := googleOAuthHandlers[0]
		mux.HandleFunc("GET /api/auth/oauth/google", googleOAuthHandler.HandleGoogleOAuthLogin)
		mux.HandleFunc("GET /api/auth/oauth/google/callback", googleOAuthHandler.HandleGoogleOAuthCallback)
	}

	return mux
}

func main() {
	var userRegistrationHandler *handlers.UserRegistrationHandler
	var userSessionHandler *handlers.UserSessionHandler
	var googleOAuthHandler *handlers.GoogleOAuthHandler

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

		accountRepository, accountRepoError := repositories.NewGormAccountRepository(databaseConnection)
		if accountRepoError != nil {
			log.Fatalf("Failed to initialize account repository: %v", accountRepoError)
		}

		appBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("APP_URL")), "/")
		if appBaseURL == "" {
			appBaseURL = "http://localhost:3000"
		}

		googleOAuthConfig := &oauth2.Config{
			ClientID:     strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
			RedirectURL:  appBaseURL + "/api/auth/oauth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://accounts.google.com/o/oauth2/auth",
				TokenURL:  "https://oauth2.googleapis.com/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		}

		initializedOAuthHandler, oauthHandlerError := handlers.NewGoogleOAuthHandler(
			accountRepository,
			userSessionRepository,
			googleOAuthConfig,
			jwtSecret,
			appBaseURL,
			isProduction,
		)
		if oauthHandlerError != nil {
			log.Fatalf("Failed to initialize Google OAuth handler: %v", oauthHandlerError)
		}
		googleOAuthHandler = initializedOAuthHandler
	}

	applicationMux := CreateApplicationMux(userRegistrationHandler, userSessionHandler, googleOAuthHandler)

	log.Println("Server is running on http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", applicationMux))
}
