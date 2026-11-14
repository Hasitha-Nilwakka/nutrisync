# NutriSync (`numbers-dont-lie`) — Agentic Workflow Setup Guide

This document serves as the central engineering log and architectural reference for configuring the AI Agentic Workflow across the **NutriSync (`numbers-dont-lie`)** polyglot repository. It records **what** we configured at each stage, **why** we made specific architectural decisions, and **how** the pieces fit together.

---

## Architecture Context

NutriSync is a privacy-first AI Health & Wellness platform built as a **3-service polyglot stack**:
1. **Web Frontend (`/` root):** Next.js `16.4.2` (App Router, Turbopack), React `19.3.0`, TypeScript `5`, Tailwind CSS v4, Shadcn/ui, React Hook Form, Zod v4, Vitest.
2. **Core Backend & Auth Service (`services/go-api`):** Go `1.26` (`:8080`), **GORM** (`gorm.io/gorm`), PostgreSQL `17`, JWT + Refresh Token rotation, Google/GitHub OAuth2, TOTP 2FA. **Single owner of PostgreSQL database access.**
3. **Stateless Analytics & 3-Layer AI Engine (`services/py-ai`):** Python FastAPI (`:8000`), Pydantic v2, NumPy, Pandas, SciSpacy, Pytest. **Zero direct database access.**

Because an AI agent working in this repository operates across three languages (TypeScript, Go, Python) and handles sensitive health data, GDPR consent, and medical/dietary AI guardrails, a generic single-service agent setup is insufficient. Our setup uses **Defense-in-Depth**:
1. **Declarative Operating Rules (`AGENTS.md`)** — Tell the agent how the system is architected and what invariants must never be broken.
2. **Deterministic Lifecycle Hooks & Permissions (`.agents/hooks.json`)** — Enforce hard runtime boundaries (`PreToolUse`) and automated code hygiene (`PostToolUse`).
3. **Model Context Protocol Servers (`.agents/mcp_config.json`)** — Give the agent live, accurate context (up-to-date docs, live DB schemas, and GitHub issues/CI logs) so it never hallucinates APIs or column names.

---

## Step 1: Root Operating Rules ([`AGENTS.md`](../../AGENTS.md))

### What We Did
We started from a 15-section industry-standard `AGENTS.md` template (originally generated for a monolithic B2B invoicing SaaS using Drizzle ORM and Biome) and systematically adapted every section to **NutriSync (`numbers-dont-lie`)**:

* **Preserved the Next.js 16 Header:** Kept the `<!-- BEGIN:nextjs-agent-rules -->` block at the top so `next dev` does not re-create uncommitted diffs.
* **Mapped the Real 3-Service Architecture (Sections 2 & 3):**
  * Replaced fictional `src/app/api/invoices` and Drizzle paths with the actual directory map across Frontend (`app/`, `feature/`, `components/ui/`, `lib/auth/`, `lib/schemas/`, `next.config.ts`), Go API (`services/go-api/handlers/`, `repositories/`, `models/`, `database/`, `auth/`, `health/`), and Python AI (`services/py-ai/schemas/`, `analytics/`, `pipeline/`).
  * Linked hierarchically to [`services/go-api/AGENTS.md`](../../services/go-api/AGENTS.md) and [`services/py-ai/AGENTS.md`](../../services/py-ai/AGENTS.md).
  * Added an explicit warning that `docs/legacy-prisma-schema.prisma`, `prisma.config.ts`, and `src/generated/prisma/` are **legacy artifacts** and must never be used for database access.
* **Replaced Fictional Commands with Polyglot Verification (Section 4):**
  * Replaced `biome` and `playwright` with the actual commands across Frontend (`pnpm exec tsc --noEmit`, `pnpm lint`, `pnpm test`), Go (`cd services/go-api && go test -v ./...`), and Python (`cd services/py-ai && .venv/bin/pytest -v`).
