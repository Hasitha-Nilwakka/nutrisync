# Feature Spec: Post-Consent Demo Data Intercept Modal for Health Profile Onboarding

- **Status:** Completed
- **Author:** Antigravity Agent
- **Target Issue / PR:** N/A
- **Risk Level (`AGENTS.md` §10):** Medium
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `feature/health-profile/`, `lib/schemas/healthProfileSchema.ts`, `tests/healthProfileSchema.test.ts`
  - [ ] **Go Core API (`services/go-api/`)** — `handlers/`, `repositories/`, `models/`, `database/`, `auth/`, `health/`
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — `schemas/`, `analytics/`, `pipeline/`

---

## 1. Objective & Scope

- **Problem Statement:** When a newly registered user acknowledges GDPR data processing consent (`GdprConsentCard`) and enters the 4-Step Health Profile Onboarding Wizard (`HealthProfileWizard`), all 4 steps are currently prefilled with hardcoded sample values (`age: 28`, `height: 170`, `currentWeight: 70`, etc.). While prefilled dummy data makes portfolio testing fast for recruiters and reviewers, users who want to enter their own metrics from scratch have to manually overwrite or clear every prefilled field across 4 steps.
- **In Scope:**
  - Centralize the reusable preset constants (`SAMPLE_HEALTH_PROFILE_FORM_DATA` and `EMPTY_HEALTH_PROFILE_FORM_DATA`) in `lib/schemas/healthProfileSchema.ts`.
  - Add an onboarding intercept modal (`Dialog` from `components/ui/dialog.tsx`) inside `feature/health-profile/HealthProfileWizard.tsx` that automatically opens when the user arrives at the wizard after acknowledging GDPR consent.
  - Provide two explicit actions in the pop-up:
    1. **Continue with Sample Data:** Keeps/loads `SAMPLE_HEALTH_PROFILE_FORM_DATA` into the 4-step wizard (`reset(SAMPLE_HEALTH_PROFILE_FORM_DATA)`) so testers can immediately click through or tweak values.
    2. **Fill in My Own Details:** Clears the prefilled numeric/array defaults using `reset(EMPTY_HEALTH_PROFILE_FORM_DATA)` and `clearErrors()` so the user starts with clean inputs.
  - Also provide a subtle header/toolbar action inside `HealthProfileWizard` (e.g., `"Load Sample Data"` / `"Clear Form"`) so the user can change their mind at any time while on Step 1–4.
  - Add Vitest unit tests in `tests/healthProfileSchema.test.ts` verifying that `SAMPLE_HEALTH_PROFILE_FORM_DATA` passes `healthProfileSchema` validation and `EMPTY_HEALTH_PROFILE_FORM_DATA` properly requires user input before Step 1–4 submission.
- **Out of Scope:**
  - Backend database schema changes or automatic seeding of historical weight/activity entries in `services/go-api`.
  - Changes to `services/py-ai` analytics or 3-layer AI guardrails.
  - Changes to authentication or GDPR consent persistence (`POST /api/auth/consent`).
- **Acceptance Criteria:**
  - [ ] After a newly registered user accepts GDPR consent (`GdprConsentCard`), `HealthProfileWizard` renders and immediately displays a modal pop-up explaining that sample profile data has been prefilled for easy portfolio testing.
  - [ ] Clicking **"Continue with Sample Data"** closes the modal and leaves all 4 wizard steps populated with valid sample biometrics and preferences (`SAMPLE_HEALTH_PROFILE_FORM_DATA`).
  - [ ] Clicking **"Fill in My Own Details"** closes the modal and resets the wizard fields to clean/empty values (`EMPTY_HEALTH_PROFILE_FORM_DATA`) with no premature validation errors displayed until the user interacts or clicks "Next".
  - [ ] All frontend verification checks (`pnpm exec tsc --noEmit`, `pnpm lint`, `pnpm test`) pass with zero errors.
- **Pre-Flight MCP & Context Verification:**
  - [x] Checked live PostgreSQL schema via `postgres-readonly` MCP (not applicable — no DB/GORM changes).
  - [x] Checked version-accurate library docs via `node_modules/next/dist/docs/` and `components/ui/dialog.tsx` (`@base-ui/react/dialog` + `react-hook-form` + `zod` v4).
  - [x] Checked GitHub Issue / PR requirements via `github` MCP (not applicable).

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
- **Tables / Models Affected:** None.
- **Columns, Indexes & Constraints:** None.
- **Migration & Transaction Strategy:** None required.

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints (`:8080`):** None modified. `POST /api/health/profile` continues to receive the validated `HealthProfileFormData` payload upon Step 4 completion.
- **Repository & Business Logic:** None modified.
- **Go <-> Python AI Contract Sync:** None modified.

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- **Pydantic v2 Schemas (`schemas/analysis_schemas.py`):** None modified.
- **Vectorized Analytics (`analytics/wellness_engine.py`):** None modified.
- **3-Layer AI Pipeline (`pipeline/`):** None modified.

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `lib/schemas/`)
- **Zod v4 Schemas & Form Presets (`lib/schemas/healthProfileSchema.ts`):**
  - Export `SAMPLE_HEALTH_PROFILE_FORM_DATA: HealthProfileFormData` containing the realistic portfolio sample profile (`age: 28`, `height: 170`, `currentWeight: 70`, `targetWeight: 68`, `exerciseTypes: ["cardio"]`, `dietaryPreferences: ["balanced"]`, etc.).
  - Export `EMPTY_HEALTH_PROFILE_FORM_DATA: HealthProfileFormData` (using `NaN` or empty-compatible initial values via `Partial`/input state so numeric `<Input type="number" />` fields render blank `""` instead of `0` or `28`, and array selections start empty `[]`).
