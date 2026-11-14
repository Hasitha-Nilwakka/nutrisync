# ADR-0003: Stateless Python AI Service with Two-Tier PII Anonymization and 3-Layer Guardrails

- **Status:** Accepted
- **Date:** 2026-11-14
- **Deciders:** NutriSync Engineering Team
- **Affected Services:** `Go Core API` (`services/go-api`), `Python Analytics & AI` (`services/py-ai`)
- **Supersedes:** None

---

## 1. Context & Problem Statement

NutriSync generates personalized health scores and AI wellness insights from sensitive biometric histories, dietary restrictions, and free-text medical conditions. Sending raw user records or unbounded historical logs directly to an LLM introduces three major hazards:
1. **GDPR & PII Leakage:** User names, emails, or free-text identifiers inside `medical_conditions` could leak into LLM prompts.
2. **Hallucinated Math & Context Overflow:** LLMs are unreliable at arithmetic (such as BMI or weighted wellness scores) and degrade when fed raw unbounded time-series arrays.
3. **Unsafe Clinical / Dietary Advice:** An unguarded LLM could recommend allergens or exercises contraindicated by the user's stated restrictions or medical conditions.

## 2. Decision Drivers

- Enforce zero-trust PII protection across service and LLM boundaries (GDPR compliance).
- Keep all biometric and Wellness Score math 100% deterministic and reproducible.
- Deterministically block any AI recommendation that violates user dietary/medical restrictions, and guarantee graceful fallback when the LLM is offline or rate-limited.

## 3. Considered Options

1. **Option 1: Two-Tier PII Anonymization (`go-api` + `py-ai` Layer 1) + Vectorized `Pandas`/`NumPy` Wellness Engine + 3-Layer SciSpacy/LLaMA Guardrail Pipeline with deterministic clinical fallback.**
2. **Option 2: Call an external LLM API directly from `services/go-api` with raw user profile JSON.** (Fails PII scrubbing on free text, lacks Pandas time-series compaction, and lacks biomedical entity extraction.)
3. **Option 3: Let the LLM compute both the Wellness Score and insights in a single prompt.** (Non-deterministic math and high hallucination risk.)

## 4. Decision Outcome

**Chosen Option:** **Option 1**.
- **Two-Tier PII Anonymization:**
  1. `services/go-api/health/ai_orchestrator_client.go` strips all account identifiers (`userID`, email, name, OAuth tokens) and sends only `AnonymizedAIAnalysisPayload`.
  2. `services/py-ai/pipeline/layer1_scispacy_extractor.py` performs secondary CPU scrubbing on free-text `medical_conditions` to strip residual emails, phone numbers, or person names.
- **Deterministic Analytics (`wellness_engine.py`):** `Pandas` and `NumPy` compute the exact 4-part Wellness Score (`0.3 BMI + 0.3 Activity + 0.2 Progress + 0.2 Habits`, clamped to `[0.0, 100.0]`) and compact `weight_history` / `activity_history` into 7d/30d trends.
- **3-Layer AI Guardrail Pipeline:** Layer 1 extracts biomedical/dietary forbidden terms; Layer 2 builds compacted `"few_shot"` / `"zero_shot"` LLaMA prompts; Layer 3 filters out any recommendation matching Layer 1 forbidden terms and falls back to a deterministic clinical summary if the LLM is unavailable.

## 5. Consequences

### Positive
- Mathematical metrics are 100% deterministic and unit-tested.
- Even if a user types their email or phone number into `medical_conditions`, Layer 1 scrubs it before Layer 2 constructs the prompt.
- Contraindicated dietary/medical advice is deterministically blocked by Layer 3.

### Negative / Trade-offs
- Requires maintaining strict JSON schema synchronization between Go (`ai_orchestrator_client.go`) and Python (`schemas/analysis_schemas.py`).

## 6. Compliance & Enforcement

- **`AGENTS.md` / Service Rules:** Codified in `AGENTS.md` §6, §7, & §9 and `services/py-ai/AGENTS.md`.
- **Automated Tests & Skills:** Enforced via `.agents/skills/build-py-ai-pipeline/` and `.agents/skills/review/SKILL.md` (Pillar 3 & Pillar 4).
