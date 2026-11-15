// Pure mapper utilities converting normalized GET /api/health/profile responses
// into HealthProfileFormData values for PUT /api/health/profile editing.
import {
  ACTIVITY_LEVEL_OPTIONS,
  FITNESS_LEVEL_OPTIONS,
  GENDER_OPTIONS,
  HEIGHT_UNIT_OPTIONS,
  OCCUPATION_TYPE_OPTIONS,
  PREFERRED_ENVIRONMENT_OPTIONS,
  PREFERRED_TIME_OF_DAY_OPTIONS,
  PRIMARY_GOAL_OPTIONS,
  SESSION_DURATION_OPTIONS,
  TARGET_ACTIVITY_LEVEL_OPTIONS,
  WEIGHT_UNIT_OPTIONS,
  type HealthProfileFormData,
} from "@/lib/schemas/healthProfileSchema"

export interface HealthProfileResponse {
  id: string
  userId: string
  age: number
  gender: (typeof GENDER_OPTIONS)[number]
  heightCm: number
  currentWeightKg: number
  baselineWeightKg: number
  targetWeightKg: number
  preferredWeightUnit: (typeof WEIGHT_UNIT_OPTIONS)[number]
  preferredHeightUnit: (typeof HEIGHT_UNIT_OPTIONS)[number]
  bmi: number
  bmiClassification: string
  occupationType: (typeof OCCUPATION_TYPE_OPTIONS)[number]
  activityLevel: (typeof ACTIVITY_LEVEL_OPTIONS)[number]
  primaryGoal: (typeof PRIMARY_GOAL_OPTIONS)[number]
  targetActivityLevel: (typeof TARGET_ACTIVITY_LEVEL_OPTIONS)[number]
  targetWeeklyDays: number
  weeklyActivityDays: number
  exerciseTypes: string[]
  sessionDuration: (typeof SESSION_DURATION_OPTIONS)[number]
  fitnessLevel: (typeof FITNESS_LEVEL_OPTIONS)[number]
  preferredEnvironment: (typeof PREFERRED_ENVIRONMENT_OPTIONS)[number]
  preferredTimeOfDay: (typeof PREFERRED_TIME_OF_DAY_OPTIONS)[number]
  enduranceMinutes: number
  pushupsCount: number
  squatsCount: number
  dietaryPreferences: string[]
  dietaryRestrictions: string[]
  medicalConditions: string
  sleepHours: number
  waterLiters: number
  allowAiProcessing: boolean
  publicProfileVisible: boolean
  emailNotifications: boolean
  createdAt: string
  updatedAt: string
}

function roundToTwoDecimals(value: number): number {
  return Math.round(value * 100) / 100
}

// Converts normalized metric values (cm/kg) from GET /api/health/profile into the user's preferred units.
export function mapHealthProfileResponseToFormData(
  profile: HealthProfileResponse
): HealthProfileFormData {
  const heightUnit = profile.preferredHeightUnit === "in" ? "in" : "cm"
  const weightUnit = profile.preferredWeightUnit === "lbs" ? "lbs" : "kg"

  const height =
    heightUnit === "in"
      ? roundToTwoDecimals(profile.heightCm / 2.54)
      : roundToTwoDecimals(profile.heightCm)

  const currentWeight =
    weightUnit === "lbs"
      ? roundToTwoDecimals(profile.currentWeightKg / 0.45359237)
      : roundToTwoDecimals(profile.currentWeightKg)

  const targetWeight =
    weightUnit === "lbs"
      ? roundToTwoDecimals(profile.targetWeightKg / 0.45359237)
      : roundToTwoDecimals(profile.targetWeightKg)

  return {
    age: profile.age,
    gender: profile.gender,
    heightUnit,
    height,
    weightUnit,
    currentWeight,
    targetWeight,
    occupationType: profile.occupationType,
    activityLevel: profile.activityLevel,
    primaryGoal: profile.primaryGoal,
    targetActivityLevel: profile.targetActivityLevel,
    targetWeeklyDays: profile.targetWeeklyDays,
    weeklyActivityDays: profile.weeklyActivityDays,
    exerciseTypes: Array.isArray(profile.exerciseTypes) ? profile.exerciseTypes : [],
    sessionDuration: profile.sessionDuration,
    fitnessLevel: profile.fitnessLevel,
    preferredEnvironment: profile.preferredEnvironment,
    preferredTimeOfDay: profile.preferredTimeOfDay,
    enduranceMinutes: profile.enduranceMinutes,
    pushupsCount: profile.pushupsCount,
    squatsCount: profile.squatsCount,
    dietaryPreferences: Array.isArray(profile.dietaryPreferences)
      ? profile.dietaryPreferences
      : [],
    dietaryRestrictions: Array.isArray(profile.dietaryRestrictions)
      ? profile.dietaryRestrictions
      : [],
    medicalConditions: profile.medicalConditions ?? "",
    sleepHours: profile.sleepHours,
    waterLiters: profile.waterLiters,
    allowAiProcessing: Boolean(profile.allowAiProcessing),
    publicProfileVisible: Boolean(profile.publicProfileVisible),
    emailNotifications: Boolean(profile.emailNotifications),
  }
}
