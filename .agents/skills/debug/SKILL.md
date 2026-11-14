---
name: debug
description: >-
  Perform systematic root-cause analysis (RCA), reproduce bugs with targeted tests,
  and apply minimal verified fixes across Next.js, Go API, and Python AI.
  Trigger whenever the user types /debug or reports a bug, failing test, or runtime error.
---

# Systematic Root-Cause Debugging Workflow (`/debug`)

You are operating in **Root-Cause Diagnostic Mode** for **NutriSync (`numbers-dont-lie`)**.
Do **NOT** apply speculative or shotgun patches. You must isolate and verify the root cause before modifying application code.

---

## Phase 1: Reproduce & Trace Across Service Boundaries

1. **Collect Evidence:**
   - Capture the exact error message, HTTP status code, failing test output, or CI failure log (via `github` MCP if from a PR/workflow run).
2. **Trace the Polyglot Request Path:**
   - Identify where the failure originates across NutriSync's 3-service architecture:
     1. **Frontend (`/`):** Zod v4 schema validation (`lib/schemas/`), React Hook Form/component state (`feature/`), server session helper (`lib/auth/requireAuthentication.ts`), or `/api/*` proxy rewrite in `next.config.ts`.
     2. **Go Core API (`services/go-api/`, `:8080`):** HTTP handler payload parsing (`handlers/`), JWT/refresh cookie validation (`auth/`), GORM query/constraint (`repositories/`, `models/`), or biometric normalization (`health/normalization.go`).
     3. **Database (`localhost:5432/nutrisync`):** Query `postgres-readonly` MCP to verify actual table columns, types, foreign keys, and not-null constraints against GORM structs.
     4. **Python Analytics & AI (`services/py-ai/`, `:8000`):** Go <-> Python JSON payload mismatch (`ai_orchestrator_client.go` vs. `schemas/analysis_schemas.py`), `Pandas`/`NumPy` NaN/empty-series edge cases (`analytics/wellness_engine.py`), or Layer 1–3 pipeline filtering (`pipeline/`).
3. **Deterministic Reproduction:**
   - Reproduce the bug using the narrowest targeted command:
     - Frontend: `pnpm exec vitest run <path-to-test>` or `pnpm exec tsc --noEmit`
     - Go API: `cd services/go-api && go test -v ./<pkg> -run <TestName>`
     - Python AI: `cd services/py-ai && .venv/bin/pytest -v <path-to-test>`

---

## Phase 2: Root-Cause Analysis (RCA) & Risk Check

1. Formulate a concise **Root-Cause Report** stating:
   - **Symptom:** What failed and where.
   - **Root Cause:** Exact file and line range causing the defect, and *why* it happens.
   - **Proposed Minimal Fix:** The smallest safe change that resolves the root cause without breaking contracts or invariants.
2. **Approval Gate Check (`AGENTS.md` §5 & §10):**
   - If the fix requires modifying GORM database schemas, authentication/2FA logic, Wellness Score weights, or AI safety guardrails, **stop and get explicit user approval** before editing code.

---

## Phase 3: Regression Test First & Minimal Fix

1. **Write/Update a Failing Regression Test:**
   - Add or update a unit test in `tests/`, `services/go-api/`, or `services/py-ai/` that specifically reproduces the bug and fails before your fix.
   - Never delete or weaken existing unit tests (`AGENTS.md` §5 & §11).
2. **Apply the Minimal Fix:**
   - Implement the fix following NutriSync naming and comment standards while preserving user isolation, PII anonymization, and AI guardrails.
3. **Verify Fix & Full Suite:**
   - Re-run the targeted regression test to confirm it passes.
   - Run the full test suite for the affected service(s) (`pnpm test`, `go test -v ./...`, `.venv/bin/pytest -v`) and `git diff` to ensure zero regressions.
4. Report the RCA, the fix applied, and the exact test verification results.
