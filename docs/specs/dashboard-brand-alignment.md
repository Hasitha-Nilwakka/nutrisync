# Feature Spec: Dashboard Brand Alignment & Section Separation

- **Status:** Completed
- **Author:** Antigravity
- **Target Issue / PR:** N/A (Local branch `ui-improvement` follow-up)
- **Risk Level (`AGENTS.md` §10):** Low (Isolated UI styling, brand header, and section separation on `/dashboard`; no API, auth, or schema changes)
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `feature/dashboard/HealthDashboardClient.tsx`, `WellnessScorePanel.tsx`, `AiInsightsPanel.tsx`, `WeightActivitySection.tsx`, `tests/`
  - [ ] **Go Core API (`services/go-api/`)** — No changes
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — No changes

---

## 1. Objective & Scope

- **Problem Statement:**
  `app/dashboard/page.tsx` (`HealthDashboardClient`) still uses a plain text header without the NutriSync SVG logo, lacks the frosted sticky header and subtle radial backdrop established on `/`, and stacks `WellnessScorePanel`, `AiInsightsPanel`, and `WeightActivitySection` with uniform borders and minimal spacing, making the three detail sections blend together.

- **In Scope:**
  - Mirror the `/` (`LandingHeader.tsx`) sticky frosted header in `HealthDashboardClient.tsx` with `<NutriSyncBrand size="sm" />`, quick section jump links (`Overview`, `AI Insights`, `Tracking`), and existing dashboard action buttons (`Refresh`, `GdprExportButton`, `Profile`, `LogoutButton`).
  - Add the subtle logo-matched radial gradient backdrop and a concise dashboard intro strip (`NutriSync Health Dashboard` + `Signed in as {userEmail}`).
  - Create clear, restrained visual separation between the three detail sections (`#overview`, `#insights`, `#tracking`) using numbered section eyebrows (`01 · Wellness Overview`, `02 · AI Health Insights`, `03 · Weight & Activity Log`), `border-t border-border/60` dividers, `gap-10` vertical rhythm, and `rounded-2xl border-border/80 shadow-sm` card framing.
  - Add a quiet clinical disclaimer footer matching `LandingFooter.tsx`.
  - Add Vitest coverage in `tests/dashboardBrandLayout.test.ts` verifying `<NutriSyncBrand />` usage, section anchors (`#overview`, `#insights`, `#tracking`), and that `<ThemeToggle />` remains scoped to `/`.

- **Out of Scope:**
  - Adding new charts, KPI widgets, filters, backend endpoints, or altering Wellness Score / AI Insights / Weight-Activity logic.
  - Adding `<ThemeToggle />` outside `/` (preserved per `front-page-brand-theme-redesign.md`).

- **Acceptance Criteria:**
  - [x] Dashboard header renders `<NutriSyncBrand size="sm" />` in a sticky frosted bar matching `LandingHeader.tsx`.
  - [x] The three detail sections (`WellnessScorePanel`, `AiInsightsPanel`, `WeightActivitySection`) have clear visual separation (`#overview`, `#insights`, `#tracking`) without distracting visual clutter.
  - [x] `pnpm exec tsc --noEmit`, `pnpm lint`, and `pnpm test` pass.

---

## 2. Polyglot Technical Design

### 2.1–2.3 Backend & AI
- No changes.

### 2.4 Frontend UI (`feature/dashboard/`)
- **`HealthDashboardClient.tsx`:**
  - Sticky `<header className="sticky top-0 z-40 w-full border-b border-border/70 bg-background/85 backdrop-blur-md">` with `<NutriSyncBrand size="sm" />`, `md:flex` anchor links (`#overview`, `#insights`, `#tracking`), and existing action buttons.
  - Subtle radial gradient backdrop (`#0B74B8` / `#18A999`) matching `/` and `/verify-email`.
  - Restrained section wrappers with `scroll-mt-24`, numbered eyebrows, and `border-t border-border/60 pt-8` separators between sections.
- **`WellnessScorePanel.tsx`, `AiInsightsPanel.tsx`, `WeightActivitySection.tsx`:**
  - Align outer card containers to `rounded-2xl border border-border/80 bg-card p-6 shadow-sm` (matching `/` and auth cards) without changing internal functionality or copy.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- Preserves server-side session verification in `app/dashboard/page.tsx` and `401` redirect handling in `HealthDashboardClient.tsx`.
- Preserves GDPR export button (`GdprExportButton`) and clinical disclaimer wording.

---

## 4. Implementation & Verification Checklist

- [x] **Step 1 — Dashboard Header & Section Separation:** Update `HealthDashboardClient.tsx` with sticky `<NutriSyncBrand />` header, backdrop, section dividers (`#overview`, `#insights`, `#tracking`), and quiet footer.
- [x] **Step 2 — Card Surface Alignment:** Align outer card borders/radius (`rounded-2xl border-border/80 shadow-sm`) in `WellnessScorePanel.tsx`, `AiInsightsPanel.tsx`, and `WeightActivitySection.tsx`.
- [x] **Step 3 — Unit Tests & Verification:** Add `tests/dashboardBrandLayout.test.ts` and run `pnpm exec tsc --noEmit`, `pnpm lint`, and `pnpm test`.

---

## 5. Open Questions & Decisions Required

- None (restrained styling alignment following `/` patterns).
