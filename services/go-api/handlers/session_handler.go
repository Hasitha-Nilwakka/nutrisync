// This file implements HTTP handlers for user login (POST /api/auth/login),
// refresh token rotation with reuse detection (POST /api/auth/refresh),
// user logout (POST /api/auth/logout), and active session verification (GET /api/auth/session).
package handlers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
	"nutrysync/go-api/repositories"
)

const (
	accessTokenLifetimeDuration  = 15 * time.Minute
	refreshTokenLifetimeDuration = 7 * 24 * time.Hour
)

// UserLoginRequestPayload represents the JSON body accepted by POST /api/auth/login.
type UserLoginRequestPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ActiveSessionResponsePayload represents the JSON body returned by GET /api/auth/session when a session is valid.
type ActiveSessionResponsePayload struct {
	UserID    string `json:"userId"`
	SessionID string `json:"sessionId"`
}

// MessageOnlyResponsePayload represents simple JSON message responses returned by refresh and logout endpoints.
type MessageOnlyResponsePayload struct {
	Message string `json:"message"`
}

// UserSessionHandler handles login, refresh token rotation, logout, and active session verification HTTP requests.
type UserSessionHandler struct {
	sessionRepository repositories.UserSessionRepository
	jwtSecret         string
	isProduction      bool
}

// NewUserSessionHandler constructs a UserSessionHandler after validating sessionRepository and jwtSecret.
func NewUserSessionHandler(
	sessionRepository repositories.UserSessionRepository,
	jwtSecret string,
	isProduction bool,
) (*UserSessionHandler, error) {
	if sessionRepository == nil {
		return nil, errors.New("session repository must not be nil")
	}

	trimmedJWTSecret := strings.TrimSpace(jwtSecret)
	if trimmedJWTSecret == "" {
		return nil, errors.New("JWT secret must not be empty")
	}

	return &UserSessionHandler{
		sessionRepository: sessionRepository,
		jwtSecret:         trimmedJWTSecret,
		isProduction:      isProduction,
	}, nil
}

// generateRandomUUIDv4 generates a version-4 UUID string using crypto/rand so Session and RefreshToken IDs
// are always populated prior to cookie formatting even before database-default UUID assignment.
func generateRandomUUIDv4() (string, error) {
	uuidBytes := make([]byte, 16)
	if _, readError := rand.Read(uuidBytes); readError != nil {
		return "", readError
	}

	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x40
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		uuidBytes[0:4],
		uuidBytes[4:6],
		uuidBytes[6:8],
		uuidBytes[8:10],
		uuidBytes[10:16],
	), nil
}

// extractClientIPAddress extracts the client IP address from X-Forwarded-For or RemoteAddr.
func extractClientIPAddress(request *http.Request) *string {
	if forwardedFor := strings.TrimSpace(request.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		firstIP := strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
		if firstIP != "" {
			return &firstIP
		}
	}

	host, _, splitError := net.SplitHostPort(strings.TrimSpace(request.RemoteAddr))
	if splitError == nil && host != "" {
		return &host
	}

	trimmedRemote := strings.TrimSpace(request.RemoteAddr)
	if trimmedRemote != "" {
		return &trimmedRemote
	}

	return nil
}

// resolveUserPasswordHash returns the bcrypt hash stored on a models.User record,
// supporting both PasswordHash (*string) and Password (string).
func resolveUserPasswordHash(user *models.User) string {
	if user == nil {
		return ""
	}
	if user.PasswordHash != nil && strings.TrimSpace(*user.PasswordHash) != "" {
		return strings.TrimSpace(*user.PasswordHash)
	}
	return strings.TrimSpace(user.Password)
}

