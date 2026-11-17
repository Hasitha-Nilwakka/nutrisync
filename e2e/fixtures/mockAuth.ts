import type { BrowserContext, Page } from "@playwright/test"
import { MOCK_DASHBOARD_PAYLOAD, MOCK_REGENERATE_PAYLOAD } from "./mockDashboardData"

export type SessionState = "unauthenticated" | "unconsented" | "no_profile" | "authenticated" | "two_factor"

export interface MockSessionOptions {
  state?: SessionState
  email?: string
  userId?: string
}

/**
 * Creates a well-formed synthetic JWT with an unexpired `exp` claim,
 * ensuring proxy.ts and isJwtExpired validate it without premature refresh.
 */
export function createMockJwt(state: string, expSeconds = 3600): string {
  const header = Buffer.from(JSON.stringify({ alg: "HS256", typ: "JWT" })).toString("base64url")
  const exp = Math.floor(Date.now() / 1000) + expSeconds
  const payload = Buffer.from(JSON.stringify({ sub: "usr_mock_123", state, exp })).toString("base64url")
  return `${header}.${payload}.mock_sig_${state}`
}

/**
 * Configure browser cookies and API route intercepts for a given session state.
 */
export async function setupMockSession(
  context: BrowserContext,
  page: Page,
  options: MockSessionOptions = {}
) {
  const {
    state = "authenticated",
    email = state === "unconsented"
      ? "unconsented@nutrisync.dev"
      : state === "no_profile"
      ? "newuser@nutrisync.dev"
      : "tester@nutrisync.dev",
    userId = "usr_mock_123",
  } = options

  // Clear existing cookies
  await context.clearCookies()

  if (state !== "unauthenticated") {
    const token = createMockJwt(state)

    await context.addCookies([
      {
        name: "access_token",
        value: token,
        url: "http://127.0.0.1:3000",
        httpOnly: true,
        sameSite: "Lax",
      },
      {
        name: "refresh_token",
        value: `ref_${state}`,
        url: "http://127.0.0.1:3000",
        httpOnly: true,
        sameSite: "Lax",
      },
    ])
  }

  // Intercept client-side API requests via page.route
  await page.route("**/api/auth/session", async (route) => {
    if (state === "unauthenticated") {
      await route.fulfill({ status: 401, json: { error: "Unauthorized" } })
      return
    }
    await route.fulfill({
      status: 200,
      json: {
        userId,
        sessionId: "sess_mock_456",
        email,
        consented: state !== "unconsented",
        twoFactorEnabled: state === "two_factor",
        hasHealthProfile: state === "authenticated" || state === "two_factor",
      },
    })
  })

  await page.route("**/api/auth/login", async (route) => {
    let postData: Record<string, string> = {}
    try {
      postData = route.request().postDataJSON() || {}
    } catch {}

    if (postData.email?.includes("2fa")) {
      await route.fulfill({
        status: 200,
        json: { require2FA: true, ticket: "mock_2fa_ticket" },
      })
      return
    }

    await context.addCookies([
      {
        name: "access_token",
        value: createMockJwt("authenticated"),
        url: "http://127.0.0.1:3000",
        httpOnly: true,
        sameSite: "Lax",
      },
    ])

    await route.fulfill({
      status: 200,
      json: { message: "Login successful" },
    })
  })

  await page.route("**/api/auth/2fa/login*", async (route) => {
    let postData: Record<string, string> = {}
    try {
      postData = route.request().postDataJSON() || {}
    } catch {}

    if (postData.code === "123456") {
      await context.addCookies([
        {
          name: "access_token",
          value: createMockJwt("authenticated"),
          url: "http://127.0.0.1:3000",
          httpOnly: true,
          sameSite: "Lax",
        },
      ])
      await route.fulfill({
        status: 200,
        json: { message: "2FA verified" },
      })
      return
    }

    await route.fulfill({
      status: 400,
      json: { message: "Invalid or expired 2FA code." },
    })
  })

  await page.route("**/api/auth/register", async (route) => {
    await route.fulfill({
      status: 201,
      json: {
        type: "success",
        title: "Registration Successful",
        message: "Account created. Please check your email to verify your address.",
      },
    })
  })

  await page.route("**/api/auth/passwordreset", async (route) => {
    await route.fulfill({
      status: 200,
      json: {
        type: "success",
        title: "Password Reset Link Sent",
        message: "If an account exists for that email, a password reset link has been sent.",
      },
    })
  })

  await page.route("**/api/auth/passwordreset/confirm", async (route) => {
    await route.fulfill({
      status: 200,
      json: {
        type: "success",
        title: "Password Updated",
        message: "Your password has been reset. You can now log in with your new password.",
      },
    })
  })

  await page.route("**/api/health/dashboard", async (route) => {
    if (state === "unauthenticated") {
      await route.fulfill({ status: 401, json: { error: "Unauthorized" } })
      return
    }
    await route.fulfill({
      status: 200,
      json: MOCK_DASHBOARD_PAYLOAD,
    })
  })

  await page.route("**/api/health/weight", async (route) => {
    if (route.request().method() === "POST") {
      await route.fulfill({
        status: 201,
        json: {
          type: "success",
          title: "Weight Logged",
          message: "Weight entry recorded successfully.",
        },
      })
      return
    }
    await route.fallback()
  })

  await page.route("**/api/health/activity", async (route) => {
    if (route.request().method() === "POST") {
      await route.fulfill({
        status: 201,
        json: {
          type: "success",
          title: "Activity Logged",
          message: "Activity entry recorded successfully.",
        },
      })
      return
    }
    await route.fallback()
  })

  await page.route("**/api/health/**regenerate**", async (route) => {
    await route.fulfill({
      status: 200,
      json: MOCK_REGENERATE_PAYLOAD,
    })
  })

  await page.route("**/api/health/export", async (route) => {
    await route.fulfill({
      status: 200,
      headers: {
        "content-type": "application/json",
        "content-disposition": 'attachment; filename="nutrisync-export.json"',
      },
      body: JSON.stringify(MOCK_DASHBOARD_PAYLOAD),
    })
  })

  await page.route("**/api/auth/consent", async (route) => {
    let postData: Record<string, unknown> = {}
    try {
      postData = route.request().postDataJSON() || {}
    } catch {}

    if (postData.consented === false) {
      await context.clearCookies()
      await route.fulfill({
        status: 200,
        json: { message: "Account removed successfully" },
      })
      return
    }

    await context.addCookies([
      {
        name: "access_token",
        value: createMockJwt("no_profile"),
        url: "http://127.0.0.1:3000",
        httpOnly: true,
        sameSite: "Lax",
      },
    ])
    await route.fulfill({
      status: 200,
      json: { message: "Consent updated successfully" },
    })
  })

  await page.route("**/api/auth/account", async (route) => {
    if (route.request().method() === "DELETE") {
      await context.clearCookies()
      await route.fulfill({
        status: 200,
        json: { message: "Account deleted successfully" },
      })
      return
    }
    await route.fallback()
  })

  await page.route("**/api/auth/2fa/setup", async (route) => {
    await route.fulfill({
      status: 200,
      json: {
        secret: "JBSWY3DPEHPK3PXP",
        otpauthUrl: "otpauth://totp/NutriSync:test@nutrisync.dev?secret=JBSWY3DPEHPK3PXP&issuer=NutriSync",
        qrCodeDataUrl: "data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='100' height='100'><rect width='100' height='100' fill='black'/></svg>",
      },
    })
  })

  await page.route("**/api/auth/2fa/enable", async (route) => {
    await route.fulfill({
      status: 200,
      json: {
        type: "success",
        title: "2FA Enabled",
        message: "Two-factor authentication is now active.",
      },
    })
  })

  await page.route("**/api/auth/2fa/disable", async (route) => {
    await route.fulfill({
      status: 200,
      json: {
        type: "success",
        title: "2FA Disabled",
        message: "Two-factor authentication has been disabled.",
      },
    })
  })

  await page.route("**/api/health/profile", async (route) => {
    await context.addCookies([
      {
        name: "access_token",
        value: createMockJwt("authenticated"),
        url: "http://127.0.0.1:3000",
        httpOnly: true,
        sameSite: "Lax",
      },
    ])
    await route.fulfill({
      status: 200,
      json: {
        type: "success",
        title: "Profile Saved",
        message: "Health profile updated successfully.",
      },
    })
  })
}
