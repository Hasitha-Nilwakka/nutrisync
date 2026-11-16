"use client"

import { Droplets, Flame, Moon, Target } from "lucide-react"
import {
  computeBmiMarkerPercent,
  computeWeightGoalProgress,
  summarizeActivityBreakdown,
} from "@/lib/dashboardChartHelpers"
import {
  formatWeightForUnit,
  type ActivityEntryPayload,
  type HealthProfilePayload,
  type WeightEntryPayload,
  type WellnessSnapshotPayload,
} from "@/lib/schemas/dashboardSchema"

interface UserHealthInfographicSectionProps {
  healthProfile: HealthProfilePayload | null
  latestWellness: WellnessSnapshotPayload | null
  weightHistory: WeightEntryPayload[]
  activityHistory: ActivityEntryPayload[]
}

function WellnessRadarChart({ latestWellness }: { latestWellness: WellnessSnapshotPayload }) {
  const axes = [
    { label: "BMI (30%)", value: latestWellness.bmiScore },
    { label: "Activity (30%)", value: latestWellness.activityScore },
    { label: "Progress (20%)", value: latestWellness.progressScore },
    { label: "Habits (20%)", value: latestWellness.habitsScore },
  ]

  const center = 110
  const maxRadius = 68
  const angles = [-Math.PI / 2, 0, Math.PI / 2, Math.PI]

  const ringLevels = [25, 50, 75, 100]
  const polygonPoints = axes
    .map((axis, idx) => {
      const clamped = Math.max(0, Math.min(100, axis.value))
      const r = (clamped / 100) * maxRadius
      const x = center + r * Math.cos(angles[idx])
      const y = center + r * Math.sin(angles[idx])
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(" ")

  return (
    <svg
      viewBox="0 0 220 220"
      className="mx-auto h-52 w-52 overflow-visible"
      role="img"
      aria-label="4-axis wellness score balance radar chart"
    >
      {ringLevels.map((level) => {
        const r = (level / 100) * maxRadius
        const ringPoints = angles
          .map((angle) => `${(center + r * Math.cos(angle)).toFixed(1)},${(center + r * Math.sin(angle)).toFixed(1)}`)
          .join(" ")
        return (
          <polygon
            key={level}
            points={ringPoints}
            fill="none"
            stroke="currentColor"
            className="text-border"
            strokeDasharray={level === 100 ? undefined : "2 3"}
          />
        )
      })}

      {angles.map((angle, idx) => (
        <line
          key={idx}
          x1={center}
          y1={center}
          x2={center + maxRadius * Math.cos(angle)}
          y2={center + maxRadius * Math.sin(angle)}
          stroke="currentColor"
          className="text-border"
        />
      ))}

      <polygon
        points={polygonPoints}
        fill="rgba(24, 169, 153, 0.22)"
        stroke="#0B74B8"
        strokeWidth="2"
      />

      {axes.map((axis, idx) => {
        const clamped = Math.max(0, Math.min(100, axis.value))
        const r = (clamped / 100) * maxRadius
        const px = center + r * Math.cos(angles[idx])
        const py = center + r * Math.sin(angles[idx])
        const labelRadius = maxRadius + 22
        const lx = center + labelRadius * Math.cos(angles[idx])
        const ly = center + labelRadius * Math.sin(angles[idx])

        return (
          <g key={axis.label}>
            <circle cx={px} cy={py} r="4" fill="#18A999" stroke="var(--color-card)" strokeWidth="1.5" />
            <text
              x={lx}
              y={ly}
              textAnchor="middle"
              dominantBaseline="middle"
              className="fill-foreground text-[10px] font-medium"
            >
              {axis.label}
            </text>
            <text
              x={lx}
              y={ly + 12}
              textAnchor="middle"
              dominantBaseline="middle"
              className="fill-muted-foreground text-[9px]"
            >
              {Math.round(clamped)}/100
            </text>
          </g>
        )
      })}
    </svg>
  )
}

export function UserHealthInfographicSection({
  healthProfile,
  latestWellness,
  weightHistory,
  activityHistory,
}: UserHealthInfographicSectionProps) {
  const preferredUnit = healthProfile?.preferredWeightUnit ?? "kg"
  const latestLoggedWeight =
    weightHistory.length > 0
      ? [...weightHistory].sort(
          (a, b) => new Date(b.recordedAt).getTime() - new Date(a.recordedAt).getTime()
        )[0].weightKg
      : healthProfile?.currentWeightKg ?? null

  const bmiValue = latestWellness?.bmi ?? healthProfile?.bmi ?? null
  const bmiMarkerPct = bmiValue !== null ? computeBmiMarkerPercent(bmiValue) : 0
  const goalProgressPct =
    healthProfile && latestLoggedWeight !== null
      ? computeWeightGoalProgress(
          healthProfile.baselineWeightKg,
          latestLoggedWeight,
          healthProfile.targetWeightKg
        )
      : 0

  const activitySummary = summarizeActivityBreakdown(activityHistory)
  const totalIntensityMin = activitySummary.totalMinutes || 1
  const sleepPct = healthProfile ? Math.min(100, Math.round((healthProfile.sleepHours / 8) * 100)) : 0
  const hydrationPct = healthProfile ? Math.min(100, Math.round((healthProfile.waterLiters / 2.5) * 100)) : 0
  const weeklyDaysPct =
    healthProfile && healthProfile.targetWeeklyDays > 0
      ? Math.min(100, Math.round((healthProfile.weeklyActivityDays / healthProfile.targetWeeklyDays) * 100))
      : 0

  return (
    <div className="rounded-2xl border border-border/80 bg-card p-6 shadow-sm">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3 border-b border-border/60 pb-4">
        <div>
          <h2 className="text-lg font-semibold tracking-tight text-foreground">
            NutriSync Health Snapshot
          </h2>
          <p className="text-xs text-muted-foreground">
            Unified biometric, wellness balance, and daily habit overview
          </p>
        </div>
        {healthProfile && (
          <div className="flex flex-wrap items-center gap-2 text-xs">
            <span className="rounded-full border border-border/80 bg-muted/40 px-2.5 py-1 font-medium capitalize text-foreground">
              Goal: {healthProfile.primaryGoal.replace(/_/g, " ")}
            </span>
            <span className="rounded-full border border-border/80 bg-muted/40 px-2.5 py-1 font-medium capitalize text-muted-foreground">
              {healthProfile.fitnessLevel}
            </span>
          </div>
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Column 1: Biometrics & Goal Trajectory */}
        <div className="flex flex-col justify-between gap-5 rounded-xl border border-border/70 bg-muted/15 p-4">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                BMI Spectrum
              </span>
              <span className="text-sm font-semibold text-foreground">
                {bmiValue !== null ? `${bmiValue.toFixed(1)} · ${latestWellness?.bmiClassification ?? healthProfile?.bmiClassification ?? ""}` : "—"}
              </span>
            </div>
            <div className="relative pt-2">
              <div className="flex h-2.5 w-full overflow-hidden rounded-full">
                <div className="w-[19%] bg-sky-400/75" title="Underweight (<18.5)" />
                <div className="w-[27%] bg-emerald-500/80" title="Normal (18.5–24.9)" />
                <div className="w-[21%] bg-amber-400/80" title="Overweight (25–29.9)" />
                <div className="w-[33%] bg-rose-500/80" title="Obese (30+)" />
              </div>
              {bmiValue !== null && (
                <div
                  className="absolute top-0.5 -translate-x-1/2"
                  style={{ left: `${bmiMarkerPct}%` }}
                >
                  <div className="size-4 rounded-full border-2 border-card bg-foreground shadow-xs" />
                </div>
              )}
              <div className="mt-1.5 flex justify-between text-[10px] text-muted-foreground">
                <span>18.5</span>
                <span>25</span>
                <span>30+</span>
              </div>
            </div>
          </div>

          <div className="space-y-2.5 border-t border-border/60 pt-4">
            <div className="flex items-center justify-between">
              <span className="inline-flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                <Target className="size-3.5 text-primary" />
                Weight Goal Progress
              </span>
              <span className="text-xs font-semibold text-foreground">{goalProgressPct}%</span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-linear-to-r from-[#0B74B8] via-[#18A999] to-[#43B047] transition-all"
                style={{ width: `${goalProgressPct}%` }}
              />
            </div>
            {healthProfile && (
              <div className="flex justify-between text-[11px] text-muted-foreground">
                <span>Start: {formatWeightForUnit(healthProfile.baselineWeightKg, preferredUnit)}</span>
                <span>
                  Now: {latestLoggedWeight !== null ? formatWeightForUnit(latestLoggedWeight, preferredUnit) : "—"}
                </span>
                <span>Goal: {formatWeightForUnit(healthProfile.targetWeightKg, preferredUnit)}</span>
              </div>
            )}
          </div>
        </div>

        {/* Column 2: 4-Axis Wellness Balance Radar */}
        <div className="flex flex-col items-center justify-center rounded-xl border border-border/70 bg-muted/15 p-4">
          <p className="mb-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Wellness Balance
          </p>
          {latestWellness ? (
            <WellnessRadarChart latestWellness={latestWellness} />
          ) : (
            <p className="py-12 text-center text-xs text-muted-foreground">
              Wellness snapshot unavailable.
            </p>
          )}
        </div>

        {/* Column 3: Daily Habits & Activity Mix */}
        <div className="flex flex-col justify-between gap-4 rounded-xl border border-border/70 bg-muted/15 p-4">
          <div className="space-y-3">
            <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Daily Habits &amp; Weekly Activity
            </p>

            <div className="space-y-1">
              <div className="flex items-center justify-between text-xs">
                <span className="inline-flex items-center gap-1.5 text-muted-foreground">
                  <Moon className="size-3.5 text-[#0B74B8]" /> Sleep
                </span>
                <span className="font-medium text-foreground">
                  {healthProfile ? `${healthProfile.sleepHours} hrs / 8h` : "—"}
                </span>
              </div>
              <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                <div className="h-full rounded-full bg-[#0B74B8]" style={{ width: `${sleepPct}%` }} />
              </div>
            </div>

            <div className="space-y-1">
              <div className="flex items-center justify-between text-xs">
                <span className="inline-flex items-center gap-1.5 text-muted-foreground">
                  <Droplets className="size-3.5 text-[#18A999]" /> Hydration
                </span>
                <span className="font-medium text-foreground">
                  {healthProfile ? `${healthProfile.waterLiters} L / 2.5L` : "—"}
                </span>
              </div>
              <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                <div className="h-full rounded-full bg-[#18A999]" style={{ width: `${hydrationPct}%` }} />
              </div>
            </div>

            <div className="space-y-1">
              <div className="flex items-center justify-between text-xs">
                <span className="inline-flex items-center gap-1.5 text-muted-foreground">
                  <Flame className="size-3.5 text-[#43B047]" /> Weekly Target
                </span>
                <span className="font-medium text-foreground">
                  {healthProfile
                    ? `${healthProfile.weeklyActivityDays} / ${healthProfile.targetWeeklyDays} days`
                    : "—"}
                </span>
              </div>
              <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                <div className="h-full rounded-full bg-[#43B047]" style={{ width: `${weeklyDaysPct}%` }} />
              </div>
            </div>
          </div>

          <div className="space-y-2 border-t border-border/60 pt-3">
            <div className="flex items-center justify-between text-xs">
              <span className="text-muted-foreground">Logged Intensity Mix</span>
              <span className="font-medium text-foreground">{activitySummary.totalMinutes} min</span>
            </div>
            <div className="flex h-2 w-full overflow-hidden rounded-full bg-muted">
              <div
                style={{
                  width: `${(activitySummary.intensityMinutes.low / totalIntensityMin) * 100}%`,
                  backgroundColor: "#18A999",
                }}
              />
              <div
                style={{
                  width: `${(activitySummary.intensityMinutes.moderate / totalIntensityMin) * 100}%`,
                  backgroundColor: "#0B74B8",
                }}
              />
              <div
                style={{
                  width: `${(activitySummary.intensityMinutes.high / totalIntensityMin) * 100}%`,
                  backgroundColor: "#F59E0B",
                }}
              />
              <div
                style={{
                  width: `${(activitySummary.intensityMinutes.vigorous / totalIntensityMin) * 100}%`,
                  backgroundColor: "#EF4444",
                }}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
