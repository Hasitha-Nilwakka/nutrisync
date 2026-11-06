// This file contains unit tests for password verification, JWT access token signing/verification,
// and refresh token secret generation, hashing, composite cookie formatting, and constant-time comparison.
package auth

import (
	"testing"
	"time"
)

// TestVerifyUserPasswordAgainstHashValidatesMatchingAndMismatchingPasswords verifies that
// VerifyUserPasswordAgainstHash succeeds for the correct plaintext password and fails otherwise.
func TestVerifyUserPasswordAgainstHashValidatesMatchingAndMismatchingPasswords(testingInstance *testing.T) {
	plainTextPassword := "CorrectHorseBatteryStaple!99"
	passwordHash, hashError := HashUserPassword(plainTextPassword)
	if hashError != nil {
		testingInstance.Fatalf("failed to hash test password: %v", hashError)
	}

	if verifyError := VerifyUserPasswordAgainstHash(plainTextPassword, passwordHash); verifyError != nil {
		testingInstance.Fatalf("expected VerifyUserPasswordAgainstHash to succeed, got: %v", verifyError)
	}

	if verifyError := VerifyUserPasswordAgainstHash("WrongPassword!99", passwordHash); verifyError == nil {
		testingInstance.Fatal("expected VerifyUserPasswordAgainstHash to fail for wrong password")
	}

	if verifyError := VerifyUserPasswordAgainstHash("", passwordHash); verifyError == nil {
		testingInstance.Fatal("expected VerifyUserPasswordAgainstHash to fail for empty password")
	}

	if verifyError := VerifyUserPasswordAgainstHash(plainTextPassword, ""); verifyError == nil {
		testingInstance.Fatal("expected VerifyUserPasswordAgainstHash to fail for empty hash")
	}
}

// TestGenerateSignedAccessTokenAndVerifyAndParseAccessTokenRoundTrip verifies HS256 JWT generation,
// claim extraction (sub and sid), input validation, and rejection of invalid secrets or expired tokens.
func TestGenerateSignedAccessTokenAndVerifyAndParseAccessTokenRoundTrip(testingInstance *testing.T) {
	userID := "11111111-2222-3333-4444-555555555555"
	sessionID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	jwtSecret := "test-super-secret-jwt-key-2026"

	if _, err := GenerateSignedAccessToken("", sessionID, jwtSecret, 15*time.Minute); err == nil {
		testingInstance.Fatal("expected error when userID is empty")
	}

	if _, err := GenerateSignedAccessToken(userID, "", jwtSecret, 15*time.Minute); err == nil {
		testingInstance.Fatal("expected error when sessionID is empty")
	}

	if _, err := GenerateSignedAccessToken(userID, sessionID, "", 15*time.Minute); err == nil {
		testingInstance.Fatal("expected error when jwtSecret is empty")
	}

	signedToken, signError := GenerateSignedAccessToken(userID, sessionID, jwtSecret, 15*time.Minute)
	if signError != nil {
		testingInstance.Fatalf("expected GenerateSignedAccessToken to succeed, got: %v", signError)
	}

	claims, verifyError := VerifyAndParseAccessToken(signedToken, jwtSecret)
	if verifyError != nil {
		testingInstance.Fatalf("expected VerifyAndParseAccessToken to succeed, got: %v", verifyError)
	}

	if claims.Subject != userID {
		testingInstance.Errorf("expected sub %q, got %q", userID, claims.Subject)
	}

	if claims.SessionID != sessionID {
		testingInstance.Errorf("expected sid %q, got %q", sessionID, claims.SessionID)
	}

	if _, wrongSecretError := VerifyAndParseAccessToken(signedToken, "wrong-secret"); wrongSecretError == nil {
		testingInstance.Fatal("expected VerifyAndParseAccessToken to fail with wrong secret")
	}
}

// TestRefreshTokenSecretHashingFormattingAndConstantTimeVerification verifies GenerateRefreshTokenSecretAndHash,
// HashRefreshTokenSecret, FormatCompositeRefreshTokenCookieValue, ParseCompositeRefreshTokenCookieValue,
// and VerifyRefreshTokenSecretConstantTime.
func TestRefreshTokenSecretHashingFormattingAndConstantTimeVerification(testingInstance *testing.T) {
	rawSecret, secretHash, generationError := GenerateRefreshTokenSecretAndHash()
	if generationError != nil {
		testingInstance.Fatalf("expected GenerateRefreshTokenSecretAndHash to succeed, got: %v", generationError)
	}

	if len(rawSecret) != 64 {
		testingInstance.Errorf("expected 64-character hex rawSecret, got length %d", len(rawSecret))
	}

	if len(secretHash) != 64 {
		testingInstance.Errorf("expected 64-character SHA-256 hex secretHash, got length %d", len(secretHash))
	}

	if HashRefreshTokenSecret(rawSecret) != secretHash {
		testingInstance.Error("expected HashRefreshTokenSecret(rawSecret) to match secretHash")
	}

	if !VerifyRefreshTokenSecretConstantTime(rawSecret, secretHash) {
		testingInstance.Error("expected VerifyRefreshTokenSecretConstantTime to return true for matching secret and hash")
	}

	if VerifyRefreshTokenSecretConstantTime("different-secret", secretHash) {
		testingInstance.Error("expected VerifyRefreshTokenSecretConstantTime to return false for mismatching secret")
	}

	tokenID := "refresh-token-uuid-123"
	cookieValue := FormatCompositeRefreshTokenCookieValue(tokenID, rawSecret)
	parsedID, parsedSecret, parseError := ParseCompositeRefreshTokenCookieValue(cookieValue)
	if parseError != nil {
		testingInstance.Fatalf("expected ParseCompositeRefreshTokenCookieValue to succeed, got: %v", parseError)
	}

	if parsedID != tokenID || parsedSecret != rawSecret {
		testingInstance.Errorf("expected (%q, %q), got (%q, %q)", tokenID, rawSecret, parsedID, parsedSecret)
	}

	for _, malformed := range []string{"", "no-separator", ".missing-id", "missing-secret.", "a.b.c"} {
		if _, _, err := ParseCompositeRefreshTokenCookieValue(malformed); err == nil {
			testingInstance.Errorf("expected error for malformed cookie value %q, got nil", malformed)
		}
	}
}
