# Feature Spec: Python Wellness Score & Time-Series Compaction Engine (Step A)

- **Status:** Completed
- **Author:** Antigravity Agent
- **Target Issue / PR:** N/A (Core Python Analytics Engine — Step A)
- **Risk Level (`AGENTS.md` §10):** High (Touches Wellness Score math & Go<->Python analytics contract)
- **Approval Gate:** Approved
- **Affected Services:**
  - [ ] **Next.js Frontend (`/`)** — `app/`, `feature/`, `components/ui/`, `lib/`, `next.config.ts`
  - [ ] **Go Core API (`services/go-api/`)** — `handlers/`, `repositories/`, `models/`, `database/`, `auth/`, `health/`
  - [x] **Python Analytics & AI (`services/py-ai/`)** — `schemas/`, `analytics/`, `pipeline/`

---

## 1. Objective & Scope

- **Problem Statement:**
  The Go Core API (`services/go-api/health/ai_orchestrator_client.go`) already builds a PII-stripped `AnonymizedAIAnalysisPayload` and prepares to send it to `services/py-ai` (`POST /api/v1/analyze`). Inside `services/py-ai`, the Pydantic v2 request/response schemas (`schemas/analysis_schemas.py`) are complete, but `analytics/wellness_engine.py` was paused mid-implementation at line 21 (with an incomplete `calculate_bmi_component_score` function and an operator-precedence bug in `calculate_bmi_and_classification`). Before building the 3-Layer AI Pipeline (Step B) and wiring the FastAPI endpoint (Step C), we must complete and unit-test the deterministic `Pandas`/`NumPy` Wellness Score & Milestone engine and the 7-day/30-day time-series context compaction module.
- **In Scope:**
  1. Complete `services/py-ai/analytics/wellness_engine.py` with single-purpose, descriptively named functions (`snake_case`) covering:
     - `clamp_score_to_hundred_scale(score: float) -> float`
     - `calculate_bmi_and_classification(weight_kg: float, height_cm: float) -> tuple[float, str]` (fixing the `height_meters ** 2` denominator and returning native Python `float` and `str` types).
     - `calculate_bmi_component_score(bmi: float) -> float`
     - `map_activity_level_to_rank(activity_level: str) -> float`
     - `count_recent_weekly_activity_sessions(activity_history: list[AnonymizedActivityHistoryPoint]) -> tuple[int, int]`
     - `calculate_activity_component_score(...) -> float`
     - `calculate_progress_component_score(...) -> tuple[float, float]`
     - `calculate_habits_component_score(sleep_hours: float, water_liters: float) -> float`
     - `calculate_composite_wellness_score(bmi_score: float, activity_score: float, progress_score: float, habits_score: float) -> float`
     - `build_wellness_milestones(...) -> list[WellnessMilestone]`
     - `calculate_deterministic_wellness_and_milestones(user_metrics: AnonymizedUserMetrics) -> WellnessCalculationResult`
     - `compact_historical_metrics_for_ai_context(weight_history: list[AnonymizedWeightHistoryPoint], activity_history: list[AnonymizedActivityHistoryPoint]) -> dict[str, object]` (using `Pandas` and `NumPy` to compute 7-day and 30-day moving averages, net deltas, and linear trend slopes so raw unbounded arrays are never passed to Layer 2 LLM prompts).
  2. Create `services/py-ai/analytics/__init__.py` for clean module imports.
  3. Create comprehensive `pytest` unit tests in `services/py-ai/test_wellness_engine.py` verifying BMI classifications, component and composite score clamping (`[0.0, 100.0]`), exact formula parity with Go's `CalculateDeterministicWellnessAndMilestones`, milestone generation, and `Pandas`/`NumPy` 7d/30d time-series compaction across empty, single-point, and multi-point histories.
- **Out of Scope:**
  - Modifying `services/go-api/` or `Next.js` frontend files.
  - Implementing the 3-Layer AI Pipeline (`pipeline/layer1_scispacy_extractor.py`, `layer2_prompt_constructor.py`, `layer3_llama_validator.py` — reserved for Step B).
  - Wiring `POST /api/v1/analyze` in `services/py-ai/main.py` (reserved for Step C).
  - Installing any new third-party Python dependencies (`numpy` and `pandas` are already present in `services/py-ai/requirements.txt`).
