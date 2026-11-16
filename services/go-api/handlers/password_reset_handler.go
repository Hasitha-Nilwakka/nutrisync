// This file implements HTTP handlers for requesting a password reset email (POST /api/auth/passwordreset)
// and confirming a new password with a reset token (POST /api/auth/passwordreset/confirm).
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"nutrysync/go-api/auth"
	"nutrysync/go-api/repositories"
)

const (
	passwordResetTokenValidityDuration = 1 * time.Hour
	minimumPasswordResetLength         = 8
)

// PasswordResetRequestPayload represents the JSON body accepted by POST /api/auth/passwordreset.
type PasswordResetRequestPayload struct {
	EmailReset string `json:"emailReset"`
}

// PasswordResetConfirmPayload represents the JSON body accepted by POST /api/auth/passwordreset/confirm.
type PasswordResetConfirmPayload struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

// PasswordResetHandler coordinates password reset link generation, email delivery, and password updates.
type PasswordResetHandler struct {
	passwordResetRepository  repositories.PasswordResetRepository
	passwordResetEmailSender auth.PasswordResetEmailSender
}

// NewPasswordResetHandler constructs a PasswordResetHandler with the required repository and email sender dependencies.
func NewPasswordResetHandler(
	passwordResetRepository repositories.PasswordResetRepository,
	passwordResetEmailSender auth.PasswordResetEmailSender,
) (*PasswordResetHandler, error) {
	if passwordResetRepository == nil {
		return nil, errors.New("password reset repository is required")
	}
	if passwordResetEmailSender == nil {
		return nil, errors.New("password reset email sender is required")
	}

	return &PasswordResetHandler{
		passwordResetRepository:  passwordResetRepository,
		passwordResetEmailSender: passwordResetEmailSender,
	}, nil
}

// validatePasswordResetEmailAddress verifies that the submitted email address is non-empty and syntactically valid.
func validatePasswordResetEmailAddress(rawEmail string) (string, error) {
	trimmedEmail := strings.TrimSpace(rawEmail)
	if trimmedEmail == "" {
		return "", errors.New("Please enter a valid email address.")
	}

	parsedAddress, parseError := mail.ParseAddress(trimmedEmail)
	if parseError != nil || parsedAddress == nil || !strings.EqualFold(parsedAddress.Address, trimmedEmail) {
		return "", errors.New("Please enter a valid email address.")
	}

	return strings.ToLower(trimmedEmail), nil
}

// HandleRequestPasswordReset handles POST /api/auth/passwordreset, issuing a 1-hour reset token and email
// for password-enabled accounts while returning a generic 200 OK response to prevent email enumeration.
func (handler *PasswordResetHandler) HandleRequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var requestPayload PasswordResetRequestPayload
	if decodeError := json.NewDecoder(r.Body).Decode(&requestPayload); decodeError != nil {
		WriteJSONResponse(w, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Invalid Request",
			Message: "Please provide a valid JSON payload with your email address.",
		})
		return
	}

	normalizedEmail, validationError := validatePasswordResetEmailAddress(requestPayload.EmailReset)
	if validationError != nil {
		WriteJSONResponse(w, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Invalid Email",
			Message: validationError.Error(),
		})
		return
	}

	userRecord, lookupError := handler.passwordResetRepository.FindUserByEmailForPasswordReset(r.Context(), normalizedEmail)
	if lookupError != nil {
		log.Printf("password reset user lookup failed: %v", lookupError)
		WriteJSONResponse(w, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Unable to Send Reset Link",
			Message: "An unexpected error occurred. Please try again later.",
		})
		return
	}

	genericSuccessResponse := StandardAPIResponsePayload{
		Type:    "success",
		Title:   "Password Reset Link Sent",
		Message: "If an account exists for that email, a password reset link has been sent.",
	}

	// Return the same generic success response for unknown, OAuth-only, or unverified accounts to prevent enumeration.
	if userRecord == nil || strings.TrimSpace(userRecord.Password) == "" || !userRecord.Verified {
		WriteJSONResponse(w, http.StatusOK, genericSuccessResponse)
		return
	}

	rawToken, tokenHash, expiresAt, tokenError := auth.GeneratePasswordResetTokenAndExpiry(passwordResetTokenValidityDuration)
	if tokenError != nil {
		log.Printf("password reset token generation failed: %v", tokenError)
		WriteJSONResponse(w, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Unable to Send Reset Link",
			Message: "An unexpected error occurred. Please try again later.",
		})
		return
	}

	if persistError := handler.passwordResetRepository.ReplaceActivePasswordResetToken(
		r.Context(),
		userRecord.ID,
		tokenHash,
		expiresAt,
	); persistError != nil {
		log.Printf("password reset token persistence failed: %v", persistError)
		WriteJSONResponse(w, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Unable to Send Reset Link",
			Message: "An unexpected error occurred. Please try again later.",
		})
		return
	}

	if sendError := handler.passwordResetEmailSender.SendPasswordResetEmail(userRecord.Email, rawToken); sendError != nil {
		log.Printf("password reset email delivery failed: %v", sendError)
		WriteJSONResponse(w, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Unable to Send Reset Link",
			Message: "We could not send the password reset email right now. Please try again later.",
		})
		return
	}

	WriteJSONResponse(w, http.StatusOK, genericSuccessResponse)
}

