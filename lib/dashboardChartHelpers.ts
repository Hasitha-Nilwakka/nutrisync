import type {
  ActivityEntryPayload,
  ActivityIntensity,
  WeightEntryPayload,
} from "./schemas/dashboardSchema"

export interface WeightTrendPoint {
  id: string
  recordedAt: string
  weightKg: number
  displayWeight: number
  label: string
}

export interface ActivityBreakdownSummary {
  totalMinutes: number
  averageMinutes: number
  sessionCount: number
  intensityMinutes: Record<ActivityIntensity, number>
}

export function convertKgToUnitValue(weightKg: number, preferredUnit: string): number {
  const factor = preferredUnit.toLowerCase() === "lbs" ? 2.20462262 : 1
  return Math.round(weightKg * factor * 10) / 10
}

export function computeWeightTrendPoints(
  entries: WeightEntryPayload[],
  preferredUnit: string
): WeightTrendPoint[] {
  const ordered = [...entries].sort(
    (a, b) => new Date(a.recordedAt).getTime() - new Date(b.recordedAt).getTime()
  )

  return ordered.map((entry) => ({
    id: entry.id,
    recordedAt: entry.recordedAt,
    weightKg: entry.weightKg,
    displayWeight: convertKgToUnitValue(entry.weightKg, preferredUnit),
    label: new Date(entry.recordedAt).toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
    }),
  }))
}

export function summarizeActivityBreakdown(
  entries: ActivityEntryPayload[]
): ActivityBreakdownSummary {
  const intensityMinutes: Record<ActivityIntensity, number> = {
    low: 0,
    moderate: 0,
    high: 0,
    vigorous: 0,
  }

  let totalMinutes = 0
  for (const entry of entries) {
    const duration = Number.isFinite(entry.durationMinutes) ? Math.max(0, entry.durationMinutes) : 0
    totalMinutes += duration
    const key = entry.intensity.toLowerCase() as ActivityIntensity
    if (key in intensityMinutes) {
      intensityMinutes[key] += duration
    }
  }

  return {
    totalMinutes,
    averageMinutes: entries.length > 0 ? Math.round(totalMinutes / entries.length) : 0,
    sessionCount: entries.length,
    intensityMinutes,
  }
}

export function computeBmiMarkerPercent(bmi: number): number {
  if (!Number.isFinite(bmi)) {
    return 0
  }
  const minBmi = 14
  const maxBmi = 38
  const clamped = Math.max(minBmi, Math.min(maxBmi, bmi))
  return Math.round(((clamped - minBmi) / (maxBmi - minBmi)) * 100)
}

export function computeWeightGoalProgress(
  baselineWeightKg: number,
  currentWeightKg: number,
  targetWeightKg: number
): number {
  if (
    !Number.isFinite(baselineWeightKg) ||
    !Number.isFinite(currentWeightKg) ||
    !Number.isFinite(targetWeightKg)
  ) {
    return 0
  }

  const totalDistance = Math.abs(targetWeightKg - baselineWeightKg)
  if (totalDistance < 0.1) {
    return 100
  }

  const direction = targetWeightKg >= baselineWeightKg ? 1 : -1
  const progressed = (currentWeightKg - baselineWeightKg) * direction
  const pct = (progressed / totalDistance) * 100
  return Math.max(0, Math.min(100, Math.round(pct)))
}
