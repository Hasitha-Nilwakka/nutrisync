"""Layer 3: LLaMA inference client, guardrail validator, and deterministic clinical fallback.

Invokes an optional configured LLM endpoint over HTTP, validates JSON responses against
PyAIInsightsPayload, strips any recommendation or narrative containing Layer 1 forbidden
terms, and falls back deterministically whenever the LLM is unconfigured or unavailable.
"""

import json
import os
import re
from typing import Any, Dict, List, Optional

import httpx
from pydantic import BaseModel, ValidationError

from pipeline.layer1_scispacy_extractor import Layer1ClinicalExtractionResult
from pipeline.layer2_prompt_constructor import (
    ConstructedLlamaPrompt,
    construct_llama_prompt_payload,
)
from schemas.analysis_schemas import (
    AnonymizedUserMetrics,
    PrioritizedAIRecommendation,
    PyAIInsightsPayload,
    WellnessCalculationResult,
)

_CODE_FENCE_PATTERN = re.compile(r"^```(?:json)?\s*|\s*```$", re.IGNORECASE | re.MULTILINE)


class LLMClientConfiguration(BaseModel):
    """Runtime configuration for Layer 3 LLM HTTP requests."""

    api_url: str = ""
    model: str = "llama3.1"
    api_key: str = ""
    api_format: str = "auto"
    timeout_seconds: float = 10.0

    @classmethod
    def from_environment(cls) -> "LLMClientConfiguration":
        """Load LLM client settings from environment variables with safe offline defaults."""
        raw_timeout = os.environ.get("LLM_TIMEOUT_SECONDS", "10.0").strip()
        try:
            timeout_seconds = max(0.5, float(raw_timeout))
        except ValueError:
            timeout_seconds = 10.0

        return cls(
            api_url=os.environ.get("LLM_API_URL", "").strip(),
            model=os.environ.get("LLM_MODEL", "llama3.1").strip() or "llama3.1",
            api_key=os.environ.get("LLM_API_KEY", "").strip(),
            api_format=os.environ.get("LLM_API_FORMAT", "auto").strip().lower() or "auto",
            timeout_seconds=timeout_seconds,
        )


def _extract_text_from_llm_response_json(response_json: Dict[str, Any]) -> Optional[str]:
    """Extract generated text from OpenAI-compatible or Ollama JSON payloads."""
    choices = response_json.get("choices")
    if isinstance(choices, list) and choices:
        first_choice = choices[0]
        if isinstance(first_choice, dict):
            message = first_choice.get("message")
            if isinstance(message, dict) and isinstance(message.get("content"), str):
                return message["content"]
            if isinstance(first_choice.get("text"), str):
                return first_choice["text"]

    ollama_message = response_json.get("message")
    if isinstance(ollama_message, dict) and isinstance(ollama_message.get("content"), str):
        return ollama_message["content"]

    if isinstance(response_json.get("response"), str):
        return response_json["response"]

    return None


def invoke_llama_completion_endpoint(
    constructed_prompt: ConstructedLlamaPrompt,
    configuration: Optional[LLMClientConfiguration] = None,
    http_client: Optional[httpx.Client] = None,
) -> Optional[str]:
    """Send the Layer 2 prompt to the configured LLM endpoint, returning None on any failure."""
    config = configuration if configuration is not None else LLMClientConfiguration.from_environment()
    if not config.api_url:
        return None

    headers: Dict[str, str] = {"Content-Type": "application/json"}
    if config.api_key:
        headers["Authorization"] = f"Bearer {config.api_key}"

    use_ollama_generate = config.api_format == "ollama_generate" or config.api_url.rstrip("/").endswith("/api/generate")
    if use_ollama_generate:
        request_body: Dict[str, Any] = {
            "model": config.model,
            "prompt": constructed_prompt.formatted_llama_prompt,
            "stream": False,
        }
    elif config.api_format == "ollama_chat" or config.api_url.rstrip("/").endswith("/api/chat"):
        request_body = {
            "model": config.model,
            "messages": constructed_prompt.messages,
            "stream": False,
        }
    else:
        request_body = {
            "model": config.model,
            "messages": constructed_prompt.messages,
            "temperature": 0.2,
            "max_tokens": 768,
        }

    try:
        if http_client is not None:
            response = http_client.post(
                config.api_url,
                json=request_body,
                headers=headers,
                timeout=config.timeout_seconds,
            )
        else:
            with httpx.Client(timeout=config.timeout_seconds) as client:
                response = client.post(config.api_url, json=request_body, headers=headers)
    except httpx.HTTPError:
        return None

    if response.status_code != 200:
        return None

    try:
        payload = response.json()
    except ValueError:
        return None

    if not isinstance(payload, dict):
        return None
    return _extract_text_from_llm_response_json(payload)


