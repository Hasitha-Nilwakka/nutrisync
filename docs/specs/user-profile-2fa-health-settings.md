# Feature Spec: User Profile, TOTP 2FA Security & Health Profile Settings

- **Status:** Completed
- **Author:** Antigravity (`/plan`)
- **Target Issue / PR:** N/A
- **Risk Level (`AGENTS.md` §10):** High (touches authentication/2FA login challenge flow, protected route guard, and health/privacy settings)
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `app/profile/page.tsx`, `feature/profile/`, `feature/auth/login/login.tsx`, `lib/schemas/`, `tests/`
  - [ ] **Go Core API (`services/go-api/`)** — No changes required (uses existing endpoints)
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — No changes required

---

## 1. Objective & Scope

- **Problem Statement:**
  1. `app/profile/page.tsx` is currently unauthenticated and only renders a non-functional wireframe `EnableTwoFactorButton` stub.
  2. Users have no UI to set up, enable, or disable TOTP Two-Factor Authentication (2FA), and `feature/auth/login/login.tsx` does not handle the `require2FA` + `ticket` challenge response from `POST /api/auth/login`, which would silently redirect to `/dashboard` without an `access_token` cookie when 2FA is enabled.
  3. Users have no dedicated settings page to view or update their Health Profile metrics, goals, dietary preferences/restrictions, privacy toggles (`allowAiProcessing`, `publicProfileVisible`, `emailNotifications`), or export their GDPR data from `/profile`.

- **In Scope:**
  - Server-side authentication guard on `app/profile/page.tsx` using `requireAuthentication()` (matching `app/dashboard/page.tsx`), passing session metadata (`email`, `twoFactorEnabled`, `consented`, `hasHealthProfile`) to a client container.
  - New Zod v4 schemas in `lib/schemas/twoFactorSchema.ts` (6-digit TOTP `code` validation, setup response, and login-verify payload) plus unit tests in `tests/twoFactorSchema.test.ts` and `tests/profileHelpers.test.ts`.
  - Full interactive TOTP 2FA Security card/dialog in `feature/profile/TwoFactorSecurityCard.tsx` (replacing `EnableTwoFactorButton.tsx`), wired to:
    - `POST /api/auth/2fa/setup` (renders QR code data URL `qrCode`, manual `secret`, and `uri`)
    - `POST /api/auth/2fa/enable` (validates 6-digit `code` via Zod v4)
    - `POST /api/auth/2fa/disable` (validates 6-digit `code` via Zod v4 when already enabled)
  - 2FA login challenge step in `feature/auth/login/login.tsx` handling `{ require2FA: true, ticket, message }` -> `POST /api/auth/2fa/login-verify`.
  - Health Profile & Privacy Settings form in `feature/profile/HealthProfileSettingsForm.tsx` wired to `GET /api/health/profile` and `PUT /api/health/profile`, reusing `healthProfileSchema` (`lib/schemas/healthProfileSchema.ts`), plus `GdprExportButton`, navigation back to `/dashboard`, and `LogoutButton`.

- **Out of Scope:**
  - Modifying any Go API handlers, GORM models, `services/py-ai` code, or `next.config.ts` (all required `/api/auth/session`, `/api/auth/2fa/:path*`, and `/api/health/:path*` rewrites already exist).
  - Adding new Shadcn UI primitive packages or third-party dependencies.

- **Acceptance Criteria:**
  - [ ] Unauthenticated requests to `/profile` redirect to `/` via `requireAuthentication()`.
  - [ ] Authenticated users can view their email, navigate back to `/dashboard`, and log out via `LogoutButton`.
  - [ ] Users with 2FA disabled can initiate setup (`POST /api/auth/2fa/setup`), view QR code + manual secret, and enable 2FA with a valid 6-digit TOTP code (`POST /api/auth/2fa/enable`).
  - [ ] Users with 2FA enabled can disable 2FA by entering a valid 6-digit TOTP code (`POST /api/auth/2fa/disable`).
  - [ ] Logging in with a 2FA-enabled account in `feature/auth/login/login.tsx` displays the 6-digit TOTP challenge step and completes authentication via `POST /api/auth/2fa/login-verify`.
  - [ ] Users with an existing health profile can load (`GET /api/health/profile`) and update (`PUT /api/health/profile`) metrics (converted cleanly between `cm`/`in` and `kg`/`lbs`), goals, dietary preferences/restrictions, and privacy toggles (`allowAiProcessing`, `publicProfileVisible`, `emailNotifications`), and export GDPR data via `GdprExportButton` (`GET /api/health/export`).
  - [ ] `pnpm exec tsc --noEmit`, `pnpm lint`, and `pnpm test` pass with zero errors.

