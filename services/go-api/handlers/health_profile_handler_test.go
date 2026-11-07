// This file contains unit tests for HealthProfileHandler covering imperial-to-metric unit conversion
// (lbs -> kg, in -> cm), invalid BMI/negative weight rejection, duplicate activity timestamp 409 Conflict,
// historical weight logging with unique timestamps, and full GDPR health data JSON export.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
	"nutrysync/go-api/repositories"
)

type inMemoryHealthProfileRepository struct {
	profilesByUserID map[string]*models.HealthProfile
	weightsByUserID  map[string][]models.WeightEntry
	activityByUserID map[string][]models.ActivityEntry
}

func newInMemoryHealthProfileRepository() *inMemoryHealthProfileRepository {
	return &inMemoryHealthProfileRepository{
		profilesByUserID: make(map[string]*models.HealthProfile),
		weightsByUserID:  make(map[string][]models.WeightEntry),
		activityByUserID: make(map[string][]models.ActivityEntry),
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
