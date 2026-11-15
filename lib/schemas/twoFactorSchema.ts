// Zod v4 validation schemas and TypeScript contracts for TOTP Two-Factor Authentication (2FA)
// matching services/go-api/handlers/two_factor_handler.go and session_handler.go.
import { z } from "zod"

export const TOTP_CODE_REGEX = /^\d{6}$/

export const twoFactorCodeSchema = z.object({
  code: z
    .string()
    .trim()
    .regex(TOTP_CODE_REGEX, "Enter a valid 6-digit authenticator code"),
})

export const twoFactorLoginVerifySchema = z.object({
  ticket: z.string().trim().min(1, "Verification ticket is missing"),
  code: z
    .string()
    .trim()
    .regex(TOTP_CODE_REGEX, "Enter a valid 6-digit authenticator code"),
})

export type TwoFactorCodeFormData = z.infer<typeof twoFactorCodeSchema>
export type TwoFactorLoginVerifyFormData = z.infer<typeof twoFactorLoginVerifySchema>

export interface TwoFactorSetupPayload {
  secret: string
  uri: string
  qrCode: string
}

export interface TwoFactorRequiredLoginResponse {
  require2FA?: boolean
  ticket?: string
  message?: string
}