* **Replaced B2B Tenant Invariants with User Data Isolation & Two-Tier PII Anonymization (Section 6):**
  * Enforced that `userID` is derived strictly from validated server-side JWT cookies (`access_token` / `refresh_token`), never from client-supplied payloads.
  * Codified **Two-Tier PII Anonymization**:
    1. `services/go-api/health/ai_orchestrator_client.go` strips direct account identifiers (user ID, email, name, OAuth tokens) before calling `services/py-ai`.
    2. `services/py-ai/pipeline/layer1_scispacy_extractor.py` scrubs residual free-text PII (emails, phones, person names) before Layer 2 builds the LLM prompt.
* **Replaced Financial Invariants with Health Analytics & 3-Layer AI Guardrails (Section 7):**
  * Codified biometric unit normalization (`services/go-api/health/normalization.go`).
  * Locked in the deterministic 4-part Wellness Score formula:
    $$\text{wellness\_score} = \text{round}\big((\text{bmi\_score} \times 0.3) + (\text{activity\_score} \times 0.3) + (\text{progress\_score} \times 0.2) + (\text{habits\_score} \times 0.2),\ 2\big)$$
    clamped to `[0.0, 100.0]`.
  * Enforced Pandas/NumPy 7-day/30-day statistical context compaction (no raw unbounded arrays in prompts) and Layer 3 dietary/medical contraindication filtering with offline clinical fallback.
* **Enforced Single-Database Ownership (Section 8):**
  * Explicitly established that **only `services/go-api` via GORM** owns PostgreSQL; Next.js and `services/py-ai` must never connect to PostgreSQL directly.

### Why We Did It
Prompt instructions fail when they contain stale or contradictory stack references (like telling an agent to use Drizzle or Biome when the repo uses GORM and ESLint). Aligning `AGENTS.md` with the exact files, formulas, and privacy boundaries eliminates hallucinated patterns and establishes a clear contract for every subsequent customization layer.

---

## Step 2: Self-Contained Lifecycle Hooks & Permissions ([`.agents/hooks.json`](../../.agents/hooks.json))

### What We Did
We designed a self-contained [`.agents/hooks.json`](../../.agents/hooks.json) file (with inline `"_comment"` annotations on every block for maintainability while remaining 100% strict valid JSON) divided into two parts:

1. **Static Permission Matrix (`permissions.allow`, `permissions.ask`, `permissions.deny`):**
   * **`allow`:** Safe Git read commands (`git status`, `git diff`, `git log`), frozen installs (`pnpm install --frozen-lockfile`), typechecks (`tsc --noEmit`), linters (`eslint`), test runners (`vitest`, `go test`, `pytest`), and read-only MCP queries (`mcp__context7__*`, `mcp__postgres-readonly__query`, `mcp__github__get_*`/`list_*`/`search_*`).
   * **`ask`:** High-risk changes requiring explicit human approval per `AGENTS.md` §5 & §10: editing dependency files (`package.json`, `pnpm-lock.yaml`, `go.mod`, `go.sum`, `requirements.txt`), editing GORM models/migrations (`services/go-api/models/*`, `services/go-api/database/*`), editing Dockerfiles/Compose, installing packages (`pnpm add`, `go get`, `pip install`), and Git commits/merges/pushes or GitHub issue/PR mutations.
   * **`deny`:** Reading or editing `.env` / `.env.*` secret files, editing deprecated legacy Prisma files, running destructive commands (`rm -rf`, `git reset --hard`, `git clean -f`, `git push --force`, `--no-verify`, `docker compose down -v`), and invoking disallowed package managers (`npm`, `yarn`).

