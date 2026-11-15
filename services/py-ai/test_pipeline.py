"""Unit tests for NutriSync's 3-Layer AI Guardrail Pipeline (services/py-ai/pipeline/)."""

import json

import httpx

from analytics.wellness_engine import (
    calculate_deterministic_wellness_and_milestones,
    compact_historical_metrics_for_ai_context,
)
from pipeline.layer1_scispacy_extractor import (
    extract_clinical_constraints_and_scrub_pii,
    scrub_residual_pii_from_free_text,
)
from pipeline.layer2_prompt_constructor import construct_llama_prompt_payload
from pipeline.layer3_llama_validator import (
    LLMClientConfiguration,
    generate_validated_ai_insights,
    parse_llm_insights_json,
)
from schemas.analysis_schemas import (
    AnonymizedActivityHistoryPoint,
    AnonymizedCurrentState,
    AnonymizedFitnessAssessment,
    AnonymizedTargetState,
    AnonymizedUserMetrics,
    AnonymizedWeightHistoryPoint,
)


def _build_synthetic_user_metrics(
    medical_conditions: str = "type 2 diabetes and mild knee injury",
    restrictions: list[str] | None = None,
    primary_goal: str = "fat_loss",
) -> AnonymizedUserMetrics:
    return AnonymizedUserMetrics(
        current_state=AnonymizedCurrentState(
            weight=86.0,
            height=176.0,
            age=34,
            gender="male",
            activity_level="moderately_active",
            weekly_activity_days=3,
            sleep_hours=6.8,
            water_liters=2.1,
            bmi=27.76,
            bmi_classification="Overweight",
        ),
        target_state=AnonymizedTargetState(
            weight=78.0,
            activity_level="active",
            primary_goal=primary_goal,
            target_weekly_days=4,
            baseline_weight=90.0,
        ),
        preferences=["walking", "cycling", "environment:outdoor"],
        restrictions=restrictions if restrictions is not None else ["peanut_allergy", "lactose_intolerant"],
        medical_conditions=medical_conditions,
        fitness_assessment=AnonymizedFitnessAssessment(
            fitness_level="intermediate",
            session_duration="30_45_min",
            endurance_minutes=30,
            pushups_count=18,
            squats_count=22,
        ),
        weight_history=[
            AnonymizedWeightHistoryPoint(weight_kg=88.0, recorded_at="2026-11-01T08:00:00Z"),
            AnonymizedWeightHistoryPoint(weight_kg=87.0, recorded_at="2026-11-08T08:00:00Z"),
            AnonymizedWeightHistoryPoint(weight_kg=86.0, recorded_at="2026-11-15T08:00:00Z"),
        ],
        activity_history=[
            AnonymizedActivityHistoryPoint(
                exercise_type="walking",
                duration_minutes=35,
                intensity="moderate",
                recorded_at="2026-11-12T17:00:00Z",
            ),
            AnonymizedActivityHistoryPoint(
                exercise_type="cycling",
                duration_minutes=40,
                intensity="moderate",
                recorded_at="2026-11-14T17:00:00Z",
            ),
        ],
    )


def test_layer1_scrubs_residual_pii_and_extracts_forbidden_terms() -> None:
    leaky_text = (
        "Patient John Doe (john.doe@example.com, +1-555-234-5678, @johndoe, "
        "MRN-998811, https://clinic.example.org/record) has type 2 diabetes and knee injury."
    )
    scrubbed, categories = scrub_residual_pii_from_free_text(leaky_text)

    assert "john.doe@example.com" not in scrubbed
    assert "555-234-5678" not in scrubbed
    assert "@johndoe" not in scrubbed
    assert "MRN-998811" not in scrubbed
    assert "https://clinic.example.org/record" not in scrubbed
    assert "John Doe" not in scrubbed
    assert "[REDACTED_EMAIL]" in scrubbed
    assert "[REDACTED_PHONE]" in scrubbed
    assert "type 2 diabetes" in scrubbed
    assert "knee injury" in scrubbed
    assert {"email", "phone", "url", "identifier", "handle", "name"}.issubset(set(categories))

    metrics = _build_synthetic_user_metrics(medical_conditions=leaky_text)
    layer1 = extract_clinical_constraints_and_scrub_pii(metrics)
    assert "peanut" in layer1.forbidden_terms
    assert "milk" in layer1.forbidden_terms
    assert "deep squats" in layer1.forbidden_terms
    assert "sugary soda" in layer1.forbidden_terms
    assert "type 2 diabetes" in layer1.extracted_conditions


