"""Unit tests for the deterministic Wellness Score, milestone, and Pandas/NumPy compaction engine."""

from datetime import datetime, timedelta, timezone
import math

import pytest

from analytics.wellness_engine import (
    calculate_bmi_and_classification,
    calculate_bmi_component_score,
    calculate_composite_wellness_score,
    calculate_deterministic_wellness_and_milestones,
    calculate_habits_component_score,
    calculate_progress_component_score,
    clamp_score_to_hundred_scale,
    compact_historical_metrics_for_ai_context,
)
from schemas.analysis_schemas import (
    AnonymizedActivityHistoryPoint,
    AnonymizedCurrentState,
    AnonymizedFitnessAssessment,
    AnonymizedTargetState,
    AnonymizedUserMetrics,
    AnonymizedWeightHistoryPoint,
)


def _build_go_parity_user_metrics(
    weight_history: list[AnonymizedWeightHistoryPoint] | None = None,
    activity_history: list[AnonymizedActivityHistoryPoint] | None = None,
) -> AnonymizedUserMetrics:
    """Builds the exact fixture used in Go's TestCalculateDeterministicWellnessAndMilestonesFormulaAndMilestones."""
    return AnonymizedUserMetrics(
        current_state=AnonymizedCurrentState(
            weight=82.0,
            height=180.0,
            age=29,
            gender="male",
            activity_level="moderately_active",
            weekly_activity_days=2,
            sleep_hours=7.5,
            water_liters=2.5,
            bmi=25.31,
            bmi_classification="overweight",
        ),
        target_state=AnonymizedTargetState(
            weight=75.0,
            activity_level="active",
            primary_goal="weight_loss",
            target_weekly_days=4,
            baseline_weight=90.0,
        ),
        preferences=["mediterranean"],
        restrictions=[],
        medical_conditions="",
        fitness_assessment=AnonymizedFitnessAssessment(
            fitness_level="intermediate",
            session_duration="30_60_min",
            endurance_minutes=30,
            pushups_count=20,
            squats_count=30,
        ),
        weight_history=weight_history or [],
        activity_history=activity_history or [],
    )


def test_bmi_calculation_and_classifications() -> None:
    """Verifies BMI formula weight / (height_m ** 2) and all four clinical categories."""
    assert calculate_bmi_and_classification(50.0, 175.0) == (16.33, "underweight")
    assert calculate_bmi_and_classification(70.0, 175.0) == (22.86, "normal weight")
    assert calculate_bmi_and_classification(82.0, 180.0) == (25.31, "overweight")
    assert calculate_bmi_and_classification(105.0, 175.0) == (34.29, "obese")

    with pytest.raises(ValueError):
        calculate_bmi_and_classification(0.0, 175.0)
    with pytest.raises(ValueError):
        calculate_bmi_and_classification(70.0, -10.0)


def test_score_clamping_and_composite_formula_boundaries() -> None:
    """Verifies [0.0, 100.0] clamping across extreme inputs and the 0.3/0.3/0.2/0.2 weighted formula."""
    assert clamp_score_to_hundred_scale(-25.4) == 0.0
    assert clamp_score_to_hundred_scale(142.89) == 100.0
    assert clamp_score_to_hundred_scale(float("nan")) == 0.0
    assert clamp_score_to_hundred_scale(float("inf")) == 0.0

    assert calculate_bmi_component_score(5.0) == 0.0
    assert calculate_bmi_component_score(60.0) == 0.0
    assert calculate_bmi_component_score(21.7) == 100.0

    assert calculate_habits_component_score(8.0, 3.0) == 100.0
    assert 0.0 <= calculate_habits_component_score(2.0, 0.1) <= 100.0

    composite = calculate_composite_wellness_score(
        bmi_score=150.0,
        activity_score=-20.0,
        progress_score=80.0,
        habits_score=90.0,
    )
    # Clamped components: (100*0.3) + (0*0.3) + (80*0.2) + (90*0.2) = 30 + 0 + 16 + 18 = 64.0
    assert composite == 64.0


def test_progress_component_maintenance_and_goal_delta() -> None:
    """Verifies maintenance mode (<0.25kg goal delta) and standard weight trajectory progress."""
    maint_score, maint_frac = calculate_progress_component_score(70.4, 70.0, 70.0)
    assert maint_score == 95.0
    assert maint_frac == 1.0

    loss_score, loss_frac = calculate_progress_component_score(82.0, 90.0, 75.0)
    # Achieved 8.0 of 15.0 -> 8/15 = 0.533333... -> 40 + 32 = 72.0
    assert math.isclose(loss_frac, 8.0 / 15.0, rel_tol=1e-6)
    assert loss_score == 72.0


