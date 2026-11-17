"""Layer 2: Compacted LLaMA prompt constructor supporting few-shot and zero-shot strategies.

Constructs anonymized, statistically compacted prompts from Step A's deterministic
WellnessCalculationResult, Pandas/NumPy 7d/30d trends, and Layer 1 scrubbed constraints.
Raw unbounded weight_history and activity_history arrays are never injected.
"""

import json
from typing import Any, Dict, List, Literal, Optional

from pydantic import BaseModel

from pipeline.layer1_scispacy_extractor import Layer1ClinicalExtractionResult
from schemas.analysis_schemas import AnonymizedUserMetrics, WellnessCalculationResult

PromptStrategy = Literal["few_shot", "zero_shot"]

_SYSTEM_PROMPT = (
    "You are NutriSync's clinical wellness assistant. Return ONLY valid JSON matching the exact "
    "schema: {\"health_status_analysis\": str, \"progress_evaluation\": str, "
    "\"recommendations\": [{\"title\": str, \"category\": str, \"priority\": \"high|medium|low\", "
    "\"action\": str, \"expected_impact\": str}], \"weekly_summary\": str, \"monthly_summary\": str}. "
    "Rules: (1) Base all numbers strictly on the provided deterministic metrics and trends. "
    "(2) Align recommendations with primary_goal. "
    "(3) NEVER recommend any item or exercise listed in forbidden_terms. "
    "(4) Do not invent medical diagnoses or include any personal identifiers."
)


class ConstructedLlamaPrompt(BaseModel):
    """Structured prompt payload ready for Layer 3 LLM invocation."""

    prompt_strategy: PromptStrategy
    system_prompt: str
    user_prompt: str
    formatted_llama_prompt: str
    messages: List[Dict[str, str]]


def select_prompt_strategy(
    user_metrics: AnonymizedUserMetrics,
    compacted_trends: Dict[str, Any],
    requested_strategy: Optional[str] = None,
) -> PromptStrategy:
    """Select between 'few_shot' and 'zero_shot' prompting based on context richness."""
    if requested_strategy in ("few_shot", "zero_shot"):
        return requested_strategy  # type: ignore[return-value]

    weight_trends = compacted_trends.get("weight_trends", {})
    activity_trends = compacted_trends.get("activity_trends", {})
    has_history = bool(
        (isinstance(weight_trends, dict) and weight_trends.get("points_30d", 0) > 0)
        or (isinstance(activity_trends, dict) and activity_trends.get("sessions_30d", 0) > 0)
    )
    has_constraints = bool(user_metrics.restrictions or (user_metrics.medical_conditions or "").strip())
    if has_history or has_constraints:
        return "few_shot"
    return "zero_shot"


def build_few_shot_examples_block() -> str:
    """Return a structural JSON template example without fake numbers that small models might copy."""
    example_output = {
        "health_status_analysis": (
            "Current BMI is <exact_bmi> (<exact_bmi_classification>) with a composite Wellness Score "
            "of <exact_wellness_score>/100, supported by sleep and hydration habits."
        ),
        "progress_evaluation": (
            "7-day weight change is <exact_delta_7d_kg> kg with <exact_sessions_7d> activity sessions "
            "logged toward the <exact_primary_goal> goal."
        ),
        "recommendations": [
            {
                "title": "Add Low-Impact Zone-2 Cardio",
                "category": "activity",
                "priority": "high",
                "action": "Complete two 30-minute brisk walks this week to build weekly consistency.",
                "expected_impact": "Improves cardiovascular endurance and weekly activity score.",
            },
            {
                "title": "Prioritize Balanced Whole-Food Meals",
                "category": "nutrition",
                "priority": "medium",
                "action": "Pair lean plant or whole-food protein with fiber at lunch while avoiding all restricted allergens.",
                "expected_impact": "Supports steady energy and progress toward your primary goal.",
            },
        ],
        "weekly_summary": "Consistent 7-day habits with opportunity to add one structured activity session.",
        "monthly_summary": "30-day trajectory remains aligned with sustainable progress toward <exact_primary_goal>.",
    }
    return (
        "Example JSON Response:\n"
        + json.dumps(example_output, separators=(",", ":"))
    )


