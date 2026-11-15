"""Deterministic Wellness Score, milestone, and Pandas/NumPy time-series compaction engine.

This module maintains strict mathematical parity with Go's
CalculateDeterministicWellnessAndMilestones (services/go-api/health/ai_orchestrator_client.go)
and compacts unbounded historical arrays into 7-day and 30-day statistical summaries
so raw time-series arrays are never injected directly into LLM prompts.
"""

from datetime import datetime, timedelta, timezone
import math
from typing import Optional

import numpy as np
import pandas as pd

from schemas.analysis_schemas import (
    AnonymizedActivityHistoryPoint,
    AnonymizedUserMetrics,
    AnonymizedWeightHistoryPoint,
    WellnessCalculationResult,
    WellnessMilestone,
)


def clamp_score_to_hundred_scale(score: float) -> float:
    """Constrains a numeric score to [0.0, 100.0] and rounds to 2 decimal places."""
    numeric_score = float(score)
    if math.isnan(numeric_score) or math.isinf(numeric_score):
        return 0.0
    if numeric_score < 0.0:
        return 0.0
    if numeric_score > 100.0:
        return 100.0
    return round(numeric_score, 2)


def calculate_bmi_and_classification(weight_kg: float, height_cm: float) -> tuple[float, str]:
    """Computes BMI = weight_kg / (height_m ** 2) rounded to 2 decimals and classifies it."""
    if weight_kg <= 0 or height_cm <= 0:
        raise ValueError("weight_kg and height_cm must both be strictly positive")

    height_meters = float(height_cm) / 100.0
    bmi_value = round(float(weight_kg) / (height_meters ** 2), 2)

    if bmi_value < 18.5:
        classification = "underweight"
    elif bmi_value < 25.0:
        classification = "normal weight"
    elif bmi_value < 30.0:
        classification = "overweight"
    else:
        classification = "obese"

    return bmi_value, classification


def calculate_bmi_component_score(bmi: float) -> float:
    """Scores a BMI value on a [0.0, 100.0] scale centered around optimal range [18.5, 25.0)."""
    if 18.5 <= bmi < 25.0:
        distance_from_ideal_center = abs(bmi - 21.7)
        return clamp_score_to_hundred_scale(100.0 - (distance_from_ideal_center * 3.0))
    if bmi < 18.5:
        deficit = 18.5 - bmi
        return clamp_score_to_hundred_scale(85.0 - (deficit * 10.0))
    if 25.0 <= bmi < 30.0:
        excess = bmi - 25.0
        return clamp_score_to_hundred_scale(85.0 - (excess * 6.0))
    excess = bmi - 30.0
    return clamp_score_to_hundred_scale(55.0 - (excess * 3.5))


def map_activity_level_to_rank(activity_level: str) -> float:
    """Maps a categorical activity level string to a numeric tier from 1.0 to 4.0."""
    normalized_level = activity_level.strip().lower()
    if normalized_level == "sedentary":
        return 1.0
    if normalized_level in ("lightly_active", "light"):
        return 2.0
    if normalized_level in ("moderately_active", "moderate"):
        return 3.0
    if normalized_level in ("very_active", "active", "extra_active", "athlete"):
        return 4.0
    return 2.0


def _parse_iso_timestamp_utc(timestamp_text: str) -> Optional[datetime]:
    """Parses an ISO-8601 timestamp string into a UTC-aware datetime, returning None on failure."""
    cleaned = timestamp_text.strip()
    if not cleaned:
        return None
    if cleaned.endswith("Z"):
        cleaned = cleaned[:-1] + "+00:00"
    try:
        parsed = datetime.fromisoformat(cleaned)
    except ValueError:
        return None
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(timezone.utc)


