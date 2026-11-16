# Feature Spec: Front Page UI, Logo-Matched Color Scheme, and Light/Dark Mode

- **Status:** Completed
- **Author:** Antigravity
- **Target Issue / PR:** N/A (Front Page & Theme Redesign)
- **Risk Level (`AGENTS.md` §10):** Medium
- **Approval Gate:** Approved by User (2026-11-16)
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `app/`, `feature/`, `components/`, `public/`
  - [ ] **Go Core API (`services/go-api/`)** — No changes
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — No changes

---

## 1. Objective & Scope

- **Problem Statement:**
  1. The current root page (`app/page.tsx`) renders only a static hero image (`public/nutrisync-hero.jpg`) hidden on mobile/tablet (`hidden lg:block`) alongside `AuthTabs`, providing no product narrative, feature overview, or brand identity on smaller screens.
  2. `app/globals.css` uses generic neutral grayscale tokens and a conflicting `@media (prefers-color-scheme: dark)` `:root` override instead of a cohesive NutriSync brand palette.
  3. `next-themes` (`^0.4.6`) is installed and used by `components/ui/sonner.tsx`, but `ThemeProvider` and a user-facing Light/Dark mode toggle are not wired into `app/layout.tsx`.
- **In Scope:**
  - Add the user-provided NutriSync logo asset to `public/nutrisync-logo.jpg`.
  - Define a logo-matched Light & Dark mode OKLCH color palette in `app/globals.css` (Ocean Blue `#0B74B8`, Vibrant Teal `#18A999`, Botanical Green `#43B047`, Deep Slate Navy `#152432`).
  - Wire `next-themes` (`ThemeProvider` + `ThemeToggle`) in `app/layout.tsx` and expose the toggle in the front page header (plus optional placement in `/dashboard` and `/profile` headers).
  - Design and build the main front page (`app/page.tsx` + `feature/landing/*` components) with Hero + `AuthTabs` above the fold, Core Capabilities Bento Grid, 3-Step "How It Works" flow, and Footer with clinical disclaimer.
  - Add Vitest component/smoke tests in `tests/` for theme toggle and front page rendering.
- **Out of Scope:**
  - Changes to authentication endpoints, JWT/cookie handling, OAuth flows, 2FA, GORM models, or Python AI pipelines.
  - Rewriting internal `/dashboard` or `/profile` layouts beyond inheriting the updated CSS theme tokens and optional `ThemeToggle` button.
- **Acceptance Criteria:**
  - [ ] Logo asset is served from `public/nutrisync-logo.jpg` and visible on both mobile and desktop viewports.
  - [ ] Light and Dark themes switch cleanly via `ThemeToggle` (`next-themes` `class` strategy) with no hydration mismatch and no conflicting `@media (prefers-color-scheme: dark)` override.
  - [ ] All primary brand surfaces, buttons, borders, and chart tokens (`--primary`, `--secondary`, `--accent`, `--chart-1..5`) harmonize with the NutriSync logo palette and meet WCAG AA contrast.
  - [ ] `AuthTabs` (Login / Register / Google / GitHub OAuth) remains immediately accessible above the fold on desktop and mobile.
  - [ ] `pnpm exec tsc --noEmit`, `pnpm lint`, and `pnpm test` pass without errors.
- **Pre-Flight MCP & Context Verification:**
  - [x] Verified `package.json` (`next@16.4.2`, `react@19.3.0`, `next-themes@^0.4.6`, `lucide-react@^1.59.0`, `tailwindcss@^4`).
  - [x] Verified `app/globals.css` Tailwind v4 `@custom-variant dark (&:is(.dark *));` and `@theme inline` structure.
  - [x] Verified `components/ui/sonner.tsx` already consumes `useTheme()` from `next-themes`.

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
- **Tables / Models Affected:** None.

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints (`:8080`):** None added or modified. Existing `AuthTabs` continues to call `/api/auth/*` via `next.config.ts` rewrites.

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- **Impacts:** None.

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `components/`, `public/`)
- **Brand Assets (`public/`):**
  - `[NEW] public/nutrisync-logo.jpg`: Copy of the user-provided NutriSync logo (`NutriSync — Personalized AI Nutrition`).
- **Theme Infrastructure (`components/theme/`, `app/layout.tsx`, `app/globals.css`):**
  - `[NEW] components/theme/ThemeProvider.tsx`: Client wrapper around `NextThemesProvider` (`attribute="class"`, `defaultTheme="system"`, `enableSystem`, `disableTransitionOnChange`).
  - `[NEW] components/theme/ThemeToggle.tsx`: Accessible icon button toggling between `light` and `dark` themes using `useTheme()` with a `mounted` state guard.
  - `[MODIFY] app/layout.tsx`: Add `suppressHydrationWarning` to `<html>` and wrap `{children}` + `<Toaster />` in `<ThemeProvider>`.
  - `[MODIFY] app/globals.css`: Replace neutral grayscale `:root` and `.dark` variables with the NutriSync Ocean Blue / Teal / Botanical Green / Slate Navy palette; remove the conflicting `@media (prefers-color-scheme: dark)` `:root` block and hardcoded `Arial` body font.
