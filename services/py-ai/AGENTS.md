# Python Health Analytics & 3-Layer AI Engine (`services/py-ai`)

## 1. Architecture & Folder Structure
- FastAPI service running on port `8000` (`main.py` exposes `GET /health` and `POST /api/v1/analyze`).
- Stateless compute & AI microservice: receives `AnonymizedAIAnalysisPayload` from `services/go-api` and returns `PyAIAnalysisResponse` (`{ wellness, insights }`).
- Required module boundaries:
  - `schemas/analysis_schemas.py`: Pydantic v2 request/response models matching Go's `ai_orchestrator_client.go` JSON contract.
  - `analytics/wellness_engine.py`: Vectorized `Pandas` & `NumPy` calculations for BMI, 4-part Wellness Score (`0-100`), time-series moving averages/slopes, and milestones.
  - `pipeline/layer1_scispacy_extractor.py`: Layer 1 Pre-Processor (Local CPU) — secondary PII scrubbing on free-text fields + `SciSpacy`/biomedical & dietary contraindication entity extraction + forbidden term mapping.
  - `pipeline/layer2_prompt_constructor.py`: Layer 2 Dynamic Prompt Constructor — merges compacted `Pandas` time-series summaries, stated `primary_goal`, and Layer 1 entities; supports both `"few_shot"` and `"zero_shot"` strategies and applies the LLaMA chat template.
  - `pipeline/layer3_llama_validator.py`: Layer 3 Fine-Tuned LLaMA Client & Guardrail Validator — executes LLM inference (with deterministic clinical fallback when offline/rate-limited) and validates output to filter out any recommendation violating user dietary/medical restrictions or missing goal alignment.

## 2. Domain Rules
- **Wellness Score Formula:** `wellness_score = round((bmi_score * 0.3) + (activity_score * 0.3) + (progress_score * 0.2) + (habits_score * 0.2), 2)`, with each component clamped to `[0.0, 100.0]`.
- **Context Length Compaction:** Never pass raw unbounded historical arrays directly into the LLM prompt; compact `weight_history` and `activity_history` into 7-day/30-day statistical trends using `Pandas`/`NumPy` first.
- **PII Zero-Trust:** Layer 1 must scrub any residual emails, phone numbers, or person names from free-text `medical_conditions` before Layer 2 constructs the prompt.
- **Guardrail Enforcement:** Layer 3 must deterministically inspect every recommendation against Layer 1's forbidden terms (`restrictions` + extracted conditions) and remove or replace any violating recommendation before returning.

## 3. Code Quality & Restrictions
- Every function must do **one job** and have a **full, descriptive snake_case name** (never abbreviate function names).
- **No Direct Database Access:** Never connect to PostgreSQL directly; only `services/go-api` owns the database.
- **No Unapproved Dependencies:** Do not install new packages without explicit user approval.
- Every module must have corresponding `pytest` unit tests in `tests/`.
