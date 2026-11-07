// This file contains unit tests for UserRegistrationHandler, ValidateUserRegistrationRequestPayload,
// HandleUserRegistrationRequest, and HandleUserEmailVerificationRequest using mock repository and email sender implementations.
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/gorm"
	"nutrysync/go-api/models"
)

// mockUserRegistrationRepository is an in-memory test double implementing repositories.UserRegistrationRepository.
type mockUserRegistrationRepository struct {
	userByEmail             *models.User
	findByEmailError        error
	createdUser             *models.User
	createUserError         error
	userByVerificationToken *models.User
	findByTokenError        error
	verifiedUserID          string
	markVerifiedError       error
}

func (mockRepo *mockUserRegistrationRepository) FindUserByEmailAddress(emailAddress string) (*models.User, error) {
	if mockRepo.findByEmailError != nil {
		return nil, mockRepo.findByEmailError
	}
	return mockRepo.userByEmail, nil
}

func (mockRepo *mockUserRegistrationRepository) CreateUnverifiedUserRecord(user *models.User) error {
	if mockRepo.createUserError != nil {
		return mockRepo.createUserError
	}
	mockRepo.createdUser = user
	return nil
}

func (mockRepo *mockUserRegistrationRepository) FindUserByVerificationToken(verificationToken string) (*models.User, error) {
	if mockRepo.findByTokenError != nil {
		return nil, mockRepo.findByTokenError
	}
	return mockRepo.userByVerificationToken, nil
}

func (mockRepo *mockUserRegistrationRepository) MarkUserEmailAsVerified(userID string, verifiedAt time.Time) error {
	if mockRepo.markVerifiedError != nil {
		return mockRepo.markVerifiedError
	}
	mockRepo.verifiedUserID = userID
	return nil
}

// mockVerificationEmailSender is a test double implementing auth.VerificationEmailSender.
type mockVerificationEmailSender struct {
	sentRecipientEmail string
	sentToken          string
	sendError          error
}

func (mockSender *mockVerificationEmailSender) SendRegistrationVerificationEmail(recipientEmailAddress string, verificationToken string) error {
	if mockSender.sendError != nil {
		return mockSender.sendError
	}
	mockSender.sentRecipientEmail = recipientEmailAddress
	mockSender.sentToken = verificationToken
	return nil
}

// TestValidateUserRegistrationRequestPayloadRejectsInvalidInputs verifies all validation rules on UserRegistrationRequestPayload.
func TestValidateUserRegistrationRequestPayloadRejectsInvalidInputs(testingInstance *testing.T) {
	invalidPayloads := []UserRegistrationRequestPayload{
		{Email: "not-an-email", ConfirmEmail: "not-an-email", Password: "password123", ConfirmPassword: "password123"},
		{Email: "user@example.com", ConfirmEmail: "other@example.com", Password: "password123", ConfirmPassword: "password123"},
		{Email: "user@example.com", ConfirmEmail: "user@example.com", Password: "short", ConfirmPassword: "short"},
		{Email: "user@example.com", ConfirmEmail: "user@example.com", Password: "password123", ConfirmPassword: "different123"},
	}

	for index, payload := range invalidPayloads {
		if validationError := ValidateUserRegistrationRequestPayload(payload); validationError == nil {
			testingInstance.Errorf("expected validation error for case %d, got nil", index)
		}
	}
}

// TestHandleUserRegistrationRequestCreatesUnverifiedUserAndSendsVerificationEmail verifies the 201 Created happy path.
func TestHandleUserRegistrationRequestCreatesUnverifiedUserAndSendsVerificationEmail(testingInstance *testing.T) {
	mockRepository := &mockUserRegistrationRepository{
		findByEmailError: gorm.ErrRecordNotFound,
	}
	mockEmailSender := &mockVerificationEmailSender{}
	registrationHandler := NewUserRegistrationHandler(mockRepository, mockEmailSender)

	requestBody, _ := json.Marshal(UserRegistrationRequestPayload{
		Email:           "newuser@example.com",
		ConfirmEmail:    "newuser@example.com",
		Password:        "StrongPass123!",
		ConfirmPassword: "StrongPass123!",
	})

	request := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()

	registrationHandler.HandleUserRegistrationRequest(responseRecorder, request)

	if responseRecorder.Code != http.StatusCreated {
		testingInstance.Fatalf("expected status %d, got %d (body: %s)", http.StatusCreated, responseRecorder.Code, responseRecorder.Body.String())
	}

	var responsePayload StandardAPIResponsePayload
	if decodeError := json.Unmarshal(responseRecorder.Body.Bytes(), &responsePayload); decodeError != nil {
		testingInstance.Fatalf("failed to decode response JSON: %v", decodeError)
	}

	if responsePayload.Type != "success" || responsePayload.Title != "Registration Successful" {
		testingInstance.Errorf("unexpected response payload: %+v", responsePayload)
	}

	if mockRepository.createdUser == nil || mockRepository.createdUser.Email != "newuser@example.com" {
		testingInstance.Fatalf("expected unverified user to be created in repository, got %+v", mockRepository.createdUser)
	}

	if mockRepository.createdUser.Consented {
		testingInstance.Error("expected newly registered user to start with Consented == false")
	}

	if mockEmailSender.sentRecipientEmail != "newuser@example.com" || mockEmailSender.sentToken == "" {
		testingInstance.Errorf("expected verification email to be sent to newuser@example.com, got recipient=%q token=%q", mockEmailSender.sentRecipientEmail, mockEmailSender.sentToken)
	}
}