// HandleUserLoginRequest handles POST /api/auth/login:
// authenticates email & password, checks email verification, creates a 7-day Session + RefreshToken,
// signs a 15-minute access_token JWT, sets HttpOnly cookies, and responds with 200 OK.
func (handler *UserSessionHandler) HandleUserLoginRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Method Not Allowed",
			Message: "only POST method is supported",
		})
		return
	}

	var loginPayload UserLoginRequestPayload
	if decodeError := json.NewDecoder(request.Body).Decode(&loginPayload); decodeError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Validation Error",
			Message: "invalid JSON request body",
		})
		return
	}

	trimmedEmail := strings.TrimSpace(loginPayload.Email)
	if trimmedEmail == "" || strings.TrimSpace(loginPayload.Password) == "" {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Validation Error",
			Message: "email and password are required",
		})
		return
	}

	userRecord, lookupError := handler.sessionRepository.FindUserByEmailForLogin(request.Context(), trimmedEmail)
	if lookupError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Login failed",
			Message: "Internal server error",
		})
		return
	}

	passwordHash := resolveUserPasswordHash(userRecord)
	if userRecord == nil || passwordHash == "" {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Login failed",
			Message: "Invalid email or password",
		})
		return
	}

	if verifyError := auth.VerifyUserPasswordAgainstHash(loginPayload.Password, passwordHash); verifyError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Login failed",
			Message: "Invalid email or password",
		})
		return
	}

	if !userRecord.Verified && !userRecord.EmailVerified {
		WriteJSONResponse(responseWriter, http.StatusForbidden, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Verification pending",
			Message: "Verification pending, please check email",
		})
		return
	}

	rawSecret, secretHash, secretError := auth.GenerateRefreshTokenSecretAndHash()
	if secretError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Login failed",
			Message: "Internal server error",
		})
		return
	}

	sessionID, sessionUUIDError := generateRandomUUIDv4()
	refreshTokenID, tokenUUIDError := generateRandomUUIDv4()
	if sessionUUIDError != nil || tokenUUIDError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Login failed",
			Message: "Internal server error",
		})
		return
	}

	currentTimeUTC := time.Now().UTC()
	expirationTimeUTC := currentTimeUTC.Add(refreshTokenLifetimeDuration)

	var userAgentPointer *string
	if userAgentHeader := strings.TrimSpace(request.UserAgent()); userAgentHeader != "" {
		userAgentPointer = &userAgentHeader
	}

	sessionRecord := &models.Session{
		ID:        sessionID,
		UserID:    userRecord.ID,
		UserAgent: userAgentPointer,
		IPAddress: extractClientIPAddress(request),
		Revoked:   false,
		ExpiresAt: expirationTimeUTC,
	}

	initialRefreshToken := &models.RefreshToken{
		ID:        refreshTokenID,
		SessionID: sessionID,
		TokenHash: secretHash,
		Revoked:   false,
		ExpiresAt: expirationTimeUTC,
	}

	if createError := handler.sessionRepository.CreateSessionWithInitialRefreshToken(request.Context(), sessionRecord, initialRefreshToken); createError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Login failed",
			Message: "Internal server error",
		})
		return
	}

	accessToken, accessTokenError := auth.GenerateSignedAccessToken(
		userRecord.ID,
		sessionRecord.ID,
		handler.jwtSecret,
		accessTokenLifetimeDuration,
	)
	if accessTokenError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Login failed",
			Message: "Internal server error",
		})
		return
	}

	compositeRefreshToken := auth.FormatCompositeRefreshTokenCookieValue(initialRefreshToken.ID, rawSecret)
	auth.SetAuthenticationCookies(responseWriter, accessToken, compositeRefreshToken, handler.isProduction)

	WriteJSONResponse(responseWriter, http.StatusOK, StandardAPIResponsePayload{
		Type:    "success",
		Title:   "Logged in",
		Message: "You are logged in",
	})
}

