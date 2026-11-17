// Vitest unit tests for Health Profile response-to-form unit conversion mappers.
import fs from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"
import {
  mapHealthProfileResponseToFormData,
  type HealthProfileResponse,
} from "../feature/profile/healthProfileMappers"
import { healthProfileSchema } from "../lib/schemas/healthProfileSchema"

function createSampleResponse(
  overrides: Partial<HealthProfileResponse> = {}
): HealthProfileResponse {
  return {
    id: "hp-1",
    userId: "user-1",
    age: 29,
    gender: "female",
    heightCm: 170,
    currentWeightKg: 68,
    baselineWeightKg: 70,
    targetWeightKg: 62,
    preferredWeightUnit: "kg",
    preferredHeightUnit: "cm",
    bmi: 23.53,
    bmiClassification: "normal weight",
    occupationType: "sedentary_desk",
    activityLevel: "moderate",
    primaryGoal: "general_fitness",
    targetActivityLevel: "active",
    targetWeeklyDays: 4,
    weeklyActivityDays: 3,
    exerciseTypes: ["cardio", "strength"],
    sessionDuration: "30_60_min",
    fitnessLevel: "intermediate",
    preferredEnvironment: "gym",
    preferredTimeOfDay: "morning",
    enduranceMinutes: 30,
    pushupsCount: 15,
    squatsCount: 25,
    dietaryPreferences: ["mediterranean"],
    dietaryRestrictions: ["gluten_free"],
    medicalConditions: "",
    sleepHours: 7.5,
    waterLiters: 2.2,
    allowAiProcessing: true,
    publicProfileVisible: false,
    emailNotifications: true,
    createdAt: "2026-11-01T00:00:00Z",
    updatedAt: "2026-11-10T00:00:00Z",
    ...overrides,
  }
}

describe("mapHealthProfileResponseToFormData", () => {
  it("preserves metric values when preferred units are cm and kg", () => {
    const mapped = mapHealthProfileResponseToFormData(createSampleResponse())

    expect(mapped.heightUnit).toBe("cm")
    expect(mapped.height).toBe(170)
    expect(mapped.weightUnit).toBe("kg")
    expect(mapped.currentWeight).toBe(68)
    expect(mapped.targetWeight).toBe(62)
    expect(mapped.allowAiProcessing).toBe(true)
    expect(mapped.publicProfileVisible).toBe(false)
    expect(mapped.emailNotifications).toBe(true)
    expect(healthProfileSchema.safeParse(mapped).success).toBe(true)
  })

  it("converts normalized cm/kg back to inches and lbs when preferred units are imperial", () => {
    const mapped = mapHealthProfileResponseToFormData(
      createSampleResponse({
        preferredHeightUnit: "in",
        preferredWeightUnit: "lbs",
        heightCm: 177.8,
        currentWeightKg: 72.57,
        targetWeightKg: 68.04,
      })
    )

    expect(mapped.heightUnit).toBe("in")
    expect(mapped.height).toBeCloseTo(70, 1)
    expect(mapped.weightUnit).toBe("lbs")
    expect(mapped.currentWeight).toBeCloseTo(160, 0)
    expect(mapped.targetWeight).toBeCloseTo(150, 0)
    expect(healthProfileSchema.safeParse(mapped).success).toBe(true)
  })
})

describe("Profile GDPR consent gate, brand header, and account deletion", () => {
  const repoRoot = path.resolve(__dirname, "..")

  it("enforces GdprConsentCard gate in app/profile/page.tsx", () => {
    const profilePageSource = fs.readFileSync(
      path.join(repoRoot, "app/profile/page.tsx"),
      "utf8"
    )
    expect(profilePageSource).toContain("if (!session.consented)")
    expect(profilePageSource).toContain("<GdprConsentCard")
  })

  it("configures /api/auth/account rewrite and renders NutriSyncBrand, ThemeToggle, and PrivacyAndAccountDeletionCard", () => {
    const nextConfigSource = fs.readFileSync(
      path.join(repoRoot, "next.config.ts"),
      "utf8"
    )
    expect(nextConfigSource).toContain('source: "/api/auth/account"')

    const profileClientSource = fs.readFileSync(
      path.join(repoRoot, "feature/profile/ProfilePageClient.tsx"),
      "utf8"
    )
    expect(profileClientSource).toContain("<NutriSyncBrand")
    expect(profileClientSource).toContain("<ThemeToggle")
    expect(profileClientSource).toContain("<PrivacyAndAccountDeletionCard")

    const deletionCardSource = fs.readFileSync(
      path.join(repoRoot, "feature/profile/PrivacyAndAccountDeletionCard.tsx"),
      "utf8"
    )
    expect(deletionCardSource).toContain('axios.delete("/api/auth/account"')
    expect(deletionCardSource).toContain('confirmationText.trim() !== "DELETE"')
  })
})
