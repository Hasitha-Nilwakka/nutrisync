# NutriSync QLoRA Fine-Tuning, Evaluation & Ollama Serving (`services/py-ai/training/`)

This directory contains the offline synthetic clinical dataset generator, QLoRA / Apple M4 LoRA training script, guardrail & schema evaluation benchmark, and Ollama `Modelfile` for `nutrisync-llama`.

> **Isolation Rule:** Heavy ML dependencies (`torch`, `transformers`, `peft`, `trl`, `datasets`, `bitsandbytes`, `mlx-lm`) live in [`requirements-train.txt`](./requirements-train.txt) so the production `services/py-ai/requirements.txt` and Docker image remain fast and lightweight.

---

## 1. Generate Deterministic, Zero-PII Datasets

Run from `services/py-ai/`:

```bash
.venv/bin/python -m training.generate_dataset --num-samples 320 --seed 42 --output-dir training/data
```

This generates `training/data/train.jsonl` (80%), `valid.jsonl` (10%), and `test.jsonl` (10%) using the real `wellness_engine.py`, Layer 1 PII scrubber/constraint extractor, and Layer 2 prompt constructor.

---

## 2. Install Training Dependencies (Dedicated Environment)

```bash
python3 -m venv .venv-train
source .venv-train/bin/activate
pip install -r services/py-ai/training/requirements-train.txt
```

Verify configuration and LoRA math without downloading weights:

```bash
cd services/py-ai && .venv/bin/python -m training.train_qlora --dry-run
```

---

## 3. Fine-Tune `Llama-3.2-1B-Instruct`

- **Apple Silicon (M4 `mlx-lm` or `mps`):**
  ```bash
  python -m training.train_qlora --backend mlx_lm --epochs 3
  ```
- **NVIDIA CUDA GPU (4-bit NF4 QLoRA with double quantization):**
  ```bash
  python -m training.train_qlora --backend cuda_qlora --epochs 3
  ```

---

## 4. Serve `nutrisync-llama` via Local Ollama

```bash
ollama pull llama3.2:1b
ollama create nutrisync-llama -f services/py-ai/training/Modelfile
ollama run nutrisync-llama
```

---

## 5. Run Guardrail & Schema Evaluation Benchmark

```bash
cd services/py-ai
.venv/bin/python -m training.evaluate_qlora \
  --test-file training/data/test.jsonl \
  --api-url http://localhost:11434/api/chat \
  --model nutrisync-llama \
  --api-format ollama_chat
```

The benchmark reports:
1. `json_schema_validity_rate` (`PyAIInsightsPayload` compliance)
2. `primary_goal_alignment_rate`
3. `pre_layer3_forbidden_term_violation_rate`
4. `post_layer3_forbidden_term_violation_rate` (enforced to `0.0%` by Layer 3)

---

## 6. Connect `services/py-ai`

- **Local dev:** Set `LLM_API_URL=http://localhost:11434/api/chat`, `LLM_MODEL=nutrisync-llama`, `LLM_API_FORMAT=ollama_chat`, `LLM_TIMEOUT_SECONDS=15.0` (see root `.env.example`).
- **Docker Compose:** `compose.yaml` defaults `LLM_API_URL` to `http://host.docker.internal:11434/api/chat` with `host.docker.internal:host-gateway` so the containerized `py-ai` service reaches host Ollama.
