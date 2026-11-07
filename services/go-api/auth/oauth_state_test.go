// This file contains unit tests for GenerateRandomOAuthState, SetOAuthStateCookie,
// and ValidateAndClearOAuthStateCookie in auth/oauth_state.go.
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGenerateRandomOAuthStateProducesUniqueSixtyFourCharHexStrings verifies random state generation.
func TestGenerateRandomOAuthStateProducesUniqueSixtyFourCharHexStrings(testingInstance *testing.T) {
	firstState, firstErr := GenerateRandomOAuthState()
	if firstErr != nil {
		testingInstance.Fatalf("expected GenerateRandomOAuthState to succeed, got: %v", firstErr)
	}

	if len(firstState) != 64 {
		testingInstance.Errorf("expected 64-character hex state, got length %d", len(firstState))
	}

	secondState, secondErr := GenerateRandomOAuthState()
	if secondErr != nil {
		testingInstance.Fatalf("expected second GenerateRandomOAuthState to succeed, got: %v", secondErr)
	}

	if firstState == secondState {
		testingInstance.Error("expected generated OAuth states to be unique")
	}
}

// TestSetAndValidateAndClearOAuthStateCookieRoundTrip verifies setting, reading, validating, and clearing oauth_state.
func TestSetAndValidateAndClearOAuthStateCookieRoundTrip(testingInstance *testing.T) {
	setRecorder := httptest.NewRecorder()
	expectedState := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

	SetOAuthStateCookie(setRecorder, expectedState, true)

	setCookies := setRecorder.Result().Cookies()
	if len(setCookies) != 1 {
		testingInstance.Fatalf("expected 1 cookie to be set, got %d", len(setCookies))
	}

	oauthCookie := setCookies[0]
	if oauthCookie.Name != OAuthStateCookieName || oauthCookie.Value != expectedState || oauthCookie.MaxAge != OAuthStateCookieMaxAgeSeconds || !oauthCookie.HttpOnly || !oauthCookie.Secure || oauthCookie.SameSite != http.SameSiteLaxMode || oauthCookie.Path != "/" {
		testingInstance.Errorf("unexpected oauth_state cookie attributes: %+v", oauthCookie)
	}

	// Validate and clear with matching state
	callbackRequest := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/google/callback?state="+expectedState, nil)
	callbackRequest.AddCookie(oauthCookie)
	clearRecorder := httptest.NewRecorder()

	returnedState, validateErr := ValidateAndClearOAuthStateCookie(clearRecorder, callbackRequest, true)
	if validateErr != nil {
		testingInstance.Fatalf("expected ValidateAndClearOAuthStateCookie to succeed, got: %v", validateErr)
	}

	if returnedState != expectedState {
		testingInstance.Errorf("expected state %q, got %q", expectedState, returnedState)
	}

	clearedCookies := clearRecorder.Result().Cookies()
	if len(clearedCookies) != 1 || clearedCookies[0].MaxAge != -1 || clearedCookies[0].Value != "" {
		testingInstance.Errorf("expected oauth_state cookie to be cleared with MaxAge=-1, got %+v", clearedCookies)
	}

	// Missing cookie should return error
	missingRequest := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/google/callback", nil)
	missingRecorder := httptest.NewRecorder()
	if _, err := ValidateAndClearOAuthStateCookie(missingRecorder, missingRequest, false); err == nil {
		testingInstance.Fatal("expected error when oauth_state cookie is missing")
	}
}
