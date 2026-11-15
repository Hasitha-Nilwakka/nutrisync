// Zod v4 validation schemas, TypeScript payload contracts, and safe JSON parsers
// for the Interactive Health & AI Dashboard, matching services/go-api/handlers/health_profile_handler.go.
import { z } from "zod"
import { convertWeightToKg, WEIGHT_UNIT_OPTIONS } from "./healthProfileSchema"

export const ACTIVITY_INTENSITY_OPTIONS = ["low", "moderate", "high", "vigorous"] as const
export type ActivityIntensity = (typeof ACTIVITY_INTENSITY_OPTIONS)[number]

export const ACTIVITY_INTENSITY_LABELS: Record<ActivityIntensity, string> = {
  low: "Low",
  moderate: "Moderate",
  high: "High",
  vigorous: "Vigorous",
}

export const MIN_WEIGHT_KG = 20
export const MAX_WEIGHT_KG = 350

export const weightLogSchema = z
  .object({
    weight: z
      .number({ error: "Weight must be a valid number" })
      .positive("Weight must be greater than 0"),
    unit: z.enum(WEIGHT_UNIT_OPTIONS, { error: "Unit must be kg or lbs" }),
    recordedAt: z.string().trim().optional(),
  })
  .superRefine((data, ctx) => {
    const normalizedKg = convertWeightToKg(data.weight, data.unit)
    if (normalizedKg < MIN_WEIGHT_KG || normalizedKg > MAX_WEIGHT_KG) {
      ctx.addIssue({
        code: "custom",
        message:
          data.unit === "lbs"
            ? "Weight must be between 44.1 lbs and 771.6 lbs (20–350 kg)"
            : "Weight must be between 20 kg and 350 kg",
        path: ["weight"],
      })
    }

    if (data.recordedAt && data.recordedAt.length > 0 && !Number.isFinite(Date.parse(data.recordedAt))) {
      ctx.addIssue({
        code: "custom",
        message: "Recorded date & time must be a valid timestamp",
        path: ["recordedAt"],
      })
    }
  })

export const activityLogSchema = z.object({
  exerciseType: z
    .string()
    .trim()
    .min(1, "Exercise type is required")
    .max(64, "Exercise type must be at most 64 characters"),
  durationMinutes: z
    .number({ error: "Duration must be a valid number" })
    .int("Duration must be a whole number of minutes")
    .min(1, "Duration must be at least 1 minute")
    .max(1440, "Duration cannot exceed 1440 minutes (24 hours)"),
  intensity: z.enum(ACTIVITY_INTENSITY_OPTIONS, { error: "Select a valid intensity level" }),
  notes: z.string().trim().max(500, "Notes must be at most 500 characters").optional(),
  recordedAt: z
    .string()
    .trim()
    .min(1, "Recorded date & time is required")
    .refine((value) => Number.isFinite(Date.parse(value)), "Recorded date & time must be a valid timestamp"),
})

export type WeightLogFormValues = z.infer<typeof weightLogSchema>
export type ActivityLogFormValues = z.infer<typeof activityLogSchema>

