// This file implements HTTP handlers for user registration (POST /api/auth/register)
// and email verification (GET /api/auth/verify), reusing WriteJSONResponse from handlers/health.go.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"gorm.io/gorm"
	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
	"nutrysync/go-api/repositories"
)

const emailVerificationTokenValidityDuration = 24 * time.Hour

// UserRegistrationRequestPayload represents the JSON body accepted by POST /api/auth/register.
// Supports both "consented" and the legacy "concented" spelling from client forms.
type UserRegistrationRequestPayload struct {
	Email           string `json:"email"`
	ConfirmEmail    string `json:"confirmEmail"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
	Concented       bool   `json:"concented"`
	Consented       bool   `json:"consented"`
}

// StandardAPIResponsePayload represents the structured JSON response returned by registration and verification endpoints.
type StandardAPIResponsePayload struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

// UserRegistrationHandler coordinates user registration and email verification HTTP requests
// using a UserRegistrationRepository and a VerificationEmailSender.
type UserRegistrationHandler struct {
	userRegistrationRepository repositories.UserRegistrationRepository
	verificationEmailSender    auth.VerificationEmailSender
}

// NewUserRegistrationHandler constructs a UserRegistrationHandler with the provided repository and email sender dependencies.
func NewUserRegistrationHandler(
	userRegistrationRepository repositories.UserRegistrationRepository,
	verificationEmailSender auth.VerificationEmailSender,
) *UserRegistrationHandler {
	return &UserRegistrationHandler{
		userRegistrationRepository: userRegistrationRepository,
		verificationEmailSender:    verificationEmailSender,
	}
}

// ValidateUserRegistrationRequestPayload checks that email is well-formed, confirmEmail matches email,
// password is at least 8 characters, confirmPassword matches password, and consent is true.
func ValidateUserRegistrationRequestPayload(payload UserRegistrationRequestPayload) error {
	trimmedEmail := strings.TrimSpace(payload.Email)
	if trimmedEmail == "" {
		return errors.New("email is required")
	}

	parsedAddress, parseError := mail.ParseAddress(trimmedEmail)
	if parseError != nil || parsedAddress.Address != trimmedEmail || !strings.Contains(trimmedEmail, "@") {
		return errors.New("invalid email format")
	}

	if trimmedEmail != strings.TrimSpace(payload.ConfirmEmail) {
		return errors.New("email and confirmEmail do not match")
	}

	if len(payload.Password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	if payload.Password != payload.ConfirmPassword {
		return errors.New("password and confirmPassword do not match")
	}

	if !payload.Consented && !payload.Concented {
		return errors.New("user consent is required")
	}

	return nil
}

// HandleUserRegistrationRequest handles POST /api/auth/register requests:
// validates the registration payload, checks for duplicate emails, hashes the password,
// persists the unverified user with a 24-hour verification token, and sends the verification email.
func (handler *UserRegistrationHandler) HandleUserRegistrationRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Method Not Allowed",
			Message: "only POST method is supported",
		})
		return
	}

	var registrationPayload UserRegistrationRequestPayload
	decodeError := json.NewDecoder(request.Body).Decode(&registrationPayload)
	if decodeError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Validation Error",
			Message: "invalid JSON request body",
		})
		return
	}

	if validationError := ValidateUserRegistrationRequestPayload(registrationPayload); validationError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Validation Error",
			Message: validationError.Error(),
		})
		return
	}

	normalizedEmail := strings.TrimSpace(registrationPayload.Email)
	existingUser, lookupError := handler.userRegistrationRepository.FindUserByEmailAddress(normalizedEmail)
	if lookupError != nil && !errors.Is(lookupError, gorm.ErrRecordNotFound) {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Registration Failed",
			Message: "failed to check existing user account",
		})
		return
	}

	if existingUser != nil {
		WriteJSONResponse(responseWriter, http.StatusConflict, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Registration Failed",
			Message: "email already registered",
		})
		return
	}

	hashedPassword, hashError := auth.HashUserPassword(registrationPayload.Password)
	if hashError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Registration Failed",
			Message: "failed to secure user password",
		})
		return
	}

	verificationToken, tokenExpirationTime, tokenError := auth.GenerateEmailVerificationTokenAndExpiryDate(emailVerificationTokenValidityDuration)
	if tokenError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Registration Failed",
			Message: "failed to generate email verification token",
		})
		return
	}

	newUserRecord := &models.User{
		Email:                      normalizedEmail,
		Password:                   hashedPassword,
		Consented:                  registrationPayload.Consented || registrationPayload.Concented,
		Verified:                   false,
		VerificationToken:          &verificationToken,
		VerificationTokenExpiredAt: &tokenExpirationTime,
	}

	if createError := handler.userRegistrationRepository.CreateUnverifiedUserRecord(newUserRecord); createError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Registration Failed",
			Message: "failed to create user record",
		})
		return
	}

	if emailError := handler.verificationEmailSender.SendRegistrationVerificationEmail(normalizedEmail, verificationToken); emailError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Registration Failed",
			Message: "failed to send verification email",
		})
		return
	}

	WriteJSONResponse(responseWriter, http.StatusCreated, StandardAPIResponsePayload{
		Type:    "success",
		Title:   "Registration Successful",
		Message: "User created, please check your email to verify",
	})
}

// HandleUserEmailVerificationRequest handles GET /api/auth/verify?token=... requests:
// validates the token and its expiration timestamp, marks the user email as verified, and returns a JSON status response.
func (handler *UserRegistrationHandler) HandleUserEmailVerificationRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Method Not Allowed",
			Message: "only GET method is supported",
		})
		return
	}

	verificationToken := strings.TrimSpace(request.URL.Query().Get("token"))
	if verificationToken == "" {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Verification Failed",
			Message: "missing token",
		})
		return
	}

	userRecord, lookupError := handler.userRegistrationRepository.FindUserByVerificationToken(verificationToken)
	if lookupError != nil || userRecord == nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Verification Failed",
			Message: "invalid or expired verification token",
		})
		return
	}

	currentTimeUTC := time.Now().UTC()
	if userRecord.VerificationTokenExpiredAt == nil || !userRecord.VerificationTokenExpiredAt.After(currentTimeUTC) {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Verification Failed",
			Message: "verification token has expired",
		})
		return
	}

	if updateError := handler.userRegistrationRepository.MarkUserEmailAsVerified(userRecord.ID, currentTimeUTC); updateError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Verification Failed",
			Message: "failed to verify user email",
		})
		return
	}

	WriteJSONResponse(responseWriter, http.StatusOK, StandardAPIResponsePayload{
		Type:    "success",
		Title:   "Email Verified",
		Message: "user verified",
	})
}
