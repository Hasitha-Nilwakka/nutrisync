"""Synthetic clinical instruction dataset generator for NutriSync QLoRA fine-tuning.

Synthesizes anonymized health profiles across a combinatorial grid of goals, BMI classes,
7-day/30-day trend regimes, prompting strategies, and clinical/dietary constraints with
noisy synthetic PII. Every sample is processed through the real wellness engine, Layer 1
PII scrubber/constraint extractor, and Layer 2 LLaMA prompt constructor, producing
deterministic train.jsonl, valid.jsonl, and test.jsonl files in both ChatML `messages`
and LLaMA-3 `<|begin_of_text|>` prompt-completion formats.
"""

import argparse
from datetime import datetime, timedelta, timezone
import itertools
import json
from pathlib import Path
import random
from typing import Any, Dict, List, Literal, Optional, Sequence, Tuple

from analytics.wellness_engine import (
    calculate_deterministic_wellness_and_milestones,
    compact_historical_metrics_for_ai_context,
)
from pipeline.layer1_scispacy_extractor import (
    Layer1ClinicalExtractionResult,
    extract_clinical_constraints_and_scrub_pii,
)
from pipeline.layer2_prompt_constructor import construct_llama_prompt_payload
from pipeline.layer3_llama_validator import (
    filter_and_sanitize_insights_against_guardrails,
    find_matching_forbidden_term,
)
from schemas.analysis_schemas import (
    AnonymizedActivityHistoryPoint,
    AnonymizedCurrentState,
    AnonymizedFitnessAssessment,
    AnonymizedTargetState,
    AnonymizedUserMetrics,
    AnonymizedWeightHistoryPoint,
    PrioritizedAIRecommendation,
    PyAIInsightsPayload,
    WellnessCalculationResult,
)

PRIMARY_GOALS: Tuple[str, ...] = (
    "fat_loss",
    "muscle_gain",
    "maintenance",
    "endurance",
    "general_wellness",
)

BMI_SCENARIOS: Tuple[Dict[str, Any], ...] = (
    {"label": "underweight", "height_cm": 175.0, "weight_kg": 53.5, "target_weight_kg": 61.0},
    {"label": "normal weight", "height_cm": 175.0, "weight_kg": 68.5, "target_weight_kg": 67.0},
    {"label": "overweight", "height_cm": 175.0, "weight_kg": 84.5, "target_weight_kg": 74.0},
    {"label": "obese", "height_cm": 175.0, "weight_kg": 102.0, "target_weight_kg": 82.0},
)

TREND_REGIMES: Tuple[str, ...] = (
    "improving",
    "plateau",
    "regressing",
    "empty_history",
)

PROMPT_STRATEGIES: Tuple[Literal["few_shot", "zero_shot"], ...] = (
    "few_shot",
    "zero_shot",
)

CONSTRAINT_PROFILES: Tuple[Dict[str, Any], ...] = (
    {
        "name": "none",
        "restrictions": [],
        "medical_conditions": "",
    },
    {
        "name": "peanut_allergy",
        "restrictions": ["peanut_allergy"],
        "medical_conditions": "Peanut allergy. Contact Dr. Alice Vance at alice@example.com or 555-123-4567.",
    },
    {
        "name": "type_2_diabetes",
        "restrictions": ["low_sugar"],
        "medical_conditions": "Type 2 diabetes, MRN-998811, patient @john_doe.",
    },
    {
        "name": "hypertension",
        "restrictions": ["low_sodium"],
        "medical_conditions": "Hypertension and high blood pressure, call +1-800-555-0199.",
    },
    {
        "name": "knee_injury",
        "restrictions": [],
        "medical_conditions": "Left knee injury and knee pain. https://clinic.example.org",
    },
    {
        "name": "back_pain_lactose",
        "restrictions": ["lactose"],
        "medical_conditions": "Lower back pain and lactose intolerance.",
    },
    {
        "name": "vegan_gluten",
        "restrictions": ["vegan", "gluten"],
        "medical_conditions": "Celiac disease.",
    },
    {
        "name": "shellfish_gout",
        "restrictions": ["shellfish", "gout"],
        "medical_conditions": "History of gout and shellfish allergy.",
    },
)