// TestHandleUserRegistrationRequestReturnsBadRequestOnValidationFailure verifies 400 Bad Request on invalid registration data.
func TestHandleUserRegistrationRequestReturnsBadRequestOnValidationFailure(testingInstance *testing.T) {
	registrationHandler := NewUserRegistrationHandler(&mockUserRegistrationRepository{}, &mockVerificationEmailSender{})

	requestBody, _ := json.Marshal(UserRegistrationRequestPayload{
		Email:           "invalid-email",
		ConfirmEmail:    "invalid-email",
		Password:        "123",
		ConfirmPassword: "123",
		Consented:       false,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()

	registrationHandler.HandleUserRegistrationRequest(responseRecorder, request)

	if responseRecorder.Code != http.StatusBadRequest {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusBadRequest, responseRecorder.Code)
	}
}

// TestHandleUserRegistrationRequestReturnsConflictWhenEmailAlreadyExists verifies 409 Conflict when email is already registered.
func TestHandleUserRegistrationRequestReturnsConflictWhenEmailAlreadyExists(testingInstance *testing.T) {
	mockRepository := &mockUserRegistrationRepository{
		userByEmail: &models.User{
			ID:    "existing-id",
			Email: "existing@example.com",
		},
	}
	registrationHandler := NewUserRegistrationHandler(mockRepository, &mockVerificationEmailSender{})

	requestBody, _ := json.Marshal(UserRegistrationRequestPayload{
		Email:           "existing@example.com",
		ConfirmEmail:    "existing@example.com",
		Password:        "StrongPass123!",
		ConfirmPassword: "StrongPass123!",
		Concented:       true,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(requestBody))
	responseRecorder := httptest.NewRecorder()

	registrationHandler.HandleUserRegistrationRequest(responseRecorder, request)

	if responseRecorder.Code != http.StatusConflict {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusConflict, responseRecorder.Code)
	}

	var responsePayload StandardAPIResponsePayload
	_ = json.Unmarshal(responseRecorder.Body.Bytes(), &responsePayload)
	if responsePayload.Message != "email already registered" {
		testingInstance.Errorf("expected message %q, got %q", "email already registered", responsePayload.Message)
	}
}

// TestHandleUserEmailVerificationRequestMarksUserVerifiedForValidToken verifies 200 OK when a valid, non-expired token is supplied.
func TestHandleUserEmailVerificationRequestMarksUserVerifiedForValidToken(testingInstance *testing.T) {
	validToken := "valid-verification-token"
	futureExpiration := time.Now().UTC().Add(12 * time.Hour)

	mockRepository := &mockUserRegistrationRepository{
		userByVerificationToken: &models.User{
			ID:                         "user-uuid-123",
			Email:                      "verify@example.com",
			VerificationToken:          &validToken,
			VerificationTokenExpiredAt: &futureExpiration,
		},
	}
	registrationHandler := NewUserRegistrationHandler(mockRepository, &mockVerificationEmailSender{})

	request := httptest.NewRequest(http.MethodGet, "/api/auth/verify?token="+validToken, nil)
	responseRecorder := httptest.NewRecorder()

	registrationHandler.HandleUserEmailVerificationRequest(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		testingInstance.Fatalf("expected status %d, got %d", http.StatusOK, responseRecorder.Code)
	}

	if mockRepository.verifiedUserID != "user-uuid-123" {
		testingInstance.Errorf("expected MarkUserEmailAsVerified to be called for user-uuid-123, got %q", mockRepository.verifiedUserID)
	}

	var responsePayload StandardAPIResponsePayload
	_ = json.Unmarshal(responseRecorder.Body.Bytes(), &responsePayload)
	if responsePayload.Type != "success" || responsePayload.Title != "Email Verified" || responsePayload.Message != "user verified" {
		testingInstance.Errorf("unexpected verification response payload: %+v", responsePayload)
	}
}

// TestHandleUserEmailVerificationRequestReturnsBadRequestForExpiredOrInvalidToken verifies 400 Bad Request for expired or missing tokens.
func TestHandleUserEmailVerificationRequestReturnsBadRequestForExpiredOrInvalidToken(testingInstance *testing.T) {
	expiredToken := "expired-token"
	pastExpiration := time.Now().UTC().Add(-1 * time.Hour)

	mockRepository := &mockUserRegistrationRepository{
		userByVerificationToken: &models.User{
			ID:                         "user-uuid-expired",
			Email:                      "expired@example.com",
			VerificationToken:          &expiredToken,
			VerificationTokenExpiredAt: &pastExpiration,
		},
	}
	registrationHandler := NewUserRegistrationHandler(mockRepository, &mockVerificationEmailSender{})

	expiredRequest := httptest.NewRequest(http.MethodGet, "/api/auth/verify?token="+expiredToken, nil)
	expiredRecorder := httptest.NewRecorder()
	registrationHandler.HandleUserEmailVerificationRequest(expiredRecorder, expiredRequest)

	if expiredRecorder.Code != http.StatusBadRequest {
		testingInstance.Fatalf("expected status %d for expired token, got %d", http.StatusBadRequest, expiredRecorder.Code)
	}

	missingTokenRequest := httptest.NewRequest(http.MethodGet, "/api/auth/verify", nil)
	missingTokenRecorder := httptest.NewRecorder()
	registrationHandler.HandleUserEmailVerificationRequest(missingTokenRecorder, missingTokenRequest)

	if missingTokenRecorder.Code != http.StatusBadRequest {
		testingInstance.Fatalf("expected status %d for missing token, got %d", http.StatusBadRequest, missingTokenRecorder.Code)
	}
}