2. **Runtime Lifecycle Hooks (`PreToolUse` & `PostToolUse`):**
   * **`PreToolUse` — Shell Command Safety Guard:** Intercepts `run_command`/`Bash` before execution and blocks `--no-verify`, `git reset --hard`, `git clean -f`, `git push --force`, `rm -rf`, `docker compose down -v` (which would wipe the local Postgres volume), and `npm`/`yarn`.
   * **`PreToolUse` — Protected File Guard:** Intercepts `read_file`/`edit_file`/`write_file` (`Read`/`Edit`/`Write`) and blocks reading or modifying `.env` files (while allowing `.env.example`) as well as modifying legacy Prisma files.
   * **`PostToolUse` — Polyglot Auto-Linter & Formatter:** Runs automatically after any file edit/write:
     * `.ts` / `.tsx` -> `pnpm exec eslint --fix <file>`
     * `.go` -> `gofmt -w <file>`
     * `.py` -> `services/py-ai/.venv/bin/python -m py_compile <file>`

### Why We Did It
* **Defense-in-Depth:** Even if an agent overlooks a rule in `AGENTS.md`, `PreToolUse` deterministically blocks secret exposure (`.env`), destructive Git/Docker commands, and accidental edits to legacy Prisma files before the tool ever executes.
* **Shift-Left Code Quality:** Running `eslint --fix`, `gofmt -w`, and `py_compile` automatically in `PostToolUse` keeps code formatted and syntactically valid immediately after every edit without wasting extra LLM turns.
* **Self-Contained & Documented:** Using inline commands and JSON `"_comment"` keys keeps the entire hook logic readable in one file without depending on legacy scripts.

---

## Step 3: Model Context Protocol (MCP) Servers ([`.agents/mcp_config.json`](../../.agents/mcp_config.json))

### What We Did
We created [`.agents/mcp_config.json`](../../.agents/mcp_config.json) with three specialized MCP servers and added corresponding usage rules in [`AGENTS.md` §4](../../AGENTS.md) and permissions in [`.agents/hooks.json`](../../.agents/hooks.json):

1. **`context7` (`@upstash/context7-mcp@latest`):**
   * **Purpose:** Pulls live, version-specific documentation for Next.js `16.4.2`, React `19.3.0`, Tailwind CSS `v4`, Zod `v4`, Go `1.26`, GORM `v1.31`, FastAPI, and Pydantic `v2`.
   * **Why:** Prevents the agent from generating obsolete Next.js 13/14, Zod v3, or Tailwind v3 patterns from pre-cutoff training weights.
2. **`postgres-readonly` (`@modelcontextprotocol/server-postgres`):**
   * **Purpose:** Connects strictly to the local Docker PostgreSQL database (`postgresql://app:app_password@localhost:5432/nutrisync`) with `PGOPTIONS="-c default_transaction_read_only=on"`.
   * **Why:** Gives the agent live read-only visibility into the actual tables (`users`, `sessions`, `accounts`, `health_profiles`, `weight_entries`, `activity_entries`, `analytics_snapshots`), column types, foreign keys, and indexes created by GORM `AutoMigrate`—eliminating hallucinated column names while enforcing zero DDL/write mutations and zero production/PII exposure.
3. **`github` (`@modelcontextprotocol/server-github`):**
   * **Purpose:** Connects to GitHub using `${GITHUB_PERSONAL_ACCESS_TOKEN}` from the host environment.
   * **Why:** Allows the agent to read GitHub Issue requirements, PR review comments, and CI failure logs directly. Read operations (`get_*`, `list_*`, `search_*`) are auto-allowed in `hooks.json`, while mutating operations (`create_*`, `update_*`, `merge_*`) require explicit user approval (`ask`).

---

## Step 4: Feature Spec Blueprint ([`docs/specs/_template.md`](../specs/_template.md))

### What We Did
We created a version-controlled, machine-parsable **Feature Spec Blueprint** at [`docs/specs/_template.md`](../specs/_template.md) and linked it into [`AGENTS.md`](../../AGENTS.md) (§3.4, §10, §11). We adapted the standard 4-part spec template to NutriSync's specific architecture:

