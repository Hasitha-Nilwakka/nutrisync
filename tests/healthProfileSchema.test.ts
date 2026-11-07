import { describe, expect, it } from "vitest"
import {
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
})