- **Pre-Flight MCP & Context Verification:**
  - [x] Checked live Go API handler structs in `services/go-api/handlers/two_factor_handler.go`, `session_handler.go`, and `health_profile_handler.go`.
  - [x] Checked Next.js `16.4.2`, React `19.3.0`, Zod `4.6.7`, and `components/ui/` primitives (`button`, `checkbox`, `dialog`, `field`, `input`, `label`, `select`, `separator`, `showToast`, `tabs`, `textarea`, `LogoutButton`).
  - [x] Verified `next.config.ts` already rewrites `/api/auth/session`, `/api/auth/2fa/:path*`, and `/api/health/:path*` to `GO_API_URL`.

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
- **Tables / Models Affected:** `None` (no database or GORM changes required).

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
Consumes existing endpoints via Next.js `/api/*` rewrites:
- `GET /api/auth/session` -> `ActiveSessionResponsePayload`: `{ userId, sessionId, email, consented, twoFactorEnabled, hasHealthProfile }`
- `POST /api/auth/login` -> either `{ message }` or `TwoFactorRequiredLoginResponsePayload` `{ require2FA: true, ticket, message }`
- `POST /api/auth/2fa/setup` -> `TwoFactorSetupResponsePayload` `{ secret, uri, qrCode }`
- `POST /api/auth/2fa/enable` -> `{ code }` -> `{ message }`
- `POST /api/auth/2fa/disable` -> `{ code }` -> `{ message }`
- `POST /api/auth/2fa/login-verify` -> `{ ticket, code }` -> sets `access_token` / `refresh_token` cookies + `{ message }`
- `GET /api/health/profile` -> `HealthProfileResponsePayload` (normalized `heightCm`, `currentWeightKg`, `targetWeightKg`, `preferredHeightUnit`, `preferredWeightUnit`, `bmi`, `bmiClassification`, arrays, privacy booleans) or `404`
- `PUT /api/health/profile` -> `SaveHealthProfileRequestPayload` (aligned with `healthProfileSchema` in `lib/schemas/healthProfileSchema.ts`)
- `GET /api/health/export` -> `GDPRHealthDataExportResponsePayload` (consumed by existing `GdprExportButton`)

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- **Impact:** `None`.

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `lib/schemas/`)
- **Zod v4 Schemas (`lib/schemas/twoFactorSchema.ts`):**
  - `twoFactorCodeSchema`: `z.object({ code: z.string().trim().regex(/^\d{6}$/, "Enter a valid 6-digit authenticator code") })`
  - `twoFactorLoginVerifySchema`: `z.object({ ticket: z.string().min(1, "Verification ticket is missing"), code: z.string().trim().regex(/^\d{6}$/, "Enter a valid 6-digit authenticator code") })`
  - Exported types: `TwoFactorCodeFormData`, `TwoFactorLoginVerifyFormData`, `TwoFactorSetupPayload` (`{ secret: string; uri: string; qrCode: string }`), `TwoFactorRequiredLoginResponse` (`{ require2FA?: boolean; ticket?: string; message?: string }`).
- **Profile Unit Conversion Helper (`feature/profile/healthProfileMappers.ts`):**
  - Pure helper `mapHealthProfileResponseToFormData(profile: HealthProfileResponse): HealthProfileFormData` converting `heightCm` -> `in` (`Math.round((cm / 2.54) * 100) / 100`) when `preferredHeightUnit === "in"`, and `currentWeightKg`/`targetWeightKg` -> `lbs` (`Math.round((kg / 0.45359237) * 100) / 100`) when `preferredWeightUnit === "lbs"`.
