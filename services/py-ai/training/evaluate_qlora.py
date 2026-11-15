"""Guardrail & schema evaluation benchmark for NutriSync fine-tuned LLaMA models.

Evaluates any live LLM endpoint (Ollama, OpenAI-compatible) or pre-collected predictions
against `test.jsonl` and reports:
  (a) JSON schema validity rate (`PyAIInsightsPayload`)
  (b) `primary_goal` alignment rate
  (c) Pre-Layer-3 forbidden-term violation rate
  (d) Post-Layer-3 guardrail violation rate (must be 0.0%)
"""

import argparse
import json
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence

import httpx
from pydantic import BaseModel

from pipeline.layer1_scispacy_extractor import Layer1ClinicalExtractionResult
from pipeline.layer2_prompt_constructor import ConstructedLlamaPrompt
from pipeline.layer3_llama_validator import (
    LLMClientConfiguration,
    filter_and_sanitize_insights_against_guardrails,
    find_matching_forbidden_term,
    invoke_llama_completion_endpoint,
    parse_llm_insights_json,
)
from schemas.analysis_schemas import PrioritizedAIRecommendation, PyAIInsightsPayload

_GOAL_ALIGNMENT_KEYWORDS: Dict[str, tuple[str, ...]] = {
    "fat_loss": ("fat_loss", "fat loss", "calorie", "deficit", "weight", "walk", "cardio"),
    "muscle_gain": ("muscle_gain", "muscle", "strength", "resistance", "protein", "lean"),
    "maintenance": ("maintenance", "maintain", "steady", "balance", "consistency", "adherence"),
    "endurance": ("endurance", "aerobic", "stamina", "zone-2", "cycling", "walking", "cardio"),
    "general_wellness": ("general_wellness", "wellness", "sleep", "hydration", "movement", "habits"),
}


class EvaluationBenchmarkReport(BaseModel):
    """Structured evaluation metrics for NutriSync clinical LLM benchmarking."""

    total_samples: int
    valid_json_count: int
    json_schema_validity_rate: float
    goal_aligned_count: int
    primary_goal_alignment_rate: float
    pre_layer3_violation_count: int
    pre_layer3_forbidden_term_violation_rate: float
    post_layer3_violation_count: int
    post_layer3_forbidden_term_violation_rate: float
    fallback_count: int
    fallback_rate: float


def load_test_jsonl_records(
    test_file: Path,
    limit: Optional[int] = None,
) -> List[Dict[str, Any]]:
    """Load evaluation records from a JSONL file."""
    if not test_file.exists():
        raise FileNotFoundError(f"Test dataset file not found: {test_file}")

    records: List[Dict[str, Any]] = []
    with test_file.open("r", encoding="utf-8") as handle:
        for line in handle:
            stripped = line.strip()
            if not stripped:
                continue
            records.append(json.loads(stripped))
            if limit is not None and len(records) >= limit:
                break
    return records


def _flatten_insights_text(insights: PyAIInsightsPayload) -> str:
    """Concatenate all narrative and recommendation fields of PyAIInsightsPayload."""
    rec_parts = [
        f"{rec.title} {rec.category} {rec.action} {rec.expected_impact}"
        for rec in insights.recommendations
    ]
    return " ".join(
        [
            insights.health_status_analysis,
            insights.progress_evaluation,
            insights.weekly_summary,
            insights.monthly_summary,
            *rec_parts,
        ]
    )


def check_primary_goal_alignment(
    insights: PyAIInsightsPayload,
    primary_goal: str,
) -> bool:
    """Check whether insights narratives or recommendations align with primary_goal."""
    normalized_goal = (primary_goal or "general_wellness").strip().lower()
    keywords = _GOAL_ALIGNMENT_KEYWORDS.get(normalized_goal, (normalized_goal,))
    combined_text = _flatten_insights_text(insights).lower()
    return any(keyword in combined_text for keyword in keywords)


def check_insights_contain_forbidden_term(
    insights: PyAIInsightsPayload,
    forbidden_terms: List[str],
) -> Optional[str]:
    """Return the first forbidden term found anywhere in PyAIInsightsPayload, or None."""
    combined_text = _flatten_insights_text(insights)
    return find_matching_forbidden_term(combined_text, forbidden_terms)


def _build_minimal_fallback_payload(prompt_strategy: str, primary_goal: str) -> PyAIInsightsPayload:
    """Create a minimal deterministic payload when raw model output fails JSON parsing."""
    return PyAIInsightsPayload(
        prompt_strategy=prompt_strategy,
        health_status_analysis="Deterministic fallback summary due to invalid model JSON.",
        progress_evaluation=f"Progress tracking continues toward {primary_goal}.",
        recommendations=[
            PrioritizedAIRecommendation(
                title="Safe Daily Movement & Hydration",
                category="goal_alignment",
                priority="medium",
                action=f"Maintain steady hydration and low-impact movement aligned with {primary_goal}.",
                expected_impact="Preserves safety and consistency when LLM output is malformed.",
            )
        ],
        weekly_summary="Weekly fallback summary generated safely.",
        monthly_summary="Monthly fallback summary generated safely.",
    )


