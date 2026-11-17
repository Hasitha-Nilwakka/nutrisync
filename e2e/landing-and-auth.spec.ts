import { test, expect } from "@playwright/test"
import { setupMockSession } from "./fixtures/mockAuth"

test.describe("Landing Page & Authentication Flow", () => {
  test.beforeEach(async ({ context, page }) => {
    await setupMockSession(context, page, { state: "unauthenticated" })
  })

  test("renders hero headline, brand identity, and wellness pillar preview", async ({ page }) => {
    await page.goto("/")

    // Brand and headline
    await expect(page.getByRole("heading", { level: 1 })).toContainText("Numbers Don't Lie")
    await expect(page.getByText("Personalized AI Nutrition").first()).toBeVisible()
    await expect(page.getByText("3-Layer AI Guardrails").first()).toBeVisible()

    // 4 Wellness formula preview pills
    await expect(page.getByText("BMI Balance", { exact: true })).toBeVisible()
    await expect(page.getByText("Activity", { exact: true }).first()).toBeVisible()
    await expect(page.getByText("Progress", { exact: true }).first()).toBeVisible()
    await expect(page.getByText("Habits", { exact: true })).toBeVisible()
  })

  test("toggles between Login and Register tabs", async ({ page }) => {
    await page.goto("/")

    const registerTrigger = page.getByRole("tab", { name: "Register" })
    const loginTrigger = page.getByRole("tab", { name: "Login" })

    await expect(loginTrigger).toHaveAttribute("aria-selected", "true")
    await expect(page.locator('#email')).toBeVisible()

    await registerTrigger.click()
    await expect(registerTrigger).toHaveAttribute("aria-selected", "true")
    await expect(page.locator('#confirmemail')).toBeVisible()

    await loginTrigger.click()
    await expect(loginTrigger).toHaveAttribute("aria-selected", "true")
    await expect(page.locator('#confirmemail')).toBeHidden()
    await expect(page.locator('#email')).toBeVisible()
  })

  test("displays validation errors on empty login submission", async ({ page }) => {
    await page.goto("/")

    const loginForm = page.locator('form').filter({ hasText: "Log in to your profile" })
    const loginSubmitBtn = loginForm.getByRole("button", { name: "Login", exact: true })
    await loginSubmitBtn.click()

    // Expect validation errors from Zod schema
    await expect(page.getByText(/email|required/i).first()).toBeVisible()
  })

  test("displays validation errors on mismatched registration passwords", async ({ page }) => {
    await page.goto("/")
    await page.getByRole("tab", { name: "Register" }).click()

    const registerForm = page.locator('form').filter({ hasText: "Create your account" })
    await registerForm.locator('#email').fill("newtester@nutrisync.dev")
    await registerForm.locator('#confirmemail').fill("newtester@nutrisync.dev")
    await registerForm.locator('#password').fill("Password123!")
    await registerForm.locator('#passwordconfirm').fill("DifferentPassword123!")

    await registerForm.getByRole("button", { name: "Register", exact: true }).click()
    await expect(page.getByText(/passwords do not match/i)).toBeVisible()
  })

  test("opens Forgot Password dialog and submits reset request", async ({ page }) => {
    await page.goto("/")

    const forgotBtn = page.getByRole("button", { name: /forgot password/i })
    await expect(forgotBtn).toBeVisible()
    await forgotBtn.click()

    // Dialog should open
    await expect(page.getByRole("dialog")).toBeVisible()
    await expect(page.getByRole("heading", { name: /reset your password/i })).toBeVisible()

    // Fill email and submit
    await page.locator('#email-reset').fill("tester@nutrisync.dev")
    await page.getByRole("button", { name: "Send Reset Link" }).click()

    // Dialog should close on successful response
    await expect(page.getByRole("dialog")).toBeHidden({ timeout: 5000 })
  })

  test("successfully submits login form and transitions to dashboard", async ({ page }) => {
    await page.goto("/")

    const loginForm = page.locator('form').filter({ hasText: "Log in to your profile" })
    await loginForm.locator('#email').fill("tester@nutrisync.dev")
    await loginForm.locator('#password').fill("ValidPass123!")

    await loginForm.getByRole("button", { name: "Login", exact: true }).click()

    // Should redirect to /dashboard
    await page.waitForURL("**/dashboard", { timeout: 10000 })
    expect(page.url()).toContain("/dashboard")
  })
})
