"""Starter pytest template for testing services/py-ai analytics and 3-layer guardrails."""

from schemas.analysis_schemas import (
    AnonymizedAIAnalysisPayload,
    AnonymizedCurrentState,
    AnonymizedFitnessAssessment,
    AnonymizedTargetState,
    AnonymizedUserMetrics,
)


def build_synthetic_analysis_payload() -> AnonymizedAIAnalysisPayload:
    """Builds a deterministic, anonymized synthetic payload for unit testing."""
    return AnonymizedAIAnalysisPayload(
        subject_id="anon-subject-001",
        regenerate_insights=True,
        user_metrics=AnonymizedUserMetrics(
            current_state=AnonymizedCurrentState(
                weight=78.0,
                height=175.0,
                age=32,
                gender="male",
                activity_level="moderate",
                weekly_activity_days=3,
                sleep_hours=7.5,
                water_liters=2.2,
                bmi=25.47,
                bmi_classification="overweight",
            ),
            target_state=AnonymizedTargetState(
                weight=72.0,
                activity_level="active",
                primary_goal="weight_loss",
                target_weekly_days=4,
                baseline_weight=80.0,
            ),
            preferences=["mediterranean", "high_protein"],
            restrictions=["nut_allergy", "dairy_free"],
            medical_conditions="Contact john.doe@example.com or 555-123-4567; mild hypertension.",
            fitness_assessment=AnonymizedFitnessAssessment(
                fitness_level="intermediate",
                session_duration="30_60_min",
                endurance_minutes=35,
                pushups_count=25,
                squats_count=40,
            ),
            weight_history=[],
            activity_history=[],
        ),
    )


def test_synthetic_payload_validates_cleanly() -> None:
    """Verifies that the synthetic payload satisfies AnonymizedAIAnalysisPayload schema constraints."""
    payload = build_synthetic_analysis_payload()
    assert payload.subject_id == "anon-subject-001"
    assert "nut_allergy" in payload.user_metrics.restrictions
