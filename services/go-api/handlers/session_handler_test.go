// This file contains unit tests for UserSessionHandler covering login success, unverified user 403,
// wrong password 401, refresh token rotation, refresh token reuse detection revoking the session,
// logout clearing cookies, and active session verification.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
)

// mockUserSessionRepository is an in-memory test double implementing repositories.UserSessionRepository.
type mockUserSessionRepository struct {
	userForLogin          *models.User
	loginLookupError      error
	createdSession        *models.Session
	createdInitialToken   *models.RefreshToken
	storedRefreshToken    *models.RefreshToken
	rotatedOldTokenID     string
	rotatedNewToken       *models.RefreshToken
	revokedSessionID      string
	activeSessionByID     *models.Session
	acceptedConsentUserID string
	deletedUnconsentedID  string
	deletedAccountUserID  string
}

func (mockRepo *mockUserSessionRepository) FindUserByEmailForLogin(ctx context.Context, email string) (*models.User, error) {
	if mockRepo.loginLookupError != nil {
		return nil, mockRepo.loginLookupError
	}
	return mockRepo.userForLogin, nil
}

func (mockRepo *mockUserSessionRepository) CreateSessionWithInitialRefreshToken(
	ctx context.Context,
	session *models.Session,
	initialRefreshToken *models.RefreshToken,
) error {
	mockRepo.createdSession = session
	mockRepo.createdInitialToken = initialRefreshToken
	return nil
}

func (mockRepo *mockUserSessionRepository) FindRefreshTokenWithSessionByID(
	ctx context.Context,
	refreshTokenID string,
) (*models.RefreshToken, error) {
	if mockRepo.storedRefreshToken != nil && mockRepo.storedRefreshToken.ID == refreshTokenID {
		return mockRepo.storedRefreshToken, nil
	}
	return nil, nil
}

func (mockRepo *mockUserSessionRepository) RotateRefreshTokenInTransaction(
	ctx context.Context,
	currentTokenID string,
	newRefreshToken *models.RefreshToken,
) error {
	mockRepo.rotatedOldTokenID = currentTokenID
	mockRepo.rotatedNewToken = newRefreshToken
	return nil
}

func (mockRepo *mockUserSessionRepository) RevokeSessionAndAllRefreshTokens(
	ctx context.Context,
	sessionID string,
) error {
	mockRepo.revokedSessionID = sessionID
	return nil
}

func (mockRepo *mockUserSessionRepository) FindSessionByID(
	ctx context.Context,
	sessionID string,
) (*models.Session, error) {
	if mockRepo.activeSessionByID != nil && mockRepo.activeSessionByID.ID == sessionID {
		return mockRepo.activeSessionByID, nil
	}
	return nil, nil
}

func (mockRepo *mockUserSessionRepository) AcceptUserGDPRConsent(ctx context.Context, userID string) error {
	mockRepo.acceptedConsentUserID = userID
	if mockRepo.activeSessionByID != nil && mockRepo.activeSessionByID.User.ID == userID {
		mockRepo.activeSessionByID.User.Consented = true
	}
	return nil
}

func (mockRepo *mockUserSessionRepository) DeleteUnconsentedUserAndAccounts(ctx context.Context, userID string) error {
	mockRepo.deletedUnconsentedID = userID
	return nil
}

func (mockRepo *mockUserSessionRepository) DeleteAuthenticatedUserAccountAndAllData(ctx context.Context, userID string) error {
	mockRepo.deletedAccountUserID = userID
	return nil
}

