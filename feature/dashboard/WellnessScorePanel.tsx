// WellnessScorePanel renders the composite Wellness Score (0-100), BMI & classification,
// 4 weighted component score bars, and milestone progress cards from GET /api/health/dashboard.
"use client"

import { Activity, Award, CheckCircle2, HeartPulse, Scale, Sparkles, TrendingUp } from "lucide-react"
import {
  formatWeightForUnit,
  type HealthProfilePayload,
  type MilestoneItem,
  parseMilestonesJson,
  type WellnessSnapshotPayload,
} from "@/lib/schemas/dashboardSchema"

interface WellnessScorePanelProps {
  healthProfile: HealthProfilePayload | null
  latestWellness: WellnessSnapshotPayload | null
}

function getScoreTierLabel(score: number): { label: string; colorClass: string } {
  if (score >= 80) {
    return { label: "Optimal", colorClass: "text-emerald-600 dark:text-emerald-400" }
  }
  if (score >= 60) {
    return { label: "Good", colorClass: "text-sky-600 dark:text-sky-400" }
  }
  if (score >= 40) {
    return { label: "Building", colorClass: "text-amber-600 dark:text-amber-400" }
  }
  return { label: "Needs Focus", colorClass: "text-rose-600 dark:text-rose-400" }
}

function getBmiBadgeClasses(classification: string): string {
  const normalized = classification.toLowerCase()
  if (normalized.includes("normal")) {
    return "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
  }
  if (normalized.includes("overweight") || normalized.includes("underweight")) {
    return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300"
  }
  return "border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-300"
}

function getMilestoneIcon(category: string, id: string) {
  if (id.includes("weight") || category === "weight") {
    return <Scale className="size-4 text-sky-500" />
  }
  if (id.includes("activity") || category === "activity") {
    return <Activity className="size-4 text-emerald-500" />
  }
  return <Sparkles className="size-4 text-violet-500" />
}

