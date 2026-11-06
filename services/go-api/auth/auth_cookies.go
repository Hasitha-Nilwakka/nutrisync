// This file provides HTTP cookie helpers for setting and clearing the access_token
// and refresh_token authentication cookies on HTTP responses.
package auth

import (
	"net/http"
	"time"
)

const (
	// AccessTokenCookieName is the HTTP cookie name storing the signed JWT access token.
	AccessTokenCookieName = "access_token"
	// RefreshTokenCookieName is the HTTP cookie name storing the composite "<tokenID>.<rawSecret>" refresh token.
	RefreshTokenCookieName = "refresh_token"
	// AccessTokenMaxAgeSeconds defines the 15-minute lifetime (in seconds) of the access_token cookie.
	AccessTokenMaxAgeSeconds = 15 * 60
	// RefreshTokenMaxAgeSeconds defines the 7-day lifetime (in seconds) of the refresh_token cookie.
	RefreshTokenMaxAgeSeconds = 7 * 24 * 60 * 60
)

// SetAuthenticationCookies writes both the access_token and refresh_token HttpOnly cookies
// to responseWriter with SameSite=Lax, Path="/", Secure=isSecureCookie, and their respective MaxAge values.
func SetAuthenticationCookies(responseWriter http.ResponseWriter, accessToken, compositeRefreshToken string, isSecureCookie bool) {
	http.SetCookie(responseWriter, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		MaxAge:   AccessTokenMaxAgeSeconds,
		HttpOnly: true,
		Secure:   isSecureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(responseWriter, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    compositeRefreshToken,
		Path:     "/",
		MaxAge:   RefreshTokenMaxAgeSeconds,
		HttpOnly: true,
		Secure:   isSecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearAuthenticationCookies expires and clears both the access_token and refresh_token cookies
// on responseWriter by setting empty values, MaxAge=-1, and Expires=Unix(0, 0).
func ClearAuthenticationCookies(responseWriter http.ResponseWriter, isSecureCookie bool) {
	expiredTimestamp := time.Unix(0, 0)

	http.SetCookie(responseWriter, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  expiredTimestamp,
		HttpOnly: true,
		Secure:   isSecureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(responseWriter, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  expiredTimestamp,
		HttpOnly: true,
		Secure:   isSecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}
