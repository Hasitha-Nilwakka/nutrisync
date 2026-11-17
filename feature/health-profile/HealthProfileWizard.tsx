// 4-Step Health Profile Onboarding Wizard component (wireframe/monochrome) wired to POST /api/health/profile.
// Supports metric (cm/kg) and imperial (in/lbs) inputs with real-time BMI preview, step-by-step validation,
// and inline/toast error feedback without full-page reloads.
"use client"

import { useState } from "react"
import axios from "axios"
import { useRouter } from "next/navigation"
import { useForm, useWatch } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Sparkles, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
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
  ACTIVITY_LEVEL_OPTIONS,
  DIETARY_PREFERENCE_OPTIONS,
  DIETARY_RESTRICTION_OPTIONS,
  EMPTY_HEALTH_PROFILE_FORM_DATA,
  EXERCISE_TYPE_OPTIONS,
  FITNESS_LEVEL_OPTIONS,
  GENDER_OPTIONS,
  HEIGHT_UNIT_OPTIONS,
  HealthProfileFormData,
  OCCUPATION_TYPE_OPTIONS,
  PREFERRED_ENVIRONMENT_OPTIONS,
  PREFERRED_TIME_OF_DAY_OPTIONS,
  PRIMARY_GOAL_OPTIONS,
  SAMPLE_HEALTH_PROFILE_FORM_DATA,
  SESSION_DURATION_OPTIONS,
  TARGET_ACTIVITY_LEVEL_OPTIONS,
  WEIGHT_UNIT_OPTIONS,
  calculateClientBMI,
  convertHeightToCm,
  convertWeightToKg,
  healthProfileSchema,
} from "@/lib/schemas/healthProfileSchema"

const STEP_FIELD_NAMES: Record<number, (keyof HealthProfileFormData)[]> = {
  1: ["age", "gender", "heightUnit", "height", "weightUnit", "currentWeight", "targetWeight"],
  2: ["occupationType", "activityLevel", "primaryGoal", "targetActivityLevel", "targetWeeklyDays"],
  3: [
    "weeklyActivityDays",
    "exerciseTypes",
    "sessionDuration",
    "fitnessLevel",
    "preferredEnvironment",
    "preferredTimeOfDay",
    "enduranceMinutes",
    "pushupsCount",
    "squatsCount",
  ],
  4: [
    "dietaryPreferences",
    "dietaryRestrictions",
    "medicalConditions",
    "sleepHours",
    "waterLiters",
    "allowAiProcessing",
    "publicProfileVisible",
    "emailNotifications",
  ],
}

function formatOptionLabel(value: string): string {
  return value
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ")
}

