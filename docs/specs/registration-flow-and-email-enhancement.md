# Feature Spec: End-to-End Registration Flow & Verification Email Enhancement

- **Status:** Completed
- **Author:** Antigravity
- **Target Issue / PR:** N/A (Local branch `ui-improvement` follow-up)
- **Risk Level (`AGENTS.md` §10):** High (Registration handler, email verification UX, case-insensitive email lookup, SMTP email template)
- **Approval Gate:** Approved by User (`/verify-email` + SMTP failure cleanup)
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `feature/auth/register/Register.tsx`, `app/verify-email/page.tsx` (optional/recommended), `tests/`
  - [x] **Go Core API (`services/go-api/`)** — `handlers/registration_handler.go`, `repositories/user_registration_repository.go`, `auth/verification_email_sender.go`, `*_test.go`
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — No changes required

---

## 1. Objective & Scope

- **Problem Statement:**
  An end-to-end audit of the registration flow (`Register.tsx` → `next.config.ts` → `POST /api/auth/register` → `verification_email_sender.go` → `GET /api/auth/verify`) revealed five UX, reliability, and email-quality gaps:
  1. **Browser `alert()` & missing form reset (`feature/auth/register/Register.tsx`):** Successful registration triggers `alert('user registered')` instead of `showAPIResponseToast`, leaves form inputs populated, and does not show toast feedback on API errors.
  2. **Raw JSON on email link click (`GET /api/auth/verify`):** Clicking the verification link in the email opens `/api/auth/verify?token=...` in the browser and renders raw JSON (`{"type":"success","title":"Email Verified","message":"user verified"}`) instead of a branded NutriSync confirmation page with a "Log In" CTA.
  3. **Case-sensitive email registration/lookup (`registration_handler.go`, `user_registration_repository.go`):** `HandleUserRegistrationRequest` only trims whitespace without lowercasing `email`, and `FindUserByEmailAddress` uses `WHERE email = ?`, risking case-mismatch issues with login/password-reset flows.
  4. **Orphaned unverified user on SMTP failure:** If `CreateUnverifiedUserRecord` succeeds but `SendRegistrationVerificationEmail` fails, the user cannot re-register (`409 email already registered`).
  5. **Bare verification email (`verification_email_sender.go`):** Sends a single `<p>` HTML snippet with no `multipart/alternative` plain-text fallback, no NutriSync branding, no 24-hour expiration notice, no fallback URL, and no CRLF header sanitization.

- **In Scope:**
  - Replace `alert()` in `feature/auth/register/Register.tsx` with `showAPIResponseToast`, form `reset()`, and clean error handling.
  - Add a `/verify-email` frontend page (`app/verify-email/page.tsx` + `feature/auth/register/VerifyEmailStatus.tsx`) or HTML redirect from `GET /api/auth/verify` so clicking the email button lands on a branded NutriSync confirmation screen.
  - Normalize registration emails to lowercase in `registration_handler.go` and `FindUserByEmailAddress` (`LOWER(email) = ?`).
  - Upgrade `BuildRegistrationVerificationEmailMessage` in `verification_email_sender.go` to `multipart/alternative` (plain-text + branded inline-CSS HTML matching the password reset email), with CRLF header sanitization, 24-hour expiry notice, fallback URL, and security notice.
  - Update unit tests in `services/go-api/auth/verification_email_sender_test.go`, `services/go-api/handlers/registration_handler_test.go`, `services/go-api/repositories/user_registration_repository_test.go`, and `tests/registeruser.test.ts`.

- **Out of Scope:**
  - Modifying GORM database schemas (`users` table already has all required columns).
  - Adding a separate "Resend verification email" UI button (can be a follow-up if desired).
  - Changing password rules or OAuth/2FA flows.

- **Acceptance Criteria:**
  - [x] Submitting valid registration details in `Register.tsx` shows a `Sonner` toast via `showAPIResponseToast` (no `alert()`), resets the form, and surfaces backend errors cleanly.
  - [x] Registration normalizes email to lowercase and checks `LOWER(email)` for duplicates.
  - [x] Verification emails are sent as `multipart/alternative` with plain-text + branded HTML, 24-hour expiry notice, fallback link, and CRLF-sanitized headers.
  - [x] Clicking the verification link in the email verifies the user and presents a clean NutriSync UI (not raw JSON) with a link back to Log In.
  - [x] All Go unit tests (`go test -v ./...`) and Frontend checks (`tsc --noEmit`, `pnpm lint`, `pnpm test`) pass.

