// This file defines the GORM WellnessSnapshot and AIInsightRecord domain models
// mapped to the "wellness_snapshots" and "ai_insight_records" PostgreSQL tables.
package models

import "time"

// WellnessSnapshot stores a point-in-time historical Wellness Score breakdown, BMI calculation,
// and achieved milestones JSON snapshot for a user.
type WellnessSnapshot struct {
	ID                string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID            string    `gorm:"column:user_id;type:uuid;not null;index" json:"userId"`
	User              User      `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	BMI               float64   `gorm:"column:bmi;not null" json:"bmi"`
	BMIClassification string    `gorm:"column:bmi_classification;size:32;not null" json:"bmiClassification"`
	WellnessScore     float64   `gorm:"column:wellness_score;not null" json:"wellnessScore"`
	BMIScore          float64   `gorm:"column:bmi_score;not null" json:"bmiScore"`
	ActivityScore     float64   `gorm:"column:activity_score;not null" json:"activityScore"`
	ProgressScore     float64   `gorm:"column:progress_score;not null" json:"progressScore"`
	HabitsScore       float64   `gorm:"column:habits_score;not null" json:"habitsScore"`
	MilestonesJSON    string    `gorm:"column:milestones_json;type:text;not null" json:"milestonesJson"`
	RecordedAt        time.Time `gorm:"column:recorded_at;not null;index" json:"recordedAt"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

// TableName specifies the PostgreSQL table name for WellnessSnapshot records.
func (WellnessSnapshot) TableName() string {
	return "wellness_snapshots"
}

// AIInsightRecord stores a versioned AI-generated health analysis, progress evaluation,
// prioritized recommendations JSON, and weekly/monthly summaries for a user.
type AIInsightRecord struct {
	ID                   string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID               string    `gorm:"column:user_id;type:uuid;not null;index" json:"userId"`
	User                 User      `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
	Version              int       `gorm:"column:version;not null" json:"version"`
	PrimaryGoalSnapshot  string    `gorm:"column:primary_goal_snapshot;size:64;not null" json:"primaryGoalSnapshot"`
	PromptStrategy       string    `gorm:"column:prompt_strategy;size:32;not null" json:"promptStrategy"`
	HealthStatusAnalysis string    `gorm:"column:health_status_analysis;type:text;not null" json:"healthStatusAnalysis"`
	ProgressEvaluation   string    `gorm:"column:progress_evaluation;type:text;not null" json:"progressEvaluation"`
	RecommendationsJSON  string    `gorm:"column:recommendations_json;type:text;not null" json:"recommendationsJson"`
	WeeklySummary        string    `gorm:"column:weekly_summary;type:text;not null" json:"weeklySummary"`
	MonthlySummary       string    `gorm:"column:monthly_summary;type:text;not null" json:"monthlySummary"`
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime;index" json:"createdAt"`
}

// TableName specifies the PostgreSQL table name for AIInsightRecord records.
func (AIInsightRecord) TableName() string {
	return "ai_insight_records"
}
