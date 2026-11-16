package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
)

type stubPasswordResetRepository struct {
	userByEmail             *models.User
	findUserErr             error
	replacedUserID          string
	replacedTokenHash       string
	replaceTokenErr         error
	resetRecordByHash       *models.PasswordReset
	findResetErr            error
	completedResetID        string
	completedUserID         string
	completedPasswordHash   string
	completeResetErr        error
	replaceTokenCalledCount int
	completeCalledCount     int
}

func (stub *stubPasswordResetRepository) FindUserByEmailForPasswordReset(ctx context.Context, emailAddress string) (*models.User, error) {
	return stub.userByEmail, stub.findUserErr
}

func (stub *stubPasswordResetRepository) ReplaceActivePasswordResetToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	stub.replaceTokenCalledCount++
	stub.replacedUserID = userID
	stub.replacedTokenHash = tokenHash
	return stub.replaceTokenErr
}

func (stub *stubPasswordResetRepository) FindPasswordResetByTokenHash(ctx context.Context, tokenHash string) (*models.PasswordReset, error) {
	return stub.resetRecordByHash, stub.findResetErr
}

func (stub *stubPasswordResetRepository) CompletePasswordResetAndRevokeSessions(ctx context.Context, resetID string, userID string, newPasswordHash string, completedAt time.Time) error {
	stub.completeCalledCount++
	stub.completedResetID = resetID
	stub.completedUserID = userID
	stub.completedPasswordHash = newPasswordHash
	return stub.completeResetErr
}

type stubPasswordResetEmailSender struct {
	sentToEmail  string
	sentRawToken string
	sendCount    int
	sendErr      error
}

func (stub *stubPasswordResetEmailSender) SendPasswordResetEmail(recipientEmailAddress string, rawResetToken string) error {
	stub.sendCount++
	stub.sentToEmail = recipientEmailAddress
	stub.sentRawToken = rawResetToken
	return stub.sendErr
}

func TestHandleRequestPasswordReset_SuccessStoresHashAndSendsRawToken(t *testing.T) {
	repo := &stubPasswordResetRepository{
		userByEmail: &models.User{
			ID:       "user-123",
			Email:    "member@example.com",
			Password: "hashed-existing-password",
			Verified: true,
		},
	}
	emailSender := &stubPasswordResetEmailSender{}

	handler, err := NewPasswordResetHandler(repo, emailSender)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/passwordreset", bytes.NewBufferString(`{"emailReset":"Member@Example.com"}`))
	rec := httptest.NewRecorder()
	handler.HandleRequestPasswordReset(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	if repo.replaceTokenCalledCount != 1 {
		t.Fatalf("expected 1 token persisted, got %d", repo.replaceTokenCalledCount)
	}
	if emailSender.sendCount != 1 || emailSender.sentToEmail != "member@example.com" {
		t.Fatalf("expected 1 email sent to member@example.com, got count=%d to=%s", emailSender.sendCount, emailSender.sentToEmail)
	}
	if auth.HashPasswordResetToken(emailSender.sentRawToken) != repo.replacedTokenHash {
		t.Fatalf("expected persisted tokenHash to match SHA-256 of emailed rawToken")
	}
}

func TestHandleRequestPasswordReset_AntiEnumerationForUnknownAndOAuthUsers(t *testing.T) {
	testCases := []struct {
		name string
		user *models.User
	}{
		{name: "unknown user", user: nil},
		{name: "oauth-only user", user: &models.User{ID: "oauth-1", Email: "oauth@example.com", Password: ""}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubPasswordResetRepository{userByEmail: tc.user}
			emailSender := &stubPasswordResetEmailSender{}
			handler, _ := NewPasswordResetHandler(repo, emailSender)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/passwordreset", bytes.NewBufferString(`{"emailReset":"anyone@example.com"}`))
			rec := httptest.NewRecorder()
			handler.HandleRequestPasswordReset(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d", rec.Code)
			}
			if repo.replaceTokenCalledCount != 0 || emailSender.sendCount != 0 {
				t.Fatalf("expected no token or email for %s", tc.name)
			}

			var payload StandardAPIResponsePayload
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("failed to decode JSON response: %v", err)
			}
			if payload.Type != "success" {
				t.Fatalf("expected generic success type, got %q", payload.Type)
			}
		})
	}
}

