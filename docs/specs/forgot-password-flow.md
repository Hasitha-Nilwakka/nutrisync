# Feature Spec: Full Forgot Password Flow & Enhanced Email Structure

- **Status:** Completed
- **Author:** Antigravity
- **Target Issue / PR:** N/A (Local branch `ui-improvement` follow-up)
- **Risk Level (`AGENTS.md` §10):** High (Authentication flow, password hashing, session invalidation, Go API endpoints, SMTP email, Next.js rewrites)
- **Approval Gate:** Approved by User (defaults accepted)
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `app/new-password/page.tsx`, `feature/auth/resetpassword/`, `next.config.ts`, `tests/`
  - [x] **Go Core API (`services/go-api/`)** — `main.go`, `handlers/`, `repositories/`, `auth/`
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — No changes required

---

## 1. Objective & Scope

- **Problem Statement:**
  The "Forgot password?" dialog in `feature/auth/resetpassword/ResetPassword.tsx` posts to `/api/auth/passwordreset`, but the end-to-end flow is non-functional:
  1. `next.config.ts` lacks `/api/auth/passwordreset*` rewrites to `services/go-api`.
  2. `services/go-api` defines and migrates `models.PasswordReset` (`password_resets` table), but has no HTTP routes, handlers, repository methods, or email sender for password resets.
  3. `app/new-password/page.tsx` uses `useParams()` (without a `[token]` dynamic route or `<Suspense>` boundary required by Next.js 16 `cacheComponents: true`) and posts to a relative `'api/auth/passwordreset'` URL.
  4. Existing SMTP emails in `services/go-api/auth/verification_email_sender.go` use a single bare HTML `<p>` tag with no plain-text alternative, brand styling, expiry notice, or security guidance.

- **In Scope:**
  - Next.js rewrites in `next.config.ts` for `/api/auth/passwordreset` and `/api/auth/passwordreset/confirm`.
  - Go API endpoints:
    - `POST /api/auth/passwordreset` (request reset link, anti-enumeration response, 1-hour token creation, SMTP email dispatch).
    - `POST /api/auth/passwordreset/confirm` (validate token, update bcrypt password, mark token verified, revoke active sessions).
  - Repository layer (`services/go-api/repositories/password_reset_repository.go`) using the existing `models.PasswordReset` and `models.User` structs.
  - Enhanced `multipart/alternative` (plain-text + branded inline-CSS HTML) password reset email builder and Gmail SMTP sender in `services/go-api/auth/password_reset_email_sender.go`.
  - Refactored `app/new-password/page.tsx` + `feature/auth/resetpassword/NewPasswordForm.tsx` with `<Suspense>`, `?token=` query parsing, missing-token UI, and `POST /api/auth/passwordreset/confirm` integration.
  - Go unit tests (`*_test.go`) and Frontend Vitest tests (`tests/`).

- **Out of Scope:**
  - Modifying GORM table schemas (`models.User`, `models.PasswordReset` already match requirements).
  - Changing login error messages or introducing email enumeration.
  - Adding rate-limiting middleware or Redis-backed throttling (noted as future hardening).
  - Refactoring existing registration verification email tests unless additive.

- **Acceptance Criteria:**
  - [x] Submitting a registered email in `ResetPassword.tsx` triggers `POST /api/auth/passwordreset`, creates a SHA-256 hashed, 1-hour `PasswordReset` record (revoking prior unverified tokens for that user), and sends a `multipart/alternative` email linking to `${APP_URL}/new-password?token=<rawToken>`.
  - [x] Submitting an unregistered or OAuth-only email returns `200 OK` with the same non-enumerating success message without sending an email or leaking account existence.
  - [x] Visiting `/new-password?token=<rawToken>` renders `NewPasswordForm` inside `<Suspense>` (passing Next.js 16 build/typecheck), and submitting a valid matching password updates `users.password` (bcrypt cost 12), marks the token `verified = true`, revokes active `sessions`, and redirects to `/`.
  - [x] Expired, already-used, revoked, or malformed tokens return `400 Bad Request` with a safe `StandardAPIResponsePayload` error.
  - [x] All Go unit tests (`go test -v ./...`) and Frontend checks (`tsc --noEmit`, `pnpm lint`, `pnpm test`) pass.

