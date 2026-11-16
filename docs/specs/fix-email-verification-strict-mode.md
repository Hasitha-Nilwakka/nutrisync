# Feature Spec: Fix Double-Verification in Email Verification Page (React Strict Mode)

- **Status:** Completed
- **Author:** Antigravity
- **Target Issue / PR:** N/A (Local branch `ui-improvement` follow-up)
- **Risk Level (`AGENTS.md` §10):** Medium (Authentication verification UI flow; frontend-only fix)
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `feature/auth/register/VerifyEmailStatus.tsx`, `tests/registeruser.test.ts`
  - [ ] **Go Core API (`services/go-api/`)** — No changes required (`GET /api/auth/verify` single-use token behavior is working as designed)
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — No changes required

---

## 1. Objective & Scope

- **Problem Statement:**
  When a user clicks the email verification link (`/verify-email?token=...`), the UI displays **"Verification Failed — invalid or expired verification token"**, even though the user's account **was** verified in PostgreSQL and they can log in.
  **Root Cause:** In Next.js development mode (React 19 `StrictMode`), `VerifyEmailStatus.tsx` mounts, runs `useEffect` (Request #1 → `GET /api/auth/verify?token=...`, which succeeds in Go and clears `verification_token = nil`), then StrictMode runs cleanup (`cancelled = true`, discarding the `200 OK` response), and remounts `useEffect` (Request #2 → `GET /api/auth/verify?token=...`, which fails with `400` because Request #1 already consumed and cleared the single-use token).

- **In Scope:**
  - Deduplicate `GET /api/auth/verify?token=...` requests per token in `feature/auth/register/VerifyEmailStatus.tsx` using a module-level `Map<string, Promise<VerificationStatus>>` so React StrictMode's mount → cleanup → remount cycle awaits the single in-flight request instead of firing a second HTTP request.
  - Improve the fallback error copy in `VerifyEmailStatus.tsx` so if a user later revisits or refreshes an already-used link, it mentions that they can try logging in if they already verified.
  - Add a Vitest unit test verifying that `VerifyEmailStatus.tsx` deduplicates verification requests per token.

- **Out of Scope:**
  - Making `GET /api/auth/verify` return `200 OK` without a token or disabling single-use token clearing in `services/go-api`.

- **Acceptance Criteria:**
  - [x] Visiting `/verify-email?token=<validToken>` in Next.js dev mode (`StrictMode`) sends only **one** `GET /api/auth/verify?token=...` request and renders the **Email Verified** success card.
  - [x] Invalid/expired tokens still render the **Verification Failed** error card.
  - [x] `pnpm exec tsc --noEmit`, `pnpm lint`, and `pnpm test` pass.

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/`)
- None.

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- None.

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- None.

### 2.4 Frontend UI & Validation (`feature/auth/register/VerifyEmailStatus.tsx`)
- Maintain a module-level `verificationRequestCache = new Map<string, Promise<VerificationStatus>>()`.
- Helper `verifyEmailTokenOnce(token: string): Promise<VerificationStatus>` checks `verificationRequestCache.get(token)`; if absent, creates and stores the `axios.get("/api/auth/verify", { params: { token } })` promise.
- Inside `useEffect`, await `verifyEmailTokenOnce(token)` with a `cancelled` guard so StrictMode's remount receives the exact resolved `VerificationStatus` from the single HTTP call.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- Single-use token clearing in `services/go-api` remains strictly enforced.
- No tokens or PII are persisted in `localStorage` or logs; the in-memory `Map` only deduplicates within the active tab lifecycle.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Deduplicate Verification Request:** Update `feature/auth/register/VerifyEmailStatus.tsx` with `verifyEmailTokenOnce(token)` and clearer already-used guidance.
- [x] **Step 2 — Unit Test:** Add a Vitest test in `tests/registeruser.test.ts` verifying request deduplication in `VerifyEmailStatus.tsx`.
- [x] **Step 3 — Verification:** Run `pnpm exec tsc --noEmit`, `pnpm lint`, and `pnpm test`.

### Verification Commands (`AGENTS.md` §4)
- [x] **Frontend Typecheck:** `pnpm exec tsc --noEmit`
- [x] **Frontend Lint:** `pnpm lint`
- [x] **Frontend Unit Tests:** `pnpm test`

---

## 5. Open Questions & Decisions Required

- None (root cause is deterministic and fix is isolated to `VerifyEmailStatus.tsx`).
