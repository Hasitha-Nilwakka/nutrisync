# Feature Spec: Python AI `/api/v1/analyze` Endpoint (Step C)

- **Status:** Completed
- **Author:** Antigravity Agent
- **Target Issue / PR:** N/A (Step C of `python-ai-engine`)
- **Risk Level (`AGENTS.md` §10):** Medium (FastAPI endpoint wiring Step A + Step B, Go <-> Python contract boundary)
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [ ] **Next.js Frontend (`/`)** — `app/`, `feature/`, `components/ui/`, `lib/`, `next.config.ts`
  - [ ] **Go Core API (`services/go-api/`)** — `handlers/`, `repositories/`, `models/`, `database/`, `auth/`, `health/`
  - [x] **Python Analytics & AI (`services/py-ai/`)** — `main.py`, `test_main.py`, `schemas/`, `analytics/`, `pipeline/`

---

## 1. Objective & Scope

- **Problem Statement:** Step A (`services/py-ai/analytics/wellness_engine.py`) and Step B (`services/py-ai/pipeline/`) are implemented and tested, but `services/py-ai/main.py` still only exposes `GET /health`. `services/go-api/health/ai_orchestrator_client.go` expects `POST /api/v1/analyze` returning `PyAIAnalysisResponse` in `camelCase` JSON.
- **In Scope:**
  - Wire `POST /api/v1/analyze` in `services/py-ai/main.py` (`analyze_anonymized_health_metrics`), connecting `calculate_deterministic_wellness_and_milestones`, `compact_historical_metrics_for_ai_context`, `extract_clinical_constraints_and_scrub_pii`, and `generate_validated_ai_insights`.
  - Add safe `400 Bad Request` handling for domain `ValueError` (e.g., non-positive biometrics) and safe `500` structured JSON error handling without stack traces or PII.
  - Add FastAPI `TestClient` integration tests in `services/py-ai/test_main.py` covering end-to-end `POST /api/v1/analyze` happy path, `camelCase` response contract verification, PII scrubbing + guardrail enforcement through the HTTP endpoint, and invalid payload (`400`/`422`) rejections while preserving existing `GET /health` tests.
- **Out of Scope:**
  - Modifying `GET /health` behavior (`{"status": "ok", "service": "pi-ai"}`).
  - Modifying `services/go-api/`, Next.js frontend, `schemas/analysis_schemas.py`, `analytics/wellness_engine.py`, or `pipeline/`.
  - Installing new dependencies or modifying `Dockerfile` / `compose.yaml`.
- **Acceptance Criteria:**
  - [x] `POST /api/v1/analyze` accepts `AnonymizedAIAnalysisPayload` (`snake_case` JSON) and returns HTTP `200` with `PyAIAnalysisResponse` serialized in `camelCase` (`wellness` + `insights`).
  - [x] End-to-end HTTP integration tests verify deterministic Wellness Score parity, Layer 1 PII scrubbing, Layer 2 trend compaction, and Layer 3 fallback/guardrails through `TestClient(app)`.
  - [x] Invalid payloads return `422` (schema error) or `400` `{"error": "..."}` (domain `ValueError`, e.g., non-positive height/weight) with zero stack traces or PII in the response.
  - [x] Existing `GET /health` tests and `./.agents/skills/build-py-ai-pipeline/scripts/verify-py-ai.sh` pass.
