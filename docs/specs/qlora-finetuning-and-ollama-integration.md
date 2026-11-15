# Feature Spec: QLoRA Fine-Tuning, Clinical Dataset Generation, Guardrail Benchmark & Live Ollama Integration

- **Status:** Completed
- **Author:** Antigravity (Planning Mode)
- **Target Issue / PR:** N/A (Local Feature Spec)
- **Risk Level (`AGENTS.md` §10):** High (Modifies `compose.yaml`, `.env.example`, and introduces LLM fine-tuning/evaluation/serving workflows for `services/py-ai`)
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [ ] **Next.js Frontend (`/`)** — `app/`, `feature/`, `components/ui/`, `lib/`, `next.config.ts` (Not affected)
  - [ ] **Go Core API (`services/go-api/`)** — `handlers/`, `repositories/`, `models/`, `database/`, `auth/`, `health/` (Not affected)
  - [x] **Python Analytics & AI (`services/py-ai/`)** — `training/` (new), `test_training.py` (new), `compose.yaml`, `.env.example`

---

## 1. Objective & Scope

- **Problem Statement:**
  `services/py-ai` currently has a complete deterministic wellness engine (`analytics/wellness_engine.py`) and 3-layer guardrail pipeline (`pipeline/layer1_scispacy_extractor.py`, `layer2_prompt_constructor.py`, `layer3_llama_validator.py`), but lacks:
  1. A reproducible, zero-PII synthetic clinical instruction dataset grounded in the exact `wellness_engine` + Layer 1 + Layer 2 outputs.
  2. A QLoRA / LoRA fine-tuning script tailored for `meta-llama/Llama-3.2-1B-Instruct` with both CUDA 4-bit NF4 (`bitsandbytes`/`peft`/`trl`) and Apple Silicon M4 (`mps`/`mlx-lm`) support, isolated from production dependencies.
  3. An automated evaluation benchmark measuring schema validity, `primary_goal` alignment, and pre- vs. post-Layer-3 forbidden-term violation rates.
  4. Local Ollama serving configuration (`Modelfile`) and Docker Compose / environment wiring (`LLM_API_URL`, `LLM_MODEL`, `LLM_API_FORMAT`, `LLM_TIMEOUT_SECONDS`) so `services/py-ai` can call the fine-tuned `nutrisync-llama` model locally or in Docker.

- **In Scope:**
  1. **Synthetic Clinical Instruction Dataset Generator** (`services/py-ai/training/generate_dataset.py` + `services/py-ai/training/__init__.py`) generating deterministic, zero-PII `train.jsonl`, `valid.jsonl`, and `test.jsonl`.
  2. **QLoRA / LoRA Fine-Tuning Pipeline** (`services/py-ai/training/train_qlora.py` + `services/py-ai/training/requirements-train.txt`) with Pydantic v2 `QLoRATrainingConfig`, CUDA 4-bit NF4 `BitsAndBytesConfig`, Apple Silicon M4 `mps` / `mlx-lm` support, and `--dry-run` config verification mode.
  3. **Guardrail & Schema Evaluation Benchmark** (`services/py-ai/training/evaluate_qlora.py`) evaluating any model endpoint or adapter on `test.jsonl` and outputting a structured JSON/text report.
  4. **Local Apple M4 Ollama Serving & Docker Wiring**:
     - `services/py-ai/training/Modelfile` (`nutrisync-llama` based on `llama3.2:1b`)
     - `services/py-ai/training/README.md` (end-to-end dataset, training, GGUF/Ollama import, and evaluation guide)
     - `.env.example` and `compose.yaml` (`py-ai` service environment wiring)
     - `services/py-ai/test_training.py` (deterministic unit tests)

- **Out of Scope:**
  - Modifying `services/go-api/`, Next.js frontend, or PostgreSQL/GORM models.
  - Modifying `services/py-ai/requirements.txt` or `services/py-ai/Dockerfile` (production image stays slim and fast).
  - Altering the Wellness Score weights (`0.3 BMI + 0.3 Activity + 0.2 Progress + 0.2 Habits`) or breaking the Go <-> Python JSON contract.
  - Executing live GPU/M4 model training or downloading multi-GB HuggingFace weights during CI/unit tests.