- **Acceptance Criteria:**
  - [x] `calculate_bmi_and_classification` accurately calculates $\text{BMI} = \text{round}(\text{weight\_kg} / (\text{height\_cm} / 100)^2, 2)$ and classifies into `"underweight"`, `"normal weight"`, `"overweight"`, or `"obese"`.
  - [x] `calculate_composite_wellness_score` strictly enforces $\text{wellness\_score} = \text{round}((\text{bmi\_score} \times 0.3) + (\text{activity\_score} \times 0.3) + (\text{progress\_score} \times 0.2) + (\text{habits\_score} \times 0.2), 2)$ with all component and composite scores clamped to `[0.0, 100.0]`.
  - [x] `calculate_deterministic_wellness_and_milestones` produces exact parity with Go's `CalculateDeterministicWellnessAndMilestones` (`services/go-api/health/ai_orchestrator_client.go`), including the 3 milestones (`weight-5pct-step`, `weekly-activity-target`, `daily-habits-consistency`) and serializes to `camelCase` via `WellnessCalculationResult`.
  - [x] `compact_historical_metrics_for_ai_context` uses `Pandas` DataFrames/Series and `NumPy` to compute 7-day and 30-day weight/activity summaries and slopes, gracefully handling empty or 1-point arrays without `NaN` or division-by-zero errors.
  - [x] All unit tests pass via `./.agents/skills/build-py-ai-pipeline/scripts/verify-py-ai.sh`.
- **Pre-Flight MCP & Context Verification:**
  - [x] Checked live PostgreSQL schema via `postgres-readonly` MCP (Not applicable — `services/py-ai` is strictly stateless with zero DB access per `ADR-0002`).
  - [x] Checked version-accurate library docs & existing Pydantic v2 / Pandas / NumPy conventions in `services/py-ai/schemas/analysis_schemas.py` and `services/go-api/health/ai_orchestrator_client.go`.
  - [x] Checked GitHub Issue / PR requirements via `github` MCP (Not applicable — internal Step A milestone).

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
- **Tables / Models Affected:** `None` (`services/py-ai` is stateless and must never import database drivers or connect to PostgreSQL per `ADR-0002` and `AGENTS.md` §8).

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints (`:8080`):** No changes in Step A.
- **Go <-> Python AI Contract Sync:**
  `services/py-ai/analytics/wellness_engine.py` consumes `AnonymizedUserMetrics` from `services/py-ai/schemas/analysis_schemas.py` and returns `WellnessCalculationResult` (which inherits from `CamelCaseBaseModel` to serialize `bmiClassification`, `wellnessScore`, `bmiScore`, `activityScore`, `progressScore`, `habitsScore`, and `milestones[].progressPct` in exact alignment with `services/go-api/health/ai_orchestrator_client.go`).

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- **Pydantic v2 Schemas (`schemas/analysis_schemas.py`):**
  Uses existing `AnonymizedUserMetrics`, `AnonymizedWeightHistoryPoint`, `AnonymizedActivityHistoryPoint`, `WellnessMilestone`, and `WellnessCalculationResult`. No schema modifications required.
- **Vectorized Analytics (`analytics/wellness_engine.py`):**
  1. **BMI & Component Score (`bmi_score`):**
     - Normal `[18.5, 25.0)`: `clamp(100.0 - abs(bmi - 21.7) * 3.0)`
     - Underweight `< 18.5`: `clamp(85.0 - (18.5 - bmi) * 10.0)`
     - Overweight `[25.0, 30.0)`: `clamp(85.0 - (bmi - 25.0) * 6.0)`
     - Obese `>= 30.0`: `clamp(55.0 - (bmi - 30.0) * 3.5)`
  2. **Activity Component Score (`activity_score`):**
     - Combines `level_score = (activity_level_rank / 4.0) * 30.0`, `frequency_score = min(effective_active_days / target_days, 1.0) * 50.0`, and `logged_bonus = min(recent_sessions * 5.0 + total_minutes / 30.0, 20.0)`, clamped to `[0.0, 100.0]`.
  3. **Progress Component Score (`progress_score`):**
     - Evaluates trajectory from `baseline_weight` toward `target_weight`: if `total_goal_delta < 0.25`, returns `95.0` when `remaining_delta <= 1.0` else `clamp(90.0 - remaining_delta * 8.0)`. Otherwise returns `clamp(40.0 + (weight_progress_fraction * 60.0))` where `weight_progress_fraction` is clamped to `[0.0, 1.0]`.
  4. **Habits Component Score (`habits_score`):**
     - Evaluates sleep (`50.0` for `7.0..9.0`h, `38.0` for `6.0..<7.0`h, `40.0` for `>9.0..10.0`h, `max(15.0, 50.0 - abs(7.5 - sleep_hours) * 10.0)` for other `> 0`, `35.0` default) + hydration (`min(water_liters / 2.5, 1.0) * 50.0` when `> 0`, `25.0` default), clamped to `[0.0, 100.0]`.
  5. **Composite Wellness Score (`wellness_score`):**
     - `round((bmi_score * 0.3) + (activity_score * 0.3) + (progress_score * 0.2) + (habits_score * 0.2), 2)` clamped to `[0.0, 100.0]`.
  6. **Time-Series Context Compaction (`compact_historical_metrics_for_ai_context`):**
     - Converts `weight_history` and `activity_history` into `Pandas` DataFrames sorted chronologically by `recorded_at`.
     - Computes 7-day and 30-day window averages, net weight change (`delta_kg`), linear daily slope (`NumPy` `polyfit` when $\ge 2$ points, `0.0` otherwise), total activity sessions/minutes in 7d and 30d windows, and dominant exercise intensity/type.
