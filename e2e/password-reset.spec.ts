import { test, expect } from "@playwright/test"
import { setupMockSession } from "./fixtures/mockAuth"

test.describe("Password Reset Confirmation Flow", () => {
  test.beforeEach(async ({ context, page }) => {
    await setupMockSession(context, page, { state: "unauthenticated" })
  })

  test("displays error or disabled state when token query param is missing", async ({ page }) => {
    await page.goto("/new-password")
    await page.waitForLoadState("domcontentloaded")

    // Expect invalid/missing token warning or disabled form
    await expect(page.getByText(/invalid or missing reset link/i)).toBeVisible()
    await expect(page.getByRole("link", { name: /back to log in/i })).toBeVisible()
  })

  test("validates password matching on /new-password?token=valid_token", async ({ page }) => {
    await page.goto("/new-password?token=valid_mock_reset_token")
    await page.waitForLoadState("domcontentloaded")

    // Password inputs should be visible
    const newPass = page.locator('#new-password')
    const confirmPass = page.locator('#confirm-new-password')

    await expect(newPass).toBeVisible({ timeout: 10000 })
    await expect(confirmPass).toBeVisible()

    // Type mismatched passwords
    await newPass.fill("SecurePass123!")
    await confirmPass.fill("DifferentPass123!")

    await page.getByRole("button", { name: "Update Password" }).click()

    // Expect mismatch error
    await expect(page.getByText(/passwords do not match/i)).toBeVisible()
  })

  test("submits valid new password and redirects to login", async ({ page }) => {
    await page.goto("/new-password?token=valid_mock_reset_token")
    await page.waitForLoadState("domcontentloaded")

    const newPass = page.locator('#new-password')
    const confirmPass = page.locator('#confirm-new-password')
    await expect(newPass).toBeVisible({ timeout: 10000 })

    await newPass.fill("BrandNewPassword123!")
    await confirmPass.fill("BrandNewPassword123!")

    await page.getByRole("button", { name: "Update Password" }).click()

    // Should redirect to root /
    await expect(page).toHaveURL(/\/$/, { timeout: 10000 })
  })
})
