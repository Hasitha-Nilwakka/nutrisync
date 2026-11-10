// This file provides PII anonymization for outbound AI payloads, deterministic Wellness Score
// and milestone calculation fallbacks, and the HTTP client for the py-ai analytics service.
package health

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"nutrysync/go-api/models"
)

const (
	anonymizedSubjectIDSaltPrefix = "nutrisync-anon:"
	defaultPyAIServiceBaseURL     = "http://localhost:8000"
	defaultPyAIHTTPTimeout        = 15 * time.Second
)

// AnonymizedCurrentState represents the user's current physical and lifestyle metrics stripped of PII.
type AnonymizedCurrentState struct {
	Weight             float64 `json:"weight"`
	Height             float64 `json:"height"`
	Age                int     `json:"age"`
	Gender             string  `json:"gender"`
	ActivityLevel      string  `json:"activity_level"`
	WeeklyActivityDays int     `json:"weekly_activity_days"`
	SleepHours         float64 `json:"sleep_hours"`
	WaterLiters        float64 `json:"water_liters"`
	BMI                float64 `json:"bmi"`
	BMIClassification  string  `json:"bmi_classification"`
}

// AnonymizedTargetState represents the user's health and fitness goals stripped of PII.
type AnonymizedTargetState struct {
	Weight           float64 `json:"weight"`
	ActivityLevel    string  `json:"activity_level"`
	PrimaryGoal      string  `json:"primary_goal"`
	TargetWeeklyDays int     `json:"target_weekly_days"`
	BaselineWeight   float64 `json:"baseline_weight"`
}

// AnonymizedFitnessAssessment represents the user's baseline fitness assessment indicators.
type AnonymizedFitnessAssessment struct {
	FitnessLevel     string `json:"fitness_level"`
	SessionDuration  string `json:"session_duration"`
	EnduranceMinutes int    `json:"endurance_minutes"`
	PushupsCount     int    `json:"pushups_count"`
	SquatsCount      int    `json:"squats_count"`
}

// AnonymizedWeightHistoryPoint represents a single historical weight measurement without UserID or entry ID.
type AnonymizedWeightHistoryPoint struct {
	WeightKg   float64   `json:"weight_kg"`
	RecordedAt time.Time `json:"recorded_at"`
}

// AnonymizedActivityHistoryPoint represents a single historical activity session without UserID or entry ID.
type AnonymizedActivityHistoryPoint struct {
	ExerciseType    string    `json:"exercise_type"`
	DurationMinutes int       `json:"duration_minutes"`
	Intensity       string    `json:"intensity"`
	RecordedAt      time.Time `json:"recorded_at"`
}

// AnonymizedUserMetrics represents the structured health metrics sent to py-ai.
type AnonymizedUserMetrics struct {
	CurrentState      AnonymizedCurrentState           `json:"current_state"`
	TargetState       AnonymizedTargetState            `json:"target_state"`
	Preferences       []string                         `json:"preferences"`
	Restrictions      []string                         `json:"restrictions"`
	MedicalConditions string                           `json:"medical_conditions"`
	FitnessAssessment AnonymizedFitnessAssessment      `json:"fitness_assessment"`
	WeightHistory     []AnonymizedWeightHistoryPoint   `json:"weight_history"`
	ActivityHistory   []AnonymizedActivityHistoryPoint `json:"activity_history"`
}

// AnonymizedAIAnalysisPayload is the PII-free request payload sent from go-api to py-ai (POST /api/v1/analyze).
type AnonymizedAIAnalysisPayload struct {
	SubjectID          string                `json:"subject_id"`
	RegenerateInsights bool                  `json:"regenerate_insights"`
	UserMetrics        AnonymizedUserMetrics `json:"user_metrics"`
}