def test_layer2_constructs_few_shot_and_zero_shot_without_raw_history() -> None:
    metrics = _build_synthetic_user_metrics(
        medical_conditions="Email me at leak@example.com; mild knee injury."
    )
    wellness = calculate_deterministic_wellness_and_milestones(metrics)
    compacted = compact_historical_metrics_for_ai_context(metrics.weight_history, metrics.activity_history)
    layer1 = extract_clinical_constraints_and_scrub_pii(metrics)

    few_shot_prompt = construct_llama_prompt_payload(
        user_metrics=metrics,
        wellness_result=wellness,
        compacted_trends=compacted,
        layer1_result=layer1,
        requested_strategy="few_shot",
    )
    assert few_shot_prompt.prompt_strategy == "few_shot"
    assert "Example JSON Response:" in few_shot_prompt.user_prompt
    assert "leak@example.com" not in few_shot_prompt.formatted_llama_prompt
    assert "2026-11-01T08:00:00Z" not in few_shot_prompt.formatted_llama_prompt
    assert "2026-11-12T17:00:00Z" not in few_shot_prompt.formatted_llama_prompt
    assert "slope_7d_kg_per_day" in few_shot_prompt.user_prompt

    zero_shot_prompt = construct_llama_prompt_payload(
        user_metrics=metrics,
        wellness_result=wellness,
        compacted_trends=compacted,
        layer1_result=layer1,
        requested_strategy="zero_shot",
    )
    assert zero_shot_prompt.prompt_strategy == "zero_shot"
    assert "Example JSON Response:" not in zero_shot_prompt.user_prompt


def test_layer3_filters_contraindicated_llm_advice_and_sanitizes_narratives() -> None:
    metrics = _build_synthetic_user_metrics()
    wellness = calculate_deterministic_wellness_and_milestones(metrics)
    compacted = compact_historical_metrics_for_ai_context(metrics.weight_history, metrics.activity_history)
    layer1 = extract_clinical_constraints_and_scrub_pii(metrics)

    unsafe_llm_json = json.dumps(
        {
            "health_status_analysis": "Avoid sugary soda and peanut butter snacks.",
            "progress_evaluation": "Good 7-day trend.",
            "recommendations": [
                {
                    "title": "Peanut Butter Recovery Shake",
                    "category": "nutrition",
                    "priority": "high",
                    "action": "Blend peanut butter with milk after workouts.",
                    "expected_impact": "Adds protein.",
                },
                {
                    "title": "Heavy Deep Squats",
                    "category": "activity",
                    "priority": "high",
                    "action": "Perform 5 sets of deep squats.",
                    "expected_impact": "Leg strength.",
                },
                {
                    "title": "Steady Brisk Walking",
                    "category": "activity",
                    "priority": "medium",
                    "action": "Walk 30 minutes outdoors at a comfortable pace.",
                    "expected_impact": "Supports steady fat loss without joint strain.",
                },
            ],
            "weekly_summary": "Keep up the brisk walking.",
            "monthly_summary": "Steady fat loss trajectory.",
        }
    )

    def _mock_handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            json={"choices": [{"message": {"content": unsafe_llm_json}}]},
        )

    client = httpx.Client(transport=httpx.MockTransport(_mock_handler))
    config = LLMClientConfiguration(api_url="https://mock-llm.local/v1/chat/completions", model="llama3.1")

    insights = generate_validated_ai_insights(
        user_metrics=metrics,
        wellness_result=wellness,
        compacted_trends=compacted,
        layer1_result=layer1,
        configuration=config,
        http_client=client,
    )

    titles = [rec.title for rec in insights.recommendations]
    assert "Peanut Butter Recovery Shake" not in titles
    assert "Heavy Deep Squats" not in titles
    assert "Steady Brisk Walking" in titles
    assert "peanut butter" not in insights.health_status_analysis.lower()
    assert "[restricted item omitted]" in insights.health_status_analysis