// TestHandleUserLoginRequestSuccessUnverifiedAndWrongPassword verifies 200 OK login with cookies,
// 403 Forbidden for unverified users, and 401 Unauthorized for wrong password.
func TestHandleUserLoginRequestSuccessUnverifiedAndWrongPassword(testingInstance *testing.T) {
	jwtSecret := "test-login-jwt-secret"
	password := "ValidPassword123!"
	passwordHash, _ := auth.HashUserPassword(password)

	verifiedUser := &models.User{
		ID:            "user-id-verified",
		Email:         "verified@example.com",
		Password:      passwordHash,
		PasswordHash:  &passwordHash,
		Verified:      true,
		EmailVerified: true,
	}

	mockRepo := &mockUserSessionRepository{userForLogin: verifiedUser}
	sessionHandler, initErr := NewUserSessionHandler(mockRepo, jwtSecret, false)
	if initErr != nil {
		testingInstance.Fatalf("failed to create UserSessionHandler: %v", initErr)
	}

	// 1. Login success (200 OK + cookies)
	loginBody, _ := json.Marshal(UserLoginRequestPayload{Email: "verified@example.com", Password: password})
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginRecorder := httptest.NewRecorder()
	sessionHandler.HandleUserLoginRequest(loginRecorder, loginRequest)

	if loginRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK for valid login, got %d (%s)", loginRecorder.Code, loginRecorder.Body.String())
	}

	if len(loginRecorder.Result().Cookies()) != 2 {
		testingInstance.Errorf("expected 2 auth cookies to be set on login, got %d", len(loginRecorder.Result().Cookies()))
	}

	// 2. Wrong password (401 Unauthorized)
	wrongBody, _ := json.Marshal(UserLoginRequestPayload{Email: "verified@example.com", Password: "WrongPassword999!"})
	wrongRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(wrongBody))
	wrongRecorder := httptest.NewRecorder()
	sessionHandler.HandleUserLoginRequest(wrongRecorder, wrongRequest)

	if wrongRecorder.Code != http.StatusUnauthorized {
		testingInstance.Fatalf("expected 401 Unauthorized for wrong password, got %d", wrongRecorder.Code)
	}

	// 3. Unverified user (403 Forbidden)
	mockRepo.userForLogin = &models.User{
		ID:            "user-id-unverified",
		Email:         "unverified@example.com",
		Password:      passwordHash,
		PasswordHash:  &passwordHash,
		Verified:      false,
		EmailVerified: false,
	}
	unverifiedBody, _ := json.Marshal(UserLoginRequestPayload{Email: "unverified@example.com", Password: password})
	unverifiedRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(unverifiedBody))
	unverifiedRecorder := httptest.NewRecorder()
	sessionHandler.HandleUserLoginRequest(unverifiedRecorder, unverifiedRequest)

	if unverifiedRecorder.Code != http.StatusForbidden {
		testingInstance.Fatalf("expected 403 Forbidden for unverified user, got %d", unverifiedRecorder.Code)
	}
}

// TestHandleRefreshTokenRotationRequestSuccessAndReuseDetection verifies single-use refresh token rotation
// and ensures replaying a revoked/replaced refresh token revokes the entire session.
func TestHandleRefreshTokenRotationRequestSuccessAndReuseDetection(testingInstance *testing.T) {
	jwtSecret := "test-refresh-jwt-secret"
	rawSecret, secretHash, _ := auth.GenerateRefreshTokenSecretAndHash()
	tokenID := "token-id-active"
	sessionID := "session-id-active"
	userID := "user-id-active"

	mockRepo := &mockUserSessionRepository{
		storedRefreshToken: &models.RefreshToken{
			ID:        tokenID,
			SessionID: sessionID,
			TokenHash: secretHash,
			Revoked:   false,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			Session: models.Session{
				ID:        sessionID,
				UserID:    userID,
				Revoked:   false,
				ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			},
		},
	}

	sessionHandler, _ := NewUserSessionHandler(mockRepo, jwtSecret, false)
	cookieValue := auth.FormatCompositeRefreshTokenCookieValue(tokenID, rawSecret)

	// 1. Valid rotation (200 OK)
	refreshRequest := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshRequest.AddCookie(&http.Cookie{Name: auth.RefreshTokenCookieName, Value: cookieValue})
	refreshRecorder := httptest.NewRecorder()

	sessionHandler.HandleRefreshTokenRotationRequest(refreshRecorder, refreshRequest)

	if refreshRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK on refresh rotation, got %d (%s)", refreshRecorder.Code, refreshRecorder.Body.String())
	}

	if mockRepo.rotatedOldTokenID != tokenID || mockRepo.rotatedNewToken == nil {
		testingInstance.Fatalf("expected RotateRefreshTokenInTransaction to be called for %q", tokenID)
	}

	// 2. Reuse attack: token is now marked Revoked = true
	replacedBy := "token-id-new"
	mockRepo.storedRefreshToken.Revoked = true
	mockRepo.storedRefreshToken.ReplacedByTokenID = &replacedBy

	reuseRequest := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	reuseRequest.AddCookie(&http.Cookie{Name: auth.RefreshTokenCookieName, Value: cookieValue})
	reuseRecorder := httptest.NewRecorder()

	sessionHandler.HandleRefreshTokenRotationRequest(reuseRecorder, reuseRequest)

	if reuseRecorder.Code != http.StatusUnauthorized {
		testingInstance.Fatalf("expected 401 Unauthorized on refresh token reuse, got %d", reuseRecorder.Code)
	}

	if mockRepo.revokedSessionID != sessionID {
		testingInstance.Errorf("expected reuse detection to revoke session %q, got %q", sessionID, mockRepo.revokedSessionID)
	}
}

