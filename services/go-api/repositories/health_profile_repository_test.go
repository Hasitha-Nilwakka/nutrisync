// This file contains unit tests for GormHealthProfileRepository and all HealthProfileRepository methods.
package repositories

import (
	"context"
	"testing"
	"time"

	"nutrysync/go-api/models"
)

// TestNewGormHealthProfileRepositoryValidatesDatabaseConnection verifies constructor validation and interface compliance.
func TestNewGormHealthProfileRepositoryValidatesDatabaseConnection(testingInstance *testing.T) {
	if _, err := NewGormHealthProfileRepository(nil); err == nil {
		testingInstance.Fatal("expected error when initializing NewGormHealthProfileRepository with nil *gorm.DB")
	}

	dryRunDB := createTransactionalDryRunGormDatabase(testingInstance)
	repository, initErr := NewGormHealthProfileRepository(dryRunDB)
	if initErr != nil {
		testingInstance.Fatalf("expected NewGormHealthProfileRepository to succeed, got: %v", initErr)
	}

	var repoInterface HealthProfileRepository = repository
	if repoInterface == nil {
		testingInstance.Fatal("expected non-nil HealthProfileRepository")
	}
}

// TestGormHealthProfileRepositoryMethodsValidationAndDryRunExecution verifies FindHealthProfileByUserID,
// UpsertHealthProfileWithWeightEntry, CreateWeightEntry, ListWeightEntriesByUserID,
// CreateActivityEntry, and ListActivityEntriesByUserID.
func TestGormHealthProfileRepositoryMethodsValidationAndDryRunExecution(testingInstance *testing.T) {
	ctx := context.Background()
	dryRunDB := createTransactionalDryRunGormDatabase(testingInstance)
	repository, _ := NewGormHealthProfileRepository(dryRunDB)

	if _, err := repository.FindHealthProfileByUserID(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in FindHealthProfileByUserID")
	}

	foundProfile, findErr := repository.FindHealthProfileByUserID(ctx, "user-1")
	if findErr != nil || foundProfile == nil {
		testingInstance.Fatalf("expected dry-run FindHealthProfileByUserID to succeed, got profile=%+v err=%v", foundProfile, findErr)
	}

	if err := repository.UpsertHealthProfileWithWeightEntry(ctx, nil, &models.WeightEntry{}); err == nil {
		testingInstance.Fatal("expected error when profile is nil in UpsertHealthProfileWithWeightEntry")
	}

	if err := repository.UpsertHealthProfileWithWeightEntry(ctx, &models.HealthProfile{UserID: "user-1"}, nil); err == nil {
		testingInstance.Fatal("expected error when weightEntry is nil in UpsertHealthProfileWithWeightEntry")
	}

	profile := &models.HealthProfile{
		ID:              "profile-1",
		UserID:          "user-1",
		Age:             29,
		Gender:          "female",
		HeightCm:        168.0,
		CurrentWeightKg: 64.0,
		TargetWeightKg:  60.0,
	}
	weightEntry := &models.WeightEntry{
		ID:         "weight-1",
		UserID:     "user-1",
		WeightKg:   64.0,
		RawValue:   64.0,
		RawUnit:    "kg",
		RecordedAt: time.Now().UTC(),
	}

	if upsertErr := repository.UpsertHealthProfileWithWeightEntry(ctx, profile, weightEntry); upsertErr != nil {
		testingInstance.Fatalf("expected dry-run UpsertHealthProfileWithWeightEntry to succeed, got: %v", upsertErr)
	}

	if err := repository.CreateWeightEntry(ctx, nil); err == nil {
		testingInstance.Fatal("expected error when weight entry is nil in CreateWeightEntry")
	}

	if createWeightErr := repository.CreateWeightEntry(ctx, weightEntry); createWeightErr != nil {
		testingInstance.Fatalf("expected dry-run CreateWeightEntry to succeed, got: %v", createWeightErr)
	}

	if _, err := repository.ListWeightEntriesByUserID(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in ListWeightEntriesByUserID")
	}

	if _, listWeightErr := repository.ListWeightEntriesByUserID(ctx, "user-1"); listWeightErr != nil {
		testingInstance.Fatalf("expected dry-run ListWeightEntriesByUserID to succeed, got: %v", listWeightErr)
	}

	if err := repository.CreateActivityEntry(ctx, nil); err == nil {
		testingInstance.Fatal("expected error when activity entry is nil in CreateActivityEntry")
	}

	if err := repository.CreateActivityEntry(ctx, &models.ActivityEntry{UserID: "user-1"}); err == nil {
		testingInstance.Fatal("expected error when activity RecordedAt is zero in CreateActivityEntry")
	}

	activityEntry := &models.ActivityEntry{
		ID:              "activity-1",
		UserID:          "user-1",
		ExerciseType:    "cardio",
		DurationMinutes: 45,
		Intensity:       "moderate",
		RecordedAt:      time.Now().UTC(),
	}
	if createActErr := repository.CreateActivityEntry(ctx, activityEntry); createActErr != nil {
		testingInstance.Fatalf("expected dry-run CreateActivityEntry to succeed, got: %v", createActErr)
	}

	if _, err := repository.ListActivityEntriesByUserID(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in ListActivityEntriesByUserID")
	}

	if _, listActErr := repository.ListActivityEntriesByUserID(ctx, "user-1"); listActErr != nil {
		testingInstance.Fatalf("expected dry-run ListActivityEntriesByUserID to succeed, got: %v", listActErr)
	}

	if err := repository.CreateWellnessSnapshot(ctx, nil); err == nil {
		testingInstance.Fatal("expected error when snapshot is nil in CreateWellnessSnapshot")
	}

	snapshot := &models.WellnessSnapshot{
		ID:                "snapshot-1",
		UserID:            "user-1",
		BMI:               22.7,
		BMIClassification: "normal weight",
		WellnessScore:     84.5,
		BMIScore:          95.0,
		ActivityScore:     80.0,
		ProgressScore:     75.0,
		HabitsScore:       85.0,
		MilestonesJSON:    "[]",
		RecordedAt:        time.Now().UTC(),
	}
	if createSnapErr := repository.CreateWellnessSnapshot(ctx, snapshot); createSnapErr != nil {
		testingInstance.Fatalf("expected dry-run CreateWellnessSnapshot to succeed, got: %v", createSnapErr)
	}

	if _, err := repository.ListWellnessSnapshotsByUserID(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in ListWellnessSnapshotsByUserID")
	}

	if _, listSnapErr := repository.ListWellnessSnapshotsByUserID(ctx, "user-1"); listSnapErr != nil {
		testingInstance.Fatalf("expected dry-run ListWellnessSnapshotsByUserID to succeed, got: %v", listSnapErr)
	}

	if err := repository.CreateAIInsightRecord(ctx, nil); err == nil {
		testingInstance.Fatal("expected error when record is nil in CreateAIInsightRecord")
	}

	insightRecord := &models.AIInsightRecord{
		ID:                   "insight-1",
		UserID:               "user-1",
		Version:              1,
		PrimaryGoalSnapshot:  "weight_loss",
		PromptStrategy:       "cot_few_shot_v1",
		HealthStatusAnalysis: "Good baseline",
		ProgressEvaluation:   "Initial evaluation",
		RecommendationsJSON:  "[]",
		WeeklySummary:        "Weekly summary",
		MonthlySummary:       "Monthly summary",
	}
	if createInsightErr := repository.CreateAIInsightRecord(ctx, insightRecord); createInsightErr != nil {
		testingInstance.Fatalf("expected dry-run CreateAIInsightRecord to succeed, got: %v", createInsightErr)
	}

	if _, err := repository.FindLatestAIInsightRecordByUserID(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in FindLatestAIInsightRecordByUserID")
	}

	if _, findInsightErr := repository.FindLatestAIInsightRecordByUserID(ctx, "user-1"); findInsightErr != nil {
		testingInstance.Fatalf("expected dry-run FindLatestAIInsightRecordByUserID to succeed, got: %v", findInsightErr)
	}

	if _, err := repository.ListAIInsightRecordsByUserID(ctx, ""); err == nil {
		testingInstance.Fatal("expected error when userID is empty in ListAIInsightRecordsByUserID")
	}

	if _, listInsightErr := repository.ListAIInsightRecordsByUserID(ctx, "user-1"); listInsightErr != nil {
		testingInstance.Fatalf("expected dry-run ListAIInsightRecordsByUserID to succeed, got: %v", listInsightErr)
	}
}
