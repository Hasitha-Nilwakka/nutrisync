# Feature Spec: Comprehensive Playwright UI Testing Suite

- **Status:** Completed
- **Author:** Agent Antigravity
- **Target Issue / PR:** #UI-E2E-Playwright
- **Risk Level (`AGENTS.md` §10):** Medium
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `package.json`, `playwright.config.ts`, `e2e/`
  - [ ] **Go Core API (`services/go-api/`)** — Not affected (hermetic network mocks)
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — Not affected

---

## 1. Objective & Scope

- **Problem Statement:** 
  NutriSync currently has unit and schema tests in Vitest (`tests/*.test.ts`, 54 tests), but completely lacks end-to-end (E2E) browser-level test coverage. Critical user-facing flows—such as authentication tab switching, TOTP 2FA verification challenges, mandatory GDPR consent gating, the 4-step biometrics and lifestyle onboarding wizard, interactive dashboard analytics (radar infographic, weight & activity charts, quick-log dialogs, AI insight regeneration), profile settings with typed `"DELETE"` account destruction, and dark/light theme switching—have no automated browser verification against real DOM events, cookies, redirects, and network states.

- **In Scope:**
  1. **Dependency Installation:** Add `@playwright/test` devDependency to `package.json` (requires explicit user approval per `AGENTS.md` §5).
  2. **Playwright Configuration:** Create `playwright.config.ts` configured for Next.js App Router (Turbopack `pnpm dev` or `pnpm start` webServer on `http://127.0.0.1:3000`), traces/screenshots on failure, and browser projects (Desktop Chrome, Desktop Firefox, Desktop Safari/WebKit, and Mobile Chrome).
  3. **Hermetic Mock Fixtures & Route Handlers (`e2e/fixtures/`):**
     - `mockAuth.ts`: Helpers to set authenticated/unauthenticated session cookies (`access_token`, `refresh_token`), mock `/api/auth/session`, `/api/auth/login`, `/api/auth/register`, `/api/auth/verify`, `/api/auth/consent`, `/api/auth/account`, and `/api/auth/2fa/*`.
     - `mockDashboardData.ts`: Synthetic health profile, weight logs, activity logs, Wellness Score (84/100), milestones, and AI insights matching the Go API contract.
  4. **Comprehensive Test Suite Modules (`e2e/*.spec.ts`):**
     - `e2e/landing-and-auth.spec.ts`: Landing page layout, HeroSection branding, AuthTabs toggle, login validation, registration validation, forgot-password modal.
     - `e2e/two-factor-auth.spec.ts`: In-place TOTP code challenge during login, error handling for bad code, successful authentication.
     - `e2e/gdpr-consent-gate.spec.ts`: Enforcement of mandatory GDPR consent card on `/dashboard` and `/profile`, accept action leading to dashboard, decline action modal.
     - `e2e/health-profile-wizard.spec.ts`: 4-step wizard on first login: Step 1 (biometrics with live BMI calculation and imperial/metric unit toggle), Step 2 (lifestyle & goals), Step 3 (activity & fitness), Step 4 (dietary restrictions & summary submission).
     - `e2e/dashboard-analytics.spec.ts`: Full interactive dashboard: Wellness score panel, Radar infographic SVG, WeightTrendChart, ActivityTrendChart, Log Weight dialog, Log Activity dialog, AI Insights regenerate trigger, GDPR JSON export, and ThemeToggle (dark/light mode).
     - `e2e/profile-and-account-deletion.spec.ts`: `/profile` biometrics updates, TOTP 2FA setup & disable modal, and typed `"DELETE"` account deletion safety dialog.
     - `e2e/password-reset.spec.ts`: `/new-password?token=...` form validation, token states, and password update redirect.
  5. **NPM Scripts:** Add `"test:e2e"`, `"test:e2e:ui"`, and `"test:e2e:headed"` to `package.json`.

