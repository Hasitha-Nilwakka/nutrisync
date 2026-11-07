// This file defines the GORM HealthProfile, WeightEntry, and ActivityEntry domain models
// mapped to the "health_profiles", "weight_entries", and "activity_entries" PostgreSQL tables.
package models

import "time"

// HealthProfile stores a user's normalized physical metrics (cm/kg), unit preferences,
// lifestyle indicators, fitness targets, dietary preferences/restrictions, and privacy settings.
type HealthProfile struct {
	ID                      string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID                  string    `gorm:"column:user_id;type:uuid;uniqueIndex;not null" json:"userId"`
	User                    User      `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	Age                     int       `gorm:"column:age;not null" json:"age"`
	Gender                  string    `gorm:"column:gender;size:32;not null" json:"gender"`
	HeightCm                float64   `gorm:"column:height_cm;not null" json:"heightCm"`
	CurrentWeightKg         float64   `gorm:"column:current_weight_kg;not null" json:"currentWeightKg"`
	BaselineWeightKg        float64   `gorm:"column:baseline_weight_kg;not null" json:"baselineWeightKg"`
	TargetWeightKg          float64   `gorm:"column:target_weight_kg;not null" json:"targetWeightKg"`
	PreferredWeightUnit     string    `gorm:"column:preferred_weight_unit;size:8;default:'kg';not null" json:"preferredWeightUnit"`
	PreferredHeightUnit     string    `gorm:"column:preferred_height_unit;size:8;default:'cm';not null" json:"preferredHeightUnit"`
	OccupationType          string    `gorm:"column:occupation_type;size:64;not null" json:"occupationType"`
	ActivityLevel           string    `gorm:"column:activity_level;size:32;not null" json:"activityLevel"`
	PrimaryGoal             string    `gorm:"column:primary_goal;size:64;not null" json:"primaryGoal"`
	TargetActivityLevel     string    `gorm:"column:target_activity_level;size:32;not null" json:"targetActivityLevel"`
	TargetWeeklyDays        int       `gorm:"column:target_weekly_days;not null" json:"targetWeeklyDays"`
	WeeklyActivityDays      int       `gorm:"column:weekly_activity_days;not null" json:"weeklyActivityDays"`
	ExerciseTypesJSON       string    `gorm:"column:exercise_types_json;type:text;not null" json:"exerciseTypesJson"`
	SessionDuration         string    `gorm:"column:session_duration;size:32;not null" json:"sessionDuration"`
	FitnessLevel            string    `gorm:"column:fitness_level;size:32;not null" json:"fitnessLevel"`
	PreferredEnvironment    string    `gorm:"column:preferred_environment;size:32;not null" json:"preferredEnvironment"`
	PreferredTimeOfDay      string    `gorm:"column:preferred_time_of_day;size:32;not null" json:"preferredTimeOfDay"`
	EnduranceMinutes        int       `gorm:"column:endurance_minutes;not null" json:"enduranceMinutes"`
	PushupsCount            int       `gorm:"column:pushups_count;not null" json:"pushupsCount"`
	SquatsCount             int       `gorm:"column:squats_count;not null" json:"squatsCount"`
	DietaryPreferencesJSON  string    `gorm:"column:dietary_preferences_json;type:text;not null" json:"dietaryPreferencesJson"`
	DietaryRestrictionsJSON string    `gorm:"column:dietary_restrictions_json;type:text;not null" json:"dietaryRestrictionsJson"`
	MedicalConditions       string    `gorm:"column:medical_conditions;type:text" json:"medicalConditions"`
	SleepHours              float64   `gorm:"column:sleep_hours;not null" json:"sleepHours"`
	WaterLiters             float64   `gorm:"column:water_liters;not null" json:"waterLiters"`
	AllowAIProcessing       bool      `gorm:"column:allow_ai_processing;default:true;not null" json:"allowAiProcessing"`
	PublicProfileVisible    bool      `gorm:"column:public_profile_visible;default:false;not null" json:"publicProfileVisible"`
	EmailNotifications      bool      `gorm:"column:email_notifications;default:true;not null" json:"emailNotifications"`
	CreatedAt               time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt               time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

// TableName specifies the PostgreSQL table name for HealthProfile records.
func (HealthProfile) TableName() string {
	return "health_profiles"
}

// WeightEntry represents a historical weight measurement for a user with a unique RecordedAt timestamp.
type WeightEntry struct {
	ID         string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID     string    `gorm:"column:user_id;type:uuid;not null;uniqueIndex:idx_user_weight_recorded_at" json:"userId"`
	WeightKg   float64   `gorm:"column:weight_kg;not null" json:"weightKg"`
	RawValue   float64   `gorm:"column:raw_value;not null" json:"rawValue"`
	RawUnit    string    `gorm:"column:raw_unit;size:8;not null" json:"rawUnit"`
	RecordedAt time.Time `gorm:"column:recorded_at;not null;uniqueIndex:idx_user_weight_recorded_at" json:"recordedAt"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

// TableName specifies the PostgreSQL table name for WeightEntry records.
func (WeightEntry) TableName() string {
	return "weight_entries"
}

// ActivityEntry represents a logged physical activity session for a user with a unique RecordedAt timestamp.
type ActivityEntry struct {
	ID              string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID          string    `gorm:"column:user_id;type:uuid;not null;uniqueIndex:idx_user_activity_recorded_at" json:"userId"`
	ExerciseType    string    `gorm:"column:exercise_type;size:64;not null" json:"exerciseType"`
	DurationMinutes int       `gorm:"column:duration_minutes;not null" json:"durationMinutes"`
	Intensity       string    `gorm:"column:intensity;size:32;not null" json:"intensity"`
	Notes           string    `gorm:"column:notes;type:text" json:"notes,omitempty"`
	RecordedAt      time.Time `gorm:"column:recorded_at;not null;uniqueIndex:idx_user_activity_recorded_at" json:"recordedAt"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

// TableName specifies the PostgreSQL table name for ActivityEntry records.
func (ActivityEntry) TableName() string {
	return "activity_entries"
}