def _build_synthetic_histories(
    base_weight_kg: float,
    primary_goal: str,
    trend_regime: str,
    has_joint_constraint: bool,
    rng: random.Random,
) -> Tuple[List[AnonymizedWeightHistoryPoint], List[AnonymizedActivityHistoryPoint], float]:
    """Build deterministic 30-day weight and activity histories for a trend regime."""
    if trend_regime == "empty_history":
        return [], [], base_weight_kg

    reference_time = datetime(2026, 11, 15, 8, 0, tzinfo=timezone.utc)
    if primary_goal == "muscle_gain":
        improving_step = 0.08
        regressing_step = -0.06
    elif primary_goal == "fat_loss":
        improving_step = -0.10
        regressing_step = 0.07
    else:
        improving_step = -0.03
        regressing_step = 0.05

    if trend_regime == "improving":
        step_per_point = improving_step
        sessions_count = 10
        duration_base = 40
    elif trend_regime == "plateau":
        step_per_point = 0.0
        sessions_count = 6
        duration_base = 30
    else:
        step_per_point = regressing_step
        sessions_count = 3
        duration_base = 20

    weight_points: List[AnonymizedWeightHistoryPoint] = []
    start_weight = round(base_weight_kg - (step_per_point * 9.0), 2)
    for index in range(10):
        day_offset = 27 - (index * 3)
        recorded_at = (reference_time - timedelta(days=day_offset)).isoformat().replace("+00:00", "Z")
        jitter = rng.uniform(-0.08, 0.08)
        point_weight = round(max(35.0, start_weight + (step_per_point * index) + jitter), 2)
        if index == 9:
            point_weight = round(base_weight_kg, 2)
        weight_points.append(
            AnonymizedWeightHistoryPoint(weight_kg=point_weight, recorded_at=recorded_at)
        )

    safe_exercise = "cycling" if has_joint_constraint else "brisk_walking"
    activity_points: List[AnonymizedActivityHistoryPoint] = []
    for index in range(sessions_count):
        day_offset = max(0, 26 - (index * 2))
        recorded_at = (reference_time - timedelta(days=day_offset)).isoformat().replace("+00:00", "Z")
        activity_points.append(
            AnonymizedActivityHistoryPoint(
                exercise_type=safe_exercise,
                duration_minutes=duration_base + (index % 3) * 5,
                intensity="moderate",
                recorded_at=recorded_at,
            )
        )

    return weight_points, activity_points, start_weight