- **3-Layer AI Pipeline (`pipeline/`):** Not modified in Step A; Layer 2 (`layer2_prompt_constructor.py`) will consume the output of `compact_historical_metrics_for_ai_context` in Step B.

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `lib/schemas/`)
- Not affected in Step A.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **User Isolation & Authentication (`AGENTS.md` §6):**
  `services/py-ai` only processes `AnonymizedUserMetrics` after `services/go-api` validates the user's session cookie and hashes `userID` into `subject_id`.
- **GDPR & Two-Tier PII Anonymization (`AGENTS.md` §6):**
  `wellness_engine.py` operates strictly on numeric/categorical biometric series (`weight_kg`, `height_cm`, `duration_minutes`, `recorded_at`) and never logs or persists user data.
- **Biometrics & AI Safety Guardrails (`AGENTS.md` §7):**
  - Preserves the exact `0.3 / 0.3 / 0.2 / 0.2` Wellness Score component weights and clamps all scores to `[0.0, 100.0]`.
  - Prevents context-window overflow and LLM math hallucination by compacting unbounded `weight_history` and `activity_history` arrays via `Pandas` and `NumPy` before prompt construction.
- **Failure Modes & Safe Error Handling (`AGENTS.md` §9):**
  - Handles empty `weight_history` / `activity_history`, single-entry arrays, unsorted timestamps, and zero/missing `baseline_weight` or `target_weekly_days` deterministically without raising unhandled `ZeroDivisionError` or `NaN` serialization errors.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Contracts & Module Setup:** Add `services/py-ai/analytics/__init__.py` and verify imports against `services/py-ai/schemas/analysis_schemas.py`.
- [x] **Step 2 — Wellness Score & Milestone Engine:** Complete `services/py-ai/analytics/wellness_engine.py` with single-purpose, descriptively named functions for BMI, the 4 component scores, composite Wellness Score, milestones, and `Pandas`/`NumPy` 7d/30d time-series compaction.
- [x] **Step 3 — Unit & Edge-Case Tests:** Create `services/py-ai/test_wellness_engine.py` testing:
  - [x] Happy-path BMI, component scores, composite `0.3/0.3/0.2/0.2` formula, and 3 milestones.
  - [x] Score clamping at boundaries (`[0.0, 100.0]`) and parity with Go's `TestCalculateDeterministicWellnessAndMilestonesFormulaAndMilestones` fixture.
  - [x] `Pandas`/`NumPy` 7d/30d compaction with empty history, single-point history, and multi-week history (verifying zero `NaN` values and accurate slopes/moving averages).

### Verification Commands (`AGENTS.md` §4 — Only check off commands actually executed)
- [x] **Python AI Architecture & Pytest Suite:** `./.agents/skills/build-py-ai-pipeline/scripts/verify-py-ai.sh` (`cd services/py-ai && .venv/bin/pytest -v`)

---

## 5. Open Questions & Decisions Required

- [x] **No open clinical or contract ambiguities:** All component score formulas and milestone structures are explicitly defined in `services/go-api/health/ai_orchestrator_client.go` (`CalculateDeterministicWellnessAndMilestones`) and `AGENTS.md` §7. Ready for user approval to execute `/implement`.
