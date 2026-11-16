// Package main is the entry point for the Go API authentication and health check HTTP server running on port 8080.
package main

import (
	"bufio"
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

// LoadDotEnvFileIfPresent reads one or more .env files (if present), ignores blank lines and '#' comments,
// parses KEY=VALUE pairs (stripping optional surrounding quotes), and populates os.Setenv only when os.Getenv(key) is currently empty.
func LoadDotEnvFileIfPresent(filePaths ...string) {
	for _, rawFilePath := range filePaths {
		trimmedFilePath := strings.TrimSpace(rawFilePath)
		if trimmedFilePath == "" {
			continue
		}

		envFile, openError := os.Open(trimmedFilePath)
		if openError != nil {
			continue
		}

		scanner := bufio.NewScanner(envFile)
		for scanner.Scan() {
			trimmedLine := strings.TrimSpace(scanner.Text())
			if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
				continue
			}

			trimmedLine = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "export "))
			separatorIndex := strings.Index(trimmedLine, "=")
			if separatorIndex <= 0 {
				continue
			}

			key := strings.TrimSpace(trimmedLine[:separatorIndex])
			value := strings.TrimSpace(trimmedLine[separatorIndex+1:])
			if key == "" {
				continue
			}

			if len(value) >= 2 {
				if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
					(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
					value = value[1 : len(value)-1]
				}
			}

			if strings.TrimSpace(os.Getenv(key)) == "" {
				_ = os.Setenv(key, value)
			}
		}

		_ = envFile.Close()
	}
}

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
		mux.HandleFunc("POST /api/auth/consent", userSessionHandler.HandleUserGDPRConsentRequest)
		mux.HandleFunc("DELETE /api/auth/account", userSessionHandler.HandleDeleteAuthenticatedUserAccountRequest)
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
		case *handlers.PasswordResetHandler:
			if typedHandler != nil {
				mux.HandleFunc("POST /api/auth/passwordreset", typedHandler.HandleRequestPasswordReset)
				mux.HandleFunc("POST /api/auth/passwordreset/confirm", typedHandler.HandleConfirmPasswordReset)
			}
		case *handlers.HealthProfileHandler:
			if typedHandler != nil {
				mux.HandleFunc("GET /api/health/profile", typedHandler.HandleGetHealthProfile)
				mux.HandleFunc("POST /api/health/profile", typedHandler.HandleSaveHealthProfile)
				mux.HandleFunc("PUT /api/health/profile", typedHandler.HandleSaveHealthProfile)
				mux.HandleFunc("GET /api/health/weight", typedHandler.HandleListWeightEntries)
				mux.HandleFunc("POST /api/health/weight", typedHandler.HandleLogWeightEntry)
				mux.HandleFunc("GET /api/health/activity", typedHandler.HandleListActivityEntries)
				mux.HandleFunc("POST /api/health/activity", typedHandler.HandleLogActivityEntry)
				mux.HandleFunc("GET /api/health/export", typedHandler.HandleExportUserHealthData)
				mux.HandleFunc("GET /api/health/dashboard", typedHandler.HandleGetHealthDashboard)
				mux.HandleFunc("POST /api/health/insights/regenerate", typedHandler.HandleRegenerateAIInsights)
			}
		}
	}

	return mux
}

func main() {
	LoadDotEnvFileIfPresent(".env", "../../.env")

	var userRegistrationHandler *handlers.UserRegistrationHandler
	var userSessionHandler *handlers.UserSessionHandler
	var googleOAuthHandler *handlers.GoogleOAuthHandler
	var githubOAuthHandler *handlers.GitHubOAuthHandler
	var twoFactorHandler *handlers.TwoFactorHandler
	var passwordResetHandler *handlers.PasswordResetHandler
	var healthProfileHandler *handlers.HealthProfileHandler

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
			jwtSecret = strings.TrimSpace(os.Getenv("ACCESS_TOKEN_SECRET"))
		}
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

		googleClientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
		if googleClientID == "" {
			googleClientID = strings.TrimSpace(os.Getenv("OAUTH_CLIENT_ID"))
		}

		googleClientSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
		if googleClientSecret == "" {
			googleClientSecret = strings.TrimSpace(os.Getenv("OAUTH_CLIENT_SECRET"))
		}

		googleOAuthConfig := &oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
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

		passwordResetRepository, passwordResetRepoError := repositories.NewGormPasswordResetRepository(databaseConnection)
		if passwordResetRepoError != nil {
			log.Fatalf("Failed to initialize password reset repository: %v", passwordResetRepoError)
		}

		passwordResetEmailSender := auth.NewGmailSMTPPasswordResetEmailSender()
		initializedPasswordResetHandler, passwordResetHandlerError := handlers.NewPasswordResetHandler(
			passwordResetRepository,
			passwordResetEmailSender,
		)
		if passwordResetHandlerError != nil {
			log.Fatalf("Failed to initialize password reset handler: %v", passwordResetHandlerError)
		}
		passwordResetHandler = initializedPasswordResetHandler

		healthProfileRepository, healthRepoError := repositories.NewGormHealthProfileRepository(databaseConnection)
		if healthRepoError != nil {
			log.Fatalf("Failed to initialize health profile repository: %v", healthRepoError)
		}

		initializedHealthProfileHandler, healthHandlerError := handlers.NewHealthProfileHandler(
			healthProfileRepository,
			userSessionRepository,
			jwtSecret,
		)
		if healthHandlerError != nil {
			log.Fatalf("Failed to initialize health profile handler: %v", healthHandlerError)
		}
		healthProfileHandler = initializedHealthProfileHandler
	}

	applicationMux := CreateApplicationMux(
		userRegistrationHandler,
		userSessionHandler,
		googleOAuthHandler,
		githubOAuthHandler,
		twoFactorHandler,
		passwordResetHandler,
		healthProfileHandler,
	)

	log.Println("Server is running on http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", applicationMux))
}