def build_synthetic_user_metrics_for_scenario(
    primary_goal: str,
    bmi_scenario: Dict[str, Any],
    trend_regime: str,
    constraint_profile: Dict[str, Any],
    rng: random.Random,
) -> AnonymizedUserMetrics:
    """Construct a deterministic AnonymizedUserMetrics instance for a grid scenario."""
    weight_kg = float(bmi_scenario["weight_kg"])
    height_cm = float(bmi_scenario["height_cm"])
    target_weight_kg = float(bmi_scenario["target_weight_kg"])
    if primary_goal == "maintenance":
        target_weight_kg = weight_kg
    elif primary_goal == "muscle_gain" and target_weight_kg <= weight_kg:
        target_weight_kg = round(weight_kg + 3.0, 1)

    bmi_value = round(weight_kg / ((height_cm / 100.0) ** 2), 2)
    conditions_lower = str(constraint_profile["medical_conditions"]).lower()
    has_joint_constraint = "knee" in conditions_lower or "back" in conditions_lower

    weight_history, activity_history, baseline_weight = _build_synthetic_histories(
        base_weight_kg=weight_kg,
        primary_goal=primary_goal,
        trend_regime=trend_regime,
        has_joint_constraint=has_joint_constraint,
        rng=rng,
    )

    if trend_regime == "improving":
        weekly_days = 4
        sleep_hours = 7.6
        water_liters = 2.6
        activity_level = "moderately_active"
    elif trend_regime == "plateau":
        weekly_days = 3
        sleep_hours = 6.8
        water_liters = 2.1
        activity_level = "lightly_active"
    else:
        weekly_days = 1
        sleep_hours = 6.0
        water_liters = 1.5
        activity_level = "sedentary"

    return AnonymizedUserMetrics(
        current_state=AnonymizedCurrentState(
            weight=weight_kg,
            height=height_cm,
            age=rng.randint(24, 58),
            gender=rng.choice(["female", "male", "non_binary"]),
            activity_level=activity_level,
            weekly_activity_days=weekly_days,
            sleep_hours=sleep_hours,
            water_liters=water_liters,
            bmi=bmi_value,
            bmi_classification=str(bmi_scenario["label"]),
        ),
        target_state=AnonymizedTargetState(
            weight=target_weight_kg,
            activity_level="moderately_active",
            primary_goal=primary_goal,
            target_weekly_days=4,
            baseline_weight=baseline_weight,
        ),
        preferences=["balanced_meals", "whole_foods"],
        restrictions=list(constraint_profile["restrictions"]),
        medical_conditions=str(constraint_profile["medical_conditions"]),
        fitness_assessment=AnonymizedFitnessAssessment(
            fitness_level="intermediate" if weekly_days >= 3 else "beginner",
            session_duration="30-45 min",
            endurance_minutes=30 if weekly_days >= 3 else 15,
            pushups_count=15 if weekly_days >= 3 else 6,
            squats_count=0 if has_joint_constraint else 20,
        ),
        weight_history=weight_history,
        activity_history=activity_history,
    )


def _select_safe_nutrition_recommendation(
    primary_goal: str,
    layer1_result: Layer1ClinicalExtractionResult,
) -> PrioritizedAIRecommendation:
    """Create a restriction-safe nutrition recommendation aligned with primary_goal."""
    dietary_set = set(layer1_result.dietary_constraints)
    if "low_sugar" in dietary_set or "diabetes" in dietary_set:
        title = "High-Fiber Glycemic-Steady Meals"
        action = "Pair non-starchy vegetables with legumes, quinoa, and olive oil at each meal."
        impact = "Stabilizes post-meal energy and supports steady metabolic control."
    elif "low_sodium" in dietary_set or "hypertension" in dietary_set:
        title = "Herb-Seasoned Potassium-Rich Meals"
        action = "Season whole grains, leafy greens, and beans with lemon and fresh herbs instead of added salt."
        impact = "Supports healthy blood pressure balance while maintaining meal satisfaction."
    elif "vegan" in dietary_set or "gluten" in dietary_set or "celiac" in dietary_set:
        title = "Allergen-Safe Plant Protein Bowls"
        action = "Build meals around lentils, chickpeas, quinoa, and steamed greens with chia seeds."
        impact = "Delivers complete amino acids and fiber while strictly honoring dietary restrictions."
    elif "gout" in dietary_set or "shellfish" in dietary_set:
        title = "Hydration-Forward Plant & Whole-Grain Nutrition"
        action = "Combine oats, berries, chickpeas, and 2.5L daily water intake across balanced meals."
        impact = "Supports recovery and purine-conscious nutrition aligned with your goal."
    elif primary_goal == "muscle_gain":
        title = "Targeted Protein & Complex Carbohydrate Timing"
        action = "Distribute plant-forward protein, oats, and roasted vegetables across 3-4 daily meals."
        impact = "Supports lean tissue recovery and steady strength progression."
    elif primary_goal == "fat_loss":
        title = "Volume-Rich Calorie-Controlled Meals"
        action = "Center lunches and dinners on leafy greens, beans, and measured whole grains."
        impact = "Creates a sustainable calorie deficit while preserving fullness."
    else:
        title = "Consistent Whole-Food Meal Structure"
        action = "Maintain regular meal timing with fiber-rich vegetables, legumes, and healthy fats."
        impact = "Supports steady energy and long-term adherence to your goal."

    return PrioritizedAIRecommendation(
        title=title,
        category="nutrition",
        priority="medium",
        action=action,
        expected_impact=impact,
    )


