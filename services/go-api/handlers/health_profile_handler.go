// This file implements HTTP handlers for User Health Profile management (GET/POST/PUT /api/health/profile),
// historical weight tracking (GET/POST /api/health/weight), activity logging (GET/POST /api/health/activity),
// GDPR full health data JSON export (GET /api/health/export), health dashboard orchestration (GET /api/health/dashboard),
// and versioned AI insight regeneration with cached fallback (POST /api/health/insights/regenerate).
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"nutrysync/go-api/auth"
	"nutrysync/go-api/health"
	"nutrysync/go-api/models"
	"nutrysync/go-api/repositories"
)

// SaveHealthProfileRequestPayload represents the JSON body accepted by POST /api/health/profile and PUT /api/health/profile.
// Accepts height, currentWeight, and targetWeight in the user's preferred units ("cm"/"in" and "kg"/"lbs").
type SaveHealthProfileRequestPayload struct {
	Age                  int      `json:"age"`
	Gender               string   `json:"gender"`
	Height               float64  `json:"height"`
	HeightUnit           string   `json:"heightUnit"`
	CurrentWeight        float64  `json:"currentWeight"`
	TargetWeight         float64  `json:"targetWeight"`
	WeightUnit           string   `json:"weightUnit"`
	OccupationType       string   `json:"occupationType"`
	ActivityLevel        string   `json:"activityLevel"`
	PrimaryGoal          string   `json:"primaryGoal"`
	TargetActivityLevel  string   `json:"targetActivityLevel"`
	TargetWeeklyDays     int      `json:"targetWeeklyDays"`
	WeeklyActivityDays   int      `json:"weeklyActivityDays"`
	ExerciseTypes        []string `json:"exerciseTypes"`
	SessionDuration      string   `json:"sessionDuration"`
	FitnessLevel         string   `json:"fitnessLevel"`
	PreferredEnvironment string   `json:"preferredEnvironment"`
	PreferredTimeOfDay   string   `json:"preferredTimeOfDay"`
	EnduranceMinutes     int      `json:"enduranceMinutes"`
	PushupsCount         int      `json:"pushupsCount"`
	SquatsCount          int      `json:"squatsCount"`
	DietaryPreferences   []string `json:"dietaryPreferences"`
	DietaryRestrictions  []string `json:"dietaryRestrictions"`
	MedicalConditions    string   `json:"medicalConditions"`
	SleepHours           float64  `json:"sleepHours"`
	WaterLiters          float64  `json:"waterLiters"`
	AllowAIProcessing    *bool    `json:"allowAiProcessing"`
	PublicProfileVisible *bool    `json:"publicProfileVisible"`
	EmailNotifications   *bool    `json:"emailNotifications"`
}

// HealthProfileResponsePayload represents the enriched JSON representation of a user's HealthProfile.
type HealthProfileResponsePayload struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"userId"`
	Age                  int       `json:"age"`
	Gender               string    `json:"gender"`
	HeightCm             float64   `json:"heightCm"`
	CurrentWeightKg      float64   `json:"currentWeightKg"`
	BaselineWeightKg     float64   `json:"baselineWeightKg"`
	TargetWeightKg       float64   `json:"targetWeightKg"`
	PreferredWeightUnit  string    `json:"preferredWeightUnit"`
	PreferredHeightUnit  string    `json:"preferredHeightUnit"`
	BMI                  float64   `json:"bmi"`
	BMIClassification    string    `json:"bmiClassification"`
	OccupationType       string    `json:"occupationType"`
	ActivityLevel        string    `json:"activityLevel"`
	PrimaryGoal          string    `json:"primaryGoal"`
	TargetActivityLevel  string    `json:"targetActivityLevel"`
	TargetWeeklyDays     int       `json:"targetWeeklyDays"`
	WeeklyActivityDays   int       `json:"weeklyActivityDays"`
	ExerciseTypes        []string  `json:"exerciseTypes"`
	SessionDuration      string    `json:"sessionDuration"`
	FitnessLevel         string    `json:"fitnessLevel"`
	PreferredEnvironment string    `json:"preferredEnvironment"`
	PreferredTimeOfDay   string    `json:"preferredTimeOfDay"`
	EnduranceMinutes     int       `json:"enduranceMinutes"`
	PushupsCount         int       `json:"pushupsCount"`
	SquatsCount          int       `json:"squatsCount"`
	DietaryPreferences   []string  `json:"dietaryPreferences"`
	DietaryRestrictions  []string  `json:"dietaryRestrictions"`
	MedicalConditions    string    `json:"medicalConditions"`
	SleepHours           float64   `json:"sleepHours"`
	WaterLiters          float64   `json:"waterLiters"`
	AllowAIProcessing    bool      `json:"allowAiProcessing"`
	PublicProfileVisible bool      `json:"publicProfileVisible"`
	EmailNotifications   bool      `json:"emailNotifications"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// LogWeightEntryRequestPayload represents the JSON body accepted by POST /api/health/weight.
type LogWeightEntryRequestPayload struct {
	Weight     float64    `json:"weight"`
	Unit       string     `json:"unit"`
	RecordedAt *time.Time `json:"recordedAt,omitempty"`
}

