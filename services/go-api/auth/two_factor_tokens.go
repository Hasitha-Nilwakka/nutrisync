// This file provides two-factor authentication (2FA) pre-authentication JWT ticket generation/validation
// and TOTP secret key, otpauth URI, QR code PNG data URL generation, and 6-digit passcode verification helpers.
package auth

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
)

const (
	// TwoFactorPreAuthScope is the JWT scope claim required for pre-authenticated 2FA login verification tickets.
	TwoFactorPreAuthScope             = "2fa_preauth"
	defaultTwoFactorTicketTimeToLive  = 5 * time.Minute
	defaultTOTPIssuerName             = "NutriSync"
	totpSecretSizeInBytes             = 20
	totpQRCodeImageDimensionInPixels  = 200
)

// TwoFactorTicketClaims represents the JWT payload claims signed into a short-lived 2FA login ticket.
type TwoFactorTicketClaims struct {
	Scope string `json:"scope"`
	jwt.RegisteredClaims
}

// GenerateTwoFactorTicket signs a short-lived HS256 JWT containing sub: userID, scope: "2fa_preauth", iat, and exp.
// Defaults ttl to 5 minutes when ttl <= 0, and rejects empty userID or jwtSecret.
func GenerateTwoFactorTicket(userID, jwtSecret string, ttl time.Duration) (string, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return "", errors.New("user ID must not be empty")
	}

	trimmedJWTSecret := strings.TrimSpace(jwtSecret)
	if trimmedJWTSecret == "" {
		return "", errors.New("JWT secret must not be empty")
	}

	effectiveTTL := ttl
	if effectiveTTL <= 0 {
		effectiveTTL = defaultTwoFactorTicketTimeToLive
	}

	currentTimeUTC := time.Now().UTC()
	claims := TwoFactorTicketClaims{
		Scope: TwoFactorPreAuthScope,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   trimmedUserID,
			IssuedAt:  jwt.NewNumericDate(currentTimeUTC),
			ExpiresAt: jwt.NewNumericDate(currentTimeUTC.Add(effectiveTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(trimmedJWTSecret))
}

// VerifyTwoFactorTicket validates an HS256 2FA ticket JWT against jwtSecret, ensures scope == "2fa_preauth",
// checks expiration, and returns the subject userID.
func VerifyTwoFactorTicket(ticketString, jwtSecret string) (userID string, err error) {
	trimmedTicketString := strings.TrimSpace(ticketString)
	if trimmedTicketString == "" {
		return "", errors.New("two-factor ticket must not be empty")
	}

	trimmedJWTSecret := strings.TrimSpace(jwtSecret)
	if trimmedJWTSecret == "" {
		return "", errors.New("JWT secret must not be empty")
	}

	parsedClaims := &TwoFactorTicketClaims{}
	parsedToken, parseError := jwt.ParseWithClaims(
		trimmedTicketString,
		parsedClaims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(trimmedJWTSecret), nil
		},
	)
	if parseError != nil {
		return "", parseError
	}

	if !parsedToken.Valid {
		return "", errors.New("two-factor ticket is invalid")
	}

	if parsedClaims.Scope != TwoFactorPreAuthScope {
		return "", errors.New("invalid two-factor ticket scope")
	}

	trimmedSubject := strings.TrimSpace(parsedClaims.Subject)
	if trimmedSubject == "" {
		return "", errors.New("two-factor ticket is missing subject user ID")
	}

	return trimmedSubject, nil
}

// GenerateTOTPSetupDetails generates a 20-byte TOTP secret key, its otpauth:// URI,
// and a base64-encoded PNG QR code data URL ("data:image/png;base64,...") for userEmail and issuer (defaulting to "NutriSync").
func GenerateTOTPSetupDetails(userEmail, issuer string) (secretKey string, otpauthURI string, qrCodeBase64PNG string, err error) {
	trimmedUserEmail := strings.TrimSpace(userEmail)
	if trimmedUserEmail == "" {
		return "", "", "", errors.New("user email must not be empty")
	}

	effectiveIssuer := strings.TrimSpace(issuer)
	if effectiveIssuer == "" {
		effectiveIssuer = defaultTOTPIssuerName
	}

	totpKey, generateError := totp.Generate(totp.GenerateOpts{
		Issuer:      effectiveIssuer,
		AccountName: trimmedUserEmail,
		SecretSize:  totpSecretSizeInBytes,
	})
	if generateError != nil {
		return "", "", "", generateError
	}

	qrCodeImage, imageError := totpKey.Image(totpQRCodeImageDimensionInPixels, totpQRCodeImageDimensionInPixels)
	if imageError != nil {
		return "", "", "", imageError
	}

	var pngBuffer bytes.Buffer
	if encodeError := png.Encode(&pngBuffer, qrCodeImage); encodeError != nil {
		return "", "", "", encodeError
	}

	qrCodeDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBuffer.Bytes())
	return totpKey.Secret(), totpKey.URL(), qrCodeDataURL, nil
}

// ValidateTOTPPasscode validates a 6-digit TOTP passcode against secretKey using totp.Validate.
func ValidateTOTPPasscode(passcode, secretKey string) bool {
	trimmedPasscode := strings.TrimSpace(passcode)
	trimmedSecretKey := strings.TrimSpace(secretKey)
	if trimmedPasscode == "" || trimmedSecretKey == "" {
		return false
	}

	return totp.Validate(trimmedPasscode, trimmedSecretKey)
}
