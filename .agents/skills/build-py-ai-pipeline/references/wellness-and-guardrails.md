# Wellness Score Math & 3-Layer AI Guardrail Reference (`services/py-ai`)

## 1. Deterministic 4-Part Wellness Score Formula (`AGENTS.md` §7)

All Wellness Score calculations in `services/py-ai/analytics/wellness_engine.py` must strictly follow:

$$\text{wellness\_score} = \text{round}\big((\text{bmi\_score} \times 0.3) + (\text{activity\_score} \times 0.3) + (\text{progress\_score} \times 0.2) + (\text{habits\_score} \times 0.2),\ 2\big)$$

- **Component Bounds:** Each individual component (`bmi_score`, `activity_score`, `progress_score`, `habits_score`) and the final `wellness_score` must be clamped to `[0.0, 100.0]`.
- **Approval Requirement:** Never alter the `0.3 / 0.3 / 0.2 / 0.2` weights without explicit user approval (`AGENTS.md` §5 & §7).
- **Time-Series Context Compaction:**
  - Convert `weight_history` and `activity_history` into `Pandas` DataFrames/Series sorted by `recorded_at`.
  - Compute 7-day and 30-day moving averages, net deltas, and linear trend slopes (`NumPy`).
  - Handle empty histories or single-point histories gracefully without `NaN` or division-by-zero errors.

---

## 2. Three-Layer AI Pipeline Architecture (`services/py-ai/pipeline/`)

### Layer 1: Local CPU Pre-Processor (`layer1_scispacy_extractor.py`)
1. **Zero-Trust Secondary PII Scrubbing:**
   - Scrub residual email addresses, phone numbers, and person names from free-text `medical_conditions` before any text reaches Layer 2.
2. **Contraindication & Forbidden-Term Mapping:**
   - Combine structured `restrictions` (e.g., `gluten_free`, `dairy_free`, `nut_allergy`, `shellfish_allergy`, `low_sodium`, `low_sugar`) and extracted biomedical entities (e.g., hypertension, diabetes, knee injury) into a deterministic list of forbidden food/exercise terms.

### Layer 2: Dynamic Prompt Constructor (`layer2_prompt_constructor.py`)
1. **Context Injection:**
   - Merge compacted 7d/30d `Pandas` trend summaries, `primary_goal`, `preferences`, and Layer 1 forbidden terms.
   - Never inject raw unbounded `weight_history` or `activity_history` arrays.
2. **Prompting Strategies:**
   - Support both `"few_shot"` and `"zero_shot"` prompt strategies and format with the LLaMA chat template.

### Layer 3: Guardrail Validator & Clinical Fallback (`layer3_llama_validator.py`)
1. **Post-Generation Contraindication Filter:**
   - Deterministically check every LLM-generated recommendation against Layer 1's forbidden terms.
   - Remove or replace any recommendation containing a contraindicated food, allergen, or exercise.
2. **Goal Alignment & Deterministic Fallback:**
   - Ensure the output references the user's stated `primary_goal`.
   - If the LLM endpoint is unreachable, times out, or returns malformed output, return a deterministic, safe clinical summary generated from the user's Wellness Score and compacted metrics.