- **Out of Scope:**
  - Modifying production UI code or changing existing Vitest unit test files.
  - Modifying Go API backend code or Python AI service code.
  - Relying on external third-party services (live Google/GitHub OAuth servers or live SMTP servers) in automated tests; tests use hermetic `page.route` intercepts for deterministic execution.

- **Acceptance Criteria:**
  - [x] `@playwright/test` installed and configured with `playwright.config.ts`.
  - [x] All 7 comprehensive spec test suites implemented in `e2e/`.
  - [x] Tests run and pass deterministically via `pnpm test:e2e` (or `pnpm exec playwright test`).
  - [x] Vitest unit tests continue to pass 100% (`pnpm test` -> 54/54 tests pass).
  - [x] TypeScript typecheck (`pnpm exec tsc --noEmit`) and linting (`pnpm lint`) pass with 0 errors.

- **Pre-Flight MCP & Context Verification:**
  - [x] Verified existing `package.json` scripts (`dev`, `build`, `start`, `lint`, `test`).
  - [x] Checked App Router structure (`app/`, `feature/`, `components/ui/`, `lib/`).
  - [x] Verified `next.config.ts` rewrites to Go API on port `8080`.
  - [x] Confirmed zero collisions with existing Vitest configuration (`vitest.config.mts` uses `./tests/`).

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
- **Tables / Models Affected:** None.
- **Migration & Transaction Strategy:** No database changes required.

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints Affected:** None. E2E browser tests run against the Next.js frontend with mocked API responses via Playwright's `page.route` or optional live dev server wiring.

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- **Impact:** None.

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `lib/schemas/`, `e2e/`)

#### Playwright Configuration (`playwright.config.ts`):
```typescript
import { defineConfig, devices } from "@playwright/test"

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [["html", { open: "never" }], ["list"]],
  use: {
    baseURL: process.env.PLAYWRIGHT_TEST_BASE_URL || "http://127.0.0.1:3000",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "firefox",
      use: { ...devices["Desktop Firefox"] },
    },
    {
      name: "webkit",
      use: { ...devices["Desktop Safari"] },
    },
    {
      name: "Mobile Chrome",
      use: { ...devices["Pixel 5"] },
    },
  ],
  webServer: {
    command: "pnpm dev",
    url: "http://127.0.0.1:3000",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
})
```

#### Mocking Architecture (`e2e/fixtures/`):
Playwright tests will utilize reusable fixture functions to intercept network requests:
1. `mockSession(page, options)`:
   - Sets the `access_token` cookie so `requireAuthentication()` succeeds.
   - Intercepts `GET /api/auth/session` to return:
     - `userId: "usr_mock_123"`
     - `sessionId: "sess_mock_456"`
     - `email: "tester@nutrisync.dev"`
     - `consented: boolean` (configurable)
     - `twoFactorEnabled: boolean` (configurable)
     - `hasHealthProfile: boolean` (configurable)
2. `mockDashboardApis(page, options)`:
   - Intercepts `GET /api/health/dashboard` to return synthetic wellness metrics, weight history, activity history, and AI insights.
   - Intercepts `POST /api/health/weight` and `POST /api/health/activity` to simulate successful logging with updated records.
   - Intercepts `POST /api/health/ai/regenerate` to simulate AI insight updates.
   - Intercepts `GET /api/health/export` to simulate GDPR data export JSON.
3. `mockProfileApis(page, options)`:
   - Intercepts `POST /api/auth/2fa/setup`, `POST /api/auth/2fa/enable`, and `POST /api/auth/2fa/disable`.
   - Intercepts `DELETE /api/auth/account` to simulate account deletion.
   - Intercepts `POST /api/health/profile` to simulate health profile updates.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **Zero Real Credentials or PII in Tests:** All user identifiers in E2E tests are synthetic fixtures (e.g. `tester@nutrisync.dev`, `usr_mock_123`). No real passwords, JWT secrets, or live emails are utilized.
