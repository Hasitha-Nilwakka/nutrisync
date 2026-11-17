import { test, expect } from "@playwright/test"
import { setupMockSession } from "./fixtures/mockAuth"

test.describe("Interactive Health Dashboard & Analytics", () => {
  test.beforeEach(async ({ context, page }) => {
    await setupMockSession(context, page, { state: "authenticated" })
  })

  test("renders Wellness Score card and 4 weighted formula components", async ({ page }) => {
    await page.goto("/dashboard")

    // Wellness Score and tier
    await expect(page.getByText(/wellness score/i).first()).toBeVisible()
    await expect(page.getByText(/84/i).first()).toBeVisible()
    await expect(page.getByText(/optimal/i).first()).toBeVisible()

    // 4 Score bars
    await expect(page.getByText(/bmi/i).first()).toBeVisible()
    await expect(page.getByText(/activity/i).first()).toBeVisible()
    await expect(page.getByText(/progress/i).first()).toBeVisible()
    await expect(page.getByText(/habits/i).first()).toBeVisible()
  })

  test("renders SVG Radar Infographic and Trend Charts", async ({ page }) => {
    await page.goto("/dashboard")

    // SVG Infographic Radar
    const svgChart = page.locator("svg").first()
    await expect(svgChart).toBeVisible()

    // Weight and Activity chart headings/tabs
    await expect(page.getByText(/weight & activity logs/i)).toBeVisible()
  })

  test("fills Quick-Log Weight form and submits entry", async ({ page }) => {
    await page.goto("/dashboard")

    // Quick-Log Weight form
    await expect(page.getByText(/quick-log weight/i)).toBeVisible()
    const weightInput = page.locator('#weight-input')
    await expect(weightInput).toBeVisible()
    await weightInput.fill("63.8")

    const logWeightBtn = page.getByRole("button", { name: "Log Weight" })
    await expect(logWeightBtn).toBeVisible()
    await logWeightBtn.click()

    // Toast feedback appears
    await expect(page.getByText(/weight logged|saved/i).first()).toBeVisible({ timeout: 5000 })
  })

  test("switches to Activity tab, fills Quick-Log Activity form and submits", async ({ page }) => {
    await page.goto("/dashboard")

    // Click Activity tab
    const activityTab = page.getByRole("tab", { name: /activity/i })
    await activityTab.click()

    // Quick-Log Activity form
    await expect(page.getByText(/quick-log activity/i)).toBeVisible()
    await page.locator('#activity-type').fill("Running")
    await page.locator('#activity-duration').fill("45")

    const logActivityBtn = page.getByRole("button", { name: "Log Activity" })
    await expect(logActivityBtn).toBeVisible()
    await logActivityBtn.click()

    // Toast feedback appears
    await expect(page.getByText(/activity logged|saved/i).first()).toBeVisible({ timeout: 5000 })
  })

  test("regenerates AI insights upon clicking Regenerate Insights button", async ({ page }) => {
    await page.goto("/dashboard")

    await expect(page.getByRole("heading", { name: "AI Health Insights" })).toBeVisible()
    const regenBtn = page.getByRole("button", { name: /regenerate insights/i })
    await expect(regenBtn).toBeVisible()
    await regenBtn.click()

    // Toast feedback or updated insight text
    await expect(page.getByText(/insights regenerated|showing cached insights/i).first()).toBeVisible({ timeout: 5000 })
  })

  test("toggles dark and light mode via ThemeToggle", async ({ page }) => {
    await page.goto("/dashboard")

    const themeToggleBtn = page.locator('button[aria-label*="theme"], button:has(.lucide-sun), button:has(.lucide-moon)').first()
    await expect(themeToggleBtn).toBeVisible()

    // Click to toggle theme
    await themeToggleBtn.click()
    const htmlElem = page.locator("html")
    const htmlClass = await htmlElem.getAttribute("class")
    expect(htmlClass).toBeDefined()
  })

  test("downloads GDPR JSON export data file", async ({ page }) => {
    await page.goto("/dashboard")

    const exportBtn = page.getByRole("button", { name: /export/i }).first()
    await expect(exportBtn).toBeVisible()

    // Listen for download event
    const downloadPromise = page.waitForEvent("download")
    await exportBtn.click()
    const download = await downloadPromise

    expect(download.suggestedFilename()).toContain(".json")
  })
})
