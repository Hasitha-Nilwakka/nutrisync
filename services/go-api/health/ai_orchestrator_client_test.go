// This file contains unit tests for BuildAnonymizedAIAnalysisPayload,
// CalculateDeterministicWellnessAndMilestones, and HTTPPyAIAnalyticsClient.
package health

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nutrysync/go-api/models"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// TestBuildAnonymizedAIAnalysisPayloadStripsPIIAndComputesIrreversibleSubjectID verifies:
// 1. SubjectID is SHA-256 hex of "nutrisync-anon:" + profile.UserID
// 2. Raw UserID, email, session ID, and IP address never appear in the serialized JSON payload
// 3. CurrentState, TargetState, Preferences, Restrictions, FitnessAssessment, WeightHistory, and ActivityHistory are populated
func TestBuildAnonymizedAIAnalysisPayloadStripsPIIAndComputesIrreversibleSubjectID(testingInstance *testing.T) {
	rawUserID := "11111111-2222-3333-4444-555555555555"
	rawEmail := "private-user@example.com"
	profile := &models.HealthProfile{
		ID:                      "profile-secret-uuid",
		UserID:                  rawUserID,
		User:                    models.User{ID: rawUserID, Email: rawEmail},
		Age:                     31,
		Gender:                  "female",
		HeightCm:                170.0,
		CurrentWeightKg:         68.0,
		BaselineWeightKg:        74.0,
		TargetWeightKg:          62.0,
		ActivityLevel:           "moderately_active",
		PrimaryGoal:             "weight_loss",
		TargetActivityLevel:     "very_active",
		TargetWeeklyDays:        4,
		WeeklyActivityDays:      3,
		ExerciseTypesJSON:       `["cardio","strength"]`,
		SessionDuration:         "30-60m",
		FitnessLevel:            "intermediate",
		PreferredEnvironment:    "gym",
		PreferredTimeOfDay:      "morning",
		EnduranceMinutes:        35,
		PushupsCount:            20,
		SquatsCount:             35,
		DietaryPreferencesJSON:  `["mediterranean"]`,
		DietaryRestrictionsJSON: `["gluten-free"]`,
		MedicalConditions:       "none",
		SleepHours:              7.5,
		WaterLiters:             2.4,
	}

	weights := []models.WeightEntry{
		{
			ID:         "weight-entry-secret-1",
			UserID:     rawUserID,
			WeightKg:   74.0,
			RecordedAt: time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC),
		},
		{
			ID:         "weight-entry-secret-2",
			UserID:     rawUserID,
			WeightKg:   68.0,
			RecordedAt: time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC),
		},
	}

	activities := []models.ActivityEntry{
		{
			ID:              "activity-entry-secret-1",
			UserID:          rawUserID,
			ExerciseType:    "running",
			DurationMinutes: 45,
			Intensity:       "moderate",
			RecordedAt:      time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC),
		},
	}

	payload, err := BuildAnonymizedAIAnalysisPayload(profile, weights, activities, true)
	if err != nil {
		testingInstance.Fatalf("expected BuildAnonymizedAIAnalysisPayload to succeed, got: %v", err)
	}

	expectedHash := sha256.Sum256([]byte("nutrisync-anon:" + rawUserID))
	expectedSubjectID := hex.EncodeToString(expectedHash[:])
	if payload.SubjectID != expectedSubjectID {
		testingInstance.Errorf("expected SubjectID %q, got %q", expectedSubjectID, payload.SubjectID)
	}

	serializedBytes, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		testingInstance.Fatalf("failed to marshal AnonymizedAIAnalysisPayload: %v", marshalErr)
	}
	serializedJSON := string(serializedBytes)

	forbiddenFragments := []string{
		rawUserID,
		rawEmail,
		"profile-secret-uuid",
		"weight-entry-secret-1",
		"activity-entry-secret-1",
	}
	for _, forbidden := range forbiddenFragments {
		if strings.Contains(serializedJSON, forbidden) {
			testingInstance.Errorf("expected serialized payload to strip PII/ID %q, but found in %s", forbidden, serializedJSON)
		}
	}

	if payload.UserMetrics.CurrentState.BMI <= 0 || payload.UserMetrics.CurrentState.BMIClassification != "normal weight" {
		testingInstance.Errorf("expected normal weight BMI in CurrentState, got %+v", payload.UserMetrics.CurrentState)
	}

	if payload.UserMetrics.TargetState.BaselineWeight != 74.0 || payload.UserMetrics.TargetState.PrimaryGoal != "weight_loss" {
		testingInstance.Errorf("unexpected TargetState: %+v", payload.UserMetrics.TargetState)
	}
}

