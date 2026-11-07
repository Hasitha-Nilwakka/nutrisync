// This file contains unit tests for 2FA pre-authentication JWT tickets and TOTP secret/QR/passcode helpers.
package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// TestGenerateAndVerifyTwoFactorTicketRoundTripAndScopeEnforcement verifies that GenerateTwoFactorTicket
// signs a valid 5-minute HS256 JWT with scope == "2fa_preauth" and VerifyTwoFactorTicket rejects invalid tokens/scopes.
func TestGenerateAndVerifyTwoFactorTicketRoundTripAndScopeEnforcement(testingInstance *testing.T) {
	jwtSecret := "test-two-factor-jwt-secret"
	expectedUserID := "user-uuid-2fa-1"

	if _, err := GenerateTwoFactorTicket("", jwtSecret, 5*time.Minute); err == nil {
		testingInstance.Fatal("expected error when userID is empty")
	}

	if _, err := GenerateTwoFactorTicket(expectedUserID, "", 5*time.Minute); err == nil {
		testingInstance.Fatal("expected error when jwtSecret is empty")
	}

	ticket, generateErr := GenerateTwoFactorTicket(expectedUserID, jwtSecret, 5*time.Minute)
	if generateErr != nil || strings.TrimSpace(ticket) == "" {
		testingInstance.Fatalf("expected GenerateTwoFactorTicket to succeed, got ticket=%q err=%v", ticket, generateErr)
	}

	resolvedUserID, verifyErr := VerifyTwoFactorTicket(ticket, jwtSecret)
	if verifyErr != nil {
		testingInstance.Fatalf("expected VerifyTwoFactorTicket to succeed, got error: %v", verifyErr)
	}

	if resolvedUserID != expectedUserID {
		testingInstance.Errorf("expected resolved userID %q, got %q", expectedUserID, resolvedUserID)
	}

	if _, wrongSecretErr := VerifyTwoFactorTicket(ticket, "wrong-secret"); wrongSecretErr == nil {
		testingInstance.Fatal("expected error when verifying two-factor ticket with wrong JWT secret")
	}

	// Ensure a normal access token (which lacks scope == "2fa_preauth") is rejected by VerifyTwoFactorTicket.
	normalAccessToken, accessErr := GenerateSignedAccessToken(expectedUserID, "session-uuid-1", jwtSecret, 15*time.Minute)
	if accessErr != nil {
		testingInstance.Fatalf("failed to generate access token: %v", accessErr)
	}

	if _, scopeErr := VerifyTwoFactorTicket(normalAccessToken, jwtSecret); scopeErr == nil {
		testingInstance.Fatal("expected VerifyTwoFactorTicket to reject a token without scope == 2fa_preauth")
	}
}

// TestGenerateTOTPSetupDetailsAndValidateTOTPPasscode verifies TOTP secret generation, otpauth URI formatting,
// base64 PNG QR code data URL encoding, and 6-digit passcode verification.
func TestGenerateTOTPSetupDetailsAndValidateTOTPPasscode(testingInstance *testing.T) {
	if _, _, _, err := GenerateTOTPSetupDetails("", "NutriSync"); err == nil {
		testingInstance.Fatal("expected error when userEmail is empty in GenerateTOTPSetupDetails")
	}

	secretKey, otpauthURI, qrCodeBase64PNG, setupErr := GenerateTOTPSetupDetails("totp-user@example.com", "NutriSync")
	if setupErr != nil {
		testingInstance.Fatalf("expected GenerateTOTPSetupDetails to succeed, got: %v", setupErr)
	}

	if strings.TrimSpace(secretKey) == "" {
		testingInstance.Fatal("expected non-empty TOTP secretKey")
	}

	if !strings.HasPrefix(otpauthURI, "otpauth://totp/NutriSync:totp-user@example.com") {
		testingInstance.Errorf("expected otpauthURI to start with otpauth://totp/NutriSync:totp-user@example.com, got %q", otpauthURI)
	}

	if !strings.HasPrefix(qrCodeBase64PNG, "data:image/png;base64,") {
		testingInstance.Errorf("expected qrCodeBase64PNG to have data:image/png;base64, prefix, got %q", qrCodeBase64PNG)
	}

	validPasscode, codeErr := totp.GenerateCode(secretKey, time.Now().UTC())
	if codeErr != nil {
		testingInstance.Fatalf("failed to generate current TOTP code for secretKey: %v", codeErr)
	}

	if !ValidateTOTPPasscode(validPasscode, secretKey) {
		testingInstance.Fatal("expected ValidateTOTPPasscode to return true for current valid passcode")
	}

	if ValidateTOTPPasscode("000000", "") || ValidateTOTPPasscode("", secretKey) {
		testingInstance.Fatal("expected ValidateTOTPPasscode to return false for empty inputs")
	}
}
