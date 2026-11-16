# Feature Spec: Dashboard Weight Chart, Activity Chart & User Health Infographic

- **Status:** Completed
- **Author:** Antigravity
- **Target Issue / PR:** N/A (Local branch `ui-improvement` follow-up)
- **Risk Level (`AGENTS.md` §10):** Low (Frontend-only pure SVG charts & infographic section using existing `HealthDashboardPayload` data; no new dependencies, no backend/schema changes)
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `feature/dashboard/charts/WeightTrendChart.tsx`, `feature/dashboard/charts/ActivityTrendChart.tsx`, `feature/dashboard/UserHealthInfographicSection.tsx`, `feature/dashboard/WeightActivitySection.tsx`, `feature/dashboard/HealthDashboardClient.tsx`, `tests/`
  - [ ] **Go Core API (`services/go-api/`)** — No changes
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — No changes

---

## 1. Objective & Scope

- **Problem Statement:**
  `WeightActivitySection.tsx` currently renders only a minimal 56px-tall `WeightSparkline` without axes, units, target weight context, or interactive tooltips; it has no chart for `activityHistory`; and `/dashboard` lacks a holistic user infographic summarizing biometrics, wellness balance, weight-goal progress, and daily habits in one view.

- **In Scope:**
  - **Pure SVG Charting (Zero New Dependencies):** Build responsive, Light/Dark theme-aware SVG charts without adding third-party packages (`AGENTS.md` §5).
  - **Improved Weight Trend Chart (`feature/dashboard/charts/WeightTrendChart.tsx`):**
    - Converts weights to `preferredWeightUnit` (`kg` / `lbs`).
    - Renders gradient area fill, dashed horizontal gridlines, X-axis date range, optional `Target` weight reference line (`targetWeightKg`), net delta summary, and hover/focus point tooltips.
  - **New Activity Chart (`feature/dashboard/charts/ActivityTrendChart.tsx`):**
    - Renders an intensity-coded bar chart of recent activity sessions (`durationMinutes`, colored by `low` / `moderate` / `high` / `vigorous`), Y-axis minute gridlines, total/average duration summary, intensity legend, and hover/focus tooltips.
  - **New Bottom Infographic Section (`04 · Health Infographic`, `#infographic`, `feature/dashboard/UserHealthInfographicSection.tsx`):**
    - Added as a separate section at the bottom of `HealthDashboardClient.tsx` with an `Infographic` link in the sticky header.
    - 3-column bento infographic combining:
      1. **Biometric & Goal Trajectory:** BMI spectrum bar (`Underweight` / `Normal` / `Overweight` / `Obese`) + Baseline → Current → Target weight progress bar.
      2. **4-Axis Wellness Balance Radar Chart (SVG):** `BMI (30%)`, `Activity (30%)`, `Progress (20%)`, `Habits (20%)` (`0–100`).
      3. **Habits & Activity Mix:** Sleep (`sleepHours`), Hydration (`waterLiters`), weekly activity days (`weeklyActivityDays` / `targetWeeklyDays`), proportional intensity mix bar, and goal/fitness context badges.
  - **Unit Tests:** Pure helper + layout unit tests in `tests/dashboardCharts.test.ts`.

- **Out of Scope:**
  - Installing external charting libraries (`recharts`, `chart.js`, etc.).
  - Modifying `services/go-api`, `services/py-ai`, or `HealthDashboardPayload` contracts.

- **Acceptance Criteria:**
  - [x] `WeightTrendChart` displays unit-aware (`kg`/`lbs`) weight trajectory, gridlines, optional target line, and interactive point inspection.
  - [x] `ActivityTrendChart` displays intensity-colored duration bars, summary stats, and interactive session inspection inside the Activity tab.
  - [x] `UserHealthInfographicSection` renders at the bottom of `/dashboard` (`#infographic`) as Section `04 · Health Infographic` with BMI spectrum, weight-goal progress, 4-axis wellness radar, and habits/activity mix.
  - [x] `pnpm exec tsc --noEmit`, `pnpm lint`, and `pnpm test` pass.

---

## 2. Polyglot Technical Design

### 2.1–2.3 Backend & AI
- No changes; all data comes from existing `HealthDashboardPayload` (`healthProfile`, `latestWellness`, `weightHistory`, `activityHistory`).

### 2.4 Frontend UI (`feature/dashboard/`)
- **`feature/dashboard/charts/WeightTrendChart.tsx`:** Pure SVG chart + exported pure helper `computeWeightTrendPoints(...)` for deterministic unit testing.
- **`feature/dashboard/charts/ActivityTrendChart.tsx`:** Pure SVG bar chart + exported pure helper `summarizeActivityBreakdown(...)` for deterministic unit testing.
- **`feature/dashboard/UserHealthInfographicSection.tsx`:** 3-column bento infographic + exported pure helpers `computeWeightGoalProgress(...)` and `computeBmiMarkerPercent(...)`.
- **`feature/dashboard/WeightActivitySection.tsx`:** Accepts optional `targetWeightKg` and `baselineWeightKg`, replaces `WeightSparkline` with `<WeightTrendChart />`, and adds `<ActivityTrendChart />` in the Activity tab.
- **`feature/dashboard/HealthDashboardClient.tsx`:** Passes `targetWeightKg`/`baselineWeightKg` to `WeightActivitySection`, adds `Infographic` (`#infographic`) to sticky nav, and renders Section `04 · Health Infographic` above the footer.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- Client-side rendering of already-authenticated `HealthDashboardPayload` only; no new network calls, no PII logged, and clinical disclaimer footer preserved.

---

## 4. Implementation & Verification Checklist

- [x] **Step 1 — Charts (`WeightTrendChart.tsx`, `ActivityTrendChart.tsx`):** Build pure SVG charts + exported data-shaping helpers.
- [x] **Step 2 — Infographic (`UserHealthInfographicSection.tsx`):** Build Section 04 infographic with BMI bar, weight progress, 4-axis SVG radar, and habits/activity mix.
- [x] **Step 3 — Dashboard Integration (`WeightActivitySection.tsx`, `HealthDashboardClient.tsx`):** Wire charts, `targetWeightKg`/`baselineWeightKg`, `#infographic` section, and header anchor.
- [x] **Step 4 — Unit Tests & Verification:** Add `tests/dashboardCharts.test.ts` and run `pnpm exec tsc --noEmit`, `pnpm lint`, and `pnpm test`.

---

## 5. Open Questions & Decisions Required

- None (zero new dependencies, pure SVG charts using existing dashboard payload).
