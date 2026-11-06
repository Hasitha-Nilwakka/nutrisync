// This file contains unit tests for HashUserPassword and GenerateEmailVerificationTokenAndExpiryDate in the auth package.
package auth

import (
	"encoding/hex"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TestHashUserPasswordGeneratesValidBcryptHashAtCostTwelve verifies that HashUserPassword produces
// a valid bcrypt hash with cost 12 that matches the original plain-text password.
func TestHashUserPasswordGeneratesValidBcryptHashAtCostTwelve(testingInstance *testing.T) {
	plainTextPassword := "StrongPassw0rd!2026"

	hashedPassword, hashingError := HashUserPassword(plainTextPassword)
	if hashingError != nil {
		testingInstance.Fatalf("expected HashUserPassword to succeed, got error: %v", hashingError)
	}

	if hashedPassword == plainTextPassword {
		testingInstance.Fatal("expected hashed password to differ from plain-text password")
	}

	compareError := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainTextPassword))
	if compareError != nil {
		testingInstance.Fatalf("expected bcrypt.CompareHashAndPassword to succeed, got error: %v", compareError)
	}

	actualCost, costError := bcrypt.Cost([]byte(hashedPassword))
	if costError != nil {
		testingInstance.Fatalf("failed to inspect bcrypt cost: %v", costError)
	}

	if actualCost != 12 {
		testingInstance.Errorf("expected bcrypt cost 12, got %d", actualCost)
	}
}

// TestHashUserPasswordReturnsErrorWhenPasswordIsEmpty verifies that HashUserPassword rejects empty or whitespace-only passwords.
func TestHashUserPasswordReturnsErrorWhenPasswordIsEmpty(testingInstance *testing.T) {
	if _, hashingError := HashUserPassword(""); hashingError == nil {
		testingInstance.Fatal("expected error when hashing empty password, got nil")
	}

	if _, hashingError := HashUserPassword("   "); hashingError == nil {
		testingInstance.Fatal("expected error when hashing whitespace-only password, got nil")
	}
}

// TestGenerateEmailVerificationTokenAndExpiryDateGeneratesThirtyTwoByteHexTokenAndValidExpiry verifies
// that GenerateEmailVerificationTokenAndExpiryDate produces a 32-byte (64 hex character) random token and an accurate expiry timestamp.
func TestGenerateEmailVerificationTokenAndExpiryDateGeneratesThirtyTwoByteHexTokenAndValidExpiry(testingInstance *testing.T) {
	expirationDuration := 24 * time.Hour
	beforeGeneration := time.Now().UTC()

	firstToken, firstExpiryDate, firstError := GenerateEmailVerificationTokenAndExpiryDate(expirationDuration)
	if firstError != nil {
		testingInstance.Fatalf("expected GenerateEmailVerificationTokenAndExpiryDate to succeed, got error: %v", firstError)
	}

	afterGeneration := time.Now().UTC()

	// 32 bytes encoded as hexadecimal must be 64 characters long and decode back to 32 bytes.
	if len(firstToken) != 64 {
		testingInstance.Errorf("expected 64-character hex token, got length %d", len(firstToken))
	}

	decodedTokenBytes, decodeError := hex.DecodeString(firstToken)
	if decodeError != nil {
		testingInstance.Fatalf("expected valid hexadecimal token, got decode error: %v", decodeError)
	}

	if len(decodedTokenBytes) != 32 {
		testingInstance.Errorf("expected 32 decoded random bytes, got %d", len(decodedTokenBytes))
	}

	expectedMinimumExpiry := beforeGeneration.Add(expirationDuration)
	expectedMaximumExpiry := afterGeneration.Add(expirationDuration)
	if firstExpiryDate.Before(expectedMinimumExpiry) || firstExpiryDate.After(expectedMaximumExpiry) {
		testingInstance.Errorf("expected expiry timestamp between %v and %v, got %v", expectedMinimumExpiry, expectedMaximumExpiry, firstExpiryDate)
	}

	secondToken, _, secondError := GenerateEmailVerificationTokenAndExpiryDate(expirationDuration)
	if secondError != nil {
		testingInstance.Fatalf("expected second token generation to succeed, got error: %v", secondError)
	}

	if firstToken == secondToken {
		testingInstance.Error("expected two consecutively generated verification tokens to be unique")
	}
}

// TestGenerateEmailVerificationTokenAndExpiryDateReturnsErrorForNonPositiveDuration verifies that
// zero or negative expiration durations are rejected.
func TestGenerateEmailVerificationTokenAndExpiryDateReturnsErrorForNonPositiveDuration(testingInstance *testing.T) {
	if _, _, generationError := GenerateEmailVerificationTokenAndExpiryDate(0); generationError == nil {
		testingInstance.Fatal("expected error for zero expirationDuration, got nil")
	}

	if _, _, generationError := GenerateEmailVerificationTokenAndExpiryDate(-1 * time.Minute); generationError == nil {
		testingInstance.Fatal("expected error for negative expirationDuration, got nil")
	}
}
