# Feature Spec: Python 3-Layer AI Guardrail Pipeline (Step B)

- **Status:** Completed
- **Author:** Antigravity Agent
- **Target Issue / PR:** N/A (Step B of `python-ai-engine`)
- **Risk Level (`AGENTS.md` §10):** High (AI Safety Guardrails & Two-Tier PII Anonymization)
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [ ] **Next.js Frontend (`/`)** — `app/`, `feature/`, `components/ui/`, `lib/`, `next.config.ts`
  - [ ] **Go Core API (`services/go-api/`)** — `handlers/`, `repositories/`, `models/`, `database/`, `auth/`, `health/`
  - [x] **Python Analytics & AI (`services/py-ai/`)** — `schemas/`, `analytics/`, `pipeline/`

---

## 1. Objective & Scope

- **Problem Statement:** Step A delivered deterministic biometric math (`wellness_engine.py`), but `services/py-ai` still lacks the 3-Layer AI Guardrail Pipeline defined in `docs/adr/0003-two-tier-pii-and-3-layer-ai.md` (`ADR-0003`). Without `services/py-ai/pipeline/`, free-text `medical_conditions` lack secondary PII scrubbing, compacted trends cannot be formatted into `"few_shot"` / `"zero_shot"` LLaMA prompts, and LLM outputs are not deterministically filtered against user dietary/medical contraindications or backed by an offline clinical fallback.
- **In Scope:**
  - Create `services/py-ai/pipeline/__init__.py` exporting the 3-layer pipeline public entrypoints.
  - Create `services/py-ai/pipeline/layer1_scispacy_extractor.py` for secondary CPU-local PII scrubbing and biomedical/dietary forbidden-term extraction.
  - Create `services/py-ai/pipeline/layer2_prompt_constructor.py` for `"few_shot"` and `"zero_shot"` LLaMA prompt construction using only scrubbed clinical context, `WellnessCalculationResult`, and `compact_historical_metrics_for_ai_context(...)` output.
  - Create `services/py-ai/pipeline/layer3_llama_validator.py` for configurable HTTP LLM invocation (`httpx`), Pydantic v2 parsing into `PyAIInsightsPayload`, deterministic Layer 1 forbidden-term filtering, goal-alignment enforcement, and deterministic clinical fallback.
  - Create `services/py-ai/test_pipeline.py` with comprehensive deterministic unit tests (using synthetic payloads and mocked `httpx` transports, zero live LLM or DB calls).
- **Out of Scope:**
  - Wiring `POST /api/v1/analyze` in `services/py-ai/main.py` (reserved for Step C).
  - Modifying `services/go-api/`, Next.js frontend, `services/py-ai/schemas/analysis_schemas.py`, or `services/py-ai/analytics/wellness_engine.py`.
  - Installing new third-party packages or downloading external spaCy model wheels into `services/py-ai/requirements.txt` or `.venv`.
- **Acceptance Criteria:**
  - [x] Layer 1 strips emails, phone numbers, URLs, SSN/ID-like patterns, `@handles`, and person names (when NER model is present) from `medical_conditions` without corrupting clinical terms, and deterministically builds `forbidden_terms` from `restrictions` and `medical_conditions`.
  - [x] Layer 2 never accepts or serializes raw unbounded `weight_history` or `activity_history` arrays; it builds both `"few_shot"` and `"zero_shot"` LLaMA prompts from compacted 7d/30d trends, `WellnessCalculationResult`, `primary_goal`, and scrubbed Layer 1 constraints.
  - [x] Layer 3 validates LLM JSON into `PyAIInsightsPayload`, blocks/filters any `PrioritizedAIRecommendation` or narrative text matching Layer 1 `forbidden_terms`, enforces `primary_goal` alignment, and returns a safe deterministic clinical `PyAIInsightsPayload` whenever the LLM is unconfigured, unreachable, timed out, rate-limited (`429`), or returns malformed output.
  - [x] `services/py-ai/test_pipeline.py` and `./.agents/skills/build-py-ai-pipeline/scripts/verify-py-ai.sh` pass with zero skipped or relaxed assertions.