// HandleRefreshTokenRotationRequest handles POST /api/auth/refresh:
// validates the composite refresh_token cookie, performs reuse detection (revoking the entire session if a rotated/revoked token is replayed),
// rotates the refresh token in a database transaction, issues a new 15-minute access_token, and sets both cookies.
func (handler *UserSessionHandler) HandleRefreshTokenRotationRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only POST method is supported",
		})
		return
	}

	refreshCookie, cookieError := request.Cookie(auth.RefreshTokenCookieName)
	if cookieError != nil || refreshCookie == nil {
		auth.ClearAuthenticationCookies(responseWriter, handler.isProduction)
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	refreshTokenID, rawSecret, parseError := auth.ParseCompositeRefreshTokenCookieValue(refreshCookie.Value)
	if parseError != nil {
		auth.ClearAuthenticationCookies(responseWriter, handler.isProduction)
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	storedToken, lookupError := handler.sessionRepository.FindRefreshTokenWithSessionByID(request.Context(), refreshTokenID)
	if lookupError != nil || storedToken == nil {
		auth.ClearAuthenticationCookies(responseWriter, handler.isProduction)
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	if !auth.VerifyRefreshTokenSecretConstantTime(rawSecret, storedToken.TokenHash) {
		auth.ClearAuthenticationCookies(responseWriter, handler.isProduction)
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	// Reuse detection: if a previously rotated or revoked refresh token is presented again,
	// immediately revoke the entire session and all associated refresh tokens.
	if storedToken.Revoked || storedToken.ReplacedByTokenID != nil {
		_ = handler.sessionRepository.RevokeSessionAndAllRefreshTokens(request.Context(), storedToken.SessionID)
		auth.ClearAuthenticationCookies(responseWriter, handler.isProduction)
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "refresh token reuse detected",
		})
		return
	}

	currentTimeUTC := time.Now().UTC()
	if currentTimeUTC.After(storedToken.ExpiresAt) || storedToken.Session.Revoked || currentTimeUTC.After(storedToken.Session.ExpiresAt) {
		auth.ClearAuthenticationCookies(responseWriter, handler.isProduction)
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "session expired or revoked",
		})
		return
	}

	newRawSecret, newSecretHash, secretError := auth.GenerateRefreshTokenSecretAndHash()
	newTokenID, uuidError := generateRandomUUIDv4()
	if secretError != nil || uuidError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "internal server error",
		})
		return
	}

	newRefreshToken := &models.RefreshToken{
		ID:        newTokenID,
		SessionID: storedToken.SessionID,
		TokenHash: newSecretHash,
		Revoked:   false,
		ExpiresAt: currentTimeUTC.Add(refreshTokenLifetimeDuration),
	}

	if rotateError := handler.sessionRepository.RotateRefreshTokenInTransaction(request.Context(), storedToken.ID, newRefreshToken); rotateError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "internal server error",
		})
		return
	}

	newAccessToken, accessTokenError := auth.GenerateSignedAccessToken(
		storedToken.Session.UserID,
		storedToken.SessionID,
		handler.jwtSecret,
		accessTokenLifetimeDuration,
	)
	if accessTokenError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "internal server error",
		})
		return
	}

	newCompositeRefreshToken := auth.FormatCompositeRefreshTokenCookieValue(newRefreshToken.ID, newRawSecret)
	auth.SetAuthenticationCookies(responseWriter, newAccessToken, newCompositeRefreshToken, handler.isProduction)

	WriteJSONResponse(responseWriter, http.StatusOK, MessageOnlyResponsePayload{
		Message: "session refreshed",
	})
}

// HandleUserLogoutRequest handles POST /api/auth/logout:
// revokes the active session and its refresh tokens when a valid refresh_token cookie is present,
// and always clears the access_token and refresh_token cookies.
func (handler *UserSessionHandler) HandleUserLogoutRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only POST method is supported",
		})
		return
	}

	if refreshCookie, cookieError := request.Cookie(auth.RefreshTokenCookieName); cookieError == nil && refreshCookie != nil {
		if refreshTokenID, rawSecret, parseError := auth.ParseCompositeRefreshTokenCookieValue(refreshCookie.Value); parseError == nil {
			if storedToken, lookupError := handler.sessionRepository.FindRefreshTokenWithSessionByID(request.Context(), refreshTokenID); lookupError == nil && storedToken != nil {
				if auth.VerifyRefreshTokenSecretConstantTime(rawSecret, storedToken.TokenHash) {
					_ = handler.sessionRepository.RevokeSessionAndAllRefreshTokens(request.Context(), storedToken.SessionID)
				}
			}
		}
	}

	auth.ClearAuthenticationCookies(responseWriter, handler.isProduction)
	WriteJSONResponse(responseWriter, http.StatusOK, MessageOnlyResponsePayload{
		Message: "Logged out successfully",
	})
}

// HandleVerifyActiveSessionRequest handles GET /api/auth/session:
// verifies the access_token cookie JWT and ensures the backing Session row exists, belongs to the user,
// is not revoked, and has not expired.
func (handler *UserSessionHandler) HandleVerifyActiveSessionRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only GET method is supported",
		})
		return
	}

	accessCookie, cookieError := request.Cookie(auth.AccessTokenCookieName)
	if cookieError != nil || accessCookie == nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	claims, verifyError := auth.VerifyAndParseAccessToken(accessCookie.Value, handler.jwtSecret)
	if verifyError != nil || claims == nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	sessionRecord, lookupError := handler.sessionRepository.FindSessionByID(request.Context(), claims.SessionID)
	currentTimeUTC := time.Now().UTC()
	if lookupError != nil ||
		sessionRecord == nil ||
		sessionRecord.UserID != claims.Subject ||
		sessionRecord.Revoked ||
		!currentTimeUTC.Before(sessionRecord.ExpiresAt) {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	WriteJSONResponse(responseWriter, http.StatusOK, ActiveSessionResponsePayload{
		UserID:    sessionRecord.UserID,
		SessionID: sessionRecord.ID,
	})
}