def parse_llm_insights_json(
    raw_response_text: str,
    prompt_strategy: str,
) -> Optional[PyAIInsightsPayload]:
    """Parse and validate raw LLM text into PyAIInsightsPayload, returning None if malformed."""
    if not raw_response_text or not raw_response_text.strip():
        return None

    cleaned = _CODE_FENCE_PATTERN.sub("", raw_response_text).strip()
    start_index = cleaned.find("{")
    end_index = cleaned.rfind("}")
    if start_index == -1 or end_index == -1 or end_index < start_index:
        return None

    json_slice = cleaned[start_index : end_index + 1]
    try:
        parsed_dict = json.loads(json_slice)
    except ValueError:
        return None

    if not isinstance(parsed_dict, dict):
        return None

    parsed_dict.setdefault("prompt_strategy", prompt_strategy)
    try:
        return PyAIInsightsPayload.model_validate(parsed_dict)
    except ValidationError:
        return None


def find_matching_forbidden_term(text: str, forbidden_terms: List[str]) -> Optional[str]:
    """Return the first Layer 1 forbidden term matched in text, or None if clean."""
    if not text or not forbidden_terms:
        return None

    for term in forbidden_terms:
        stripped = term.strip()
        if not stripped:
            continue
        pattern = re.compile(rf"(?i)\b{re.escape(stripped)}s?\b")
        if pattern.search(text):
            return stripped
    return None


def _sanitize_narrative_text(text: str, forbidden_terms: List[str]) -> str:
    """Replace any forbidden term occurrences in narrative text with a safe placeholder."""
    sanitized = text
    for term in forbidden_terms:
        stripped = term.strip()
        if not stripped:
            continue
        pattern = re.compile(rf"(?i)\b{re.escape(stripped)}s?\b")
        sanitized = pattern.sub("[restricted item omitted]", sanitized)
    return sanitized


def _build_safe_goal_aligned_recommendation(
    primary_goal: str,
    layer1_result: Layer1ClinicalExtractionResult,
) -> PrioritizedAIRecommendation:
    """Create a deterministic, goal-aligned, restriction-compliant recommendation."""
    goal_key = (primary_goal or "general_wellness").strip().lower()
    has_joint_or_exercise_limit = bool(layer1_result.exercise_constraints)

    if "loss" in goal_key or "cut" in goal_key:
        action = (
            "Pair a modest daily calorie deficit with 30-minute low-impact walks 3 times weekly."
            if has_joint_or_exercise_limit
            else "Maintain a sustainable 300-400 kcal daily deficit with 3 structured cardio/resistance sessions."
        )
        return PrioritizedAIRecommendation(
            title="Sustainable Calorie & Low-Impact Activity Plan",
            category="goal_alignment",
            priority="high",
            action=action,
            expected_impact="Supports steady fat-loss progress while respecting all dietary and joint constraints.",
        )

    if "muscle" in goal_key or "gain" in goal_key or "strength" in goal_key:
        action = (
            "Focus on controlled, joint-friendly resistance training and spread protein intake across meals."
            if has_joint_or_exercise_limit
            else "Complete 3 progressive resistance sessions weekly with balanced post-workout protein and hydration."
        )
        return PrioritizedAIRecommendation(
            title="Progressive Resistance & Recovery Nutrition",
            category="goal_alignment",
            priority="high",
            action=action,
            expected_impact="Supports lean strength gains aligned with your target while honoring all restrictions.",
        )

    return PrioritizedAIRecommendation(
        title="Consistent Daily Movement & Hydration Routine",
        category="habits",
        priority="medium",
        action="Aim for 7-8 hours of sleep, 2.0+ liters of water daily, and consistent moderate movement.",
        expected_impact="Stabilizes energy, recovery, and composite wellness score.",
    )