- **Pre-Flight MCP & Context Verification:**
  - [x] Inspected `feature/auth/register/Register.tsx`, `services/go-api/handlers/registration_handler.go`, `services/go-api/repositories/user_registration_repository.go`, and `services/go-api/auth/verification_email_sender.go`.
  - [x] Verified `next.config.ts` already rewrites `/api/auth/register` and `/api/auth/verify` to `services/go-api`.

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/`)
- **Tables / Models Affected:** None (`models.User` already has `Email`, `Verified`, `VerifiedAt`, `VerificationToken`, `VerificationTokenExpiredAt`).

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **`services/go-api/auth/verification_email_sender.go`:**
  - Upgrade `BuildRegistrationVerificationEmailMessage` to produce `multipart/alternative` (`text/plain` + branded `text/html` matching `password_reset_email_sender.go`), sanitize `From`/`To` headers against `\r`/`\n`, and point the CTA to `${APP_URL}/verify-email?token=${token}` (or keep `/api/auth/verify?token=...` if redirecting — see Open Questions).
- **`services/go-api/handlers/registration_handler.go`:**
  - Normalize `Email` and `ConfirmEmail` via `strings.ToLower(strings.TrimSpace(...))`.
  - If `SendRegistrationVerificationEmail` fails after `CreateUnverifiedUserRecord`, either roll back the unverified user creation inside a transaction/cleanup or return a clear error (see Open Questions).
- **`services/go-api/repositories/user_registration_repository.go`:**
  - Update `FindUserByEmailAddress` to query `Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(emailAddress)))`.

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- No changes required.

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `tests/`)
- **`feature/auth/register/Register.tsx`:**
  - Replace `alert('user registered')` with `showAPIResponseToast(response.data)` (with fallback `APIResponse`), call `reset()`, and show `showAPIResponseToast` + `setError("root", ...)` on failure.
- **`app/verify-email/page.tsx` + `feature/auth/register/VerifyEmailStatus.tsx` (if `/verify-email?token=...` UX is approved):**
  - Wrapped in `<Suspense>` (Next.js 16 `cacheComponents: true` compliant).
  - Calls `GET /api/auth/verify?token=...` on mount and renders a branded NutriSync card showing verifying / verified / invalid-or-expired state with a `Back to Log In` button.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **Email Header Injection:** Strip `\r` and `\n` from `senderEmailAddress` and `recipientEmailAddress` before constructing MIME headers.
- **No Unauthenticated User Data Exposure:** `GET /api/auth/verify` returns only status `StandardAPIResponsePayload` (`type`, `title`, `message`) without PII or user records.
- **Bcrypt Cost 12 & 24h Token Expiry:** Preserved as-is.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Go Email & Repository Updates:** Upgrade `verification_email_sender.go` to `multipart/alternative` branded template + CRLF sanitization, lowercase email normalization in `registration_handler.go` & `user_registration_repository.go`.
- [x] **Step 2 — Frontend `Register.tsx` & `/verify-email` Page:** Replace `alert()` with `Sonner` toasts + `reset()`, add `/verify-email` page with `<Suspense>`.
- [x] **Step 3 — Unit Tests:** Update `verification_email_sender_test.go`, `registration_handler_test.go`, `user_registration_repository_test.go`, and `tests/registeruser.test.ts`.
- [x] **Step 4 — Verification:** Run `tsc --noEmit`, `pnpm lint`, `pnpm test`, and `go test -v ./...`.

---

## 5. Open Questions & Decisions Required

- [x] **Verification Link Destination (`/verify-email?token=...` vs `GET /api/auth/verify` redirect):** Approved `${APP_URL}/verify-email?token=...` keeping `GET /api/auth/verify` as a pure JSON API.
- [x] **SMTP Failure Rollback:** Approved deleting the newly inserted unverified user record if `SendRegistrationVerificationEmail` fails during `POST /api/auth/register`.
