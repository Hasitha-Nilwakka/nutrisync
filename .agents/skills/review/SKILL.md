---
name: review
description: >-
  Run a read-only pre-PR code, security, privacy (GDPR/PII), AI guardrail, and
  spec-compliance audit across Next.js, Go API, and Python AI.
  Trigger whenever the user types /review or asks to review or audit current changes.
---

# Pre-PR Code, Security & Spec-Compliance Audit (`/review`)

You are operating in **Read-Only Principal Auditor Mode** for **NutriSync (`numbers-dont-lie`)**.
**Strict Boundary:** Do **NOT** modify any files during this workflow. Your job is to audit the working tree/branch changes against the feature spec and `AGENTS.md`, then output a structured review report.

---

## Step 1: Gather Audit Context

1. Run `git status` and `git diff` (and inspect untracked files) to identify every modified or added file across `Next.js` (`/`), `services/go-api/`, and `services/py-ai/`.
2. Read [`AGENTS.md`](../../../AGENTS.md), the applicable service rules ([`services/go-api/AGENTS.md`](../../../services/go-api/AGENTS.md), [`services/py-ai/AGENTS.md`](../../../services/py-ai/AGENTS.md)), and the corresponding feature spec in `docs/specs/` (if one exists).

---

## Step 2: Execute the 6-Pillar NutriSync Audit

Evaluate every changed file against these six mandatory pillars:

1. **Spec & Scope Compliance (`AGENTS.md` §1 & §10):**
   - Do the changes fulfill all acceptance criteria and checklist items in `docs/specs/<feature>.md`?
   - Were any unrelated files, unapproved dependencies (`package.json`, `go.mod`, `requirements.txt`), or legacy Prisma artifacts touched?
2. **User Isolation & Authentication (`AGENTS.md` §6):**
   - Is the acting `userID` derived strictly from validated server-side JWT/session cookies (`access_token`/`refresh_token`)?
   - Is every GORM query in `services/go-api/repositories/` scoped to the authenticated `userID` so cross-user access is impossible?
3. **GDPR, Two-Tier PII Anonymization & Secret Safety (`AGENTS.md` §5, §6, §9):**
   - Does `services/go-api/health/ai_orchestrator_client.go` strip all direct identifiers (`userID`, email, name, OAuth tokens) before calling `services/py-ai`?
   - Does `services/py-ai/pipeline/layer1_scispacy_extractor.py` scrub residual free-text PII before Layer 2 builds the LLM prompt?
   - Are `.env` secrets, tokens, and raw SQL/stack traces completely absent from client components, error responses, and logs?
4. **Biometrics, Wellness Score & 3-Layer AI Guardrails (`AGENTS.md` §7):**
   - Does the Wellness Score strictly match `round((bmi_score * 0.3) + (activity_score * 0.3) + (progress_score * 0.2) + (habits_score * 0.2), 2)` clamped to `[0.0, 100.0]`?
   - Are `weight_history` and `activity_history` compacted into 7d/30d trends via `Pandas`/`NumPy` rather than passed as unbounded raw arrays?
   - Does Layer 3 deterministically filter recommendations against Layer 1 forbidden terms and provide a clinical fallback when the LLM is offline?
5. **Single-DB Ownership & Cross-Service Contracts (`AGENTS.md` §8 & §9):**
   - Is PostgreSQL accessed exclusively inside `services/go-api` via GORM (zero direct DB access in Next.js or `services/py-ai`)?
   - Are JSON contracts synchronized between `ai_orchestrator_client.go` and `schemas/analysis_schemas.py`, and are new endpoints mapped in `next.config.ts`?
6. **Code Quality & Test Coverage (`AGENTS.md` §3.5 & §12):**
   - Does every function do **one job** with a **full, descriptive, unabbreviated name** and appropriate file/function comments?
   - Are there deterministic unit tests and negative authorization (`401`/`403`) / validation (`400`) tests for the changed code?

---

## Step 3: Run Read-Only Verification & Report (Subagent Fan-Out)

1. Delegate in parallel (or execute directly for small diffs):
   - **`security-auditor` subagent ([`.agents/agents/security-auditor.md`](../../agents/security-auditor.md)):** Adversarial diff & ADR compliance check against the 6 Pillars above.
   - **`test-runner` subagent ([`.agents/agents/test-runner.md`](../../agents/test-runner.md)):** Execute the non-mutating test and typecheck suites across affected services (`verify-frontend.sh`, `verify-go-endpoint.sh`, `verify-py-ai.sh`).
2. Synthesize their briefs into a structured **Pre-PR Audit Report**:
   - **Summary Table:** Status (`[PASS]`, `[WARNING]`, `[BLOCKER]`) for each of the 6 Pillars.
   - **Detailed Findings:** Exact clickable file/line links (`[file.go](file:///.../file.go#L10-L20)`) and concrete fix instructions for any `[BLOCKER]` or `[WARNING]`.
   - **Verdict:** `READY FOR /pr` (if zero blockers and all checks pass) or `CHANGES REQUIRED`.
