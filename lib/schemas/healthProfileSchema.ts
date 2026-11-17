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
    age: z
      .number({ error: "Please enter your age" })
      .int("Age must be a whole number")
      .min(13, "Age must be at least 13")
      .max(120, "Age must be at most 120"),
    gender: z.enum(GENDER_OPTIONS, { error: "Please select a gender option" }),
    heightUnit: z.enum(HEIGHT_UNIT_OPTIONS, { error: "Please select a height unit" }),
    height: z
      .number({ error: "Please enter your height" })
      .positive("Height must be positive"),
    weightUnit: z.enum(WEIGHT_UNIT_OPTIONS, { error: "Please select a weight unit" }),
    currentWeight: z
      .number({ error: "Please enter your current weight" })
      .positive("Current weight must be positive"),
    targetWeight: z
      .number({ error: "Please enter your target weight" })
      .positive("Target weight must be positive"),

    // Step 2: Lifestyle & Goals
    occupationType: z.enum(OCCUPATION_TYPE_OPTIONS, {
      error: "Please select your occupation activity type",
    }),
    activityLevel: z.enum(ACTIVITY_LEVEL_OPTIONS, {
      error: "Please select your current activity level",
    }),
    primaryGoal: z.enum(PRIMARY_GOAL_OPTIONS, {
      error: "Please select your primary wellness goal",
    }),
    targetActivityLevel: z.enum(TARGET_ACTIVITY_LEVEL_OPTIONS, {
      error: "Please select your target activity level",
    }),
    targetWeeklyDays: z
      .number({ error: "Please enter target weekly workout days" })
      .int("Target workout days must be a whole number")
      .min(0, "Minimum 0 days")
      .max(7, "Maximum 7 days"),

    // Step 3: Initial Fitness Assessment
    weeklyActivityDays: z
      .number({ error: "Please enter your current weekly active days" })
      .int("Current active days must be a whole number")
      .min(0, "Minimum 0 days")
      .max(7, "Maximum 7 days"),
    exerciseTypes: z.array(z.string()).min(1, "Select at least one exercise type"),
    sessionDuration: z.enum(SESSION_DURATION_OPTIONS, {
      error: "Please select average session duration",
    }),
    fitnessLevel: z.enum(FITNESS_LEVEL_OPTIONS, {
      error: "Please select your fitness level",
    }),
    preferredEnvironment: z.enum(PREFERRED_ENVIRONMENT_OPTIONS, {
      error: "Please select your preferred workout environment",
    }),
    preferredTimeOfDay: z.enum(PREFERRED_TIME_OF_DAY_OPTIONS, {
      error: "Please select your preferred workout time",
    }),
    enduranceMinutes: z
      .number({ error: "Please enter continuous endurance minutes (0 if none)" })
      .int("Endurance minutes must be a whole number")
      .min(0, "Minimum 0 minutes")
      .max(600, "Maximum 600 minutes"),
    pushupsCount: z
      .number({ error: "Please enter max push-up reps (0 if none)" })
      .int("Push-up reps must be a whole number")
      .min(0, "Minimum 0 reps")
      .max(500, "Maximum 500 reps"),
    squatsCount: z
      .number({ error: "Please enter max squat reps (0 if none)" })
      .int("Squat reps must be a whole number")
      .min(0, "Minimum 0 reps")
      .max(500, "Maximum 500 reps"),

    // Step 4: Nutrition, Habits & Privacy
    dietaryPreferences: z.array(z.string()),
    dietaryRestrictions: z.array(z.string()),
    medicalConditions: z.string().optional(),
    sleepHours: z
      .number({ error: "Please enter average daily sleep hours" })
      .min(1, "Minimum 1 hour")
      .max(24, "Maximum 24 hours"),
    waterLiters: z
      .number({ error: "Please enter daily water intake in liters" })
      .min(0.1, "Minimum 0.1 L")
      .max(15, "Maximum 15 L"),
    allowAiProcessing: z.boolean(),
    publicProfileVisible: z.boolean(),
    emailNotifications: z.boolean(),
  })
  .superRefine((data, ctx) => {
    if (typeof data.height === "number" && Number.isFinite(data.height)) {
      const heightCm = convertHeightToCm(data.height, data.heightUnit)
      if (heightCm < 80 || heightCm > 260) {
        ctx.addIssue({
          code: "custom",
          path: ["height"],
          message: "Height must be between 80 cm (31.5 in) and 260 cm (102.4 in)",
        })
      }
    }

    if (typeof data.currentWeight === "number" && Number.isFinite(data.currentWeight)) {
      const currentWeightKg = convertWeightToKg(data.currentWeight, data.weightUnit)
      if (currentWeightKg < 20 || currentWeightKg > 350) {
        ctx.addIssue({
          code: "custom",
          path: ["currentWeight"],
          message: "Current weight must be between 20 kg (44.1 lbs) and 350 kg (771.6 lbs)",
        })
      }
    }

    if (typeof data.targetWeight === "number" && Number.isFinite(data.targetWeight)) {
      const targetWeightKg = convertWeightToKg(data.targetWeight, data.weightUnit)
      if (targetWeightKg < 20 || targetWeightKg > 350) {
        ctx.addIssue({
          code: "custom",
          path: ["targetWeight"],
          message: "Target weight must be between 20 kg (44.1 lbs) and 350 kg (771.6 lbs)",
        })
      }
    }

    if (
      typeof data.currentWeight === "number" &&
      Number.isFinite(data.currentWeight) &&
      typeof data.height === "number" &&
      Number.isFinite(data.height)
    ) {
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
    }
  })

