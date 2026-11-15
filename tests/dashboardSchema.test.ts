// Vitest unit tests for Interactive Health & AI Dashboard Zod v4 schemas and JSON parsers.
import { describe, expect, it } from "vitest"
import {
  activityLogSchema,
  formatWeightForUnit,
  parseMilestonesJson,
  parseRecommendationsJson,
  weightLogSchema,
} from "../lib/schemas/dashboardSchema"

describe("weightLogSchema", () => {
  it("accepts valid weight in kg and optional recordedAt", () => {
    const result = weightLogSchema.safeParse({
      weight: 72.5,
      unit: "kg",
      recordedAt: "2026-11-15T10:30:00Z",
    })
    expect(result.success).toBe(true)
  })

  it("accepts valid weight in lbs within 20–350 kg normalized bounds", () => {
    const result = weightLogSchema.safeParse({
      weight: 160,
      unit: "lbs",
    })
    expect(result.success).toBe(true)
  })

  it("rejects weight below 20 kg normalized bound", () => {
    const kgResult = weightLogSchema.safeParse({ weight: 19.5, unit: "kg" })
    expect(kgResult.success).toBe(false)

    const lbsResult = weightLogSchema.safeParse({ weight: 40, unit: "lbs" })
    expect(lbsResult.success).toBe(false)
  })

  it("rejects weight above 350 kg normalized bound", () => {
    const kgResult = weightLogSchema.safeParse({ weight: 351, unit: "kg" })
    expect(kgResult.success).toBe(false)

    const lbsResult = weightLogSchema.safeParse({ weight: 800, unit: "lbs" })
    expect(lbsResult.success).toBe(false)
  })

  it("rejects non-positive weight or invalid recordedAt timestamp", () => {
    expect(weightLogSchema.safeParse({ weight: 0, unit: "kg" }).success).toBe(false)
    expect(
      weightLogSchema.safeParse({ weight: 70, unit: "kg", recordedAt: "not-a-date" }).success
    ).toBe(false)
  })
})

describe("activityLogSchema", () => {
  it("accepts a valid activity entry", () => {
    const result = activityLogSchema.safeParse({
      exerciseType: "Running",
      durationMinutes: 45,
      intensity: "high",
      notes: "5k tempo run",
      recordedAt: "2026-11-15T08:00",
    })
    expect(result.success).toBe(true)
  })

  it("rejects empty exerciseType or missing recordedAt", () => {
    expect(
      activityLogSchema.safeParse({
        exerciseType: "   ",
        durationMinutes: 30,
        intensity: "moderate",
        recordedAt: "2026-11-15T08:00",
      }).success
    ).toBe(false)

    expect(
      activityLogSchema.safeParse({
        exerciseType: "Cycling",
        durationMinutes: 30,
        intensity: "moderate",
        recordedAt: "",
      }).success
    ).toBe(false)
  })

  it("rejects durationMinutes outside 1–1440 or non-integer", () => {
    expect(
      activityLogSchema.safeParse({
        exerciseType: "Yoga",
        durationMinutes: 0,
        intensity: "low",
        recordedAt: "2026-11-15T08:00",
      }).success
    ).toBe(false)

    expect(
      activityLogSchema.safeParse({
        exerciseType: "Yoga",
        durationMinutes: 1441,
        intensity: "low",
        recordedAt: "2026-11-15T08:00",
      }).success
    ).toBe(false)

    expect(
      activityLogSchema.safeParse({
        exerciseType: "Yoga",
        durationMinutes: 30.5,
        intensity: "low",
        recordedAt: "2026-11-15T08:00",
      }).success
    ).toBe(false)
  })
})

describe("Dashboard JSON parsers and formatters", () => {
  it("parses valid milestonesJson and clamps progressPct to 0..100", () => {
    const raw = JSON.stringify([
      {
        id: "weight-5pct-step",
        category: "weight",
        title: "5% Weight Milestone",
        description: "Halfway to target",
        achieved: false,
        progressPct: 120,
      },
    ])
    const parsed = parseMilestonesJson(raw)
    expect(parsed).toHaveLength(1)
    expect(parsed[0].id).toBe("weight-5pct-step")
    expect(parsed[0].progressPct).toBe(100)
  })

  it("returns empty array on malformed or empty milestonesJson / recommendationsJson", () => {
    expect(parseMilestonesJson("")).toEqual([])
    expect(parseMilestonesJson("{bad-json")).toEqual([])
    expect(parseRecommendationsJson(null)).toEqual([])
    expect(parseRecommendationsJson("not-an-array")).toEqual([])
  })

  it("parses valid recommendationsJson", () => {
    const raw = JSON.stringify([
      {
        title: "Increase Zone 2 Cardio",
        category: "activity",
        priority: "high",
        action: "Add two 30-minute brisk walks",
        expectedImpact: "Improves weekly activity score",
      },
    ])
    const parsed = parseRecommendationsJson(raw)
    expect(parsed).toHaveLength(1)
    expect(parsed[0].priority).toBe("high")
    expect(parsed[0].action).toContain("30-minute")
  })

  it("formats weight in kg or lbs", () => {
    expect(formatWeightForUnit(75.34, "kg")).toBe("75.3 kg")
    expect(formatWeightForUnit(75, "lbs")).toBe("165.3 lbs")
  })
})
