// This file contains unit tests verifying TableName mappings, GORM struct tags, and JSON serialization
// for the Session and RefreshToken models in models/session.go.
package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestSessionAndRefreshTokenTableNames verifies that Session.TableName() and RefreshToken.TableName()
// return "sessions" and "refresh_tokens" respectively.
func TestSessionAndRefreshTokenTableNames(testingInstance *testing.T) {
	sessionModel := Session{}
	if sessionModel.TableName() != "sessions" {
		testingInstance.Errorf("expected Session table name %q, got %q", "sessions", sessionModel.TableName())
	}

	refreshTokenModel := RefreshToken{}
	if refreshTokenModel.TableName() != "refresh_tokens" {
		testingInstance.Errorf("expected RefreshToken table name %q, got %q", "refresh_tokens", refreshTokenModel.TableName())
	}
}

// TestSessionAndRefreshTokenGormTagsAndSensitiveFieldOmission verifies required GORM tags on Session and RefreshToken
// and ensures RefreshToken.TokenHash is omitted from JSON serialization.
func TestSessionAndRefreshTokenGormTagsAndSensitiveFieldOmission(testingInstance *testing.T) {
	sessionType := reflect.TypeOf(Session{})
	userIDField, found := sessionType.FieldByName("UserID")
	if !found || !strings.Contains(userIDField.Tag.Get("gorm"), "column:user_id") {
		testingInstance.Errorf("expected Session.UserID to have column:user_id gorm tag")
	}

	refreshTokenType := reflect.TypeOf(RefreshToken{})
	tokenHashField, found := refreshTokenType.FieldByName("TokenHash")
	if !found || !strings.Contains(tokenHashField.Tag.Get("gorm"), "column:token_hash") {
		testingInstance.Errorf("expected RefreshToken.TokenHash to have column:token_hash gorm tag")
	}

	tokenInstance := RefreshToken{
		ID:        "token-id-1",
		TokenHash: "secret-sha256-hash",
		SessionID: "session-id-1",
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	serializedBytes, marshalError := json.Marshal(tokenInstance)
	if marshalError != nil {
		testingInstance.Fatalf("failed to marshal RefreshToken to JSON: %v", marshalError)
	}

	if strings.Contains(string(serializedBytes), "secret-sha256-hash") {
		testingInstance.Error("expected RefreshToken JSON serialization not to expose TokenHash")
	}
}