def count_recent_weekly_activity_sessions(
    activity_history: list[AnonymizedActivityHistoryPoint],
) -> tuple[int, int]:
    """Counts activity sessions and total minutes within the 7-day window ending at latest or now."""
    if not activity_history:
        return 0, 0

    reference_time = datetime.now(timezone.utc)
    parsed_entries: list[tuple[Optional[datetime], int]] = []

    for entry in activity_history:
        parsed_time = _parse_iso_timestamp_utc(entry.recorded_at)
        if parsed_time is not None and parsed_time > reference_time:
            reference_time = parsed_time
        duration = entry.duration_minutes if entry.duration_minutes > 0 else 0
        parsed_entries.append((parsed_time, duration))

    seven_days_ago = reference_time - timedelta(days=7)
    recent_sessions = 0
    total_minutes = 0

    for parsed_time, duration in parsed_entries:
        if parsed_time is not None and parsed_time >= seven_days_ago:
            recent_sessions += 1
            total_minutes += duration

    if recent_sessions == 0:
        recent_sessions = len(parsed_entries)
        total_minutes = sum(duration for _, duration in parsed_entries)

    return recent_sessions, total_minutes


def calculate_activity_component_score(
    activity_level: str,
    weekly_activity_days: int,
    target_weekly_days: int,
    recent_sessions: int,
    total_minutes: int,
) -> float:
    """Computes the 0-100 activity component score from level rank, weekly frequency, and logged sessions."""
    resolved_target_days = target_weekly_days if target_weekly_days > 0 else 4
    effective_active_days = max(float(weekly_activity_days), float(recent_sessions))

    frequency_ratio = min(effective_active_days / float(resolved_target_days), 1.25)
    level_score = (map_activity_level_to_rank(activity_level) / 4.0) * 30.0
    frequency_score = min(frequency_ratio, 1.0) * 50.0
    logged_bonus = min(float(recent_sessions) * 5.0 + float(total_minutes) / 30.0, 20.0)

    return clamp_score_to_hundred_scale(level_score + frequency_score + logged_bonus)


def calculate_progress_component_score(
    current_weight_kg: float,
    baseline_weight_kg: float,
    target_weight_kg: float,
) -> tuple[float, float]:
    """Evaluates weight trajectory from baseline toward target, returning (progress_score, progress_fraction)."""
    total_goal_delta = abs(baseline_weight_kg - target_weight_kg)
    remaining_delta = abs(current_weight_kg - target_weight_kg)

    if total_goal_delta < 0.25:
        if remaining_delta <= 1.0:
            return 95.0, 1.0
        weight_progress_fraction = clamp_score_to_hundred_scale(100.0 - remaining_delta * 10.0) / 100.0
        progress_score = clamp_score_to_hundred_scale(90.0 - remaining_delta * 8.0)
        return progress_score, weight_progress_fraction

    achieved_delta = total_goal_delta - remaining_delta
    weight_progress_fraction = min(max(achieved_delta / total_goal_delta, 0.0), 1.0)
    progress_score = clamp_score_to_hundred_scale(40.0 + (weight_progress_fraction * 60.0))
    return progress_score, weight_progress_fraction


def calculate_habits_component_score(sleep_hours: float, water_liters: float) -> float:
    """Evaluates daily sleep duration and water intake on a clamped [0.0, 100.0] scale."""
    sleep_score = 35.0
    if 7.0 <= sleep_hours <= 9.0:
        sleep_score = 50.0
    elif 6.0 <= sleep_hours < 7.0:
        sleep_score = 38.0
    elif 9.0 < sleep_hours <= 10.0:
        sleep_score = 40.0
    elif sleep_hours > 0.0:
        sleep_score = max(15.0, 50.0 - abs(7.5 - sleep_hours) * 10.0)

    hydration_score = 25.0
    if water_liters > 0.0:
        hydration_ratio = min(water_liters / 2.5, 1.0)
        hydration_score = hydration_ratio * 50.0

    return clamp_score_to_hundred_scale(sleep_score + hydration_score)