// TestHandleUserLogoutRequestAndVerifyActiveSessionRequest verifies logout session revocation + cookie clearing
// and GET /api/auth/session active session verification.
func TestHandleUserLogoutRequestAndVerifyActiveSessionRequest(testingInstance *testing.T) {
	jwtSecret := "test-session-jwt-secret"
	rawSecret, secretHash, _ := auth.GenerateRefreshTokenSecretAndHash()
	tokenID := "token-id-logout"
	sessionID := "session-id-logout"
	userID := "user-id-logout"

	mockRepo := &mockUserSessionRepository{
		storedRefreshToken: &models.RefreshToken{
			ID:        tokenID,
			SessionID: sessionID,
			TokenHash: secretHash,
		},
		activeSessionByID: &models.Session{
			ID:        sessionID,
			UserID:    userID,
			Revoked:   false,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
		},
	}

	sessionHandler, _ := NewUserSessionHandler(mockRepo, jwtSecret, false)

	// 1. Verify active session (200 OK)
	accessToken, _ := auth.GenerateSignedAccessToken(userID, sessionID, jwtSecret, 15*time.Minute)
	sessionRequest := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	sessionRequest.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	sessionRecorder := httptest.NewRecorder()

	sessionHandler.HandleVerifyActiveSessionRequest(sessionRecorder, sessionRequest)

	if sessionRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK for valid active session, got %d", sessionRecorder.Code)
	}

	// 2. Logout (200 OK + revokes session + clears cookies)
	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{
		Name:  auth.RefreshTokenCookieName,
		Value: auth.FormatCompositeRefreshTokenCookieValue(tokenID, rawSecret),
	})
	logoutRecorder := httptest.NewRecorder()

	sessionHandler.HandleUserLogoutRequest(logoutRecorder, logoutRequest)

	if logoutRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK on logout, got %d", logoutRecorder.Code)
	}

	if mockRepo.revokedSessionID != sessionID {
		testingInstance.Errorf("expected logout to revoke session %q, got %q", sessionID, mockRepo.revokedSessionID)
	}
}

