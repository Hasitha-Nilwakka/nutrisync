// Starter Vitest unit test template for tests/<feature>Schema.test.ts.
import { describe, expect, it } from "vitest"
import { domainFeatureSchema } from "../lib/schemas/healthProfileSchema" // Replace with target schema import

describe("domainFeatureSchema validation", () => {
  it("accepts valid input within physiological bounds", () => {
    const validInput = {
      recordedValue: 72.5,
      unit: "kg",
      notes: "Morning weigh-in",
    }
    // Replace with target schema:
    expect(validInput.recordedValue).toBeGreaterThan(0)
    void domainFeatureSchema
  })

  it("rejects non-positive or out-of-bounds values", () => {
    const invalidInput = {
      recordedValue: -5,
      unit: "kg",
    }
    expect(invalidInput.recordedValue).toBeLessThan(0)
  })
})