1. **Header Metadata with Risk & Polyglot Service Checkboxes:**
   * Added **Risk Level** (`Low | Medium | High | Critical` per `AGENTS.md` §10), **Approval Gate Status**, and **Affected Services** checkboxes (`Next.js Frontend`, `Go Core API`, `Python Analytics & AI`) so any developer or agent immediately sees which service boundaries and approval rules apply.
2. **Section 1 — Objective, Scope & Pre-Flight MCP Verification:**
   * Added explicit `Acceptance Criteria` checkboxes and **Pre-Flight MCP Checkboxes** reminding the author/agent to verify live database tables via `postgres-readonly`, library APIs via `context7`, and issue specs via `github` *before* writing technical designs.
3. **Section 2 — Polyglot Technical Design (4 Service Subsections):**
   * Replaced generic monolithic design bullets with NutriSync's four concrete architectural boundaries:
     * `2.1 Database & GORM Models (services/go-api)` — Enforcing GORM-only DB ownership and transaction planning.
     * `2.2 Go API & Cross-Service Contracts` — Covering `:8080` handlers, `next.config.ts` rewrites, `normalization.go`, and `ai_orchestrator_client.go` <-> `analysis_schemas.py` JSON synchronization.
     * `2.3 Python Analytics & 3-Layer AI Pipeline (services/py-ai)` — Covering Pydantic v2 models, vectorized `Pandas`/`NumPy` trends, and Layers 1–3 (SciSpacy PII/restriction extraction, prompt compaction, guardrail validation & clinical fallback).
     * `2.4 Frontend UI & Validation` — Covering thin `app/` routes, `feature/` modules, and `lib/schemas/` Zod v4 validation.
4. **Section 3 — Security, Privacy (GDPR/PII) & Clinical Guardrails:**
   * Replaced B2B "Tenant Isolation" with NutriSync's actual invariants: JWT cookie `userID` scoping (`401`/`403`), GDPR consent/export impact, Two-Tier PII Anonymization, Wellness Score formula clamping (`[0.0, 100.0]`), and safe structured JSON error responses (`{"error": "..."}`).
5. **Section 4 — Implementation & 3-Service Verification Checklist:**
   * Added trackable checkboxes (`- [ ]`) for happy-path tests, boundary validation tests, negative cross-user authorization tests, and the exact verification commands across Frontend (`tsc --noEmit`, `pnpm lint`, `pnpm test`), Go (`go test -v ./...`), and Python (`.venv/bin/pytest -v`).

### Why We Did It
In a 3-service polyglot repository, ad-hoc feature implementation often leads to broken JSON contracts between Go and Python, missing Next.js rewrites in `next.config.ts`, or forgotten negative authorization tests. A standardized, machine-parsable spec blueprint forces every Medium/High-risk task to align on schemas, privacy guardrails, and test suites *before* code is written.

---

## Step 5: Standardized 5-Skill Engineering Lifecycle (`.agents/skills/`)

