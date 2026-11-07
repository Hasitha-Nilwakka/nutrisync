// This file defines the HealthProfileRepository interface and its GORM-backed implementation
// for persisting and querying HealthProfile records, historical WeightEntry logs, and ActivityEntry logs.
package repositories

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"nutrysync/go-api/models"
)

// ErrDuplicateActivityTimestamp is returned when an ActivityEntry with the same UserID and RecordedAt already exists.
var ErrDuplicateActivityTimestamp = errors.New("duplicate activity entry for the same timestamp is not allowed")

// HealthProfileRepository defines the persistence operations for user health profiles,
// historical weight tracking, and activity logging.
type HealthProfileRepository interface {
	FindHealthProfileByUserID(ctx context.Context, userID string) (*models.HealthProfile, error)
	UpsertHealthProfileWithWeightEntry(ctx context.Context, profile *models.HealthProfile, weightEntry *models.WeightEntry) error
	CreateWeightEntry(ctx context.Context, entry *models.WeightEntry) error
	ListWeightEntriesByUserID(ctx context.Context, userID string) ([]models.WeightEntry, error)
	CreateActivityEntry(ctx context.Context, entry *models.ActivityEntry) error
	ListActivityEntriesByUserID(ctx context.Context, userID string) ([]models.ActivityEntry, error)
}

// GormHealthProfileRepository implements HealthProfileRepository using a GORM *gorm.DB connection.
type GormHealthProfileRepository struct {
	databaseConnection *gorm.DB
}

// NewGormHealthProfileRepository constructs a GormHealthProfileRepository after verifying databaseConnection is non-nil.
func NewGormHealthProfileRepository(databaseConnection *gorm.DB) (*GormHealthProfileRepository, error) {
	if databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	return &GormHealthProfileRepository{
		databaseConnection: databaseConnection,
	}, nil
}

// FindHealthProfileByUserID retrieves a HealthProfile record by UserID.
// Returns (nil, nil) when gorm.ErrRecordNotFound occurs.
func (repository *GormHealthProfileRepository) FindHealthProfileByUserID(
	ctx context.Context,
	userID string,
) (*models.HealthProfile, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return nil, errors.New("user ID must not be empty")
	}

	var profileRecord models.HealthProfile
	queryResult := repository.databaseConnection.WithContext(ctx).
		Where("user_id = ?", trimmedUserID).
		First(&profileRecord)
	if queryResult.Error != nil {
		if errors.Is(queryResult.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, queryResult.Error
	}

	return &profileRecord, nil
}

// UpsertHealthProfileWithWeightEntry creates or updates a user's HealthProfile and inserts a corresponding
// WeightEntry inside a single GORM transaction.
func (repository *GormHealthProfileRepository) UpsertHealthProfileWithWeightEntry(
	ctx context.Context,
	profile *models.HealthProfile,
	weightEntry *models.WeightEntry,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	if profile == nil || strings.TrimSpace(profile.UserID) == "" {
		return errors.New("health profile with non-empty user ID is required")
	}

	if weightEntry == nil {
		return errors.New("weight entry must not be nil")
	}

	return repository.databaseConnection.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		var existingProfile models.HealthProfile
		lookupResult := transactionDB.Where("user_id = ?", profile.UserID).First(&existingProfile)
		if lookupResult.Error != nil && !errors.Is(lookupResult.Error, gorm.ErrRecordNotFound) {
			return lookupResult.Error
		}

		if errors.Is(lookupResult.Error, gorm.ErrRecordNotFound) || strings.TrimSpace(existingProfile.ID) == "" {
			if profile.BaselineWeightKg <= 0 {
				profile.BaselineWeightKg = profile.CurrentWeightKg
			}
			if createProfileErr := transactionDB.Create(profile).Error; createProfileErr != nil {
				return createProfileErr
			}
		} else {
			profile.ID = existingProfile.ID
			if profile.BaselineWeightKg <= 0 && existingProfile.BaselineWeightKg > 0 {
				profile.BaselineWeightKg = existingProfile.BaselineWeightKg
			}

			profileUpdates := map[string]any{
				"age":                       profile.Age,
				"gender":                    profile.Gender,
				"height_cm":                 profile.HeightCm,
				"current_weight_kg":         profile.CurrentWeightKg,
				"baseline_weight_kg":        profile.BaselineWeightKg,
				"target_weight_kg":          profile.TargetWeightKg,
				"preferred_weight_unit":     profile.PreferredWeightUnit,
				"preferred_height_unit":     profile.PreferredHeightUnit,
				"occupation_type":           profile.OccupationType,
				"activity_level":            profile.ActivityLevel,
				"primary_goal":              profile.PrimaryGoal,
				"target_activity_level":     profile.TargetActivityLevel,
				"target_weekly_days":        profile.TargetWeeklyDays,
				"weekly_activity_days":      profile.WeeklyActivityDays,
				"exercise_types_json":       profile.ExerciseTypesJSON,
				"session_duration":          profile.SessionDuration,
				"fitness_level":             profile.FitnessLevel,
				"preferred_environment":     profile.PreferredEnvironment,
				"preferred_time_of_day":     profile.PreferredTimeOfDay,
				"endurance_minutes":         profile.EnduranceMinutes,
				"pushups_count":             profile.PushupsCount,
				"squats_count":              profile.SquatsCount,
				"dietary_preferences_json":  profile.DietaryPreferencesJSON,
				"dietary_restrictions_json": profile.DietaryRestrictionsJSON,
				"medical_conditions":        profile.MedicalConditions,
				"sleep_hours":               profile.SleepHours,
				"water_liters":              profile.WaterLiters,
				"allow_ai_processing":       profile.AllowAIProcessing,
				"public_profile_visible":    profile.PublicProfileVisible,
				"email_notifications":       profile.EmailNotifications,
			}

			if updateErr := transactionDB.Model(&models.HealthProfile{}).
				Where("user_id = ?", profile.UserID).
				Updates(profileUpdates).Error; updateErr != nil {
				return updateErr
			}
		}

		weightEntry.UserID = profile.UserID
		return transactionDB.Create(weightEntry).Error
	})
}

