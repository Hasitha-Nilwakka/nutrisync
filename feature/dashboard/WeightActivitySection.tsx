// WeightActivitySection provides tabbed Weight & Activity history views and Zod-validated quick-log forms
// integrated with POST /api/health/weight and POST /api/health/activity.
"use client"

import { useState } from "react"
import axios from "axios"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Activity, Clock, Loader2, Plus, Scale } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { showAPIResponseToast } from "@/components/ui/showToast"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import {
  ACTIVITY_INTENSITY_LABELS,
  ACTIVITY_INTENSITY_OPTIONS,
  type ActivityEntryPayload,
  type ActivityLogFormValues,
  activityLogSchema,
  formatWeightForUnit,
  type WeightEntryPayload,
  type WeightLogFormValues,
  weightLogSchema,
} from "@/lib/schemas/dashboardSchema"
import { WEIGHT_UNIT_OPTIONS } from "@/lib/schemas/healthProfileSchema"

interface WeightActivitySectionProps {
  weightHistory: WeightEntryPayload[]
  activityHistory: ActivityEntryPayload[]
  preferredWeightUnit: string
  onEntryLogged: () => Promise<void>
}

function getCurrentLocalDateTimeInput(): string {
  const now = new Date()
  const offsetMs = now.getTimezoneOffset() * 60_000
  return new Date(now.getTime() - offsetMs).toISOString().slice(0, 16)
}