- **Routes & Feature Modules:**
  - `app/profile/page.tsx`: Server Component calling `await requireAuthentication()`, rendering `<ProfilePageClient session={session} />`.
  - `feature/profile/ProfilePageClient.tsx`: Header with user email, 2FA status badge, Back to Dashboard `<Link href="/dashboard">`, `GdprExportButton`, and `LogoutButton`; renders `TwoFactorSecurityCard` and `HealthProfileSettingsForm` (or an onboarding CTA linking to `/dashboard` if `!session.hasHealthProfile` or `GET /api/health/profile` returns `404`).
  - `feature/profile/TwoFactorSecurityCard.tsx`: Manages 2FA enabled/disabled state, setup dialog (`Dialog` from `components/ui/dialog.tsx`) showing QR code `<img>` (`qrCode`), copyable `secret`, `otpauth://` `uri` link, and 6-digit verification form; plus disable 2FA dialog/form when `twoFactorEnabled` is true.
  - `feature/profile/HealthProfileSettingsForm.tsx`: React Hook Form + `zodResolver(healthProfileSchema)` pre-populated from `GET /api/health/profile` and submitting via `PUT /api/health/profile` with `showToast` feedback and live BMI preview (`calculateClientBMI`).
  - `feature/profile/EnableTwoFactorButton.tsx`: Delete (replaced by `TwoFactorSecurityCard.tsx`) or keep as thin re-export? **Plan:** Remove unused wireframe `EnableTwoFactorButton.tsx` once `app/profile/page.tsx` uses `ProfilePageClient`.
  - `feature/auth/login/login.tsx`: Update `onSubmit` to inspect `response.data?.require2FA && response.data?.ticket`, switch to a 2FA code input step validated with `twoFactorCodeSchema`, and submit `{ ticket, code }` to `POST /api/auth/2fa/login-verify` before redirecting to `/dashboard`.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **User Isolation & Authentication (`AGENTS.md` §6):**
  - `app/profile/page.tsx` enforces server-side authentication via `requireAuthentication()` (`GET /api/auth/session` with `access_token` cookie; redirects to `/` on failure).
  - All client Axios calls use relative `/api/*` paths with `withCredentials: true` and redirect to `/` on `401 Unauthorized`.
- **GDPR & Two-Tier PII Anonymization (`AGENTS.md` §6):**
  - `HealthProfileSettingsForm` exposes `allowAiProcessing`, `publicProfileVisible`, and `emailNotifications` toggles directly via `PUT /api/health/profile`, and `ProfilePageClient` includes `GdprExportButton` (`GET /api/health/export`).
- **Biometrics & AI Safety Guardrails (`AGENTS.md` §7):**
  - Reuses `healthProfileSchema` physiological bounds (`age` 13–120, `heightCm` 50–300, `weightKg` 20–500, `sleepHours` 1–24, `waterLiters` 0.1–15).
- **Failure Modes & Safe Error Handling (`AGENTS.md` §9):**
  - Surfaces structured API `error` / `message` strings via `FieldError` and `showToast` without exposing stack traces or tokens; `secret` and `qrCode` in 2FA setup are held only in component state during setup and cleared on dialog close/success.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Schemas & Pure Mappers:**
  - Create `lib/schemas/twoFactorSchema.ts` (`twoFactorCodeSchema`, `twoFactorLoginVerifySchema`, and TypeScript types).
  - Create `feature/profile/healthProfileMappers.ts` (`mapHealthProfileResponseToFormData`).
- [x] **Step 2 — 2FA Login Challenge in `feature/auth/login/login.tsx`:**
  - Wire `require2FA` + `ticket` state transition and `POST /api/auth/2fa/login-verify` submission with Zod v4 validation.
- [x] **Step 3 — Profile Feature Components (`feature/profile/`):**
  - Create `feature/profile/TwoFactorSecurityCard.tsx` (setup QR/secret/URI + enable + disable dialogs).
  - Create `feature/profile/HealthProfileSettingsForm.tsx` (`GET` + `PUT /api/health/profile` with unit conversion, goals, dietary preferences/restrictions, and privacy toggles).
  - Create `feature/profile/ProfilePageClient.tsx` (navigation back to `/dashboard`, `LogoutButton`, `GdprExportButton`, `TwoFactorSecurityCard`, `HealthProfileSettingsForm`).
  - Remove wireframe `feature/profile/EnableTwoFactorButton.tsx`.
- [x] **Step 4 — Protected Route (`app/profile/page.tsx`):**
  - Update `app/profile/page.tsx` to call `requireAuthentication()` and render `<ProfilePageClient session={session} />`.
- [x] **Step 5 — Vitest Unit Tests (`tests/`):**
  - Add `tests/twoFactorSchema.test.ts` (valid 6-digit codes, non-digit/length rejections, whitespace trimming, missing ticket rejection).
  - Add `tests/profileHelpers.test.ts` (metric/imperial round-trip mapping in `mapHealthProfileResponseToFormData`).

### Verification Commands (`AGENTS.md` §4)
- [x] **Frontend Typecheck:** `pnpm exec tsc --noEmit`
- [x] **Frontend Lint:** `pnpm lint`
- [x] **Frontend Unit Tests:** `pnpm test`

---

## 5. Open Questions & Decisions Required

- [x] **GDPR Consent Gate on `/profile`:** Approved: allow 2FA security management and GDPR data export on `/profile`, and show an onboarding CTA linking to `/dashboard` if `!session.consented` or `!session.hasHealthProfile`.
- [x] **Removing `EnableTwoFactorButton.tsx`:** Approved: removed `feature/profile/EnableTwoFactorButton.tsx` once replaced by `TwoFactorSecurityCard.tsx`.
