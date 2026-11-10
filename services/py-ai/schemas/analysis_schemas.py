from typing import List, Optional
from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel


# INBOUND REQUEST SCHEMAS

class AnonymizedCurrentState(BaseModel):
    weight: float
    height: float
    age: int
    gender: str
    activity_level: str
    weekly_activity_days: int
    sleep_hours: float
    water_liters: float
    bmi: float
    bmi_classification: str


class AnonymizedTargetState(BaseModel):
    weight: float
    activity_level: str
    primary_goal: str
    target_weekly_days: int
    baseline_weight: float


class AnonymizedFitnessAssessment(BaseModel):
    fitness_level: str
    session_duration: str
    endurance_minutes: int
    pushups_count: int
    squats_count: int


class AnonymizedWeightHistoryPoint(BaseModel):
    weight_kg: float
    recorded_at: str


class AnonymizedActivityHistoryPoint(BaseModel):
    exercise_type: str
    duration_minutes: int
    intensity: str
    recorded_at: str


class AnonymizedUserMetrics(BaseModel):
    current_state: AnonymizedCurrentState
    target_state: AnonymizedTargetState
    preferences: List[str]
    restrictions: List[str]
    medical_conditions: Optional[str] = ""
    fitness_assessment: AnonymizedFitnessAssessment
    weight_history: List[AnonymizedWeightHistoryPoint] = Field(default_factory=list)
    activity_history: List[AnonymizedActivityHistoryPoint] = Field(default_factory=list)


class AnonymizedAIAnalysisPayload(BaseModel):
    subject_id: str
    regenerate_insights: Optional[bool] = False
    user_metrics: AnonymizedUserMetrics

#OUTBOUND RESPONSE SCHEMAS

class CamelCaseBaseModel(BaseModel):
    """Base model that automatically serializes to camelCase for Go consumption."""
    model_config = ConfigDict(
        alias_generator=to_camel,
        populate_by_name=True,
        from_attributes=True
    )


class WellnessMilestone(CamelCaseBaseModel):
    id: str
    category: str
    title: str
    description: str
    achieved: bool
    progress_pct: float  # Automatically converts to 'progressPct'


class WellnessCalculationResult(CamelCaseBaseModel):
    bmi: float
    bmi_classification: str  # Automatically converts to 'bmiClassification'
    wellness_score: float     # Automatically converts to 'wellnessScore'
    bmi_score: float          # Automatically converts to 'bmiScore'
    activity_score: float     # Automatically converts to 'activityScore'
    progress_score: float     # Automatically converts to 'progressScore'
    habits_score: float       # Automatically converts to 'habitsScore'
    milestones: List[WellnessMilestone]


class PrioritizedAIRecommendation(CamelCaseBaseModel):
    title: str
    category: str
    priority: str
    action: str
    expected_impact: str     # Automatically converts to 'expectedImpact'


class PyAIInsightsPayload(CamelCaseBaseModel):
    prompt_strategy: str          # Automatically converts to 'promptStrategy'
    health_status_analysis: str   # Automatically converts to 'healthStatusAnalysis'
    progress_evaluation: str      # Automatically converts to 'progressEvaluation'
    recommendations: List[PrioritizedAIRecommendation]
    weekly_summary: str           # Automatically converts to 'weeklySummary'
    monthly_summary: str          # Automatically converts to 'monthlySummary'


class PyAIAnalysisResponse(CamelCaseBaseModel):
    wellness: WellnessCalculationResult
    insights: PyAIInsightsPayload
