// This file contains unit tests for SetAuthenticationCookies and ClearAuthenticationCookies in auth/auth_cookies.go.
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSetAuthenticationCookiesSetsAccessAndRefreshTokenCookiesWithExpectedAttributes verifies that
// SetAuthenticationCookies writes both cookies with HttpOnly, SameSiteLaxMode, Path="/", Secure flag, and expected MaxAge.
func TestSetAuthenticationCookiesSetsAccessAndRefreshTokenCookiesWithExpectedAttributes(testingInstance *testing.T) {
	responseRecorder := httptest.NewRecorder()

	SetAuthenticationCookies(responseRecorder, "jwt-access-token", "id123.secret456", true)

	cookies := responseRecorder.Result().Cookies()
	if len(cookies) != 2 {
		testingInstance.Fatalf("expected 2 cookies to be set, got %d", len(cookies))
	}

	cookiesByName := make(map[string]*http.Cookie, len(cookies))
	for _, cookie := range cookies {
		cookiesByName[cookie.Name] = cookie
	}

	accessCookie := cookiesByName[AccessTokenCookieName]
	if accessCookie == nil {
		testingInstance.Fatalf("expected %q cookie to be present", AccessTokenCookieName)
	}
	if accessCookie.Value != "jwt-access-token" || accessCookie.MaxAge != AccessTokenMaxAgeSeconds || !accessCookie.HttpOnly || !accessCookie.Secure || accessCookie.SameSite != http.SameSiteLaxMode || accessCookie.Path != "/" {
		testingInstance.Errorf("unexpected access_token cookie attributes: %+v", accessCookie)
	}

	refreshCookie := cookiesByName[RefreshTokenCookieName]
	if refreshCookie == nil {
		testingInstance.Fatalf("expected %q cookie to be present", RefreshTokenCookieName)
	}
	if refreshCookie.Value != "id123.secret456" || refreshCookie.MaxAge != RefreshTokenMaxAgeSeconds || !refreshCookie.HttpOnly || !refreshCookie.Secure || refreshCookie.SameSite != http.SameSiteLaxMode || refreshCookie.Path != "/" {
		testingInstance.Errorf("unexpected refresh_token cookie attributes: %+v", refreshCookie)
	}
}

// TestClearAuthenticationCookiesExpiresBothCookies verifies that ClearAuthenticationCookies sets empty values and MaxAge -1.
func TestClearAuthenticationCookiesExpiresBothCookies(testingInstance *testing.T) {
	responseRecorder := httptest.NewRecorder()

	ClearAuthenticationCookies(responseRecorder, false)

	cookies := responseRecorder.Result().Cookies()
	if len(cookies) != 2 {
		testingInstance.Fatalf("expected 2 cookies to be cleared, got %d", len(cookies))
	}

	for _, cookie := range cookies {
		if cookie.Value != "" || cookie.MaxAge != -1 || !cookie.HttpOnly || cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
			testingInstance.Errorf("expected cleared cookie attributes for %q, got %+v", cookie.Name, cookie)
		}
	}
}
