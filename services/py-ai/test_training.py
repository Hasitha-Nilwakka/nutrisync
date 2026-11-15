"""Deterministic unit tests for NutriSync dataset generation, QLoRA config, and evaluation."""

import json
from pathlib import Path

import pytest
from pydantic import ValidationError

from pipeline.layer3_llama_validator import find_matching_forbidden_term
from schemas.analysis_schemas import PyAIInsightsPayload
from training.evaluate_qlora import (
    evaluate_predictions_against_test_records,
    load_test_jsonl_records,
)
from training.generate_dataset import (
    generate_and_save_dataset_splits,
    generate_dataset_records,
)
from training.train_qlora import (
    QLoRATrainingConfig,
    compute_effective_batch_size,
    compute_lora_scaling_factor,
    execute_training_pipeline,
    main as train_qlora_main,
)


def test_generate_dataset_is_deterministic_and_zero_pii(tmp_path: Path) -> None:
    """Dataset generator must produce identical splits with zero residual PII tokens."""
    first_run = generate_dataset_records(num_samples=40, seed=42)
    second_run = generate_dataset_records(num_samples=40, seed=42)
    assert first_run == second_run

    counts = generate_and_save_dataset_splits(output_dir=tmp_path, num_samples=40, seed=42)
    assert counts == {"train": 32, "valid": 4, "test": 4, "total": 40}
    assert (tmp_path / "train.jsonl").exists()
    assert (tmp_path / "valid.jsonl").exists()
    assert (tmp_path / "test.jsonl").exists()

    raw_pii_markers = (
        "alice@example.com",
        "555-123-4567",
        "Alice Vance",
        "MRN-998811",
        "@john_doe",
        "+1-800-555-0199",
        "https://clinic.example.org",
    )

    for split_name in ("train.jsonl", "valid.jsonl", "test.jsonl"):
        rows = load_test_jsonl_records(tmp_path / split_name)
        for row in rows:
            assert len(row["messages"]) == 3
            assert row["messages"][0]["role"] == "system"
            assert row["messages"][1]["role"] == "user"
            assert row["messages"][2]["role"] == "assistant"
            assert row["prompt"].startswith("<|begin_of_text|>")
            assert row["completion"].endswith("<|eot_id|>")
            assert row["text"].startswith(row["prompt"])

            serialized_content = json.dumps(row["messages"]) + row["text"]
            for pii_marker in raw_pii_markers:
                assert pii_marker not in serialized_content


def test_generated_completions_validate_schema_and_zero_forbidden_terms() -> None:
    """Every ground-truth completion must satisfy PyAIInsightsPayload and have 0 forbidden terms."""
    records = generate_dataset_records(num_samples=80, seed=7)
    for row in records:
        assistant_json = row["messages"][2]["content"]
        parsed = PyAIInsightsPayload.model_validate_json(assistant_json)
        assert len(parsed.recommendations) >= 2

        forbidden_terms = row["metadata"]["forbidden_terms"]
        matched = find_matching_forbidden_term(assistant_json, forbidden_terms)
        assert matched is None, f"Unexpected forbidden term '{matched}' in {row['metadata']['sample_id']}"


def test_qlora_training_config_validation_and_dry_run(tmp_path: Path) -> None:
    """QLoRATrainingConfig must validate hyperparameters and support dependency-free dry-run."""
    config = QLoRATrainingConfig(dataset_dir=str(tmp_path), output_dir=str(tmp_path / "out"))
    assert config.base_model == "meta-llama/Llama-3.2-1B-Instruct"
    assert config.lora_r == 16
    assert config.lora_alpha == 32
    assert compute_lora_scaling_factor(config) == 2.0
    assert compute_effective_batch_size(config) == 16
    assert len(config.target_modules) == 7

    with pytest.raises(ValidationError):
        QLoRATrainingConfig(lora_r=2)

    with pytest.raises(ValidationError):
        QLoRATrainingConfig(target_modules=[])

    summary = execute_training_pipeline(config=config, dry_run=True)
    assert summary["mode"] == "dry_run"
    assert summary["lora_math"]["scaling_factor"] == 2.0
    assert train_qlora_main(["--dry-run", "--dataset-dir", str(tmp_path)]) == 0


def test_evaluate_qlora_benchmark_metrics_math_and_post_layer3_zero_violations() -> None:
    """Evaluation benchmark must accurately compute schema, goal, pre- and post-Layer-3 rates."""
    records = [
        {
            "metadata": {
                "primary_goal": "fat_loss",
                "prompt_strategy": "few_shot",
                "forbidden_terms": ["peanut", "peanuts"],
            }
        },
        {
            "metadata": {
                "primary_goal": "muscle_gain",
                "prompt_strategy": "few_shot",
                "forbidden_terms": ["peanut", "peanuts"],
            }
        },
        {
            "metadata": {
                "primary_goal": "endurance",
                "prompt_strategy": "zero_shot",
                "forbidden_terms": ["deep squats"],
            }
        },
    ]

    clean_fat_loss = json.dumps(
        {
            "health_status_analysis": "BMI is 26.0 (overweight) with steady hydration.",
            "progress_evaluation": "Consistent 7-day progress toward fat_loss.",
            "recommendations": [
                {
                    "title": "Brisk Walking",
                    "category": "activity",
                    "priority": "high",
                    "action": "Walk 30 minutes three times weekly for fat_loss.",
                    "expected_impact": "Supports calorie deficit.",
                }
            ],
            "weekly_summary": "Good weekly adherence.",
            "monthly_summary": "Steady 30-day progress.",
        }
    )

    unsafe_muscle_gain = json.dumps(
        {
            "health_status_analysis": "BMI is 22.0 (normal weight).",
            "progress_evaluation": "Good momentum toward muscle_gain.",
            "recommendations": [
                {
                    "title": "Peanut Butter Shake",
                    "category": "nutrition",
                    "priority": "high",
                    "action": "Blend peanuts and oats after resistance training.",
                    "expected_impact": "Adds calories for muscle_gain.",
                }
            ],
            "weekly_summary": "Strong lifting week.",
            "monthly_summary": "Consistent strength gains.",
        }
    )

    malformed_endurance = "not valid json {{"

    report = evaluate_predictions_against_test_records(
        records=records,
        raw_predictions=[clean_fat_loss, unsafe_muscle_gain, malformed_endurance],
    )

    assert report.total_samples == 3
    assert report.valid_json_count == 2
    assert report.json_schema_validity_rate == 66.67
    assert report.goal_aligned_count == 2
    assert report.primary_goal_alignment_rate == 66.67
    assert report.pre_layer3_violation_count == 1
    assert report.pre_layer3_forbidden_term_violation_rate == 33.33
    assert report.post_layer3_violation_count == 0
    assert report.post_layer3_forbidden_term_violation_rate == 0.0
    assert report.fallback_count == 1
    assert report.fallback_rate == 33.33
