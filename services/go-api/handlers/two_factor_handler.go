// This file implements HTTP handlers for optional TOTP Two-Factor Authentication (2FA):
// setup (POST /api/auth/2fa/setup), enable (POST /api/auth/2fa/enable),
// disable (POST /api/auth/2fa/disable), and login verification (POST /api/auth/2fa/login-verify).
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
	"nutrysync/go-api/repositories"
)

const twoFactorIssuerName = "NutriSync"

// TwoFactorSetupResponsePayload represents the JSON body returned by POST /api/auth/2fa/setup.
type TwoFactorSetupResponsePayload struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
	QRCode string `json:"qrCode"`
}

// TwoFactorCodeRequestPayload represents the JSON body accepted by POST /api/auth/2fa/enable and POST /api/auth/2fa/disable.
type TwoFactorCodeRequestPayload struct {
	Code string `json:"code"`
}

// TwoFactorLoginVerifyRequestPayload represents the JSON body accepted by POST /api/auth/2fa/login-verify.
type TwoFactorLoginVerifyRequestPayload struct {
	Ticket string `json:"ticket"`
	Code   string `json:"code"`
}

// TwoFactorHandler handles TOTP two-factor setup, activation, deactivation, and login verification endpoints.
type TwoFactorHandler struct {
	twoFactorRepository repositories.TwoFactorRepository
	sessionRepository   repositories.UserSessionRepository
	jwtSecret           string
	isProduction        bool
}

// NewTwoFactorHandler constructs a TwoFactorHandler after validating its dependencies and JWT secret.
func NewTwoFactorHandler(
	twoFactorRepository repositories.TwoFactorRepository,
	sessionRepository repositories.UserSessionRepository,
	jwtSecret string,
	isProduction bool,
) (*TwoFactorHandler, error) {
	if twoFactorRepository == nil {
		return nil, errors.New("two-factor repository must not be nil")
	}

	if sessionRepository == nil {
		return nil, errors.New("session repository must not be nil")
	}

	trimmedJWTSecret := strings.TrimSpace(jwtSecret)
	if trimmedJWTSecret == "" {
		return nil, errors.New("JWT secret must not be empty")
	}

	return &TwoFactorHandler{
		twoFactorRepository: twoFactorRepository,
		sessionRepository:   sessionRepository,
		jwtSecret:           trimmedJWTSecret,
		isProduction:        isProduction,
	}, nil
}

// authenticateUserViaAccessTokenCookie validates the access_token cookie and resolves the authenticated models.User.
func (handler *TwoFactorHandler) authenticateUserViaAccessTokenCookie(request *http.Request) (*models.User, error) {
	accessCookie, cookieError := request.Cookie(auth.AccessTokenCookieName)
	if cookieError != nil || accessCookie == nil {
		return nil, errors.New("missing access_token cookie")
	}

	claims, verifyError := auth.VerifyAndParseAccessToken(accessCookie.Value, handler.jwtSecret)
	if verifyError != nil || claims == nil {
		return nil, errors.New("invalid access_token cookie")
	}

	sessionRecord, sessionLookupError := handler.sessionRepository.FindSessionByID(request.Context(), claims.SessionID)
	if sessionLookupError != nil {
		return nil, sessionLookupError
	}

	if sessionRecord != nil {
		currentTimeUTC := time.Now().UTC()
		if sessionRecord.UserID != claims.Subject || sessionRecord.Revoked || !currentTimeUTC.Before(sessionRecord.ExpiresAt) {
			return nil, errors.New("session is expired or revoked")
		}
	}

	userRecord, userLookupError := handler.twoFactorRepository.FindUserByID(request.Context(), claims.Subject)
	if userLookupError != nil || userRecord == nil {
		return nil, errors.New("authenticated user not found")
	}

	return userRecord, nil
}

// HandleTwoFactorSetupRequest handles POST /api/auth/2fa/setup:
// authenticates the user via access_token cookie, generates a TOTP secret and QR code PNG data URL,
// saves the secret in TwoFactorPendingSecret, and returns { "secret", "uri", "qrCode" }.
func (handler *TwoFactorHandler) HandleTwoFactorSetupRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only POST method is supported",
		})
		return
	}

	userRecord, authError := handler.authenticateUserViaAccessTokenCookie(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	accountEmail := strings.TrimSpace(userRecord.Email)
	if accountEmail == "" {
		accountEmail = userRecord.ID
	}

	secretKey, otpauthURI, qrCodeBase64PNG, setupError := auth.GenerateTOTPSetupDetails(accountEmail, twoFactorIssuerName)
	if setupError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "internal server error",
		})
		return
	}

	if saveError := handler.twoFactorRepository.SetPendingTwoFactorSecret(request.Context(), userRecord.ID, secretKey); saveError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "internal server error",
		})
		return
	}

	WriteJSONResponse(responseWriter, http.StatusOK, TwoFactorSetupResponsePayload{
		Secret: secretKey,
		URI:    otpauthURI,
		QRCode: qrCodeBase64PNG,
	})
}

