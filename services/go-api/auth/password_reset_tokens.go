// This file provides cryptographically random token generation and SHA-256 hashing helpers
// used by the password reset workflow so raw reset tokens are never persisted in PostgreSQL.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const passwordResetTokenByteLength = 32

// HashPasswordResetToken computes the lowercase SHA-256 hex digest of a raw password reset token.
func HashPasswordResetToken(rawResetToken string) string {
	trimmedToken := strings.TrimSpace(rawResetToken)
	digest := sha256.Sum256([]byte(trimmedToken))
	return hex.EncodeToString(digest[:])
}

// GeneratePasswordResetTokenAndExpiry creates a 32-byte cryptographically random hex token,
// its SHA-256 hex digest for database storage, and a UTC expiration timestamp offset by validityDuration.
func GeneratePasswordResetTokenAndExpiry(validityDuration time.Duration) (string, string, time.Time, error) {
	if validityDuration <= 0 {
		return "", "", time.Time{}, errors.New("password reset token validity duration must be positive")
	}

	randomTokenBytes := make([]byte, passwordResetTokenByteLength)
	if _, readError := rand.Read(randomTokenBytes); readError != nil {
		return "", "", time.Time{}, readError
	}

	rawResetToken := hex.EncodeToString(randomTokenBytes)
	tokenHash := HashPasswordResetToken(rawResetToken)
	expiresAt := time.Now().UTC().Add(validityDuration)

	return rawResetToken, tokenHash, expiresAt, nil
}