// HandleConfirmPasswordReset handles POST /api/auth/passwordreset/confirm, validating the reset token,
// updating the user's bcrypt password hash, marking the token used, and revoking active sessions.
func (handler *PasswordResetHandler) HandleConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var requestPayload PasswordResetConfirmPayload
	if decodeError := json.NewDecoder(r.Body).Decode(&requestPayload); decodeError != nil {
		WriteJSONResponse(w, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Invalid Request",
			Message: "Please provide a valid JSON payload.",
		})
		return
	}

	rawToken := strings.TrimSpace(requestPayload.Token)
	trimmedNewPassword := strings.TrimSpace(requestPayload.NewPassword)

	if rawToken == "" {
		WriteJSONResponse(w, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Invalid or Expired Link",
			Message: "This password reset link is invalid or has expired. Please request a new one.",
		})
		return
	}

	if len(trimmedNewPassword) < minimumPasswordResetLength {
		WriteJSONResponse(w, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Invalid Password",
			Message: "Password must be at least 8 characters long.",
		})
		return
	}

	tokenHash := auth.HashPasswordResetToken(rawToken)
	resetRecord, lookupError := handler.passwordResetRepository.FindPasswordResetByTokenHash(r.Context(), tokenHash)
	if lookupError != nil {
		log.Printf("password reset token lookup failed: %v", lookupError)
		WriteJSONResponse(w, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Password Reset Failed",
			Message: "An unexpected error occurred. Please try again later.",
		})
		return
	}

	nowUTC := time.Now().UTC()
	if resetRecord == nil ||
		resetRecord.Verified ||
		resetRecord.RevokedAt != nil ||
		nowUTC.After(resetRecord.VerificationTokenExpiry) {
		WriteJSONResponse(w, http.StatusBadRequest, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Invalid or Expired Link",
			Message: "This password reset link is invalid or has expired. Please request a new one.",
		})
		return
	}

	newPasswordHash, hashError := auth.HashUserPassword(trimmedNewPassword)
	if hashError != nil {
		log.Printf("password reset bcrypt hashing failed: %v", hashError)
		WriteJSONResponse(w, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Password Reset Failed",
			Message: "An unexpected error occurred. Please try again later.",
		})
		return
	}

	if completeError := handler.passwordResetRepository.CompletePasswordResetAndRevokeSessions(
		r.Context(),
		resetRecord.ID,
		resetRecord.UserID,
		newPasswordHash,
		nowUTC,
	); completeError != nil {
		log.Printf("password reset completion failed: %v", completeError)
		WriteJSONResponse(w, http.StatusInternalServerError, StandardAPIResponsePayload{
			Type:    "error",
			Title:   "Password Reset Failed",
			Message: "An unexpected error occurred. Please try again later.",
		})
		return
	}

	WriteJSONResponse(w, http.StatusOK, StandardAPIResponsePayload{
		Type:    "success",
		Title:   "Password Updated",
		Message: "Your password has been reset. You can now log in with your new password.",
	})
}