// LogActivityEntryRequestPayload represents the JSON body accepted by POST /api/health/activity.
type LogActivityEntryRequestPayload struct {
	ExerciseType    string    `json:"exerciseType"`
	DurationMinutes int       `json:"durationMinutes"`
	Intensity       string    `json:"intensity"`
	Notes           string    `json:"notes,omitempty"`
	RecordedAt      time.Time `json:"recordedAt"`
}

// GDPRUserMetadataExportPayload represents the user metadata section in GET /api/health/export.
type GDPRUserMetadataExportPayload struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Consented bool      `json:"consented"`
	ExportedAt time.Time `json:"exportedAt"`
}

// GDPRHealthDataExportResponsePayload represents the full GDPR JSON export returned by GET /api/health/export.
type GDPRHealthDataExportResponsePayload struct {
	User            GDPRUserMetadataExportPayload `json:"user"`
	HealthProfile   *HealthProfileResponsePayload `json:"healthProfile"`
	WeightEntries   []models.WeightEntry          `json:"weightEntries"`
	ActivityEntries []models.ActivityEntry        `json:"activityEntries"`
}

// HealthDashboardResponsePayload represents the aggregated analytics and AI insights payload returned by GET /api/health/dashboard.
type HealthDashboardResponsePayload struct {
	HealthProfile      *HealthProfileResponsePayload `json:"healthProfile"`
	LatestWellness     *models.WellnessSnapshot      `json:"latestWellness"`
	WellnessHistory    []models.WellnessSnapshot     `json:"wellnessHistory"`
	WeightHistory      []models.WeightEntry          `json:"weightHistory"`
	ActivityHistory    []models.ActivityEntry        `json:"activityHistory"`
	LatestInsights     *models.AIInsightRecord       `json:"latestInsights"`
	InsightHistory     []models.AIInsightRecord      `json:"insightHistory"`
	AIServiceAvailable bool                          `json:"aiServiceAvailable"`
	IsCachedFallback   bool                          `json:"isCachedFallback"`
}

// RegenerateAIInsightsResponsePayload represents the JSON response returned by POST /api/health/insights/regenerate.
type RegenerateAIInsightsResponsePayload struct {
	LatestInsights     *models.AIInsightRecord  `json:"latestInsights"`
	InsightHistory     []models.AIInsightRecord `json:"insightHistory"`
	AIServiceAvailable bool                     `json:"aiServiceAvailable"`
	IsCachedFallback   bool                     `json:"isCachedFallback"`
}

// HealthProfileHandler handles HTTP requests for user health profiles, weight logs, activity logs,
// GDPR exports, wellness score snapshots, and versioned AI insights orchestration.
type HealthProfileHandler struct {
	healthProfileRepository repositories.HealthProfileRepository
	sessionRepository       repositories.UserSessionRepository
	pyAIClient              health.PyAIAnalyticsClient
	jwtSecret               string
	weightTimestampMutex    sync.Mutex
	lastWeightTimestampUTC  time.Time
}

// NewHealthProfileHandler constructs a HealthProfileHandler after validating its dependencies.
// Accepts an optional health.PyAIAnalyticsClient (defaulting to health.NewHTTPPyAIAnalyticsClient when omitted or nil).
func NewHealthProfileHandler(
	healthProfileRepository repositories.HealthProfileRepository,
	sessionRepository repositories.UserSessionRepository,
	jwtSecret string,
	optionalPyAIClient ...health.PyAIAnalyticsClient,
) (*HealthProfileHandler, error) {
	if healthProfileRepository == nil {
		return nil, errors.New("health profile repository must not be nil")
	}

	if sessionRepository == nil {
		return nil, errors.New("session repository must not be nil")
	}

	trimmedJWTSecret := strings.TrimSpace(jwtSecret)
	if trimmedJWTSecret == "" {
		return nil, errors.New("JWT secret must not be empty")
	}

	var configuredPyAIClient health.PyAIAnalyticsClient
	if len(optionalPyAIClient) > 0 && optionalPyAIClient[0] != nil {
		configuredPyAIClient = optionalPyAIClient[0]
	} else {
		configuredPyAIClient = health.NewHTTPPyAIAnalyticsClient("", nil)
	}

	return &HealthProfileHandler{
		healthProfileRepository: healthProfileRepository,
		sessionRepository:       sessionRepository,
		pyAIClient:              configuredPyAIClient,
		jwtSecret:               trimmedJWTSecret,
	}, nil
}

// authenticateSessionFromRequest validates the access_token cookie and returns the active models.Session.
func (handler *HealthProfileHandler) authenticateSessionFromRequest(request *http.Request) (*models.Session, error) {
	accessCookie, cookieError := request.Cookie(auth.AccessTokenCookieName)
	if cookieError != nil || accessCookie == nil {
		return nil, errors.New("missing access_token cookie")
	}

	claims, verifyError := auth.VerifyAndParseAccessToken(accessCookie.Value, handler.jwtSecret)
	if verifyError != nil || claims == nil {
		return nil, errors.New("invalid access_token cookie")
	}

	sessionRecord, lookupError := handler.sessionRepository.FindSessionByID(request.Context(), claims.SessionID)
	currentTimeUTC := time.Now().UTC()
	if lookupError != nil ||
		sessionRecord == nil ||
		sessionRecord.UserID != claims.Subject ||
		sessionRecord.Revoked ||
		!currentTimeUTC.Before(sessionRecord.ExpiresAt) {
		return nil, errors.New("expired or revoked session")
	}

	return sessionRecord, nil
}

