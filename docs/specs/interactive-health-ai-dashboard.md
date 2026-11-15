# Feature Spec: Interactive Health & AI Dashboard

- **Status:** Completed
- **Author:** Antigravity Agent (`/plan`)
- **Target Issue / PR:** N/A (Interactive Health & AI Dashboard)
- **Risk Level (`AGENTS.md` §10):** Medium (Frontend feature module, Zod v4 trust-boundary schemas, and `/api/health/*` integration across multiple files; no backend/GORM/AI formula changes)
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `app/dashboard/page.tsx`, `feature/dashboard/`, `lib/schemas/`, `tests/`
  - [ ] **Go Core API (`services/go-api/`)** — No changes required (integrates with existing `:8080` endpoints)
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — No changes required (consumed indirectly via Go API)

---

## 1. Objective & Scope

- **Problem Statement:**
  `app/dashboard/page.tsx` currently authenticates the user, enforces GDPR consent (`GdprConsentCard`) and health profile onboarding (`HealthProfileWizard`), and then renders a placeholder `<div>` with `userId` and `LogoutButton`. Users have no frontend interface to view their Composite Wellness Score, 4 component scores, milestone progress, 3-Layer AI Insights, weight/activity history, quick-log forms, or GDPR JSON export.
- **In Scope:**
  - **Zod v4 Schemas & Contract Helpers (`lib/schemas/dashboardSchema.ts`):**
    - Quick-log validation schemas (`weightLogSchema`, `activityLogSchema`) aligned with Go API physiological bounds (`20.0–350.0 kg` normalized, `1–1440` minutes, required ISO `recordedAt` for activity).
    - TypeScript types and safe JSON parsers (`parseMilestonesJson`, `parseRecommendationsJson`) for `HealthDashboardResponsePayload`, `WellnessSnapshot`, `AIInsightRecord`, `WeightEntry`, `ActivityEntry`, and `GDPRHealthDataExportResponsePayload`.
  - **Feature UI Module (`feature/dashboard/`):**
    - `HealthDashboardClient.tsx` — main interactive client container fetching `GET /api/health/dashboard` via relative `/api/health/dashboard` rewrite, managing loading/error/refresh states, top navigation bar (link to `/profile`, GDPR Export button, `LogoutButton`), and dashboard layout.
    - `WellnessScorePanel.tsx` — Composite Wellness Score (`0–100`), BMI + BMI classification badge, and 4 component score bars (`bmiScore`, `activityScore`, `progressScore`, `habitsScore`) plus milestone progress cards (`weight-5pct-step`, `weekly-activity-target`, `daily-habits-consistency`).
    - `AiInsightsPanel.tsx` — 3-Layer AI Insights panel rendering `healthStatusAnalysis`, `progressEvaluation`, prioritized recommendations (`title`, `category`, `priority`, `action`, `expectedImpact`), `weeklySummary`, `monthlySummary`, `aiServiceAvailable` / `isCachedFallback` status indicator, `allowAiProcessing` disabled notice when applicable, and "Regenerate Insights" button (`POST /api/health/insights/regenerate`).
    - `WeightActivitySection.tsx` — Weight & Activity history views (with unit-aware display using `healthProfile.preferredWeightUnit`) and React Hook Form + Zod v4 quick-log forms (`POST /api/health/weight`, `POST /api/health/activity`) with `409 Conflict` duplicate timestamp handling.
    - `GdprExportButton.tsx` — Triggers `GET /api/health/export` and downloads `nutrisync-health-export-YYYY-MM-DD.json`.
  - **Thin App Router Page (`app/dashboard/page.tsx`):**
    - Keep `requireAuthentication()`, `!session.consented` (`GdprConsentCard`), and `!session.hasHealthProfile` (`HealthProfileWizard`) gates intact; replace the placeholder `<div>` with `<HealthDashboardClient userEmail={session.email} />`.
  - **Vitest Unit Tests (`tests/dashboardSchema.test.ts`):**
    - Test valid/invalid boundaries for `weightLogSchema` (`kg` and `lbs` conversion bounds), `activityLogSchema` (duration `1–1440`, required `recordedAt`), and safe JSON parsing (`parseMilestonesJson`, `parseRecommendationsJson`) on valid, empty, and malformed JSON strings.
