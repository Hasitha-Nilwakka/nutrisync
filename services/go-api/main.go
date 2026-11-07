// Package main is the entry point for the Go API authentication and health check HTTP server running on port 8080.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
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
// and optional user registration, session/authentication, Google OAuth, and GitHub OAuth routes registered.
func CreateApplicationMux(
	userRegistrationHandler *handlers.UserRegistrationHandler,
	userSessionHandler *handlers.UserSessionHandler,
	oauthHandlers ...any,
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

	for _, candidateHandler := range oauthHandlers {
		switch typedHandler := candidateHandler.(type) {
		case *handlers.GoogleOAuthHandler:
			if typedHandler != nil {
				mux.HandleFunc("GET /api/auth/oauth/google", typedHandler.HandleGoogleOAuthLogin)
				mux.HandleFunc("GET /api/auth/oauth/google/callback", typedHandler.HandleGoogleOAuthCallback)
			}
		case *handlers.GitHubOAuthHandler:
			if typedHandler != nil {
				mux.HandleFunc("GET /api/auth/oauth/github", typedHandler.HandleGitHubOAuthLogin)
				mux.HandleFunc("GET /api/auth/oauth/github/callback", typedHandler.HandleGitHubOAuthCallback)
			}
		case *handlers.TwoFactorHandler:
			if typedHandler != nil {
				mux.HandleFunc("POST /api/auth/2fa/setup", typedHandler.HandleTwoFactorSetupRequest)
				mux.HandleFunc("POST /api/auth/2fa/enable", typedHandler.HandleTwoFactorEnableRequest)
				mux.HandleFunc("POST /api/auth/2fa/disable", typedHandler.HandleTwoFactorDisableRequest)
				mux.HandleFunc("POST /api/auth/2fa/login-verify", typedHandler.HandleTwoFactorLoginVerifyRequest)
			}
		}
	}

	return mux
}

func main() {
	var userRegistrationHandler *handlers.UserRegistrationHandler
	var userSessionHandler *handlers.UserSessionHandler
	var googleOAuthHandler *handlers.GoogleOAuthHandler
	var githubOAuthHandler *handlers.GitHubOAuthHandler
	var twoFactorHandler *handlers.TwoFactorHandler

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

		initializedGoogleHandler, googleHandlerError := handlers.NewGoogleOAuthHandler(
			accountRepository,
			userSessionRepository,
			googleOAuthConfig,
			jwtSecret,
			appBaseURL,
			isProduction,
		)
		if googleHandlerError != nil {
			log.Fatalf("Failed to initialize Google OAuth handler: %v", googleHandlerError)
		}
		googleOAuthHandler = initializedGoogleHandler

		githubOAuthConfig := &oauth2.Config{
			ClientID:     strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET")),
			RedirectURL:  appBaseURL + "/api/auth/oauth/github/callback",
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     github.Endpoint,
		}

		initializedGitHubHandler, githubHandlerError := handlers.NewGitHubOAuthHandler(
			accountRepository,
			userSessionRepository,
			githubOAuthConfig,
			jwtSecret,
			appBaseURL,
			isProduction,
		)
		if githubHandlerError != nil {
			log.Fatalf("Failed to initialize GitHub OAuth handler: %v", githubHandlerError)
		}
		githubOAuthHandler = initializedGitHubHandler

		initializedTwoFactorHandler, twoFactorHandlerError := handlers.NewTwoFactorHandler(
			userSessionRepository,
			userSessionRepository,
			jwtSecret,
			isProduction,
		)
		if twoFactorHandlerError != nil {
			log.Fatalf("Failed to initialize two-factor handler: %v", twoFactorHandlerError)
		}
		twoFactorHandler = initializedTwoFactorHandler
	}

	applicationMux := CreateApplicationMux(
		userRegistrationHandler,
		userSessionHandler,
		googleOAuthHandler,
		githubOAuthHandler,
		twoFactorHandler,
	)

	log.Println("Server is running on http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", applicationMux))
}
