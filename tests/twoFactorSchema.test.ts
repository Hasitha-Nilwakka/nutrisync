// Vitest unit tests for TOTP Two-Factor Authentication Zod v4 schemas.
import { describe, expect, it } from "vitest"
import {
  twoFactorCodeSchema,
  twoFactorLoginVerifySchema,
} from "../lib/schemas/twoFactorSchema"

describe("twoFactorCodeSchema", () => {
  it("accepts a valid 6-digit numeric TOTP code", () => {
    const result = twoFactorCodeSchema.safeParse({ code: "123456" })
    expect(result.success).toBe(true)
    if (result.success) {
      expect(result.data.code).toBe("123456")
    }
  })

  it("trims whitespace around a valid 6-digit TOTP code", () => {
    const result = twoFactorCodeSchema.safeParse({ code: "  042819  " })
    expect(result.success).toBe(true)
    if (result.success) {
      expect(result.data.code).toBe("042819")
    }
  })

  it("rejects codes shorter or longer than 6 digits", () => {
    expect(twoFactorCodeSchema.safeParse({ code: "12345" }).success).toBe(false)
    expect(twoFactorCodeSchema.safeParse({ code: "1234567" }).success).toBe(false)
    expect(twoFactorCodeSchema.safeParse({ code: "" }).success).toBe(false)
  })

  it("rejects non-numeric characters in TOTP codes", () => {
    expect(twoFactorCodeSchema.safeParse({ code: "12345a" }).success).toBe(false)
    expect(twoFactorCodeSchema.safeParse({ code: "12-345" }).success).toBe(false)
  })
})

describe("twoFactorLoginVerifySchema", () => {
  it("accepts a valid ticket and 6-digit code", () => {
    const result = twoFactorLoginVerifySchema.safeParse({
      ticket: "ticket-abc-123",
      code: "654321",
    })
    expect(result.success).toBe(true)
  })

  it("rejects an empty or whitespace-only ticket", () => {
    expect(
      twoFactorLoginVerifySchema.safeParse({ ticket: "", code: "654321" }).success
    ).toBe(false)
    expect(
      twoFactorLoginVerifySchema.safeParse({ ticket: "   ", code: "654321" }).success
    ).toBe(false)
  })

  it("rejects an invalid 6-digit code even with a valid ticket", () => {
    expect(
      twoFactorLoginVerifySchema.safeParse({
        ticket: "ticket-abc-123",
        code: "abc123",
      }).success
    ).toBe(false)
  })
})
