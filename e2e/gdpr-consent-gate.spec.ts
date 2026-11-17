import { test, expect } from "@playwright/test"
import { setupMockSession } from "./fixtures/mockAuth"

test.describe("Mandatory GDPR Consent Gating", () => {
  test.beforeEach(async ({ context, page }) => {
    await setupMockSession(context, page, { state: "unconsented" })
  })

  test("blocks access to /dashboard and renders GdprConsentCard when unconsented", async ({ page }) => {
    await page.goto("/dashboard")

    // GdprConsentCard must be displayed
    await expect(page.getByText(/data privacy & health data consent/i)).toBeVisible()
    await expect(page.getByText(/health & lifestyle data collected/i)).toBeVisible()

    // Accept button should be disabled until checkbox is checked
    const acceptBtn = page.getByRole("button", { name: "Accept & Continue" })
    await expect(acceptBtn).toBeDisabled()
  })

  test("blocks access to /profile and renders GdprConsentCard when unconsented", async ({ page }) => {
    await page.goto("/profile")

    await expect(page.getByText(/data privacy & health data consent/i)).toBeVisible()
    const acceptBtn = page.getByRole("button", { name: "Accept & Continue" })
    await expect(acceptBtn).toBeDisabled()
  })

  test("enables accept button upon checking consent checkbox and records consent", async ({ page }) => {
    await page.goto("/dashboard")

    // Click the custom Checkbox root button
    const checkbox = page.locator('[data-slot="checkbox"]')
    await checkbox.click()

    const acceptBtn = page.getByRole("button", { name: "Accept & Continue" })
    await expect(acceptBtn).toBeEnabled()

    await acceptBtn.click()

    // Should transition to onboarding wizard step 1
    await expect(page.getByText("Demographics & Physical Metrics").first()).toBeVisible({ timeout: 10000 })
  })

  test("declining consent removes unconsented account and redirects to landing", async ({ page }) => {
    await page.goto("/dashboard")

    const declineBtn = page.getByRole("button", { name: "Decline & Delete Account" })
    await expect(declineBtn).toBeVisible()
    await declineBtn.click()

    // Expect redirection to root /
    await expect(page).toHaveURL("http://127.0.0.1:3000/", { timeout: 10000 })
    await expect(page.getByRole("heading", { level: 1 })).toContainText("Numbers Don't Lie")
  })
})