def _contains_hallucinated_or_copied_metrics(
    text: str,
    wellness_result: Optional[WellnessCalculationResult],
) -> bool:
    """Return True if text copies few-shot placeholders/numbers or contradicts deterministic BMI/Wellness Score."""
    if not text or not text.strip():
        return True
    if "<exact_" in text or "24.2" in text and (wellness_result is None or round(wellness_result.bmi, 1) != 24.2):
        return True
    if wellness_result is None:
        return False

    expected_bmi_1d = f"{wellness_result.bmi:.1f}"
    expected_bmi_2d = f"{wellness_result.bmi:.2f}"
    expected_score_1d = f"{wellness_result.wellness_score:.1f}"
    expected_score_2d = f"{wellness_result.wellness_score:.2f}"
    expected_score_int = str(int(round(wellness_result.wellness_score)))

    bmi_matches = re.findall(r"(?i)bmi(?:\s+is|\s+of|:)?\s*(\d+(?:\.\d+)?)", text)
    for matched_bmi in bmi_matches:
        try:
            if abs(float(matched_bmi) - wellness_result.bmi) > 0.5:
                return True
        except ValueError:
            return True

    score_matches = re.findall(r"(?i)wellness\s+score(?:\s+is|\s+of|:)?\s*(\d+(?:\.\d+)?)", text)
    for matched_score in score_matches:
        try:
            if abs(float(matched_score) - wellness_result.wellness_score) > 1.0:
                return True
        except ValueError:
            return True

    has_expected_bmi = expected_bmi_1d in text or expected_bmi_2d in text
    has_expected_score = (
        expected_score_1d in text or expected_score_2d in text or f"{expected_score_int}/100" in text
    )
    if bmi_matches and not has_expected_bmi:
        return True
    if score_matches and not has_expected_score:
        return True
    return False


def filter_and_sanitize_insights_against_guardrails(
    insights: PyAIInsightsPayload,
    layer1_result: Layer1ClinicalExtractionResult,
    primary_goal: str,
    wellness_result: Optional[WellnessCalculationResult] = None,
    compacted_trends: Optional[Dict[str, Any]] = None,
) -> PyAIInsightsPayload:
    """Enforce Layer 1 forbidden-term filtering, numeric BMI/score grounding, and goal alignment on LLM insights."""
    forbidden_terms = layer1_result.forbidden_terms
    safe_recommendations: List[PrioritizedAIRecommendation] = []

    for rec in insights.recommendations:
        combined_rec_text = f"{rec.title} {rec.category} {rec.action} {rec.expected_impact}"
        if "<exact_" in combined_rec_text:
            continue
        if find_matching_forbidden_term(combined_rec_text, forbidden_terms) is None:
            safe_recommendations.append(rec)

    if not safe_recommendations:
        safe_recommendations.append(
            _build_safe_goal_aligned_recommendation(primary_goal=primary_goal, layer1_result=layer1_result)
        )

    health_status = _sanitize_narrative_text(insights.health_status_analysis, forbidden_terms)
    progress_eval = _sanitize_narrative_text(insights.progress_evaluation, forbidden_terms)
    weekly_summary = _sanitize_narrative_text(insights.weekly_summary, forbidden_terms)
    monthly_summary = _sanitize_narrative_text(insights.monthly_summary, forbidden_terms)

    normalized_goal = (primary_goal or "general_wellness").strip()
    readable_goal = normalized_goal.replace("_", " ")

    if wellness_result is not None and _contains_hallucinated_or_copied_metrics(health_status, wellness_result):
        health_status = (
            f"Current BMI is {wellness_result.bmi:.1f} ({wellness_result.bmi_classification}) with an "
            f"overall Wellness Score of {wellness_result.wellness_score:.1f}/100 (BMI Score "
            f"{wellness_result.bmi_score:.0f}, Activity {wellness_result.activity_score:.0f}, "
            f"Progress {wellness_result.progress_score:.0f}, Habits {wellness_result.habits_score:.0f})."
        )

    if wellness_result is not None and (
        "<exact_" in progress_eval
        or ("fat_loss" in progress_eval and "fat_loss" not in normalized_goal.lower())
        or _contains_hallucinated_or_copied_metrics(progress_eval, wellness_result)
    ):
        weight_trends = (compacted_trends or {}).get("weight_trends", {})
        activity_trends = (compacted_trends or {}).get("activity_trends", {})
        delta_7d = weight_trends.get("delta_7d_kg", 0.0) if isinstance(weight_trends, dict) else 0.0
        sessions_7d = activity_trends.get("sessions_7d", 0) if isinstance(activity_trends, dict) else 0
        minutes_7d = activity_trends.get("minutes_7d", 0) if isinstance(activity_trends, dict) else 0
        progress_eval = (
            f"7-day weight delta is {delta_7d:+.2f} kg with {sessions_7d} logged activity sessions "
            f"({minutes_7d} total minutes) aligned toward your {readable_goal} goal."
        )

    if "<exact_" in monthly_summary:
        monthly_summary = monthly_summary.replace("<exact_primary_goal>", readable_goal)

    return PyAIInsightsPayload(
        prompt_strategy=insights.prompt_strategy,
        health_status_analysis=health_status,
        progress_evaluation=progress_eval,
        recommendations=safe_recommendations,
        weekly_summary=weekly_summary,
        monthly_summary=monthly_summary,
    )