export interface HealthProfilePayload {
  id: string
  userId: string
  age: number
  gender: string
  heightCm: number
  currentWeightKg: number
  baselineWeightKg: number
  targetWeightKg: number
  preferredWeightUnit: "kg" | "lbs" | string
  preferredHeightUnit: "cm" | "in" | string
  bmi: number
  bmiClassification: string
  occupationType: string
  activityLevel: string
  primaryGoal: string
  targetActivityLevel: string
  targetWeeklyDays: number
  weeklyActivityDays: number
  exerciseTypes: string[]
  sessionDuration: string
  fitnessLevel: string
  preferredEnvironment: string
  preferredTimeOfDay: string
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

export interface MilestoneItem {
  id: string
  category: string
  title: string
  description: string
  achieved: boolean
  progressPct: number
}

export interface WellnessSnapshotPayload {
  id: string
  userId: string
  bmi: number
  bmiClassification: string
  wellnessScore: number
  bmiScore: number
  activityScore: number
  progressScore: number
  habitsScore: number
  milestonesJson: string
  recordedAt: string
  createdAt: string
}

export interface AIRecommendationItem {
  title: string
  category: string
  priority: string
  action: string
  expectedImpact: string
}

export interface AIInsightRecordPayload {
  id: string
  userId: string
  version: number
  primaryGoalSnapshot: string
  promptStrategy: string
  healthStatusAnalysis: string
  progressEvaluation: string
  recommendationsJson: string
  weeklySummary: string
  monthlySummary: string
  createdAt: string
}

export interface WeightEntryPayload {
  id: string
  userId: string
  weightKg: number
  rawValue: number
  rawUnit: string
  recordedAt: string
  createdAt: string
}

export interface ActivityEntryPayload {
  id: string
  userId: string
  exerciseType: string
  durationMinutes: number
  intensity: string
  notes?: string
  recordedAt: string
  createdAt: string
}

export interface HealthDashboardPayload {
  healthProfile: HealthProfilePayload | null
  latestWellness: WellnessSnapshotPayload | null
  wellnessHistory: WellnessSnapshotPayload[]
  weightHistory: WeightEntryPayload[]
  activityHistory: ActivityEntryPayload[]
  latestInsights: AIInsightRecordPayload | null
  insightHistory: AIInsightRecordPayload[]
  aiServiceAvailable: boolean
  isCachedFallback: boolean
}

export interface RegenerateAIInsightsPayload {
  latestInsights: AIInsightRecordPayload | null
  insightHistory: AIInsightRecordPayload[]
  aiServiceAvailable: boolean
  isCachedFallback: boolean
}

export interface GDPRHealthDataExportPayload {
  user: {
    id: string
    email: string
    consented: boolean
    exportedAt: string
  }
  healthProfile: HealthProfilePayload | null
  weightEntries: WeightEntryPayload[]
  activityEntries: ActivityEntryPayload[]
}

// Safely parses WellnessSnapshot.milestonesJson into typed MilestoneItem records.
export function parseMilestonesJson(rawJson: string | null | undefined): MilestoneItem[] {
  if (!rawJson || rawJson.trim().length === 0) {
    return []
  }

  try {
    const parsed: unknown = JSON.parse(rawJson)
    if (!Array.isArray(parsed)) {
      return []
    }

    return parsed
      .filter((item): item is Record<string, unknown> => typeof item === "object" && item !== null)
      .map((item) => ({
        id: typeof item.id === "string" ? item.id : "",
        category: typeof item.category === "string" ? item.category : "general",
        title: typeof item.title === "string" ? item.title : "Milestone",
        description: typeof item.description === "string" ? item.description : "",
        achieved: Boolean(item.achieved),
        progressPct:
          typeof item.progressPct === "number" && Number.isFinite(item.progressPct)
            ? Math.max(0, Math.min(100, item.progressPct))
            : 0,
      }))
  } catch {
    return []
  }
}

// Safely parses AIInsightRecord.recommendationsJson into typed AIRecommendationItem records.
export function parseRecommendationsJson(rawJson: string | null | undefined): AIRecommendationItem[] {
  if (!rawJson || rawJson.trim().length === 0) {
    return []
  }

  try {
    const parsed: unknown = JSON.parse(rawJson)
    if (!Array.isArray(parsed)) {
      return []
    }

    return parsed
      .filter((item): item is Record<string, unknown> => typeof item === "object" && item !== null)
      .map((item) => ({
        title: typeof item.title === "string" ? item.title : "Recommendation",
        category: typeof item.category === "string" ? item.category : "general",
        priority: typeof item.priority === "string" ? item.priority : "medium",
        action: typeof item.action === "string" ? item.action : "",
        expectedImpact: typeof item.expectedImpact === "string" ? item.expectedImpact : "",
      }))
  } catch {
    return []
  }
}

// Formats a weight in kilograms into the user's preferred unit ("kg" or "lbs").
export function formatWeightForUnit(weightKg: number, preferredUnit: string): string {
  if (!Number.isFinite(weightKg)) {
    return "—"
  }
  if (preferredUnit.toLowerCase() === "lbs") {
    const lbs = Math.round(weightKg * 2.20462262 * 10) / 10
    return `${lbs} lbs`
  }
  const kg = Math.round(weightKg * 10) / 10
  return `${kg} kg`
}