def calculate_composite_wellness_score(
    bmi_score: float,
    activity_score: float,
    progress_score: float,
    habits_score: float,
) -> float:
    """Computes the weighted composite Wellness Score clamped to [0.0, 100.0]."""
    clamped_bmi = clamp_score_to_hundred_scale(bmi_score)
    clamped_activity = clamp_score_to_hundred_scale(activity_score)
    clamped_progress = clamp_score_to_hundred_scale(progress_score)
    clamped_habits = clamp_score_to_hundred_scale(habits_score)

    weighted_sum = (
        (clamped_bmi * 0.3)
        + (clamped_activity * 0.3)
        + (clamped_progress * 0.2)
        + (clamped_habits * 0.2)
    )
    return clamp_score_to_hundred_scale(weighted_sum)


def build_wellness_milestones(
    current_weight_kg: float,
    target_weight_kg: float,
    weight_progress_fraction: float,
    effective_active_days: float,
    target_days: int,
    total_logged_activities: int,
    sleep_hours: float,
    water_liters: float,
    habits_score: float,
) -> list[WellnessMilestone]:
    """Builds the 3 deterministic wellness milestones matching Go's WellnessCalculationResult."""
    weight_progress_pct = round(weight_progress_fraction * 100.0, 2)
    completed_five_pct_steps = int(math.floor(weight_progress_pct / 5.0))
    next_five_pct_target = min((completed_five_pct_steps + 1) * 5, 100)

    return [
        WellnessMilestone(
            id="weight-5pct-step",
            category="weight",
            title=f"Weight Goal Progress ({completed_five_pct_steps * 5}% Step)",
            description=(
                f"Progressed {weight_progress_pct:.1f}% toward target weight "
                f"({current_weight_kg:.1f} kg -> {target_weight_kg:.1f} kg; "
                f"next 5% milestone at {next_five_pct_target}%)."
            ),
            achieved=(completed_five_pct_steps >= 1) or (weight_progress_pct >= 100.0),
            progress_pct=weight_progress_pct,
        ),
        WellnessMilestone(
            id="weekly-activity-target",
            category="activity",
            title="Weekly Activity Consistency",
            description=(
                f"Active {effective_active_days:.0f} of {target_days} target weekly days "
                f"({total_logged_activities} logged sessions)."
            ),
            achieved=int(effective_active_days) >= target_days,
            progress_pct=clamp_score_to_hundred_scale((effective_active_days / float(target_days)) * 100.0),
        ),
        WellnessMilestone(
            id="daily-habits-consistency",
            category="habits",
            title="Sleep & Hydration Balance",
            description=f"Logging {sleep_hours:.1f} hrs sleep and {water_liters:.1f} L water daily.",
            achieved=(sleep_hours >= 7.0) and (water_liters >= 2.0),
            progress_pct=habits_score,
        ),
    ]


