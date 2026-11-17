# Feature Spec: End-to-End Access & Refresh Token Rotation + Silent Refresh

- **Status:** Completed
- **Author:** Antigravity
- **Target Issue / PR:** N/A
- **Risk Level (`AGENTS.md` §10):** High (Session/JWT refresh flow, GORM row-level locking on refresh token rotation, Next.js auth proxy/middleware, client HTTP interceptor)
- **Approval Gate:** Approved by User (recommended approaches)
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `proxy.ts` (Next.js 16 request proxy), `lib/api/apiClient.ts` (or global `axios` interceptor setup), `feature/dashboard/*`, `feature/profile/*`, `tests/`
  - [x] **Go Core API (`services/go-api/`)** — `repositories/user_session_repository.go`, `repositories/user_session_repository_test.go`
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — No changes

---

## 1. Objective & Scope

- **Problem Statement:**
  An end-to-end inspection of NutriSync's access token (`access_token`, 15 min JWT) and refresh token (`refresh_token`, 7 day `<id>.<secret>`) revealed that **automatic token refresh is completely unimplemented on the frontend** and **refresh rotation has a concurrency race condition on the backend**:
  1. **Server Navigation (`lib/auth/requireAuthentication.ts`):** Checks only `access_token` via `GET /api/auth/session`. Once the 15-minute `access_token` expires, navigating to `/dashboard` or `/profile` immediately redirects to `/` even when a valid 7-day `refresh_token` cookie exists.
  2. **Client API Calls:** Components import raw `axios` with no `401` interceptor, so any API call after 15 minutes fails with `401` instead of calling `POST /api/auth/refresh` and retrying.
  3. **Backend Rotation Race (`services/go-api/repositories/user_session_repository.go`):** `RotateRefreshTokenInTransaction` updates the old `RefreshToken` (`Where("id = ?")`) without `FOR UPDATE` row locking or `revoked = false`, allowing concurrent refresh requests to rotate the same token simultaneously instead of serializing/enforcing single-use.

- **In Scope:**
  - **Go API Concurrency Hardening (`user_session_repository.go`):**
    - Lock the `RefreshToken` row inside `RotateRefreshTokenInTransaction` (`clause.Locking{Strength: "UPDATE"}`) and verify `revoked == false && replaced_by_token_id IS NULL` inside the transaction before inserting the new `RefreshToken` and marking the old one revoked. Return a sentinel error (`ErrRefreshTokenAlreadyRotated`) if already rotated so `session_handler.go` triggers reuse revocation cleanly.
  - **Next.js Silent Refresh on Navigation (`proxy.ts`):**
    - On protected routes (`/dashboard`, `/profile`), if `access_token` is missing or expired (`decodeJwtExp` / `GET /api/auth/session` fails) while `refresh_token` is present, call `POST ${GO_API_URL}/api/auth/refresh`, forward `Set-Cookie` headers to the browser response, and inject the new `access_token` into the forwarded request headers so `requireAuthentication()` succeeds seamlessly.
  - **Client-Side Single-Flight `401` Refresh Interceptor (`lib/api/apiClient.ts` + `AuthInterceptorProvider` or shared `apiClient`):**
    - Deduplicate concurrent `401` refresh calls via a single-flight `inFlightRefreshPromise` calling `POST /api/auth/refresh`, then retry the failed request once (skipping `/api/auth/login`, `/api/auth/register`, `/api/auth/refresh`, `/api/auth/logout`).
  - **Unit Tests:** Go repository concurrency/sentinel tests + Frontend Vitest tests for single-flight refresh & proxy logic.

- **Out of Scope:**
  - OAuth 2FA bypass & `two_factor_handler.go` session check (noted as separate follow-up security items).
  - Changing token TTLs (15 min access / 7 day refresh) or GORM table schemas.

- **Acceptance Criteria:**
  - [x] Navigating to `/dashboard` or `/profile` with an expired/missing `access_token` and valid `refresh_token` silently rotates tokens via `POST /api/auth/refresh`, updates browser cookies, and renders the page without redirecting to `/`.
  - [x] Client API calls receiving `401` trigger a single-flight `POST /api/auth/refresh` and retry once; concurrent `401`s share the same refresh promise.
  - [x] `RotateRefreshTokenInTransaction` locks the token row (`FOR UPDATE`) and rejects already-revoked tokens inside the transaction.
  - [x] All Go (`go test -v ./...`) and Frontend (`tsc --noEmit`, `pnpm lint`, `pnpm test`) checks pass.

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models
- No schema changes. Uses existing `sessions` and `refresh_tokens` tables.

### 2.2 Go API (`services/go-api/`)
- **`repositories/user_session_repository.go`:**
  - Export `ErrRefreshTokenAlreadyRotated = errors.New("refresh token already rotated or revoked")`.
  - In `RotateRefreshTokenInTransaction`: query `models.RefreshToken` with `Clauses(clause.Locking{Strength: "UPDATE"})` inside `db.Transaction(...)`, verify `!lockedToken.Revoked && lockedToken.ReplacedByTokenID == nil`, then insert `newRefreshToken` and update `revoked = true`, `replaced_by_token_id = newRefreshToken.ID`.
- **`handlers/session_handler.go`:**
  - If `RotateRefreshTokenInTransaction` returns `ErrRefreshTokenAlreadyRotated`, revoke session & all refresh tokens and return `401 Unauthorized` (`"refresh token reuse detected"`).

### 2.3 Python Analytics & AI (`services/py-ai/`)
- No changes.

### 2.4 Next.js Frontend (`/`)
- **`proxy.ts`:** Next.js 16 request proxy intercepting `/dashboard` and `/profile`, checking `access_token` / `refresh_token` cookies, refreshing via `${GO_API_URL}/api/auth/refresh` when needed, and forwarding updated cookies to both browser and Server Components.
- **`lib/api/setupAxiosRefreshInterceptor.ts` (or `lib/api/apiClient.ts`):** Single-flight `axios` response interceptor (`inFlightRefreshPromise`) mounted in `ThemeProvider` / root client layout or imported by authenticated feature modules.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- Preserves `HttpOnly`, `SameSite=Lax`, `Path=/` cookie attributes; tokens are never exposed to `localStorage` or client JS.
- Single-flight refresh in JS + `FOR UPDATE` row locking in PostgreSQL prevent false reuse-detection logouts under concurrent requests.

---

## 4. Implementation & Verification Checklist

- [x] **Step 1 — Go Rotation Locking:** Update `RotateRefreshTokenInTransaction` (`FOR UPDATE` + `ErrRefreshTokenAlreadyRotated`) and `session_handler.go` + tests.
- [x] **Step 2 — Next.js Navigation Refresh (`proxy.ts`):** Implement silent refresh for `/dashboard` and `/profile` when `access_token` is missing/expired and `refresh_token` is present.
- [x] **Step 3 — Client Single-Flight `401` Interceptor:** Add single-flight `POST /api/auth/refresh` interceptor and wire it across authenticated requests.
- [x] **Step 4 — Verification:** Run Go and Frontend verification suites.

---

## 5. Open Questions & Decisions Required

- [x] **Client Interceptor Scope:** Approved attaching the single-flight `401` refresh interceptor idempotently to `axios` in `app/layout.tsx` (`AuthRefreshProvider`).
- [x] **Follow-up Security Items:** Approved keeping scope focused on Access/Refresh Tokens (2FA/OAuth findings tracked separately).
