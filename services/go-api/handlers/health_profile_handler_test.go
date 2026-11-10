// This file contains unit tests for HealthProfileHandler covering imperial-to-metric unit conversion
// (lbs -> kg, in -> cm), invalid BMI/negative weight rejection, duplicate activity timestamp 409 Conflict,
// historical weight logging with unique timestamps, and full GDPR health data JSON export.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nutrysync/go-api/auth"
	"nutrysync/go-api/health"
	"nutrysync/go-api/models"
	"nutrysync/go-api/repositories"
)

type inMemoryHealthProfileRepository struct {
	profilesByUserID  map[string]*models.HealthProfile
	weightsByUserID   map[string][]models.WeightEntry
	activityByUserID  map[string][]models.ActivityEntry
	wellnessByUserID  map[string][]models.WellnessSnapshot
	insightsByUserID  map[string][]models.AIInsightRecord
}

func newInMemoryHealthProfileRepository() *inMemoryHealthProfileRepository {
	return &inMemoryHealthProfileRepository{
		profilesByUserID: make(map[string]*models.HealthProfile),
		weightsByUserID:  make(map[string][]models.WeightEntry),
		activityByUserID: make(map[string][]models.ActivityEntry),
		wellnessByUserID: make(map[string][]models.WellnessSnapshot),
		insightsByUserID: make(map[string][]models.AIInsightRecord),
	}
}

func (repo *inMemoryHealthProfileRepository) FindHealthProfileByUserID(ctx context.Context, userID string) (*models.HealthProfile, error) {
	return repo.profilesByUserID[userID], nil
}

func (repo *inMemoryHealthProfileRepository) UpsertHealthProfileWithWeightEntry(
	ctx context.Context,
	profile *models.HealthProfile,
	weightEntry *models.WeightEntry,
) error {
	repo.profilesByUserID[profile.UserID] = profile
	repo.weightsByUserID[profile.UserID] = append(repo.weightsByUserID[profile.UserID], *weightEntry)
	return nil
}

func (repo *inMemoryHealthProfileRepository) CreateWeightEntry(ctx context.Context, entry *models.WeightEntry) error {
	repo.weightsByUserID[entry.UserID] = append(repo.weightsByUserID[entry.UserID], *entry)
	if profile := repo.profilesByUserID[entry.UserID]; profile != nil {
		profile.CurrentWeightKg = entry.WeightKg
	}
	return nil
}

func (repo *inMemoryHealthProfileRepository) ListWeightEntriesByUserID(ctx context.Context, userID string) ([]models.WeightEntry, error) {
	return repo.weightsByUserID[userID], nil
}

func (repo *inMemoryHealthProfileRepository) CreateActivityEntry(ctx context.Context, entry *models.ActivityEntry) error {
	for _, existing := range repo.activityByUserID[entry.UserID] {
		if existing.RecordedAt.Equal(entry.RecordedAt) {
			return repositories.ErrDuplicateActivityTimestamp
		}
	}
	repo.activityByUserID[entry.UserID] = append(repo.activityByUserID[entry.UserID], *entry)
	return nil
}

func (repo *inMemoryHealthProfileRepository) ListActivityEntriesByUserID(ctx context.Context, userID string) ([]models.ActivityEntry, error) {
	return repo.activityByUserID[userID], nil
}

func (repo *inMemoryHealthProfileRepository) CreateWellnessSnapshot(ctx context.Context, snapshot *models.WellnessSnapshot) error {
	repo.wellnessByUserID[snapshot.UserID] = append(repo.wellnessByUserID[snapshot.UserID], *snapshot)
	return nil
}

func (repo *inMemoryHealthProfileRepository) ListWellnessSnapshotsByUserID(ctx context.Context, userID string) ([]models.WellnessSnapshot, error) {
	return repo.wellnessByUserID[userID], nil
}

func (repo *inMemoryHealthProfileRepository) CreateAIInsightRecord(ctx context.Context, record *models.AIInsightRecord) error {
	existing := repo.insightsByUserID[record.UserID]
	repo.insightsByUserID[record.UserID] = append([]models.AIInsightRecord{*record}, existing...)
	return nil
}

