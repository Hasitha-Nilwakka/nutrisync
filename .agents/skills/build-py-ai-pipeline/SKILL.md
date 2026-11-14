---
name: build-py-ai-pipeline
description: >-
  Build, extend, and verify Python FastAPI health analytics, Pandas/NumPy Wellness Score
  calculations, and the 3-Layer SciSpacy/LLaMA guardrail pipeline in services/py-ai.
  Use whenever modifying schemas, analytics, or AI pipeline modules in services/py-ai.
---

# Build Python Analytics & 3-Layer AI Pipeline (`build-py-ai-pipeline`)

Follow this runbook when creating or modifying modules inside `services/py-ai` (`:8000`) for **NutriSync (`numbers-dont-lie`)**.

---

## 1. Progressive Context & Boilerplate

Before writing code, consult these bundled resources as needed:
- **Wellness Score & 3-Layer Guardrail Rules:** Read [`references/wellness-and-guardrails.md`](./references/wellness-and-guardrails.md) for the exact 4-part Wellness Score formula, `Pandas`/`NumPy` compaction requirements, and Layer 1–3 PII/contraindication rules.
- **Pytest Starter Template:** Copy and adapt [`assets/test_pipeline.template.py`](./assets/test_pipeline.template.py).

---

## 2. Step-by-Step Implementation Flow

1. **Maintain Go <-> Python Contract Synchronization (`schemas/analysis_schemas.py`):**
   - Inbound requests use `snake_case` (`AnonymizedAIAnalysisPayload`), while outbound responses inherit from `CamelCaseBaseModel` (`PyAIAnalysisResponse`) so Go's `services/go-api/health/ai_orchestrator_client.go` can unmarshal `camelCase` JSON directly.
   - Any schema change in `services/py-ai/schemas/analysis_schemas.py` requires verifying and updating `services/go-api/health/ai_orchestrator_client.go` in the same task.
2. **Vectorized Analytics (`analytics/wellness_engine.py`):**
   - Compute the 4-part Wellness Score using `round((bmi_score * 0.3) + (activity_score * 0.3) + (progress_score * 0.2) + (habits_score * 0.2), 2)` with each component and final score clamped to `[0.0, 100.0]`.
   - Use `Pandas` and `NumPy` to compact `weight_history` and `activity_history` into 7-day and 30-day moving averages and slopes—**never** pass raw unbounded arrays to the LLM prompt.
3. **3-Layer AI Pipeline (`pipeline/`):**
   - **Layer 1 (`layer1_scispacy_extractor.py`):** Scrub residual free-text PII (emails, phone numbers, person names) from `medical_conditions` and map `restrictions` + extracted conditions into a deterministic forbidden-term set.
   - **Layer 2 (`layer2_prompt_constructor.py`):** Inject compacted `Pandas` trends, `primary_goal`, and Layer 1 constraints using `"few_shot"` or `"zero_shot"` prompting and LLaMA chat templates.
   - **Layer 3 (`layer3_llama_validator.py`):** Inspect every generated recommendation against Layer 1's forbidden terms, filter or replace violating items, verify goal alignment, and provide a deterministic clinical fallback when the LLM endpoint is offline or rate-limited.
4. **Code Quality & Stateless Boundary:**
   - Every function must do **one job** and use a **full, descriptive `snake_case` name** (never abbreviate).
   - **Zero Database Access:** `services/py-ai` must never import database drivers or connect to PostgreSQL directly.
5. **Add Pytest Unit Coverage (`services/py-ai/test_*.py`):**
   - Adapt [`assets/test_pipeline.template.py`](./assets/test_pipeline.template.py) to test Wellness Score clamping (`[0.0, 100.0]`), PII scrubbing, forbidden-term removal, and offline LLM fallback.

---

## 3. Deterministic Verification

Run the bundled verification script to check for forbidden DB imports, syntax errors, and full `pytest` pass status:

```bash
./.agents/skills/build-py-ai-pipeline/scripts/verify-py-ai.sh
```