function WeightSparkline({ entries }: { entries: WeightEntryPayload[] }) {
  if (entries.length < 2) {
    return null
  }

  const ordered = [...entries].sort(
    (a, b) => new Date(a.recordedAt).getTime() - new Date(b.recordedAt).getTime()
  )
  const values = ordered.map((item) => item.weightKg)
  const minVal = Math.min(...values)
  const maxVal = Math.max(...values)
  const range = maxVal - minVal || 1
  const width = 260
  const height = 56
  const padding = 6

  const points = ordered
    .map((entry, idx) => {
      const x = padding + (idx / (ordered.length - 1)) * (width - padding * 2)
      const y = height - padding - ((entry.weightKg - minVal) / range) * (height - padding * 2)
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(" ")

  return (
    <div className="rounded-lg border bg-muted/20 p-3">
      <div className="mb-1 flex items-center justify-between text-[11px] text-muted-foreground">
        <span>Weight Trend ({ordered.length} entries)</span>
        <span>
          {minVal.toFixed(1)}–{maxVal.toFixed(1)} kg
        </span>
      </div>
      <svg viewBox={`0 0 ${width} ${height}`} className="h-14 w-full overflow-visible" aria-label="Weight trend chart">
        <polyline
          fill="none"
          stroke="currentColor"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="text-primary"
          points={points}
        />
      </svg>
    </div>
  )
}

export function WeightActivitySection({
  weightHistory,
  activityHistory,
  preferredWeightUnit,
  onEntryLogged,
}: WeightActivitySectionProps) {
  const [activityConflictError, setActivityConflictError] = useState<string | null>(null)
  const defaultUnit: "kg" | "lbs" = preferredWeightUnit.toLowerCase() === "lbs" ? "lbs" : "kg"

  const weightForm = useForm<WeightLogFormValues>({
    resolver: zodResolver(weightLogSchema),
    defaultValues: {
      weight: undefined,
      unit: defaultUnit,
      recordedAt: "",
    },
  })

  const activityForm = useForm<ActivityLogFormValues>({
    resolver: zodResolver(activityLogSchema),
    defaultValues: {
      exerciseType: "",
      durationMinutes: 30,
      intensity: "moderate",
      notes: "",
      recordedAt: getCurrentLocalDateTimeInput(),
    },
  })

  const handleLogWeight = async (values: WeightLogFormValues) => {
    try {
      const payload: { weight: number; unit: "kg" | "lbs"; recordedAt?: string } = {
        weight: values.weight,
        unit: values.unit,
      }
      if (values.recordedAt && values.recordedAt.trim().length > 0) {
        payload.recordedAt = new Date(values.recordedAt).toISOString()
      }

      await axios.post("/api/health/weight", payload, { withCredentials: true })
      showAPIResponseToast({
        type: "success",
        title: "Weight Logged",
        message: `Saved ${values.weight} ${values.unit} and updated your wellness score.`,
      })
      weightForm.reset({ weight: undefined, unit: values.unit, recordedAt: "" })
      await onEntryLogged()
    } catch (error) {
      const message =
        axios.isAxiosError(error) && typeof error.response?.data?.message === "string"
          ? error.response.data.message
          : "Failed to log weight entry."
      showAPIResponseToast({ type: "error", title: "Weight Log Failed", message })
    }
  }

  const handleLogActivity = async (values: ActivityLogFormValues) => {
    setActivityConflictError(null)
    try {
      await axios.post(
        "/api/health/activity",
        {
          exerciseType: values.exerciseType.trim(),
          durationMinutes: values.durationMinutes,
          intensity: values.intensity,
          notes: values.notes?.trim() || undefined,
          recordedAt: new Date(values.recordedAt).toISOString(),
        },
        { withCredentials: true }
      )
      showAPIResponseToast({
        type: "success",
        title: "Activity Logged",
        message: `Saved ${values.exerciseType} (${values.durationMinutes} min).`,
      })
      activityForm.reset({
        exerciseType: "",
        durationMinutes: 30,
        intensity: values.intensity,
        notes: "",
        recordedAt: getCurrentLocalDateTimeInput(),
      })
      await onEntryLogged()
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.status === 409) {
        const conflictMsg =
          typeof error.response.data?.message === "string"
            ? error.response.data.message
            : "An activity entry already exists for this exact timestamp."
        setActivityConflictError(conflictMsg)
        showAPIResponseToast({ type: "error", title: "Duplicate Timestamp", message: conflictMsg })
        return
      }

      const message =
        axios.isAxiosError(error) && typeof error.response?.data?.message === "string"
          ? error.response.data.message
          : "Failed to log activity entry."
      showAPIResponseToast({ type: "error", title: "Activity Log Failed", message })
    }
  }

  const sortedWeights = [...weightHistory].sort(
    (a, b) => new Date(b.recordedAt).getTime() - new Date(a.recordedAt).getTime()
  )
  const sortedActivities = [...activityHistory].sort(
    (a, b) => new Date(b.recordedAt).getTime() - new Date(a.recordedAt).getTime()
  )

  return (
    <section className="rounded-xl border bg-card p-6 shadow-xs">
      <Tabs defaultValue="weight">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-lg font-semibold tracking-tight">Weight & Activity Logs</h2>
          <TabsList className="w-full sm:w-auto">
            <TabsTrigger value="weight" className="gap-1.5 px-4">
              <Scale className="size-3.5" />
              <span>Weight ({sortedWeights.length})</span>
            </TabsTrigger>
            <TabsTrigger value="activity" className="gap-1.5 px-4">
              <Activity className="size-3.5" />
              <span>Activity ({sortedActivities.length})</span>
            </TabsTrigger>
          </TabsList>
        </div>

        {/* Weight Tab */}
        <TabsContent value="weight" className="grid gap-6 pt-2 lg:grid-cols-[340px_1fr]">
          <form onSubmit={weightForm.handleSubmit(handleLogWeight)} className="space-y-4 rounded-lg border p-4">
            <h3 className="text-sm font-semibold">Quick-Log Weight</h3>

            <div className="grid grid-cols-[1fr_92px] gap-2">
              <div className="space-y-1.5">
                <Label htmlFor="weight-input">Weight</Label>
                <Input
                  id="weight-input"
                  type="number"
                  step="0.1"
                  placeholder={defaultUnit === "lbs" ? "160.0" : "72.5"}
                  {...weightForm.register("weight", { valueAsNumber: true })}
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="weight-unit">Unit</Label>
                <select
                  id="weight-unit"
                  {...weightForm.register("unit")}
                  className="h-8 w-full rounded-lg border border-input bg-background px-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/40"
                >
                  {WEIGHT_UNIT_OPTIONS.map((unit) => (
                    <option key={unit} value={unit}>
                      {unit}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            {weightForm.formState.errors.weight && (
              <p className="text-xs text-destructive">{weightForm.formState.errors.weight.message}</p>
            )}

            <div className="space-y-1.5">
              <Label htmlFor="weight-recorded-at">Recorded At (optional — defaults to now)</Label>
              <Input id="weight-recorded-at" type="datetime-local" {...weightForm.register("recordedAt")} />
              {weightForm.formState.errors.recordedAt && (
                <p className="text-xs text-destructive">{weightForm.formState.errors.recordedAt.message}</p>
              )}
            </div>

            <Button type="submit" className="w-full" disabled={weightForm.formState.isSubmitting}>
              {weightForm.formState.isSubmitting ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Plus className="size-4" />
              )}
              <span>Log Weight</span>
            </Button>
          </form>

          <div className="space-y-3">
            <WeightSparkline entries={weightHistory} />

            <div className="max-h-72 space-y-2 overflow-y-auto pr-1">
              {sortedWeights.length === 0 ? (
                <p className="rounded-lg border border-dashed p-4 text-center text-xs text-muted-foreground">
                  No weight entries logged yet.
                </p>
              ) : (
                sortedWeights.map((entry) => (
                  <div
                    key={entry.id}
                    className="flex items-center justify-between rounded-lg border px-3.5 py-2.5 text-sm"
                  >
                    <div>
                      <p className="font-medium tabular-nums">
                        {formatWeightForUnit(entry.weightKg, preferredWeightUnit)}
                      </p>
                      <p className="text-[11px] text-muted-foreground">
                        Logged as {entry.rawValue} {entry.rawUnit}
                      </p>
                    </div>
                    <span className="text-xs text-muted-foreground">
                      {new Date(entry.recordedAt).toLocaleString()}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>
        </TabsContent>

        {/* Activity Tab */}
        <TabsContent value="activity" className="grid gap-6 pt-2 lg:grid-cols-[340px_1fr]">
          <form onSubmit={activityForm.handleSubmit(handleLogActivity)} className="space-y-3.5 rounded-lg border p-4">
            <h3 className="text-sm font-semibold">Quick-Log Activity</h3>

            <div className="space-y-1.5">
              <Label htmlFor="activity-type">Exercise Type</Label>
              <Input
                id="activity-type"
                placeholder="e.g. Running, Strength Training, Cycling"
                {...activityForm.register("exerciseType")}
              />
              {activityForm.formState.errors.exerciseType && (
                <p className="text-xs text-destructive">{activityForm.formState.errors.exerciseType.message}</p>
              )}
            </div>

            <div className="grid grid-cols-2 gap-2">
              <div className="space-y-1.5">
                <Label htmlFor="activity-duration">Duration (min)</Label>
                <Input
                  id="activity-duration"
                  type="number"
                  min={1}
                  max={1440}
                  {...activityForm.register("durationMinutes", { valueAsNumber: true })}
                />
                {activityForm.formState.errors.durationMinutes && (
                  <p className="text-xs text-destructive">{activityForm.formState.errors.durationMinutes.message}</p>
                )}
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="activity-intensity">Intensity</Label>
                <select
                  id="activity-intensity"
                  {...activityForm.register("intensity")}
                  className="h-8 w-full rounded-lg border border-input bg-background px-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/40"
                >
                  {ACTIVITY_INTENSITY_OPTIONS.map((level) => (
                    <option key={level} value={level}>
                      {ACTIVITY_INTENSITY_LABELS[level]}
                    </option>
                  ))}
                </select>
                {activityForm.formState.errors.intensity && (
                  <p className="text-xs text-destructive">{activityForm.formState.errors.intensity.message}</p>
                )}
              </div>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="activity-recorded-at">Recorded At</Label>
              <Input id="activity-recorded-at" type="datetime-local" {...activityForm.register("recordedAt")} />
              {activityForm.formState.errors.recordedAt && (
                <p className="text-xs text-destructive">{activityForm.formState.errors.recordedAt.message}</p>
              )}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="activity-notes">Notes (optional)</Label>
              <Textarea
                id="activity-notes"
                rows={2}
                placeholder="Distance, sets/reps, or how you felt..."
                {...activityForm.register("notes")}
              />
              {activityForm.formState.errors.notes && (
                <p className="text-xs text-destructive">{activityForm.formState.errors.notes.message}</p>
              )}
            </div>

            {activityConflictError && (
              <p className="rounded-md border border-destructive/30 bg-destructive/10 p-2 text-xs text-destructive">
                {activityConflictError}
              </p>
            )}

            <Button type="submit" className="w-full" disabled={activityForm.formState.isSubmitting}>
              {activityForm.formState.isSubmitting ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Plus className="size-4" />
              )}
              <span>Log Activity</span>
            </Button>
          </form>

          <div className="max-h-96 space-y-2 overflow-y-auto pr-1">
            {sortedActivities.length === 0 ? (
              <p className="rounded-lg border border-dashed p-4 text-center text-xs text-muted-foreground">
                No activity entries logged yet.
              </p>
            ) : (
              sortedActivities.map((entry) => (
                <div key={entry.id} className="space-y-1 rounded-lg border px-3.5 py-2.5 text-sm">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{entry.exerciseType}</span>
                      <span className="rounded-full border px-2 py-0.5 text-[11px] capitalize text-muted-foreground">
                        {entry.intensity}
                      </span>
                    </div>
                    <span className="flex items-center gap-1 text-xs font-medium tabular-nums">
                      <Clock className="size-3 text-muted-foreground" />
                      {entry.durationMinutes} min
                    </span>
                  </div>
                  {entry.notes && <p className="text-xs text-muted-foreground">{entry.notes}</p>}
                  <p className="text-[11px] text-muted-foreground">
                    {new Date(entry.recordedAt).toLocaleString()}
                  </p>
                </div>
              ))
            )}
          </div>
        </TabsContent>
      </Tabs>
    </section>
  )
}
