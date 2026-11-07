// This file contains unit tests for GitHubOAuthHandler covering OAuth login redirect,
// callback state validation, unverified email rejection, new user creation, and existing user linking.
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
)

// TestHandleGitHubOAuthLoginSetsStateCookieAndRedirectsToGitHub verifies GET /api/auth/oauth/github.
func TestHandleGitHubOAuthLoginSetsStateCookieAndRedirectsToGitHub(testingInstance *testing.T) {
	oauthConfig := &oauth2.Config{
		ClientID:     "github-client-id",
		ClientSecret: "github-client-secret",
		RedirectURL:  "http://localhost:3000/api/auth/oauth/github/callback",
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}

	githubHandler, initErr := NewGitHubOAuthHandler(
		&mockAccountRepository{},
		&mockUserSessionRepository{},
		oauthConfig,
		"jwt-secret",
		"http://localhost:3000",
		false,
	)
	if initErr != nil {
		testingInstance.Fatalf("expected NewGitHubOAuthHandler to succeed, got: %v", initErr)
	}

	loginRequest := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github", nil)
	loginRecorder := httptest.NewRecorder()

	githubHandler.HandleGitHubOAuthLogin(loginRecorder, loginRequest)

	if loginRecorder.Code != http.StatusTemporaryRedirect {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, loginRecorder.Code)
	}

	locationHeader := loginRecorder.Header().Get("Location")
	if !strings.HasPrefix(locationHeader, "https://github.com/login/oauth/authorize") {
		testingInstance.Errorf("expected redirect Location to GitHub authorize URL, got %q", locationHeader)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != auth.OAuthStateCookieName || cookies[0].Value == "" {
		testingInstance.Errorf("expected oauth_state cookie to be set, got %+v", cookies)
	}
}

