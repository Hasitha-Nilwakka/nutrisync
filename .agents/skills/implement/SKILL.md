---
name: implement
description: >-
  Execute an approved feature specification in docs/specs/ step-by-step with
  targeted polyglot verification across Next.js, Go API, and Python AI.
  Trigger whenever the user types /implement or asks to implement an approved spec.
---

# Spec-Driven Implementation Workflow (`/implement`)

You are operating in **Disciplined Execution Mode** for **NutriSync (`numbers-dont-lie`)**.
Your job is to execute an approved specification from `docs/specs/<feature-name>.md` step-by-step, verifying and checking off each item as you go.

---

## Step 1: Pre-Flight & Approval Verification

1. **Read Rules & Target Spec:**
   - Read root [`AGENTS.md`](../../../AGENTS.md), the target `docs/specs/<feature-name>.md`, and the relevant service instructions ([`services/go-api/AGENTS.md`](../../../services/go-api/AGENTS.md) and/or [`services/py-ai/AGENTS.md`](../../../services/py-ai/AGENTS.md)).
2. **Verify Approval & Open Questions:**
   - Check the spec's `Risk Level`, `Approval Gate`, and `## 5. Open Questions & Decisions Required`.
   - If any blocking open question remains unresolved or a `High`/`Critical` risk change (such as GORM schema edits, auth/2FA changes, Wellness Score weight changes, or new dependencies) has not been approved by the user, **stop and ask before writing code**.
3. **Preserve Working Tree:**
   - Run `git status` and ensure pre-existing uncommitted changes are preserved.
   - Update the spec's status header to `- **Status:** In Progress`.

---

## Step 2: Step-by-Step Polyglot Execution

Work through `## 4. Implementation & Verification Checklist` in `docs/specs/<feature-name>.md` in strict order:

1. **Step 2.1 — Contracts & Schemas First:**
   - Update GORM models (`services/go-api/models/`), Pydantic v2 schemas (`services/py-ai/schemas/analysis_schemas.py`), and Zod v4 schemas (`lib/schemas/`).
   - Keep `services/go-api/health/ai_orchestrator_client.go` and `services/py-ai/schemas/analysis_schemas.py` strictly synchronized.
   - Never touch legacy Prisma files (`docs/legacy-prisma-schema.prisma`, `src/generated/prisma/`).
2. **Step 2.2 — Core Service & AI Logic:**
   - Write single-purpose functions with full, descriptive, unabbreviated names (`PascalCase`/`camelCase` in Go; `snake_case` in Python) and clear file/function comments.
   - **Go API (`services/go-api`):** Derive `userID` strictly from validated JWT/session cookies (`access_token`/`refresh_token`), scope every GORM query to `userID`, normalize biometrics via `normalization.go`, and strip account identifiers into `AnonymizedAIAnalysisPayload`.
   - **Python AI (`services/py-ai`):** Preserve the deterministic Wellness Score formula (`round((bmi_score * 0.3) + (activity_score * 0.3) + (progress_score * 0.2) + (habits_score * 0.2), 2)` clamped to `[0.0, 100.0]`), compact time-series arrays with `Pandas`/`NumPy`, scrub residual PII in Layer 1, and enforce Layer 3 forbidden-term validation + deterministic clinical fallback.
3. **Step 2.3 — Frontend & Routing Integration:**
   - Keep `app/` route files thin (`page.tsx`, `layout.tsx`), place feature components in `feature/`, validate forms with Zod v4 + React Hook Form, and route API requests through `/api/*` rewrites in `next.config.ts`.
4. **Step 2.4 — Unit & Negative Security Tests:**
   - Add/update unit tests for every new function/schema across `tests/` (Vitest), `services/go-api/` (`*_test.go`), and `services/py-ai/` (`test_*.py`).
   - Always include boundary validation failure tests (`400`) and negative authorization/cross-user isolation tests (`401`/`403`).
   - Check off each `- [x]` item in `docs/specs/<feature-name>.md` as you complete and verify it.

---

## Step 3: Full Verification & Handoff

1. Run the verification commands required for all affected services (`AGENTS.md` §4):
   - **Frontend:** `pnpm exec tsc --noEmit`, `pnpm lint`, `pnpm test`
   - **Go API:** `cd services/go-api && go test -v ./...`
   - **Python AI:** `cd services/py-ai && .venv/bin/pytest -v`
2. Check off the executed verification boxes in `docs/specs/<feature-name>.md` and update its status to `- **Status:** Completed`.
3. Inspect `git diff` to confirm no accidental or out-of-scope edits occurred.
4. Provide the 5-part handoff report defined in `AGENTS.md` §15 (*Changes, Safety & Architecture Compliance, Verification, Limitations, Follow-ups*).