- **Pre-Flight MCP & Context Verification:**
  - [x] Verified `services/go-api/health/ai_orchestrator_client.go` contract (`POST /api/v1/analyze`, `AnonymizedAIAnalysisPayload` -> `camelCase` `PyAIAnalysisResponse`, `2xx` expectation).
  - [x] Inspected `services/py-ai/main.py` and `services/py-ai/test_main.py`.
  - [ ] Checked live PostgreSQL schema via `postgres-readonly` MCP (Not Applicable — `services/py-ai` is stateless per `ADR-0002` and `ADR-0003`).

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
> **Rule (`AGENTS.md` §8):** PostgreSQL is owned exclusively by `services/go-api` via GORM.
- **Tables / Models Affected:** `None`. `services/py-ai` remains strictly stateless.

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints (`:8080`):** No Go changes.
- **Go <-> Python AI Contract Sync:** `POST /api/v1/analyze` on `:8000` consumes `AnonymizedAIAnalysisPayload` and returns `PyAIAnalysisResponse` (`response_model=PyAIAnalysisResponse`) serialized with `by_alias=True` (`CamelCaseBaseModel`) matching `services/go-api/health/ai_orchestrator_client.go`.

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- **Endpoint (`services/py-ai/main.py`):**
  - Preserve `app = FastAPI(title="NutriSync AI & Analytics Service")` and `@app.get("/health")` / `get_python_service_health_status()`.
  - Add `@app.post("/api/v1/analyze", response_model=PyAIAnalysisResponse)` implemented by `analyze_anonymized_health_metrics(payload: AnonymizedAIAnalysisPayload) -> PyAIAnalysisResponse`:
    1. Compute deterministic Step A metrics: `wellness_result = calculate_deterministic_wellness_and_milestones(payload.user_metrics)` and `compacted_trends = compact_historical_metrics_for_ai_context(payload.user_metrics.weight_history, payload.user_metrics.activity_history)`.
    2. Run Layer 1: `layer1_result = extract_clinical_constraints_and_scrub_pii(payload.user_metrics)`.
    3. Run Layer 2 + Layer 3: `insights_result = generate_validated_ai_insights(user_metrics=payload.user_metrics, wellness_result=wellness_result, compacted_trends=compacted_trends, layer1_result=layer1_result)`.
    4. Return `PyAIAnalysisResponse(wellness=wellness_result, insights=insights_result)`.
  - Catch `ValueError` from `calculate_deterministic_wellness_and_milestones` and return `JSONResponse(status_code=400, content={"error": str(exc)})`.
  - Add a safe fallback exception handler for unhandled exceptions returning `JSONResponse(status_code=500, content={"error": "Internal analysis error"})`.
- **Integration Tests (`services/py-ai/test_main.py`):**
  - Preserve all 3 existing `GET /health` tests.
  - Add `TestClient` integration tests:
    - `test_analyze_endpoint_returns_camel_case_wellness_and_insights`: posts full Go-compatible `AnonymizedAIAnalysisPayload`, verifies HTTP `200`, `wellness` (`bmiClassification`, `wellnessScore`, `milestones[0]["progressPct"]`), and `insights` (`promptStrategy`, `healthStatusAnalysis`, `recommendations[0]["expectedImpact"]`).
    - `test_analyze_endpoint_enforces_pii_scrubbing_and_guardrails_with_mocked_llm`: monkeypatches `LLM_API_URL` / `invoke_llama_completion_endpoint` or `httpx.Client` to verify unsafe LLM output is filtered through the HTTP endpoint.
    - `test_analyze_endpoint_rejects_non_positive_biometrics_with_http_400`: sends `weight=0` / `height=0` and asserts HTTP `400` `{"error": "..."}`.
    - `test_analyze_endpoint_rejects_malformed_payload_with_http_422`: sends missing `user_metrics` and asserts HTTP `422`.

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `lib/schemas/`)
- **Routes & Feature Modules:** None.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **User Isolation & Authentication (`AGENTS.md` §6):** Endpoint receives only `AnonymizedAIAnalysisPayload` (`subject_id` is already SHA-256 hashed by Go).
- **GDPR & Two-Tier PII Anonymization (`AGENTS.md` §6):** `extract_clinical_constraints_and_scrub_pii` runs before `generate_validated_ai_insights` on every `/api/v1/analyze` call.
- **Biometrics & AI Safety Guardrails (`AGENTS.md` §7):** `wellness_engine` computes deterministic scores clamped to `[0.0, 100.0]`, compacts historical arrays, and Layer 3 enforces `forbidden_terms` + clinical fallback.
- **Failure Modes & Safe Error Handling (`AGENTS.md` §9):** `400` and `500` responses return structured `{"error": "..."}` JSON with zero stack traces, SQL details, or unredacted `medical_conditions`.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Wire Endpoint (`services/py-ai/main.py`):** Implement `POST /api/v1/analyze` (`analyze_anonymized_health_metrics`) and safe `400`/`500` error handling while preserving `GET /health`.
- [x] **Step 2 — Integration Tests (`services/py-ai/test_main.py`):** Add FastAPI `TestClient` integration tests for happy path (`camelCase` contract), guardrail/PII enforcement, `400` domain validation rejection, and `422` schema rejection.

### Verification Commands (`AGENTS.md` §4 — Only check off commands actually executed)
- [x] **Python AI Pipeline Verification:** `./.agents/skills/build-py-ai-pipeline/scripts/verify-py-ai.sh`
- [x] **Python AI Unit Tests:** `cd services/py-ai && .venv/bin/pytest -v`
- [x] **Python Syntax Check:** `cd services/py-ai && .venv/bin/python -m compileall -q .`

---

## 5. Open Questions & Decisions Required

- None — the Go <-> Python HTTP contract (`services/go-api/health/ai_orchestrator_client.go` <-> `services/py-ai/schemas/analysis_schemas.py`), Step A (`wellness_engine.py`), and Step B (`pipeline/`) are already settled and verified.