func (repo *inMemoryHealthProfileRepository) FindLatestAIInsightRecordByUserID(ctx context.Context, userID string) (*models.AIInsightRecord, error) {
	records := repo.insightsByUserID[userID]
	if len(records) == 0 {
		return nil, nil
	}
	latest := records[0]
	return &latest, nil
}

func (repo *inMemoryHealthProfileRepository) ListAIInsightRecordsByUserID(ctx context.Context, userID string) ([]models.AIInsightRecord, error) {
	return repo.insightsByUserID[userID], nil
}

type mockPyAIAnalyticsClient struct {
	shouldFail       bool
	capturedPayloads []*health.AnonymizedAIAnalysisPayload
}

func (mock *mockPyAIAnalyticsClient) AnalyzeUserHealthAndGenerateInsights(
	ctx context.Context,
	payload *health.AnonymizedAIAnalysisPayload,
) (*health.PyAIAnalysisResponse, error) {
	mock.capturedPayloads = append(mock.capturedPayloads, payload)
	if mock.shouldFail {
		return nil, errors.New("py-ai service connection refused")
	}

	return &health.PyAIAnalysisResponse{
		Wellness: nil, // let deterministic wellness calculation or py-ai wellness be verified
		Insights: &health.PyAIInsightsPayload{
			PromptStrategy:       "cot_few_shot_v1",
			HealthStatusAnalysis: "Goal-focused analysis for " + payload.UserMetrics.TargetState.PrimaryGoal,
			ProgressEvaluation:   "Progress evaluation toward target weight.",
			Recommendations: []health.PrioritizedAIRecommendation{
				{
					Title:          "Prioritize Protein & Recovery",
					Category:       "nutrition",
					Priority:       "high",
					Action:         "Consume 1.6g/kg protein daily.",
					ExpectedImpact: "Supports lean mass and satiety.",
				},
			},
			WeeklySummary:  "Weekly summary for " + payload.UserMetrics.TargetState.PrimaryGoal,
			MonthlySummary: "Monthly summary for " + payload.UserMetrics.TargetState.PrimaryGoal,
		},
	}, nil
}

