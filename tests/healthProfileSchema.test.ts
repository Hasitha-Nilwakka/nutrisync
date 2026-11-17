import { describe, expect, it } from "vitest"
import {
  EMPTY_HEALTH_PROFILE_FORM_DATA,
  SAMPLE_HEALTH_PROFILE_FORM_DATA,
  calculateClientBMI,
  convertHeightToCm,
  convertWeightToKg,
  healthProfileSchema,
} from "../lib/schemas/healthProfileSchema"

const validMetricProfile = {
  age: 30,
  gender: "male" as const,
  heightUnit: "cm" as const,
  height: 180,
  weightUnit: "kg" as const,
  currentWeight: 75,
  targetWeight: 72,
  occupationType: "sedentary_desk" as const,
  activityLevel: "moderate" as const,
  primaryGoal: "general_fitness" as const,
  targetActivityLevel: "active" as const,
  targetWeeklyDays: 4,
  weeklyActivityDays: 3,
  exerciseTypes: ["cardio", "strength"],
  sessionDuration: "30_60_min" as const,
  fitnessLevel: "intermediate" as const,
  preferredEnvironment: "gym" as const,
  preferredTimeOfDay: "morning" as const,
  enduranceMinutes: 30,
  pushupsCount: 20,
  squatsCount: 35,
  dietaryPreferences: ["high_protein"],
  dietaryRestrictions: ["dairy_free"],
  medicalConditions: "",
  sleepHours: 7.5,
  waterLiters: 2.5,
  allowAiProcessing: true,
  publicProfileVisible: false,
  emailNotifications: true,
}

describe("Health profile schema and unit conversion helpers", () => {
  it("Accepts a valid metric health profile", () => {
    const result = healthProfileSchema.safeParse(validMetricProfile)
    expect(result.success).toBe(true)
  })

  it("Accepts a valid imperial health profile and converts in->cm and lbs->kg", () => {
    expect(convertHeightToCm(70, "in")).toBe(177.8)
    expect(convertWeightToKg(154.32, "lbs")).toBe(70)

    const imperialProfile = {
      ...validMetricProfile,
      heightUnit: "in" as const,
      height: 70,
      weightUnit: "lbs" as const,
      currentWeight: 154.32,
      targetWeight: 145,
    }

    const result = healthProfileSchema.safeParse(imperialProfile)
    expect(result.success).toBe(true)
  })

  it("Rejects negative or out-of-range weights and unrealistic BMI values", () => {
    const negativeWeight = healthProfileSchema.safeParse({
      ...validMetricProfile,
      currentWeight: -60,
    })
    expect(negativeWeight.success).toBe(false)

    const unrealisticBMI = healthProfileSchema.safeParse({
      ...validMetricProfile,
      height: 220,
      currentWeight: 22,
    })
    expect(unrealisticBMI.success).toBe(false)
  })

  it("Calculates live client BMI preview accurately", () => {
    const bmiPreview = calculateClientBMI(72, "kg", 180, "cm")
    expect(bmiPreview).toEqual({
      bmi: 22.22,
      classification: "normal weight",
    })
  })

  it("Accepts SAMPLE_HEALTH_PROFILE_FORM_DATA preset as a valid onboarding profile", () => {
    const result = healthProfileSchema.safeParse(SAMPLE_HEALTH_PROFILE_FORM_DATA)
    expect(result.success).toBe(true)
  })

  it("Rejects EMPTY_HEALTH_PROFILE_FORM_DATA when unpopulated, requiring user input", () => {
    const result = healthProfileSchema.safeParse(EMPTY_HEALTH_PROFILE_FORM_DATA)
    expect(result.success).toBe(false)
    if (!result.success) {
      const fieldIssues = result.error.issues.map((issue) => issue.path[0])
      expect(fieldIssues).toContain("age")
      expect(fieldIssues).toContain("height")
      expect(fieldIssues).toContain("currentWeight")
      expect(fieldIssues).toContain("exerciseTypes")
    }
  })

  it("Provides human-readable inline error messages without NaN or expected number artifacts", () => {
    const nanProfile = {
      ...EMPTY_HEALTH_PROFILE_FORM_DATA,
      age: NaN,
      height: NaN,
      currentWeight: NaN,
      targetWeight: NaN,
      targetWeeklyDays: NaN,
      weeklyActivityDays: NaN,
      enduranceMinutes: NaN,
      pushupsCount: NaN,
      squatsCount: NaN,
      sleepHours: NaN,
      waterLiters: NaN,
    }

    const result = healthProfileSchema.safeParse(nanProfile)
    expect(result.success).toBe(false)
    if (!result.success) {
      const messagesByPath = Object.fromEntries(
        result.error.issues.map((issue) => [issue.path[0], issue.message])
      )

      expect(messagesByPath.age).toBe("Please enter your age")
      expect(messagesByPath.height).toBe("Please enter your height")
      expect(messagesByPath.currentWeight).toBe("Please enter your current weight")
      expect(messagesByPath.targetWeight).toBe("Please enter your target weight")
      expect(messagesByPath.targetWeeklyDays).toBe("Please enter target weekly workout days")
      expect(messagesByPath.weeklyActivityDays).toBe("Please enter your current weekly active days")
      expect(messagesByPath.enduranceMinutes).toBe("Please enter continuous endurance minutes (0 if none)")
      expect(messagesByPath.pushupsCount).toBe("Please enter max push-up reps (0 if none)")
      expect(messagesByPath.squatsCount).toBe("Please enter max squat reps (0 if none)")
      expect(messagesByPath.sleepHours).toBe("Please enter average daily sleep hours")
      expect(messagesByPath.waterLiters).toBe("Please enter daily water intake in liters")

      // Ensure no raw schema type error artifacts ("nan", "NaN", "expected number") leak to user UI
      for (const issue of result.error.issues) {
        expect(issue.message.toLowerCase()).not.toContain("nan")
        expect(issue.message.toLowerCase()).not.toContain("expected number")
      }
    }
  })
})
