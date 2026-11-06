// This file contains unit tests verifying the GORM schema tags and JSON serialization security rules of models.User.
package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestUserModelGormStructTagsMatchRegistrationRequirements uses reflection to verify that User struct fields
// retain required GORM constraints (primary key UUID generation, unique email index, not-null rules, and token index).
func TestUserModelGormStructTagsMatchRegistrationRequirements(testingInstance *testing.T) {
	userType := reflect.TypeOf(User{})

	expectedGormTagsByField := map[string][]string{
		"ID":                {"primaryKey", "default:gen_random_uuid()"},
		"Email":             {"uniqueIndex", "not null"},
		"Password":          {"not null"},
		"Consented":         {"not null"},
		"Verified":          {"default:false"},
		"VerificationToken": {"index"},
	}

	for fieldName, expectedTagSubstrings := range expectedGormTagsByField {
		structField, found := userType.FieldByName(fieldName)
		if !found {
			testingInstance.Fatalf("expected field %q to exist on User model", fieldName)
		}

		gormTag := structField.Tag.Get("gorm")
		for _, expectedSubstring := range expectedTagSubstrings {
			if !strings.Contains(gormTag, expectedSubstring) {
				testingInstance.Errorf("expected field %q gorm tag %q to contain %q", fieldName, gormTag, expectedSubstring)
			}
		}
	}
}

// TestUserModelJSONSerializationOmitsSensitiveFields ensures that marshaling a User instance to JSON
// never exposes sensitive authentication fields such as Password or VerificationToken.
func TestUserModelJSONSerializationOmitsSensitiveFields(testingInstance *testing.T) {
	verificationToken := "secret-verification-token"
	expirationTime := time.Now().Add(24 * time.Hour)

	userInstance := User{
		ID:                         "11111111-1111-1111-1111-111111111111",
		Email:                      "user@example.com",
		Password:                   "hashed-password-secret",
		Consented:                  true,
		Verified:                   false,
		VerificationToken:          &verificationToken,
		VerificationTokenExpiredAt: &expirationTime,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}

	serializedJSONBytes, marshalError := json.Marshal(userInstance)
	if marshalError != nil {
		testingInstance.Fatalf("failed to marshal User struct to JSON: %v", marshalError)
	}

	serializedJSONString := string(serializedJSONBytes)
	if strings.Contains(serializedJSONString, "hashed-password-secret") {
		testingInstance.Errorf("expected JSON output not to expose Password")
	}

	if strings.Contains(serializedJSONString, "secret-verification-token") {
		testingInstance.Errorf("expected JSON output not to expose VerificationToken")
	}
}
