---
name: plan
description: >-
  Research the NutriSync codebase, query MCP servers (GitHub, PostgreSQL, Context7),
  and generate a structured implementation specification in docs/specs/ before writing any application code.
  Trigger whenever the user types /plan or asks to plan, design, or spec out a feature or issue.
---

# Spec-Driven Planning Workflow (`/plan`)

You are operating in **Architecture & Planning Mode** for **NutriSync (`numbers-dont-lie`)**.
**Strict Boundary:** Do **NOT** modify any application source files (`app/`, `feature/`, `components/`, `lib/`, `services/go-api/`, `services/py-ai/`) during this workflow. The only file you may create or edit is the target feature specification inside `docs/specs/`.

---

## Step 1: Parallel Discovery (Via `explorer` Subagents)

To keep this planning context clean and fast, read [`AGENTS.md`](../../../AGENTS.md) and spawn read-only `explorer` subagents ([`.agents/agents/explorer.md`](../../agents/explorer.md)) in parallel when investigating multi-service features:

- **Subagent A (Codebase & ADR Patterns):**
  - Scan [`docs/adr/README.md`](../../../docs/adr/README.md) and service `AGENTS.md` files (`services/go-api/AGENTS.md`, `services/py-ai/AGENTS.md`).
  - Locate existing canonical routes, handlers, GORM repositories, Python analytics/pipeline modules, and Next.js schemas/rewrites related to the user's request.
  - Return a <300-word brief of relevant files, canonical patterns to mirror, and ADR constraints.
- **Subagent B (MCP Schema, Docs & Issue Context):**
  - If a GitHub Issue or PR `#<number>` is referenced, fetch its requirements via the `github` MCP server.
  - If database persistence is involved, inspect live tables, columns, indexes, and foreign keys via the `postgres-readonly` MCP server (`localhost:5432/nutrisync`) — never reference legacy Prisma files.
  - If framework/library APIs are involved (Next.js 16.4, React 19.3, Tailwind v4, Zod v4, Go 1.26, GORM, FastAPI, Pydantic v2), verify syntax via the `context7` MCP server or `node_modules/next/dist/docs/`.
  - Return a <300-word brief of live schema columns, API syntax, and issue acceptance criteria.

Wait for both subagent briefs before drafting the specification.

---

## Step 2: Draft the Specification

1. Read [`docs/specs/_template.md`](../../../docs/specs/_template.md).
2. Create a new specification file at `docs/specs/<kebab-case-feature-name>.md` following **every section** of `_template.md`:
   - **Header Metadata:** Set `Status: Draft`, classify the `Risk Level` (`Low | Medium | High | Critical` per `AGENTS.md` §10), mark the `Approval Gate`, and check off the `Affected Services` (`Next.js Frontend`, `Go Core API`, `Python Analytics & AI`).
   - **Section 1 (Objective & Scope):** Define the problem statement, concrete in-scope deliverables, explicit out-of-scope boundaries, measurable acceptance criteria, and completed MCP pre-flight checks.
   - **Section 2 (Polyglot Technical Design):** Detail exact GORM struct changes (`services/go-api/models/`), Go HTTP routes/handlers and `next.config.ts` rewrites, Go <-> Python JSON contract synchronization (`ai_orchestrator_client.go` <-> `analysis_schemas.py`), Python 3-Layer AI / Wellness Score impacts, and Frontend Zod v4 schemas (`lib/schemas/`) + `feature/` components.
   - **Section 3 (Security, Privacy & Clinical Guardrails):** Explicitly document server-side JWT cookie `userID` isolation (`401`/`403`), GDPR consent/export impacts, Two-Tier PII Anonymization, Wellness Score clamping (`[0.0, 100.0]`), Layer 3 contraindication filtering, and safe JSON error shapes.
   - **Section 4 (Implementation & Verification Checklist):** Break the implementation into ordered, atomic `- [ ]` steps referencing exact file paths, descriptive function names, unit/negative test cases, and the exact verification commands from `AGENTS.md` §4.
   - **Section 5 (Open Questions & Decisions Required):** List any unresolved clinical formulas, privacy/consent questions, schema trade-offs, or UX ambiguities that require human decision before coding.

---

## Step 3: Hand Off for Human Review

1. Present a concise summary in chat linking to `docs/specs/<kebab-case-feature-name>.md`, highlighting:
   - Affected services and classified risk level.
   - Key architectural and contract decisions.
   - Any open questions from `## 5. Open Questions & Decisions Required`.
2. **Stop and ask for explicit user approval** before writing or modifying any application code.
