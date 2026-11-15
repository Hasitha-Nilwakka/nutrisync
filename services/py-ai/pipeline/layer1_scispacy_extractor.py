"""Layer 1: CPU-local secondary PII scrubber and clinical/dietary constraint extractor.

Implements the second tier of NutriSync's Two-Tier PII Anonymization architecture
(ADR-0003) before any user-supplied free text reaches Layer 2 LLM prompt construction.
"""

from functools import lru_cache
import re
from typing import Dict, List, Optional, Tuple

from pydantic import BaseModel, Field
import spacy
from spacy.language import Language

from schemas.analysis_schemas import AnonymizedUserMetrics

_EMAIL_PATTERN = re.compile(r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b")
_URL_PATTERN = re.compile(r"\b(?:https?://|www\.)\S+\b", re.IGNORECASE)
_PHONE_PATTERN = re.compile(
    r"(?<!\w)(?:\+?\d{1,3}[\s.-]?)?(?:\(?\d{2,4}\)?[\s.-]?)\d{3,4}[\s.-]?\d{3,4}(?!\w)"
)
_SSN_OR_ID_PATTERN = re.compile(r"\b\d{3}-\d{2}-\d{4}\b|\b(?:MRN|ID|SSN)[:#\s-]*[A-Z0-9-]{4,}\b", re.IGNORECASE)
_HANDLE_PATTERN = re.compile(r"(?<!\w)@[A-Za-z0-9_]{2,30}\b")
_TITLED_NAME_PATTERN = re.compile(
    r"\b(?:Dr\.|Mr\.|Mrs\.|Ms\.|Patient|Name[:\s])\s*([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)\b"
)

_CLINICAL_ENTITY_PATTERNS: List[Dict[str, str]] = [
    {"label": "CONDITION", "pattern": "type 2 diabetes"},
    {"label": "CONDITION", "pattern": "type 1 diabetes"},
    {"label": "CONDITION", "pattern": "diabetes"},
    {"label": "CONDITION", "pattern": "hypertension"},
    {"label": "CONDITION", "pattern": "high blood pressure"},
    {"label": "CONDITION", "pattern": "celiac"},
    {"label": "CONDITION", "pattern": "lactose intolerance"},
    {"label": "CONDITION", "pattern": "lactose intolerant"},
    {"label": "CONDITION", "pattern": "peanut allergy"},
    {"label": "CONDITION", "pattern": "tree nut allergy"},
    {"label": "CONDITION", "pattern": "shellfish allergy"},
    {"label": "CONDITION", "pattern": "knee injury"},
    {"label": "CONDITION", "pattern": "knee pain"},
    {"label": "CONDITION", "pattern": "back pain"},
    {"label": "CONDITION", "pattern": "asthma"},
    {"label": "CONDITION", "pattern": "pregnancy"},
    {"label": "CONDITION", "pattern": "pregnant"},
    {"label": "CONDITION", "pattern": "heart disease"},
    {"label": "CONDITION", "pattern": "kidney disease"},
    {"label": "CONDITION", "pattern": "renal disease"},
    {"label": "CONDITION", "pattern": "gout"},
]

_RESTRICTION_TO_FORBIDDEN_DIETARY_TERMS: Dict[str, List[str]] = {
    "peanut": ["peanut", "peanuts", "peanut butter", "groundnut"],
    "tree_nut": ["almond", "almonds", "walnut", "walnuts", "cashew", "cashews", "pecan", "hazelnut", "pistachio", "tree nut"],
    "dairy": ["milk", "cheese", "yogurt", "butter", "cream", "whey", "casein", "lactose", "dairy"],
    "lactose": ["milk", "cheese", "yogurt", "butter", "cream", "whey", "lactose", "dairy"],
    "gluten": ["wheat", "barley", "rye", "bread", "pasta", "gluten", "seitan", "couscous"],
    "celiac": ["wheat", "barley", "rye", "bread", "pasta", "gluten", "seitan", "couscous"],
    "shellfish": ["shrimp", "crab", "lobster", "prawn", "oyster", "mussel", "clam", "scallop", "shellfish"],
    "soy": ["soy", "soya", "tofu", "tempeh", "edamame", "soy sauce"],
    "egg": ["egg", "eggs", "mayonnaise"],
    "vegetarian": ["beef", "pork", "chicken", "turkey", "lamb", "fish", "salmon", "tuna", "bacon", "steak", "poultry", "meat"],
    "vegan": [
        "beef", "pork", "chicken", "turkey", "lamb", "fish", "salmon", "tuna",
        "bacon", "steak", "poultry", "meat", "egg", "eggs", "milk", "cheese",
        "yogurt", "butter", "honey", "whey", "dairy",
    ],
    "halal": ["pork", "bacon", "ham", "alcohol", "wine", "beer"],
    "kosher": ["pork", "bacon", "ham", "shellfish", "shrimp", "crab", "lobster"],
    "low_sodium": ["high-sodium", "salty snacks", "cured meat", "soy sauce", "processed deli meat"],
    "low_sugar": ["sugary soda", "candy", "refined sugar", "pastry", "syrup"],
    "diabetes": ["sugary soda", "candy", "refined sugar", "pastry", "syrup"],
    "hypertension": ["high-sodium", "salty snacks", "cured meat", "soy sauce"],
    "gout": ["organ meat", "shellfish", "beer", "high-purine"],
}

_CONDITION_TO_FORBIDDEN_EXERCISE_TERMS: Dict[str, List[str]] = {
    "knee": ["deep squats", "box jumps", "plyometrics", "high-impact running", "sprinting", "jumping lunges"],
    "joint": ["box jumps", "plyometrics", "high-impact running"],
    "back": ["heavy deadlifts", "barbell squats", "high-impact twisting"],
    "hypertension": ["heavy isometric lifting", "max-effort breath-hold"],
    "asthma": ["cold-air sprinting", "unmonitored max-effort sprint"],
    "pregnant": ["contact sports", "supine crunches", "heavy breath-hold lifting", "hot yoga"],
    "pregnancy": ["contact sports", "supine crunches", "heavy breath-hold lifting", "hot yoga"],
    "heart": ["max-effort HIIT", "unsupervised sprinting"],
}


class Layer1ClinicalExtractionResult(BaseModel):
    """Structured output of Layer 1 PII scrubbing and clinical constraint extraction."""

    scrubbed_medical_conditions: str = ""
    detected_pii_categories: List[str] = Field(default_factory=list)
    normalized_restrictions: List[str] = Field(default_factory=list)
    extracted_conditions: List[str] = Field(default_factory=list)
    forbidden_terms: List[str] = Field(default_factory=list)
    dietary_constraints: List[str] = Field(default_factory=list)
    exercise_constraints: List[str] = Field(default_factory=list)


@lru_cache(maxsize=1)
def load_spacy_nlp_pipeline() -> Language:
    """Load an available spaCy/SciSpacy model or fall back to CPU-local blank English + EntityRuler."""
    for model_name in ("en_core_sci_sm", "en_core_web_sm"):
        try:
            nlp = spacy.load(model_name)
            break
        except OSError:
            continue
    else:
        nlp = spacy.blank("en")

    if "entity_ruler" not in nlp.pipe_names:
        ruler = nlp.add_pipe("entity_ruler")
        ruler.add_patterns(_CLINICAL_ENTITY_PATTERNS)  # type: ignore[attr-defined]
    return nlp


def scrub_residual_pii_from_free_text(
    raw_text: str,
    nlp: Optional[Language] = None,
) -> Tuple[str, List[str]]:
    """Scrub residual PII (emails, phones, URLs, IDs, handles, and names) from free text."""
    cleaned = (raw_text or "").strip()
    if not cleaned:
        return "", []

    detected_categories: set[str] = set()

    if _EMAIL_PATTERN.search(cleaned):
        cleaned = _EMAIL_PATTERN.sub("[REDACTED_EMAIL]", cleaned)
        detected_categories.add("email")

    if _URL_PATTERN.search(cleaned):
        cleaned = _URL_PATTERN.sub("[REDACTED_URL]", cleaned)
        detected_categories.add("url")

    if _SSN_OR_ID_PATTERN.search(cleaned):
        cleaned = _SSN_OR_ID_PATTERN.sub("[REDACTED_ID]", cleaned)
        detected_categories.add("identifier")

    if _PHONE_PATTERN.search(cleaned):
        cleaned = _PHONE_PATTERN.sub("[REDACTED_PHONE]", cleaned)
        detected_categories.add("phone")

    if _HANDLE_PATTERN.search(cleaned):
        cleaned = _HANDLE_PATTERN.sub("[REDACTED_HANDLE]", cleaned)
        detected_categories.add("handle")

    if _TITLED_NAME_PATTERN.search(cleaned):
        cleaned = _TITLED_NAME_PATTERN.sub(
            lambda match: match.group(0).replace(match.group(1), "[REDACTED_NAME]"),
            cleaned,
        )
        detected_categories.add("name")

    pipeline = nlp if nlp is not None else load_spacy_nlp_pipeline()
    if "ner" in pipeline.pipe_names:
        doc = pipeline(cleaned)
        person_spans = [ent for ent in doc.ents if ent.label_ == "PERSON"]
        for span in reversed(person_spans):
            cleaned = cleaned[: span.start_char] + "[REDACTED_NAME]" + cleaned[span.end_char :]
            detected_categories.add("name")

    return cleaned, sorted(detected_categories)


def _normalize_token(raw_token: str) -> str:
    """Normalize a restriction or condition token into lowercase underscore form."""
    return re.sub(r"[^a-z0-9]+", "_", raw_token.strip().lower()).strip("_")


def build_forbidden_terms_from_restrictions_and_conditions(
    restrictions: List[str],
    scrubbed_medical_conditions: str,
    nlp: Optional[Language] = None,
) -> Tuple[List[str], List[str], List[str], List[str], List[str]]:
    """Map restrictions and scrubbed medical conditions into deterministic forbidden terms."""
    normalized_restrictions: List[str] = []
    extracted_conditions: List[str] = []
    forbidden_terms: set[str] = set()
    dietary_constraints: set[str] = set()
    exercise_constraints: set[str] = set()

    for raw_restriction in restrictions:
        normalized = _normalize_token(raw_restriction)
        if not normalized:
            continue
        normalized_restrictions.append(normalized)

        matched_known_rule = False
        for key, terms in _RESTRICTION_TO_FORBIDDEN_DIETARY_TERMS.items():
            if key in normalized:
                forbidden_terms.update(terms)
                dietary_constraints.add(key)
                matched_known_rule = True

        for key, terms in _CONDITION_TO_FORBIDDEN_EXERCISE_TERMS.items():
            if key in normalized:
                forbidden_terms.update(terms)
                exercise_constraints.add(key)
                matched_known_rule = True

        if not matched_known_rule:
            readable_term = normalized.replace("_allergy", "").replace("_intolerance", "").replace("_", " ").strip()
            if readable_term:
                forbidden_terms.add(readable_term)
                dietary_constraints.add(normalized)

    pipeline = nlp if nlp is not None else load_spacy_nlp_pipeline()
    if scrubbed_medical_conditions:
        doc = pipeline(scrubbed_medical_conditions.lower())
        for ent in doc.ents:
            if ent.label_ == "CONDITION":
                extracted_conditions.append(ent.text.strip())

        lowered_text = scrubbed_medical_conditions.lower()
        for key, terms in _RESTRICTION_TO_FORBIDDEN_DIETARY_TERMS.items():
            if key.replace("_", " ") in lowered_text:
                forbidden_terms.update(terms)
                dietary_constraints.add(key)
                if key.replace("_", " ") not in extracted_conditions:
                    extracted_conditions.append(key.replace("_", " "))

        for key, terms in _CONDITION_TO_FORBIDDEN_EXERCISE_TERMS.items():
            if key.replace("_", " ") in lowered_text:
                forbidden_terms.update(terms)
                exercise_constraints.add(key)
                if key.replace("_", " ") not in extracted_conditions:
                    extracted_conditions.append(key.replace("_", " "))

    return (
        sorted(set(normalized_restrictions)),
        sorted(set(extracted_conditions)),
        sorted(forbidden_terms),
        sorted(dietary_constraints),
        sorted(exercise_constraints),
    )


def extract_clinical_constraints_and_scrub_pii(
    user_metrics: AnonymizedUserMetrics,
) -> Layer1ClinicalExtractionResult:
    """Run Layer 1 secondary PII scrubbing and clinical/dietary forbidden-term extraction."""
    nlp = load_spacy_nlp_pipeline()
    scrubbed_conditions, pii_categories = scrub_residual_pii_from_free_text(
        user_metrics.medical_conditions or "",
        nlp=nlp,
    )
    (
        normalized_restrictions,
        extracted_conditions,
        forbidden_terms,
        dietary_constraints,
        exercise_constraints,
    ) = build_forbidden_terms_from_restrictions_and_conditions(
        restrictions=user_metrics.restrictions,
        scrubbed_medical_conditions=scrubbed_conditions,
        nlp=nlp,
    )
    return Layer1ClinicalExtractionResult(
        scrubbed_medical_conditions=scrubbed_conditions,
        detected_pii_categories=pii_categories,
        normalized_restrictions=normalized_restrictions,
        extracted_conditions=extracted_conditions,
        forbidden_terms=forbidden_terms,
        dietary_constraints=dietary_constraints,
        exercise_constraints=exercise_constraints,
    )
