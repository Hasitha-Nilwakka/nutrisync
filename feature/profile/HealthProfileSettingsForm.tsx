// Health Profile & Privacy Settings form wired to GET /api/health/profile and PUT /api/health/profile.
// Allows editing biometric metrics, fitness goals, dietary preferences/restrictions, and privacy toggles.
"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import axios from "axios"
import { useRouter } from "next/navigation"
import { useForm, useWatch, type SubmitHandler } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Activity, AlertCircle, Loader2, Save } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSeparator,
  FieldSet,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { showAPIResponseToast } from "@/components/ui/showToast"
import { Spinner } from "@/components/ui/spinner"
import { Textarea } from "@/components/ui/textarea"
import {
  mapHealthProfileResponseToFormData,
  type HealthProfileResponse,
} from "@/feature/profile/healthProfileMappers"
import {
  ACTIVITY_LEVEL_OPTIONS,
  DIETARY_PREFERENCE_OPTIONS,
  DIETARY_RESTRICTION_OPTIONS,
  EXERCISE_TYPE_OPTIONS,
  FITNESS_LEVEL_OPTIONS,
  GENDER_OPTIONS,
  HEIGHT_UNIT_OPTIONS,
  OCCUPATION_TYPE_OPTIONS,
  PREFERRED_ENVIRONMENT_OPTIONS,
  PREFERRED_TIME_OF_DAY_OPTIONS,
  PRIMARY_GOAL_OPTIONS,
  SESSION_DURATION_OPTIONS,
  TARGET_ACTIVITY_LEVEL_OPTIONS,
  WEIGHT_UNIT_OPTIONS,
  calculateClientBMI,
  healthProfileSchema,
  type HealthProfileFormData,
} from "@/lib/schemas/healthProfileSchema"

interface HealthProfileSettingsFormProps {
  consented: boolean
  hasHealthProfile: boolean
}

function formatOptionLabel(value: string): string {
  return value
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ")
}

