"""Integration tests for NutriSync's FastAPI endpoints in services/py-ai/main.py."""

import json

from fastapi.testclient import TestClient

from main import app, get_python_service_health_status

client = TestClient(app)


def _build_anonymized_analyze_request_payload(
    weight_kg: float = 82.0,
    height_cm: float = 178.0,
    medical_conditions: str = "Email leak@example.com; mild knee injury",
) -> dict:
    return {
        "subject_id": "sha256-anon-subject-001",
        "regenerate_insights": True,
        "user_metrics": {
            "current_state": {
                "weight": weight_kg,
                "height": height_cm,
                "age": 32,
                "gender": "male",
                "activity_level": "moderately_active",
                "weekly_activity_days": 3,
                "sleep_hours": 7.5,
                "water_liters": 2.4,
                "bmi": 25.88,
                "bmi_classification": "Overweight",
            },
            "target_state": {
                "weight": 76.0,
                "activity_level": "active",
                "primary_goal": "fat_loss",
                "target_weekly_days": 4,
                "baseline_weight": 88.0,
            },
            "preferences": ["walking", "cycling", "environment:outdoor"],
            "restrictions": ["peanut_allergy"],
            "medical_conditions": medical_conditions,
            "fitness_assessment": {
                "fitness_level": "intermediate",
                "session_duration": "30_45_min",
                "endurance_minutes": 30,
                "pushups_count": 20,
                "squats_count": 25,
            },
            "weight_history": [
                {"weight_kg": 84.0, "recorded_at": "2026-11-01T08:00:00Z"},
                {"weight_kg": 83.0, "recorded_at": "2026-11-08T08:00:00Z"},
                {"weight_kg": 82.0, "recorded_at": "2026-11-15T08:00:00Z"},
            ],
            "activity_history": [
                {
                    "exercise_type": "walking",
                    "duration_minutes": 35,
                    "intensity": "moderate",
                    "recorded_at": "2026-11-12T17:00:00Z",
                },
                {
                    "exercise_type": "cycling",
                    "duration_minutes": 40,
                    "intensity": "moderate",
                    "recorded_at": "2026-11-14T17:00:00Z",
                },
            ],
        },
    }


def test_get_python_service_health_status_function_returns_expected_dictionary():
    response_dictionary = get_python_service_health_status()
    assert response_dictionary == {"status": "ok", "service": "pi-ai"}


def test_health_endpoint_returns_http_status_code_two_hundred():
    http_response = client.get("/health")
    assert http_response.status_code == 200


def test_health_endpoint_returns_expected_json_payload():
    http_response = client.get("/health")
    assert http_response.json() == {"status": "ok", "service": "pi-ai"}


def test_analyze_endpoint_returns_camel_case_wellness_and_insights():
    payload = _build_anonymized_analyze_request_payload()
    http_response = client.post("/api/v1/analyze", json=payload)

    assert http_response.status_code == 200
    body = http_response.json()

    assert "wellness" in body
    assert "insights" in body

    wellness = body["wellness"]
    assert wellness["bmiClassification"] == "overweight"
    assert 0.0 <= wellness["wellnessScore"] <= 100.0
    assert "bmiScore" in wellness
    assert "activityScore" in wellness
    assert "progressScore" in wellness
    assert "habitsScore" in wellness
    assert len(wellness["milestones"]) >= 1
    assert "progressPct" in wellness["milestones"][0]

    insights = body["insights"]
    assert insights["promptStrategy"] in ("few_shot", "zero_shot")
    assert "healthStatusAnalysis" in insights
    assert "progressEvaluation" in insights
    assert "weeklySummary" in insights
    assert "monthlySummary" in insights
    assert len(insights["recommendations"]) >= 1
    assert "expectedImpact" in insights["recommendations"][0]


def test_analyze_endpoint_enforces_pii_scrubbing_and_guardrails_with_mocked_llm(monkeypatch):
    captured_prompts: list[str] = []

    def _fake_invoke_llama(constructed_prompt, configuration=None, http_client=None):
        captured_prompts.append(constructed_prompt.formatted_llama_prompt)
        return json.dumps(
            {
                "health_status_analysis": "Avoid peanut butter snacks.",
                "progress_evaluation": "Good momentum.",
                "recommendations": [
                    {
                        "title": "Peanut Snack",
                        "category": "nutrition",
                        "priority": "high",
                        "action": "Eat peanut butter toast.",
                        "expected_impact": "Energy.",
                    },
                    {
                        "title": "Low-Impact Cycling",
                        "category": "activity",
                        "priority": "medium",
                        "action": "Cycle 30 minutes at moderate effort.",
                        "expected_impact": "Supports fat loss without knee strain.",
                    },
                ],
                "weekly_summary": "Consistent week.",
                "monthly_summary": "Consistent month.",
            }
        )

    monkeypatch.setattr("pipeline.layer3_llama_validator.invoke_llama_completion_endpoint", _fake_invoke_llama)

    payload = _build_anonymized_analyze_request_payload()
    http_response = client.post("/api/v1/analyze", json=payload)

    assert http_response.status_code == 200
    assert len(captured_prompts) == 1
    assert "leak@example.com" not in captured_prompts[0]
    assert "2026-11-01T08:00:00Z" not in captured_prompts[0]

    insights = http_response.json()["insights"]
    titles = [rec["title"] for rec in insights["recommendations"]]
    assert "Peanut Snack" not in titles
    assert "Low-Impact Cycling" in titles
    assert "peanut" not in insights["healthStatusAnalysis"].lower()


def test_analyze_endpoint_rejects_non_positive_biometrics_with_http_400():
    payload = _build_anonymized_analyze_request_payload(weight_kg=0.0, height_cm=178.0)
    http_response = client.post("/api/v1/analyze", json=payload)

    assert http_response.status_code == 400
    assert "error" in http_response.json()


def test_analyze_endpoint_rejects_malformed_payload_with_http_422():
    http_response = client.post("/api/v1/analyze", json={"subject_id": "abc"})
    assert http_response.status_code == 422