- **Acceptance Criteria:**
  - [x] `generate_dataset.py` produces deterministic `train.jsonl`, `valid.jsonl`, and `test.jsonl` using real `wellness_engine.py`, Layer 1, and Layer 2 functions, with 0 residual PII tokens (`@`, phone numbers, URLs, SSN/MRN, titled names) in prompts/completions and 0% forbidden-term violations in ground-truth completions.
  - [x] Every ground-truth completion in `train.jsonl`, `valid.jsonl`, and `test.jsonl` validates strictly against `PyAIInsightsPayload`.
  - [x] `train_qlora.py` validates `QLoRATrainingConfig` (Pydantic v2), supports `--dry-run` without requiring `torch`/`peft`/`bitsandbytes`/`mlx-lm` to be installed, and documents both CUDA NF4 and Apple M4 (`mps` / `mlx-lm`) paths.
  - [x] `evaluate_qlora.py` computes and outputs a structured report with (a) JSON schema validity rate, (b) `primary_goal` alignment rate, (c) pre-Layer-3 forbidden-term violation rate, and (d) post-Layer-3 guardrail violation rate (`0.0%`).
  - [x] `services/py-ai/training/Modelfile`, `README.md`, `.env.example`, and `compose.yaml` accurately wire `LLM_API_URL`, `LLM_MODEL`, `LLM_API_FORMAT`, and `LLM_TIMEOUT_SECONDS` matching `LLMClientConfiguration.from_environment()` in `services/py-ai/pipeline/layer3_llama_validator.py`.
  - [x] `cd services/py-ai && .venv/bin/pytest -v` passes all existing 18 tests plus all new tests in `test_training.py`.

- **Pre-Flight MCP & Context Verification:**
  - [x] Verified `services/py-ai/schemas/analysis_schemas.py` (`AnonymizedUserMetrics`, `PyAIInsightsPayload`, `PrioritizedAIRecommendation`).
  - [x] Verified `services/py-ai/analytics/wellness_engine.py` (`calculate_deterministic_wellness_and_milestones`, `compact_historical_metrics_for_ai_context`).
  - [x] Verified `services/py-ai/pipeline/layer1_scispacy_extractor.py` (`extract_clinical_constraints_and_scrub_pii`).
  - [x] Verified `services/py-ai/pipeline/layer2_prompt_constructor.py` (`construct_llama_prompt_payload`, `_SYSTEM_PROMPT`).
  - [x] Verified `services/py-ai/pipeline/layer3_llama_validator.py` (`LLMClientConfiguration.from_environment`, `parse_llm_insights_json`, `find_matching_forbidden_term`, `filter_and_sanitize_insights_against_guardrails`).
  - [x] Verified `compose.yaml` and `.env.example` current contents.
  - [x] Verified baseline pytest suite (`18 passed` in `services/py-ai`).

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
- **Tables / Models Affected:** `None` (stateless `services/py-ai` training/evaluation tooling and Docker/env configuration only).

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints (`:8080`):** `None`.
- **Go <-> Python AI Contract Sync:** Preserved 100%. All synthetic payloads and ground-truth completions conform to `AnonymizedUserMetrics` and `PyAIInsightsPayload` in `services/py-ai/schemas/analysis_schemas.py`.

### 2.3 Python Analytics, Training, Evaluation & 3-Layer AI Pipeline (`services/py-ai/`)