- **Pre-Flight MCP & Context Verification:**
  - [x] Checked live PostgreSQL MCP (`postgres-readonly` returned `password authentication failed for user "app"`); verified `users` and `password_resets` schemas directly in `services/go-api/models/user.go` and `services/go-api/database/postgres.go`.
  - [x] Checked Next.js 16 `useSearchParams` / `useParams` docs in `node_modules/next/dist/docs/` (`cacheComponents: true` requires `<Suspense>` around `useSearchParams()`).
  - [ ] Checked GitHub Issue / PR requirements (N/A — direct user request).

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
> **Rule (`AGENTS.md` §8):** PostgreSQL is owned exclusively by `services/go-api` via GORM.
- **Tables / Models Affected:** None (reuses existing `models.User`, `models.PasswordReset`, and `models.Session` already migrated by `database.RunDatabaseMigrations`).
- **Columns, Indexes & Constraints (Existing):**
  - `password_resets`: `id` (uuid PK), `user_id` (uuid, `not null`, `index`, FK → `users.id` `ON DELETE CASCADE`), `verification_token` (`not null`, `index` — will store SHA-256 hex digest of the 32-byte random token), `verification_token_expiry` (`not null`), `verified` (`default:false`, `not null`), `revoked_at` (`*time.Time`), `created_at`.
- **Migration & Transaction Strategy:**
  - `CreatePasswordResetToken` runs in `db.Transaction(...)`: sets `revoked_at = NOW()` on any existing unverified, unrevoked `password_resets` rows for `user_id`, then inserts the new `PasswordReset` row.
  - `CompletePasswordReset` runs in `db.Transaction(...)`: updates `users.password`, marks the `PasswordReset` row `verified = true` and `revoked_at = NOW()`, and deletes active `sessions` rows for `user_id`.

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints (`:8080`):**
  1. `POST /api/auth/passwordreset` → `PasswordResetHandler.HandleRequestPasswordReset`
     - Request: `{ "emailReset": "user@example.com" }`
     - Response (`200 OK`): `{ "type": "success", "title": "Password Reset Link Sent", "message": "If an account exists for that email, a password reset link has been sent." }`
     - Response (`400 Bad Request`): `{ "type": "error", "title": "Invalid Email", "message": "Please enter a valid email address." }`
  2. `POST /api/auth/passwordreset/confirm` → `PasswordResetHandler.HandleConfirmPasswordReset`
     - Request: `{ "token": "<raw-hex-token>", "newPassword": "<min-8-chars>" }`
     - Response (`200 OK`): `{ "type": "success", "title": "Password Updated", "message": "Your password has been reset. You can now log in with your new password." }`
     - Response (`400 Bad Request`): `{ "type": "error", "title": "Invalid or Expired Link", "message": "This password reset link is invalid or has expired. Please request a new one." }` (or validation error if password `< 8` chars).
- **Repository & Business Logic:**
  - `services/go-api/repositories/password_reset_repository.go`:
    - `PasswordResetRepository` interface + `GormPasswordResetRepository` with single-job methods:
      - `FindUserByEmailForPasswordReset(ctx, email)`
      - `ReplaceActivePasswordResetToken(ctx, userID, tokenHash, expiresAt)`
      - `FindPasswordResetByTokenHash(ctx, tokenHash)`
      - `CompletePasswordResetAndRevokeSessions(ctx, resetID, userID, newPasswordHash, completedAt)`
  - `services/go-api/auth/password_reset_tokens.go`:
    - `GeneratePasswordResetTokenAndExpiry(ttl time.Duration) (rawToken string, tokenHash string, expiresAt time.Time, err error)`
    - `HashPasswordResetToken(rawToken string) string` (SHA-256 hex digest)
  - `services/go-api/auth/password_reset_email_sender.go`:
    - `PasswordResetEmailSender` interface + `GmailSMTPPasswordResetEmailSender`
    - `BuildPasswordResetEmailMessage(senderEmailAddress, recipientEmailAddress, applicationBaseURL, rawResetToken) ([]byte, error)` — builds a `multipart/alternative` MIME message with CRLF header-injection sanitization, plain-text fallback, and branded inline-CSS HTML (`NutriSync`, `#0B74B8`→`#18A999`→`#43B047` accent bar, `Reset Password` CTA button, 1-hour expiry notice, fallback URL, and security disclaimer).
  - `services/go-api/handlers/password_reset_handler.go`:
    - `PasswordResetHandler` wired in `services/go-api/main.go` (`CreateApplicationMux`).
