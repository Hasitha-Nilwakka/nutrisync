import { test, expect } from "@playwright/test"
import { setupMockSession } from "./fixtures/mockAuth"

test.describe("Profile Management & Account Deletion Flow", () => {
  test.beforeEach(async ({ context, page }) => {
    await setupMockSession(context, page, { state: "authenticated" })
  })

  test("renders profile account summary, badges, and biometrics form", async ({ page }) => {
    await page.goto("/profile")

    // Account summary header
    await expect(page.getByText(/tester@nutrisync.dev/i).first()).toBeVisible()
    await expect(page.getByText(/two-factor authentication \(totp\)/i)).toBeVisible()

    // Health profile settings form inputs
    await expect(page.locator('#profile-age, input[name="age"]').first()).toBeVisible()
  })

  test("opens 2FA setup dialog and displays secret key and QR code", async ({ page }) => {
    await page.goto("/profile")

    const setup2faBtn = page.getByRole("button", { name: /enable 2fa/i })
    await expect(setup2faBtn).toBeVisible()
    await setup2faBtn.click()

    // Dialog should open
    await expect(page.getByRole("dialog")).toBeVisible()
    await expect(page.getByRole("heading", { name: /set up authenticator app/i })).toBeVisible()

    // Secret key and QR code should be present
    await expect(page.getByRole("img", { name: /totp two-factor authentication qr code/i })).toBeVisible()
    await expect(page.getByRole("textbox", { name: "TOTP Manual Secret Key" })).toHaveValue(/JBSWY3DPEHPK3PXP/i)
  })

  test("enforces typed 'DELETE' verification before enabling account deletion button", async ({ page }) => {
    await page.goto("/profile")

    // Locate Delete Account card button
    const deleteTrigger = page.getByRole("button", { name: "Delete Account & Data" })
    await expect(deleteTrigger).toBeVisible()
    await deleteTrigger.click()

    // Confirmation dialog opens
    await expect(page.getByRole("dialog")).toBeVisible()
    await expect(page.getByText(/permanently delete account & profile/i)).toBeVisible()

    const confirmInput = page.locator('#delete-account-confirm-input')
    await expect(confirmInput).toBeVisible()

    // Confirm button must be disabled initially
    const confirmBtn = page.getByRole("dialog").getByRole("button", { name: "Permanently Delete Account" })
    await expect(confirmBtn).toBeDisabled()

    // Type incorrect text
    await confirmInput.fill("delete")
    await expect(confirmBtn).toBeDisabled()

    // Type exact required text "DELETE"
    await confirmInput.fill("DELETE")
    await expect(confirmBtn).toBeEnabled()

    // Confirm deletion
    await confirmBtn.click()

    // Should redirect to root /
    await page.waitForURL("**/", { timeout: 10000 })
    expect(page.url()).not.toContain("/profile")
  })
})