export default function HealthProfileWizard() {
  const router = useRouter()
  const [currentStep, setCurrentStep] = useState<number>(1)
  const [isDemoInterceptOpen, setIsDemoInterceptOpen] = useState<boolean>(true)
  const [isUsingSampleData, setIsUsingSampleData] = useState<boolean>(true)
  const [formInstanceKey, setFormInstanceKey] = useState<number>(0)

  const {
    register,
    handleSubmit,
    trigger,
    setValue,
    setError,
    clearErrors,
    reset,
    control,
    formState: { errors, isSubmitting },
  } = useForm<HealthProfileFormData>({
    resolver: zodResolver(healthProfileSchema),
    defaultValues: SAMPLE_HEALTH_PROFILE_FORM_DATA,
    mode: "onChange",
  })

  const handleContinueWithSampleData = () => {
    reset(SAMPLE_HEALTH_PROFILE_FORM_DATA)
    clearErrors()
    setIsUsingSampleData(true)
    setIsDemoInterceptOpen(false)
    setFormInstanceKey((prev) => prev + 1)
  }

  const handleStartWithBlankProfile = () => {
    reset(EMPTY_HEALTH_PROFILE_FORM_DATA)
    clearErrors()
    setCurrentStep(1)
    setIsUsingSampleData(false)
    setIsDemoInterceptOpen(false)
    setFormInstanceKey((prev) => prev + 1)
  }

  const watchedHeight = useWatch({ control, name: "height" })
  const watchedHeightUnit = useWatch({ control, name: "heightUnit" })
  const watchedCurrentWeight = useWatch({ control, name: "currentWeight" })
  const watchedWeightUnit = useWatch({ control, name: "weightUnit" })
  const watchedExerciseTypes = useWatch({ control, name: "exerciseTypes" }) ?? []
  const watchedDietaryPreferences = useWatch({ control, name: "dietaryPreferences" }) ?? []
  const watchedDietaryRestrictions = useWatch({ control, name: "dietaryRestrictions" }) ?? []
  const watchedAllowAiProcessing = useWatch({ control, name: "allowAiProcessing" })
  const watchedPublicProfileVisible = useWatch({ control, name: "publicProfileVisible" })
  const watchedEmailNotifications = useWatch({ control, name: "emailNotifications" })

  const liveBMIPreview = calculateClientBMI(
    Number(watchedCurrentWeight),
    watchedWeightUnit ?? "kg",
    Number(watchedHeight),
    watchedHeightUnit ?? "cm"
  )

  const toggleStringArraySelection = (
    fieldName: "exerciseTypes" | "dietaryPreferences" | "dietaryRestrictions",
    currentItems: string[],
    item: string
  ) => {
    const nextItems = currentItems.includes(item)
      ? currentItems.filter((existing) => existing !== item)
      : [...currentItems, item]
    setValue(fieldName, nextItems, { shouldValidate: true })
  }

  const handleNextStep = async () => {
    clearErrors("root")
    const fieldsToValidate = STEP_FIELD_NAMES[currentStep]
    const isStepValid = await trigger(fieldsToValidate)
    if (isStepValid && currentStep < 4) {
      setCurrentStep((prev) => prev + 1)
    }
  }

  const handlePreviousStep = () => {
    clearErrors("root")
    if (currentStep > 1) {
      setCurrentStep((prev) => prev - 1)
    }
  }

  const onSubmit = async (data: HealthProfileFormData) => {
    clearErrors("root")
    try {
      await axios.post("/api/health/profile", data)
      showAPIResponseToast({
        type: "success",
        title: "Health profile saved",
        message: "Your health profile and baseline metrics have been saved.",
      })
      router.refresh()
    } catch (error) {
      const apiErrorMessage = axios.isAxiosError(error)
        ? error.response?.data?.message || "Failed to save health profile. Please check your inputs."
        : "Failed to save health profile. Please check your inputs."
      setError("root", { message: apiErrorMessage })
      showAPIResponseToast({
        type: "error",
        title: "Validation Error",
        message: apiErrorMessage,
      })
    }
  }

  return (
    <div className="mx-auto flex min-h-[calc(100dvh-8rem)] w-full max-w-2xl flex-col justify-center px-4 py-8">
      <div className="w-full rounded-lg border p-6">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2 text-sm">
          <div className="flex items-center gap-2">
            <span className="font-medium">Step {currentStep} of 4</span>
            <span className="text-muted-foreground">
              {currentStep === 1 && "Demographics & Physical Metrics"}
              {currentStep === 2 && "Lifestyle & Goals"}
              {currentStep === 3 && "Initial Fitness Assessment"}
              {currentStep === 4 && "Nutrition, Habits & Privacy"}
            </span>
          </div>

          <div className="flex items-center gap-2">
            {isUsingSampleData ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-8 text-xs text-muted-foreground hover:text-destructive"
                onClick={handleStartWithBlankProfile}
              >
                <Trash2 className="size-3.5" />
                Clear Sample Data
              </Button>
            ) : (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-8 text-xs text-muted-foreground hover:text-primary"
                onClick={handleContinueWithSampleData}
              >
                <Sparkles className="size-3.5" />
                Load Sample Data
              </Button>
            )}
          </div>
        </div>

        <FieldSeparator />

        <form key={formInstanceKey} className="mt-4" onSubmit={handleSubmit(onSubmit)}>
          {currentStep === 1 && (
            <FieldSet>
              <FieldLegend>Step 1: Demographics &amp; Physical Metrics</FieldLegend>
              <FieldDescription>
                Choose your preferred measurement units. All measurements are automatically normalized to metric (cm /
                kg) for wellness calculations.
              </FieldDescription>

              <FieldGroup>
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field>
                    <FieldLabel htmlFor="age">Age (years)</FieldLabel>
                    <Input
                      id="age"
                      type="number"
                      min={13}
                      max={120}
                      {...register("age", { valueAsNumber: true })}
                    />
                    {errors.age && <FieldError>{errors.age.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="gender">Gender</FieldLabel>
                    <select
                      id="gender"
                      className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                      {...register("gender")}
                    >
                      {GENDER_OPTIONS.map((option) => (
                        <option key={option} value={option}>
                          {formatOptionLabel(option)}
                        </option>
                      ))}
                    </select>
                    {errors.gender && <FieldError>{errors.gender.message}</FieldError>}
                  </Field>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field>
                    <FieldLabel htmlFor="heightUnit">Preferred Height Unit</FieldLabel>
                    <select
                      id="heightUnit"
                      className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                      {...register("heightUnit")}
                    >
                      {HEIGHT_UNIT_OPTIONS.map((unit) => (
                        <option key={unit} value={unit}>
                          {unit.toUpperCase()}
                        </option>
                      ))}
                    </select>
                    {errors.heightUnit && <FieldError>{errors.heightUnit.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="height">Height ({watchedHeightUnit})</FieldLabel>
                    <Input
                      id="height"
                      type="number"
                      step="0.1"
                      {...register("height", { valueAsNumber: true })}
                    />
                    {watchedHeightUnit === "in" && Number(watchedHeight) > 0 && (
                      <FieldDescription>
                        Normalized: {convertHeightToCm(Number(watchedHeight), "in")} cm
                      </FieldDescription>
                    )}
                    {errors.height && <FieldError>{errors.height.message}</FieldError>}
                  </Field>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                  <Field>
                    <FieldLabel htmlFor="weightUnit">Preferred Weight Unit</FieldLabel>
                    <select
                      id="weightUnit"
                      className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                      {...register("weightUnit")}
                    >
                      {WEIGHT_UNIT_OPTIONS.map((unit) => (
                        <option key={unit} value={unit}>
                          {unit.toUpperCase()}
                        </option>
                      ))}
                    </select>
                    {errors.weightUnit && <FieldError>{errors.weightUnit.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="currentWeight">Current Weight ({watchedWeightUnit})</FieldLabel>
                    <Input
                      id="currentWeight"
                      type="number"
                      step="0.1"
                      {...register("currentWeight", { valueAsNumber: true })}
                    />
                    {watchedWeightUnit === "lbs" && Number(watchedCurrentWeight) > 0 && (
                      <FieldDescription>
                        Normalized: {convertWeightToKg(Number(watchedCurrentWeight), "lbs")} kg
                      </FieldDescription>
                    )}
                    {errors.currentWeight && <FieldError>{errors.currentWeight.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="targetWeight">Target Weight ({watchedWeightUnit})</FieldLabel>
                    <Input
                      id="targetWeight"
                      type="number"
                      step="0.1"
                      {...register("targetWeight", { valueAsNumber: true })}
                    />
                    {errors.targetWeight && <FieldError>{errors.targetWeight.message}</FieldError>}
                  </Field>
                </div>

                {liveBMIPreview && (
                  <div className="rounded-md border p-3 text-sm">
                    <strong>Estimated BMI Preview:</strong> {liveBMIPreview.bmi} ({liveBMIPreview.classification})
                  </div>
                )}
              </FieldGroup>
            </FieldSet>
          )}

          {currentStep === 2 && (
            <FieldSet>
              <FieldLegend>Step 2: Lifestyle &amp; Goals</FieldLegend>
              <FieldDescription>
                Tell us about your daily activity baseline and primary wellness targets.
              </FieldDescription>

              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="occupationType">Occupation Activity Type</FieldLabel>
                  <select
                    id="occupationType"
                    className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                    {...register("occupationType")}
                  >
                    {OCCUPATION_TYPE_OPTIONS.map((option) => (
                      <option key={option} value={option}>
                        {formatOptionLabel(option)}
                      </option>
                    ))}
                  </select>
                  {errors.occupationType && <FieldError>{errors.occupationType.message}</FieldError>}
                </Field>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field>
                    <FieldLabel htmlFor="activityLevel">Current Activity Level</FieldLabel>
                    <select
                      id="activityLevel"
                      className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                      {...register("activityLevel")}
                    >
                      {ACTIVITY_LEVEL_OPTIONS.map((option) => (
                        <option key={option} value={option}>
                          {formatOptionLabel(option)}
                        </option>
                      ))}
                    </select>
                    {errors.activityLevel && <FieldError>{errors.activityLevel.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="targetActivityLevel">Target Activity Level</FieldLabel>
                    <select
                      id="targetActivityLevel"
                      className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                      {...register("targetActivityLevel")}
                    >
                      {TARGET_ACTIVITY_LEVEL_OPTIONS.map((option) => (
                        <option key={option} value={option}>
                          {formatOptionLabel(option)}
                        </option>
                      ))}
                    </select>
                    {errors.targetActivityLevel && <FieldError>{errors.targetActivityLevel.message}</FieldError>}
                  </Field>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field>
                    <FieldLabel htmlFor="primaryGoal">Primary Wellness Goal</FieldLabel>
                    <select
                      id="primaryGoal"
                      className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                      {...register("primaryGoal")}
                    >
                      {PRIMARY_GOAL_OPTIONS.map((option) => (
                        <option key={option} value={option}>
                          {formatOptionLabel(option)}
                        </option>
                      ))}
                    </select>
                    {errors.primaryGoal && <FieldError>{errors.primaryGoal.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="targetWeeklyDays">Target Workout Days / Week (0–7)</FieldLabel>
                    <Input
                      id="targetWeeklyDays"
                      type="number"
                      min={0}
                      max={7}
                      {...register("targetWeeklyDays", { valueAsNumber: true })}
                    />
                    {errors.targetWeeklyDays && <FieldError>{errors.targetWeeklyDays.message}</FieldError>}
                  </Field>
                </div>
              </FieldGroup>
            </FieldSet>
          )}

          {currentStep === 3 && (
            <FieldSet>
              <FieldLegend>Step 3: Initial Fitness Assessment</FieldLegend>
              <FieldDescription>
                Establish your current exercise habits and baseline conditioning.
              </FieldDescription>

              <FieldGroup>
                <Field>
                  <FieldLabel>Preferred Exercise Types (select at least one)</FieldLabel>
                  <div className="grid grid-cols-2 gap-2 pt-1 sm:grid-cols-4">
                    {EXERCISE_TYPE_OPTIONS.map((typeOption) => (
                      <Field key={typeOption} orientation="horizontal">
                        <Checkbox
                          id={`exercise-${typeOption}`}
                          checked={watchedExerciseTypes.includes(typeOption)}
                          onCheckedChange={() =>
                            toggleStringArraySelection("exerciseTypes", watchedExerciseTypes, typeOption)
                          }
                        />
                        <FieldLabel htmlFor={`exercise-${typeOption}`} className="font-normal">
                          {formatOptionLabel(typeOption)}
                        </FieldLabel>
                      </Field>
                    ))}
                  </div>
                  {errors.exerciseTypes && <FieldError>{errors.exerciseTypes.message}</FieldError>}
                </Field>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field>
                    <FieldLabel htmlFor="weeklyActivityDays">Current Active Days / Week (0–7)</FieldLabel>
                    <Input
                      id="weeklyActivityDays"
                      type="number"
                      min={0}
                      max={7}
                      {...register("weeklyActivityDays", { valueAsNumber: true })}
                    />
                    {errors.weeklyActivityDays && <FieldError>{errors.weeklyActivityDays.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="sessionDuration">Average Session Duration</FieldLabel>
                    <select
                      id="sessionDuration"
                      className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                      {...register("sessionDuration")}
                    >
                      {SESSION_DURATION_OPTIONS.map((option) => (
                        <option key={option} value={option}>
                          {formatOptionLabel(option)}
                        </option>
                      ))}
                    </select>
                    {errors.sessionDuration && <FieldError>{errors.sessionDuration.message}</FieldError>}
                  </Field>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                  <Field>
                    <FieldLabel htmlFor="fitnessLevel">Self-Assessed Fitness</FieldLabel>
                    <select
                      id="fitnessLevel"
                      className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                      {...register("fitnessLevel")}
                    >
                      {FITNESS_LEVEL_OPTIONS.map((option) => (
                        <option key={option} value={option}>
                          {formatOptionLabel(option)}
                        </option>
                      ))}
                    </select>
                    {errors.fitnessLevel && <FieldError>{errors.fitnessLevel.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="preferredEnvironment">Environment</FieldLabel>
                    <select
                      id="preferredEnvironment"
                      className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                      {...register("preferredEnvironment")}
                    >
                      {PREFERRED_ENVIRONMENT_OPTIONS.map((option) => (
                        <option key={option} value={option}>
                          {formatOptionLabel(option)}
                        </option>
                      ))}
                    </select>
                    {errors.preferredEnvironment && <FieldError>{errors.preferredEnvironment.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="preferredTimeOfDay">Preferred Time</FieldLabel>
                    <select
                      id="preferredTimeOfDay"
                      className="h-9 w-full rounded-md border bg-transparent px-3 text-sm"
                      {...register("preferredTimeOfDay")}
                    >
                      {PREFERRED_TIME_OF_DAY_OPTIONS.map((option) => (
                        <option key={option} value={option}>
                          {formatOptionLabel(option)}
                        </option>
                      ))}
                    </select>
                    {errors.preferredTimeOfDay && <FieldError>{errors.preferredTimeOfDay.message}</FieldError>}
                  </Field>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                  <Field>
                    <FieldLabel htmlFor="enduranceMinutes">Continuous Jog/Walk (min)</FieldLabel>
                    <Input
                      id="enduranceMinutes"
                      type="number"
                      min={0}
                      max={600}
                      {...register("enduranceMinutes", { valueAsNumber: true })}
                    />
                    {errors.enduranceMinutes && <FieldError>{errors.enduranceMinutes.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="pushupsCount">Max Push-ups (reps)</FieldLabel>
                    <Input
                      id="pushupsCount"
                      type="number"
                      min={0}
                      max={500}
                      {...register("pushupsCount", { valueAsNumber: true })}
                    />
                    {errors.pushupsCount && <FieldError>{errors.pushupsCount.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="squatsCount">Max Squats (reps)</FieldLabel>
                    <Input
                      id="squatsCount"
                      type="number"
                      min={0}
                      max={500}
                      {...register("squatsCount", { valueAsNumber: true })}
                    />
                    {errors.squatsCount && <FieldError>{errors.squatsCount.message}</FieldError>}
                  </Field>
                </div>
              </FieldGroup>
            </FieldSet>
          )}

          {currentStep === 4 && (
            <FieldSet>
              <FieldLegend>Step 4: Nutrition, Habits &amp; Privacy</FieldLegend>
              <FieldDescription>
                Configure your dietary preferences, daily recovery habits, and AI privacy controls.
              </FieldDescription>

              <FieldGroup>
                <Field>
                  <FieldLabel>Dietary Preferences</FieldLabel>
                  <div className="grid grid-cols-2 gap-2 pt-1 sm:grid-cols-3">
                    {DIETARY_PREFERENCE_OPTIONS.map((pref) => (
                      <Field key={pref} orientation="horizontal">
                        <Checkbox
                          id={`diet-pref-${pref}`}
                          checked={watchedDietaryPreferences.includes(pref)}
                          onCheckedChange={() =>
                            toggleStringArraySelection("dietaryPreferences", watchedDietaryPreferences, pref)
                          }
                        />
                        <FieldLabel htmlFor={`diet-pref-${pref}`} className="font-normal">
                          {formatOptionLabel(pref)}
                        </FieldLabel>
                      </Field>
                    ))}
                  </div>
                  {errors.dietaryPreferences && <FieldError>{errors.dietaryPreferences.message}</FieldError>}
                </Field>

                <Field>
                  <FieldLabel>Dietary Restrictions / Allergies</FieldLabel>
                  <div className="grid grid-cols-2 gap-2 pt-1 sm:grid-cols-3">
                    {DIETARY_RESTRICTION_OPTIONS.map((restriction) => (
                      <Field key={restriction} orientation="horizontal">
                        <Checkbox
                          id={`diet-rest-${restriction}`}
                          checked={watchedDietaryRestrictions.includes(restriction)}
                          onCheckedChange={() =>
                            toggleStringArraySelection(
                              "dietaryRestrictions",
                              watchedDietaryRestrictions,
                              restriction
                            )
                          }
                        />
                        <FieldLabel htmlFor={`diet-rest-${restriction}`} className="font-normal">
                          {formatOptionLabel(restriction)}
                        </FieldLabel>
                      </Field>
                    ))}
                  </div>
                  {errors.dietaryRestrictions && <FieldError>{errors.dietaryRestrictions.message}</FieldError>}
                </Field>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field>
                    <FieldLabel htmlFor="sleepHours">Average Daily Sleep (hours)</FieldLabel>
                    <Input
                      id="sleepHours"
                      type="number"
                      step="0.5"
                      min={1}
                      max={24}
                      {...register("sleepHours", { valueAsNumber: true })}
                    />
                    {errors.sleepHours && <FieldError>{errors.sleepHours.message}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="waterLiters">Daily Water Intake (liters)</FieldLabel>
                    <Input
                      id="waterLiters"
                      type="number"
                      step="0.1"
                      min={0.1}
                      max={15}
                      {...register("waterLiters", { valueAsNumber: true })}
                    />
                    {errors.waterLiters && <FieldError>{errors.waterLiters.message}</FieldError>}
                  </Field>
                </div>

                <Field>
                  <FieldLabel htmlFor="medicalConditions">Medical Notes / Injury Considerations (Optional)</FieldLabel>
                  <Textarea
                    id="medicalConditions"
                    placeholder="e.g., lower back sensitivity, mild asthma..."
                    {...register("medicalConditions")}
                  />
                  {errors.medicalConditions && <FieldError>{errors.medicalConditions.message}</FieldError>}
                </Field>

                <FieldSeparator />

                <Field orientation="horizontal">
                  <Checkbox
                    id="allowAiProcessing"
                    checked={Boolean(watchedAllowAiProcessing)}
                    onCheckedChange={(checked) =>
                      setValue("allowAiProcessing", checked === true, { shouldValidate: true })
                    }
                  />
                  <FieldLabel htmlFor="allowAiProcessing" className="font-normal">
                    Allow anonymized AI health insight generation from my wellness metrics
                  </FieldLabel>
                </Field>

                <Field orientation="horizontal">
                  <Checkbox
                    id="emailNotifications"
                    checked={Boolean(watchedEmailNotifications)}
                    onCheckedChange={(checked) =>
                      setValue("emailNotifications", checked === true, { shouldValidate: true })
                    }
                  />
                  <FieldLabel htmlFor="emailNotifications" className="font-normal">
                    Receive weekly wellness progress summaries by email
                  </FieldLabel>
                </Field>

                <Field orientation="horizontal">
                  <Checkbox
                    id="publicProfileVisible"
                    checked={Boolean(watchedPublicProfileVisible)}
                    onCheckedChange={(checked) =>
                      setValue("publicProfileVisible", checked === true, { shouldValidate: true })
                    }
                  />
                  <FieldLabel htmlFor="publicProfileVisible" className="font-normal">
                    Allow my anonymized fitness milestones to be visible in community benchmarks
                  </FieldLabel>
                </Field>
              </FieldGroup>
            </FieldSet>
          )}

          {errors.root && (
            <div className="mt-4">
              <FieldError>{errors.root.message}</FieldError>
            </div>
          )}

          <div className="mt-6 flex items-center justify-between gap-3">
            <Button
              type="button"
              variant="outline"
              disabled={currentStep === 1 || isSubmitting}
              onClick={handlePreviousStep}
            >
              Back
            </Button>

            {currentStep < 4 ? (
              <Button type="button" onClick={handleNextStep}>
                Next
              </Button>
            ) : (
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting && <Spinner />}
                {isSubmitting ? "Saving Profile..." : "Complete Profile"}
              </Button>
            )}
          </div>
        </form>
      </div>

      <Dialog open={isDemoInterceptOpen} onOpenChange={setIsDemoInterceptOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Sparkles className="size-5 text-primary" />
              Explore with Sample Profile Data?
            </DialogTitle>
            <DialogDescription className="text-left leading-relaxed">
              Because NutriSync is a portfolio project, we have prefilled this health profile with realistic sample metrics so you can test the wellness engine and AI insights immediately.
            </DialogDescription>
          </DialogHeader>

          <p className="text-xs text-muted-foreground">
            Would you like to continue with prefilled sample data, or clear the form and fill in your own biometrics from scratch?
          </p>

          <DialogFooter className="gap-2 sm:gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={handleStartWithBlankProfile}
            >
              Fill in My Own Details
            </Button>
            <Button
              type="button"
              onClick={handleContinueWithSampleData}
            >
              Continue with Sample Data
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