- **Out of Scope:**
  - Modifying Go API handlers, GORM models, PostgreSQL schema, or Python AI pipeline.
  - Installing new npm charting libraries (`recharts`, etc.) — `package.json` has no charting library; we will build clean, accessible SVG/Tailwind visual gauges, progress bars, and trend sparklines using existing `lucide-react` and Tailwind v4.
  - Modifying `components/ui/` primitives (we will compose existing `button.tsx`, `input.tsx`, `label.tsx`, `select.tsx`, `tabs.tsx`, `textarea.tsx`, `separator.tsx`, `showToast.ts`, `LogoutButton.tsx`).
- **Acceptance Criteria:**
  - [ ] Authenticated, consented users with a health profile see the full Interactive Health & AI Dashboard at `/dashboard`.
  - [ ] `GET /api/health/dashboard` data renders Wellness Score (`0–100`), 4 component scores, BMI classification, and milestone progress cards.
  - [ ] AI Insights panel displays analysis, evaluation, prioritized recommendations, weekly/monthly summaries, service/fallback status, and regenerates via `POST /api/health/insights/regenerate`.
  - [ ] Weight and activity quick-log forms validate with Zod v4, submit to `POST /api/health/weight` and `POST /api/health/activity` (handling `409 Conflict`), and refresh dashboard data on success.
  - [ ] GDPR export button fetches `GET /api/health/export` and downloads a `.json` file; header links to `/profile` and renders `LogoutButton`.
  - [ ] `pnpm exec tsc --noEmit`, `pnpm lint`, and `pnpm test` all pass.
- **Pre-Flight MCP & Context Verification:**
  - [x] Checked live Go API contracts in `services/go-api/handlers/health_profile_handler.go` and `services/go-api/models/`.
  - [x] Checked `next.config.ts` rewrites (`/api/health/:path*`, `/api/auth/*`).
  - [x] Checked Next.js 16.4.2 Server/Client Component docs in `node_modules/next/dist/docs/` and Zod v4.6.7 docs via `context7` MCP.

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
- **Tables / Models Affected:** None (read/write exclusively through existing Go API endpoints).

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints Consumed (via `next.config.ts` `/api/health/*` rewrite to `:8080`):**
  - `GET /api/health/dashboard` (`200 OK` | `401` | `404`) -> `HealthDashboardResponsePayload`
  - `POST /api/health/insights/regenerate` (`200 OK` | `401` | `404`) -> `RegenerateAIInsightsResponsePayload`
  - `POST /api/health/weight` (`201 Created` | `400` | `401`) -> `WeightEntry`
  - `POST /api/health/activity` (`201 Created` | `400` | `401` | `409`) -> `ActivityEntry`
  - `GET /api/health/export` (`200 OK` | `401`) -> `GDPRHealthDataExportResponsePayload`

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- No direct changes. Frontend renders `WellnessSnapshot` and `AIInsightRecord` structures produced by `services/py-ai` and persisted/returned by `services/go-api`.

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `lib/schemas/`)
- **Schemas (`lib/schemas/dashboardSchema.ts`):**
  - `ACTIVITY_INTENSITY_OPTIONS = ["low", "moderate", "high", "vigorous"] as const`
  - `weightLogSchema`: `{ weight: number > 0, unit: z.enum(WEIGHT_UNIT_OPTIONS), recordedAt?: string }` with `.superRefine()` enforcing `convertWeightToKg(weight, unit)` within `[20, 350]` kg.
  - `activityLogSchema`: `{ exerciseType: string (min 1, max 64), durationMinutes: int (1..1440), intensity: z.enum(ACTIVITY_INTENSITY_OPTIONS), notes?: string (max 500), recordedAt: string (non-empty valid date/ISO string) }`.
  - Safe JSON parsers `parseMilestonesJson(raw: string)` and `parseRecommendationsJson(raw: string)` returning typed arrays with `try/catch` fallback to `[]`.
- **Components (`feature/dashboard/`):**
  - `HealthDashboardClient.tsx` (`"use client"`): Fetches `/api/health/dashboard` on mount (`withCredentials: true`), handles `401` redirect to `/`, `404` fallback message, and passes refresh callbacks to child panels.
  - `WellnessScorePanel.tsx`: Composite score ring/gauge, BMI & classification badge, 4 component bars, and milestone cards.
  - `AiInsightsPanel.tsx`: Analysis, progress evaluation, prioritized recommendations (sorted/badged by `priority`), weekly/monthly summaries, `aiServiceAvailable` & `isCachedFallback` badges, and Regenerate button.
  - `WeightActivitySection.tsx`: `Tabs` (`components/ui/tabs.tsx`) switching between Weight and Activity views, each with a quick-log `react-hook-form` + `zodResolver` form and chronological history list/table.
  - `GdprExportButton.tsx`: Calls `GET /api/health/export` and triggers JSON download.
