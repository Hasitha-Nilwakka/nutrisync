import fs from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"
import {
  computeBmiMarkerPercent,
  computeWeightGoalProgress,
  computeWeightTrendPoints,
  summarizeActivityBreakdown,
} from "../lib/dashboardChartHelpers"

const repoRoot = path.resolve(__dirname, "..")

describe("Dashboard charts and user health infographic", () => {
  it("sorts weight entries chronologically and converts to lbs when preferred", () => {
    const points = computeWeightTrendPoints(
      [
        {
          id: "w2",
          userId: "u1",
          weightKg: 70,
          rawValue: 70,
          rawUnit: "kg",
          recordedAt: "2026-11-10T10:00:00Z",
          createdAt: "2026-11-10T10:00:00Z",
        },
        {
          id: "w1",
          userId: "u1",
          weightKg: 72,
          rawValue: 72,
          rawUnit: "kg",
          recordedAt: "2026-11-01T10:00:00Z",
          createdAt: "2026-11-01T10:00:00Z",
        },
      ],
      "lbs"
    )

    expect(points).toHaveLength(2)
    expect(points[0].id).toBe("w1")
    expect(points[0].displayWeight).toBeCloseTo(158.7, 1)
    expect(points[1].id).toBe("w2")
    expect(points[1].displayWeight).toBeCloseTo(154.3, 1)
  })

  it("summarizes activity duration and intensity breakdown", () => {
    const summary = summarizeActivityBreakdown([
      {
        id: "a1",
        userId: "u1",
        exerciseType: "Running",
        durationMinutes: 30,
        intensity: "high",
        notes: "",
        recordedAt: "2026-11-09T08:00:00Z",
        createdAt: "2026-11-09T08:00:00Z",
      },
      {
        id: "a2",
        userId: "u1",
        exerciseType: "Walking",
        durationMinutes: 45,
        intensity: "low",
        notes: "",
        recordedAt: "2026-11-10T08:00:00Z",
        createdAt: "2026-11-10T08:00:00Z",
      },
    ])

    expect(summary.totalMinutes).toBe(75)
    expect(summary.averageMinutes).toBe(38)
    expect(summary.sessionCount).toBe(2)
    expect(summary.intensityMinutes.high).toBe(30)
    expect(summary.intensityMinutes.low).toBe(45)
  })

  it("computes clamped BMI marker and weight goal progress percentages", () => {
    expect(computeBmiMarkerPercent(10)).toBe(0)
    expect(computeBmiMarkerPercent(50)).toBe(100)
    expect(computeBmiMarkerPercent(22)).toBeGreaterThan(20)
    expect(computeBmiMarkerPercent(22)).toBeLessThan(50)

    expect(computeWeightGoalProgress(80, 75, 70)).toBe(50)
    expect(computeWeightGoalProgress(80, 70, 70)).toBe(100)
    expect(computeWeightGoalProgress(80, 82, 70)).toBe(0)
  })

  it("wires Section 04 Health Infographic at the bottom of HealthDashboardClient", () => {
    const dashboardSource = fs.readFileSync(
      path.join(repoRoot, "feature/dashboard/HealthDashboardClient.tsx"),
      "utf8"
    )
    expect(dashboardSource).toContain('href="#infographic"')
    expect(dashboardSource).toContain('id="infographic"')
    expect(dashboardSource).toContain("04 · Health Infographic")
    expect(dashboardSource).toContain("<UserHealthInfographicSection")
  })
})
