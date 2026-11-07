// This file contains unit tests verifying TableName and GORM struct tags for the Account model in models/account.go.
package models

import (
	"reflect"
	"strings"
	"testing"
)

// TestAccountTableNameReturnsAccounts verifies that Account.TableName() returns "accounts".
func TestAccountTableNameReturnsAccounts(testingInstance *testing.T) {
	accountModel := Account{}
	if accountModel.TableName() != "accounts" {
		testingInstance.Errorf("expected Account table name %q, got %q", "accounts", accountModel.TableName())
	}
}

// TestAccountGormStructTagsMatchOAuthRequirements verifies that Account fields define the expected
// columns, foreign keys, and composite unique index (idx_provider_user).
func TestAccountGormStructTagsMatchOAuthRequirements(testingInstance *testing.T) {
	accountType := reflect.TypeOf(Account{})

	expectedGormTagsByField := map[string][]string{
		"ID":             {"primaryKey", "default:gen_random_uuid()"},
		"UserID":         {"column:user_id", "not null", "index"},
		"Provider":       {"column:provider", "size:50", "not null", "uniqueIndex:idx_provider_user"},
		"ProviderUserID": {"column:provider_user_id", "size:255", "not null", "uniqueIndex:idx_provider_user"},
		"ProviderEmail":  {"column:provider_email", "size:255", "not null"},
	}

	for fieldName, expectedSubstrings := range expectedGormTagsByField {
		structField, found := accountType.FieldByName(fieldName)
		if !found {
			testingInstance.Fatalf("expected field %q on Account struct", fieldName)
		}

		gormTag := structField.Tag.Get("gorm")
		for _, expectedSubstring := range expectedSubstrings {
			if !strings.Contains(gormTag, expectedSubstring) {
				testingInstance.Errorf("expected Account.%s gorm tag %q to contain %q", fieldName, gormTag, expectedSubstring)
			}
		}
	}
}