func TestHandleRequestPasswordReset_InvalidEmailReturns400(t *testing.T) {
	handler, _ := NewPasswordResetHandler(&stubPasswordResetRepository{}, &stubPasswordResetEmailSender{})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/passwordreset", bytes.NewBufferString(`{"emailReset":"not-an-email"}`))
	rec := httptest.NewRecorder()
	handler.HandleRequestPasswordReset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestHandleRequestPasswordReset_EmailFailureReturns500(t *testing.T) {
	repo := &stubPasswordResetRepository{
		userByEmail: &models.User{ID: "user-1", Email: "member@example.com", Password: "hash", Verified: true},
	}
	emailSender := &stubPasswordResetEmailSender{sendErr: errors.New("smtp error")}
	handler, _ := NewPasswordResetHandler(repo, emailSender)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/passwordreset", bytes.NewBufferString(`{"emailReset":"member@example.com"}`))
	rec := httptest.NewRecorder()
	handler.HandleRequestPasswordReset(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d", rec.Code)
	}
}

func TestHandleConfirmPasswordReset_SuccessUpdatesPasswordAndRevokesSessions(t *testing.T) {
	rawToken := "valid-raw-token-123"
	repo := &stubPasswordResetRepository{
		resetRecordByHash: &models.PasswordReset{
			ID:                      "reset-1",
			UserID:                  "user-123",
			VerificationToken:       auth.HashPasswordResetToken(rawToken),
			VerificationTokenExpiry: time.Now().UTC().Add(30 * time.Minute),
			Verified:                false,
		},
	}
	handler, _ := NewPasswordResetHandler(repo, &stubPasswordResetEmailSender{})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/passwordreset/confirm",
		bytes.NewBufferString(`{"token":"valid-raw-token-123","newPassword":"StrongPassword123"}`),
	)
	rec := httptest.NewRecorder()
	handler.HandleConfirmPasswordReset(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.completeCalledCount != 1 || repo.completedUserID != "user-123" || repo.completedResetID != "reset-1" {
		t.Fatalf("expected CompletePasswordResetAndRevokeSessions to be called for reset-1 / user-123")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.completedPasswordHash), []byte("StrongPassword123")); err != nil {
		t.Fatalf("expected stored password hash to match newPassword: %v", err)
	}
}

func TestHandleConfirmPasswordReset_RejectsInvalidExpiredOrShortPassword(t *testing.T) {
	now := time.Now().UTC()
	revokedTime := now.Add(-1 * time.Minute)

	testCases := []struct {
		name   string
		body   string
		record *models.PasswordReset
	}{
		{
			name:   "missing token",
			body:   `{"token":"","newPassword":"StrongPassword123"}`,
			record: nil,
		},
		{
			name:   "short password",
			body:   `{"token":"abc","newPassword":"short"}`,
			record: nil,
		},
		{
			name:   "unknown token",
			body:   `{"token":"unknown","newPassword":"StrongPassword123"}`,
			record: nil,
		},
		{
			name: "expired token",
			body: `{"token":"expired","newPassword":"StrongPassword123"}`,
			record: &models.PasswordReset{
				ID:                      "r1",
				UserID:                  "u1",
				VerificationTokenExpiry: now.Add(-5 * time.Minute),
			},
		},
		{
			name: "already verified token",
			body: `{"token":"used","newPassword":"StrongPassword123"}`,
			record: &models.PasswordReset{
				ID:                      "r2",
				UserID:                  "u1",
				VerificationTokenExpiry: now.Add(15 * time.Minute),
				Verified:                true,
			},
		},
		{
			name: "revoked token",
			body: `{"token":"revoked","newPassword":"StrongPassword123"}`,
			record: &models.PasswordReset{
				ID:                      "r3",
				UserID:                  "u1",
				VerificationTokenExpiry: now.Add(15 * time.Minute),
				RevokedAt:               &revokedTime,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubPasswordResetRepository{resetRecordByHash: tc.record}
			handler, _ := NewPasswordResetHandler(repo, &stubPasswordResetEmailSender{})

			req := httptest.NewRequest(http.MethodPost, "/api/auth/passwordreset/confirm", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()
			handler.HandleConfirmPasswordReset(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request for %s, got %d", tc.name, rec.Code)
			}
			if repo.completeCalledCount != 0 {
				t.Fatalf("expected no password update for %s", tc.name)
			}
		})
	}
}

func TestHandleRequestPasswordReset_UnverifiedUserReturnsGeneric200WithoutSendingReset(t *testing.T) {
	repo := &stubPasswordResetRepository{
		userByEmail: &models.User{
			ID:       "unverified-user-1",
			Email:    "pending@example.com",
			Password: "hashed-password",
			Verified: false,
		},
	}
	emailSender := &stubPasswordResetEmailSender{}
	handler, _ := NewPasswordResetHandler(repo, emailSender)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/passwordreset", bytes.NewBufferString(`{"emailReset":"pending@example.com"}`))
	rec := httptest.NewRecorder()
	handler.HandleRequestPasswordReset(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.replaceTokenCalledCount != 0 || emailSender.sendCount != 0 {
		t.Fatalf("expected no reset token or email for unverified user")
	}

	var payload StandardAPIResponsePayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Type != "success" {
		t.Fatalf("expected generic success type, got %q", payload.Type)
	}
}