- **Route (`app/dashboard/page.tsx`):**
  - Server Component keeping `requireAuthentication()` + consent/profile gates and rendering `<HealthDashboardClient userEmail={session.email} />`.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **User Isolation & Authentication (`AGENTS.md` §6):**
  - `app/dashboard/page.tsx` validates JWT/session server-side via `requireAuthentication()`.
  - All client `axios` requests use relative `/api/health/*` paths with `withCredentials: true` so the Go API derives `userID` strictly from the `HttpOnly` `access_token` cookie. No `userId` is ever sent in request bodies or query parameters.
- **GDPR & Two-Tier PII Anonymization (`AGENTS.md` §6):**
  - `GdprConsentCard` remains gated before dashboard access.
  - `GdprExportButton` provides one-click access to `GET /api/health/export`.
  - No PII is sent to `POST /api/health/insights/regenerate` (empty body; Go API orchestrates anonymized payload).
- **Biometrics & AI Safety Guardrails (`AGENTS.md` §7):**
  - Client Zod v4 schemas mirror Go physiological bounds (`20–350 kg`, `1–1440 min`) before submission.
  - AI fallback state (`isCachedFallback`, `!aiServiceAvailable`) is surfaced transparently in `AiInsightsPanel` without blocking the rest of the dashboard.
- **Failure Modes & Safe Error Handling (`AGENTS.md` §9):**
  - Uses `showAPIResponseToast` (`components/ui/showToast.ts`) and inline form error messages; handles `401` (redirect to `/`), `409` (duplicate activity timestamp), and `5xx`/network errors cleanly without exposing raw stack traces.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Zod v4 Schemas & Types:** Create `lib/schemas/dashboardSchema.ts` with `weightLogSchema`, `activityLogSchema`, TypeScript interfaces matching Go payloads, and `parseMilestonesJson` / `parseRecommendationsJson`.
- [x] **Step 2 — Vitest Unit Tests:** Create `tests/dashboardSchema.test.ts` covering valid/invalid `weightLogSchema` (`kg`/`lbs`), `activityLogSchema` (`1–1440` min, required `recordedAt`), and JSON parser edge cases.
- [x] **Step 3 — Dashboard Feature Components:** Create `feature/dashboard/WellnessScorePanel.tsx`, `feature/dashboard/AiInsightsPanel.tsx`, `feature/dashboard/WeightActivitySection.tsx`, `feature/dashboard/GdprExportButton.tsx`, and `feature/dashboard/HealthDashboardClient.tsx`.
- [x] **Step 4 — Wire Thin App Router Page:** Update `app/dashboard/page.tsx` to render `<HealthDashboardClient userEmail={session.email} />` after auth/consent/profile gates.
- [x] **Step 5 — Verify Frontend:** Run `./.agents/skills/create-nextjs-feature/scripts/verify-frontend.sh` (`tsc --noEmit`, `eslint`, `vitest`).

### Verification Commands (`AGENTS.md` §4)
- [x] **Frontend Typecheck:** `pnpm exec tsc --noEmit`
- [x] **Frontend Lint:** `pnpm lint`
- [x] **Frontend Unit Tests:** `pnpm test`

---

## 5. Open Questions & Decisions Required

- [ ] **Weight Quick-Log Timestamp Input:** `POST /api/health/weight` accepts an optional `recordedAt` (defaulting to current UTC time on the Go server if omitted), whereas `POST /api/health/activity` requires `recordedAt`. Should the Weight quick-log form default to "Log for now" (omitting `recordedAt` or pre-filling current local `datetime-local`) with an optional date/time picker? (Recommended: pre-fill `datetime-local` with current time for both forms, or make it optional on weight and required on activity).
- [ ] **Insight History Viewer:** `GET /api/health/dashboard` returns `insightHistory` (all past `AIInsightRecord` versions). Should the AI Insights panel show the latest insight by default with a compact collapsible/version-selector for `insightHistory`? (Recommended: yes, show latest insight prominently + a simple version selector/history drawer for prior versions).