def test_layer3_parses_fenced_ollama_json_and_serializes_camel_case() -> None:
    fenced = (
        "```json\n"
        + json.dumps(
            {
                "health_status_analysis": "Healthy baseline.",
                "progress_evaluation": "On track.",
                "recommendations": [
                    {
                        "title": "Daily Hydration",
                        "category": "habits",
                        "priority": "medium",
                        "action": "Drink 2.5L of water daily.",
                        "expected_impact": "Improves recovery.",
                    }
                ],
                "weekly_summary": "Consistent week.",
                "monthly_summary": "Consistent month.",
            }
        )
        + "\n```"
    )
    parsed = parse_llm_insights_json(fenced, prompt_strategy="few_shot")
    assert parsed is not None
    serialized = parsed.model_dump(by_alias=True)
    assert serialized["promptStrategy"] == "few_shot"
    assert serialized["healthStatusAnalysis"] == "Healthy baseline."
    assert serialized["recommendations"][0]["expectedImpact"] == "Improves recovery."


def test_layer3_falls_back_deterministically_when_offline_timeout_429_or_malformed() -> None:
    metrics = _build_synthetic_user_metrics(primary_goal="muscle_gain")
    wellness = calculate_deterministic_wellness_and_milestones(metrics)
    compacted = compact_historical_metrics_for_ai_context(metrics.weight_history, metrics.activity_history)
    layer1 = extract_clinical_constraints_and_scrub_pii(metrics)

    # 1. Unconfigured endpoint
    offline_insights = generate_validated_ai_insights(
        user_metrics=metrics,
        wellness_result=wellness,
        compacted_trends=compacted,
        layer1_result=layer1,
        configuration=LLMClientConfiguration(api_url=""),
    )
    assert len(offline_insights.recommendations) >= 1
    assert f"{wellness.wellness_score:.2f}" in offline_insights.health_status_analysis

    # 2. HTTP 429 rate limit
    rate_limit_client = httpx.Client(
        transport=httpx.MockTransport(lambda req: httpx.Response(429, json={"error": "rate_limited"}))
    )
    config = LLMClientConfiguration(api_url="https://mock-llm.local/v1/chat/completions")
    rate_limited_insights = generate_validated_ai_insights(
        user_metrics=metrics,
        wellness_result=wellness,
        compacted_trends=compacted,
        layer1_result=layer1,
        configuration=config,
        http_client=rate_limit_client,
    )
    assert len(rate_limited_insights.recommendations) >= 1

    # 3. Timeout exception
    def _timeout_handler(request: httpx.Request) -> httpx.Response:
        raise httpx.ReadTimeout("timed out", request=request)

    timeout_client = httpx.Client(transport=httpx.MockTransport(_timeout_handler))
    timeout_insights = generate_validated_ai_insights(
        user_metrics=metrics,
        wellness_result=wellness,
        compacted_trends=compacted,
        layer1_result=layer1,
        configuration=config,
        http_client=timeout_client,
    )
    assert len(timeout_insights.recommendations) >= 1

    # 4. Malformed non-JSON output
    malformed_client = httpx.Client(
        transport=httpx.MockTransport(
            lambda req: httpx.Response(200, json={"choices": [{"message": {"content": "not json at all"}}]})
        )
    )
    malformed_insights = generate_validated_ai_insights(
        user_metrics=metrics,
        wellness_result=wellness,
        compacted_trends=compacted,
        layer1_result=layer1,
        configuration=config,
        http_client=malformed_client,
    )
    assert len(malformed_insights.recommendations) >= 1