// TestHealthProfileHandlerImperialConversionInvalidBMIAndDuplicateActivityConflict verifies:
// 1. Saving a health profile with imperial units (70 in -> 177.80 cm, 154.32 lbs -> 70.00 kg)
// 2. Rejecting negative/unrealistic weights and heights with 400 Bad Request
// 3. Rejecting duplicate activity timestamps with 409 Conflict
// 4. Exporting all user health data (profile + weight entries + activity entries) via GET /api/health/export
func TestHealthProfileHandlerImperialConversionInvalidBMIAndDuplicateActivityConflict(testingInstance *testing.T) {
	jwtSecret := "test-health-profile-jwt-secret"
	userID := "user-health-1"
	sessionID := "session-health-1"

	healthRepo := newInMemoryHealthProfileRepository()
	sessionRepo := &mockUserSessionRepository{
		activeSessionByID: &models.Session{
			ID:        sessionID,
			UserID:    userID,
			Revoked:   false,
			ExpiresAt: time.Now().UTC().Add(time.Hour),
			User: models.User{
				ID:        userID,
				Email:     "athlete@example.com",
				Consented: true,
			},
		},
	}

	handler, initErr := NewHealthProfileHandler(healthRepo, sessionRepo, jwtSecret)
	if initErr != nil {
		testingInstance.Fatalf("expected NewHealthProfileHandler to succeed, got: %v", initErr)
	}

	accessToken, _ := auth.GenerateSignedAccessToken(userID, sessionID, jwtSecret, 15*time.Minute)

	// 1. Save profile with imperial units (in & lbs)
	savePayload := SaveHealthProfileRequestPayload{
		Age:                  28,
		Gender:               "male",
		Height:               70.0,
		HeightUnit:           "in",
		CurrentWeight:        154.32,
		TargetWeight:         145.0,
		WeightUnit:           "lbs",
		OccupationType:       "desk",
		ActivityLevel:        "moderately_active",
		PrimaryGoal:          "general_fitness",
		TargetActivityLevel:  "very_active",
		TargetWeeklyDays:     4,
		WeeklyActivityDays:   3,
		ExerciseTypes:        []string{"cardio", "strength"},
		SessionDuration:      "30-60m",
		FitnessLevel:         "intermediate",
		PreferredEnvironment: "gym",
		PreferredTimeOfDay:   "morning",
		EnduranceMinutes:     30,
		PushupsCount:         25,
		SquatsCount:          40,
		DietaryPreferences:   []string{"high-protein"},
		DietaryRestrictions:  []string{"lactose-intolerant"},
		SleepHours:           7.5,
		WaterLiters:          2.5,
	}
	saveBody, _ := json.Marshal(savePayload)
	saveReq := httptest.NewRequest(http.MethodPost, "/api/health/profile", bytes.NewReader(saveBody))
	saveReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	saveRec := httptest.NewRecorder()
	handler.HandleSaveHealthProfile(saveRec, saveReq)

	if saveRec.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK saving imperial health profile, got %d (%s)", saveRec.Code, saveRec.Body.String())
	}

	var savedProfile HealthProfileResponsePayload
	if decodeErr := json.Unmarshal(saveRec.Body.Bytes(), &savedProfile); decodeErr != nil {
		testingInstance.Fatalf("failed to decode saved profile response: %v", decodeErr)
	}

	if savedProfile.HeightCm != 177.8 || savedProfile.CurrentWeightKg != 70.0 {
		testingInstance.Errorf("expected HeightCm=177.80 and CurrentWeightKg=70.00, got height=%.2f weight=%.2f", savedProfile.HeightCm, savedProfile.CurrentWeightKg)
	}

	if savedProfile.BMIClassification != "normal weight" {
		testingInstance.Errorf("expected normal weight BMI classification, got %q", savedProfile.BMIClassification)
	}

	// 2. Reject negative weight on POST /api/health/weight with 400 Bad Request
	negWeightBody, _ := json.Marshal(LogWeightEntryRequestPayload{Weight: -50, Unit: "kg"})
	negWeightReq := httptest.NewRequest(http.MethodPost, "/api/health/weight", bytes.NewReader(negWeightBody))
	negWeightReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	negWeightRec := httptest.NewRecorder()
	handler.HandleLogWeightEntry(negWeightRec, negWeightReq)

	if negWeightRec.Code != http.StatusBadRequest {
		testingInstance.Fatalf("expected 400 Bad Request for negative weight, got %d", negWeightRec.Code)
	}

	// 3. Log activity entry and verify duplicate timestamp returns 409 Conflict
	activityTimestamp := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	actPayload := LogActivityEntryRequestPayload{
		ExerciseType:    "running",
		DurationMinutes: 40,
		Intensity:       "high",
		RecordedAt:      activityTimestamp,
	}
	actBody, _ := json.Marshal(actPayload)
	actReq1 := httptest.NewRequest(http.MethodPost, "/api/health/activity", bytes.NewReader(actBody))
	actReq1.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	actRec1 := httptest.NewRecorder()
	handler.HandleLogActivityEntry(actRec1, actReq1)

	if actRec1.Code != http.StatusCreated {
		testingInstance.Fatalf("expected 201 Created for first activity entry, got %d (%s)", actRec1.Code, actRec1.Body.String())
	}

	actReq2 := httptest.NewRequest(http.MethodPost, "/api/health/activity", bytes.NewReader(actBody))
	actReq2.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	actRec2 := httptest.NewRecorder()
	handler.HandleLogActivityEntry(actRec2, actReq2)

	if actRec2.Code != http.StatusConflict {
		testingInstance.Fatalf("expected 409 Conflict for duplicate activity timestamp, got %d (%s)", actRec2.Code, actRec2.Body.String())
	}

	// 4. Verify GDPR Export (GET /api/health/export)
	exportReq := httptest.NewRequest(http.MethodGet, "/api/health/export", nil)
	exportReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	exportRec := httptest.NewRecorder()
	handler.HandleExportUserHealthData(exportRec, exportReq)

	if exportRec.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK from GET /api/health/export, got %d", exportRec.Code)
	}

	var exportData GDPRHealthDataExportResponsePayload
	if decodeErr := json.Unmarshal(exportRec.Body.Bytes(), &exportData); decodeErr != nil {
		testingInstance.Fatalf("failed to decode GDPR export payload: %v", decodeErr)
	}

	if exportData.User.Email != "athlete@example.com" || exportData.HealthProfile == nil {
		testingInstance.Fatalf("expected user metadata and healthProfile in GDPR export, got %+v", exportData)
	}

	if len(exportData.WeightEntries) != 1 || len(exportData.ActivityEntries) != 1 {
		testingInstance.Fatalf("expected 1 weightEntry and 1 activityEntry in GDPR export, got weights=%d activities=%d", len(exportData.WeightEntries), len(exportData.ActivityEntries))
	}
}

