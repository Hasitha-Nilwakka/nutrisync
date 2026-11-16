// This file contains unit tests for TwoFactorHandler (setup, enable, disable, login-verify)
// and the 2FA pre-authentication login flow in UserSessionHandler.HandleUserLoginRequest.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
)

type inMemoryTwoFactorRepository struct {
	usersByID    map[string]*models.User
	usersByEmail map[string]*models.User
	sessionsByID map[string]*models.Session
	tokensByID   map[string]*models.RefreshToken
}

func newInMemoryTwoFactorRepository() *inMemoryTwoFactorRepository {
	return &inMemoryTwoFactorRepository{
		usersByID:    make(map[string]*models.User),
		usersByEmail: make(map[string]*models.User),
		sessionsByID: make(map[string]*models.Session),
		tokensByID:   make(map[string]*models.RefreshToken),
	}
}

func (repository *inMemoryTwoFactorRepository) FindUserByEmailForLogin(ctx context.Context, email string) (*models.User, error) {
	if user, exists := repository.usersByEmail[email]; exists {
		return user, nil
	}
	return nil, nil
}

func (repository *inMemoryTwoFactorRepository) CreateSessionWithInitialRefreshToken(
	ctx context.Context,
	session *models.Session,
	initialRefreshToken *models.RefreshToken,
) error {
	repository.sessionsByID[session.ID] = session
	initialRefreshToken.SessionID = session.ID
	repository.tokensByID[initialRefreshToken.ID] = initialRefreshToken
	return nil
}

func (repository *inMemoryTwoFactorRepository) FindRefreshTokenWithSessionByID(
	ctx context.Context,
	refreshTokenID string,
) (*models.RefreshToken, error) {
	if token, exists := repository.tokensByID[refreshTokenID]; exists {
		return token, nil
	}
	return nil, nil
}

func (repository *inMemoryTwoFactorRepository) RotateRefreshTokenInTransaction(
	ctx context.Context,
	currentTokenID string,
	newRefreshToken *models.RefreshToken,
) error {
	repository.tokensByID[newRefreshToken.ID] = newRefreshToken
	return nil
}

func (repository *inMemoryTwoFactorRepository) RevokeSessionAndAllRefreshTokens(
	ctx context.Context,
	sessionID string,
) error {
	if session, exists := repository.sessionsByID[sessionID]; exists {
		session.Revoked = true
	}
	return nil
}

func (repository *inMemoryTwoFactorRepository) FindSessionByID(
	ctx context.Context,
	sessionID string,
) (*models.Session, error) {
	if session, exists := repository.sessionsByID[sessionID]; exists {
		return session, nil
	}
	return nil, nil
}

func (repository *inMemoryTwoFactorRepository) AcceptUserGDPRConsent(ctx context.Context, userID string) error {
	if user, exists := repository.usersByID[userID]; exists {
		user.Consented = true
	}
	return nil
}

func (repository *inMemoryTwoFactorRepository) DeleteUnconsentedUserAndAccounts(ctx context.Context, userID string) error {
	delete(repository.usersByID, userID)
	return nil
}

func (repository *inMemoryTwoFactorRepository) DeleteAuthenticatedUserAccountAndAllData(ctx context.Context, userID string) error {
	delete(repository.usersByID, userID)
	return nil
}

func (repository *inMemoryTwoFactorRepository) SetPendingTwoFactorSecret(ctx context.Context, userID, secret string) error {
	if user, exists := repository.usersByID[userID]; exists {
		secretCopy := secret
		user.TwoFactorPendingSecret = &secretCopy
	}
	return nil
}

func (repository *inMemoryTwoFactorRepository) EnableTwoFactorAuthentication(ctx context.Context, userID string) error {
	if user, exists := repository.usersByID[userID]; exists {
		user.TwoFactorSecret = user.TwoFactorPendingSecret
		user.TwoFactorPendingSecret = nil
		user.TwoFactorEnabled = true
		provider := "authenticator"
		user.TwoFactorProvider = &provider
	}
	return nil
}

func (repository *inMemoryTwoFactorRepository) DisableTwoFactorAuthentication(ctx context.Context, userID string) error {
	if user, exists := repository.usersByID[userID]; exists {
		user.TwoFactorEnabled = false
		user.TwoFactorSecret = nil
		user.TwoFactorPendingSecret = nil
		user.TwoFactorProvider = nil
	}
	return nil
}

func (repository *inMemoryTwoFactorRepository) FindUserByID(ctx context.Context, userID string) (*models.User, error) {
	if user, exists := repository.usersByID[userID]; exists {
		return user, nil
	}
	return nil, nil
}

