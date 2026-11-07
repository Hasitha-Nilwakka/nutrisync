// Zod validation schema and TypeScript types for the 4-Step User Health Profile Onboarding Wizard,
// matching SaveHealthProfileRequestPayload in services/go-api/handlers/health_profile_handler.go.
import { z } from "zod"

export const GENDER_OPTIONS = ["male", "female", "other", "prefer_not_to_say"] as const
export const HEIGHT_UNIT_OPTIONS = ["cm", "in"] as const
export const WEIGHT_UNIT_OPTIONS = ["kg", "lbs"] as const
export const OCCUPATION_TYPE_OPTIONS = [
  "sedentary_desk",
  "lightly_active",
  "moderately_active",
  "physically_demanding",
] as const
export const ACTIVITY_LEVEL_OPTIONS = [
  "sedentary",
  "light",
  "moderate",
  "active",
  "very_active",
] as const
export const PRIMARY_GOAL_OPTIONS = [
  "weight_loss",
  "muscle_gain",
  "general_fitness",
  "endurance",
  "flexibility",
] as const
export const TARGET_ACTIVITY_LEVEL_OPTIONS = [
  "light",
  "moderate",
  "active",
  "very_active",
] as const
export const EXERCISE_TYPE_OPTIONS = ["cardio", "strength", "flexibility", "sports"] as const
export const SESSION_DURATION_OPTIONS = ["15_30_min", "30_60_min", "60_plus_min"] as const
export const FITNESS_LEVEL_OPTIONS = ["beginner", "intermediate", "advanced"] as const
export const PREFERRED_ENVIRONMENT_OPTIONS = ["home", "gym", "outdoors", "mixed"] as const
export const PREFERRED_TIME_OF_DAY_OPTIONS = ["morning", "afternoon", "evening", "flexible"] as const

export const DIETARY_PREFERENCE_OPTIONS = [
  "balanced",
  "vegetarian",
  "vegan",
  "pescatarian",
  "keto",
  "paleo",
  "mediterranean",
  "high_protein",
  "low_carb",
  "intermittent_fasting",
] as const

export const DIETARY_RESTRICTION_OPTIONS = [
  "gluten_free",
  "dairy_free",
  "nut_allergy",
  "shellfish_allergy",
  "soy_free",
  "egg_free",
  "halal",
  "kosher",
  "low_sodium",
  "low_sugar",
] as const

// Converts height in "cm" or "in" to centimeters rounded to 2 decimal places.
export function convertHeightToCm(height: number, unit: "cm" | "in"): number {
  const cm = unit === "in" ? height * 2.54 : height
  return Math.round(cm * 100) / 100
}

// Converts weight in "kg" or "lbs" to kilograms rounded to 2 decimal places.
export function convertWeightToKg(weight: number, unit: "kg" | "lbs"): number {
  const kg = unit === "lbs" ? weight * 0.45359237 : weight
  return Math.round(kg * 100) / 100
}

// Calculates BMI and returns { bmi, classification } or null if inputs are non-positive.
export function calculateClientBMI(
  weight: number,
  weightUnit: "kg" | "lbs",
  height: number,
  heightUnit: "cm" | "in"
): { bmi: number; classification: string } | null {
  if (!Number.isFinite(weight) || !Number.isFinite(height) || weight <= 0 || height <= 0) {
    return null
  }

  const weightKg = convertWeightToKg(weight, weightUnit)
  const heightCm = convertHeightToCm(height, heightUnit)
  if (heightCm <= 0) {
    return null
  }

  const heightMeters = heightCm / 100
  const bmi = Math.round((weightKg / (heightMeters * heightMeters)) * 100) / 100

  let classification = "obese"
  if (bmi < 18.5) {
    classification = "underweight"
  } else if (bmi < 25) {
    classification = "normal weight"
  } else if (bmi < 30) {
    classification = "overweight"
  }

  return { bmi, classification }
}

