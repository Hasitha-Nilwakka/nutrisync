"""QLoRA / LoRA fine-tuning pipeline for NutriSync LLaMA-3.2-1B-Instruct.

Supports:
1. CUDA 4-bit NF4 QLoRA (BitsAndBytesConfig with double quantization + PEFT + TRL SFTTrainer).
2. Apple Silicon M4 local training via MLX (`mlx-lm`) or PyTorch MPS (`float16` LoRA).
3. Dependency-free `--dry-run` configuration & LoRA math verification for CI and pytest.

LoRA Decomposition Math:
    W' = W_0 + (lora_alpha / r) * (B @ A)
where W_0 in R^{d x k} is frozen (4-bit NF4 quantized on CUDA), A in R^{r x k},
B in R^{d x r}, and scaling = lora_alpha / r (default 32 / 16 = 2.0).
"""

import argparse
import importlib.util
import json
from pathlib import Path
import platform
import subprocess
import sys
from typing import Any, Dict, List, Literal, Optional, Sequence

from pydantic import BaseModel, Field, field_validator

TrainingBackend = Literal["auto", "cuda_qlora", "mps_lora", "mlx_lm"]
ResolvedBackend = Literal["cuda_qlora", "mps_lora", "mlx_lm", "cpu_dry_run"]


class QLoRATrainingConfig(BaseModel):
    """Pydantic v2 configuration for NutriSync QLoRA / LoRA fine-tuning."""

    base_model: str = "meta-llama/Llama-3.2-1B-Instruct"
    dataset_dir: str = "services/py-ai/training/data"
    output_dir: str = "services/py-ai/training/outputs/nutrisync-llama-1b-qlora"

    # LoRA low-rank adaptation hyperparameters: delta_W = (lora_alpha / lora_r) * (B @ A)
    lora_r: int = Field(default=16, ge=4, le=128)
    lora_alpha: int = Field(default=32, ge=4, le=256)
    lora_dropout: float = Field(default=0.05, ge=0.0, le=0.5)
    target_modules: List[str] = Field(
        default_factory=lambda: [
            "q_proj",
            "k_proj",
            "v_proj",
            "o_proj",
            "gate_proj",
            "up_proj",
            "down_proj",
        ]
    )

    # 4-bit NF4 quantization settings (used on CUDA via bitsandbytes)
    use_4bit_quantization: bool = True
    bnb_4bit_quant_type: Literal["nf4", "fp4"] = "nf4"
    bnb_4bit_use_double_quant: bool = True
    bnb_4bit_compute_dtype: Literal["bfloat16", "float16"] = "bfloat16"

    # Optimization & sequence settings
    max_seq_length: int = Field(default=2048, ge=256, le=8192)
    num_train_epochs: int = Field(default=3, ge=1, le=20)
    per_device_train_batch_size: int = Field(default=2, ge=1, le=64)
    gradient_accumulation_steps: int = Field(default=8, ge=1, le=64)
    learning_rate: float = Field(default=2e-4, gt=0.0, le=1e-2)
    warmup_ratio: float = Field(default=0.05, ge=0.0, le=0.5)
    lr_scheduler_type: str = "cosine"
    seed: int = 42
    backend: TrainingBackend = "auto"

    @field_validator("target_modules")
    @classmethod
    def validate_non_empty_target_modules(cls, value: List[str]) -> List[str]:
        """Ensure at least one transformer projection module is targeted by LoRA."""
        cleaned = [item.strip() for item in value if item and item.strip()]
        if not cleaned:
            raise ValueError("target_modules must contain at least one module name")
        return cleaned


def compute_lora_scaling_factor(config: QLoRATrainingConfig) -> float:
    """Compute the LoRA adapter scaling factor alpha / r."""
    return round(float(config.lora_alpha) / float(config.lora_r), 4)


def compute_effective_batch_size(config: QLoRATrainingConfig) -> int:
    """Compute the effective optimization batch size across gradient accumulation steps."""
    return int(config.per_device_train_batch_size) * int(config.gradient_accumulation_steps)


def _is_module_available(module_name: str) -> bool:
    """Check whether a Python module is importable without importing heavy weights."""
    return importlib.util.find_spec(module_name) is not None


def detect_training_backend(requested_backend: TrainingBackend = "auto") -> ResolvedBackend:
    """Resolve the active training backend for CUDA, Apple Silicon M4, or CPU dry-run."""
    if requested_backend != "auto":
        return requested_backend

    if _is_module_available("torch"):
        import torch  # noqa: WPS433

        if torch.cuda.is_available():
            return "cuda_qlora"

    is_apple_silicon = sys.platform == "darwin" and platform.machine() == "arm64"
    if is_apple_silicon and _is_module_available("mlx_lm"):
        return "mlx_lm"

    if _is_module_available("torch"):
        import torch  # noqa: WPS433

        if hasattr(torch.backends, "mps") and torch.backends.mps.is_available():
            return "mps_lora"

    return "cpu_dry_run"


