// This file provides OAuth CSRF state generation and HttpOnly cookie helpers
// for initiating and validating OAuth 2.0 authorization code redirects.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	// OAuthStateCookieName is the HttpOnly cookie name storing the temporary OAuth CSRF state token.
	OAuthStateCookieName = "oauth_state"
	// OAuthStateCookieMaxAgeSeconds defines the 10-minute lifetime (in seconds) of the oauth_state cookie.
	OAuthStateCookieMaxAgeSeconds = 10 * 60
	oauthStateByteLength          = 32
)

// GenerateRandomOAuthState generates 32 cryptographically random bytes encoded as a 64-character hex string.
func GenerateRandomOAuthState() (string, error) {
	randomBytes := make([]byte, oauthStateByteLength)
	if _, readError := rand.Read(randomBytes); readError != nil {
		return "", readError
	}

	return hex.EncodeToString(randomBytes), nil
}

// SetOAuthStateCookie writes the oauth_state cookie with a 10-minute TTL, HttpOnly, SameSite=Lax, and Path="/".
func SetOAuthStateCookie(responseWriter http.ResponseWriter, state string, isProduction bool) {
	http.SetCookie(responseWriter, &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    strings.TrimSpace(state),
		Path:     "/",
		MaxAge:   OAuthStateCookieMaxAgeSeconds,
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	})
}

// ValidateAndClearOAuthStateCookie reads the oauth_state cookie from request, immediately clears it
// on responseWriter with MaxAge=-1, validates it against the "state" query parameter if present, and returns the state string.
func ValidateAndClearOAuthStateCookie(
	responseWriter http.ResponseWriter,
	request *http.Request,
	isProduction bool,
) (string, error) {
	http.SetCookie(responseWriter, &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
	})

	stateCookie, cookieError := request.Cookie(OAuthStateCookieName)
	if cookieError != nil || stateCookie == nil {
		return "", errors.New("missing oauth_state cookie")
	}

	cookieState := strings.TrimSpace(stateCookie.Value)
	if cookieState == "" {
		return "", errors.New("empty oauth_state cookie value")
	}

	if request.URL != nil {
		queryState := strings.TrimSpace(request.URL.Query().Get("state"))
		if queryState != "" && subtle.ConstantTimeCompare([]byte(cookieState), []byte(queryState)) != 1 {
			return "", errors.New("oauth state mismatch")
		}
	}

	return cookieState, nil
}