// TestHandleGitHubOAuthCallbackStateMismatchNewUserAndExistingUserLinking verifies:
// 1. State mismatch redirects to /?error=oauth_state_invalid
// 2. Unverified emails redirect to /?error=oauth_email_unverified
// 3. New GitHub user creation (Consented: false) redirects to /dashboard?onboarding=consent
// 4. Existing user by email (Consented: true) links GitHub account and redirects to /dashboard
func TestHandleGitHubOAuthCallbackStateMismatchNewUserAndExistingUserLinking(testingInstance *testing.T) {
	returnVerifiedEmail := true

	mockGitHubServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "mock-github-access-token",
				"token_type":   "bearer",
			})
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(GitHubUserProfilePayload{
				ID:    987654321,
				Login: "octocat",
			})
		case "/user/emails":
			w.Header().Set("Content-Type", "application/json")
			if !returnVerifiedEmail {
				_ = json.NewEncoder(w).Encode([]GitHubUserEmailEntry{
					{Email: "unverified@github.com", Primary: true, Verified: false},
				})
				return
			}
			_ = json.NewEncoder(w).Encode([]GitHubUserEmailEntry{
				{Email: "secondary@github.com", Primary: false, Verified: true},
				{Email: "primary-octocat@github.com", Primary: true, Verified: true},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockGitHubServer.Close()

	oauthConfig := &oauth2.Config{
		ClientID:     "github-client-id",
		ClientSecret: "github-client-secret",
		RedirectURL:  "http://localhost:3000/api/auth/oauth/github/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  mockGitHubServer.URL + "/login/oauth/authorize",
			TokenURL: mockGitHubServer.URL + "/login/oauth/access_token",
		},
	}

	mockAccountRepo := &mockAccountRepository{}
	mockSessionRepo := &mockUserSessionRepository{}

	githubHandler, _ := NewGitHubOAuthHandler(
		mockAccountRepo,
		mockSessionRepo,
		oauthConfig,
		"jwt-secret-github",
		"http://localhost:3000",
		false,
	)
	githubHandler.userEndpointURL = mockGitHubServer.URL + "/user"
	githubHandler.userEmailsEndpointURL = mockGitHubServer.URL + "/user/emails"
	githubHandler.httpClient = mockGitHubServer.Client()

	// 1. State mismatch -> redirects to /?error=oauth_state_invalid
	mismatchReq := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?state=wrong&code=abc", nil)
	mismatchReq.AddCookie(&http.Cookie{Name: auth.OAuthStateCookieName, Value: "expected"})
	mismatchRec := httptest.NewRecorder()
	githubHandler.HandleGitHubOAuthCallback(mismatchRec, mismatchReq)

	if mismatchRec.Header().Get("Location") != "http://localhost:3000/?error=oauth_state_invalid" {
		testingInstance.Fatalf("expected redirect to /?error=oauth_state_invalid, got %q", mismatchRec.Header().Get("Location"))
	}

	// 2. Unverified email -> redirects to /?error=oauth_email_unverified
	returnVerifiedEmail = false
	validState := "valid-github-state-token"
	unverifiedReq := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?state="+validState+"&code=code-unverified", nil)
	unverifiedReq.AddCookie(&http.Cookie{Name: auth.OAuthStateCookieName, Value: validState})
	unverifiedRec := httptest.NewRecorder()
	githubHandler.HandleGitHubOAuthCallback(unverifiedRec, unverifiedReq)

	if unverifiedRec.Header().Get("Location") != "http://localhost:3000/?error=oauth_email_unverified" {
		testingInstance.Fatalf("expected redirect to /?error=oauth_email_unverified, got %q", unverifiedRec.Header().Get("Location"))
	}

	// 3. New GitHub user (Consented: false) -> creates User & Account and redirects to /dashboard?onboarding=consent
	returnVerifiedEmail = true
	newUserReq := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?state="+validState+"&code=code-new-user", nil)
	newUserReq.AddCookie(&http.Cookie{Name: auth.OAuthStateCookieName, Value: validState})
	newUserRec := httptest.NewRecorder()
	githubHandler.HandleGitHubOAuthCallback(newUserRec, newUserReq)

	if newUserRec.Code != http.StatusSeeOther {
		testingInstance.Fatalf("expected 303 SeeOther redirect, got %d", newUserRec.Code)
	}

	if newUserRec.Header().Get("Location") != "http://localhost:3000/dashboard?onboarding=consent" {
		testingInstance.Errorf("expected redirect to /dashboard?onboarding=consent, got %q", newUserRec.Header().Get("Location"))
	}

	if mockAccountRepo.createdUser == nil || mockAccountRepo.createdAccount == nil || mockAccountRepo.createdAccount.Provider != "github" || mockAccountRepo.createdAccount.ProviderUserID != "987654321" {
		testingInstance.Fatalf("expected GitHub Account & User to be created, got account=%+v", mockAccountRepo.createdAccount)
	}

	// 4. Existing user by email (Consented: true) -> links GitHub Account and redirects to /dashboard
	mockAccountRepo.userByEmail = &models.User{
		ID:        "existing-github-user-uuid",
		Email:     "primary-octocat@github.com",
		Consented: true,
		Verified:  false,
	}

	linkReq := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/github/callback?state="+validState+"&code=code-link-user", nil)
	linkReq.AddCookie(&http.Cookie{Name: auth.OAuthStateCookieName, Value: validState})
	linkRec := httptest.NewRecorder()
	githubHandler.HandleGitHubOAuthCallback(linkRec, linkReq)

	if linkRec.Header().Get("Location") != "http://localhost:3000/dashboard" {
		testingInstance.Errorf("expected redirect to /dashboard for consented user, got %q", linkRec.Header().Get("Location"))
	}

	if mockAccountRepo.linkedAccount == nil || mockAccountRepo.linkedAccount.UserID != "existing-github-user-uuid" || mockAccountRepo.linkedAccount.Provider != "github" {
		testingInstance.Errorf("expected LinkAccountToExistingUser to link existing-github-user-uuid, got %+v", mockAccountRepo.linkedAccount)
	}
}