def build_cuda_quantization_config(config: QLoRATrainingConfig) -> Any:
    """Build a 4-bit NF4 BitsAndBytesConfig for CUDA QLoRA training."""
    import torch  # noqa: WPS433
    from transformers import BitsAndBytesConfig  # noqa: WPS433

    compute_dtype = torch.bfloat16 if config.bnb_4bit_compute_dtype == "bfloat16" else torch.float16
    return BitsAndBytesConfig(
        load_in_4bit=config.use_4bit_quantization,
        bnb_4bit_quant_type=config.bnb_4bit_quant_type,
        bnb_4bit_use_double_quant=config.bnb_4bit_use_double_quant,
        bnb_4bit_compute_dtype=compute_dtype,
    )


def build_peft_lora_config(config: QLoRATrainingConfig) -> Any:
    """Build a PEFT LoraConfig targeting all attention and MLP projection layers."""
    from peft import LoraConfig  # noqa: WPS433

    return LoraConfig(
        r=config.lora_r,
        lora_alpha=config.lora_alpha,
        lora_dropout=config.lora_dropout,
        target_modules=config.target_modules,
        bias="none",
        task_type="CAUSAL_LM",
    )


def build_dry_run_summary(
    config: QLoRATrainingConfig,
    resolved_backend: Optional[ResolvedBackend] = None,
) -> Dict[str, Any]:
    """Produce a deterministic summary of config, LoRA math, and dataset readiness."""
    backend = resolved_backend or detect_training_backend(config.backend)
    dataset_path = Path(config.dataset_dir)
    split_files = {
        split: (dataset_path / f"{split}.jsonl").exists()
        for split in ("train", "valid", "test")
    }
    return {
        "mode": "dry_run",
        "resolved_backend": backend,
        "base_model": config.base_model,
        "lora_math": {
            "rank_r": config.lora_r,
            "lora_alpha": config.lora_alpha,
            "scaling_factor": compute_lora_scaling_factor(config),
            "lora_dropout": config.lora_dropout,
            "target_modules": config.target_modules,
        },
        "quantization": {
            "enabled_on_cuda": config.use_4bit_quantization,
            "quant_type": config.bnb_4bit_quant_type,
            "double_quant": config.bnb_4bit_use_double_quant,
            "compute_dtype": config.bnb_4bit_compute_dtype,
        },
        "effective_batch_size": compute_effective_batch_size(config),
        "dataset_splits_present": split_files,
        "output_dir": config.output_dir,
    }


def run_mlx_lm_training(config: QLoRATrainingConfig) -> Dict[str, Any]:
    """Execute LoRA fine-tuning on Apple Silicon M4 using mlx_lm.lora."""
    if not _is_module_available("mlx_lm"):
        raise RuntimeError(
            "mlx-lm is not installed. Install training dependencies via "
            "`pip install -r services/py-ai/training/requirements-train.txt`."
        )

    dataset_path = Path(config.dataset_dir)
    train_file = dataset_path / "train.jsonl"
    valid_file = dataset_path / "valid.jsonl"
    if not train_file.exists() or not valid_file.exists():
        raise FileNotFoundError(
            f"Missing train.jsonl or valid.jsonl in {dataset_path}. Run generate_dataset.py first."
        )

    output_path = Path(config.output_dir)
    output_path.mkdir(parents=True, exist_ok=True)
    command = [
        sys.executable,
        "-m",
        "mlx_lm.lora",
        "--model",
        config.base_model,
        "--train",
        "--data",
        str(dataset_path),
        "--adapter-path",
        str(output_path),
        "--batch-size",
        str(config.per_device_train_batch_size),
        "--iters",
        str(max(100, config.num_train_epochs * 100)),
        "--learning-rate",
        str(config.learning_rate),
        "--seed",
        str(config.seed),
    ]
    subprocess.run(command, check=True)
    return {"status": "completed", "backend": "mlx_lm", "output_dir": str(output_path)}