def _select_safe_activity_recommendation(
    primary_goal: str,
    layer1_result: Layer1ClinicalExtractionResult,
) -> PrioritizedAIRecommendation:
    """Create an exercise recommendation honoring all Layer 1 exercise constraints."""
    exercise_set = set(layer1_result.exercise_constraints)
    has_joint_limit = bool(exercise_set & {"knee", "joint", "back"})
    has_cardio_limit = bool(exercise_set & {"hypertension", "asthma", "heart", "pregnant", "pregnancy"})

    if has_joint_limit:
        action = "Complete three 30-minute low-impact stationary cycling or pool sessions per week."
        impact = f"Advances {primary_goal} fitness safely while protecting sensitive joints."
    elif has_cardio_limit:
        action = "Perform three 25-minute moderate-pace walks with controlled nasal breathing."
        impact = f"Builds {primary_goal} consistency without excessive cardiovascular strain."
    elif primary_goal == "muscle_gain":
        action = "Schedule three progressive resistance sessions weekly focusing on controlled tempo."
        impact = "Stimulates lean muscle adaptation aligned with your muscle_gain target."
    elif primary_goal == "endurance":
        action = "Add two 35-minute steady Zone-2 aerobic sessions and one mobility session weekly."
        impact = "Expands aerobic capacity and endurance stamina."
    else:
        action = "Complete three 30-minute brisk walks or moderate resistance circuits this week."
        impact = f"Supports steady {primary_goal} momentum and improves weekly activity score."

    return PrioritizedAIRecommendation(
        title="Goal-Aligned Safe Movement Plan",
        category="activity",
        priority="high",
        action=action,
        expected_impact=impact,
    )


def generate_safe_clinical_target_insights(
    user_metrics: AnonymizedUserMetrics,
    wellness_result: WellnessCalculationResult,
    compacted_trends: Dict[str, Any],
    layer1_result: Layer1ClinicalExtractionResult,
    prompt_strategy: str,
) -> PyAIInsightsPayload:
    """Construct a deterministic, schema-valid, zero-violation target PyAIInsightsPayload."""
    primary_goal = user_metrics.target_state.primary_goal
    weight_trends = compacted_trends.get("weight_trends", {})
    activity_trends = compacted_trends.get("activity_trends", {})
    delta_7d = float(weight_trends.get("delta_7d_kg", 0.0) or 0.0)
    sessions_7d = int(activity_trends.get("sessions_7d", 0) or 0)
    minutes_7d = int(activity_trends.get("minutes_7d", 0) or 0)
    sessions_30d = int(activity_trends.get("sessions_30d", 0) or 0)

    health_status = (
        f"BMI is {wellness_result.bmi:.2f} ({wellness_result.bmi_classification}) with a composite "
        f"Wellness Score of {wellness_result.wellness_score:.2f}/100 (BMI {wellness_result.bmi_score:.1f}, "
        f"Activity {wellness_result.activity_score:.1f}, Habits {wellness_result.habits_score:.1f})."
    )
    progress_eval = (
        f"7-day weight delta is {delta_7d:+.2f} kg with {sessions_7d} logged sessions "
        f"({minutes_7d} minutes) supporting your {primary_goal} objective."
    )

    recommendations = [
        _select_safe_activity_recommendation(primary_goal=primary_goal, layer1_result=layer1_result),
        _select_safe_nutrition_recommendation(primary_goal=primary_goal, layer1_result=layer1_result),
    ]
    if wellness_result.habits_score < 85.0:
        recommendations.append(
            PrioritizedAIRecommendation(
                title="Sleep & Hydration Consistency",
                category="habits",
                priority="medium",
                action=(
                    f"Increase sleep toward 7.5-8.0 hours (currently {user_metrics.current_state.sleep_hours:.1f}h) "
                    f"and target 2.5L water daily (currently {user_metrics.current_state.water_liters:.1f}L)."
                ),
                expected_impact="Improves recovery and raises your habits score component.",
            )
        )

    candidate = PyAIInsightsPayload(
        prompt_strategy=prompt_strategy,
        health_status_analysis=health_status,
        progress_evaluation=progress_eval,
        recommendations=recommendations,
        weekly_summary=(
            f"Completed {sessions_7d} sessions ({minutes_7d} min) over 7 days toward {primary_goal}; "
            f"maintain steady adherence."
        ),
        monthly_summary=(
            f"30-day progress score is {wellness_result.progress_score:.1f}/100 across {sessions_30d} "
            f"recorded activity sessions."
        ),
    )
    return filter_and_sanitize_insights_against_guardrails(
        insights=candidate,
        layer1_result=layer1_result,
        primary_goal=primary_goal,
    )