export function WellnessScorePanel({ healthProfile, latestWellness }: WellnessScorePanelProps) {
  if (!latestWellness) {
    return (
      <section className="rounded-xl border bg-card p-6 shadow-xs">
        <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
          <HeartPulse className="size-4" />
          <span>Wellness Score</span>
        </div>
        <p className="mt-3 text-sm text-muted-foreground">
          No wellness snapshot has been recorded yet. Log a weight or activity entry to compute your score.
        </p>
      </section>
    )
  }

  const clampedScore = Math.max(0, Math.min(100, Math.round(latestWellness.wellnessScore * 10) / 10))
  const tier = getScoreTierLabel(clampedScore)
  const radius = 44
  const circumference = 2 * Math.PI * radius
  const strokeDashoffset = circumference - (clampedScore / 100) * circumference

  const componentBars = [
    { label: "BMI Score", weight: "30%", value: latestWellness.bmiScore, icon: Scale },
    { label: "Activity Score", weight: "30%", value: latestWellness.activityScore, icon: Activity },
    { label: "Progress Score", weight: "20%", value: latestWellness.progressScore, icon: TrendingUp },
    { label: "Habits Score", weight: "20%", value: latestWellness.habitsScore, icon: HeartPulse },
  ]

  const milestones: MilestoneItem[] = parseMilestonesJson(latestWellness.milestonesJson)
  const preferredUnit = healthProfile?.preferredWeightUnit ?? "kg"

  return (
    <section className="flex flex-col gap-6 rounded-xl border bg-card p-6 shadow-xs">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">Wellness Overview</h2>
          <p className="text-xs text-muted-foreground">
            Updated {new Date(latestWellness.recordedAt).toLocaleString()}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium">
            BMI {latestWellness.bmi.toFixed(1)}
          </span>
          <span
            className={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium capitalize ${getBmiBadgeClasses(latestWellness.bmiClassification)}`}
          >
            {latestWellness.bmiClassification}
          </span>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-[200px_1fr] md:items-center">
        {/* Composite Wellness Score Ring */}
        <div className="flex flex-col items-center justify-center rounded-lg border bg-muted/20 p-4">
          <div className="relative flex size-28 items-center justify-center">
            <svg className="size-28 -rotate-90" viewBox="0 0 100 100" aria-label={`Wellness Score ${clampedScore} out of 100`}>
              <circle
                cx="50"
                cy="50"
                r={radius}
                fill="none"
                stroke="currentColor"
                strokeWidth="8"
                className="text-muted/40"
              />
              <circle
                cx="50"
                cy="50"
                r={radius}
                fill="none"
                stroke="currentColor"
                strokeWidth="8"
                strokeLinecap="round"
                strokeDasharray={circumference}
                strokeDashoffset={strokeDashoffset}
                className={tier.colorClass}
              />
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              <span className="text-2xl font-bold tabular-nums">{clampedScore}</span>
              <span className="text-[11px] text-muted-foreground">/ 100</span>
            </div>
          </div>
          <span className={`mt-2 text-xs font-semibold ${tier.colorClass}`}>{tier.label}</span>
          {healthProfile && (
            <p className="mt-1 text-center text-[11px] text-muted-foreground">
              {formatWeightForUnit(healthProfile.currentWeightKg, preferredUnit)} →{" "}
              {formatWeightForUnit(healthProfile.targetWeightKg, preferredUnit)}
            </p>
          )}
        </div>

        {/* 4 Component Scores */}
        <div className="grid gap-3 sm:grid-cols-2">
          {componentBars.map((component) => {
            const Icon = component.icon
            const pct = Math.max(0, Math.min(100, Math.round(component.value)))
            return (
              <div key={component.label} className="rounded-lg border p-3">
                <div className="flex items-center justify-between text-xs">
                  <span className="flex items-center gap-1.5 font-medium">
                    <Icon className="size-3.5 text-muted-foreground" />
                    {component.label}
                  </span>
                  <span className="text-muted-foreground">{component.weight}</span>
                </div>
                <div className="mt-2 flex items-baseline justify-between">
                  <span className="text-lg font-semibold tabular-nums">{pct}</span>
                  <span className="text-xs text-muted-foreground">/ 100</span>
                </div>
                <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    className="h-full rounded-full bg-primary transition-all"
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Milestone Progress Cards */}
      <div className="space-y-3">
        <div className="flex items-center gap-2 text-sm font-medium">
          <Award className="size-4 text-primary" />
          <span>Milestones</span>
        </div>

        {milestones.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            Milestones will appear once your wellness snapshot includes milestone tracking.
          </p>
        ) : (
          <div className="grid gap-3 md:grid-cols-3">
            {milestones.map((milestone) => {
              const pct = Math.round(milestone.progressPct)
              return (
                <div
                  key={milestone.id || milestone.title}
                  className="flex flex-col justify-between gap-3 rounded-lg border p-3.5"
                >
                  <div className="space-y-1">
                    <div className="flex items-start justify-between gap-2">
                      <span className="flex items-center gap-1.5 text-sm font-medium">
                        {getMilestoneIcon(milestone.category, milestone.id)}
                        {milestone.title}
                      </span>
                      {milestone.achieved && (
                        <CheckCircle2 className="size-4 shrink-0 text-emerald-500" aria-label="Achieved" />
                      )}
                    </div>
                    {milestone.description && (
                      <p className="text-xs text-muted-foreground">{milestone.description}</p>
                    )}
                  </div>

                  <div>
                    <div className="mb-1 flex items-center justify-between text-[11px] text-muted-foreground">
                      <span>{milestone.achieved ? "Completed" : "In progress"}</span>
                      <span className="font-medium tabular-nums">{pct}%</span>
                    </div>
                    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                      <div
                        className={`h-full rounded-full transition-all ${
                          milestone.achieved ? "bg-emerald-500" : "bg-primary"
                        }`}
                        style={{ width: `${pct}%` }}
                      />
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </section>
  )
}
