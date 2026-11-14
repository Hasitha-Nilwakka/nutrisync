// Starter Zod v4 validation schema template for lib/schemas/<feature>Schema.ts.
// Keep field names and numeric bounds synchronized with the corresponding Go handler payload.
import { z } from "zod"

export const METRIC_UNIT_OPTIONS = ["kg", "lbs"] as const

export const domainFeatureSchema = z.object({
  recordedValue: z
    .number({ message: "Please enter a valid number" })
    .positive("Value must be greater than zero")
    .max(500, "Value exceeds physiological bounds"),
  unit: z.enum(METRIC_UNIT_OPTIONS, {
    message: "Please select a valid unit",
  }),
  notes: z
    .string()
    .trim()
    .max(280, "Notes must be 280 characters or fewer")
    .optional(),
})

export type DomainFeatureFormValues = z.infer<typeof domainFeatureSchema>