def build_deterministic_clinical_fallback_insights(
    user_metrics: AnonymizedUserMetrics,
    wellness_result: WellnessCalculationResult,
    compacted_trends: Dict[str, Any],
    layer1_result: Layer1ClinicalExtractionResult,
    prompt_strategy: str,
) -> PyAIInsightsPayload:
    """Generate deterministic, guardrail-compliant insights when the LLM is unavailable."""
    primary_goal = user_metrics.target_state.primary_goal or "general_wellness"
    weight_trends = compacted_trends.get("weight_trends", {})
    activity_trends = compacted_trends.get("activity_trends", {})

    delta_7d = weight_trends.get("delta_7d_kg", 0.0) if isinstance(weight_trends, dict) else 0.0
    sessions_7d = activity_trends.get("sessions_7d", 0) if isinstance(activity_trends, dict) else 0
    minutes_7d = activity_trends.get("minutes_7d", 0) if isinstance(activity_trends, dict) else 0

    health_status = (
        f"Current BMI is {wellness_result.bmi:.2f} ({wellness_result.bmi_classification}) with an "
        f"overall Wellness Score of {wellness_result.wellness_score:.2f}/100."
    )
    progress_eval = (
        f"7-day weight delta is {delta_7d:+.2f} kg with {sessions_7d} logged activity sessions "
        f"({minutes_7d} total minutes) aligned toward {primary_goal}."
    )

    recommendations: List[PrioritizedAIRecommendation] = [
        _build_safe_goal_aligned_recommendation(primary_goal=primary_goal, layer1_result=layer1_result),
    ]
    if wellness_result.habits_score < 80.0:
        recommendations.append(
            PrioritizedAIRecommendation(
                title="Optimize Sleep & Daily Hydration",
                category="habits",
                priority="medium",
                action=(
                    f"Target 7.5 hours of sleep (currently {user_metrics.current_state.sleep_hours:.1f}h) "
                    f"and at least 2.5L water (currently {user_metrics.current_state.water_liters:.1f}L)."
                ),
                expected_impact="Boosts recovery and habits component score toward 80+.",
            )
        )

    fallback = PyAIInsightsPayload(
        prompt_strategy=prompt_strategy,
        health_status_analysis=health_status,
        progress_evaluation=progress_eval,
        recommendations=recommendations,
        weekly_summary=(
            f"Logged {sessions_7d} sessions ({minutes_7d} min) over the past 7 days with "
            f"activity score {wellness_result.activity_score:.1f}/100."
        ),
        monthly_summary=(
            f"30-day progress score stands at {wellness_result.progress_score:.1f}/100; "
            f"continue steady adherence to {primary_goal}."
        ),
    )
    return filter_and_sanitize_insights_against_guardrails(
        insights=fallback,
        layer1_result=layer1_result,
        primary_goal=primary_goal,
    )


def generate_validated_ai_insights(
    user_metrics: AnonymizedUserMetrics,
    wellness_result: WellnessCalculationResult,
    compacted_trends: Dict[str, Any],
    layer1_result: Layer1ClinicalExtractionResult,
    requested_strategy: Optional[str] = None,
    configuration: Optional[LLMClientConfiguration] = None,
    http_client: Optional[httpx.Client] = None,
) -> PyAIInsightsPayload:
    """Run Layer 2 prompt construction and Layer 3 LLM invocation + guardrail validation."""
    constructed_prompt = construct_llama_prompt_payload(
        user_metrics=user_metrics,
        wellness_result=wellness_result,
        compacted_trends=compacted_trends,
        layer1_result=layer1_result,
        requested_strategy=requested_strategy,
    )
    raw_llm_output = invoke_llama_completion_endpoint(
        constructed_prompt=constructed_prompt,
        configuration=configuration,
        http_client=http_client,
    )
    if raw_llm_output is not None:
        parsed_insights = parse_llm_insights_json(
            raw_response_text=raw_llm_output,
            prompt_strategy=constructed_prompt.prompt_strategy,
        )
        if parsed_insights is not None:
            return filter_and_sanitize_insights_against_guardrails(
                insights=parsed_insights,
                layer1_result=layer1_result,
                primary_goal=user_metrics.target_state.primary_goal,
                wellness_result=wellness_result,
                compacted_trends=compacted_trends,
            )

    return build_deterministic_clinical_fallback_insights(
        user_metrics=user_metrics,
        wellness_result=wellness_result,
        compacted_trends=compacted_trends,
        layer1_result=layer1_result,
        prompt_strategy=constructed_prompt.prompt_strategy,
    )