export type HealthProfileFormData = z.infer<typeof healthProfileSchema>

// Canonical sample profile data for quick testing and portfolio showcase exploration.
export const SAMPLE_HEALTH_PROFILE_FORM_DATA: HealthProfileFormData = {
  age: 28,
  gender: "prefer_not_to_say",
  heightUnit: "cm",
  height: 170,
  weightUnit: "kg",
  currentWeight: 70,
  targetWeight: 68,
  occupationType: "sedentary_desk",
  activityLevel: "moderate",
  primaryGoal: "general_fitness",
  targetActivityLevel: "active",
  targetWeeklyDays: 4,
  weeklyActivityDays: 3,
  exerciseTypes: ["cardio"],
  sessionDuration: "30_60_min",
  fitnessLevel: "intermediate",
  preferredEnvironment: "mixed",
  preferredTimeOfDay: "flexible",
  enduranceMinutes: 25,
  pushupsCount: 15,
  squatsCount: 25,
  dietaryPreferences: ["balanced"],
  dietaryRestrictions: [],
  medicalConditions: "",
  sleepHours: 7.5,
  waterLiters: 2.0,
  allowAiProcessing: true,
  publicProfileVisible: false,
  emailNotifications: true,
}

// Clean initial values when a user opts out of sample data and wants to fill in their own metrics.
export const EMPTY_HEALTH_PROFILE_FORM_DATA: HealthProfileFormData = {
  age: undefined as unknown as number,
  gender: "prefer_not_to_say",
  heightUnit: "cm",
  height: undefined as unknown as number,
  weightUnit: "kg",
  currentWeight: undefined as unknown as number,
  targetWeight: undefined as unknown as number,
  occupationType: "sedentary_desk",
  activityLevel: "moderate",
  primaryGoal: "general_fitness",
  targetActivityLevel: "active",
  targetWeeklyDays: undefined as unknown as number,
  weeklyActivityDays: undefined as unknown as number,
  exerciseTypes: [],
  sessionDuration: "30_60_min",
  fitnessLevel: "beginner",
  preferredEnvironment: "mixed",
  preferredTimeOfDay: "flexible",
  enduranceMinutes: undefined as unknown as number,
  pushupsCount: undefined as unknown as number,
  squatsCount: undefined as unknown as number,
  dietaryPreferences: [],
  dietaryRestrictions: [],
  medicalConditions: "",
  sleepHours: undefined as unknown as number,
  waterLiters: undefined as unknown as number,
  allowAiProcessing: true,
  publicProfileVisible: false,
  emailNotifications: false,
}