- **Pre-Flight MCP & Context Verification:**
  - [x] Reviewed `docs/adr/0003-two-tier-pii-and-3-layer-ai.md`, `services/py-ai/AGENTS.md`, `services/py-ai/schemas/analysis_schemas.py`, `services/py-ai/analytics/wellness_engine.py`, and `services/go-api/health/ai_orchestrator_client.go`.
  - [x] Verified `.venv` installed packages (`spacy 3.8.18`, `httpx 0.28.1`, `pydantic 2.14.3` on Python `3.15.0`); confirmed neither `en_core_sci_sm` nor `en_core_web_sm` is installed in `.venv` and no `LLM_*` env vars exist yet in `.env.example` or `compose.yaml`.
  - [ ] Checked live PostgreSQL schema via `postgres-readonly` MCP (Not Applicable — `services/py-ai` is stateless per `ADR-0002` and `ADR-0003`).

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
> **Rule (`AGENTS.md` §8):** PostgreSQL is owned exclusively by `services/go-api` via GORM.
- **Tables / Models Affected:** `None`. `services/py-ai` is strictly stateless and never connects to PostgreSQL.

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints (`:8080`):** No Go changes in Step B.
- **Go <-> Python AI Contract Sync:** Output strictly conforms to `PyAIInsightsPayload` in `services/py-ai/schemas/analysis_schemas.py` (`prompt_strategy`, `health_status_analysis`, `progress_evaluation`, `recommendations: List[PrioritizedAIRecommendation]`, `weekly_summary`, `monthly_summary`), which serializes to camelCase via `CamelCaseBaseModel` to match `services/go-api/health/ai_orchestrator_client.go`.

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
Every function uses full, descriptive, unabbreviated `snake_case` names and a single clear responsibility (`services/py-ai/AGENTS.md`).

- **Layer 1 (`services/py-ai/pipeline/layer1_scispacy_extractor.py`):**
  - `Layer1ClinicalExtractionResult(BaseModel)`: `scrubbed_medical_conditions: str`, `detected_pii_categories: list[str]`, `normalized_restrictions: list[str]`, `extracted_conditions: list[str]`, `forbidden_terms: list[str]`, `dietary_constraints: list[str]`, `exercise_constraints: list[str]`.
  - `load_spacy_nlp_pipeline() -> spacy.language.Language`: Attempts `spacy.load("en_core_sci_sm")`, then `spacy.load("en_core_web_sm")`, and falls back deterministically to `spacy.blank("en")` with an `EntityRuler` for clinical/dietary pattern matching (zero network download, CPU-only).
  - `scrub_residual_pii_from_free_text(raw_text: str, nlp: spacy.language.Language | None = None) -> tuple[str, list[str]]`: Strips emails (`[REDACTED_EMAIL]`), phone numbers (`[REDACTED_PHONE]`), URLs (`[REDACTED_URL]`), SSN/ID patterns (`[REDACTED_ID]`), `@handles` (`[REDACTED_HANDLE]`), and spaCy `PERSON` entities (`[REDACTED_NAME]`) when NER is loaded.
  - `build_forbidden_terms_from_restrictions_and_conditions(restrictions: list[str], scrubbed_medical_conditions: str) -> tuple[list[str], list[str], list[str], list[str]]`: Maps canonical `restrictions` (e.g., `peanut_allergy`, `lactose_intolerant`, `gluten_free`, `shellfish_allergy`, `vegetarian`, `vegan`, `diabetes`, `hypertension`, `knee_injury`, `asthma`) and extracted free-text clinical conditions into `(normalized_restrictions, extracted_conditions, forbidden_terms, dietary_and_exercise_constraints)`.
  - `extract_clinical_constraints_and_scrub_pii(user_metrics: AnonymizedUserMetrics) -> Layer1ClinicalExtractionResult`: Public Layer 1 entrypoint.