- **GDPR Consent Verification:** E2E test suite explicitly asserts that unconsented users cannot bypass `GdprConsentCard` to access health data, and tests both the consent accept path and decline path.
- **Account Deletion Safety:** E2E test suite verifies that the account deletion button remains disabled until the exact string `"DELETE"` is entered by the user.
- **Authentication Isolation:** Verifies that attempting to access `/dashboard` or `/profile` without an active session immediately redirects to the unauthenticated landing page `/`.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [ ] **Step 1 — Dependency Installation:** Request explicit user approval and install `@playwright/test` devDependency (`pnpm add -D @playwright/test`) and install browser binaries (`pnpm exec playwright install chromium firefox webkit`).
- [x] **Step 1 — Approval & Dependency Installation:**
  - Install `@playwright/test` devDependency via `pnpm add -D @playwright/test`.
  - Install Playwright browser binaries via `pnpm exec playwright install chromium`.
- [x] **Step 2 — Configuration & Scripts:**
  - Create `playwright.config.ts`.
  - Add `"test:e2e"`, `"test:e2e:ui"`, and `"test:e2e:headed"` scripts to `package.json`.
- [x] **Step 3 — Mock Fixtures & Helpers:**
  - Implement `e2e/fixtures/mockAuth.ts` and `e2e/mockGoServer.mjs`.
  - Implement `e2e/fixtures/mockDashboardData.ts`.
- [x] **Step 4 — Test Suites Implementation:**
  - [x] `e2e/landing-and-auth.spec.ts` (Landing, hero, auth tabs, validation, forgot password modal).
  - [x] `e2e/two-factor-auth.spec.ts` (TOTP 2FA challenge and code verification).
  - [x] `e2e/gdpr-consent-gate.spec.ts` (Mandatory GDPR gate on `/dashboard` and `/profile`).
  - [x] `e2e/health-profile-wizard.spec.ts` (4-step onboarding wizard with live BMI preview).
  - [x] `e2e/dashboard-analytics.spec.ts` (Wellness score, radar infographic, trend charts, quick logs, AI insights regenerate, theme toggle, export).
  - [x] `e2e/profile-and-account-deletion.spec.ts` (Profile settings, 2FA setup/disable, typed "DELETE" modal).
  - [x] `e2e/password-reset.spec.ts` (`/new-password` form and token states).
- [x] **Step 5 — Verification & Gate Validation:**
  - Run Playwright E2E suite: `pnpm exec playwright test` -> 27/27 passed
  - Run Vitest suite: `pnpm test` -> 54/54 passed
  - Run TypeScript typecheck: `pnpm exec tsc --noEmit` -> 0 errors
  - Run ESLint: `pnpm lint` -> 0 errors

### Verification Commands (`AGENTS.md` §4)
- [x] `pnpm exec playwright test` (E2E browser tests: 27/27 passed)
- [x] `pnpm test` (Unit/integration tests in Vitest: 54/54 passed)
- [x] `pnpm exec tsc --noEmit` (TypeScript typecheck: 0 errors)
- [x] `pnpm lint` (ESLint checks: 0 errors)

---

## 5. Open Questions & Decisions Required

- [x] **Dependency Approval:** Per `AGENTS.md` §5, installing new packages requires explicit user authorization. Do you approve adding `@playwright/test` to `devDependencies` in `package.json`?
  - *(Approved)*: `@playwright/test` installed as devDependency.
- [x] **Browser Target Scope:** Should we install and run against all three engines (Chromium, Firefox, WebKit) + Mobile Chrome, or focus default CI execution on Chromium with optional multi-browser flags?
  - *(Approved)*: Configured all projects in `playwright.config.ts`; `pnpm test:e2e` defaults to `chromium` for fast developer feedback (`playwright test --project=chromium`), with `pnpm test:e2e:all` for full cross-browser validation.
- [x] **Execution Mode for Web Server:** Should Playwright launch `pnpm dev` (Turbopack) or run against a built production instance (`pnpm build && pnpm start`)?
  - *(Approved)*: Configured `pnpm dev` with `reuseExistingServer: true` and hermetic mock Go server daemon on port 8088.
