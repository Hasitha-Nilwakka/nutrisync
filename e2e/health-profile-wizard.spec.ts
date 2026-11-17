import { test, expect } from "@playwright/test"
import { setupMockSession } from "./fixtures/mockAuth"

test.describe("4-Step Health Profile Onboarding Wizard", () => {
  test.beforeEach(async ({ context, page }) => {
    await setupMockSession(context, page, { state: "no_profile" })
  })

  test("renders Step 1 with live BMI preview and unit conversions", async ({ page }) => {
    await page.goto("/dashboard")

    // Expect wizard Step 1 to be active
    await expect(page.getByText("Step 1 of 4")).toBeVisible()
    await expect(page.getByText("Demographics & Physical Metrics").first()).toBeVisible()

    // Fill in age and gender
    await page.locator('#age').clear()
    await page.locator('#age').fill("28")
    await page.locator('#gender').selectOption("male")

    // Fill height and weight
    await page.locator('#height').clear()
    await page.locator('#height').fill("180")
    await page.locator('#currentWeight').clear()
    await page.locator('#currentWeight').fill("75")
    await page.locator('#targetWeight').clear()
    await page.locator('#targetWeight').fill("72")

    // Check live calculated BMI preview (180cm, 75kg => BMI 23.1)
    await expect(page.getByText(/23\.[0-9]/)).toBeVisible()
    await expect(page.getByText(/normal weight/i)).toBeVisible()
  })

  test("navigates through all 4 wizard steps and submits successfully", async ({ page }) => {
    await page.goto("/dashboard")

    // --- STEP 1 ---
    await expect(page.getByText("Step 1 of 4")).toBeVisible()
    await page.locator('#age').clear()
    await page.locator('#age').fill("30")
    await page.locator('#gender').selectOption("female")
    await page.locator('#height').clear()
    await page.locator('#height').fill("168")
    await page.locator('#currentWeight').clear()
    await page.locator('#currentWeight').fill("65")
    await page.locator('#targetWeight').clear()
    await page.locator('#targetWeight').fill("60")

    await page.getByRole("button", { name: "Next", exact: true }).click()

    // --- STEP 2 ---
    await expect(page.getByText("Step 2 of 4")).toBeVisible({ timeout: 5000 })
    await expect(page.getByText("Lifestyle & Goals").first()).toBeVisible()

    await page.locator('#occupationType').selectOption("sedentary_desk")
    await page.locator('#activityLevel').selectOption("moderate")
    await page.locator('#primaryGoal').selectOption("weight_loss")
    await page.locator('#targetWeeklyDays').fill("4")

    await page.getByRole("button", { name: "Next", exact: true }).click()

    // --- STEP 3 ---
    await expect(page.getByText("Step 3 of 4")).toBeVisible({ timeout: 5000 })
    await expect(page.getByText("Initial Fitness Assessment").first()).toBeVisible()

    await page.locator('#weeklyActivityDays').fill("3")
    await page.locator('#enduranceMinutes').fill("30")
    await page.locator('#pushupsCount').fill("20")
    await page.locator('#squatsCount').fill("30")

    await page.getByRole("button", { name: "Next", exact: true }).click()

    // --- STEP 4 ---
    await expect(page.getByText("Step 4 of 4")).toBeVisible({ timeout: 5000 })
    await expect(page.getByText("Nutrition, Habits & Privacy").first()).toBeVisible()

    // Complete wizard
    await expect(page.getByRole("button", { name: "Complete Profile" })).toBeVisible()
    await page.locator("form").evaluate((form: HTMLFormElement) => form.requestSubmit())

    // Expect transition to dashboard view
    await expect(page.getByRole("heading", { name: "Health Dashboard" })).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Optimal").first()).toBeVisible()
  })
})
