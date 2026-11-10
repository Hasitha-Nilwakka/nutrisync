// This file contains unit tests for the WellnessSnapshot and AIInsightRecord GORM models.
package models

import (
	"reflect"
	"strings"
	"testing"
)

// TestWellnessSnapshotAndAIInsightRecordTableNames verifies the PostgreSQL table names for WellnessSnapshot and AIInsightRecord.
func TestWellnessSnapshotAndAIInsightRecordTableNames(testingInstance *testing.T) {
	if (WellnessSnapshot{}).TableName() != "wellness_snapshots" {
		testingInstance.Errorf("expected table 'wellness_snapshots', got %q", (WellnessSnapshot{}).TableName())
	}

	if (AIInsightRecord{}).TableName() != "ai_insight_records" {
		testingInstance.Errorf("expected table 'ai_insight_records', got %q", (AIInsightRecord{}).TableName())
	}
}

// TestWellnessSnapshotAndAIInsightRecordGormAndJSONTags verifies required struct fields and tags on WellnessSnapshot and AIInsightRecord.
func TestWellnessSnapshotAndAIInsightRecordGormAndJSONTags(testingInstance *testing.T) {
	snapshotType := reflect.TypeOf(WellnessSnapshot{})
	requiredSnapshotFields := map[string]string{
		"ID":                "id",
		"UserID":            "userId",
		"BMI":               "bmi",
		"BMIClassification": "bmiClassification",
		"WellnessScore":     "wellnessScore",
		"BMIScore":          "bmiScore",
		"ActivityScore":     "activityScore",
		"ProgressScore":     "progressScore",
		"HabitsScore":       "habitsScore",
		"MilestonesJSON":    "milestonesJson",
		"RecordedAt":        "recordedAt",
		"CreatedAt":         "createdAt",
	}

	for fieldName, expectedJSONTag := range requiredSnapshotFields {
		field, found := snapshotType.FieldByName(fieldName)
		if !found {
			testingInstance.Fatalf("expected WellnessSnapshot.%s field to exist", fieldName)
		}
		if field.Tag.Get("json") != expectedJSONTag {
			testingInstance.Errorf("expected WellnessSnapshot.%s json tag %q, got %q", fieldName, expectedJSONTag, field.Tag.Get("json"))
		}
	}

	userField, foundUser := snapshotType.FieldByName("User")
	if !foundUser || !strings.Contains(userField.Tag.Get("gorm"), "OnDelete:CASCADE") || userField.Tag.Get("json") != "-" {
		testingInstance.Fatal("expected WellnessSnapshot.User to have OnDelete:CASCADE and json:\"-\"")
	}

	insightType := reflect.TypeOf(AIInsightRecord{})
	requiredInsightFields := map[string]string{
		"ID":                   "id",
		"UserID":               "userId",
		"Version":              "version",
		"PrimaryGoalSnapshot":  "primaryGoalSnapshot",
		"PromptStrategy":       "promptStrategy",
		"HealthStatusAnalysis": "healthStatusAnalysis",
		"ProgressEvaluation":   "progressEvaluation",
		"RecommendationsJSON":  "recommendationsJson",
		"WeeklySummary":        "weeklySummary",
		"MonthlySummary":       "monthlySummary",
		"CreatedAt":            "createdAt",
	}

	for fieldName, expectedJSONTag := range requiredInsightFields {
		field, found := insightType.FieldByName(fieldName)
		if !found {
			testingInstance.Fatalf("expected AIInsightRecord.%s field to exist", fieldName)
		}
		if field.Tag.Get("json") != expectedJSONTag {
			testingInstance.Errorf("expected AIInsightRecord.%s json tag %q, got %q", fieldName, expectedJSONTag, field.Tag.Get("json"))
		}
	}
}