// HandleTwoFactorEnableRequest handles POST /api/auth/2fa/enable:
// authenticates the user via access_token cookie, validates the submitted 6-digit TOTP code against TwoFactorPendingSecret,
// promotes TwoFactorPendingSecret to TwoFactorSecret with TwoFactorEnabled = true, and responds 200 OK.
func (handler *TwoFactorHandler) HandleTwoFactorEnableRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only POST method is supported",
		})
		return
	}

	userRecord, authError := handler.authenticateUserViaAccessTokenCookie(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	var codePayload TwoFactorCodeRequestPayload
	if decodeError := json.NewDecoder(request.Body).Decode(&codePayload); decodeError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "invalid JSON request body",
		})
		return
	}

	if userRecord.TwoFactorPendingSecret == nil || strings.TrimSpace(*userRecord.TwoFactorPendingSecret) == "" {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "two-factor setup has not been initiated",
		})
		return
	}

	if !auth.ValidateTOTPPasscode(codePayload.Code, *userRecord.TwoFactorPendingSecret) {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "invalid two-factor verification code",
		})
		return
	}

	if enableError := handler.twoFactorRepository.EnableTwoFactorAuthentication(request.Context(), userRecord.ID); enableError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "internal server error",
		})
		return
	}

	WriteJSONResponse(responseWriter, http.StatusOK, MessageOnlyResponsePayload{
		Message: "Two-factor authentication enabled successfully",
	})
}

// HandleTwoFactorDisableRequest handles POST /api/auth/2fa/disable:
// authenticates the user via access_token cookie, validates the submitted 6-digit TOTP code against TwoFactorSecret,
// disables 2FA, and responds 200 OK.
func (handler *TwoFactorHandler) HandleTwoFactorDisableRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only POST method is supported",
		})
		return
	}

	userRecord, authError := handler.authenticateUserViaAccessTokenCookie(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	var codePayload TwoFactorCodeRequestPayload
	if decodeError := json.NewDecoder(request.Body).Decode(&codePayload); decodeError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "invalid JSON request body",
		})
		return
	}

	if !userRecord.TwoFactorEnabled || userRecord.TwoFactorSecret == nil || strings.TrimSpace(*userRecord.TwoFactorSecret) == "" {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "two-factor authentication is not enabled",
		})
		return
	}

	if !auth.ValidateTOTPPasscode(codePayload.Code, *userRecord.TwoFactorSecret) {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "invalid two-factor verification code",
		})
		return
	}

	if disableError := handler.twoFactorRepository.DisableTwoFactorAuthentication(request.Context(), userRecord.ID); disableError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "internal server error",
		})
		return
	}

	WriteJSONResponse(responseWriter, http.StatusOK, MessageOnlyResponsePayload{
		Message: "Two-factor authentication disabled",
	})
}

// HandleTwoFactorLoginVerifyRequest handles POST /api/auth/2fa/login-verify:
// validates the 5-minute 2fa_preauth JWT ticket and 6-digit TOTP code against user.TwoFactorSecret,
// creates a Session + RefreshToken, sets authentication cookies, and responds 200 OK.
func (handler *TwoFactorHandler) HandleTwoFactorLoginVerifyRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only POST method is supported",
		})
		return
	}

	var verifyPayload TwoFactorLoginVerifyRequestPayload
	if decodeError := json.NewDecoder(request.Body).Decode(&verifyPayload); decodeError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "invalid JSON request body",
		})
		return
	}

	userID, ticketError := auth.VerifyTwoFactorTicket(verifyPayload.Ticket, handler.jwtSecret)
	if ticketError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "invalid or expired two-factor ticket",
		})
		return
	}

	userRecord, lookupError := handler.twoFactorRepository.FindUserByID(request.Context(), userID)
	if lookupError != nil ||
		userRecord == nil ||
		!userRecord.TwoFactorEnabled ||
		userRecord.TwoFactorSecret == nil ||
		strings.TrimSpace(*userRecord.TwoFactorSecret) == "" {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	if !auth.ValidateTOTPPasscode(verifyPayload.Code, *userRecord.TwoFactorSecret) {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "invalid two-factor verification code",
		})
		return
	}

	rawSecret, secretHash, secretError := auth.GenerateRefreshTokenSecretAndHash()
	sessionID, sessionUUIDError := generateRandomUUIDv4()
	refreshTokenID, tokenUUIDError := generateRandomUUIDv4()
	if secretError != nil || sessionUUIDError != nil || tokenUUIDError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "internal server error",
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

	if createError := handler.sessionRepository.CreateSessionWithInitialRefreshToken(
		request.Context(),
		sessionRecord,
		initialRefreshToken,
	); createError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "internal server error",
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
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "internal server error",
		})
		return
	}

	compositeRefreshToken := auth.FormatCompositeRefreshTokenCookieValue(initialRefreshToken.ID, rawSecret)
	auth.SetAuthenticationCookies(responseWriter, accessToken, compositeRefreshToken, handler.isProduction)

	WriteJSONResponse(responseWriter, http.StatusOK, MessageOnlyResponsePayload{
		Message: "Login successful",
	})
}
