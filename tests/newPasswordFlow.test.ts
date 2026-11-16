import fs from "node:fs"
import path from "node:path"
import { describe, expect, it } from "vitest"
import { passwordChangeSchema } from "../lib/schemas/passwordChangeSchema"

const repoRoot = path.resolve(__dirname, "..")

describe("forgot password & new password flow", () => {
  it("validates matching 8+ character passwords and rejects invalid inputs", () => {
    const validResult = passwordChangeSchema.safeParse({
      password: "StrongPassword123",
      confirmPassword: "StrongPassword123",
    })
    expect(validResult.success).toBe(true)

    const shortResult = passwordChangeSchema.safeParse({
      password: "short",
      confirmPassword: "short",
    })
    expect(shortResult.success).toBe(false)

    const mismatchResult = passwordChangeSchema.safeParse({
      password: "StrongPassword123",
      confirmPassword: "DifferentPassword123",
    })
    expect(mismatchResult.success).toBe(false)
  })

  it("configures Next.js rewrites for passwordreset and passwordreset/confirm", () => {
    const nextConfigSource = fs.readFileSync(
      path.join(repoRoot, "next.config.ts"),
      "utf8"
    )
    expect(nextConfigSource).toContain('source: "/api/auth/passwordreset"')
    expect(nextConfigSource).toContain(
      'source: "/api/auth/passwordreset/confirm"'
    )
  })

  it("wraps NewPasswordForm in Suspense and posts to /api/auth/passwordreset/confirm", () => {
    const pageSource = fs.readFileSync(
      path.join(repoRoot, "app/new-password/page.tsx"),
      "utf8"
    )
    expect(pageSource).toContain("<Suspense")
    expect(pageSource).toContain("<NewPasswordForm")

    const formSource = fs.readFileSync(
      path.join(repoRoot, "feature/auth/resetpassword/NewPasswordForm.tsx"),
      "utf8"
    )
    expect(formSource).toContain('searchParams.get("token")')
    expect(formSource).toContain('axios.post("/api/auth/passwordreset/confirm"')
  })
})