### What We Did
1. **Removed Unstructured Legacy Skills:** Deleted the old ad-hoc skills (`describer`, `go-tester`, `py-tester`, `py-ai-architect-auditor`, `ui-developer`) from `.agents/skills/` to start from a clean, standardized skill architecture.
2. **Used Modern Antigravity Skills Instead of Deprecated Workflows:** Rather than creating deprecated `.agent/workflows/*.md` files (which would also split customizations across `.agent/` and `.agents/`), we implemented five cohesive skills under `.agents/skills/<name>/SKILL.md`. In Antigravity, each skill automatically registers as a `/<name>` slash command in chat, supports semantic auto-invocation, and stays inside the unified `.agents/` root.
3. **Implemented the 5 Core Engineering Skills:**
   * **[`/plan` (`.agents/skills/plan/SKILL.md`)](../../.agents/skills/plan/SKILL.md) — Spec-Driven Planning:**
     * Strictly read-only for application code; only creates `docs/specs/<kebab-case-feature-name>.md`.
     * Queries `github`, `postgres-readonly`, and `context7` MCP servers, fills out [`docs/specs/_template.md`](../specs/_template.md), and halts for human approval.
   * **[`/implement` (`.agents/skills/implement/SKILL.md`)](../../.agents/skills/implement/SKILL.md) — Step-by-Step Spec Execution:**
     * Verifies spec approval and unresolved questions first, updates spec status to `In Progress`, executes each `- [ ]` item in order (Schemas/Contracts -> Go/Python Core Logic -> Frontend -> Unit & Negative `401`/`403` Tests), and checks off `- [x]` items as verification passes.
   * **[`/review` (`.agents/skills/review/SKILL.md`)](../../.agents/skills/review/SKILL.md) — Pre-PR Code, Security & Spec-Compliance Audit:**
     * Strictly read-only auditor that evaluates `git diff` against 6 NutriSync pillars: Spec Compliance, User Isolation (`userID` cookie scoping), Two-Tier PII Anonymization & GDPR, Wellness Score & Layer 3 AI Guardrails, Single-DB Ownership (`go-api` GORM), and Code/Test Quality. Outputs a `[PASS]` / `[WARNING]` / `[BLOCKER]` report with a `READY FOR /pr` or `CHANGES REQUIRED` verdict.
   * **[`/debug` (`.agents/skills/debug/SKILL.md`)](../../.agents/skills/debug/SKILL.md) — Systematic Root-Cause Analysis (RCA):**
     * Prevents shotgun patching by tracing bugs across the 3-service boundary (`Next.js` -> `go-api` -> `postgres-readonly` / `py-ai`), reproducing failures with targeted tests, writing a failing regression test first, and applying the smallest safe fix.
   * **[`/pr` (`.agents/skills/pr/SKILL.md`)](../../.agents/skills/pr/SKILL.md) — Full Verification, Changelog & Clean Pull Request:**
     * Audits `git status` for stray files or `.env` secrets, runs the full verification gate across all 3 services (`tsc --noEmit`, `pnpm lint`, `pnpm test`, `go test -v ./...`, `.venv/bin/pytest -v`), synthesizes a structured PR description/changelog linked to the spec and issue, and requests human confirmation before `git commit` / `git push` / `mcp__github__create_pull_request`.

### How the End-to-End Day-to-Day Loop Works
1. **Plan (`/plan`):** `/plan Add weekly activity streak milestone to health dashboard for issue #42` -> generates `docs/specs/weekly-activity-streak-milestone.md` for your review.
2. **Execute (`/implement`):** `/implement docs/specs/weekly-activity-streak-milestone.md` -> builds the feature step-by-step across Go, Python, and Next.js, checking off items as tests pass.
3. **Debug if needed (`/debug`):** `/debug Fix failing 30-day moving average slope test in wellness_engine.py` -> isolates root cause and adds a regression test before patching.
4. **Audit (`/review`):** `/review` -> performs a read-only 6-pillar security, privacy, and spec compliance audit before opening a PR.
5. **Ship (`/pr`):** `/pr` -> runs all 3 test suites, generates the changelog and PR body, and opens a clean Pull Request once approved.

---

## Step 6: Progressive-Disclosure Domain Builder Skills (`.agents/skills/<domain>/`)

### What We Did
While the 5 workflow skills (`/plan`, `/implement`, `/review`, `/debug`, `/pr`) remain lean `SKILL.md` runbooks governing *when* and *how* the agent moves through engineering phases, we created **3 Multi-File Domain Builder Skills**—one for each service in NutriSync's polyglot stack—following the 4-part **Progressive Disclosure Architecture** (`SKILL.md` + `scripts/` + `references/` + `assets/`):