// nextUniqueWeightTimestampUTC returns a monotonically increasing UTC timestamp at millisecond precision
// so every weight entry logged for a user has a unique RecordedAt timestamp.
func (handler *HealthProfileHandler) nextUniqueWeightTimestampUTC(requested *time.Time) time.Time {
	handler.weightTimestampMutex.Lock()
	defer handler.weightTimestampMutex.Unlock()

	candidate := time.Now().UTC().Truncate(time.Millisecond)
	if requested != nil && !requested.IsZero() {
		candidate = requested.UTC().Truncate(time.Millisecond)
	}

	if !candidate.After(handler.lastWeightTimestampUTC) {
		candidate = handler.lastWeightTimestampUTC.Add(time.Millisecond)
	}

	handler.lastWeightTimestampUTC = candidate
	return candidate
}

// marshalStringSliceJSON encodes a string slice as a JSON array string (defaulting nil slices to "[]").
func marshalStringSliceJSON(items []string) string {
	if items == nil {
		items = []string{}
	}
	encodedBytes, encodeErr := json.Marshal(items)
	if encodeErr != nil {
		return "[]"
	}
	return string(encodedBytes)
}

// unmarshalStringSliceJSON decodes a JSON array string into a []string (returning an empty slice on error).
func unmarshalStringSliceJSON(rawJSON string) []string {
	var parsed []string
	if strings.TrimSpace(rawJSON) == "" {
		return []string{}
	}
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil || parsed == nil {
		return []string{}
	}
	return parsed
}

// buildWellnessSnapshotFromCalculation converts a health.WellnessCalculationResult into a models.WellnessSnapshot.
func (handler *HealthProfileHandler) buildWellnessSnapshotFromCalculation(
	userID string,
	result *health.WellnessCalculationResult,
) *models.WellnessSnapshot {
	snapshotID, _ := generateRandomUUIDv4()
	milestones := result.Milestones
	if milestones == nil {
		milestones = []health.WellnessMilestone{}
	}
	encodedMilestones, encodeErr := json.Marshal(milestones)
	if encodeErr != nil {
		encodedMilestones = []byte("[]")
	}

	return &models.WellnessSnapshot{
		ID:                snapshotID,
		UserID:            userID,
		BMI:               result.BMI,
		BMIClassification: result.BMIClassification,
		WellnessScore:     result.WellnessScore,
		BMIScore:          result.BMIScore,
		ActivityScore:     result.ActivityScore,
		ProgressScore:     result.ProgressScore,
		HabitsScore:       result.HabitsScore,
		MilestonesJSON:    string(encodedMilestones),
		RecordedAt:        handler.nextUniqueWeightTimestampUTC(nil),
	}
}

// buildAIInsightRecordFromPyAI constructs a new versioned models.AIInsightRecord from a health.PyAIInsightsPayload.
func buildAIInsightRecordFromPyAI(
	userID string,
	version int,
	primaryGoal string,
	insights *health.PyAIInsightsPayload,
) *models.AIInsightRecord {
	recordID, _ := generateRandomUUIDv4()
	recommendations := insights.Recommendations
	if recommendations == nil {
		recommendations = []health.PrioritizedAIRecommendation{}
	}
	encodedRecommendations, encodeErr := json.Marshal(recommendations)
	if encodeErr != nil {
		encodedRecommendations = []byte("[]")
	}

	promptStrategy := strings.TrimSpace(insights.PromptStrategy)
	if promptStrategy == "" {
		promptStrategy = "cot_few_shot_v1"
	}

	return &models.AIInsightRecord{
		ID:                   recordID,
		UserID:               userID,
		Version:              version,
		PrimaryGoalSnapshot:  strings.TrimSpace(primaryGoal),
		PromptStrategy:       promptStrategy,
		HealthStatusAnalysis: strings.TrimSpace(insights.HealthStatusAnalysis),
		ProgressEvaluation:   strings.TrimSpace(insights.ProgressEvaluation),
		RecommendationsJSON:  string(encodedRecommendations),
		WeeklySummary:        strings.TrimSpace(insights.WeeklySummary),
		MonthlySummary:       strings.TrimSpace(insights.MonthlySummary),
		CreatedAt:            time.Now().UTC(),
	}
}

