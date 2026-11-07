// This file contains unit tests for GoogleOAuthHandler covering OAuth login redirect,
// state mismatch rejection, federated account linking for existing users, and new OAuth user onboarding redirect.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"
	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
)

// mockAccountRepository is an in-memory test double implementing repositories.AccountRepository.
type mockAccountRepository struct {
	accountByProvider *models.Account
	userByEmail       *models.User
	createdUser       *models.User
	createdAccount    *models.Account
	linkedAccount     *models.Account
}

func (mockRepo *mockAccountRepository) FindAccountByProviderAndID(
	ctx context.Context,
	provider, providerUserID string,
) (*models.Account, error) {
	return mockRepo.accountByProvider, nil
}

func (mockRepo *mockAccountRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return mockRepo.userByEmail, nil
}

func (mockRepo *mockAccountRepository) CreateOAuthUserAndAccount(
	ctx context.Context,
	user *models.User,
	account *models.Account,
) error {
	mockRepo.createdUser = user
	mockRepo.createdAccount = account
	return nil
}

func (mockRepo *mockAccountRepository) LinkAccountToExistingUser(
	ctx context.Context,
	account *models.Account,
) error {
	mockRepo.linkedAccount = account
	return nil
}

// TestHandleGoogleOAuthLoginSetsStateCookieAndRedirectsToGoogle verifies GET /api/auth/oauth/google.
func TestHandleGoogleOAuthLoginSetsStateCookieAndRedirectsToGoogle(testingInstance *testing.T) {
	oauthConfig := &oauth2.Config{
		ClientID:     "google-client-id",
		ClientSecret: "google-client-secret",
		RedirectURL:  "http://localhost:3000/api/auth/oauth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	oauthHandler, initErr := NewGoogleOAuthHandler(
		&mockAccountRepository{},
		&mockUserSessionRepository{},
		oauthConfig,
		"jwt-secret",
		"http://localhost:3000",
		false,
	)
	if initErr != nil {
		testingInstance.Fatalf("expected NewGoogleOAuthHandler to succeed, got: %v", initErr)
	}

	loginRequest := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/google", nil)
	loginRecorder := httptest.NewRecorder()

	oauthHandler.HandleGoogleOAuthLogin(loginRecorder, loginRequest)

	if loginRecorder.Code != http.StatusTemporaryRedirect {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, loginRecorder.Code)
	}

	locationHeader := loginRecorder.Header().Get("Location")
	if !strings.HasPrefix(locationHeader, "https://accounts.google.com/o/oauth2/v2/auth") {
		testingInstance.Errorf("expected redirect Location to Google auth URL, got %q", locationHeader)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != auth.OAuthStateCookieName || cookies[0].Value == "" {
		testingInstance.Errorf("expected oauth_state cookie to be set, got %+v", cookies)
	}
}

// TestHandleGoogleOAuthCallbackStateMismatchNewUserAndExistingUserLinking verifies:
// 1. State mismatch redirects to /?error=oauth_state_invalid
// 2. New OAuth user creation (Consented: false) redirects to /dashboard?onboarding=consent
// 3. Existing user with Consented: true links account and redirects to /dashboard
func TestHandleGoogleOAuthCallbackStateMismatchNewUserAndExistingUserLinking(testingInstance *testing.T) {
	mockGoogleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "mock-google-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case "/userinfo":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(GoogleOpenIDUserInfoPayload{
				Sub:           "google-sub-12345",
				Email:         "federated@example.com",
				EmailVerified: true,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockGoogleServer.Close()

	oauthConfig := &oauth2.Config{
		ClientID:     "google-client-id",
		ClientSecret: "google-client-secret",
		RedirectURL:  "http://localhost:3000/api/auth/oauth/google/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  mockGoogleServer.URL + "/auth",
			TokenURL: mockGoogleServer.URL + "/token",
		},
	}

	mockAccountRepo := &mockAccountRepository{}
	mockSessionRepo := &mockUserSessionRepository{}

	oauthHandler, _ := NewGoogleOAuthHandler(
		mockAccountRepo,
		mockSessionRepo,
		oauthConfig,
		"jwt-secret-oauth",
		"http://localhost:3000",
		false,
	)
	oauthHandler.userInfoEndpointURL = mockGoogleServer.URL + "/userinfo"
	oauthHandler.httpClient = mockGoogleServer.Client()

	// 1. State mismatch -> redirects to /?error=oauth_state_invalid
	mismatchReq := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/google/callback?state=wrong&code=abc", nil)
	mismatchReq.AddCookie(&http.Cookie{Name: auth.OAuthStateCookieName, Value: "expected"})
	mismatchRec := httptest.NewRecorder()
	oauthHandler.HandleGoogleOAuthCallback(mismatchRec, mismatchReq)

	if mismatchRec.Header().Get("Location") != "http://localhost:3000/?error=oauth_state_invalid" {
		testingInstance.Fatalf("expected redirect to /?error=oauth_state_invalid, got %q", mismatchRec.Header().Get("Location"))
	}

	// 2. New OAuth user (Consented: false) -> creates User & Account and redirects to /dashboard?onboarding=consent
	validState := "valid-state-token-123"
	newUserReq := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/google/callback?state="+validState+"&code=auth-code-1", nil)
	newUserReq.AddCookie(&http.Cookie{Name: auth.OAuthStateCookieName, Value: validState})
	newUserRec := httptest.NewRecorder()
	oauthHandler.HandleGoogleOAuthCallback(newUserRec, newUserReq)

	if newUserRec.Code != http.StatusSeeOther {
		testingInstance.Fatalf("expected 303 SeeOther redirect, got %d", newUserRec.Code)
	}

	if newUserRec.Header().Get("Location") != "http://localhost:3000/dashboard?onboarding=consent" {
		testingInstance.Errorf("expected redirect to /dashboard?onboarding=consent, got %q", newUserRec.Header().Get("Location"))
	}

	if mockAccountRepo.createdUser == nil || mockAccountRepo.createdAccount == nil {
		testingInstance.Fatal("expected CreateOAuthUserAndAccount to be called for new user")
	}

	// 3. Existing user by email (Consented: true) -> links Account and redirects to /dashboard
	mockAccountRepo.userByEmail = &models.User{
		ID:        "existing-user-uuid",
		Email:     "federated@example.com",
		Consented: true,
		Verified:  false,
	}

	linkReq := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/google/callback?state="+validState+"&code=auth-code-2", nil)
	linkReq.AddCookie(&http.Cookie{Name: auth.OAuthStateCookieName, Value: validState})
	linkRec := httptest.NewRecorder()
	oauthHandler.HandleGoogleOAuthCallback(linkRec, linkReq)

	if linkRec.Header().Get("Location") != "http://localhost:3000/dashboard" {
		testingInstance.Errorf("expected redirect to /dashboard for consented user, got %q", linkRec.Header().Get("Location"))
	}

	if mockAccountRepo.linkedAccount == nil || mockAccountRepo.linkedAccount.UserID != "existing-user-uuid" {
		testingInstance.Errorf("expected LinkAccountToExistingUser to link existing-user-uuid, got %+v", mockAccountRepo.linkedAccount)
	}
}
