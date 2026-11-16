# Feature Spec: End-to-End User Profile, GDPR Consent Withdrawal & Account Deletion

- **Status:** Completed
- **Author:** Antigravity
- **Target Issue / PR:** N/A (Main branch follow-up)
- **Risk Level (`AGENTS.md` §10):** High (Authenticated account/data deletion endpoint `DELETE /api/auth/account`, multi-table GORM transaction, GDPR consent gate on `/profile`, Next.js rewrite)
- **Approval Gate:** Approved by User (recommended: `DELETE /api/auth/account` + typed `DELETE` confirmation)
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `app/profile/page.tsx`, `feature/profile/ProfilePageClient.tsx`, `feature/profile/PrivacyAndAccountDeletionCard.tsx`, `next.config.ts`, `tests/`
  - [x] **Go Core API (`services/go-api/`)** — `main.go`, `handlers/session_handler.go`, `repositories/user_session_repository.go`, `*_test.go`
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — No changes

---

## 1. Objective & Scope

- **Problem Statement:**
  An end-to-end audit of `/profile` and GDPR consent handling revealed four gaps:
  1. **Consent gate bypass (`app/profile/page.tsx`):** Unlike `app/dashboard/page.tsx`, `app/profile/page.tsx` does not gate `!session.consented` users with `<GdprConsentCard />`.
  2. **No GDPR consent withdrawal or account deletion for consented users:** `GdprConsentCard.tsx` promises users can withdraw consent and delete their data at any time, and `POST /api/auth/consent` (`consented: false`) explicitly rejects consented users (`DeleteUnconsentedUserAndAccounts` errors when `Consented == true`) and does not delete health tables (`HealthProfile`, `WeightEntry`, `ActivityEntry`, `WellnessSnapshot`, `AIInsightRecord`).
  3. **No Danger Zone / Delete Account UI on `/profile`:** Users have no way on `/profile` to withdraw mandatory GDPR consent (with deletion confirmation prompt) or permanently delete their account/profile.
  4. **Header/Theme mismatch (`ProfilePageClient.tsx`):** `/profile` lacks the sticky frosted `<NutriSyncBrand size="sm" />` + `<ThemeToggle />` header and radial backdrop used on `/dashboard`.

- **In Scope:**
  - Enforce `if (!session.consented) return <GdprConsentCard email={session.email} />` in `app/profile/page.tsx`.
  - Add authenticated `DELETE /api/auth/account` endpoint in `services/go-api/handlers/session_handler.go` + `main.go` + `next.config.ts` rewrite (`/api/auth/account`).
  - Add `DeleteAuthenticatedUserAccountAndAllData(ctx, userID)` in `services/go-api/repositories/user_session_repository.go` wrapped in `db.Transaction(...)` deleting all user-scoped records (`AIInsightRecord`, `WellnessSnapshot`, `ActivityEntry`, `WeightEntry`, `HealthProfile`, `Account`, `RefreshToken`, `Session`, `PasswordReset`, `User`) and clearing session cookies.
  - Add `PrivacyAndAccountDeletionCard.tsx` on `/profile` (visible by default) with:
    - **Mandatory GDPR Consent toggle / withdrawal:** Attempting to withdraw mandatory GDPR consent prompts a confirmation modal explaining that mandatory GDPR consent is required to keep an account and offering **Confirm & Delete Account & Data** (or Cancel to keep consent).
    - **Default Delete Account & All Data button:** Opens the same explicit confirmation dialog (`DELETE` confirmation or explicit destructive action) to permanently delete the user's account and health data.
  - Align `ProfilePageClient.tsx` header & backdrop with `HealthDashboardClient.tsx` (`<NutriSyncBrand size="sm" />`, `<ThemeToggle />`, `Dashboard`, `GdprExportButton`, `LogoutButton`).
  - Unit & negative security tests (`401 Unauthorized` when unauthenticated, `405 Method Not Allowed`, full deletion verification in Go + Frontend tests).

- **Out of Scope:**
  - Changing initial `GdprConsentCard` behavior for unconsented users (`POST /api/auth/consent` remains for onboarding).
  - Modifying GORM database schemas (`models/`).