// WellnessMilestone represents a single tracked health or fitness milestone.
type WellnessMilestone struct {
	ID          string  `json:"id"`
	Category    string  `json:"category"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Achieved    bool    `json:"achieved"`
	ProgressPct float64 `json:"progressPct"`
}

// WellnessCalculationResult contains the composite Wellness Score, component scores (0-100),
// BMI details, and achieved/in-progress milestones.
type WellnessCalculationResult struct {
	BMI               float64             `json:"bmi"`
	BMIClassification string              `json:"bmiClassification"`
	WellnessScore     float64             `json:"wellnessScore"`
	BMIScore          float64             `json:"bmiScore"`
	ActivityScore     float64             `json:"activityScore"`
	ProgressScore     float64             `json:"progressScore"`
	HabitsScore       float64             `json:"habitsScore"`
	Milestones        []WellnessMilestone `json:"milestones"`
}

// PrioritizedAIRecommendation represents a single actionable recommendation with priority and expected impact.
type PrioritizedAIRecommendation struct {
	Title          string `json:"title"`
	Category       string `json:"category"`
	Priority       string `json:"priority"`
	Action         string `json:"action"`
	ExpectedImpact string `json:"expectedImpact"`
}

// PyAIInsightsPayload represents the AI-generated insights section returned by py-ai.
type PyAIInsightsPayload struct {
	PromptStrategy       string                        `json:"promptStrategy"`
	HealthStatusAnalysis string                        `json:"healthStatusAnalysis"`
	ProgressEvaluation   string                        `json:"progressEvaluation"`
	Recommendations      []PrioritizedAIRecommendation `json:"recommendations"`
	WeeklySummary        string                        `json:"weeklySummary"`
	MonthlySummary       string                        `json:"monthlySummary"`
}

// PyAIAnalysisResponse represents the full JSON response returned by POST ${PY_AI_URL}/api/v1/analyze.
type PyAIAnalysisResponse struct {
	Wellness *WellnessCalculationResult `json:"wellness"`
	Insights *PyAIInsightsPayload       `json:"insights"`
}

// PyAIAnalyticsClient defines the contract for communicating with the py-ai analytics microservice.
type PyAIAnalyticsClient interface {
	AnalyzeUserHealthAndGenerateInsights(
		ctx context.Context,
		payload *AnonymizedAIAnalysisPayload,
	) (*PyAIAnalysisResponse, error)
}

// HTTPPyAIAnalyticsClient implements PyAIAnalyticsClient over HTTP against ${PY_AI_URL}/api/v1/analyze.
type HTTPPyAIAnalyticsClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPPyAIAnalyticsClient constructs an HTTPPyAIAnalyticsClient using baseURL (or PY_AI_URL / http://localhost:8000 if blank).
func NewHTTPPyAIAnalyticsClient(baseURL string, httpClient *http.Client) *HTTPPyAIAnalyticsClient {
	resolvedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if resolvedBaseURL == "" {
		resolvedBaseURL = strings.TrimRight(strings.TrimSpace(os.Getenv("PY_AI_URL")), "/")
	}
	if resolvedBaseURL == "" {
		resolvedBaseURL = defaultPyAIServiceBaseURL
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultPyAIHTTPTimeout}
	}

	return &HTTPPyAIAnalyticsClient{
		baseURL:    resolvedBaseURL,
		httpClient: httpClient,
	}
}

// decodeJSONStringArray parses a JSON-encoded string array or returns an empty slice.
func decodeJSONStringArray(rawJSON string) []string {
	if strings.TrimSpace(rawJSON) == "" {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal([]byte(rawJSON), &values); err != nil || values == nil {
		return []string{}
	}
	return values
}

// BuildAnonymizedAIAnalysisPayload constructs a PII-stripped AnonymizedAIAnalysisPayload from a user's
// HealthProfile, WeightEntry history, and ActivityEntry history.
// Never includes email, raw user ID, session ID, or IP address; uses SHA-256("nutrisync-anon:" + profile.UserID).
func BuildAnonymizedAIAnalysisPayload(
	profile *models.HealthProfile,
	weights []models.WeightEntry,
	activities []models.ActivityEntry,
	regenerateInsights bool,
) (*AnonymizedAIAnalysisPayload, error) {
	if profile == nil || strings.TrimSpace(profile.UserID) == "" {
		return nil, errors.New("health profile with non-empty user ID is required")
	}

	subjectHash := sha256.Sum256([]byte(anonymizedSubjectIDSaltPrefix + strings.TrimSpace(profile.UserID)))
	subjectID := hex.EncodeToString(subjectHash[:])

	bmiValue, bmiClassification, bmiErr := CalculateAndValidateBMI(profile.CurrentWeightKg, profile.HeightCm)
	if bmiErr != nil {
		return nil, bmiErr
	}

	baselineWeight := profile.BaselineWeightKg
	if baselineWeight <= 0 {
		if len(weights) > 0 && weights[0].WeightKg > 0 {
			baselineWeight = weights[0].WeightKg
		} else {
			baselineWeight = profile.CurrentWeightKg
		}
	}

	preferences := make([]string, 0)
	for _, exerciseType := range decodeJSONStringArray(profile.ExerciseTypesJSON) {
		if trimmed := strings.TrimSpace(exerciseType); trimmed != "" {
			preferences = append(preferences, trimmed)
		}
	}
	if env := strings.TrimSpace(profile.PreferredEnvironment); env != "" {
		preferences = append(preferences, "environment:"+env)
	}
	if tod := strings.TrimSpace(profile.PreferredTimeOfDay); tod != "" {
		preferences = append(preferences, "time_of_day:"+tod)
	}
	for _, dietaryPref := range decodeJSONStringArray(profile.DietaryPreferencesJSON) {
		if trimmed := strings.TrimSpace(dietaryPref); trimmed != "" {
			preferences = append(preferences, trimmed)
		}
	}

	restrictions := make([]string, 0)
	for _, restriction := range decodeJSONStringArray(profile.DietaryRestrictionsJSON) {
		if trimmed := strings.TrimSpace(restriction); trimmed != "" {
			restrictions = append(restrictions, trimmed)
		}
	}

	weightHistory := make([]AnonymizedWeightHistoryPoint, 0, len(weights))
	for _, entry := range weights {
		weightHistory = append(weightHistory, AnonymizedWeightHistoryPoint{
			WeightKg:   entry.WeightKg,
			RecordedAt: entry.RecordedAt.UTC(),
		})
	}

	activityHistory := make([]AnonymizedActivityHistoryPoint, 0, len(activities))
	for _, entry := range activities {
		activityHistory = append(activityHistory, AnonymizedActivityHistoryPoint{
			ExerciseType:    entry.ExerciseType,
			DurationMinutes: entry.DurationMinutes,
			Intensity:       entry.Intensity,
			RecordedAt:      entry.RecordedAt.UTC(),
		})
	}

	return &AnonymizedAIAnalysisPayload{
		SubjectID:          subjectID,
		RegenerateInsights: regenerateInsights,
		UserMetrics: AnonymizedUserMetrics{
			CurrentState: AnonymizedCurrentState{
				Weight:             profile.CurrentWeightKg,
				Height:             profile.HeightCm,
				Age:                profile.Age,
				Gender:             profile.Gender,
				ActivityLevel:      profile.ActivityLevel,
				WeeklyActivityDays: profile.WeeklyActivityDays,
				SleepHours:         profile.SleepHours,
				WaterLiters:        profile.WaterLiters,
				BMI:                bmiValue,
				BMIClassification:  bmiClassification,
			},
			TargetState: AnonymizedTargetState{
				Weight:           profile.TargetWeightKg,
				ActivityLevel:    profile.TargetActivityLevel,
				PrimaryGoal:      profile.PrimaryGoal,
				TargetWeeklyDays: profile.TargetWeeklyDays,
				BaselineWeight:   baselineWeight,
			},
			Preferences:       preferences,
			Restrictions:      restrictions,
			MedicalConditions: strings.TrimSpace(profile.MedicalConditions),
			FitnessAssessment: AnonymizedFitnessAssessment{
				FitnessLevel:     profile.FitnessLevel,
				SessionDuration:  profile.SessionDuration,
				EnduranceMinutes: profile.EnduranceMinutes,
				PushupsCount:     profile.PushupsCount,
				SquatsCount:      profile.SquatsCount,
			},
			WeightHistory:   weightHistory,
			ActivityHistory: activityHistory,
		},
	}, nil
}

// clampScore constrains a score to [0.0, 100.0] rounded to 2 decimal places.
func clampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return roundToTwoDecimalPlaces(score)
}

// calculateBMIComponentScore scores a BMI value on a 0-100 scale (optimal around 18.5-24.9).
func calculateBMIComponentScore(bmi float64) float64 {
	switch {
	case bmi >= 18.5 && bmi < 25.0:
		distanceFromIdealCenter := math.Abs(bmi - 21.7)
		return clampScore(100.0 - (distanceFromIdealCenter * 3.0))
	case bmi < 18.5:
		deficit := 18.5 - bmi
		return clampScore(85.0 - (deficit * 10.0))
	case bmi >= 25.0 && bmi < 30.0:
		excess := bmi - 25.0
		return clampScore(85.0 - (excess * 6.0))
	default:
		excess := bmi - 30.0
		return clampScore(55.0 - (excess * 3.5))
	}
}

// activityLevelRank maps an activity level string to a numeric tier (1-4).
func activityLevelRank(level string) float64 {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "sedentary":
		return 1.0
	case "lightly_active", "light":
		return 2.0
	case "moderately_active", "moderate":
		return 3.0
	case "very_active", "active", "extra_active", "athlete":
		return 4.0
	default:
		return 2.0
	}
}

// countRecentWeeklyActivities counts activity entries recorded within the 7-day window ending at the latest activity or now.
func countRecentWeeklyActivities(activities []models.ActivityEntry) (int, int) {
	if len(activities) == 0 {
		return 0, 0
	}

	referenceTime := time.Now().UTC()
	for _, entry := range activities {
		if entry.RecordedAt.After(referenceTime) {
			referenceTime = entry.RecordedAt.UTC()
		}
	}

	sevenDaysAgo := referenceTime.Add(-7 * 24 * time.Hour)
	recentSessions := 0
	totalMinutes := 0
	for _, entry := range activities {
		if !entry.RecordedAt.Before(sevenDaysAgo) {
			recentSessions++
			if entry.DurationMinutes > 0 {
				totalMinutes += entry.DurationMinutes
			}
		}
	}

	if recentSessions == 0 {
		// Fall back to total logged sessions if all entries are older than 7 days relative to wall clock.
		recentSessions = len(activities)
		for _, entry := range activities {
			if entry.DurationMinutes > 0 {
				totalMinutes += entry.DurationMinutes
			}
		}
	}

	return recentSessions, totalMinutes
}

// CalculateDeterministicWellnessAndMilestones computes the 4 Wellness Score components (0-100),
// overall composite WellnessScore = round((BMIScore*0.3)+(ActivityScore*0.3)+(ProgressScore*0.2)+(HabitsScore*0.2), 2),
// and goal/activity/habit milestones.
func CalculateDeterministicWellnessAndMilestones(
	profile *models.HealthProfile,
	weights []models.WeightEntry,
	activities []models.ActivityEntry,
) (*WellnessCalculationResult, error) {
	if profile == nil {
		return nil, errors.New("health profile must not be nil")
	}

	currentWeightKg := profile.CurrentWeightKg
	if len(weights) > 0 && weights[len(weights)-1].WeightKg > 0 {
		currentWeightKg = weights[len(weights)-1].WeightKg
	}

	bmiValue, bmiClass, bmiErr := CalculateAndValidateBMI(currentWeightKg, profile.HeightCm)
	if bmiErr != nil {
		return nil, bmiErr
	}

	bmiScore := calculateBMIComponentScore(bmiValue)

	// Activity Score (0-100): combines weekly frequency vs target, activity level rank, and logged activity sessions.
	targetDays := profile.TargetWeeklyDays
	if targetDays <= 0 {
		targetDays = 4
	}

	recentSessions, totalMinutes := countRecentWeeklyActivities(activities)
	effectiveActiveDays := float64(profile.WeeklyActivityDays)
	if float64(recentSessions) > effectiveActiveDays {
		effectiveActiveDays = float64(recentSessions)
	}

	frequencyRatio := effectiveActiveDays / float64(targetDays)
	if frequencyRatio > 1.25 {
		frequencyRatio = 1.25
	}

	levelScore := (activityLevelRank(profile.ActivityLevel) / 4.0) * 30.0
	frequencyScore := math.Min(frequencyRatio, 1.0) * 50.0
	loggedBonus := math.Min(float64(recentSessions)*5.0+float64(totalMinutes)/30.0, 20.0)
	activityScore := clampScore(levelScore + frequencyScore + loggedBonus)

	// Progress Score (0-100): evaluates weight trajectory from BaselineWeightKg toward TargetWeightKg.
	baselineWeight := profile.BaselineWeightKg
	if baselineWeight <= 0 {
		if len(weights) > 0 && weights[0].WeightKg > 0 {
			baselineWeight = weights[0].WeightKg
		} else {
			baselineWeight = currentWeightKg
		}
	}

	targetWeight := profile.TargetWeightKg
	if targetWeight <= 0 {
		targetWeight = currentWeightKg
	}

	totalGoalDelta := math.Abs(baselineWeight - targetWeight)
	remainingDelta := math.Abs(currentWeightKg - targetWeight)

	var weightProgressFraction float64
	var progressScore float64
	if totalGoalDelta < 0.25 {
		if remainingDelta <= 1.0 {
			weightProgressFraction = 1.0
			progressScore = 95.0
		} else {
			weightProgressFraction = clampScore(100.0-remainingDelta*10.0) / 100.0
			progressScore = clampScore(90.0 - remainingDelta*8.0)
		}
	} else {
		achievedDelta := totalGoalDelta - remainingDelta
		weightProgressFraction = achievedDelta / totalGoalDelta
		if weightProgressFraction < 0 {
			weightProgressFraction = 0
		}
		if weightProgressFraction > 1.0 {
			weightProgressFraction = 1.0
		}
		// Base 40 points for establishing baseline + up to 60 points for progress toward target weight.
		progressScore = clampScore(40.0 + (weightProgressFraction * 60.0))
	}

	// Habits Score (0-100): evaluates sleep hours (optimal 7-9h) and water intake (optimal >= 2.0-2.5L).
	sleepScore := 35.0
	switch {
	case profile.SleepHours >= 7.0 && profile.SleepHours <= 9.0:
		sleepScore = 50.0
	case profile.SleepHours >= 6.0 && profile.SleepHours < 7.0:
		sleepScore = 38.0
	case profile.SleepHours > 9.0 && profile.SleepHours <= 10.0:
		sleepScore = 40.0
	case profile.SleepHours > 0:
		sleepScore = math.Max(15.0, 50.0-math.Abs(7.5-profile.SleepHours)*10.0)
	}

	hydrationScore := 25.0
	if profile.WaterLiters > 0 {
		hydrationRatio := profile.WaterLiters / 2.5
		if hydrationRatio > 1.0 {
			hydrationRatio = 1.0
		}
		hydrationScore = hydrationRatio * 50.0
	}
	habitsScore := clampScore(sleepScore + hydrationScore)

	wellnessScore := roundToTwoDecimalPlaces(
		(bmiScore * 0.3) + (activityScore * 0.3) + (progressScore * 0.2) + (habitsScore * 0.2),
	)

	weightProgressPct := roundToTwoDecimalPlaces(weightProgressFraction * 100.0)
	completedFivePctSteps := int(math.Floor(weightProgressPct / 5.0))
	nextFivePctTarget := (completedFivePctSteps + 1) * 5
	if nextFivePctTarget > 100 {
		nextFivePctTarget = 100
	}

	milestones := []WellnessMilestone{
		{
			ID:       "weight-5pct-step",
			Category: "weight",
			Title:    fmt.Sprintf("Weight Goal Progress (%d%% Step)", completedFivePctSteps*5),
			Description: fmt.Sprintf(
				"Progressed %.1f%% toward target weight (%.1f kg -> %.1f kg; next 5%% milestone at %d%%).",
				weightProgressPct,
				currentWeightKg,
				targetWeight,
				nextFivePctTarget,
			),
			Achieved:    completedFivePctSteps >= 1 || weightProgressPct >= 100.0,
			ProgressPct: weightProgressPct,
		},
		{
			ID:       "weekly-activity-target",
			Category: "activity",
			Title:    "Weekly Activity Consistency",
			Description: fmt.Sprintf(
				"Active %.0f of %d target weekly days (%d logged sessions).",
				effectiveActiveDays,
				targetDays,
				len(activities),
			),
			Achieved:    int(effectiveActiveDays) >= targetDays,
			ProgressPct: clampScore((effectiveActiveDays / float64(targetDays)) * 100.0),
		},
		{
			ID:       "daily-habits-consistency",
			Category: "habits",
			Title:    "Sleep & Hydration Balance",
			Description: fmt.Sprintf(
				"Logging %.1f hrs sleep and %.1f L water daily.",
				profile.SleepHours,
				profile.WaterLiters,
			),
			Achieved:    profile.SleepHours >= 7.0 && profile.WaterLiters >= 2.0,
			ProgressPct: habitsScore,
		},
	}

	return &WellnessCalculationResult{
		BMI:               bmiValue,
		BMIClassification: bmiClass,
		WellnessScore:     wellnessScore,
		BMIScore:          bmiScore,
		ActivityScore:     activityScore,
		ProgressScore:     progressScore,
		HabitsScore:       habitsScore,
		Milestones:        milestones,
	}, nil
}

// AnalyzeUserHealthAndGenerateInsights sends a POST request to ${PY_AI_URL}/api/v1/analyze with the anonymized payload.
func (client *HTTPPyAIAnalyticsClient) AnalyzeUserHealthAndGenerateInsights(
	ctx context.Context,
	payload *AnonymizedAIAnalysisPayload,
) (*PyAIAnalysisResponse, error) {
	if client == nil {
		return nil, errors.New("py-ai analytics client must not be nil")
	}

	if payload == nil {
		return nil, errors.New("anonymized AI analysis payload must not be nil")
	}

	requestBodyBytes, encodeErr := json.Marshal(payload)
	if encodeErr != nil {
		return nil, encodeErr
	}

	endpointURL := strings.TrimRight(client.baseURL, "/") + "/api/v1/analyze"
	httpRequest, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(requestBodyBytes))
	if reqErr != nil {
		return nil, reqErr
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	httpClient := client.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultPyAIHTTPTimeout}
	}

	httpResponse, doErr := httpClient.Do(httpRequest)
	if doErr != nil {
		return nil, doErr
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("py-ai analyze endpoint returned HTTP status %d", httpResponse.StatusCode)
	}

	var parsedResponse PyAIAnalysisResponse
	if decodeErr := json.NewDecoder(httpResponse.Body).Decode(&parsedResponse); decodeErr != nil {
		return nil, decodeErr
	}

	return &parsedResponse, nil
}