def test_deterministic_wellness_parity_with_go_fixture_and_camel_case_serialization() -> None:
    """Verifies exact formula parity with Go's CalculateDeterministicWellnessAndMilestones and camelCase output."""
    initial_metrics = _build_go_parity_user_metrics()
    initial_result = calculate_deterministic_wellness_and_milestones(initial_metrics)

    expected_initial_composite = round(
        (initial_result.bmi_score * 0.3)
        + (initial_result.activity_score * 0.3)
        + (initial_result.progress_score * 0.2)
        + (initial_result.habits_score * 0.2),
        2,
    )
    assert initial_result.wellness_score == expected_initial_composite
    assert initial_result.bmi == 25.31
    assert initial_result.bmi_classification == "overweight"
    assert len(initial_result.milestones) == 3
    assert [m.id for m in initial_result.milestones] == [
        "weight-5pct-step",
        "weekly-activity-target",
        "daily-habits-consistency",
    ]

    # Verify camelCase serialization matches Go's json tags
    dumped = initial_result.model_dump(by_alias=True)
    assert "bmiClassification" in dumped
    assert "wellnessScore" in dumped
    assert "progressPct" in dumped["milestones"][0]

    now = datetime.now(timezone.utc)
    activities = [
        AnonymizedActivityHistoryPoint(
            exercise_type="running",
            duration_minutes=45,
            intensity="high",
            recorded_at=(now - timedelta(hours=1)).isoformat(),
        ),
        AnonymizedActivityHistoryPoint(
            exercise_type="cycling",
            duration_minutes=45,
            intensity="moderate",
            recorded_at=(now - timedelta(hours=24)).isoformat(),
        ),
        AnonymizedActivityHistoryPoint(
            exercise_type="strength",
            duration_minutes=50,
            intensity="high",
            recorded_at=(now - timedelta(hours=48)).isoformat(),
        ),
        AnonymizedActivityHistoryPoint(
            exercise_type="swimming",
            duration_minutes=40,
            intensity="moderate",
            recorded_at=(now - timedelta(hours=72)).isoformat(),
        ),
    ]
    weights = [
        AnonymizedWeightHistoryPoint(weight_kg=90.0, recorded_at=(now - timedelta(days=30)).isoformat()),
        AnonymizedWeightHistoryPoint(weight_kg=76.0, recorded_at=now.isoformat()),
    ]

    updated_metrics = _build_go_parity_user_metrics(weight_history=weights, activity_history=activities)
    updated_result = calculate_deterministic_wellness_and_milestones(updated_metrics)

    assert updated_result.wellness_score > initial_result.wellness_score
    assert updated_result.milestones[1].achieved is True
    assert updated_result.milestones[2].achieved is True


def test_compact_historical_metrics_empty_and_single_point() -> None:
    """Verifies Pandas/NumPy compaction handles empty and 1-point arrays without NaN or division errors."""
    empty_summary = compact_historical_metrics_for_ai_context([], [])
    assert empty_summary["weight_trends"]["total_entries"] == 0
    assert empty_summary["weight_trends"]["latest_kg"] is None
    assert empty_summary["weight_trends"]["slope_7d_kg_per_day"] == 0.0
    assert empty_summary["activity_trends"]["sessions_7d"] == 0
    assert empty_summary["activity_trends"]["dominant_exercise_type"] == "none"

    now_iso = datetime.now(timezone.utc).isoformat()
    single_summary = compact_historical_metrics_for_ai_context(
        weight_history=[AnonymizedWeightHistoryPoint(weight_kg=80.0, recorded_at=now_iso)],
        activity_history=[
            AnonymizedActivityHistoryPoint(
                exercise_type="walking",
                duration_minutes=30,
                intensity="low",
                recorded_at=now_iso,
            )
        ],
    )
    assert single_summary["weight_trends"]["total_entries"] == 1
    assert single_summary["weight_trends"]["avg_7d_kg"] == 80.0
    assert single_summary["weight_trends"]["delta_7d_kg"] == 0.0
    assert single_summary["weight_trends"]["slope_7d_kg_per_day"] == 0.0
    assert single_summary["activity_trends"]["sessions_7d"] == 1
    assert single_summary["activity_trends"]["minutes_7d"] == 30
    assert single_summary["activity_trends"]["dominant_exercise_type"] == "walking"


def test_compact_historical_metrics_multi_week_trends_and_sorting() -> None:
    """Verifies 7d/30d moving averages, net deltas, and negative linear slope on unsorted multi-week data."""
    base_time = datetime(2026, 10, 1, 8, 0, tzinfo=timezone.utc)
    # Unsorted entries spanning 25 days with steady -0.2 kg/day loss
    unsorted_weights = [
        AnonymizedWeightHistoryPoint(weight_kg=76.0, recorded_at=(base_time + timedelta(days=20)).isoformat()),
        AnonymizedWeightHistoryPoint(weight_kg=80.0, recorded_at=base_time.isoformat()),
        AnonymizedWeightHistoryPoint(weight_kg=77.4, recorded_at=(base_time + timedelta(days=13)).isoformat()),
        AnonymizedWeightHistoryPoint(weight_kg=75.0, recorded_at=(base_time + timedelta(days=25)).isoformat()),
    ]
    activities = [
        AnonymizedActivityHistoryPoint(
            exercise_type="running",
            duration_minutes=40,
            intensity="high",
            recorded_at=(base_time + timedelta(days=2)).isoformat(),
        ),
        AnonymizedActivityHistoryPoint(
            exercise_type="running",
            duration_minutes=30,
            intensity="moderate",
            recorded_at=(base_time + timedelta(days=21)).isoformat(),
        ),
        AnonymizedActivityHistoryPoint(
            exercise_type="cycling",
            duration_minutes=50,
            intensity="high",
            recorded_at=(base_time + timedelta(days=24)).isoformat(),
        ),
    ]

    summary = compact_historical_metrics_for_ai_context(unsorted_weights, activities)
    wt = summary["weight_trends"]
    act = summary["activity_trends"]

    assert wt["total_entries"] == 4
    assert wt["latest_kg"] == 75.0
    assert wt["delta_30d_kg"] == -5.0
    assert wt["delta_7d_kg"] == -1.0
    assert wt["slope_30d_kg_per_day"] == -0.2
    assert not math.isnan(wt["avg_7d_kg"])
    assert not math.isnan(wt["avg_30d_kg"])

    assert act["sessions_7d"] == 2
    assert act["minutes_7d"] == 80
    assert act["sessions_30d"] == 3
    assert act["minutes_30d"] == 120
    assert act["dominant_exercise_type"] == "running"
    assert act["dominant_intensity"] == "high"
