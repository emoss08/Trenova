"""Pipeline configuration, validated before any GPU work starts."""

from __future__ import annotations

from pathlib import Path
from typing import Literal

import yaml
from pydantic import BaseModel, ConfigDict, Field, ValidationError, model_validator

DEFAULT_TARGET_MODULES = (
    "q_proj",
    "k_proj",
    "v_proj",
    "o_proj",
    "gate_proj",
    "up_proj",
    "down_proj",
)


class ConfigError(ValueError):
    """The configuration file is missing, unreadable or invalid."""


class _Settings(BaseModel):
    model_config = ConfigDict(extra="forbid", frozen=True)


class LoraSettings(_Settings):
    r: int = Field(16, ge=1, le=512)
    alpha: int = Field(32, ge=1)
    dropout: float = Field(0.05, ge=0.0, lt=1.0)
    target_modules: tuple[str, ...] = Field(DEFAULT_TARGET_MODULES, min_length=1)


class SFTSettings(_Settings):
    learning_rate: float = Field(2e-4, gt=0.0, lt=1.0)
    epochs: float = Field(2.0, gt=0.0, le=50.0)
    per_device_batch_size: int = Field(1, ge=1)
    gradient_accumulation_steps: int = Field(16, ge=1)
    max_length: int = Field(8192, ge=512)
    warmup_ratio: float = Field(0.03, ge=0.0, lt=1.0)
    lr_scheduler: Literal["cosine", "linear", "constant", "constant_with_warmup"] = "cosine"
    weight_decay: float = Field(0.0, ge=0.0)
    logging_steps: int = Field(10, ge=1)
    save_total_limit: int = Field(2, ge=1)
    gradient_checkpointing: bool = True
    max_dropped_fraction: float = Field(0.02, ge=0.0, le=1.0)


class DPOSettings(_Settings):
    enabled: bool = True
    beta: float = Field(0.1, gt=0.0)
    learning_rate: float = Field(5e-6, gt=0.0, lt=1.0)
    epochs: float = Field(1.0, gt=0.0, le=20.0)
    per_device_batch_size: int = Field(1, ge=1)
    gradient_accumulation_steps: int = Field(16, ge=1)
    max_length: int = Field(8192, ge=512)
    logging_steps: int = Field(10, ge=1)
    min_pairs: int = Field(50, ge=1)


class TargetSettings(_Settings):
    """The recipe that turns a confirmed answer into the reply the model is trained to give."""

    keep_unverified: bool = True
    verified_confidence: float = Field(0.95, ge=0.0, le=1.0)
    unverified_confidence: float = Field(0.7, ge=0.0, le=1.0)
    overall_confidence: float = Field(0.9, ge=0.0, le=1.0)
    review_status: Literal["Ready", "NeedsReview"] = "Ready"
    source: str = Field("ai", min_length=1, max_length=32)
    default_document_kind: str = Field("RateConfirmation", min_length=1)
    evidence_context_chars: int = Field(60, ge=0, le=500)
    preference_outcomes: tuple[Literal["Corrected", "Missed", "Unconfirmed"], ...] = (
        "Corrected",
        "Missed",
    )


class PredictSettings(_Settings):
    max_model_len: int = Field(16384, ge=1024)
    max_tokens: int = Field(2048, ge=128)
    tensor_parallel_size: int = Field(1, ge=1)
    gpu_memory_utilization: float = Field(0.9, gt=0.0, le=1.0)


class PipelineConfig(_Settings):
    base_model: str = Field(min_length=1)
    revision: str | None = None
    trust_remote_code: bool = False
    quantization: Literal["none", "4bit"] = "none"
    seed: int = 42
    lora: LoraSettings = LoraSettings()
    targets: TargetSettings = TargetSettings()
    sft: SFTSettings = SFTSettings()
    dpo: DPOSettings = DPOSettings()
    predict: PredictSettings = PredictSettings()

    @model_validator(mode="after")
    def _prediction_fits_training(self) -> PipelineConfig:
        if self.predict.max_model_len < self.sft.max_length:
            raise ValueError("predict.max_model_len must be at least sft.max_length")
        return self


def load_config(path: str | Path) -> PipelineConfig:
    """Read and validate a YAML pipeline configuration."""
    source = Path(path)
    try:
        raw = yaml.safe_load(source.read_text(encoding="utf-8"))
    except OSError as err:
        raise ConfigError(f"cannot read {source}: {err}") from err
    except yaml.YAMLError as err:
        raise ConfigError(f"{source} is not valid YAML: {err}") from err
    if not isinstance(raw, dict):
        raise ConfigError(f"{source} must hold a mapping of settings")
    try:
        return PipelineConfig.model_validate(raw)
    except ValidationError as err:
        raise ConfigError(f"{source} is invalid:\n{err}") from err