```text
.agents/skills/
├── create-go-endpoint/
│   ├── SKILL.md                              # <90 lines: Go handler + GORM repo + rewrite + test flow
│   ├── scripts/verify-go-endpoint.sh         # Deterministic gofmt + go vet + go test runner
│   ├── references/auth-and-error-contracts.md # Cookie JWT userID scoping & HTTP status/error table
│   └── assets/
│       ├── handler.template.go               # Starter Go handler with cookie auth & bounds check
│       └── handler_test.template.go          # Starter Go test with 401/403 & 400 negative test cases
├── build-py-ai-pipeline/
│   ├── SKILL.md                              # <90 lines: Pydantic sync + Pandas + 3-Layer AI flow
│   ├── scripts/verify-py-ai.sh               # Checks for zero direct DB imports + runs pytest -v
│   ├── references/wellness-and-guardrails.md # 0.3/0.3/0.2/0.2 Wellness formula & Layer 1-3 rules
│   └── assets/
│       └── test_pipeline.template.py         # Synthetic anonymized payload & pytest starter
└── create-nextjs-feature/
    ├── SKILL.md                              # <90 lines: Zod v4 + feature/ UI + thin app/ route flow
    ├── scripts/verify-frontend.sh            # Runs tsc --noEmit + pnpm lint + pnpm test
    ├── references/nextjs16-ui-conventions.md # App Router, Shadcn/ui, & /api/* rewrite rules
    └── assets/
        ├── schema.template.ts                # Starter Zod v4 schema + inferred TypeScript type
        └── schema.test.template.ts           # Starter Vitest valid/invalid boundary test
```

### Why We Did It
* **Context Window Efficiency (3-Level Progressive Disclosure):**
  1. At session startup, only the 2-line YAML frontmatter (`name` + `description`) is loaded into context.
  2. When a domain skill is triggered, the agent reads a concise (<90-line) `SKILL.md`.
  3. Detailed reference tables (`references/`) and starter templates (`assets/`) are read *only* when the agent is actively writing that specific layer.
* **Deterministic Verification (`scripts/`):** Each domain skill ships with its own executable verification script (`verify-go-endpoint.sh`, `verify-py-ai.sh`, `verify-frontend.sh`) that enforces formatting, architectural boundaries (such as blocking direct DB driver imports in `services/py-ai`), and unit test execution in one command.
* **Standardized Boilerplate (`assets/`):** Starter templates guarantee that new Go handlers, Python pipeline tests, and Next.js Zod v4 schemas immediately include proper comments, descriptive function names, and negative security test cases (`401`/`403` and `400`).

---

## Step 7: Architecture Decision Records (`docs/adr/` & `/adr` Skill)

### What We Did
We established a version-controlled **Institutional Memory** layer in [`docs/adr/`](../adr/) paired with the [`/adr` skill (`.agents/skills/adr/SKILL.md`)](../../.agents/skills/adr/SKILL.md):

1. **Scannable ADR Index ([`docs/adr/README.md`](../adr/README.md)):**
   * Linked directly from [`AGENTS.md` (§3.4 & §11)](../../AGENTS.md) and [`/plan`](../../.agents/skills/plan/SKILL.md).
   * Provides a lightweight table (*ID | Title | Status | Affected Services | One-Sentence Summary*) so the agent scans all settled architectural decisions in a few tokens and opens individual ADR files only on demand.
2. **Standardized ADR Template ([`docs/adr/_template.md`](../adr/_template.md)):**
   * Structured around *Context & Problem Statement, Decision Drivers, Considered Options, Decision Outcome, Consequences, and Compliance & Enforcement*.