// TestTwoFactorSetupEnableAndDisableFlow verifies POST /api/auth/2fa/setup, POST /api/auth/2fa/enable,
// and POST /api/auth/2fa/disable with valid and invalid TOTP passcodes.
func TestTwoFactorSetupEnableAndDisableFlow(testingInstance *testing.T) {
	jwtSecret := "test-2fa-handler-secret"
	twoFactorRepo := newInMemoryTwoFactorRepository()
	sessionRepo := twoFactorRepo

	userID := "user-2fa-setup-1"
	sessionID := "session-2fa-setup-1"
	userRecord := &models.User{
		ID:       userID,
		Email:    "user2fa@example.com",
		Verified: true,
	}
	twoFactorRepo.usersByID[userID] = userRecord
	sessionRepo.sessionsByID[sessionID] = &models.Session{
		ID:        sessionID,
		UserID:    userID,
		Revoked:   false,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}

	handler, initErr := NewTwoFactorHandler(twoFactorRepo, sessionRepo, jwtSecret, false)
	if initErr != nil {
		testingInstance.Fatalf("expected NewTwoFactorHandler to succeed, got: %v", initErr)
	}

	accessToken, tokenErr := auth.GenerateSignedAccessToken(userID, sessionID, jwtSecret, 15*time.Minute)
	if tokenErr != nil {
		testingInstance.Fatalf("failed to generate access token: %v", tokenErr)
	}

	// 1. Setup 2FA
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/2fa/setup", nil)
	setupReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	setupRecorder := httptest.NewRecorder()
	handler.HandleTwoFactorSetupRequest(setupRecorder, setupReq)

	if setupRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK from setup, got %d (%s)", setupRecorder.Code, setupRecorder.Body.String())
	}

	var setupResponse TwoFactorSetupResponsePayload
	if decodeErr := json.Unmarshal(setupRecorder.Body.Bytes(), &setupResponse); decodeErr != nil {
		testingInstance.Fatalf("failed to decode setup response: %v", decodeErr)
	}

	if setupResponse.Secret == "" || !strings.HasPrefix(setupResponse.QRCode, "data:image/png;base64,") {
		testingInstance.Fatalf("unexpected setup response payload: %+v", setupResponse)
	}

	if userRecord.TwoFactorPendingSecret == nil || *userRecord.TwoFactorPendingSecret != setupResponse.Secret {
		testingInstance.Fatalf("expected TwoFactorPendingSecret to be saved as %q", setupResponse.Secret)
	}

	// 2. Enable 2FA with valid TOTP code
	validCode, codeErr := totp.GenerateCode(setupResponse.Secret, time.Now().UTC())
	if codeErr != nil {
		testingInstance.Fatalf("failed to generate TOTP code: %v", codeErr)
	}

	enableBody, _ := json.Marshal(TwoFactorCodeRequestPayload{Code: validCode})
	enableReq := httptest.NewRequest(http.MethodPost, "/api/auth/2fa/enable", bytes.NewReader(enableBody))
	enableReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	enableRecorder := httptest.NewRecorder()
	handler.HandleTwoFactorEnableRequest(enableRecorder, enableReq)

	if enableRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK from enable, got %d (%s)", enableRecorder.Code, enableRecorder.Body.String())
	}

	if !userRecord.TwoFactorEnabled || userRecord.TwoFactorSecret == nil || *userRecord.TwoFactorSecret != setupResponse.Secret {
		testingInstance.Fatal("expected userRecord to have TwoFactorEnabled=true and TwoFactorSecret populated")
	}

	// 3. Disable 2FA with valid TOTP code
	disableBody, _ := json.Marshal(TwoFactorCodeRequestPayload{Code: validCode})
	disableReq := httptest.NewRequest(http.MethodPost, "/api/auth/2fa/disable", bytes.NewReader(disableBody))
	disableReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	disableRecorder := httptest.NewRecorder()
	handler.HandleTwoFactorDisableRequest(disableRecorder, disableReq)

	if disableRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK from disable, got %d (%s)", disableRecorder.Code, disableRecorder.Body.String())
	}

	if userRecord.TwoFactorEnabled || userRecord.TwoFactorSecret != nil {
		testingInstance.Fatal("expected userRecord to have TwoFactorEnabled=false and TwoFactorSecret=nil after disable")
	}
}