def generate_dataset_records(
    num_samples: int = 320,
    seed: int = 42,
) -> List[Dict[str, Any]]:
    """Generate deterministic, zero-PII instruction-tuning records from the real pipeline."""
    if num_samples < 3:
        raise ValueError("num_samples must be at least 3 to populate train, valid, and test splits")

    rng = random.Random(seed)
    full_grid = list(
        itertools.product(
            PRIMARY_GOALS,
            BMI_SCENARIOS,
            TREND_REGIMES,
            PROMPT_STRATEGIES,
            CONSTRAINT_PROFILES,
        )
    )
    rng.shuffle(full_grid)

    selected_scenarios: List[Tuple[str, Dict[str, Any], str, Literal["few_shot", "zero_shot"], Dict[str, Any]]] = []
    for index in range(num_samples):
        selected_scenarios.append(full_grid[index % len(full_grid)])

    records: List[Dict[str, Any]] = []
    for index, (primary_goal, bmi_scenario, trend_regime, strategy, constraint_profile) in enumerate(selected_scenarios):
        user_metrics = build_synthetic_user_metrics_for_scenario(
            primary_goal=primary_goal,
            bmi_scenario=bmi_scenario,
            trend_regime=trend_regime,
            constraint_profile=constraint_profile,
            rng=rng,
        )
        wellness_result = calculate_deterministic_wellness_and_milestones(user_metrics)
        compacted_trends = compact_historical_metrics_for_ai_context(
            weight_history=user_metrics.weight_history,
            activity_history=user_metrics.activity_history,
        )
        layer1_result = extract_clinical_constraints_and_scrub_pii(user_metrics)
        constructed_prompt = construct_llama_prompt_payload(
            user_metrics=user_metrics,
            wellness_result=wellness_result,
            compacted_trends=compacted_trends,
            layer1_result=layer1_result,
            requested_strategy=strategy,
        )
        target_insights = generate_safe_clinical_target_insights(
            user_metrics=user_metrics,
            wellness_result=wellness_result,
            compacted_trends=compacted_trends,
            layer1_result=layer1_result,
            prompt_strategy=constructed_prompt.prompt_strategy,
        )
        completion_dict = target_insights.model_dump(by_alias=False)
        completion_json = json.dumps(completion_dict, separators=(",", ":"))

        matched_forbidden = find_matching_forbidden_term(completion_json, layer1_result.forbidden_terms)
        if matched_forbidden is not None:
            raise ValueError(
                f"Generated completion violated forbidden term '{matched_forbidden}' on sample {index}"
            )

        records.append(
            {
                "messages": constructed_prompt.messages + [{"role": "assistant", "content": completion_json}],
                "prompt": constructed_prompt.formatted_llama_prompt,
                "completion": f"{completion_json}<|eot_id|>",
                "text": f"{constructed_prompt.formatted_llama_prompt}{completion_json}<|eot_id|>",
                "metadata": {
                    "sample_id": f"nutrisync-{index:05d}",
                    "primary_goal": primary_goal,
                    "bmi_classification": wellness_result.bmi_classification,
                    "trend_regime": trend_regime,
                    "prompt_strategy": constructed_prompt.prompt_strategy,
                    "constraint_profile": str(constraint_profile["name"]),
                    "forbidden_terms": layer1_result.forbidden_terms,
                    "detected_pii_categories": layer1_result.detected_pii_categories,
                },
            }
        )

    return records