#### Deliverable 1: Synthetic Clinical Instruction Dataset Generator (`services/py-ai/training/generate_dataset.py`)
- **Purpose:** Deterministically synthesize realistic, anonymized user profiles across a combinatorial clinical grid, run them through the real NutriSync analytics + Layer 1 + Layer 2 pipeline, and pair each prompt with a guardrail-verified `PyAIInsightsPayload` JSON completion.
- **Combinatorial Grid:**
  - `primary_goal`: `["fat_loss", "muscle_gain", "maintenance", "endurance", "general_wellness"]`
  - `bmi_classification` scenarios (via calibrated height/weight pairs): `"underweight"`, `"normal weight"`, `"overweight"`, `"obese"`
  - `trend_regime` (7d/30d weight & activity histories): `"improving"`, `"plateau"`, `"regressing"`, `"empty_history"`
  - `prompt_strategy`: `"few_shot"`, `"zero_shot"`
  - `constraint_profile` (restrictions + noisy free-text `medical_conditions` containing synthetic PII to exercise Layer 1 scrubbing):
    - `none` (no restrictions)
    - `peanut_allergy` (`restrictions=["peanut_allergy"]`, noisy text `"Peanut allergy. Contact Dr. Alice Vance at alice@example.com or 555-123-4567."`)
    - `type_2_diabetes` (`restrictions=["low_sugar"]`, noisy text `"Type 2 diabetes, MRN-998811, patient @john_doe."`)
    - `hypertension` (`restrictions=["low_sodium"]`, noisy text `"Hypertension and high blood pressure, call +1-800-555-0199."`)
    - `knee_injury` (`restrictions=[]`, noisy text `"Left knee injury and knee pain. https://clinic.example.org"`)
    - `back_pain_lactose` (`restrictions=["lactose"]`, noisy text `"Lower back pain and lactose intolerance."`)
    - `vegan_gluten` (`restrictions=["vegan", "gluten"]`, noisy text `"Celiac disease."`)
    - `shellfish_gout` (`restrictions=["shellfish", "gout"]`, noisy text `"History of gout and shellfish allergy."`)
- **Pipeline Execution Per Sample:**
  1. Build `AnonymizedUserMetrics` deterministically from a seeded `random.Random(seed)` + scenario parameters.
  2. Call `calculate_deterministic_wellness_and_milestones(user_metrics)` -> `WellnessCalculationResult`.
  3. Call `compact_historical_metrics_for_ai_context(weight_history, activity_history)` -> `compacted_trends`.
  4. Call `extract_clinical_constraints_and_scrub_pii(user_metrics)` -> `Layer1ClinicalExtractionResult`.
  5. Call `construct_llama_prompt_payload(user_metrics, wellness_result, compacted_trends, layer1_result, requested_strategy=strategy)` -> `ConstructedLlamaPrompt`.
  6. Generate a rich, scenario-specific `PyAIInsightsPayload` completion grounded in the exact deterministic `wellness_result` numbers, `compacted_trends` deltas/sessions, `primary_goal`, and `layer1_result` constraints (never referencing any term in `layer1_result.forbidden_terms`).
  7. Validate the completion with `PyAIInsightsPayload.model_validate(...)`, serialize with `model_dump_json(by_alias=False)` (matching Layer 2 `_SYSTEM_PROMPT` snake_case contract), and assert `find_matching_forbidden_term(completion_json, layer1_result.forbidden_terms) is None`.
- **Output Format (each `.jsonl` line in `train.jsonl`, `valid.jsonl`, `test.jsonl`):**
  - `messages`: `constructed_prompt.messages + [{"role": "assistant", "content": completion_json}]` (ChatML format for `trl.SFTTrainer` / `mlx-lm`)
  - `prompt`: `constructed_prompt.formatted_llama_prompt` (LLaMA-3 `<|begin_of_text|>...<|start_header_id|>assistant<|end_header_id|>\n`)
  - `completion`: `f"{completion_json}<|eot_id|>"`
  - `text`: `f"{constructed_prompt.formatted_llama_prompt}{completion_json}<|eot_id|>"`
  - `metadata`: `{"sample_id": ..., "split": ..., "primary_goal": ..., "bmi_classification": ..., "prompt_strategy": ..., "constraint_profile": ..., "forbidden_terms": layer1_result.forbidden_terms, "detected_pii_categories": layer1_result.detected_pii_categories}`
- **Deterministic Split & CLI:**
  - Seeded shuffle (`seed=42`) split `80% train / 10% valid / 10% test` (stratified or deterministic shuffled with configurable `--num-samples`, `--seed`, `--output-dir`).
  - Single-purpose `snake_case` functions (`build_synthetic_user_metrics_for_scenario`, `generate_safe_clinical_target_insights`, `generate_dataset_splits`, `write_jsonl_split_file`, `main`).