def format_compacted_metrics_summary(
    user_metrics: AnonymizedUserMetrics,
    wellness_result: WellnessCalculationResult,
    compacted_trends: Dict[str, Any],
    layer1_result: Layer1ClinicalExtractionResult,
) -> str:
    """Format anonymized biometrics, wellness scores, compacted trends, and Layer 1 constraints."""
    compact_context = {
        "biometrics": {
            "age": user_metrics.current_state.age,
            "gender": user_metrics.current_state.gender,
            "weight_kg": user_metrics.current_state.weight,
            "target_weight_kg": user_metrics.target_state.weight,
            "primary_goal": user_metrics.target_state.primary_goal,
            "activity_level": user_metrics.current_state.activity_level,
            "weekly_activity_days": user_metrics.current_state.weekly_activity_days,
            "target_weekly_days": user_metrics.target_state.target_weekly_days,
            "sleep_hours": user_metrics.current_state.sleep_hours,
            "water_liters": user_metrics.current_state.water_liters,
        },
        "deterministic_wellness": {
            "bmi": wellness_result.bmi,
            "bmi_classification": wellness_result.bmi_classification,
            "wellness_score": wellness_result.wellness_score,
            "bmi_score": wellness_result.bmi_score,
            "activity_score": wellness_result.activity_score,
            "progress_score": wellness_result.progress_score,
            "habits_score": wellness_result.habits_score,
        },
        "compacted_trends": compacted_trends,
        "preferences": user_metrics.preferences,
        "scrubbed_medical_conditions": layer1_result.scrubbed_medical_conditions,
        "forbidden_terms": layer1_result.forbidden_terms,
        "dietary_constraints": layer1_result.dietary_constraints,
        "exercise_constraints": layer1_result.exercise_constraints,
    }
    return json.dumps(compact_context, separators=(",", ":"), sort_keys=True)


def construct_llama_prompt_payload(
    user_metrics: AnonymizedUserMetrics,
    wellness_result: WellnessCalculationResult,
    compacted_trends: Dict[str, Any],
    layer1_result: Layer1ClinicalExtractionResult,
    requested_strategy: Optional[str] = None,
) -> ConstructedLlamaPrompt:
    """Build the complete Layer 2 LLaMA chat messages and formatted prompt string."""
    strategy = select_prompt_strategy(
        user_metrics=user_metrics,
        compacted_trends=compacted_trends,
        requested_strategy=requested_strategy,
    )
    metrics_json = format_compacted_metrics_summary(
        user_metrics=user_metrics,
        wellness_result=wellness_result,
        compacted_trends=compacted_trends,
        layer1_result=layer1_result,
    )

    primary_goal = user_metrics.target_state.primary_goal or "general_wellness"
    grounding_instruction = (
        f"IMPORTANT: You MUST cite the user's exact BMI ({wellness_result.bmi:.1f}, "
        f"{wellness_result.bmi_classification}), exact Wellness Score ({wellness_result.wellness_score:.1f}/100), "
        f"and exact primary_goal ('{primary_goal}'). Never output '<...>' placeholders or invented numbers."
    )

    if strategy == "few_shot":
        user_prompt = (
            f"{build_few_shot_examples_block()}\n\n"
            f"{grounding_instruction}\n"
            f"Analyze the following anonymized health context and return ONLY valid JSON:\n{metrics_json}"
        )
    else:
        user_prompt = (
            f"{grounding_instruction}\n"
            f"Analyze the following anonymized health context and return ONLY valid JSON:\n{metrics_json}"
        )

    formatted_llama_prompt = (
        "<|begin_of_text|><|start_header_id|>system<|end_header_id|>\n"
        f"{_SYSTEM_PROMPT}<|eot_id|><|start_header_id|>user<|end_header_id|>\n"
        f"{user_prompt}<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n"
    )
    messages = [
        {"role": "system", "content": _SYSTEM_PROMPT},
        {"role": "user", "content": user_prompt},
    ]

    return ConstructedLlamaPrompt(
        prompt_strategy=strategy,
        system_prompt=_SYSTEM_PROMPT,
        user_prompt=user_prompt,
        formatted_llama_prompt=formatted_llama_prompt,
        messages=messages,
    )