// recalculateWellnessAndPersistOptionalAIInsights recalculates the user's Wellness Score snapshot
// (always persisting a new models.WellnessSnapshot on metric changes, falling back to deterministic calculation
// if py-ai is unavailable or AI processing is disabled) and saves a new versioned models.AIInsightRecord
// whenever AllowAIProcessing is true, regeneration is needed, and py-ai returns insights.
func (handler *HealthProfileHandler) recalculateWellnessAndPersistOptionalAIInsights(
	ctx context.Context,
	userID string,
	profileWasUpdated bool,
	forceRegenerateInsights bool,
) (*models.WellnessSnapshot, *models.AIInsightRecord, bool) {
	profileRecord, profileErr := handler.healthProfileRepository.FindHealthProfileByUserID(ctx, userID)
	if profileErr != nil || profileRecord == nil {
		return nil, nil, false
	}

	weights, _ := handler.healthProfileRepository.ListWeightEntriesByUserID(ctx, userID)
	activities, _ := handler.healthProfileRepository.ListActivityEntriesByUserID(ctx, userID)
	latestInsight, _ := handler.healthProfileRepository.FindLatestAIInsightRecordByUserID(ctx, userID)

	shouldRequestInsights := profileRecord.AllowAIProcessing && (forceRegenerateInsights ||
		profileWasUpdated ||
		latestInsight == nil ||
		!strings.EqualFold(strings.TrimSpace(latestInsight.PrimaryGoalSnapshot), strings.TrimSpace(profileRecord.PrimaryGoal)))

	var wellnessResult *health.WellnessCalculationResult
	var pyAIInsights *health.PyAIInsightsPayload
	aiServiceAvailable := false

	if profileRecord.AllowAIProcessing && handler.pyAIClient != nil {
		anonPayload, payloadErr := health.BuildAnonymizedAIAnalysisPayload(
			profileRecord,
			weights,
			activities,
			shouldRequestInsights,
		)
		if payloadErr == nil {
			pyAIResponse, callErr := handler.pyAIClient.AnalyzeUserHealthAndGenerateInsights(ctx, anonPayload)
			if callErr == nil && pyAIResponse != nil {
				aiServiceAvailable = true
				wellnessResult = pyAIResponse.Wellness
				pyAIInsights = pyAIResponse.Insights
			}
		}
	}

	if wellnessResult == nil {
		deterministicResult, calcErr := health.CalculateDeterministicWellnessAndMilestones(profileRecord, weights, activities)
		if calcErr == nil {
			wellnessResult = deterministicResult
		}
	}

	var savedSnapshot *models.WellnessSnapshot
	if wellnessResult != nil {
		snapshot := handler.buildWellnessSnapshotFromCalculation(userID, wellnessResult)
		if createErr := handler.healthProfileRepository.CreateWellnessSnapshot(ctx, snapshot); createErr == nil {
			savedSnapshot = snapshot
		}
	}

	var savedInsight *models.AIInsightRecord
	if shouldRequestInsights && pyAIInsights != nil {
		nextVersion := 1
		if latestInsight != nil && latestInsight.Version >= 1 {
			nextVersion = latestInsight.Version + 1
		}
		newInsightRecord := buildAIInsightRecordFromPyAI(userID, nextVersion, profileRecord.PrimaryGoal, pyAIInsights)
		if createErr := handler.healthProfileRepository.CreateAIInsightRecord(ctx, newInsightRecord); createErr == nil {
			savedInsight = newInsightRecord
		}
	}

	return savedSnapshot, savedInsight, aiServiceAvailable
}

// buildHealthProfileResponsePayload converts a models.HealthProfile into a HealthProfileResponsePayload with BMI and decoded arrays.
func buildHealthProfileResponsePayload(profile *models.HealthProfile) HealthProfileResponsePayload {
	bmiValue, bmiClass, _ := health.CalculateAndValidateBMI(profile.CurrentWeightKg, profile.HeightCm)

	return HealthProfileResponsePayload{
		ID:                   profile.ID,
		UserID:               profile.UserID,
		Age:                  profile.Age,
		Gender:               profile.Gender,
		HeightCm:             profile.HeightCm,
		CurrentWeightKg:      profile.CurrentWeightKg,
		BaselineWeightKg:     profile.BaselineWeightKg,
		TargetWeightKg:       profile.TargetWeightKg,
		PreferredWeightUnit:  profile.PreferredWeightUnit,
		PreferredHeightUnit:  profile.PreferredHeightUnit,
		BMI:                  bmiValue,
		BMIClassification:    bmiClass,
		OccupationType:       profile.OccupationType,
		ActivityLevel:        profile.ActivityLevel,
		PrimaryGoal:          profile.PrimaryGoal,
		TargetActivityLevel:  profile.TargetActivityLevel,
		TargetWeeklyDays:     profile.TargetWeeklyDays,
		WeeklyActivityDays:   profile.WeeklyActivityDays,
		ExerciseTypes:        unmarshalStringSliceJSON(profile.ExerciseTypesJSON),
		SessionDuration:      profile.SessionDuration,
		FitnessLevel:         profile.FitnessLevel,
		PreferredEnvironment: profile.PreferredEnvironment,
		PreferredTimeOfDay:   profile.PreferredTimeOfDay,
		EnduranceMinutes:     profile.EnduranceMinutes,
		PushupsCount:         profile.PushupsCount,
		SquatsCount:          profile.SquatsCount,
		DietaryPreferences:   unmarshalStringSliceJSON(profile.DietaryPreferencesJSON),
		DietaryRestrictions:  unmarshalStringSliceJSON(profile.DietaryRestrictionsJSON),
		MedicalConditions:    profile.MedicalConditions,
		SleepHours:           profile.SleepHours,
		WaterLiters:          profile.WaterLiters,
		AllowAIProcessing:    profile.AllowAIProcessing,
		PublicProfileVisible: profile.PublicProfileVisible,
		EmailNotifications:   profile.EmailNotifications,
		CreatedAt:            profile.CreatedAt,
		UpdatedAt:            profile.UpdatedAt,
	}
}