#### Deliverable 2: QLoRA / LoRA Fine-Tuning Script (`services/py-ai/training/train_qlora.py` + `services/py-ai/training/requirements-train.txt`)
- **`services/py-ai/training/requirements-train.txt`:**
  - Keeps heavy ML dependencies isolated from `services/py-ai/requirements.txt`:
    - `torch>=2.3.0`
    - `transformers>=4.44.0`
    - `peft>=0.12.0`
    - `trl>=0.9.6`
    - `datasets>=2.20.0`
    - `accelerate>=0.33.0`
    - `bitsandbytes>=0.43.0; platform_system == "Linux"`
    - `mlx-lm>=0.18.0; sys_platform == "darwin" and platform_machine == "arm64"`
- **`QLoRATrainingConfig` (Pydantic v2 `BaseModel`):**
  - `base_model: str = "meta-llama/Llama-3.2-1B-Instruct"`
  - `dataset_dir: str = "services/py-ai/training/data"`
  - `output_dir: str = "services/py-ai/training/outputs/nutrisync-llama-1b-qlora"`
  - `lora_r: int = 16` (rank $r \in [4, 128]$; low-rank decomposition $\Delta W = \frac{\alpha}{r} B A$)
  - `lora_alpha: int = 32` (scaling factor $\alpha / r = 2.0$)
  - `lora_dropout: float = 0.05`
  - `target_modules: List[str] = ["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"]`
  - `use_4bit_quantization: bool = True` (4-bit NF4 `BitsAndBytesConfig` with `bnb_4bit_use_double_quant=True`, `bnb_4bit_compute_dtype="bfloat16"` on CUDA)
  - `max_seq_length: int = 2048`
  - `num_train_epochs: int = 3`
  - `per_device_train_batch_size: int = 2`
  - `gradient_accumulation_steps: int = 8` (effective batch size = 16)
  - `learning_rate: float = 2e-4`
  - `warmup_ratio: float = 0.05`
  - `lr_scheduler_type: str = "cosine"`
  - `seed: int = 42`
  - `backend: Literal["auto", "cuda_qlora", "mps_lora", "mlx_lm"] = "auto"`
- **Backend Detection & Execution (`detect_training_backend`):**
  - Lazy-imports `torch` / `transformers` / `peft` / `trl` / `bitsandbytes` / `mlx_lm` only when executing training (never at module import time, so `--dry-run` and `pytest` run in <1s without GPU/ML packages).
  - `"cuda_qlora"`: `torch.cuda.is_available()` -> `BitsAndBytesConfig(load_in_4bit=True, bnb_4bit_quant_type="nf4", bnb_4bit_use_double_quant=True, bnb_4bit_compute_dtype=torch.bfloat16)` + `LoraConfig` + `SFTTrainer`.
  - `"mlx_lm"`: Apple Silicon (`darwin` + `arm64`) when `mlx_lm` is installed -> runs `mlx_lm.lora` adapter training using `train.jsonl` / `valid.jsonl`.
  - `"mps_lora"`: Apple Silicon `torch.backends.mps.is_available()` fallback using `float16` LoRA (`peft` + `trl`) without `bitsandbytes`.
- **CLI Flags:** `--dry-run` (validates `QLoRATrainingConfig`, verifies dataset files if present or prints resolved config & LoRA parameter/math summary, exits 0), `--backend`, `--base-model`, `--dataset-dir`, `--output-dir`, `--epochs`, `--lora-r`, `--lora-alpha`.

#### Deliverable 3: Guardrail & Schema Evaluation Benchmark (`services/py-ai/training/evaluate_qlora.py`)
- **Purpose:** Evaluate any live LLM HTTP endpoint (Ollama `/api/chat`, `/api/generate`, or OpenAI-compatible) or a mock/callable predictor against `test.jsonl` and produce a structured benchmark report.
- **Metrics Computed (`EvaluationBenchmarkReport` Pydantic v2 model):**
  1. `total_samples: int`
  2. `json_schema_validity_rate: float` — % of raw model responses that parse cleanly via `parse_llm_insights_json` (`PyAIInsightsPayload`).
  3. `primary_goal_alignment_rate: float` — % of valid responses whose recommendations/narratives align with the sample's `primary_goal` (checked via deterministic keyword/category alignment helper `check_primary_goal_alignment`).
  4. `pre_layer3_forbidden_term_violation_rate: float` — % of parsed responses containing at least one term from `metadata["forbidden_terms"]` *before* Layer 3 filtering.
  5. `post_layer3_forbidden_term_violation_rate: float` — % of responses containing any term from `metadata["forbidden_terms"]` *after* running `filter_and_sanitize_insights_against_guardrails` (must be `0.0%`).
  6. `fallback_rate: float` — % of samples where raw output was malformed/empty and required deterministic fallback.
