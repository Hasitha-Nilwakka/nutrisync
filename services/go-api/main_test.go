// This file contains unit tests for handleServiceHealthCheck and CreateApplicationMux in package main.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"nutrysync/go-api/handlers"
	"nutrysync/go-api/models"
)

type stubMainSessionRepository struct{}

func (stubMainSessionRepository) FindUserByEmailForLogin(ctx context.Context, email string) (*models.User, error) {
	return nil, nil
}
func (stubMainSessionRepository) CreateSessionWithInitialRefreshToken(ctx context.Context, session *models.Session, initialRefreshToken *models.RefreshToken) error {
	return nil
}
func (stubMainSessionRepository) FindRefreshTokenWithSessionByID(ctx context.Context, refreshTokenID string) (*models.RefreshToken, error) {
	return nil, nil
}
func (stubMainSessionRepository) RotateRefreshTokenInTransaction(ctx context.Context, currentTokenID string, newRefreshToken *models.RefreshToken) error {
	return nil
}
func (stubMainSessionRepository) RevokeSessionAndAllRefreshTokens(ctx context.Context, sessionID string) error {
	return nil
}
func (stubMainSessionRepository) FindSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {
	return nil, nil
}
func (stubMainSessionRepository) AcceptUserGDPRConsent(ctx context.Context, userID string) error {
	return nil
}
func (stubMainSessionRepository) DeleteUnconsentedUserAndAccounts(ctx context.Context, userID string) error {
	return nil
}
func (stubMainSessionRepository) DeleteAuthenticatedUserAccountAndAllData(ctx context.Context, userID string) error {
	return nil
}

type stubMainAccountRepository struct{}

func (stubMainAccountRepository) FindAccountByProviderAndID(ctx context.Context, provider, providerUserID string) (*models.Account, error) {
	return nil, nil
}
func (stubMainAccountRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, nil
}
func (stubMainAccountRepository) CreateOAuthUserAndAccount(ctx context.Context, user *models.User, account *models.Account) error {
	return nil
}
func (stubMainAccountRepository) LinkAccountToExistingUser(ctx context.Context, account *models.Account) error {
	return nil
}

func TestHandleServiceHealthCheckReturnsExpectedJSONResponse(testingInstance *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	responseRecorder := httptest.NewRecorder()

	handleServiceHealthCheck(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
	}

	expectedContentType := "application/json"
	actualContentType := responseRecorder.Header().Get("Content-Type")
	if actualContentType != expectedContentType {
		testingInstance.Fatalf("expected Content-Type %q, got %q", expectedContentType, actualContentType)
	}

	var responseBody map[string]string
	decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody)
	if decodeError != nil {
		testingInstance.Fatalf("failed to decode JSON response body: %v", decodeError)
	}

	if responseBody["status"] != "ok" {
		testingInstance.Errorf("expected status %q, got %q", "ok", responseBody["status"])
	}

	if responseBody["service"] != "go-api" {
		testingInstance.Errorf("expected service %q, got %q", "go-api", responseBody["service"])
	}
}

func TestCreateApplicationMuxRegistersHealthRegistrationAndSessionRoutes(testingInstance *testing.T) {
	registrationHandler := handlers.NewUserRegistrationHandler(nil, nil)
	sessionHandler, err := handlers.NewUserSessionHandler(stubMainSessionRepository{}, "test-secret", false)
	if err != nil {
		testingInstance.Fatalf("failed to initialize sessionHandler: %v", err)
	}

	googleHandler, _ := handlers.NewGoogleOAuthHandler(
		stubMainAccountRepository{},
		stubMainSessionRepository{},
		&oauth2.Config{
			ClientID: "google-id",
			Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/auth"},
		},
		"test-secret",
		"http://localhost:3000",
		false,
	)

	githubHandler, _ := handlers.NewGitHubOAuthHandler(
		stubMainAccountRepository{},
		stubMainSessionRepository{},
		&oauth2.Config{
			ClientID: "github-id",
			Endpoint: github.Endpoint,
		},
		"test-secret",
		"http://localhost:3000",
		false,
	)

	mux := CreateApplicationMux(registrationHandler, sessionHandler, googleHandler, githubHandler)

	routesToCheck := []struct {
		method         string
		path           string
		expectedStatus int
	}{
		{method: http.MethodGet, path: "/health", expectedStatus: http.StatusOK},
		{method: http.MethodGet, path: "/api/auth/verify", expectedStatus: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/auth/logout", expectedStatus: http.StatusOK},
		{method: http.MethodGet, path: "/api/auth/session", expectedStatus: http.StatusUnauthorized},
		{method: http.MethodGet, path: "/api/auth/oauth/google", expectedStatus: http.StatusTemporaryRedirect},
		{method: http.MethodGet, path: "/api/auth/oauth/google/callback", expectedStatus: http.StatusSeeOther},
		{method: http.MethodGet, path: "/api/auth/oauth/github", expectedStatus: http.StatusTemporaryRedirect},
		{method: http.MethodGet, path: "/api/auth/oauth/github/callback", expectedStatus: http.StatusSeeOther},
	}

	for _, route := range routesToCheck {
		request := httptest.NewRequest(route.method, route.path, nil)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)

		if recorder.Code != route.expectedStatus {
			testingInstance.Errorf("expected route %s %s to return %d, got %d", route.method, route.path, route.expectedStatus, recorder.Code)
		}
	}
}

func TestLoadDotEnvFileIfPresentParsesKeyValuesAndPreservesExistingEnv(testingInstance *testing.T) {
	tempDir := testingInstance.TempDir()
	dotEnvPath := filepath.Join(tempDir, ".env")

	dotEnvContents := `
# Comment line to ignore
TEST_DOTENV_NEW_KEY = "loaded-from-dotenv"
TEST_DOTENV_SINGLE_QUOTED='single-quoted-value'
TEST_DOTENV_EXISTING_KEY=should-not-overwrite
`
	if writeErr := os.WriteFile(dotEnvPath, []byte(dotEnvContents), 0600); writeErr != nil {
		testingInstance.Fatalf("failed to write temporary .env file: %v", writeErr)
	}

	_ = os.Unsetenv("TEST_DOTENV_NEW_KEY")
	_ = os.Unsetenv("TEST_DOTENV_SINGLE_QUOTED")
	testingInstance.Setenv("TEST_DOTENV_EXISTING_KEY", "already-set")
	testingInstance.Cleanup(func() {
		_ = os.Unsetenv("TEST_DOTENV_NEW_KEY")
		_ = os.Unsetenv("TEST_DOTENV_SINGLE_QUOTED")
	})

	LoadDotEnvFileIfPresent(filepath.Join(tempDir, "nonexistent.env"), dotEnvPath)

	if got := os.Getenv("TEST_DOTENV_NEW_KEY"); got != "loaded-from-dotenv" {
		testingInstance.Errorf("expected TEST_DOTENV_NEW_KEY to be %q, got %q", "loaded-from-dotenv", got)
	}

	if got := os.Getenv("TEST_DOTENV_SINGLE_QUOTED"); got != "single-quoted-value" {
		testingInstance.Errorf("expected TEST_DOTENV_SINGLE_QUOTED to be %q, got %q", "single-quoted-value", got)
	}

	if got := os.Getenv("TEST_DOTENV_EXISTING_KEY"); got != "already-set" {
		testingInstance.Errorf("expected TEST_DOTENV_EXISTING_KEY to remain %q, got %q", "already-set", got)
	}
}