- **Layer 2 (`services/py-ai/pipeline/layer2_prompt_constructor.py`):**
  - `ConstructedLlamaPrompt(BaseModel)`: `prompt_strategy: Literal["few_shot", "zero_shot"]`, `system_prompt: str`, `user_prompt: str`, `formatted_llama_prompt: str`, `messages: list[dict[str, str]]`.
  - `select_prompt_strategy(user_metrics: AnonymizedUserMetrics, compacted_trends: dict[str, object], requested_strategy: str | None = None) -> Literal["few_shot", "zero_shot"]`: Honors explicit `"few_shot"` / `"zero_shot"` override; defaults to `"few_shot"` when historical trend data or medical/dietary constraints are present, and `"zero_shot"` otherwise.
  - `build_few_shot_examples_block() -> str`: Provides compact anonymized JSON examples adhering to `PyAIInsightsPayload`.
  - `format_compacted_metrics_summary(user_metrics: AnonymizedUserMetrics, wellness_result: WellnessCalculationResult, compacted_trends: dict[str, object], layer1_result: Layer1ClinicalExtractionResult) -> str`: Formats `WellnessCalculationResult`, `compacted_trends` (`weight_trends`, `activity_trends`), `primary_goal`, `preferences`, `scrubbed_medical_conditions`, and `forbidden_terms` — never raw `weight_history` or `activity_history`.
  - `construct_llama_prompt_payload(user_metrics: AnonymizedUserMetrics, wellness_result: WellnessCalculationResult, compacted_trends: dict[str, object], layer1_result: Layer1ClinicalExtractionResult, requested_strategy: str | None = None) -> ConstructedLlamaPrompt`: Public Layer 2 entrypoint.

- **Layer 3 (`services/py-ai/pipeline/layer3_llama_validator.py`):**
  - `LLMClientConfiguration(BaseModel)`: Reads optional `LLM_API_URL`, `LLM_MODEL` (default `"llama3.1"`), `LLM_API_KEY`, and `LLM_TIMEOUT_SECONDS` (default `10.0`) from `os.environ`.
  - `invoke_llama_completion_endpoint(constructed_prompt: ConstructedLlamaPrompt, configuration: LLMClientConfiguration | None = None, http_client: httpx.Client | None = None) -> str | None`: Sends completion/chat request via `httpx.Client`; returns `None` immediately if `LLM_API_URL` is unset or on timeout, connection failure, `429` rate limit, or non-2xx HTTP status.
  - `parse_llm_insights_json(raw_response_text: str, prompt_strategy: str) -> PyAIInsightsPayload | None`: Extracts JSON (stripping optional markdown fences) and validates via `PyAIInsightsPayload.model_validate`.
  - `find_matching_forbidden_term(text: str, forbidden_terms: list[str]) -> str | None`: Case-insensitive word-boundary check against Layer 1 `forbidden_terms`.
  - `filter_and_sanitize_insights_against_guardrails(insights: PyAIInsightsPayload, layer1_result: Layer1ClinicalExtractionResult, primary_goal: str) -> PyAIInsightsPayload`: Drops any `PrioritizedAIRecommendation` containing a forbidden term, redacts/replaces forbidden terms if present in narrative summaries, and guarantees at least one safe, `primary_goal`-aligned recommendation.
  - `build_deterministic_clinical_fallback_insights(user_metrics: AnonymizedUserMetrics, wellness_result: WellnessCalculationResult, compacted_trends: dict[str, object], layer1_result: Layer1ClinicalExtractionResult, prompt_strategy: str) -> PyAIInsightsPayload`: Generates safe, deterministic, restriction-compliant `PyAIInsightsPayload` from Step A metrics and trends when the LLM is unavailable or yields invalid output.
  - `generate_validated_ai_insights(user_metrics: AnonymizedUserMetrics, wellness_result: WellnessCalculationResult, compacted_trends: dict[str, object], layer1_result: Layer1ClinicalExtractionResult, requested_strategy: str | None = None, http_client: httpx.Client | None = None) -> PyAIInsightsPayload`: Public Layer 3 entrypoint.

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `lib/schemas/`)
- **Routes & Feature Modules:** None in Step B.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **User Isolation & Authentication (`AGENTS.md` §6):** `services/py-ai` receives only `AnonymizedAIAnalysisPayload` (hashed `subject_id`, zero user email/name/tokens) from `services/go-api`.
- **GDPR & Two-Tier PII Anonymization (`AGENTS.md` §6):** Layer 1 (`layer1_scispacy_extractor.py`) performs mandatory zero-trust secondary scrubbing on `medical_conditions` before Layer 2 builds any prompt, ensuring residual emails, phone numbers, URLs, IDs, or names never reach the LLM.
- **Biometrics & AI Safety Guardrails (`AGENTS.md` §7):**
  - Layer 2 receives only Step A's deterministic `WellnessCalculationResult` and `compact_historical_metrics_for_ai_context` trends (never raw unbounded arrays).
  - Layer 3 deterministically filters every recommendation and narrative field against Layer 1 `forbidden_terms` and falls back to `build_deterministic_clinical_fallback_insights` whenever the LLM fails or is offline.