- **Front Page Modules (`feature/landing/`, `app/page.tsx`):**
  - `[NEW] feature/landing/LandingHeader.tsx`: Sticky translucent header with NutriSync logo mark, anchor links (`Features`, `How It Works`, `Privacy`), and `<ThemeToggle />`.
  - `[NEW] feature/landing/HeroSection.tsx`: Responsive 60/40 desktop split (stacked on mobile) featuring the NutriSync logo showcase, value proposition headline, trust/metric pills (`0–100 Wellness Score`, `2-Tier PII Anonymization`, `GDPR Export`), and the elevated `AuthTabs` card.
  - `[NEW] feature/landing/FeaturesSection.tsx`: 3-card Bento grid highlighting (1) Deterministic Wellness Score Engine, (2) 3-Layer AI Nutrition Guardrails, and (3) Privacy, 2FA & GDPR Control.
  - `[NEW] feature/landing/HowItWorksSection.tsx`: 3-step onboarding-to-insights flow (`01. Profile & Goals`, `02. Daily Biometrics`, `03. Guardrailed AI Insights`).
  - `[NEW] feature/landing/LandingFooter.tsx`: Brand summary, anchor navigation, and non-diagnostic medical disclaimer.
  - `[MODIFY] app/page.tsx`: Compose `LandingHeader`, `HeroSection`, `FeaturesSection`, `HowItWorksSection`, and `LandingFooter` while keeping the route file thin.
- **Zod v4 Schemas & Forms (`lib/schemas/`):**
  - No schema changes required; existing `AuthTabs` schemas remain untouched.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **User Isolation & Authentication (`AGENTS.md` §6):** `AuthTabs` on `/` continues to authenticate through `/api/auth/*` HTTP-only cookies without client-side token storage.
- **GDPR & Two-Tier PII Anonymization (`AGENTS.md` §6):** Front page copy accurately describes NutriSync's existing GDPR consent (`POST /api/auth/consent`), data export (`GET /api/health/export`), and 2-tier PII stripping without inventing unimplemented features.
- **Biometrics & AI Safety Guardrails (`AGENTS.md` §7):** Front page copy accurately states the deterministic Wellness Score formula (`30% BMI + 30% Activity + 20% Progress + 20% Habits`) and includes a clear non-medical-advice disclaimer in the footer.
- **Failure Modes & Safe Error Handling (`AGENTS.md` §9):** `ThemeToggle` guards against SSR hydration mismatches and gracefully falls back if `localStorage` is unavailable.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [ ] **Step 1 — Brand Asset & Theme Tokens:** Copy logo to `public/nutrisync-logo.jpg`; update `app/globals.css` with Light & Dark OKLCH tokens matching the logo.
- [ ] **Step 2 — Theme Provider & Toggle:** Create `components/theme/ThemeProvider.tsx` and `components/theme/ThemeToggle.tsx`; wire into `app/layout.tsx` (and optionally `HealthDashboardClient.tsx` / `ProfilePageClient.tsx` headers).
- [ ] **Step 3 — Front Page Content & Components:** Implement `feature/landing/LandingHeader.tsx`, `HeroSection.tsx`, `FeaturesSection.tsx`, `HowItWorksSection.tsx`, and `LandingFooter.tsx`; update `app/page.tsx`.
- [ ] **Step 4 — Unit Tests:** Add Vitest tests in `tests/` for `ThemeToggle` and front page component structure.

### Verification Commands (`AGENTS.md` §4 — Only check off commands actually executed)
- [ ] **Frontend Typecheck:** `pnpm exec tsc --noEmit`
- [ ] **Frontend Lint:** `pnpm lint`
- [ ] **Frontend Unit Tests:** `pnpm test`

---

## 5. Open Questions & Decisions Required

- [ ] **Logo Framing in Dark Mode:** Because the uploaded logo (`uploaded_media_1794864746246.jpg`) has a light/white background, should we display it inside a clean rounded light badge/card in Dark Mode (fast, zero asset edits), or would you prefer a transparent-background PNG/SVG version?
- [ ] **Theme Toggle Scope:** Should `<ThemeToggle />` also be added to the `/dashboard` and `/profile` headers so users can switch themes after logging in, or only on the front page for now?
- [ ] **Existing Hero Image (`public/nutrisync-hero.jpg`):** Should we keep `nutrisync-hero.jpg` as a secondary visual in the "How It Works" or Hero section, or replace it with the new logo-led hero showcase?
