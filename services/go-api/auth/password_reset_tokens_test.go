package auth

import (
	"testing"
	"time"
)

func TestGeneratePasswordResetTokenAndExpiry(t *testing.T) {
	rawToken, tokenHash, expiresAt, err := GeneratePasswordResetTokenAndExpiry(1 * time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(rawToken) != 64 {
		t.Fatalf("expected 64-char hex raw token, got length %d", len(rawToken))
	}

	if len(tokenHash) != 64 {
		t.Fatalf("expected 64-char SHA-256 hex digest, got length %d", len(tokenHash))
	}

	if rawToken == tokenHash {
		t.Fatalf("expected raw token and SHA-256 hash to differ")
	}

	if HashPasswordResetToken(rawToken) != tokenHash {
		t.Fatalf("expected HashPasswordResetToken to match returned tokenHash")
	}

	if !expiresAt.After(time.Now().UTC().Add(55 * time.Minute)) {
		t.Fatalf("expected expiry ~1h in the future, got %v", expiresAt)
	}
}

func TestGeneratePasswordResetTokenAndExpiry_RejectsNonPositiveTTL(t *testing.T) {
	if _, _, _, err := GeneratePasswordResetTokenAndExpiry(0); err == nil {
		t.Fatalf("expected error for zero TTL")
	}

	if _, _, _, err := GeneratePasswordResetTokenAndExpiry(-5 * time.Minute); err == nil {
		t.Fatalf("expected error for negative TTL")
	}
}
