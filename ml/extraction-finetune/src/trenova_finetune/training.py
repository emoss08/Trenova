"""Supervised fine-tuning and preference training with LoRA adapters."""

from __future__ import annotations

import logging
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from . import dataset
from .config import PipelineConfig
from .lengths import enforce_budget, partition_by_length
from .models import compute_dtype, conversation_tokens, load_causal_lm, load_tokenizer

log = logging.getLogger(__name__)


@dataclass(frozen=True)
class StageResult:
    output: Path | None
    metrics: dict[str, Any]
    note: str | None = None
    skipped: bool = False


def lora_config(config: PipelineConfig) -> Any:
    from peft import LoraConfig

    return LoraConfig(
        r=config.lora.r,
        lora_alpha=config.lora.alpha,
        lora_dropout=config.lora.dropout,
        target_modules=list(config.lora.target_modules),
        bias="none",
        task_type="CAUSAL_LM",
    )


def precision_flags() -> dict[str, bool]:
    import torch

    bf16 = compute_dtype() == torch.bfloat16
    return {"bf16": bf16, "fp16": not bf16}


def last_metrics(history: list[dict[str, Any]]) -> dict[str, Any]:
    metrics: dict[str, Any] = {}
    for entry in history:
        for key, value in entry.items():
            if isinstance(value, int | float) and not isinstance(value, bool):
                metrics[key] = value
    return metrics


def train_sft(config: PipelineConfig, dataset_dir: Path, output: Path) -> StageResult:
    from datasets import Dataset
    from trl import SFTConfig, SFTTrainer

    tokenizer = load_tokenizer(config.base_model, config, config.revision)
    train, validation = dataset.load_sft(dataset_dir)

    def tokens(record: dict) -> int:
        return conversation_tokens(tokenizer, record["prompt"] + record["completion"])

    fitted_train = partition_by_length(train, tokens, config.sft.max_length)
    enforce_budget(fitted_train, config.sft.max_dropped_fraction, "training")
    fitted_validation = partition_by_length(validation, tokens, config.sft.max_length)
    log.info(
        "SFT: %d training and %d validation examples fit; %d and %d are longer than %d tokens",
        len(fitted_train.kept),
        len(fitted_validation.kept),
        fitted_train.dropped,
        fitted_validation.dropped,
        config.sft.max_length,
    )

    def as_dataset(records: list[dict]) -> Any:
        return Dataset.from_list(
            [{"prompt": r["prompt"], "completion": r["completion"]} for r in records]
        )

    has_validation = bool(fitted_validation.kept)
    args = SFTConfig(
        output_dir=str(output / "checkpoints"),
        num_train_epochs=config.sft.epochs,
        per_device_train_batch_size=config.sft.per_device_batch_size,
        per_device_eval_batch_size=config.sft.per_device_batch_size,
        gradient_accumulation_steps=config.sft.gradient_accumulation_steps,
        learning_rate=config.sft.learning_rate,
        lr_scheduler_type=config.sft.lr_scheduler,
        warmup_steps=config.sft.warmup_ratio,
        weight_decay=config.sft.weight_decay,
        logging_steps=config.sft.logging_steps,
        save_strategy="epoch",
        eval_strategy="epoch" if has_validation else "no",
        save_total_limit=config.sft.save_total_limit,
        gradient_checkpointing=config.sft.gradient_checkpointing,
        max_length=config.sft.max_length,
        completion_only_loss=True,
        packing=False,
        seed=config.seed,
        report_to="none",
        **precision_flags(),
    )
    trainer = SFTTrainer(
        model=load_causal_lm(
            config.base_model,
            config,
            revision=config.revision,
            quantize=config.quantization == "4bit",
        ),
        args=args,
        train_dataset=as_dataset(fitted_train.kept),
        eval_dataset=as_dataset(fitted_validation.kept) if has_validation else None,
        processing_class=tokenizer,
        peft_config=lora_config(config),
    )
    trainer.train()

    adapter = output / "adapter"
    trainer.save_model(str(adapter))
    tokenizer.save_pretrained(str(adapter))

    metrics = last_metrics(trainer.state.log_history)
    metrics.update(
        {
            "train_examples": len(fitted_train.kept),
            "validation_examples": len(fitted_validation.kept),
            "dropped_for_length": fitted_train.dropped + fitted_validation.dropped,
        }
    )
    return StageResult(output=adapter, metrics=metrics)


def train_dpo(
    config: PipelineConfig,
    dataset_dir: Path,
    model_dir: Path,
    output: Path,
) -> StageResult:
    pairs = dataset.load_preferences(dataset_dir)
    if len(pairs) < config.dpo.min_pairs:
        return StageResult(
            output=None,
            metrics={"pairs": len(pairs)},
            note=f"{len(pairs)} preference pair(s); at least {config.dpo.min_pairs} are required",
            skipped=True,
        )

    from datasets import Dataset
    from trl import DPOConfig, DPOTrainer

    tokenizer = load_tokenizer(model_dir, config)

    def tokens(record: dict) -> int:
        chosen = conversation_tokens(tokenizer, record["prompt"] + record["chosen"])
        rejected = conversation_tokens(tokenizer, record["prompt"] + record["rejected"])
        return max(chosen, rejected)

    fitted = partition_by_length(pairs, tokens, config.dpo.max_length)
    enforce_budget(fitted, config.sft.max_dropped_fraction, "preference")

    args = DPOConfig(
        output_dir=str(output / "checkpoints"),
        beta=config.dpo.beta,
        num_train_epochs=config.dpo.epochs,
        per_device_train_batch_size=config.dpo.per_device_batch_size,
        gradient_accumulation_steps=config.dpo.gradient_accumulation_steps,
        learning_rate=config.dpo.learning_rate,
        max_length=config.dpo.max_length,
        logging_steps=config.dpo.logging_steps,
        gradient_checkpointing=config.sft.gradient_checkpointing,
        save_strategy="no",
        seed=config.seed,
        report_to="none",
        **precision_flags(),
    )
    trainer = DPOTrainer(
        model=load_causal_lm(model_dir, config, quantize=config.quantization == "4bit"),
        ref_model=None,
        args=args,
        train_dataset=Dataset.from_list(
            [
                {"prompt": r["prompt"], "chosen": r["chosen"], "rejected": r["rejected"]}
                for r in fitted.kept
            ]
        ),
        processing_class=tokenizer,
        peft_config=lora_config(config),
    )
    trainer.train()

    adapter = output / "adapter"
    trainer.save_model(str(adapter))
    tokenizer.save_pretrained(str(adapter))

    metrics = last_metrics(trainer.state.log_history)
    metrics.update({"pairs": len(fitted.kept), "dropped_for_length": fitted.dropped})
    return StageResult(output=adapter, metrics=metrics)