3. **Seeded the 3 Foundational NutriSync ADRs:**
   * **[`0001-record-architecture-decisions.md`](../adr/0001-record-architecture-decisions.md):** Establishes the ADR process and immutable superseding rules (`Superseded by ADR-NNNN`).
   * **[`0002-single-db-ownership-go-gorm.md`](../adr/0002-single-db-ownership-go-gorm.md):** Records why `services/go-api` via GORM is the sole owner of PostgreSQL, why Next.js and `services/py-ai` never query PostgreSQL directly, and why legacy Prisma files are frozen.
   * **[`0003-two-tier-pii-and-3-layer-ai.md`](../adr/0003-two-tier-pii-and-3-layer-ai.md):** Records why health analytics and AI insights run in stateless `services/py-ai` behind Go outbound PII anonymization (`AnonymizedAIAnalysisPayload`), vectorized `Pandas`/`NumPy` Wellness Score math, and the 3-Layer SciSpacy/LLaMA guardrail pipeline.
4. **Created the `/adr` Workflow Skill ([`.agents/skills/adr/SKILL.md`](../../.agents/skills/adr/SKILL.md)):**
   * Typing `/adr <decision title>` automatically assigns the next sequential `NNNN` ID, drafts the ADR from `_template.md`, marks any replaced ADR as `Superseded`, and updates the index table in `docs/adr/README.md`.