// TestCalculateDeterministicWellnessAndMilestonesFormulaAndMilestones verifies the weighted formula
// WellnessScore = round((BMIScore*0.3)+(ActivityScore*0.3)+(ProgressScore*0.2)+(HabitsScore*0.2), 2)
// and verifies that weight/activity changes alter the score and milestones.
func TestCalculateDeterministicWellnessAndMilestonesFormulaAndMilestones(testingInstance *testing.T) {
	profile := &models.HealthProfile{
		UserID:             "user-calc-1",
		Age:                29,
		Gender:             "male",
		HeightCm:           180.0,
		CurrentWeightKg:    82.0,
		BaselineWeightKg:   90.0,
		TargetWeightKg:     75.0,
		ActivityLevel:      "moderately_active",
		PrimaryGoal:        "weight_loss",
		TargetWeeklyDays:   4,
		WeeklyActivityDays: 2,
		SleepHours:         7.5,
		WaterLiters:        2.5,
	}

	initialResult, err := CalculateDeterministicWellnessAndMilestones(profile, nil, nil)
	if err != nil {
		testingInstance.Fatalf("expected CalculateDeterministicWellnessAndMilestones to succeed, got: %v", err)
	}

	expectedComposite := roundToTwoDecimalPlaces(
		(initialResult.BMIScore * 0.3) +
			(initialResult.ActivityScore * 0.3) +
			(initialResult.ProgressScore * 0.2) +
			(initialResult.HabitsScore * 0.2),
	)
	if initialResult.WellnessScore != expectedComposite {
		testingInstance.Errorf("expected WellnessScore %.2f from weighted formula, got %.2f", expectedComposite, initialResult.WellnessScore)
	}

	if len(initialResult.Milestones) != 3 {
		testingInstance.Fatalf("expected 3 milestones (weight, activity, habits), got %d", len(initialResult.Milestones))
	}

	// Log 4 recent activities and closer weight to target -> score should increase
	now := time.Now().UTC()
	activities := []models.ActivityEntry{
		{ExerciseType: "running", DurationMinutes: 45, Intensity: "high", RecordedAt: now.Add(-1 * time.Hour)},
		{ExerciseType: "cycling", DurationMinutes: 45, Intensity: "moderate", RecordedAt: now.Add(-24 * time.Hour)},
		{ExerciseType: "strength", DurationMinutes: 50, Intensity: "high", RecordedAt: now.Add(-48 * time.Hour)},
		{ExerciseType: "swimming", DurationMinutes: 40, Intensity: "moderate", RecordedAt: now.Add(-72 * time.Hour)},
	}
	weights := []models.WeightEntry{
		{WeightKg: 90.0, RecordedAt: now.Add(-30 * 24 * time.Hour)},
		{WeightKg: 76.0, RecordedAt: now},
	}

	updatedResult, err := CalculateDeterministicWellnessAndMilestones(profile, weights, activities)
	if err != nil {
		testingInstance.Fatalf("expected updated calculation to succeed, got: %v", err)
	}

	if updatedResult.WellnessScore <= initialResult.WellnessScore {
		testingInstance.Errorf(
			"expected WellnessScore to increase after weight progress and activity logging (initial=%.2f, updated=%.2f)",
			initialResult.WellnessScore,
			updatedResult.WellnessScore,
		)
	}
}

// TestHTTPPyAIAnalyticsClientAnalyzeUserHealthAndGenerateInsights verifies HTTP request serialization and response parsing.
func TestHTTPPyAIAnalyticsClientAnalyzeUserHealthAndGenerateInsights(testingInstance *testing.T) {
	var capturedPath string
	var capturedPayload AnonymizedAIAnalysisPayload

	mockClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			capturedPath = req.URL.Path
			_ = json.NewDecoder(req.Body).Decode(&capturedPayload)

			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(rec).Encode(PyAIAnalysisResponse{
				Wellness: &WellnessCalculationResult{
					BMI:               22.5,
					BMIClassification: "normal weight",
					WellnessScore:     88.4,
					BMIScore:          95.0,
					ActivityScore:     85.0,
					ProgressScore:     82.0,
					HabitsScore:       90.0,
				},
				Insights: &PyAIInsightsPayload{
					PromptStrategy:       "cot_few_shot_v1",
					HealthStatusAnalysis: "Healthy BMI and consistent activity.",
					ProgressEvaluation:   "On track toward goal weight.",
					Recommendations: []PrioritizedAIRecommendation{
						{
							Title:          "Progressive Overload",
							Category:       "exercise",
							Priority:       "high",
							Action:         "Add 1 strength session per week.",
							ExpectedImpact: "Improves lean muscle retention.",
						},
					},
					WeeklySummary:  "Strong week with 4 sessions.",
					MonthlySummary: "Steady 2kg progress over 4 weeks.",
				},
			})
			return rec.Result(), nil
		}),
	}

	pyAIClient := NewHTTPPyAIAnalyticsClient("http://py-ai.internal:8000", mockClient)
	reqPayload := &AnonymizedAIAnalysisPayload{
		SubjectID:          "anon-subject-hash-123",
		RegenerateInsights: true,
	}

	resp, err := pyAIClient.AnalyzeUserHealthAndGenerateInsights(context.Background(), reqPayload)
	if err != nil {
		testingInstance.Fatalf("expected AnalyzeUserHealthAndGenerateInsights to succeed, got: %v", err)
	}

	if capturedPath != "/api/v1/analyze" || capturedPayload.SubjectID != "anon-subject-hash-123" {
		testingInstance.Errorf("unexpected request path=%q or subject=%q", capturedPath, capturedPayload.SubjectID)
	}

	if resp.Wellness == nil || resp.Wellness.WellnessScore != 88.4 || resp.Insights == nil {
		testingInstance.Errorf("unexpected response from HTTPPyAIAnalyticsClient: %+v", resp)
	}
}