// CreateWeightEntry inserts a new WeightEntry and updates current_weight_kg on the user's HealthProfile inside a transaction.
func (repository *GormHealthProfileRepository) CreateWeightEntry(
	ctx context.Context,
	entry *models.WeightEntry,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	if entry == nil || strings.TrimSpace(entry.UserID) == "" {
		return errors.New("weight entry with non-empty user ID is required")
	}

	return repository.databaseConnection.WithContext(ctx).Transaction(func(transactionDB *gorm.DB) error {
		if createErr := transactionDB.Create(entry).Error; createErr != nil {
			return createErr
		}

		return transactionDB.Model(&models.HealthProfile{}).
			Where("user_id = ?", entry.UserID).
			Update("current_weight_kg", entry.WeightKg).Error
	})
}

// ListWeightEntriesByUserID returns all WeightEntry records for userID ordered by recorded_at ASC.
func (repository *GormHealthProfileRepository) ListWeightEntriesByUserID(
	ctx context.Context,
	userID string,
) ([]models.WeightEntry, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return nil, errors.New("user ID must not be empty")
	}

	var entries []models.WeightEntry
	queryResult := repository.databaseConnection.WithContext(ctx).
		Where("user_id = ?", trimmedUserID).
		Order("recorded_at ASC").
		Find(&entries)
	if queryResult.Error != nil {
		return nil, queryResult.Error
	}

	return entries, nil
}

// CreateActivityEntry checks for an existing ActivityEntry with the same (UserID, RecordedAt),
// returning ErrDuplicateActivityTimestamp on collision, or inserts the new ActivityEntry.
func (repository *GormHealthProfileRepository) CreateActivityEntry(
	ctx context.Context,
	entry *models.ActivityEntry,
) error {
	if repository == nil || repository.databaseConnection == nil {
		return errors.New("database connection must not be nil")
	}

	if entry == nil || strings.TrimSpace(entry.UserID) == "" {
		return errors.New("activity entry with non-empty user ID is required")
	}

	if entry.RecordedAt.IsZero() {
		return errors.New("activity recordedAt timestamp is required")
	}

	var existingCount int64
	if countError := repository.databaseConnection.WithContext(ctx).
		Model(&models.ActivityEntry{}).
		Where("user_id = ? AND recorded_at = ?", strings.TrimSpace(entry.UserID), entry.RecordedAt).
		Count(&existingCount).Error; countError != nil {
		return countError
	}

	if existingCount > 0 {
		return ErrDuplicateActivityTimestamp
	}

	if createError := repository.databaseConnection.WithContext(ctx).Create(entry).Error; createError != nil {
		if strings.Contains(strings.ToLower(createError.Error()), "duplicate") ||
			strings.Contains(strings.ToLower(createError.Error()), "unique") {
			return ErrDuplicateActivityTimestamp
		}
		return createError
	}

	return nil
}

// ListActivityEntriesByUserID returns all ActivityEntry records for userID ordered by recorded_at ASC.
func (repository *GormHealthProfileRepository) ListActivityEntriesByUserID(
	ctx context.Context,
	userID string,
) ([]models.ActivityEntry, error) {
	if repository == nil || repository.databaseConnection == nil {
		return nil, errors.New("database connection must not be nil")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return nil, errors.New("user ID must not be empty")
	}

	var entries []models.ActivityEntry
	queryResult := repository.databaseConnection.WithContext(ctx).
		Where("user_id = ?", trimmedUserID).
		Order("recorded_at ASC").
		Find(&entries)
	if queryResult.Error != nil {
		return nil, queryResult.Error
	}

	return entries, nil
}