def run_cuda_or_mps_training(
    config: QLoRATrainingConfig,
    backend: Literal["cuda_qlora", "mps_lora"],
) -> Dict[str, Any]:
    """Execute HuggingFace PEFT + TRL SFTTrainer fine-tuning on CUDA (4-bit) or MPS (float16)."""
    for required_pkg in ("torch", "transformers", "peft", "trl", "datasets"):
        if not _is_module_available(required_pkg):
            raise RuntimeError(
                f"Missing required training package '{required_pkg}'. Install via "
                "`pip install -r services/py-ai/training/requirements-train.txt`."
            )

    from datasets import load_dataset  # noqa: WPS433
    import torch  # noqa: WPS433
    from transformers import AutoModelForCausalLM, AutoTokenizer  # noqa: WPS433
    from trl import SFTConfig, SFTTrainer  # noqa: WPS433

    dataset_path = Path(config.dataset_dir)
    data_files = {
        "train": str(dataset_path / "train.jsonl"),
        "validation": str(dataset_path / "valid.jsonl"),
    }
    dataset = load_dataset("json", data_files=data_files)

    tokenizer = AutoTokenizer.from_pretrained(config.base_model, use_fast=True)
    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token

    model_kwargs: Dict[str, Any] = {"device_map": "auto"}
    if backend == "cuda_qlora" and config.use_4bit_quantization:
        model_kwargs["quantization_config"] = build_cuda_quantization_config(config)
    else:
        model_kwargs["torch_dtype"] = torch.float16

    model = AutoModelForCausalLM.from_pretrained(config.base_model, **model_kwargs)
    lora_config = build_peft_lora_config(config)

    output_path = Path(config.output_dir)
    output_path.mkdir(parents=True, exist_ok=True)
    training_args = SFTConfig(
        output_dir=str(output_path),
        num_train_epochs=config.num_train_epochs,
        per_device_train_batch_size=config.per_device_train_batch_size,
        gradient_accumulation_steps=config.gradient_accumulation_steps,
        learning_rate=config.learning_rate,
        warmup_ratio=config.warmup_ratio,
        lr_scheduler_type=config.lr_scheduler_type,
        max_seq_length=config.max_seq_length,
        dataset_text_field="text",
        bf16=(backend == "cuda_qlora" and config.bnb_4bit_compute_dtype == "bfloat16"),
        fp16=(backend == "mps_lora"),
        logging_steps=10,
        save_strategy="epoch",
        eval_strategy="epoch",
        seed=config.seed,
    )

    trainer = SFTTrainer(
        model=model,
        args=training_args,
        train_dataset=dataset["train"],
        eval_dataset=dataset["validation"],
        processing_class=tokenizer,
        peft_config=lora_config,
    )
    trainer.train()
    trainer.save_model(str(output_path))
    tokenizer.save_pretrained(str(output_path))
    return {"status": "completed", "backend": backend, "output_dir": str(output_path)}


def execute_training_pipeline(
    config: QLoRATrainingConfig,
    dry_run: bool = False,
) -> Dict[str, Any]:
    """Validate configuration and execute either dry-run verification or hardware training."""
    resolved_backend = detect_training_backend(config.backend)
    if dry_run or resolved_backend == "cpu_dry_run":
        return build_dry_run_summary(config=config, resolved_backend=resolved_backend)
    if resolved_backend == "mlx_lm":
        return run_mlx_lm_training(config)
    return run_cuda_or_mps_training(config=config, backend=resolved_backend)


def main(argv: Optional[Sequence[str]] = None) -> int:
    """CLI entrypoint for NutriSync QLoRA / LoRA fine-tuning."""
    parser = argparse.ArgumentParser(description="Fine-tune LLaMA-3.2-1B-Instruct with QLoRA/LoRA for NutriSync.")
    parser.add_argument("--base-model", default="meta-llama/Llama-3.2-1B-Instruct")
    parser.add_argument("--dataset-dir", default="services/py-ai/training/data")
    parser.add_argument("--output-dir", default="services/py-ai/training/outputs/nutrisync-llama-1b-qlora")
    parser.add_argument("--lora-r", type=int, default=16)
    parser.add_argument("--lora-alpha", type=int, default=32)
    parser.add_argument("--epochs", type=int, default=3)
    parser.add_argument(
        "--backend",
        choices=["auto", "cuda_qlora", "mps_lora", "mlx_lm"],
        default="auto",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Validate QLoRATrainingConfig, compute LoRA math, and exit without training.",
    )
    args = parser.parse_args(argv)

    config = QLoRATrainingConfig(
        base_model=args.base_model,
        dataset_dir=args.dataset_dir,
        output_dir=args.output_dir,
        lora_r=args.lora_r,
        lora_alpha=args.lora_alpha,
        num_train_epochs=args.epochs,
        backend=args.backend,
    )
    result = execute_training_pipeline(config=config, dry_run=args.dry_run)
    print(json.dumps(result, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