// HandleGetHealthProfile handles GET /api/health/profile:
// returns the authenticated user's HealthProfile with calculated BMI and decoded arrays, or 404 Not Found.
func (handler *HealthProfileHandler) HandleGetHealthProfile(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only GET method is supported",
		})
		return
	}

	sessionRecord, authError := handler.authenticateSessionFromRequest(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	profileRecord, lookupError := handler.healthProfileRepository.FindHealthProfileByUserID(request.Context(), sessionRecord.UserID)
	if lookupError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to load health profile",
		})
		return
	}

	if profileRecord == nil {
		WriteJSONResponse(responseWriter, http.StatusNotFound, MessageOnlyResponsePayload{
			Message: "health profile not found",
		})
		return
	}

	WriteJSONResponse(responseWriter, http.StatusOK, buildHealthProfileResponsePayload(profileRecord))
}

// HandleSaveHealthProfile handles POST /api/health/profile and PUT /api/health/profile:
// validates demographics, normalizes height (cm/in -> cm) and weights (kg/lbs -> kg), validates BMI,
// upserts the HealthProfile, and logs a timestamped WeightEntry.
func (handler *HealthProfileHandler) HandleSaveHealthProfile(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost && request.Method != http.MethodPut {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only POST and PUT methods are supported",
		})
		return
	}

	sessionRecord, authError := handler.authenticateSessionFromRequest(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	var payload SaveHealthProfileRequestPayload
	if decodeError := json.NewDecoder(request.Body).Decode(&payload); decodeError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "invalid JSON request body",
		})
		return
	}

	if payload.Age < 13 || payload.Age > 120 {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "age must be between 13 and 120",
		})
		return
	}

	if strings.TrimSpace(payload.Gender) == "" {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "gender is required",
		})
		return
	}

	preferredHeightUnit := strings.ToLower(strings.TrimSpace(payload.HeightUnit))
	if preferredHeightUnit == "" {
		preferredHeightUnit = "cm"
	}

	preferredWeightUnit := strings.ToLower(strings.TrimSpace(payload.WeightUnit))
	if preferredWeightUnit == "" {
		preferredWeightUnit = "kg"
	}

	normalizedHeightCm, heightError := health.NormalizeHeightToCentimeters(payload.Height, preferredHeightUnit)
	if heightError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: heightError.Error(),
		})
		return
	}

	normalizedCurrentWeightKg, currentWeightError := health.NormalizeWeightToKilograms(payload.CurrentWeight, preferredWeightUnit)
	if currentWeightError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: currentWeightError.Error(),
		})
		return
	}

	normalizedTargetWeightKg, targetWeightError := health.NormalizeWeightToKilograms(payload.TargetWeight, preferredWeightUnit)
	if targetWeightError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: targetWeightError.Error(),
		})
		return
	}

	if _, _, bmiError := health.CalculateAndValidateBMI(normalizedCurrentWeightKg, normalizedHeightCm); bmiError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: bmiError.Error(),
		})
		return
	}

	if _, _, targetBMIError := health.CalculateAndValidateBMI(normalizedTargetWeightKg, normalizedHeightCm); targetBMIError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: targetBMIError.Error(),
		})
		return
	}

	allowAIProcessing := true
	if payload.AllowAIProcessing != nil {
		allowAIProcessing = *payload.AllowAIProcessing
	}

	publicProfileVisible := false
	if payload.PublicProfileVisible != nil {
		publicProfileVisible = *payload.PublicProfileVisible
	}

	emailNotifications := true
	if payload.EmailNotifications != nil {
		emailNotifications = *payload.EmailNotifications
	}

	profileID, _ := generateRandomUUIDv4()
	weightEntryID, _ := generateRandomUUIDv4()

	profileRecord := &models.HealthProfile{
		ID:                      profileID,
		UserID:                  sessionRecord.UserID,
		Age:                     payload.Age,
		Gender:                  strings.TrimSpace(payload.Gender),
		HeightCm:                normalizedHeightCm,
		CurrentWeightKg:         normalizedCurrentWeightKg,
		BaselineWeightKg:        normalizedCurrentWeightKg,
		TargetWeightKg:          normalizedTargetWeightKg,
		PreferredWeightUnit:     preferredWeightUnit,
		PreferredHeightUnit:     preferredHeightUnit,
		OccupationType:          strings.TrimSpace(payload.OccupationType),
		ActivityLevel:           strings.TrimSpace(payload.ActivityLevel),
		PrimaryGoal:             strings.TrimSpace(payload.PrimaryGoal),
		TargetActivityLevel:     strings.TrimSpace(payload.TargetActivityLevel),
		TargetWeeklyDays:        payload.TargetWeeklyDays,
		WeeklyActivityDays:      payload.WeeklyActivityDays,
		ExerciseTypesJSON:       marshalStringSliceJSON(payload.ExerciseTypes),
		SessionDuration:         strings.TrimSpace(payload.SessionDuration),
		FitnessLevel:            strings.TrimSpace(payload.FitnessLevel),
		PreferredEnvironment:    strings.TrimSpace(payload.PreferredEnvironment),
		PreferredTimeOfDay:      strings.TrimSpace(payload.PreferredTimeOfDay),
		EnduranceMinutes:        payload.EnduranceMinutes,
		PushupsCount:            payload.PushupsCount,
		SquatsCount:             payload.SquatsCount,
		DietaryPreferencesJSON:  marshalStringSliceJSON(payload.DietaryPreferences),
		DietaryRestrictionsJSON: marshalStringSliceJSON(payload.DietaryRestrictions),
		MedicalConditions:       strings.TrimSpace(payload.MedicalConditions),
		SleepHours:              payload.SleepHours,
		WaterLiters:             payload.WaterLiters,
		AllowAIProcessing:       allowAIProcessing,
		PublicProfileVisible:    publicProfileVisible,
		EmailNotifications:      emailNotifications,
	}

	weightEntry := &models.WeightEntry{
		ID:         weightEntryID,
		UserID:     sessionRecord.UserID,
		WeightKg:   normalizedCurrentWeightKg,
		RawValue:   payload.CurrentWeight,
		RawUnit:    preferredWeightUnit,
		RecordedAt: handler.nextUniqueWeightTimestampUTC(nil),
	}

	if upsertError := handler.healthProfileRepository.UpsertHealthProfileWithWeightEntry(
		request.Context(),
		profileRecord,
		weightEntry,
	); upsertError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to save health profile",
		})
		return
	}

	_, _, _ = handler.recalculateWellnessAndPersistOptionalAIInsights(request.Context(), sessionRecord.UserID, true, false)

	WriteJSONResponse(responseWriter, http.StatusOK, buildHealthProfileResponsePayload(profileRecord))
}