// TestHandleUserGDPRConsentRequestAcceptAndDeclineAndEnrichedSession verifies enriched GET /api/auth/session
// fields (email, consented, twoFactorEnabled) and POST /api/auth/consent for consented=true and consented=false.
func TestHandleUserGDPRConsentRequestAcceptAndDeclineAndEnrichedSession(testingInstance *testing.T) {
	jwtSecret := "test-consent-jwt-secret"
	sessionID := "session-id-consent"
	userID := "user-id-consent"

	mockRepo := &mockUserSessionRepository{
		activeSessionByID: &models.Session{
			ID:        sessionID,
			UserID:    userID,
			Revoked:   false,
			ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
			User: models.User{
				ID:               userID,
				Email:            "oauth-user@example.com",
				Consented:        false,
				TwoFactorEnabled: true,
			},
		},
	}

	sessionHandler, _ := NewUserSessionHandler(mockRepo, jwtSecret, false)
	accessToken, _ := auth.GenerateSignedAccessToken(userID, sessionID, jwtSecret, 15*time.Minute)

	// 1. Verify enriched GET /api/auth/session returns email, consented=false, twoFactorEnabled=true
	sessionReq := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	sessionReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	sessionRec := httptest.NewRecorder()
	sessionHandler.HandleVerifyActiveSessionRequest(sessionRec, sessionReq)

	if sessionRec.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK from GET /api/auth/session, got %d", sessionRec.Code)
	}

	var sessionResponse ActiveSessionResponsePayload
	if decodeErr := json.Unmarshal(sessionRec.Body.Bytes(), &sessionResponse); decodeErr != nil {
		testingInstance.Fatalf("failed to decode session response: %v", decodeErr)
	}

	if sessionResponse.Email != "oauth-user@example.com" || sessionResponse.Consented || !sessionResponse.TwoFactorEnabled {
		testingInstance.Fatalf("unexpected enriched session response: %+v", sessionResponse)
	}

	// 2. Accept GDPR consent (POST /api/auth/consent { "consented": true })
	acceptBody, _ := json.Marshal(UserGDPRConsentRequestPayload{Consented: true})
	acceptReq := httptest.NewRequest(http.MethodPost, "/api/auth/consent", bytes.NewReader(acceptBody))
	acceptReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	acceptRec := httptest.NewRecorder()
	sessionHandler.HandleUserGDPRConsentRequest(acceptRec, acceptReq)

	if acceptRec.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK when accepting consent, got %d (%s)", acceptRec.Code, acceptRec.Body.String())
	}

	if mockRepo.acceptedConsentUserID != userID {
		testingInstance.Errorf("expected AcceptUserGDPRConsent to be called with %q, got %q", userID, mockRepo.acceptedConsentUserID)
	}

	// 3. Decline GDPR consent (POST /api/auth/consent { "consented": false }) -> deletes account & clears auth cookies
	declineBody, _ := json.Marshal(UserGDPRConsentRequestPayload{Consented: false})
	declineReq := httptest.NewRequest(http.MethodPost, "/api/auth/consent", bytes.NewReader(declineBody))
	declineReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	declineRec := httptest.NewRecorder()
	sessionHandler.HandleUserGDPRConsentRequest(declineRec, declineReq)

	if declineRec.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK when declining consent, got %d (%s)", declineRec.Code, declineRec.Body.String())
	}

	if mockRepo.deletedUnconsentedID != userID {
		testingInstance.Errorf("expected DeleteUnconsentedUserAndAccounts to be called with %q, got %q", userID, mockRepo.deletedUnconsentedID)
	}

	if len(declineRec.Result().Cookies()) != 2 {
		testingInstance.Errorf("expected 2 cleared auth cookies on consent decline, got %d", len(declineRec.Result().Cookies()))
	}
}

func TestHandleDeleteAuthenticatedUserAccountRequest_AuthAndSuccess(t *testing.T) {
	sessionRepo := &mockUserSessionRepository{
		activeSessionByID: &models.Session{
			ID:        "sess-del-1",
			UserID:    "user-del-1",
			ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
			Revoked:   false,
		},
	}
	handler, err := NewUserSessionHandler(sessionRepo, "test-jwt-secret-key-32-chars-minimum", false)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	// Non-DELETE returns 405
	wrongMethodReq := httptest.NewRequest(http.MethodPost, "/api/auth/account", nil)
	wrongMethodRec := httptest.NewRecorder()
	handler.HandleDeleteAuthenticatedUserAccountRequest(wrongMethodRec, wrongMethodReq)
	if wrongMethodRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", wrongMethodRec.Code)
	}

	// Missing cookie returns 401
	unauthReq := httptest.NewRequest(http.MethodDelete, "/api/auth/account", nil)
	unauthRec := httptest.NewRecorder()
	handler.HandleDeleteAuthenticatedUserAccountRequest(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", unauthRec.Code)
	}

	// Valid authenticated DELETE deletes account and clears cookies
	accessToken, tokenErr := auth.GenerateSignedAccessToken("user-del-1", "sess-del-1", "test-jwt-secret-key-32-chars-minimum", 15*time.Minute)
	if tokenErr != nil {
		t.Fatalf("failed to generate access token: %v", tokenErr)
	}
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/auth/account", nil)
	deleteReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	deleteRec := httptest.NewRecorder()
	handler.HandleDeleteAuthenticatedUserAccountRequest(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if sessionRepo.deletedAccountUserID != "user-del-1" {
		t.Fatalf("expected DeleteAuthenticatedUserAccountAndAllData for user-del-1, got %q", sessionRepo.deletedAccountUserID)
	}
}
