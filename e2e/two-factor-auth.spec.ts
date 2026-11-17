import { test, expect } from "@playwright/test"
import { setupMockSession } from "./fixtures/mockAuth"

test.describe("Two-Factor Authentication Flow", () => {
  test.beforeEach(async ({ context, page }) => {
    await setupMockSession(context, page, { state: "unauthenticated" })
  })

  test("triggers in-place 2FA challenge and rejects invalid TOTP code", async ({ page }) => {
    await page.goto("/")

    // Login with account requiring 2FA
    const loginForm = page.locator('form').filter({ hasText: "Log in to your profile" })
    await loginForm.locator('#email').fill("user-with-2fa@nutrisync.dev")
    await loginForm.locator('#password').fill("ValidPass123!")
    await loginForm.getByRole("button", { name: "Login", exact: true }).click()

    // Expect 2FA prompt to appear
    await expect(page.getByText(/two-factor authentication/i)).toBeVisible({ timeout: 5000 })
    const codeInput = page.locator('#two-factor-code')
    await expect(codeInput).toBeVisible()

    // Submit invalid code
    await codeInput.fill("000000")
    await page.getByRole("button", { name: "Verify & Login" }).click()

    // Expect error feedback
    await expect(page.getByText(/invalid/i)).toBeVisible({ timeout: 5000 })
  })

  test("successfully completes 2FA challenge with valid code and enters dashboard", async ({ page }) => {
    await page.goto("/")

    const loginForm = page.locator('form').filter({ hasText: "Log in to your profile" })
    await loginForm.locator('#email').fill("user-with-2fa@nutrisync.dev")
    await loginForm.locator('#password').fill("ValidPass123!")
    await loginForm.getByRole("button", { name: "Login", exact: true }).click()

    // Enter correct mock code
    const codeInput = page.locator('#two-factor-code')
    await expect(codeInput).toBeVisible({ timeout: 5000 })
    await codeInput.fill("123456")
    await page.getByRole("button", { name: "Verify & Login" }).click()

    // Expect redirection to /dashboard
    await page.waitForURL("**/dashboard", { timeout: 10000 })
    expect(page.url()).toContain("/dashboard")
  })
})