export function HealthProfileSettingsForm({
  consented,
  hasHealthProfile,
}: HealthProfileSettingsFormProps) {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(consented && hasHealthProfile)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [profileMissing, setProfileMissing] = useState(!consented || !hasHealthProfile)

  const {
    register,
    handleSubmit,
    reset,
    setValue,
    control,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<HealthProfileFormData>({
    resolver: zodResolver(healthProfileSchema),
    mode: "onBlur",
  })

  const watchedHeight = useWatch({ control, name: "height" })
  const watchedHeightUnit = useWatch({ control, name: "heightUnit" })
  const watchedCurrentWeight = useWatch({ control, name: "currentWeight" })
  const watchedWeightUnit = useWatch({ control, name: "weightUnit" })
  const watchedExerciseTypes = useWatch({ control, name: "exerciseTypes" }) ?? []
  const watchedDietaryPreferences =
    useWatch({ control, name: "dietaryPreferences" }) ?? []
  const watchedDietaryRestrictions =
    useWatch({ control, name: "dietaryRestrictions" }) ?? []
  const watchedAllowAiProcessing = useWatch({ control, name: "allowAiProcessing" })
  const watchedPublicProfileVisible = useWatch({
    control,
    name: "publicProfileVisible",
  })
  const watchedEmailNotifications = useWatch({
    control,
    name: "emailNotifications",
  })

  useEffect(() => {
    if (!consented || !hasHealthProfile) {
      return
    }

    let cancelled = false

    axios
      .get<HealthProfileResponse>("/api/health/profile", {
        withCredentials: true,
      })
      .then((response) => {
        if (cancelled) {
          return
        }
        reset(mapHealthProfileResponseToFormData(response.data))
        setProfileMissing(false)
        setLoadError(null)
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return
        }
        if (axios.isAxiosError(error)) {
          if (error.response?.status === 401) {
            router.replace("/")
            router.refresh()
            return
          }
          if (error.response?.status === 404) {
            setProfileMissing(true)
            return
          }
          setLoadError(
            typeof error.response?.data?.error === "string"
              ? error.response.data.error
              : "Unable to load your health profile settings."
          )
          return
        }
        setLoadError("Unable to load your health profile settings.")
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [consented, hasHealthProfile, reset, router])

  const toggleArrayField = (
    fieldName: "exerciseTypes" | "dietaryPreferences" | "dietaryRestrictions",
    currentValues: string[],
    item: string
  ) => {
    const nextValues = currentValues.includes(item)
      ? currentValues.filter((entry) => entry !== item)
      : [...currentValues, item]
    setValue(fieldName, nextValues, { shouldDirty: true, shouldValidate: true })
  }

  const onSubmit: SubmitHandler<HealthProfileFormData> = async (data) => {
    try {
      const response = await axios.put<HealthProfileResponse>(
        "/api/health/profile",
        data,
        { withCredentials: true }
      )
      reset(mapHealthProfileResponseToFormData(response.data))
      showAPIResponseToast({
        type: "success",
        title: "Health Profile Updated",
        message: "Your metrics, goals, and privacy preferences have been saved.",
      })
      router.refresh()
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.status === 401) {
        router.replace("/")
        router.refresh()
        return
      }
      const message =
        axios.isAxiosError(error) && typeof error.response?.data?.error === "string"
          ? error.response.data.error
          : "Unable to save your health profile changes."
      showAPIResponseToast({
        type: "error",
        title: "Update Failed",
        message,
      })
    }
  }

  if (profileMissing) {
    return (
      <section className="w-full rounded-xl border border-border bg-card p-6 shadow-xs">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="space-y-1">
            <h2 className="text-lg font-semibold tracking-tight">
              Health Profile & Privacy Settings
            </h2>
            <p className="text-sm text-muted-foreground">
              {!consented
                ? "Complete your GDPR consent on the dashboard before configuring your health profile."
                : "Complete your initial health profile onboarding on the dashboard to unlock profile editing."}
            </p>
          </div>
          <Button render={<Link href="/dashboard" />}>
            Go to Dashboard Onboarding
          </Button>
        </div>
      </section>
    )
  }

  if (isLoading) {
    return (
      <section className="flex w-full items-center justify-center rounded-xl border border-border bg-card p-10 shadow-xs">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Loading your health profile settings...
        </div>
      </section>
    )
  }

  if (loadError) {
    return (
      <section className="w-full rounded-xl border border-destructive/40 bg-card p-6 shadow-xs">
        <div className="flex items-center gap-2 text-sm text-destructive">
          <AlertCircle className="size-4 shrink-0" />
          <span>{loadError}</span>
        </div>
      </section>
    )
  }

  const bmiPreview =
    typeof watchedCurrentWeight === "number" &&
    typeof watchedHeight === "number" &&
    (watchedWeightUnit === "kg" || watchedWeightUnit === "lbs") &&
    (watchedHeightUnit === "cm" || watchedHeightUnit === "in")
      ? calculateClientBMI(
          watchedCurrentWeight,
          watchedWeightUnit,
          watchedHeight,
          watchedHeightUnit
        )
      : null

  return (
    <section className="w-full rounded-xl border border-border bg-card p-6 shadow-xs">
      <div className="mb-6 flex flex-col justify-between gap-2 sm:flex-row sm:items-center">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">
            Health Profile & Privacy Settings
          </h2>
          <p className="text-sm text-muted-foreground">
            Update your biometric targets, fitness assessment, dietary preferences, and data privacy controls.
          </p>
        </div>
        {bmiPreview && (
          <div className="inline-flex items-center gap-2 rounded-lg border border-border bg-muted/50 px-3 py-1.5 text-sm">
            <Activity className="size-4 text-primary" />
            <span className="font-medium">BMI: {bmiPreview.bmi}</span>
            <span className="text-muted-foreground capitalize">
              ({bmiPreview.classification})
            </span>
          </div>
        )}
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-8">
        {/* Metrics & Units */}
        <FieldSet>
          <FieldLegend>Biometrics & Physical Metrics</FieldLegend>
          <FieldDescription>
            Values are normalized to metric units on the server and displayed in your preferred units.
          </FieldDescription>
          <FieldGroup className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Field>
              <FieldLabel htmlFor="settings-age">Age</FieldLabel>
              <Input
                id="settings-age"
                type="number"
                {...register("age", { valueAsNumber: true })}
              />
              {errors.age && <FieldError>{errors.age.message}</FieldError>}
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-gender">Gender</FieldLabel>
              <select
                id="settings-gender"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("gender")}
              >
                {GENDER_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>
                    {formatOptionLabel(opt)}
                  </option>
                ))}
              </select>
              {errors.gender && <FieldError>{errors.gender.message}</FieldError>}
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-height-unit">Height Unit</FieldLabel>
              <select
                id="settings-height-unit"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("heightUnit")}
              >
                {HEIGHT_UNIT_OPTIONS.map((unit) => (
                  <option key={unit} value={unit}>
                    {unit}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-height">
                Height ({watchedHeightUnit ?? "cm"})
              </FieldLabel>
              <Input
                id="settings-height"
                type="number"
                step="0.01"
                {...register("height", { valueAsNumber: true })}
              />
              {errors.height && <FieldError>{errors.height.message}</FieldError>}
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-weight-unit">Weight Unit</FieldLabel>
              <select
                id="settings-weight-unit"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("weightUnit")}
              >
                {WEIGHT_UNIT_OPTIONS.map((unit) => (
                  <option key={unit} value={unit}>
                    {unit}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-current-weight">
                Current Weight ({watchedWeightUnit ?? "kg"})
              </FieldLabel>
              <Input
                id="settings-current-weight"
                type="number"
                step="0.01"
                {...register("currentWeight", { valueAsNumber: true })}
              />
              {errors.currentWeight && (
                <FieldError>{errors.currentWeight.message}</FieldError>
              )}
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-target-weight">
                Target Weight ({watchedWeightUnit ?? "kg"})
              </FieldLabel>
              <Input
                id="settings-target-weight"
                type="number"
                step="0.01"
                {...register("targetWeight", { valueAsNumber: true })}
              />
              {errors.targetWeight && (
                <FieldError>{errors.targetWeight.message}</FieldError>
              )}
            </Field>
          </FieldGroup>
        </FieldSet>

        <FieldSeparator />

        {/* Goals & Fitness */}
        <FieldSet>
          <FieldLegend>Goals & Activity Profile</FieldLegend>
          <FieldGroup className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Field>
              <FieldLabel htmlFor="settings-primary-goal">Primary Goal</FieldLabel>
              <select
                id="settings-primary-goal"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("primaryGoal")}
              >
                {PRIMARY_GOAL_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>
                    {formatOptionLabel(opt)}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-occupation">Occupation Type</FieldLabel>
              <select
                id="settings-occupation"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("occupationType")}
              >
                {OCCUPATION_TYPE_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>
                    {formatOptionLabel(opt)}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-activity-level">
                Current Activity Level
              </FieldLabel>
              <select
                id="settings-activity-level"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("activityLevel")}
              >
                {ACTIVITY_LEVEL_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>
                    {formatOptionLabel(opt)}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-target-activity">
                Target Activity Level
              </FieldLabel>
              <select
                id="settings-target-activity"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("targetActivityLevel")}
              >
                {TARGET_ACTIVITY_LEVEL_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>
                    {formatOptionLabel(opt)}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-weekly-days">
                Current Weekly Activity Days
              </FieldLabel>
              <Input
                id="settings-weekly-days"
                type="number"
                {...register("weeklyActivityDays", { valueAsNumber: true })}
              />
              {errors.weeklyActivityDays && (
                <FieldError>{errors.weeklyActivityDays.message}</FieldError>
              )}
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-target-days">
                Target Weekly Activity Days
              </FieldLabel>
              <Input
                id="settings-target-days"
                type="number"
                {...register("targetWeeklyDays", { valueAsNumber: true })}
              />
              {errors.targetWeeklyDays && (
                <FieldError>{errors.targetWeeklyDays.message}</FieldError>
              )}
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-fitness-level">Fitness Level</FieldLabel>
              <select
                id="settings-fitness-level"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("fitnessLevel")}
              >
                {FITNESS_LEVEL_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>
                    {formatOptionLabel(opt)}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-session-duration">
                Session Duration
              </FieldLabel>
              <select
                id="settings-session-duration"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("sessionDuration")}
              >
                {SESSION_DURATION_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>
                    {formatOptionLabel(opt)}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-environment">
                Preferred Environment
              </FieldLabel>
              <select
                id="settings-environment"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("preferredEnvironment")}
              >
                {PREFERRED_ENVIRONMENT_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>
                    {formatOptionLabel(opt)}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-time-of-day">
                Preferred Time of Day
              </FieldLabel>
              <select
                id="settings-time-of-day"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs"
                {...register("preferredTimeOfDay")}
              >
                {PREFERRED_TIME_OF_DAY_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>
                    {formatOptionLabel(opt)}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-endurance">
                Endurance (Minutes)
              </FieldLabel>
              <Input
                id="settings-endurance"
                type="number"
                {...register("enduranceMinutes", { valueAsNumber: true })}
              />
              {errors.enduranceMinutes && (
                <FieldError>{errors.enduranceMinutes.message}</FieldError>
              )}
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-pushups">Pushups Count</FieldLabel>
              <Input
                id="settings-pushups"
                type="number"
                {...register("pushupsCount", { valueAsNumber: true })}
              />
              {errors.pushupsCount && (
                <FieldError>{errors.pushupsCount.message}</FieldError>
              )}
            </Field>

            <Field>
              <FieldLabel htmlFor="settings-squats">Squats Count</FieldLabel>
              <Input
                id="settings-squats"
                type="number"
                {...register("squatsCount", { valueAsNumber: true })}
              />
              {errors.squatsCount && (
                <FieldError>{errors.squatsCount.message}</FieldError>
              )}
            </Field>
          </FieldGroup>

          <div className="mt-4 space-y-2">
            <FieldLabel>Exercise Types</FieldLabel>
            <div className="flex flex-wrap gap-3">
              {EXERCISE_TYPE_OPTIONS.map((exercise) => (
                <label
                  key={exercise}
                  className="flex cursor-pointer items-center gap-2 rounded-md border border-border px-3 py-1.5 text-sm"
                >
                  <Checkbox
                    checked={watchedExerciseTypes.includes(exercise)}
                    onCheckedChange={() =>
                      toggleArrayField("exerciseTypes", watchedExerciseTypes, exercise)
                    }
                  />
                  <span>{formatOptionLabel(exercise)}</span>
                </label>
              ))}
            </div>
            {errors.exerciseTypes && (
              <FieldError>{errors.exerciseTypes.message}</FieldError>
            )}
          </div>
        </FieldSet>

        <FieldSeparator />

        {/* Nutrition & Habits */}
        <FieldSet>
          <FieldLegend>Nutrition, Habits & Medical Notes</FieldLegend>
          <div className="space-y-4">
            <div className="space-y-2">
              <FieldLabel>Dietary Preferences</FieldLabel>
              <div className="flex flex-wrap gap-2.5">
                {DIETARY_PREFERENCE_OPTIONS.map((pref) => (
                  <label
                    key={pref}
                    className="flex cursor-pointer items-center gap-2 rounded-md border border-border px-3 py-1.5 text-sm"
                  >
                    <Checkbox
                      checked={watchedDietaryPreferences.includes(pref)}
                      onCheckedChange={() =>
                        toggleArrayField(
                          "dietaryPreferences",
                          watchedDietaryPreferences,
                          pref
                        )
                      }
                    />
                    <span>{formatOptionLabel(pref)}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className="space-y-2">
              <FieldLabel>Dietary Restrictions & Allergies</FieldLabel>
              <div className="flex flex-wrap gap-2.5">
                {DIETARY_RESTRICTION_OPTIONS.map((restriction) => (
                  <label
                    key={restriction}
                    className="flex cursor-pointer items-center gap-2 rounded-md border border-border px-3 py-1.5 text-sm"
                  >
                    <Checkbox
                      checked={watchedDietaryRestrictions.includes(restriction)}
                      onCheckedChange={() =>
                        toggleArrayField(
                          "dietaryRestrictions",
                          watchedDietaryRestrictions,
                          restriction
                        )
                      }
                    />
                    <span>{formatOptionLabel(restriction)}</span>
                  </label>
                ))}
              </div>
            </div>

            <FieldGroup className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="settings-sleep">
                  Average Sleep (Hours / Night)
                </FieldLabel>
                <Input
                  id="settings-sleep"
                  type="number"
                  step="0.5"
                  {...register("sleepHours", { valueAsNumber: true })}
                />
                {errors.sleepHours && (
                  <FieldError>{errors.sleepHours.message}</FieldError>
                )}
              </Field>

              <Field>
                <FieldLabel htmlFor="settings-water">
                  Daily Water Intake (Liters)
                </FieldLabel>
                <Input
                  id="settings-water"
                  type="number"
                  step="0.1"
                  {...register("waterLiters", { valueAsNumber: true })}
                />
                {errors.waterLiters && (
                  <FieldError>{errors.waterLiters.message}</FieldError>
                )}
              </Field>
            </FieldGroup>

            <Field>
              <FieldLabel htmlFor="settings-medical">
                Medical Conditions (Optional)
              </FieldLabel>
              <Textarea
                id="settings-medical"
                rows={3}
                placeholder="Note any relevant conditions or injuries (automatically scrubbed for PII before AI analysis)."
                {...register("medicalConditions")}
              />
            </Field>
          </div>
        </FieldSet>

        <FieldSeparator />

        {/* Privacy Toggles */}
        <FieldSet>
          <FieldLegend>Privacy & Communication Preferences</FieldLegend>
          <FieldDescription>
            Control how NutriSync processes your anonymized health metrics and sends notifications.
          </FieldDescription>
          <div className="space-y-3">
            <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3">
              <Checkbox
                checked={Boolean(watchedAllowAiProcessing)}
                onCheckedChange={(checked) =>
                  setValue("allowAiProcessing", Boolean(checked), {
                    shouldDirty: true,
                  })
                }
              />
              <div className="space-y-0.5">
                <span className="text-sm font-medium">Allow AI Processing</span>
                <p className="text-xs text-muted-foreground">
                  Permit anonymized biometric trends to generate personalized AI wellness insights.
                </p>
              </div>
            </label>

            <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3">
              <Checkbox
                checked={Boolean(watchedPublicProfileVisible)}
                onCheckedChange={(checked) =>
                  setValue("publicProfileVisible", Boolean(checked), {
                    shouldDirty: true,
                  })
                }
              />
              <div className="space-y-0.5">
                <span className="text-sm font-medium">Public Profile Visible</span>
                <p className="text-xs text-muted-foreground">
                  Allow your non-sensitive wellness summary to be visible to others.
                </p>
              </div>
            </label>

            <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3">
              <Checkbox
                checked={Boolean(watchedEmailNotifications)}
                onCheckedChange={(checked) =>
                  setValue("emailNotifications", Boolean(checked), {
                    shouldDirty: true,
                  })
                }
              />
              <div className="space-y-0.5">
                <span className="text-sm font-medium">Email Notifications</span>
                <p className="text-xs text-muted-foreground">
                  Receive account security alerts and weekly progress summaries via email.
                </p>
              </div>
            </label>
          </div>
        </FieldSet>

        <div className="flex items-center justify-end gap-3 pt-2">
          <Button type="submit" disabled={isSubmitting || !isDirty}>
            {isSubmitting ? <Spinner /> : <Save className="size-4" />}
            {isSubmitting ? "Saving Changes..." : "Save Health Settings"}
          </Button>
        </div>
      </form>
    </section>
  )
}