export const healthProfileSchema = z
  .object({
    // Step 1: Demographics & Physical Metrics
    age: z.number().int().min(13, "Age must be at least 13").max(120, "Age must be at most 120"),
    gender: z.enum(GENDER_OPTIONS),
    heightUnit: z.enum(HEIGHT_UNIT_OPTIONS),
    height: z.number().positive("Height must be positive"),
    weightUnit: z.enum(WEIGHT_UNIT_OPTIONS),
    currentWeight: z.number().positive("Current weight must be positive"),
    targetWeight: z.number().positive("Target weight must be positive"),

    // Step 2: Lifestyle & Goals
    occupationType: z.enum(OCCUPATION_TYPE_OPTIONS),
    activityLevel: z.enum(ACTIVITY_LEVEL_OPTIONS),
    primaryGoal: z.enum(PRIMARY_GOAL_OPTIONS),
    targetActivityLevel: z.enum(TARGET_ACTIVITY_LEVEL_OPTIONS),
    targetWeeklyDays: z.number().int().min(0, "Minimum 0 days").max(7, "Maximum 7 days"),

    // Step 3: Initial Fitness Assessment
    weeklyActivityDays: z.number().int().min(0, "Minimum 0 days").max(7, "Maximum 7 days"),
    exerciseTypes: z.array(z.string()).min(1, "Select at least one exercise type"),
    sessionDuration: z.enum(SESSION_DURATION_OPTIONS),
    fitnessLevel: z.enum(FITNESS_LEVEL_OPTIONS),
    preferredEnvironment: z.enum(PREFERRED_ENVIRONMENT_OPTIONS),
    preferredTimeOfDay: z.enum(PREFERRED_TIME_OF_DAY_OPTIONS),
    enduranceMinutes: z.number().int().min(0, "Minimum 0 minutes").max(600, "Maximum 600 minutes"),
    pushupsCount: z.number().int().min(0, "Minimum 0 reps").max(500, "Maximum 500 reps"),
    squatsCount: z.number().int().min(0, "Minimum 0 reps").max(500, "Maximum 500 reps"),

    // Step 4: Nutrition, Habits & Privacy
    dietaryPreferences: z.array(z.string()),
    dietaryRestrictions: z.array(z.string()),
    medicalConditions: z.string().optional(),
    sleepHours: z.number().min(1, "Minimum 1 hour").max(24, "Maximum 24 hours"),
    waterLiters: z.number().min(0.1, "Minimum 0.1 L").max(15, "Maximum 15 L"),
    allowAiProcessing: z.boolean(),
    publicProfileVisible: z.boolean(),
    emailNotifications: z.boolean(),
  })
  .superRefine((data, ctx) => {
    const heightCm = convertHeightToCm(data.height, data.heightUnit)
    if (heightCm < 80 || heightCm > 260) {
      ctx.addIssue({
        code: "custom",
        path: ["height"],
        message: "Height must be between 80 cm (31.5 in) and 260 cm (102.4 in)",
      })
    }

    const currentWeightKg = convertWeightToKg(data.currentWeight, data.weightUnit)
    if (currentWeightKg < 20 || currentWeightKg > 350) {
      ctx.addIssue({
        code: "custom",
        path: ["currentWeight"],
        message: "Current weight must be between 20 kg (44.1 lbs) and 350 kg (771.6 lbs)",
      })
    }

    const targetWeightKg = convertWeightToKg(data.targetWeight, data.weightUnit)
    if (targetWeightKg < 20 || targetWeightKg > 350) {
      ctx.addIssue({
        code: "custom",
        path: ["targetWeight"],
        message: "Target weight must be between 20 kg (44.1 lbs) and 350 kg (771.6 lbs)",
      })
    }

    const bmiDetails = calculateClientBMI(
      data.currentWeight,
      data.weightUnit,
      data.height,
      data.heightUnit
    )
    if (!bmiDetails || bmiDetails.bmi < 10 || bmiDetails.bmi > 80) {
      ctx.addIssue({
        code: "custom",
        path: ["currentWeight"],
        message: "Calculated BMI is outside the realistic human range (10.0 - 80.0)",
      })
    }
  })

export type HealthProfileFormData = z.infer<typeof healthProfileSchema>
