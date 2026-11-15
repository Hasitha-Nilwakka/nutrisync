"""FastAPI entrypoint for NutriSync's stateless Python analytics and 3-layer AI service."""

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from analytics.wellness_engine import (
    calculate_deterministic_wellness_and_milestones,
    compact_historical_metrics_for_ai_context,
)
from pipeline.layer1_scispacy_extractor import extract_clinical_constraints_and_scrub_pii
from pipeline.layer3_llama_validator import generate_validated_ai_insights
from schemas.analysis_schemas import (
    AnonymizedAIAnalysisPayload,
    PyAIAnalysisResponse,
)

app = FastAPI(title="NutriSync AI & Analytics Service")


@app.exception_handler(Exception)
def handle_unexpected_internal_error(request: Request, exc: Exception) -> JSONResponse:
    """Return a safe structured JSON error without leaking stack traces or PII."""
    return JSONResponse(
        status_code=500,
        content={"error": "Internal analysis error"},
    )


@app.get("/health")
def get_python_service_health_status():
    """Return service liveness metadata."""
    return {"status": "ok", "service": "pi-ai"}


@app.post("/api/v1/analyze", response_model=PyAIAnalysisResponse)
def analyze_anonymized_health_metrics(payload: AnonymizedAIAnalysisPayload):
    """Compute deterministic wellness metrics and 3-layer guardrailed AI insights."""
    if payload.user_metrics.current_state.weight <= 0 or payload.user_metrics.current_state.height <= 0:
        return JSONResponse(
            status_code=400,
            content={"error": "weight and height must both be strictly positive"},
        )

    try:
        wellness_result = calculate_deterministic_wellness_and_milestones(payload.user_metrics)
    except ValueError as exc:
        return JSONResponse(status_code=400, content={"error": str(exc)})

    compacted_trends = compact_historical_metrics_for_ai_context(
        weight_history=payload.user_metrics.weight_history,
        activity_history=payload.user_metrics.activity_history,
    )
    layer1_result = extract_clinical_constraints_and_scrub_pii(payload.user_metrics)
    insights_result = generate_validated_ai_insights(
        user_metrics=payload.user_metrics,
        wellness_result=wellness_result,
        compacted_trends=compacted_trends,
        layer1_result=layer1_result,
    )
    return PyAIAnalysisResponse(wellness=wellness_result, insights=insights_result)
