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


PRODUCTION_MAX_TOKENS = 5000


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
    max_tokens: int = Field(PRODUCTION_MAX_TOKENS, ge=128)
    tensor_parallel_size: int = Field(1, ge=1)
    gpu_memory_utilization: float = Field(0.9, gt=0.0, le=1.0)


ServeQuantization = Literal["fp8", "awq", "awq_marlin", "gptq", "gptq_marlin", "compressed-tensors"]
KVCacheDType = Literal["auto", "fp8", "fp8_e4m3", "fp8_e5m2"]
StructuredOutputsBackend = Literal["auto", "xgrammar", "guidance", "outlines", "lm-format-enforcer"]
SpeculativeMethod = Literal["ngram", "draft_model", "eagle", "eagle3"]
DRAFTED_METHODS = ("draft_model", "eagle", "eagle3")


class SpeculativeSettings(_Settings):
    """Speculative decoding: a cheap proposer drafts tokens the model then checks in one pass.

    `ngram` drafts from the prompt itself, which suits extraction: most of a reply is copied
    from the document text. The others need a draft model or head trained for the base model.
    """

    method: SpeculativeMethod
    num_speculative_tokens: int = Field(ge=1, le=16)
    model: str | None = Field(None, min_length=1)
    prompt_lookup_min: int | None = Field(None, ge=1, le=64)
    prompt_lookup_max: int | None = Field(None, ge=1, le=64)

    @model_validator(mode="after")
    def _fits_method(self) -> SpeculativeSettings:
        if self.method in DRAFTED_METHODS and self.model is None:
            raise ValueError(f"speculative method {self.method} needs a draft model")
        if self.method == "ngram":
            if self.model is not None:
                raise ValueError("speculative method ngram drafts from the prompt; drop model")
            if self.prompt_lookup_min is None and self.prompt_lookup_max is None:
                raise ValueError("speculative method ngram needs prompt_lookup_min or _max")
        elif self.prompt_lookup_min is not None or self.prompt_lookup_max is not None:
            raise ValueError("prompt_lookup_min and prompt_lookup_max only apply to ngram")
        if (
            self.prompt_lookup_min is not None
            and self.prompt_lookup_max is not None
            and self.prompt_lookup_min > self.prompt_lookup_max
        ):
            raise ValueError("prompt_lookup_min must not exceed prompt_lookup_max")
        return self

    def as_vllm(self) -> dict[str, int | str]:
        return self.model_dump(exclude_none=True)


class ServeSettings(_Settings):
    """How `vllm serve` runs the model Trenova calls; each knob is benchmarked, not guessed."""

    host: str = Field("0.0.0.0", min_length=1)
    port: int = Field(8000, ge=1, le=65535)
    max_model_len: int = Field(16384, ge=1024)
    tensor_parallel_size: int = Field(1, ge=1)
    gpu_memory_utilization: float = Field(0.9, gt=0.0, le=1.0)
    enable_prefix_caching: bool = True
    enable_chunked_prefill: bool = True
    max_num_seqs: int | None = Field(None, ge=1, le=4096)
    max_num_batched_tokens: int | None = Field(None, ge=256)
    kv_cache_dtype: KVCacheDType = "auto"
    quantization: ServeQuantization | None = None
    structured_outputs_backend: StructuredOutputsBackend = "auto"
    speculative: SpeculativeSettings | None = None
    report_cached_tokens: bool = True

    @model_validator(mode="after")
    def _batch_holds_a_prompt(self) -> ServeSettings:
        if (
            not self.enable_chunked_prefill
            and self.max_num_batched_tokens is not None
            and self.max_num_batched_tokens < self.max_model_len
        ):
            raise ValueError(
                "without chunked prefill, max_num_batched_tokens must be at least max_model_len"
            )
        return self


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
    serve: ServeSettings = ServeSettings()

    @model_validator(mode="after")
    def _prediction_fits_training(self) -> PipelineConfig:
        if self.predict.max_model_len < self.sft.max_length:
            raise ValueError("predict.max_model_len must be at least sft.max_length")
        if self.serve.max_model_len < self.sft.max_length:
            raise ValueError("serve.max_model_len must be at least sft.max_length")
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