- **Feature Module (`feature/health-profile/HealthProfileWizard.tsx`):**
  - Add state `const [isDemoInterceptOpen, setIsDemoInterceptOpen] = useState<boolean>(true)` so the intercept dialog opens immediately after the user completes GDPR consent and lands on `HealthProfileWizard`.
  - Render `<Dialog open={isDemoInterceptOpen} onOpenChange={setIsDemoInterceptOpen}>` using `components/ui/dialog.tsx`:
    - **Title:** `Sample Profile Data Ready for Testing`
    - **Description:** Explain that because NutriSync is a portfolio showcase, we have prefilled a realistic health profile so recruiters and testers can explore the platform right away—or they can clear it and enter their own details from scratch.
    - **Primary Action (`handleContinueWithSampleData`):** Calls `reset(SAMPLE_HEALTH_PROFILE_FORM_DATA)`, `clearErrors()`, sets `currentStep(1)`, and closes the dialog.
    - **Secondary Action (`handleStartWithBlankProfile`):** Calls `reset(EMPTY_HEALTH_PROFILE_FORM_DATA)`, `clearErrors()`, sets `currentStep(1)`, and closes the dialog.
  - Add a compact quick-action button in the wizard header (`"Clear Prefilled Data"` / `"Reset Sample Data"`) so users can switch between sample data and blank inputs even after dismissing the modal.
- **API Integration:** Unchanged (`axios.post("/api/health/profile", data)`).

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **User Isolation & Authentication (`AGENTS.md` §6):**
  - `app/dashboard/page.tsx` continues to enforce `requireAuthentication()` and `if (!session.consented) return <GdprConsentCard />` before `HealthProfileWizard` is ever mounted.
- **GDPR & Two-Tier PII Anonymization (`AGENTS.md` §6):**
  - The intercept modal only appears **after** the user has explicitly accepted GDPR health data processing consent (`session.consented === true`).
  - No data is persisted to the backend until the user completes Step 4 and clicks `"Complete Profile"`.
- **Biometrics & AI Safety Guardrails (`AGENTS.md` §7):**
  - Whether the user starts from `SAMPLE_HEALTH_PROFILE_FORM_DATA` or `EMPTY_HEALTH_PROFILE_FORM_DATA`, every step (`STEP_FIELD_NAMES[1..4]`) and final submission is validated by `healthProfileSchema` (enforcing physiological bounds: height `80–260 cm`, weight `20–350 kg`, BMI `10–80`) before calling `POST /api/health/profile`.
- **Failure Modes & Safe Error Handling (`AGENTS.md` §9):**
  - Clearing the form resets validation errors cleanly via `clearErrors()` so the user is not greeted by a wall of red error messages before typing.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Form Presets & Schema Helpers:** Export `SAMPLE_HEALTH_PROFILE_FORM_DATA` and `EMPTY_HEALTH_PROFILE_FORM_DATA` in `lib/schemas/healthProfileSchema.ts`.
- [x] **Step 2 — Post-Consent Intercept Dialog in Wizard:** Update `feature/health-profile/HealthProfileWizard.tsx` to render the post-consent `Dialog` intercept with `"Continue with Sample Data"` and `"Fill in My Own Details"` handlers, plus a header quick-action to load/clear sample data.
- [x] **Step 3 — Unit Tests:** Update `tests/healthProfileSchema.test.ts` to test `SAMPLE_HEALTH_PROFILE_FORM_DATA` acceptance and `EMPTY_HEALTH_PROFILE_FORM_DATA` rejection on unpopulated submission.

### Verification Commands (`AGENTS.md` §4 — Only check off commands actually executed)
- [x] **Frontend Typecheck:** `pnpm exec tsc --noEmit`
- [x] **Frontend Lint:** `pnpm lint`
- [x] **Frontend Unit Tests:** `pnpm exec vitest run tests/healthProfileSchema.test.ts`

---

## 5. Open Questions & Decisions Required

- [x] **Scope of Sample Data:** Confirmed with user — only prefill/clear the 4-step `HealthProfileWizard` client-side form fields (no backend historical log seeding needed).
- [x] **Intercept Trigger Location:** Confirmed with user — display the modal pop-up when `HealthProfileWizard` opens immediately after GDPR consent is accepted.
