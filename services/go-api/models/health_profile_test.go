// This file contains unit tests for HealthProfile, WeightEntry, and ActivityEntry GORM models.
package models

import (
	"reflect"
	"strings"
	"testing"
)

// TestHealthProfileWeightEntryAndActivityEntryTableNames verifies table names for the health domain models.
func TestHealthProfileWeightEntryAndActivityEntryTableNames(testingInstance *testing.T) {
	if (HealthProfile{}).TableName() != "health_profiles" {
		testingInstance.Errorf("expected table 'health_profiles', got %q", (HealthProfile{}).TableName())
	}

	if (WeightEntry{}).TableName() != "weight_entries" {
		testingInstance.Errorf("expected table 'weight_entries', got %q", (WeightEntry{}).TableName())
	}

	if (ActivityEntry{}).TableName() != "activity_entries" {
		testingInstance.Errorf("expected table 'activity_entries', got %q", (ActivityEntry{}).TableName())
	}
}

// TestHealthModelsGormStructTags verifies unique indexes and required columns on HealthProfile, WeightEntry, and ActivityEntry.
func TestHealthModelsGormStructTags(testingInstance *testing.T) {
	profileType := reflect.TypeOf(HealthProfile{})
	userIDField, found := profileType.FieldByName("UserID")
	if !found || !strings.Contains(userIDField.Tag.Get("gorm"), "uniqueIndex") {
		testingInstance.Fatal("expected HealthProfile.UserID to have a uniqueIndex gorm tag")
	}

	weightType := reflect.TypeOf(WeightEntry{})
	weightRecordedAt, foundWeightTime := weightType.FieldByName("RecordedAt")
	if !foundWeightTime || !strings.Contains(weightRecordedAt.Tag.Get("gorm"), "idx_user_weight_recorded_at") {
		testingInstance.Fatal("expected WeightEntry.RecordedAt to include uniqueIndex:idx_user_weight_recorded_at")
	}

	activityType := reflect.TypeOf(ActivityEntry{})
	activityRecordedAt, foundActivityTime := activityType.FieldByName("RecordedAt")
	if !foundActivityTime || !strings.Contains(activityRecordedAt.Tag.Get("gorm"), "idx_user_activity_recorded_at") {
		testingInstance.Fatal("expected ActivityEntry.RecordedAt to include uniqueIndex:idx_user_activity_recorded_at")
	}
}