def split_dataset_records(
    records: Sequence[Dict[str, Any]],
) -> Tuple[List[Dict[str, Any]], List[Dict[str, Any]], List[Dict[str, Any]]]:
    """Split generated records deterministically into 80% train, 10% valid, and 10% test."""
    total = len(records)
    if total < 3:
        raise ValueError("At least 3 records are required to form train, valid, and test splits")

    valid_count = max(1, int(round(total * 0.1)))
    test_count = max(1, int(round(total * 0.1)))
    train_count = total - valid_count - test_count
    if train_count < 1:
        train_count = total - 2
        valid_count = 1
        test_count = 1

    train_records = list(records[:train_count])
    valid_records = list(records[train_count : train_count + valid_count])
    test_records = list(records[train_count + valid_count :])

    for split_name, split_rows in (
        ("train", train_records),
        ("valid", valid_records),
        ("test", test_records),
    ):
        for row in split_rows:
            row["metadata"]["split"] = split_name

    return train_records, valid_records, test_records


def write_jsonl_split_file(output_path: Path, records: Sequence[Dict[str, Any]]) -> None:
    """Write a list of dataset records to a UTF-8 JSONL file."""
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with output_path.open("w", encoding="utf-8") as handle:
        for row in records:
            handle.write(json.dumps(row, ensure_ascii=False) + "\n")


def generate_and_save_dataset_splits(
    output_dir: Path,
    num_samples: int = 320,
    seed: int = 42,
) -> Dict[str, int]:
    """Generate and persist train.jsonl, valid.jsonl, and test.jsonl in output_dir."""
    records = generate_dataset_records(num_samples=num_samples, seed=seed)
    train_records, valid_records, test_records = split_dataset_records(records)

    write_jsonl_split_file(output_dir / "train.jsonl", train_records)
    write_jsonl_split_file(output_dir / "valid.jsonl", valid_records)
    write_jsonl_split_file(output_dir / "test.jsonl", test_records)

    return {
        "train": len(train_records),
        "valid": len(valid_records),
        "test": len(test_records),
        "total": len(records),
    }


def main(argv: Optional[Sequence[str]] = None) -> int:
    """CLI entrypoint for generating synthetic clinical instruction datasets."""
    parser = argparse.ArgumentParser(
        description="Generate deterministic, zero-PII NutriSync clinical instruction datasets."
    )
    parser.add_argument(
        "--num-samples",
        type=int,
        default=320,
        help="Total number of synthetic samples to generate across train/valid/test (default: 320).",
    )
    parser.add_argument(
        "--seed",
        type=int,
        default=42,
        help="Deterministic random seed (default: 42).",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path(__file__).resolve().parent / "data",
        help="Directory where train.jsonl, valid.jsonl, and test.jsonl will be written.",
    )
    args = parser.parse_args(argv)

    counts = generate_and_save_dataset_splits(
        output_dir=args.output_dir,
        num_samples=args.num_samples,
        seed=args.seed,
    )
    print(json.dumps({"output_dir": str(args.output_dir), "counts": counts}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
