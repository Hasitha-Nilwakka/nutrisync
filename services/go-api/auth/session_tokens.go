// This file provides password verification, HS256 JWT access token generation/validation,
// and cryptographic refresh token secret generation, hashing, cookie formatting, and constant-time comparison helpers.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultAccessTokenTimeToLive = 15 * time.Minute
	refreshTokenSecretByteLength = 32
	compositeCookieSeparator     = "."
)

// AccessTokenClaims represents the JWT payload claims signed into an access token.
// Subject ("sub") stores the user ID, and SessionID ("sid") stores the session ID.
type AccessTokenClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// VerifyUserPasswordAgainstHash compares a plaintext password against a stored bcrypt password hash.
// Returns nil if the password matches the hash, or an error if either value is empty or comparison fails.
func VerifyUserPasswordAgainstHash(plainTextPassword, passwordHash string) error {
	if strings.TrimSpace(plainTextPassword) == "" {
		return errors.New("plain-text password must not be empty")
	}

	if strings.TrimSpace(passwordHash) == "" {
		return errors.New("password hash must not be empty")
	}

	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(plainTextPassword))
}

// GenerateSignedAccessToken signs an HS256 JWT containing sub: userID, sid: sessionID, iat, and exp.
// Defaults ttl to 15 minutes when ttl <= 0, and rejects empty userID, sessionID, or jwtSecret.
func GenerateSignedAccessToken(userID, sessionID, jwtSecret string, ttl time.Duration) (string, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return "", errors.New("user ID must not be empty")
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return "", errors.New("session ID must not be empty")
	}

	trimmedJWTSecret := strings.TrimSpace(jwtSecret)
	if trimmedJWTSecret == "" {
		return "", errors.New("JWT secret must not be empty")
	}

	effectiveTTL := ttl
	if effectiveTTL <= 0 {
		effectiveTTL = defaultAccessTokenTimeToLive
	}

	currentTimeUTC := time.Now().UTC()
	claims := AccessTokenClaims{
		SessionID: trimmedSessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   trimmedUserID,
			IssuedAt:  jwt.NewNumericDate(currentTimeUTC),
			ExpiresAt: jwt.NewNumericDate(currentTimeUTC.Add(effectiveTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(trimmedJWTSecret))
}

// VerifyAndParseAccessToken validates an HS256 JWT string against jwtSecret, checks expiration,
// and returns the parsed *AccessTokenClaims.
func VerifyAndParseAccessToken(tokenString, jwtSecret string) (*AccessTokenClaims, error) {
	trimmedTokenString := strings.TrimSpace(tokenString)
	if trimmedTokenString == "" {
		return nil, errors.New("access token string must not be empty")
	}

	trimmedJWTSecret := strings.TrimSpace(jwtSecret)
	if trimmedJWTSecret == "" {
		return nil, errors.New("JWT secret must not be empty")
	}

	parsedClaims := &AccessTokenClaims{}
	parsedToken, parseError := jwt.ParseWithClaims(
		trimmedTokenString,
		parsedClaims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(trimmedJWTSecret), nil
		},
	)
	if parseError != nil {
		return nil, parseError
	}

	if !parsedToken.Valid {
		return nil, errors.New("access token is invalid")
	}

	if strings.TrimSpace(parsedClaims.Subject) == "" || strings.TrimSpace(parsedClaims.SessionID) == "" {
		return nil, errors.New("access token is missing required sub or sid claims")
	}

	return parsedClaims, nil
}

// HashRefreshTokenSecret computes and returns the lowercase hexadecimal SHA-256 digest of rawSecret.
func HashRefreshTokenSecret(rawSecret string) string {
	digest := sha256.Sum256([]byte(rawSecret))
	return hex.EncodeToString(digest[:])
}

// GenerateRefreshTokenSecretAndHash generates 32 cryptographically random bytes encoded as a
// 64-character hex string (rawSecret) along with its SHA-256 hex digest (secretHash).
func GenerateRefreshTokenSecretAndHash() (rawSecret string, secretHash string, err error) {
	randomBytes := make([]byte, refreshTokenSecretByteLength)
	if _, readError := rand.Read(randomBytes); readError != nil {
		return "", "", readError
	}

	rawSecret = hex.EncodeToString(randomBytes)
	secretHash = HashRefreshTokenSecret(rawSecret)
	return rawSecret, secretHash, nil
}

// FormatCompositeRefreshTokenCookieValue joins refreshTokenID and rawSecret into "<refreshTokenID>.<rawSecret>".
func FormatCompositeRefreshTokenCookieValue(refreshTokenID, rawSecret string) string {
	return strings.TrimSpace(refreshTokenID) + compositeCookieSeparator + strings.TrimSpace(rawSecret)
}

// ParseCompositeRefreshTokenCookieValue splits a "<refreshTokenID>.<rawSecret>" cookie value and validates both parts.
func ParseCompositeRefreshTokenCookieValue(cookieValue string) (refreshTokenID string, rawSecret string, err error) {
	trimmedCookieValue := strings.TrimSpace(cookieValue)
	if trimmedCookieValue == "" {
		return "", "", errors.New("refresh token cookie value must not be empty")
	}

	parts := strings.Split(trimmedCookieValue, compositeCookieSeparator)
	if len(parts) != 2 {
		return "", "", errors.New("malformed refresh token cookie value")
	}

	refreshTokenID = strings.TrimSpace(parts[0])
	rawSecret = strings.TrimSpace(parts[1])
	if refreshTokenID == "" || rawSecret == "" {
		return "", "", errors.New("refresh token ID and raw secret must not be empty")
	}

	return refreshTokenID, rawSecret, nil
}

// VerifyRefreshTokenSecretConstantTime hashes rawSecret with SHA-256 and compares it against
// expectedTokenHash in constant time using crypto/subtle.ConstantTimeCompare.
func VerifyRefreshTokenSecretConstantTime(rawSecret, expectedTokenHash string) bool {
	if strings.TrimSpace(rawSecret) == "" || strings.TrimSpace(expectedTokenHash) == "" {
		return false
	}

	computedHash := HashRefreshTokenSecret(rawSecret)
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(expectedTokenHash)) == 1
}