def evaluate_predictions_against_test_records(
    records: Sequence[Dict[str, Any]],
    raw_predictions: Sequence[Optional[str]],
) -> EvaluationBenchmarkReport:
    """Compute schema validity, goal alignment, and pre/post-Layer-3 violation rates."""
    if len(records) != len(raw_predictions):
        raise ValueError(
            f"Record count ({len(records)}) must match raw_predictions count ({len(raw_predictions)})"
        )
    total = len(records)
    if total == 0:
        raise ValueError("At least one test record is required for evaluation")

    valid_json_count = 0
    goal_aligned_count = 0
    pre_layer3_violation_count = 0
    post_layer3_violation_count = 0
    fallback_count = 0

    for record, raw_prediction in zip(records, raw_predictions):
        metadata = record.get("metadata", {})
        primary_goal = str(metadata.get("primary_goal", "general_wellness"))
        prompt_strategy = str(metadata.get("prompt_strategy", "few_shot"))
        forbidden_terms = list(metadata.get("forbidden_terms", []))

        layer1_result = Layer1ClinicalExtractionResult(forbidden_terms=forbidden_terms)
        parsed = parse_llm_insights_json(raw_prediction or "", prompt_strategy=prompt_strategy)

        if parsed is not None:
            valid_json_count += 1
            if check_primary_goal_alignment(parsed, primary_goal):
                goal_aligned_count += 1
            if check_insights_contain_forbidden_term(parsed, forbidden_terms) is not None:
                pre_layer3_violation_count += 1
            sanitized = filter_and_sanitize_insights_against_guardrails(
                insights=parsed,
                layer1_result=layer1_result,
                primary_goal=primary_goal,
            )
        else:
            fallback_count += 1
            if raw_prediction and find_matching_forbidden_term(raw_prediction, forbidden_terms) is not None:
                pre_layer3_violation_count += 1
            sanitized = filter_and_sanitize_insights_against_guardrails(
                insights=_build_minimal_fallback_payload(prompt_strategy, primary_goal),
                layer1_result=layer1_result,
                primary_goal=primary_goal,
            )

        if check_insights_contain_forbidden_term(sanitized, forbidden_terms) is not None:
            post_layer3_violation_count += 1

    return EvaluationBenchmarkReport(
        total_samples=total,
        valid_json_count=valid_json_count,
        json_schema_validity_rate=round((valid_json_count / total) * 100.0, 2),
        goal_aligned_count=goal_aligned_count,
        primary_goal_alignment_rate=round((goal_aligned_count / total) * 100.0, 2),
        pre_layer3_violation_count=pre_layer3_violation_count,
        pre_layer3_forbidden_term_violation_rate=round((pre_layer3_violation_count / total) * 100.0, 2),
        post_layer3_violation_count=post_layer3_violation_count,
        post_layer3_forbidden_term_violation_rate=round((post_layer3_violation_count / total) * 100.0, 2),
        fallback_count=fallback_count,
        fallback_rate=round((fallback_count / total) * 100.0, 2),
    )


def _reconstruct_prompt_from_record(record: Dict[str, Any]) -> ConstructedLlamaPrompt:
    """Build a ConstructedLlamaPrompt from a serialized test.jsonl record."""
    messages = [
        {"role": str(msg["role"]), "content": str(msg["content"])}
        for msg in record.get("messages", [])
        if msg.get("role") in ("system", "user")
    ]
    system_prompt = next((m["content"] for m in messages if m["role"] == "system"), "")
    user_prompt = next((m["content"] for m in messages if m["role"] == "user"), "")
    strategy = record.get("metadata", {}).get("prompt_strategy", "few_shot")
    if strategy not in ("few_shot", "zero_shot"):
        strategy = "few_shot"
    return ConstructedLlamaPrompt(
        prompt_strategy=strategy,
        system_prompt=system_prompt,
        user_prompt=user_prompt,
        formatted_llama_prompt=str(record.get("prompt", "")),
        messages=messages,
    )


def run_live_endpoint_evaluation(
    test_file: Path,
    configuration: LLMClientConfiguration,
    limit: Optional[int] = None,
    http_client: Optional[httpx.Client] = None,
) -> EvaluationBenchmarkReport:
    """Query a configured LLM endpoint across test.jsonl and return the benchmark report."""
    records = load_test_jsonl_records(test_file=test_file, limit=limit)
    raw_predictions: List[Optional[str]] = []

    for record in records:
        constructed_prompt = _reconstruct_prompt_from_record(record)
        raw_output = invoke_llama_completion_endpoint(
            constructed_prompt=constructed_prompt,
            configuration=configuration,
            http_client=http_client,
        )
        raw_predictions.append(raw_output)

    return evaluate_predictions_against_test_records(
        records=records,
        raw_predictions=raw_predictions,
    )


def main(argv: Optional[Sequence[str]] = None) -> int:
    """CLI entrypoint for running the NutriSync guardrail and schema evaluation benchmark."""
    parser = argparse.ArgumentParser(
        description="Evaluate a live LLM endpoint against NutriSync test.jsonl."
    )
    parser.add_argument(
        "--test-file",
        type=Path,
        default=Path(__file__).resolve().parent / "data" / "test.jsonl",
    )
    parser.add_argument("--api-url", default="http://localhost:11434/api/chat")
    parser.add_argument("--model", default="nutrisync-llama")
    parser.add_argument("--api-format", default="ollama_chat")
    parser.add_argument("--timeout-seconds", type=float, default=15.0)
    parser.add_argument("--limit", type=int, default=None)
    parser.add_argument("--output-report", type=Path, default=None)
    args = parser.parse_args(argv)

    configuration = LLMClientConfiguration(
        api_url=args.api_url,
        model=args.model,
        api_format=args.api_format,
        timeout_seconds=args.timeout_seconds,
    )
    report = run_live_endpoint_evaluation(
        test_file=args.test_file,
        configuration=configuration,
        limit=args.limit,
    )
    report_json = report.model_dump_json(indent=2)
    if args.output_report is not None:
        args.output_report.parent.mkdir(parents=True, exist_ok=True)
        args.output_report.write_text(report_json + "\n", encoding="utf-8")
    print(report_json)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