- **Failure Modes & Safe Error Handling (`AGENTS.md` §9):** Network timeouts, `429` rate limits, missing `LLM_API_URL`, or malformed LLM JSON never raise unhandled exceptions or log unredacted `medical_conditions`; Layer 3 degrades cleanly to the deterministic clinical fallback.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Package & Layer 1 (`layer1_scispacy_extractor.py`):** Create `services/py-ai/pipeline/__init__.py` and `services/py-ai/pipeline/layer1_scispacy_extractor.py` with CPU-local spaCy loading (`en_core_sci_sm` -> `en_core_web_sm` -> `spacy.blank("en")` fallback), regex/NER PII scrubbing, and biomedical/dietary forbidden-term extraction.
- [x] **Step 2 — Layer 2 (`layer2_prompt_constructor.py`):** Implement `"few_shot"` and `"zero_shot"` LLaMA prompt construction consuming only `WellnessCalculationResult`, `compact_historical_metrics_for_ai_context`, and `Layer1ClinicalExtractionResult`.
- [x] **Step 3 — Layer 3 (`layer3_llama_validator.py`):** Implement `httpx` LLM client (`LLM_API_URL`, `LLM_MODEL`, `LLM_API_KEY`, `LLM_TIMEOUT_SECONDS`), Pydantic v2 response parsing, Layer 1 forbidden-term filtering, goal alignment, and deterministic clinical fallback.
- [x] **Step 4 — Unit & Guardrail Tests (`services/py-ai/test_pipeline.py`):**
  - [x] Layer 1 PII scrubbing tests (emails, phones, URLs, IDs, handles in `medical_conditions` redacted; clinical terms preserved; `forbidden_terms` extracted for allergies/conditions).
  - [x] Layer 2 `"few_shot"` and `"zero_shot"` prompt tests (verifying compacted trends present, raw `weight_history`/`activity_history` timestamps/arrays absent, and PII scrubbed).
  - [x] Layer 3 guardrail filtering tests (mocked LLM returning contraindicated food/exercise is blocked/replaced; `primary_goal` alignment preserved).
  - [x] Layer 3 offline/rate-limit/malformed-JSON fallback tests (`LLM_API_URL` unset, `httpx.TimeoutException`, HTTP `429`, invalid JSON all return valid `PyAIInsightsPayload` via `CamelCaseBaseModel`).

### Verification Commands (`AGENTS.md` §4 — Only check off commands actually executed)
- [x] **Python AI Pipeline Verification:** `./.agents/skills/build-py-ai-pipeline/scripts/verify-py-ai.sh`
- [x] **Python AI Unit Tests:** `cd services/py-ai && .venv/bin/pytest -v`
- [x] **Python Syntax Check:** `cd services/py-ai && .venv/bin/python -m compileall -q .`

---

## 5. Open Questions & Decisions Required

- [ ] **Q1 — spaCy Model Strategy:** `.venv` currently has `spacy==3.8.18` without `en_core_sci_sm` or `en_core_web_sm` installed. Is it approved for Step B to use `spacy.blank("en")` + `EntityRuler` + regex scrubbing as the zero-download CPU baseline (auto-upgrading if `en_core_sci_sm` / `en_core_web_sm` is present), keeping `requirements.txt` unchanged?
- [ ] **Q2 — LLM Endpoint Protocol:** `.env.example` and `compose.yaml` do not yet define an LLM endpoint. Should Layer 3 default to an OpenAI-compatible `/v1/chat/completions` payload (with Ollama `/api/chat` support via `LLM_API_FORMAT` / URL detection) when `LLM_API_URL` is configured, and fall back to the deterministic clinical summary whenever `LLM_API_URL` is unset?