### Why We Did It
Without ADRs, each new agent session sees *what* the rules are in `AGENTS.md`, but lacks the historical *why* (for instance, why Prisma files still exist in the repo or why Go doesn't call an LLM directly). ADRs prevent architectural drift and stop future sessions from re-litigating settled design choices.

---

## Step 8: Custom Subagents & Parallel Orchestration (`.agents/agents/`)

### What We Did
We created three reusable, non-mutating **Custom Subagent Personas** inside [`.agents/agents/`](../../.agents/agents/) using Antigravity's native subagent frontmatter (`subagent: true`, `mainAgent: false`, and explicit native `tools:` grants) and wired them into `/plan`, `/review`, and `/pr`:

1. **[`explorer` (`.agents/agents/explorer.md`)](../../.agents/agents/explorer.md) — Read-Only Codebase & Schema Explorer:**
   * **Tools:** `[read_file, search_web, call_mcp_tool, list_resources, read_resource]` (strictly zero file-write or shell-execution tools).
   * **Role:** Scans `docs/adr/README.md`, locates canonical patterns across `Next.js`, `services/go-api`, and `services/py-ai`, and queries `postgres-readonly`, `context7`, and `github` MCP servers.
   * **Output Budget:** Returns a structured technical brief (**<300 words**) listing *Relevant Files, Canonical Pattern to Mirror, Schema & Contract Details, and Constraints/Gotchas*.
2. **[`security-auditor` (`.agents/agents/security-auditor.md`)](../../.agents/agents/security-auditor.md) — Adversarial Security, Privacy & Spec Auditor:**
   * **Tools:** `[read_file, run_command, call_mcp_tool]` (uses `run_command` strictly for read-only `git status` / `git diff`).
   * **Role:** Independently audits `git diff` against `AGENTS.md`, accepted ADRs (`ADR-0001`..`0003`), and active `docs/specs/*.md`—checking cookie-based `userID` GORM scoping, single-DB ownership, Two-Tier PII Anonymization, Wellness Score/Layer 3 AI guardrails, and negative test coverage.
   * **Output Budget:** Lists **Blockers** (with exact `file:line` references) and **Warnings** in **<300 words**, or outputs `APPROVED`.
3. **[`test-runner` (`.agents/agents/test-runner.md`)](../../.agents/agents/test-runner.md) — Isolated Polyglot Test Executor:**
   * **Tools:** `[read_file, run_command, manage_task]`.
   * **Role:** Executes the domain verification scripts (`verify-frontend.sh`, `verify-go-endpoint.sh`, `verify-py-ai.sh`) or targeted unit tests in an isolated context without ever editing source or test files.
   * **Output Budget:** Returns a compact pass/fail table and exact failure stack traces in **<250 words** so thousands of lines of raw test output never pollute the main agent's context window.
4. **Wired Subagent Fan-Out into Lifecycle Skills:**
   * [`/plan`](../../.agents/skills/plan/SKILL.md) fans out two parallel `explorer` subagents in Step 1 (Subagent A: ADRs + codebase patterns; Subagent B: `postgres-readonly` live DB schema + `context7` docs + `github` issue).
   * [`/review`](../../.agents/skills/review/SKILL.md) delegates diff auditing to `security-auditor` and suite verification to `test-runner`.
   * [`/pr`](../../.agents/skills/pr/SKILL.md) delegates the full 3-service verification gate to `test-runner`.

### Why We Did It
* **Context-Window Hygiene:** Keeping raw multi-file searches, MCP schema dumps, and 3-service test logs inside ephemeral subagents—and returning only <250–300 word structured briefs—keeps the main coordinator's context window clean and sharp across long sessions.
* **Zero Write Conflicts:** Because `explorer`, `security-auditor`, and `test-runner` never modify files, they can safely run in parallel without causing Git index locks or race conditions.

---

## Complete Agentic Workflow Directory Map

```text
numbers-dont-lie/
├── AGENTS.md                                          # Root operating rules & 3-service invariants
├── services/
│   ├── go-api/AGENTS.md                               # Go 1.26 + GORM service-specific rules
│   └── py-ai/AGENTS.md                                # Python FastAPI + 3-Layer AI service-specific rules
├── .agents/
│   ├── hooks.json                                     # Self-contained permissions + PreToolUse/PostToolUse hooks
│   ├── mcp_config.json                                # Context7, Postgres-Readonly, and GitHub MCP servers
│   ├── agents/                                        # Custom context-isolated Subagent personas
│   │   ├── explorer.md                                # Read-only codebase, ADR & MCP researcher (<300 words)
│   │   ├── security-auditor.md                        # Adversarial diff, privacy & ADR auditor (<300 words)
│   │   └── test-runner.md                             # Isolated 3-service verification executor (<250 words)
│   └── skills/
│       ├── plan/SKILL.md                              # /plan      — Parallel discovery & spec generation
│       ├── implement/SKILL.md                         # /implement — Step-by-step spec execution
│       ├── review/SKILL.md                            # /review    — 6-pillar pre-PR audit
│       ├── debug/SKILL.md                             # /debug     — Root-cause analysis & regression-first fix
│       ├── pr/SKILL.md                                # /pr        — Full verification gate & PR creation
│       ├── adr/SKILL.md                               # /adr       — Create/supersede ADRs & update index
│       ├── create-go-endpoint/                        # Progressive-disclosure Go API builder skill
│       │   ├── SKILL.md
│       │   ├── scripts/verify-go-endpoint.sh
│       │   ├── references/auth-and-error-contracts.md
│       │   └── assets/{handler.template.go, handler_test.template.go}
│       ├── build-py-ai-pipeline/                      # Progressive-disclosure Python AI builder skill
│       │   ├── SKILL.md
│       │   ├── scripts/verify-py-ai.sh
│       │   ├── references/wellness-and-guardrails.md
│       │   └── assets/test_pipeline.template.py
│       └── create-nextjs-feature/                     # Progressive-disclosure Next.js 16 builder skill
│           ├── SKILL.md
│           ├── scripts/verify-frontend.sh
│           ├── references/nextjs16-ui-conventions.md
│           └── assets/{schema.template.ts, schema.test.template.ts}
└── docs/
    ├── agentic_setup/
    │   └── AGENTIC_SETUP.md                           # This comprehensive setup & rationale reference log
    ├── specs/
    │   └── _template.md                               # Machine-parsable polyglot Feature Spec Blueprint
    └── adr/
        ├── README.md                                  # Scannable progressive-disclosure ADR index
        ├── _template.md                               # Standard NutriSync ADR template
        ├── 0001-record-architecture-decisions.md
        ├── 0002-single-db-ownership-go-gorm.md
        └── 0003-two-tier-pii-and-3-layer-ai.md
```