- **Go <-> Python AI Contract Sync:** None.

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- No changes required.

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `lib/schemas/`, `next.config.ts`)
- **Rewrites (`next.config.ts`):**
  - Add `/api/auth/passwordreset` → `${goApiBaseUrl}/api/auth/passwordreset`
  - Add `/api/auth/passwordreset/confirm` → `${goApiBaseUrl}/api/auth/passwordreset/confirm`
- **Routes & Feature Modules:**
  - `app/new-password/page.tsx`: Thin Server/Client wrapper rendering `<Suspense>` around `<NewPasswordForm />` inside a centered layout matching the NutriSync brand.
  - `feature/auth/resetpassword/NewPasswordForm.tsx`: Client component reading `token` via `useSearchParams()`, validating with `passwordChangeSchema` (`mode: "onTouched"`), showing a clear error card with a link back to `/` when `token` is missing, and posting `{ token, newPassword }` to `/api/auth/passwordreset/confirm` (with leading `/`).
  - `feature/auth/resetpassword/ResetPassword.tsx`: Already posts `{ emailReset }` to `/api/auth/passwordreset` and handles `APIResponse` toasts; verify compatibility with the new Go handler response shape.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **Anti-Enumeration:** `POST /api/auth/passwordreset` always returns `200 OK` with the same generic message whether the email is unregistered, OAuth-only (`Password == ""`), or registered with a password.
- **Token Security:** Raw 32-byte `crypto/rand` tokens are sent only in the email link; only their SHA-256 hex digest is persisted in `password_resets.verification_token`. Tokens expire in 1 hour, are single-use (`verified = true`), and prior active tokens for the user are revoked when a new reset is requested.
- **Session Invalidation:** Completing a password reset deletes all active `sessions` for that `user_id` inside the same database transaction.
- **Email Header Injection & Secret Safety:** Strip `\r` and `\n` from email header fields; never log raw tokens, email addresses, passwords, or `EMAIL_APP_PASSWORD`.
- **Safe Error Handling:** Return structured `StandardAPIResponsePayload` JSON (`type`, `title`, `message`) with zero raw SQL/GORM errors or stack traces.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Go Auth Helpers & Email Sender:** Add `services/go-api/auth/password_reset_tokens.go` and `services/go-api/auth/password_reset_email_sender.go` (+ unit tests) for SHA-256 token generation and `multipart/alternative` branded HTML/text emails.
- [x] **Step 2 — Go Repository & Handler:** Add `services/go-api/repositories/password_reset_repository.go`, `services/go-api/handlers/password_reset_handler.go` (+ unit tests), and register routes in `services/go-api/main.go`.
- [x] **Step 3 — Next.js Rewrites & New Password UI:** Update `next.config.ts`, create `feature/auth/resetpassword/NewPasswordForm.tsx`, update `app/new-password/page.tsx` with `<Suspense>`, and add Vitest tests in `tests/`.
- [x] **Step 4 — Verification:** Run full Frontend and Go API verification suites.

### Verification Commands (`AGENTS.md` §4)
- [x] **Frontend Typecheck:** `pnpm exec tsc --noEmit`
- [x] **Frontend Lint:** `pnpm lint`
- [x] **Frontend Unit Tests:** `pnpm test`
- [x] **Go API Unit Tests:** `cd services/go-api && go test -v ./...`

---

## 5. Open Questions & Decisions Required

- [ ] **OAuth-only accounts:** When an OAuth-only user (`Password == ""`) requests a password reset, should we silently no-op (returning the generic `200 OK` message), or send an informational email explaining they sign in with Google/GitHub? (Default in this spec: silent no-op with generic `200 OK` to prevent provider/account enumeration.)
- [ ] **Registration verification email:** Should we keep `BuildRegistrationVerificationEmailMessage` as-is for now, or also upgrade it to use the same branded `multipart/alternative` template in a follow-up? (Default in this spec: keep registration email unchanged to keep existing tests untouched.)