// HandleLogWeightEntry handles POST /api/health/weight:
// normalizes the submitted weight into kilograms, assigns a unique RecordedAt timestamp, and persists the WeightEntry.
func (handler *HealthProfileHandler) HandleLogWeightEntry(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only POST method is supported",
		})
		return
	}

	sessionRecord, authError := handler.authenticateSessionFromRequest(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	var payload LogWeightEntryRequestPayload
	if decodeError := json.NewDecoder(request.Body).Decode(&payload); decodeError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "invalid JSON request body",
		})
		return
	}

	unit := strings.ToLower(strings.TrimSpace(payload.Unit))
	if unit == "" {
		unit = "kg"
	}

	normalizedWeightKg, normalizeError := health.NormalizeWeightToKilograms(payload.Weight, unit)
	if normalizeError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: normalizeError.Error(),
		})
		return
	}

	entryID, _ := generateRandomUUIDv4()
	weightEntry := &models.WeightEntry{
		ID:         entryID,
		UserID:     sessionRecord.UserID,
		WeightKg:   normalizedWeightKg,
		RawValue:   payload.Weight,
		RawUnit:    unit,
		RecordedAt: handler.nextUniqueWeightTimestampUTC(payload.RecordedAt),
	}

	if createError := handler.healthProfileRepository.CreateWeightEntry(request.Context(), weightEntry); createError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to log weight entry",
		})
		return
	}

	_, _, _ = handler.recalculateWellnessAndPersistOptionalAIInsights(request.Context(), sessionRecord.UserID, false, false)

	WriteJSONResponse(responseWriter, http.StatusCreated, weightEntry)
}

// HandleListWeightEntries handles GET /api/health/weight:
// returns the authenticated user's chronological WeightEntry history.
func (handler *HealthProfileHandler) HandleListWeightEntries(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only GET method is supported",
		})
		return
	}

	sessionRecord, authError := handler.authenticateSessionFromRequest(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	entries, listError := handler.healthProfileRepository.ListWeightEntriesByUserID(request.Context(), sessionRecord.UserID)
	if listError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to list weight entries",
		})
		return
	}

	WriteJSONResponse(responseWriter, http.StatusOK, entries)
}