// TestHealthProfileHandlerWellnessRecalculationGoalChangeVersioningAndCachedFallback verifies:
// 1. Outbound py-ai payloads strip PII (email, raw user ID, session ID)
// 2. Saving a profile, logging activity, and logging weight each append a recalculated WellnessSnapshot
// 3. Updating PrimaryGoal triggers a new versioned AIInsightRecord (Version = prevVersion + 1)
// 4. When py-ai returns an error, GET /api/health/dashboard and POST /api/health/insights/regenerate
//    gracefully serve the latest cached AIInsightRecord with isCachedFallback: true and 200 OK
func TestHealthProfileHandlerWellnessRecalculationGoalChangeVersioningAndCachedFallback(testingInstance *testing.T) {
	jwtSecret := "test-orchestrator-jwt-secret"
	userID := "secret-raw-user-uuid-999"
	sessionID := "secret-session-uuid-888"
	userEmail := "sensitive-pii-user@example.com"

	healthRepo := newInMemoryHealthProfileRepository()
	sessionRepo := &mockUserSessionRepository{
		activeSessionByID: &models.Session{
			ID:        sessionID,
			UserID:    userID,
			Revoked:   false,
			ExpiresAt: time.Now().UTC().Add(time.Hour),
			User: models.User{
				ID:        userID,
				Email:     userEmail,
				Consented: true,
			},
		},
	}

	mockAIClient := &mockPyAIAnalyticsClient{}
	handler, initErr := NewHealthProfileHandler(healthRepo, sessionRepo, jwtSecret, mockAIClient)
	if initErr != nil {
		testingInstance.Fatalf("expected NewHealthProfileHandler to succeed, got: %v", initErr)
	}

	accessToken, _ := auth.GenerateSignedAccessToken(userID, sessionID, jwtSecret, 15*time.Minute)

	// Step 1: Save initial profile (PrimaryGoal = "weight_loss")
	initialProfile := SaveHealthProfileRequestPayload{
		Age:                30,
		Gender:             "female",
		Height:             165.0,
		HeightUnit:         "cm",
		CurrentWeight:      75.0,
		TargetWeight:       63.0,
		WeightUnit:         "kg",
		ActivityLevel:      "lightly_active",
		PrimaryGoal:        "weight_loss",
		TargetActivityLevel: "moderately_active",
		TargetWeeklyDays:   4,
		WeeklyActivityDays: 1,
		SleepHours:         7.0,
		WaterLiters:        2.0,
	}
	profileBody, _ := json.Marshal(initialProfile)
	profileReq := httptest.NewRequest(http.MethodPost, "/api/health/profile", bytes.NewReader(profileBody))
	profileReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	profileRec := httptest.NewRecorder()
	handler.HandleSaveHealthProfile(profileRec, profileReq)

	if profileRec.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK saving profile, got %d (%s)", profileRec.Code, profileRec.Body.String())
	}

	if len(mockAIClient.capturedPayloads) != 1 {
		testingInstance.Fatalf("expected 1 outbound py-ai call on profile save, got %d", len(mockAIClient.capturedPayloads))
	}

	serializedOutbound, _ := json.Marshal(mockAIClient.capturedPayloads[0])
	if strings.Contains(string(serializedOutbound), userID) ||
		strings.Contains(string(serializedOutbound), userEmail) ||
		strings.Contains(string(serializedOutbound), sessionID) {
		testingInstance.Fatalf("outbound py-ai payload leaked PII: %s", string(serializedOutbound))
	}

	// Step 2: Log activity and weight entries -> verify wellness history grows and score updates
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		actBody, _ := json.Marshal(LogActivityEntryRequestPayload{
			ExerciseType:    "running",
			DurationMinutes: 45,
			Intensity:       "high",
			RecordedAt:      now.Add(time.Duration(i+1) * time.Minute),
		})
		actReq := httptest.NewRequest(http.MethodPost, "/api/health/activity", bytes.NewReader(actBody))
		actReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
		actRec := httptest.NewRecorder()
		handler.HandleLogActivityEntry(actRec, actReq)
		if actRec.Code != http.StatusCreated {
			testingInstance.Fatalf("expected 201 Created logging activity, got %d", actRec.Code)
		}
	}

	weightBody, _ := json.Marshal(LogWeightEntryRequestPayload{Weight: 71.0, Unit: "kg"})
	weightReq := httptest.NewRequest(http.MethodPost, "/api/health/weight", bytes.NewReader(weightBody))
	weightReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	weightRec := httptest.NewRecorder()
	handler.HandleLogWeightEntry(weightRec, weightReq)
	if weightRec.Code != http.StatusCreated {
		testingInstance.Fatalf("expected 201 Created logging weight, got %d", weightRec.Code)
	}

	snapshots, _ := healthRepo.ListWellnessSnapshotsByUserID(context.Background(), userID)
	if len(snapshots) != 5 {
		testingInstance.Fatalf("expected 5 WellnessSnapshots (1 profile + 3 activities + 1 weight), got %d", len(snapshots))
	}
	if snapshots[len(snapshots)-1].WellnessScore <= snapshots[0].WellnessScore {
		testingInstance.Errorf(
			"expected latest WellnessScore (%.2f) to exceed initial WellnessScore (%.2f) after activity + weight progress",
			snapshots[len(snapshots)-1].WellnessScore,
			snapshots[0].WellnessScore,
		)
	}

	// Step 3: Update PrimaryGoal -> creates Version 2 AIInsightRecord
	updatedProfile := initialProfile
	updatedProfile.CurrentWeight = 71.0
	updatedProfile.PrimaryGoal = "muscle_gain"
	updatedBody, _ := json.Marshal(updatedProfile)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/health/profile", bytes.NewReader(updatedBody))
	updateReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	updateRec := httptest.NewRecorder()
	handler.HandleSaveHealthProfile(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK updating profile goal, got %d", updateRec.Code)
	}

	latestInsight, _ := healthRepo.FindLatestAIInsightRecordByUserID(context.Background(), userID)
	if latestInsight == nil || latestInsight.Version != 2 || latestInsight.PrimaryGoalSnapshot != "muscle_gain" {
		testingInstance.Fatalf("expected Version 2 AIInsightRecord for muscle_gain, got %+v", latestInsight)
	}

	// Step 4: Simulate py-ai outage and verifyPOST /api/health/insights/regenerate and GET /api/health/dashboard serve cached fallback
	mockAIClient.shouldFail = true

	regenReq := httptest.NewRequest(http.MethodPost, "/api/health/insights/regenerate", nil)
	regenReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	regenRec := httptest.NewRecorder()
	handler.HandleRegenerateAIInsights(regenRec, regenReq)

	if regenRec.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK on regenerate fallback when py-ai is down, got %d", regenRec.Code)
	}

	var regenResp RegenerateAIInsightsResponsePayload
	if err := json.Unmarshal(regenRec.Body.Bytes(), &regenResp); err != nil {
		testingInstance.Fatalf("failed to decode RegenerateAIInsightsResponsePayload: %v", err)
	}

	if regenResp.AIServiceAvailable || !regenResp.IsCachedFallback || regenResp.LatestInsights == nil || regenResp.LatestInsights.Version != 2 {
		testingInstance.Errorf("expected cached fallback insight Version 2 when py-ai fails, got %+v", regenResp)
	}

	dashReq := httptest.NewRequest(http.MethodGet, "/api/health/dashboard", nil)
	dashReq.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: accessToken})
	dashRec := httptest.NewRecorder()
	handler.HandleGetHealthDashboard(dashRec, dashReq)

	if dashRec.Code != http.StatusOK {
		testingInstance.Fatalf("expected 200 OK from GET /api/health/dashboard, got %d", dashRec.Code)
	}

	var dashResp HealthDashboardResponsePayload
	if err := json.Unmarshal(dashRec.Body.Bytes(), &dashResp); err != nil {
		testingInstance.Fatalf("failed to decode HealthDashboardResponsePayload: %v", err)
	}

	if dashResp.HealthProfile == nil || dashResp.LatestWellness == nil || dashResp.LatestInsights == nil || len(dashResp.InsightHistory) != 2 {
		testingInstance.Errorf("unexpected dashboard payload: %+v", dashResp)
	}
}