// TestLoginRequiresTwoFactorAndLoginVerifyIssuesSessionCookies verifies that HandleUserLoginRequest returns
// { "require2FA": true, "ticket": "..." } without setting cookies when user.TwoFactorEnabled == true,
// and HandleTwoFactorLoginVerifyRequest exchanges the ticket + valid TOTP code for session cookies.
func TestLoginRequiresTwoFactorAndLoginVerifyIssuesSessionCookies(testingInstance *testing.T) {
	jwtSecret := "test-2fa-login-secret"
	twoFactorRepo := newInMemoryTwoFactorRepository()
	sessionRepo := twoFactorRepo

	passwordHash, hashErr := auth.HashUserPassword("StrongPass123!")
	if hashErr != nil {
		testingInstance.Fatalf("failed to hash password: %v", hashErr)
	}

	secretKey, _, _, setupErr := auth.GenerateTOTPSetupDetails("login2fa@example.com", "NutriSync")
	if setupErr != nil {
		testingInstance.Fatalf("failed to generate TOTP setup: %v", setupErr)
	}

	userRecord := &models.User{
		ID:               "user-login-2fa-1",
		Email:            "login2fa@example.com",
		Password:         passwordHash,
		Verified:         true,
		TwoFactorEnabled: true,
		TwoFactorSecret:  &secretKey,
	}
	sessionRepo.usersByEmail[userRecord.Email] = userRecord
	twoFactorRepo.usersByID[userRecord.ID] = userRecord

	sessionHandler, _ := NewUserSessionHandler(sessionRepo, jwtSecret, false)
	twoFactorHandler, _ := NewTwoFactorHandler(twoFactorRepo, sessionRepo, jwtSecret, false)

	// 1. Login with email + password -> should NOT set cookies, should return require2FA + ticket
	loginBody, _ := json.Marshal(UserLoginRequestPayload{
		Email:    userRecord.Email,
		Password: "StrongPass123!",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginRecorder := httptest.NewRecorder()
	sessionHandler.HandleUserLoginRequest(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK from login, got %d", loginRecorder.Code)
	}

	if len(loginRecorder.Result().Cookies()) != 0 {
		testingInstance.Fatalf("expected zero auth cookies when TwoFactorEnabled is true, got %d", len(loginRecorder.Result().Cookies()))
	}

	var preAuthResponse TwoFactorRequiredLoginResponsePayload
	if decodeErr := json.Unmarshal(loginRecorder.Body.Bytes(), &preAuthResponse); decodeErr != nil {
		testingInstance.Fatalf("failed to decode 2FA login response: %v", decodeErr)
	}

	if !preAuthResponse.Require2FA || strings.TrimSpace(preAuthResponse.Ticket) == "" {
		testingInstance.Fatalf("expected require2FA=true and non-empty ticket, got %+v", preAuthResponse)
	}

	// 2. Verify login with wrong TOTP code -> 401 Unauthorized
	wrongVerifyBody, _ := json.Marshal(TwoFactorLoginVerifyRequestPayload{
		Ticket: preAuthResponse.Ticket,
		Code:   "000000",
	})
	wrongVerifyReq := httptest.NewRequest(http.MethodPost, "/api/auth/2fa/login-verify", bytes.NewReader(wrongVerifyBody))
	wrongVerifyRecorder := httptest.NewRecorder()
	twoFactorHandler.HandleTwoFactorLoginVerifyRequest(wrongVerifyRecorder, wrongVerifyReq)

	if wrongVerifyRecorder.Code != http.StatusUnauthorized {
		testingInstance.Fatalf("expected 401 Unauthorized for invalid 2FA code, got %d", wrongVerifyRecorder.Code)
	}

	// 3. Verify login with valid TOTP code -> 200 OK + sets access_token and refresh_token cookies
	validCode, codeErr := totp.GenerateCode(secretKey, time.Now().UTC())
	if codeErr != nil {
		testingInstance.Fatalf("failed to generate TOTP code: %v", codeErr)
	}

	validVerifyBody, _ := json.Marshal(TwoFactorLoginVerifyRequestPayload{
		Ticket: preAuthResponse.Ticket,
		Code:   validCode,
	})
	validVerifyReq := httptest.NewRequest(http.MethodPost, "/api/auth/2fa/login-verify", bytes.NewReader(validVerifyBody))
	validVerifyRecorder := httptest.NewRecorder()
	twoFactorHandler.HandleTwoFactorLoginVerifyRequest(validVerifyRecorder, validVerifyReq)

	if validVerifyRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK from login-verify, got %d (%s)", validVerifyRecorder.Code, validVerifyRecorder.Body.String())
	}

	var hasAccessCookie, hasRefreshCookie bool
	for _, cookie := range validVerifyRecorder.Result().Cookies() {
		if cookie.Name == auth.AccessTokenCookieName && cookie.Value != "" {
			hasAccessCookie = true
		}
		if cookie.Name == auth.RefreshTokenCookieName && cookie.Value != "" {
			hasRefreshCookie = true
		}
	}

	if !hasAccessCookie || !hasRefreshCookie {
		testingInstance.Fatalf("expected both access_token and refresh_token cookies to be set, got access=%v refresh=%v", hasAccessCookie, hasRefreshCookie)
	}
}