// HandleLogActivityEntry handles POST /api/health/activity:
// validates exerciseType, durationMinutes > 0, intensity, and recordedAt, and rejects duplicate timestamps with 409 Conflict.
func (handler *HealthProfileHandler) HandleLogActivityEntry(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only POST method is supported",
		})
		return
	}

	sessionRecord, authError := handler.authenticateSessionFromRequest(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	var payload LogActivityEntryRequestPayload
	if decodeError := json.NewDecoder(request.Body).Decode(&payload); decodeError != nil {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "invalid JSON request body",
		})
		return
	}

	if strings.TrimSpace(payload.ExerciseType) == "" {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "exerciseType is required",
		})
		return
	}

	if payload.DurationMinutes <= 0 || payload.DurationMinutes > 1440 {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "durationMinutes must be between 1 and 1440",
		})
		return
	}

	if strings.TrimSpace(payload.Intensity) == "" {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "intensity is required",
		})
		return
	}

	if payload.RecordedAt.IsZero() {
		WriteJSONResponse(responseWriter, http.StatusBadRequest, MessageOnlyResponsePayload{
			Message: "recordedAt timestamp is required",
		})
		return
	}

	entryID, _ := generateRandomUUIDv4()
	activityEntry := &models.ActivityEntry{
		ID:              entryID,
		UserID:          sessionRecord.UserID,
		ExerciseType:    strings.TrimSpace(payload.ExerciseType),
		DurationMinutes: payload.DurationMinutes,
		Intensity:       strings.TrimSpace(payload.Intensity),
		Notes:           strings.TrimSpace(payload.Notes),
		RecordedAt:      payload.RecordedAt.UTC(),
	}

	if createError := handler.healthProfileRepository.CreateActivityEntry(request.Context(), activityEntry); createError != nil {
		if errors.Is(createError, repositories.ErrDuplicateActivityTimestamp) {
			WriteJSONResponse(responseWriter, http.StatusConflict, MessageOnlyResponsePayload{
				Message: "Duplicate activity entry for the same timestamp is not allowed",
			})
			return
		}

		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to log activity entry",
		})
		return
	}

	_, _, _ = handler.recalculateWellnessAndPersistOptionalAIInsights(request.Context(), sessionRecord.UserID, false, false)

	WriteJSONResponse(responseWriter, http.StatusCreated, activityEntry)
}

// HandleListActivityEntries handles GET /api/health/activity:
// returns the authenticated user's chronological ActivityEntry history.
func (handler *HealthProfileHandler) HandleListActivityEntries(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only GET method is supported",
		})
		return
	}

	sessionRecord, authError := handler.authenticateSessionFromRequest(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	entries, listError := handler.healthProfileRepository.ListActivityEntriesByUserID(request.Context(), sessionRecord.UserID)
	if listError != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to list activity entries",
		})
		return
	}

	WriteJSONResponse(responseWriter, http.StatusOK, entries)
}

// HandleExportUserHealthData handles GET /api/health/export:
// returns a comprehensive GDPR JSON export of the authenticated user's metadata, HealthProfile,
// historical WeightEntries with timestamps, and historical ActivityEntries with timestamps.
func (handler *HealthProfileHandler) HandleExportUserHealthData(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only GET method is supported",
		})
		return
	}

	sessionRecord, authError := handler.authenticateSessionFromRequest(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	profileRecord, profileErr := handler.healthProfileRepository.FindHealthProfileByUserID(request.Context(), sessionRecord.UserID)
	if profileErr != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to export health profile",
		})
		return
	}

	weightEntries, weightErr := handler.healthProfileRepository.ListWeightEntriesByUserID(request.Context(), sessionRecord.UserID)
	if weightErr != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to export weight entries",
		})
		return
	}

	activityEntries, activityErr := handler.healthProfileRepository.ListActivityEntriesByUserID(request.Context(), sessionRecord.UserID)
	if activityErr != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to export activity entries",
		})
		return
	}

	var profilePayload *HealthProfileResponsePayload
	if profileRecord != nil {
		built := buildHealthProfileResponsePayload(profileRecord)
		profilePayload = &built
	}

	if weightEntries == nil {
		weightEntries = []models.WeightEntry{}
	}

	if activityEntries == nil {
		activityEntries = []models.ActivityEntry{}
	}

	exportResponse := GDPRHealthDataExportResponsePayload{
		User: GDPRUserMetadataExportPayload{
			ID:         sessionRecord.UserID,
			Email:      sessionRecord.User.Email,
			Consented:  sessionRecord.User.Consented,
			ExportedAt: time.Now().UTC(),
		},
		HealthProfile:   profilePayload,
		WeightEntries:   weightEntries,
		ActivityEntries: activityEntries,
	}

	WriteJSONResponse(responseWriter, http.StatusOK, exportResponse)
}

