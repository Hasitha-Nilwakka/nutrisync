// Package auth provides cryptographic and token-generation helpers for user registration and authentication.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// bcryptPasswordHashingCost defines the work factor (12) used when hashing user passwords.
	bcryptPasswordHashingCost = 12
	// emailVerificationTokenByteLength defines the number of cryptographically random bytes (32)
	// generated before hex-encoding into a 64-character verification token.
	emailVerificationTokenByteLength = 32
)

// HashUserPassword hashes a plain-text user password using bcrypt with cost 12.
// Returns the bcrypt hash string or an error if the password is empty or hashing fails.
func HashUserPassword(plainTextPassword string) (string, error) {
	if strings.TrimSpace(plainTextPassword) == "" {
		return "", errors.New("password must not be empty")
	}

	hashedPasswordBytes, hashingError := bcrypt.GenerateFromPassword([]byte(plainTextPassword), bcryptPasswordHashingCost)
	if hashingError != nil {
		return "", hashingError
	}

	return string(hashedPasswordBytes), nil
}

// GenerateEmailVerificationTokenAndExpiryDate generates a 32-byte cryptographically secure random token
// encoded as a 64-character hexadecimal string, along with its UTC expiration timestamp.
// Returns the hex token, the computed expiry time, or an error if expirationDuration is not positive or random generation fails.
func GenerateEmailVerificationTokenAndExpiryDate(expirationDuration time.Duration) (string, time.Time, error) {
	if expirationDuration <= 0 {
		return "", time.Time{}, errors.New("expiration duration must be greater than zero")
	}

	randomTokenBytes := make([]byte, emailVerificationTokenByteLength)
	if _, readError := rand.Read(randomTokenBytes); readError != nil {
		return "", time.Time{}, readError
	}

	verificationTokenHex := hex.EncodeToString(randomTokenBytes)
	expirationTimestamp := time.Now().UTC().Add(expirationDuration)

	return verificationTokenHex, expirationTimestamp, nil
}