- **Acceptance Criteria:**
  - [x] Unconsented users visiting `/profile` are gated by `<GdprConsentCard />`.
  - [x] Consented users on `/profile` see a Privacy & Danger Zone card by default with both GDPR consent status (withdrawal prompts permanent deletion) and a direct "Delete Account & Profile" action.
  - [x] `DELETE /api/auth/account` derives `userID` strictly from the server-side `access_token` cookie + active `Session` (`401` if missing/invalid), deletes all user & health records atomically, clears auth cookies, and redirects to `/`.
  - [x] `ProfilePageClient.tsx` uses the sticky `<NutriSyncBrand />` + `<ThemeToggle />` header matching `/dashboard`.
  - [x] All Go (`go test -v ./...`) and Frontend (`tsc --noEmit`, `pnpm lint`, `pnpm test`) checks pass.

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/`)
- No schema edits. Deletes across existing tables scoped strictly by `user_id = ?` / `id = ?` inside `db.Transaction(...)`.

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **Endpoint:** `DELETE /api/auth/account` → `UserSessionHandler.HandleDeleteAuthenticatedUserAccountRequest`
  - Derives `userID` from validated `access_token` JWT cookie + active `Session` (`401 Unauthorized` if invalid/missing).
  - Calls `DeleteAuthenticatedUserAccountAndAllData(ctx, userID)`.
  - Clears `access_token` and `refresh_token` cookies via `auth.ClearAuthenticationCookies(w, isProduction)`.
  - Returns `200 OK` `{ "type": "success", "title": "Account Deleted", "message": "Your account and all health data have been permanently deleted." }`.

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- No changes.

### 2.4 Frontend UI & Rewrites (`app/profile/`, `feature/profile/`, `next.config.ts`)
- **`next.config.ts`:** Add rewrite `/api/auth/account` → `${goApiBaseUrl}/api/auth/account`.
- **`app/profile/page.tsx`:** Add `if (!session.consented) return <GdprConsentCard email={session.email} />`.
- **`feature/profile/PrivacyAndAccountDeletionCard.tsx`:**
  - Displays current mandatory GDPR consent status (`Granted` — required for NutriSync account) with a **"Withdraw Consent"** action that opens the confirmation dialog warning that withdrawing mandatory consent permanently deletes their profile & account.
  - Also displays a default **"Delete Account & Profile"** Danger Zone button opening the same confirmation dialog (`DELETE` typed confirmation or explicit confirm button) calling `DELETE /api/auth/account`, showing `showRedirectToast`, and redirecting to `/`.
- **`feature/profile/ProfilePageClient.tsx`:** Sticky frosted header with `<NutriSyncBrand size="sm" />`, `<ThemeToggle />`, `Dashboard` link, `GdprExportButton`, and `LogoutButton`, plus `PrivacyAndAccountDeletionCard`.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **Strict Server-Side Identity:** `userID` is derived exclusively from the validated `access_token` JWT and active `Session` row; client-supplied IDs/emails are never accepted.
- **Atomic Erasure (GDPR Art. 17):** All user health, AI insight, OAuth, session, and account rows are deleted inside a single `db.Transaction(...)`.
- **Negative Authorization Tests:** Go unit tests verify `401 Unauthorized` when cookies are missing/invalid and `405 Method Not Allowed` for non-DELETE methods.

---

## 4. Implementation & Verification Checklist

- [x] **Step 1 — Go Repository & Handler:** Add `DeleteAuthenticatedUserAccountAndAllData` in `user_session_repository.go`, `HandleDeleteAuthenticatedUserAccountRequest` in `session_handler.go`, and register `DELETE /api/auth/account` in `main.go`.
- [x] **Step 2 — Next.js Rewrite & Consent Gate:** Add `/api/auth/account` rewrite in `next.config.ts` and `GdprConsentCard` gate in `app/profile/page.tsx`.
- [x] **Step 3 — Profile UI & Danger Zone:** Create `PrivacyAndAccountDeletionCard.tsx` and update `ProfilePageClient.tsx` header/layout.
- [x] **Step 4 — Tests & Verification:** Add Go handler/repository tests + Vitest tests (`tests/profileHelpers.test.ts`), then run full Go and Frontend verification.

---

## 5. Open Questions & Decisions Required

- [x] **Account Deletion vs Health-Profile-Only Deletion:** Approved `DELETE /api/auth/account` (Delete Account & All Health Data).
- [x] **Confirmation Friction:** Approved typing `DELETE` to confirm in the Withdraw Consent / Delete Account modal.