- **CLI & API:**
  - `evaluate_predictions_against_test_records(records, raw_predictions)` (pure, deterministic function for unit testing and offline evaluation).
  - `run_live_endpoint_evaluation(...)` using `LLMClientConfiguration` + `httpx.Client`.
  - CLI flags: `--test-file`, `--api-url`, `--model`, `--api-format`, `--output-report`, `--limit`.

#### Deliverable 4: Local Apple M4 Ollama Serving & Docker Wiring
- **`services/py-ai/training/Modelfile`:**
  - `FROM llama3.2:1b`
  - Commented `ADAPTER ./outputs/nutrisync-llama-1b-qlora` (or merged GGUF path) instructions for importing the fine-tuned LoRA adapter into Ollama.
  - `PARAMETER temperature 0.2`
  - `PARAMETER top_p 0.9`
  - `PARAMETER num_ctx 4096`
  - `SYSTEM """..."""` matching `_SYSTEM_PROMPT` from `pipeline/layer2_prompt_constructor.py`.
- **`services/py-ai/training/README.md`:**
  - Step-by-step instructions for:
    1. Generating `train.jsonl` / `valid.jsonl` / `test.jsonl`
    2. Installing `training/requirements-train.txt` in a dedicated training venv
    3. Running `train_qlora.py` on Apple Silicon M4 (`mlx-lm` / `mps`) or CUDA GPU (`cuda_qlora`)
    4. Creating and serving `nutrisync-llama` in Ollama (`ollama create nutrisync-llama -f services/py-ai/training/Modelfile`)
    5. Running `evaluate_qlora.py` against `http://localhost:11434/api/chat`
    6. Connecting `services/py-ai` locally and via Docker Compose
- **`.env.example`:**
  - Add `# Python AI / Ollama LLM Configuration` section:
    ```env
    LLM_API_URL=http://localhost:11434/api/chat
    LLM_MODEL=nutrisync-llama
    LLM_API_FORMAT=ollama_chat
    LLM_TIMEOUT_SECONDS=15.0
    ```
- **`compose.yaml` (`py-ai` service):**
  - Add environment variables:
    ```yaml
    environment:
      GO_API_URL: ${GO_API_URL:-http://go-api:8080}
      LLM_API_URL: ${LLM_API_URL:-http://host.docker.internal:11434/api/chat}
      LLM_MODEL: ${LLM_MODEL:-nutrisync-llama}
      LLM_API_FORMAT: ${LLM_API_FORMAT:-ollama_chat}
      LLM_TIMEOUT_SECONDS: ${LLM_TIMEOUT_SECONDS:-15.0}
    extra_hosts:
      - "host.docker.internal:host-gateway"
    ```

### 2.4 Frontend UI & Validation (`app/`, `feature/`, `lib/schemas/`)
- Not affected.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **User Isolation & Authentication (`AGENTS.md` §6):**
  - `services/py-ai` remains stateless and receives only `AnonymizedAIAnalysisPayload`. No user IDs, emails, or tokens are used in training or evaluation.
- **GDPR & Two-Tier PII Anonymization (`AGENTS.md` §6):**
  - `generate_dataset.py` intentionally injects synthetic noisy PII (`@handles`, emails, phone numbers, MRNs, titled names, clinic URLs) into raw synthetic `medical_conditions` and passes them through `extract_clinical_constraints_and_scrub_pii` (Layer 1) before calling `construct_llama_prompt_payload` (Layer 2).
  - Every generated JSONL row is verified to contain zero raw PII tokens in `messages`, `prompt`, `completion`, and `text`.