// HandleGetHealthDashboard handles GET /api/health/dashboard:
// returns the user's HealthProfile, latest WellnessSnapshot, WellnessSnapshot history, WeightEntry history,
// ActivityEntry history, latest AIInsightRecord, AIInsightRecord history, aiServiceAvailable, and isCachedFallback.
func (handler *HealthProfileHandler) HandleGetHealthDashboard(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only GET method is supported",
		})
		return
	}

	sessionRecord, authError := handler.authenticateSessionFromRequest(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	ctx := request.Context()
	profileRecord, profileErr := handler.healthProfileRepository.FindHealthProfileByUserID(ctx, sessionRecord.UserID)
	if profileErr != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to load health profile",
		})
		return
	}

	if profileRecord == nil {
		WriteJSONResponse(responseWriter, http.StatusNotFound, MessageOnlyResponsePayload{
			Message: "health profile not found",
		})
		return
	}

	weightHistory, _ := handler.healthProfileRepository.ListWeightEntriesByUserID(ctx, sessionRecord.UserID)
	activityHistory, _ := handler.healthProfileRepository.ListActivityEntriesByUserID(ctx, sessionRecord.UserID)
	wellnessHistory, _ := handler.healthProfileRepository.ListWellnessSnapshotsByUserID(ctx, sessionRecord.UserID)
	latestInsights, _ := handler.healthProfileRepository.FindLatestAIInsightRecordByUserID(ctx, sessionRecord.UserID)

	aiServiceAvailable := true
	isCachedFallback := false

	needsInitialWellness := len(wellnessHistory) == 0
	needsLiveInsights := profileRecord.AllowAIProcessing && (latestInsights == nil ||
		!strings.EqualFold(strings.TrimSpace(latestInsights.PrimaryGoalSnapshot), strings.TrimSpace(profileRecord.PrimaryGoal)))

	if needsInitialWellness || needsLiveInsights {
		_, generatedInsight, serviceReachable := handler.recalculateWellnessAndPersistOptionalAIInsights(
			ctx,
			sessionRecord.UserID,
			false,
			needsLiveInsights,
		)
		aiServiceAvailable = serviceReachable
		if generatedInsight != nil {
			latestInsights = generatedInsight
		} else if !serviceReachable && latestInsights != nil {
			isCachedFallback = true
		}
		wellnessHistory, _ = handler.healthProfileRepository.ListWellnessSnapshotsByUserID(ctx, sessionRecord.UserID)
	}

	insightHistory, _ := handler.healthProfileRepository.ListAIInsightRecordsByUserID(ctx, sessionRecord.UserID)

	var latestWellness *models.WellnessSnapshot
	if len(wellnessHistory) > 0 {
		lastSnapshot := wellnessHistory[len(wellnessHistory)-1]
		latestWellness = &lastSnapshot
	}

	if wellnessHistory == nil {
		wellnessHistory = []models.WellnessSnapshot{}
	}
	if weightHistory == nil {
		weightHistory = []models.WeightEntry{}
	}
	if activityHistory == nil {
		activityHistory = []models.ActivityEntry{}
	}
	if insightHistory == nil {
		insightHistory = []models.AIInsightRecord{}
	}

	builtProfile := buildHealthProfileResponsePayload(profileRecord)
	dashboardPayload := HealthDashboardResponsePayload{
		HealthProfile:      &builtProfile,
		LatestWellness:     latestWellness,
		WellnessHistory:    wellnessHistory,
		WeightHistory:      weightHistory,
		ActivityHistory:    activityHistory,
		LatestInsights:     latestInsights,
		InsightHistory:     insightHistory,
		AIServiceAvailable: aiServiceAvailable,
		IsCachedFallback:   isCachedFallback,
	}

	WriteJSONResponse(responseWriter, http.StatusOK, dashboardPayload)
}

// HandleRegenerateAIInsights handles POST /api/health/insights/regenerate:
// forces live regeneration of AI insights via py-ai and falls back to the latest cached AIInsightRecord
// with isCachedFallback: true and 200 OK if py-ai is unavailable.
func (handler *HealthProfileHandler) HandleRegenerateAIInsights(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only POST method is supported",
		})
		return
	}

	sessionRecord, authError := handler.authenticateSessionFromRequest(request)
	if authError != nil {
		WriteJSONResponse(responseWriter, http.StatusUnauthorized, MessageOnlyResponsePayload{
			Message: "unauthorized",
		})
		return
	}

	ctx := request.Context()
	profileRecord, profileErr := handler.healthProfileRepository.FindHealthProfileByUserID(ctx, sessionRecord.UserID)
	if profileErr != nil {
		WriteJSONResponse(responseWriter, http.StatusInternalServerError, MessageOnlyResponsePayload{
			Message: "failed to load health profile",
		})
		return
	}

	if profileRecord == nil {
		WriteJSONResponse(responseWriter, http.StatusNotFound, MessageOnlyResponsePayload{
			Message: "health profile not found",
		})
		return
	}

	cachedInsightsBeforeCall, _ := handler.healthProfileRepository.FindLatestAIInsightRecordByUserID(ctx, sessionRecord.UserID)
	_, generatedInsight, serviceReachable := handler.recalculateWellnessAndPersistOptionalAIInsights(
		ctx,
		sessionRecord.UserID,
		false,
		true,
	)

	insightHistory, _ := handler.healthProfileRepository.ListAIInsightRecordsByUserID(ctx, sessionRecord.UserID)
	if insightHistory == nil {
		insightHistory = []models.AIInsightRecord{}
	}

	latestInsights := generatedInsight
	isCachedFallback := false
	if latestInsights == nil && cachedInsightsBeforeCall != nil {
		latestInsights = cachedInsightsBeforeCall
		isCachedFallback = true
	}

	WriteJSONResponse(responseWriter, http.StatusOK, RegenerateAIInsightsResponsePayload{
		LatestInsights:     latestInsights,
		InsightHistory:     insightHistory,
		AIServiceAvailable: serviceReachable,
		IsCachedFallback:   isCachedFallback,
	})
}
