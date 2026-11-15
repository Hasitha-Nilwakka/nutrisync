"""3-Layer AI Guardrail Pipeline for stateless health insights in services/py-ai."""

from pipeline.layer1_scispacy_extractor import (
    Layer1ClinicalExtractionResult,
    build_forbidden_terms_from_restrictions_and_conditions,
    extract_clinical_constraints_and_scrub_pii,
    load_spacy_nlp_pipeline,
    scrub_residual_pii_from_free_text,
)
from pipeline.layer2_prompt_constructor import (
    ConstructedLlamaPrompt,
    build_few_shot_examples_block,
    construct_llama_prompt_payload,
    format_compacted_metrics_summary,
    select_prompt_strategy,
)
from pipeline.layer3_llama_validator import (
    LLMClientConfiguration,
    build_deterministic_clinical_fallback_insights,
    filter_and_sanitize_insights_against_guardrails,
    find_matching_forbidden_term,
    generate_validated_ai_insights,
    invoke_llama_completion_endpoint,
    parse_llm_insights_json,
)

__all__ = [
    "ConstructedLlamaPrompt",
    "LLMClientConfiguration",
    "Layer1ClinicalExtractionResult",
    "build_deterministic_clinical_fallback_insights",
    "build_few_shot_examples_block",
    "build_forbidden_terms_from_restrictions_and_conditions",
    "construct_llama_prompt_payload",
    "extract_clinical_constraints_and_scrub_pii",
    "filter_and_sanitize_insights_against_guardrails",
    "find_matching_forbidden_term",
    "format_compacted_metrics_summary",
    "generate_validated_ai_insights",
    "invoke_llama_completion_endpoint",
    "load_spacy_nlp_pipeline",
    "parse_llm_insights_json",
    "scrub_residual_pii_from_free_text",
    "select_prompt_strategy",
]
