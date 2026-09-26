"""Loading base models and tokenizers for training and merging."""

from __future__ import annotations

from pathlib import Path
from typing import Any

from .config import PipelineConfig


class HardwareError(RuntimeError):
    """The machine cannot run this stage."""


def require_cuda() -> Any:
    import torch

    if not torch.cuda.is_available():
        raise HardwareError("training needs a CUDA GPU; none is visible to PyTorch")
    return torch


def compute_dtype() -> Any:
    torch = require_cuda()
    return torch.bfloat16 if torch.cuda.is_bf16_supported() else torch.float16


def load_tokenizer(source: str | Path, config: PipelineConfig, revision: str | None = None) -> Any:
    from transformers import AutoTokenizer

    tokenizer = AutoTokenizer.from_pretrained(
        str(source),
        revision=revision,
        trust_remote_code=config.trust_remote_code,
    )
    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token
    if tokenizer.chat_template is None:
        raise HardwareError(f"{source} has no chat template; choose an instruction-tuned model")
    return tokenizer


def load_causal_lm(
    source: str | Path,
    config: PipelineConfig,
    *,
    revision: str | None = None,
    quantize: bool = False,
) -> Any:
    from transformers import AutoModelForCausalLM, BitsAndBytesConfig

    dtype = compute_dtype()
    kwargs: dict[str, Any] = {
        "revision": revision,
        "trust_remote_code": config.trust_remote_code,
        "dtype": dtype,
    }
    if quantize:
        kwargs["quantization_config"] = BitsAndBytesConfig(
            load_in_4bit=True,
            bnb_4bit_quant_type="nf4",
            bnb_4bit_compute_dtype=dtype,
            bnb_4bit_use_double_quant=True,
        )
        kwargs["device_map"] = {"": 0}
    model = AutoModelForCausalLM.from_pretrained(str(source), **kwargs)
    model.config.use_cache = False
    if quantize:
        from peft import prepare_model_for_kbit_training

        model = prepare_model_for_kbit_training(
            model,
            use_gradient_checkpointing=config.sft.gradient_checkpointing,
        )
    return model


def conversation_tokens(tokenizer: Any, messages: list) -> int:
    return len(tokenizer.apply_chat_template(messages, tokenize=True, return_dict=False))