- **Biometrics & AI Safety Guardrails (`AGENTS.md` §7):**
  - All wellness metrics in generated datasets are computed by `calculate_deterministic_wellness_and_milestones` (exact `0.3 BMI + 0.3 Activity + 0.2 Progress + 0.2 Habits` formula, clamped `[0.0, 100.0]`).
  - Every ground-truth completion is verified against `layer1_result.forbidden_terms` via `find_matching_forbidden_term` (`0.0%` violation rate).
  - `evaluate_qlora.py` explicitly verifies that post-Layer-3 guardrail filtering achieves `0.0%` forbidden-term violations even if a model emits unsafe pre-Layer-3 tokens.
- **Failure Modes & Safe Error Handling (`AGENTS.md` §9):**
  - When Ollama (`nutrisync-llama`) is offline or times out (`LLM_TIMEOUT_SECONDS`), `layer3_llama_validator.py` deterministically falls back to `build_deterministic_clinical_fallback_insights`.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Training Package & Requirements:** Create `services/py-ai/training/__init__.py` and `services/py-ai/training/requirements-train.txt` (keeping `services/py-ai/requirements.txt` untouched).
- [x] **Step 2 — Synthetic Dataset Generator:** Implement `services/py-ai/training/generate_dataset.py` calling `wellness_engine.py`, `layer1_scispacy_extractor.py`, and `layer2_prompt_constructor.py` to generate deterministic, zero-PII `train.jsonl`, `valid.jsonl`, and `test.jsonl`.
- [x] **Step 3 — QLoRA Fine-Tuning Script:** Implement `services/py-ai/training/train_qlora.py` with `QLoRATrainingConfig`, CUDA 4-bit NF4 `BitsAndBytesConfig`, Apple Silicon M4 (`mps`/`mlx-lm`) support, and `--dry-run` mode.
- [x] **Step 4 — Evaluation Benchmark:** Implement `services/py-ai/training/evaluate_qlora.py` measuring JSON schema validity, `primary_goal` alignment, pre-Layer-3 forbidden-term violation rate, and post-Layer-3 guardrail violation rate.
- [x] **Step 5 — Ollama Modelfile, README, `.env.example` & `compose.yaml`:** Create `services/py-ai/training/Modelfile` and `services/py-ai/training/README.md`, and wire `LLM_API_URL`, `LLM_MODEL`, `LLM_API_FORMAT`, and `LLM_TIMEOUT_SECONDS` into `.env.example` and `compose.yaml`.
- [x] **Step 6 — Deterministic Unit Tests (`services/py-ai/test_training.py`):** Add unit tests covering:
  - Deterministic dataset generation & train/valid/test split counts
  - Zero residual PII in generated prompts/completions
  - 0% forbidden-term violations & `PyAIInsightsPayload` schema validity on all generated ground-truth completions
  - `QLoRATrainingConfig` hyperparameter validation & `--dry-run` execution
  - `evaluate_qlora.py` benchmark metric math (schema validity, goal alignment, pre-Layer-3 vs. 0.0% post-Layer-3 violation rate)

### Verification Commands (`AGENTS.md` §4 — Only check off commands actually executed)
- [x] **Python AI Unit Tests:** `cd services/py-ai && .venv/bin/pytest -v`
- [x] **Dataset Generator CLI Smoke Check:** `cd services/py-ai && .venv/bin/python -m training.generate_dataset --num-samples 40 --output-dir /tmp/nutrisync_ds_check` (ran `--num-samples 320 --seed 42 --output-dir training/data`)
- [x] **QLoRA Dry-Run CLI Check:** `cd services/py-ai && .venv/bin/python -m training.train_qlora --dry-run`
- [x] **Docker Compose Config Validation:** `docker compose config -q`

---

## 5. Open Questions & Decisions Required

- [x] **Dataset Size & Git Tracking:** Gitignored `services/py-ai/training/data/*.jsonl` and `services/py-ai/training/outputs/` in `.gitignore` and generated initial 320-sample dataset in `services/py-ai/training/data/`.
- [x] **Authorization for `compose.yaml` & `.env.example`:** Explicitly authorized by user; updated `compose.yaml` and `.env.example`.