def calculate_deterministic_wellness_and_milestones(
    user_metrics: AnonymizedUserMetrics,
) -> WellnessCalculationResult:
    """Computes BMI, 4 component scores, composite Wellness Score, and 3 milestones."""
    current_state = user_metrics.current_state
    target_state = user_metrics.target_state
    weight_history = user_metrics.weight_history
    activity_history = user_metrics.activity_history

    current_weight_kg = current_state.weight
    if weight_history and weight_history[-1].weight_kg > 0:
        current_weight_kg = weight_history[-1].weight_kg

    bmi_value, bmi_classification = calculate_bmi_and_classification(
        weight_kg=current_weight_kg,
        height_cm=current_state.height,
    )
    bmi_score = calculate_bmi_component_score(bmi_value)

    resolved_target_days = target_state.target_weekly_days if target_state.target_weekly_days > 0 else 4
    recent_sessions, total_minutes = count_recent_weekly_activity_sessions(activity_history)
    effective_active_days = max(float(current_state.weekly_activity_days), float(recent_sessions))

    activity_score = calculate_activity_component_score(
        activity_level=current_state.activity_level,
        weekly_activity_days=current_state.weekly_activity_days,
        target_weekly_days=resolved_target_days,
        recent_sessions=recent_sessions,
        total_minutes=total_minutes,
    )

    baseline_weight_kg = target_state.baseline_weight
    if baseline_weight_kg <= 0:
        if weight_history and weight_history[0].weight_kg > 0:
            baseline_weight_kg = weight_history[0].weight_kg
        else:
            baseline_weight_kg = current_weight_kg

    target_weight_kg = target_state.weight if target_state.weight > 0 else current_weight_kg

    progress_score, weight_progress_fraction = calculate_progress_component_score(
        current_weight_kg=current_weight_kg,
        baseline_weight_kg=baseline_weight_kg,
        target_weight_kg=target_weight_kg,
    )

    habits_score = calculate_habits_component_score(
        sleep_hours=current_state.sleep_hours,
        water_liters=current_state.water_liters,
    )

    wellness_score = calculate_composite_wellness_score(
        bmi_score=bmi_score,
        activity_score=activity_score,
        progress_score=progress_score,
        habits_score=habits_score,
    )

    milestones = build_wellness_milestones(
        current_weight_kg=current_weight_kg,
        target_weight_kg=target_weight_kg,
        weight_progress_fraction=weight_progress_fraction,
        effective_active_days=effective_active_days,
        target_days=resolved_target_days,
        total_logged_activities=len(activity_history),
        sleep_hours=current_state.sleep_hours,
        water_liters=current_state.water_liters,
        habits_score=habits_score,
    )

    return WellnessCalculationResult(
        bmi=bmi_value,
        bmi_classification=bmi_classification,
        wellness_score=wellness_score,
        bmi_score=bmi_score,
        activity_score=activity_score,
        progress_score=progress_score,
        habits_score=habits_score,
        milestones=milestones,
    )


def _compute_linear_daily_slope(timestamps: pd.Series, values: pd.Series) -> float:
    """Computes the linear daily slope via NumPy polyfit, returning 0.0 when fewer than 2 distinct points."""
    if len(values) < 2:
        return 0.0

    elapsed_days = (timestamps - timestamps.iloc[0]).dt.total_seconds().to_numpy(dtype=float) / 86400.0
    numeric_values = values.to_numpy(dtype=float)

    if np.allclose(elapsed_days, elapsed_days[0]):
        return 0.0

    slope = float(np.polyfit(elapsed_days, numeric_values, 1)[0])
    if math.isnan(slope) or math.isinf(slope):
        return 0.0
    return round(slope, 4)


def _summarize_weight_window(window_df: pd.DataFrame) -> tuple[float, float, float]:
    """Returns (average_kg, net_delta_kg, linear_daily_slope) for a non-empty weight DataFrame."""
    if window_df.empty:
        return 0.0, 0.0, 0.0

    average_kg = round(float(window_df["weight_kg"].mean()), 2)
    net_delta_kg = round(float(window_df["weight_kg"].iloc[-1] - window_df["weight_kg"].iloc[0]), 2)
    slope_per_day = _compute_linear_daily_slope(window_df["recorded_at"], window_df["weight_kg"])
    return average_kg, net_delta_kg, slope_per_day


