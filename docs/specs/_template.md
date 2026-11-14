# Feature Spec: [Feature Name]

- **Status:** Draft | Approved | In Progress | Completed
- **Author:** [Developer / Agent]
- **Target Issue / PR:** #[Number]
- **Risk Level (`AGENTS.md` §10):** Low | Medium | High | Critical
- **Approval Gate:** Not Required | Pending User Approval | Approved by [Name]
- **Affected Services:**
  - [ ] **Next.js Frontend (`/`)** — `app/`, `feature/`, `components/ui/`, `lib/`, `next.config.ts`
  - [ ] **Go Core API (`services/go-api/`)** — `handlers/`, `repositories/`, `models/`, `database/`, `auth/`, `health/`
  - [ ] **Python Analytics & AI (`services/py-ai/`)** — `schemas/`, `analytics/`, `pipeline/`

---

## 1. Objective & Scope

- **Problem Statement:** What user, clinical/analytics, or system problem does this solve?
- **In Scope:** Concrete deliverables and files included in this iteration.
- **Out of Scope:** What we are explicitly NOT building or refactoring in this task (enforces smallest safe change).
- **Acceptance Criteria:**
  - [ ] Criterion 1 (measurable & testable)
  - [ ] Criterion 2 (measurable & testable)
- **Pre-Flight MCP & Context Verification:**
  - [ ] Checked live PostgreSQL schema via `postgres-readonly` MCP (if touching DB/GORM models).
  - [ ] Checked version-accurate library docs via `context7` MCP or `node_modules/next/dist/docs/` (if using Next.js 16, Zod v4, Tailwind v4, GORM, or FastAPI APIs).
  - [ ] Checked GitHub Issue / PR requirements via `github` MCP (if applicable).

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
> **Rule (`AGENTS.md` §8):** PostgreSQL is owned exclusively by `services/go-api` via GORM. Never use legacy Prisma files. Schema changes require explicit user approval before editing.
- **Tables / Models Affected:** [e.g., `User`, `HealthProfile`, `WeightEntry`, `ActivityEntry`, `AnalyticsSnapshot`, or `None`]
- **Columns, Indexes & Constraints:** [Detail exact GORM struct tags, foreign keys, and nullable pointer semantics]
- **Migration & Transaction Strategy:** [Does `database.RunDatabaseMigrations` (`AutoMigrate`) cover this safely? Are multi-step writes wrapped in `db.Transaction(...)`?]

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints (`:8080`):** [Method, path, handler name in `services/go-api/handlers/`, and whether `next.config.ts` needs a matching `/api/*` rewrite]
- **Repository & Business Logic:** [New/updated single-purpose functions in `repositories/`, `auth/`, or `health/normalization.go` using full, unabbreviated `PascalCase`/`camelCase` names]
- **Go <-> Python AI Contract Sync:** [If modifying `services/go-api/health/ai_orchestrator_client.go`, specify the exact matching Pydantic fields in `services/py-ai/schemas/analysis_schemas.py`]

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- **Pydantic v2 Schemas (`schemas/analysis_schemas.py`):** [Request/response models and field bounds]
- **Vectorized Analytics (`analytics/wellness_engine.py`):** [Any `Pandas`/`NumPy` 7d/30d trend calculations or BMI/Wellness Score interactions]
- **3-Layer AI Pipeline (`pipeline/`):**
  - **Layer 1 (`layer1_scispacy_extractor.py`):** [Secondary PII scrubbing & biomedical/dietary forbidden-term extraction]
  - **Layer 2 (`layer2_prompt_constructor.py`):** [Compacted statistical context & `"few_shot"` / `"zero_shot"` prompt template updates]
  - **Layer 3 (`layer3_llama_validator.py`):** [Post-generation forbidden-term validation, goal alignment check, and deterministic clinical fallback]

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `lib/schemas/`)
- **Routes & Feature Modules:** [Thin route files in `app/` and domain components/wizards in `feature/`]
- **Zod v4 Schemas & Forms (`lib/schemas/`):** [Client/form validation rules and React Hook Form bindings]
- **API Integration:** [Axios calls via relative `/api/*` Next.js rewrites — never hardcode backend URLs]

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **User Isolation & Authentication (`AGENTS.md` §6):**
  - How is the acting `userID` derived from validated server-side JWT/session cookies (`access_token` / `refresh_token`)?
  - How is every GORM query in `services/go-api/repositories/` scoped to `userID` to prevent cross-user access (`401` / `403`)?
- **GDPR & Two-Tier PII Anonymization (`AGENTS.md` §6):**
  - Does this touch health data requiring GDPR consent (`POST /api/auth/consent`) or data portability (`GET /api/health/export`)?
  - How do we guarantee zero account identifiers (email, name, `userID`, tokens) cross into `AnonymizedAIAnalysisPayload` and zero free-text PII reaches Layer 2 prompts?
- **Biometrics & AI Safety Guardrails (`AGENTS.md` §7):**
  - Are anthropometric/activity inputs normalized via `normalization.go` within physiological bounds?
  - Are Wellness Score weights (`0.3 BMI + 0.3 Activity + 0.2 Progress + 0.2 Habits`, clamped to `[0.0, 100.0]`) and Layer 3 contraindication filters preserved?
- **Failure Modes & Safe Error Handling (`AGENTS.md` §9):**
  - How are invalid payloads, expired sessions, DB errors, or `services/py-ai` timeouts handled? (Must return structured `{"error": "..."}` JSON with zero stack traces, SQL internals, or secret/PII logging).

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [ ] **Step 1 — Contracts & Schemas:** Update GORM models / Pydantic v2 schemas / Zod v4 schemas (with approval if required).
- [ ] **Step 2 — Backend / AI Core Logic:** Implement single-purpose, descriptively named functions in `services/go-api` and/or `services/py-ai` with file- and function-level comments.
- [ ] **Step 3 — Frontend & API Rewrites:** Implement `feature/` components, `app/` routes, and `next.config.ts` rewrites if needed.
- [ ] **Step 4 — Unit & Negative Security Tests:**
  - [ ] Happy-path unit tests added/updated.
  - [ ] Boundary validation failure tests added (`400 Bad Request` / Zod rejection).
  - [ ] Negative authorization / cross-user isolation tests added (`401 Unauthorized` / `403 Forbidden`).

### Verification Commands (`AGENTS.md` §4 — Only check off commands actually executed)
- [ ] **Frontend Typecheck:** `pnpm exec tsc --noEmit`
- [ ] **Frontend Lint:** `pnpm lint` (or `pnpm exec eslint <modified-files>`)
- [ ] **Frontend Unit Tests:** `pnpm test` (or `pnpm exec vitest run <path-to-test>`)
- [ ] **Go API Unit Tests:** `cd services/go-api && go test -v ./...`
- [ ] **Python AI Unit Tests:** `cd services/py-ai && .venv/bin/pytest -v`

---

## 5. Open Questions & Decisions Required

- [ ] [List any unresolved clinical/scoring rules, GDPR/privacy requirements, schema trade-offs, or UX ambiguities requiring human sign-off before implementation]