def compact_historical_metrics_for_ai_context(
    weight_history: list[AnonymizedWeightHistoryPoint],
    activity_history: list[AnonymizedActivityHistoryPoint],
) -> dict[str, object]:
    """Compacts unbounded weight & activity histories into 7-day and 30-day Pandas/NumPy summaries."""
    weight_summary: dict[str, object] = {
        "total_entries": 0,
        "latest_kg": None,
        "avg_7d_kg": None,
        "avg_30d_kg": None,
        "delta_7d_kg": 0.0,
        "delta_30d_kg": 0.0,
        "slope_7d_kg_per_day": 0.0,
        "slope_30d_kg_per_day": 0.0,
    }

    if weight_history:
        weight_df = pd.DataFrame(
            [{"weight_kg": float(point.weight_kg), "recorded_at": point.recorded_at} for point in weight_history]
        )
        weight_df["recorded_at"] = pd.to_datetime(weight_df["recorded_at"], utc=True, errors="coerce")
        weight_df = weight_df.dropna(subset=["recorded_at", "weight_kg"]).sort_values("recorded_at").reset_index(drop=True)

        if not weight_df.empty:
            latest_weight_time = weight_df["recorded_at"].iloc[-1]
            window_7d_df = weight_df[weight_df["recorded_at"] >= (latest_weight_time - pd.Timedelta(days=7))]
            window_30d_df = weight_df[weight_df["recorded_at"] >= (latest_weight_time - pd.Timedelta(days=30))]

            avg_7d, delta_7d, slope_7d = _summarize_weight_window(window_7d_df)
            avg_30d, delta_30d, slope_30d = _summarize_weight_window(window_30d_df)

            weight_summary = {
                "total_entries": int(len(weight_df)),
                "latest_kg": round(float(weight_df["weight_kg"].iloc[-1]), 2),
                "avg_7d_kg": avg_7d,
                "avg_30d_kg": avg_30d,
                "delta_7d_kg": delta_7d,
                "delta_30d_kg": delta_30d,
                "slope_7d_kg_per_day": slope_7d,
                "slope_30d_kg_per_day": slope_30d,
            }

    activity_summary: dict[str, object] = {
        "total_entries": 0,
        "sessions_7d": 0,
        "minutes_7d": 0,
        "avg_duration_7d_min": 0.0,
        "sessions_30d": 0,
        "minutes_30d": 0,
        "avg_duration_30d_min": 0.0,
        "dominant_exercise_type": "none",
        "dominant_intensity": "none",
    }

    if activity_history:
        activity_df = pd.DataFrame(
            [
                {
                    "exercise_type": str(point.exercise_type).strip() or "unknown",
                    "duration_minutes": max(int(point.duration_minutes), 0),
                    "intensity": str(point.intensity).strip() or "moderate",
                    "recorded_at": point.recorded_at,
                }
                for point in activity_history
            ]
        )
        activity_df["recorded_at"] = pd.to_datetime(activity_df["recorded_at"], utc=True, errors="coerce")
        activity_df = activity_df.dropna(subset=["recorded_at"]).sort_values("recorded_at").reset_index(drop=True)

        if not activity_df.empty:
            latest_activity_time = activity_df["recorded_at"].iloc[-1]
            act_7d_df = activity_df[activity_df["recorded_at"] >= (latest_activity_time - pd.Timedelta(days=7))]
            act_30d_df = activity_df[activity_df["recorded_at"] >= (latest_activity_time - pd.Timedelta(days=30))]

            sessions_7d = int(len(act_7d_df))
            minutes_7d = int(act_7d_df["duration_minutes"].sum()) if sessions_7d > 0 else 0
            avg_7d_min = round(float(minutes_7d / sessions_7d), 2) if sessions_7d > 0 else 0.0

            sessions_30d = int(len(act_30d_df))
            minutes_30d = int(act_30d_df["duration_minutes"].sum()) if sessions_30d > 0 else 0
            avg_30d_min = round(float(minutes_30d / sessions_30d), 2) if sessions_30d > 0 else 0.0

            dominant_exercise = str(activity_df["exercise_type"].mode().iloc[0])
            dominant_intensity = str(activity_df["intensity"].mode().iloc[0])

            activity_summary = {
                "total_entries": int(len(activity_df)),
                "sessions_7d": sessions_7d,
                "minutes_7d": minutes_7d,
                "avg_duration_7d_min": avg_7d_min,
                "sessions_30d": sessions_30d,
                "minutes_30d": minutes_30d,
                "avg_duration_30d_min": avg_30d_min,
                "dominant_exercise_type": dominant_exercise,
                "dominant_intensity": dominant_intensity,
            }

    return {
        "weight_trends": weight_summary,
        "activity_trends": activity_summary,
    }
