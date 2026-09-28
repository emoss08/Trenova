"""Compress a merged model with llm-compressor so vLLM serves it faster and in less memory.

Two schemes, each written in the compressed-tensors format vLLM loads without any flag:

- `fp8-dynamic`: 8-bit float weights, activations scaled per token at run time. It needs no
  calibration data and runs on GPUs with FP8 support (Ada, Hopper and later).
- `w4a16`: 4-bit weights by GPTQ, calibrated on the training data's own prompts and replies,
  so the error it minimizes is the error on extraction. It runs on any GPU vLLM supports.

Either can change what the model writes, so a quantized model is scored and benchmarked
against the model it came from before it is served.
"""

from __future__ import annotations

import random
import shutil
from dataclasses import dataclass
from importlib import metadata
from pathlib import Path
from typing import Any, Literal

from . import built
from .card import ModelCardError, read_card, save_card
from .config import PipelineConfig
from .models import load_causal_lm, load_tokenizer, require_cuda

SchemeName = Literal["fp8-dynamic", "w4a16"]


@dataclass(frozen=True)
class Scheme:
    name: SchemeName
    preset: str
    method: Literal["rtn", "gptq"]
    calibrated: bool


SCHEMES: dict[str, Scheme] = {
    "fp8-dynamic": Scheme("fp8-dynamic", "FP8_DYNAMIC", "rtn", calibrated=False),
    "w4a16": Scheme("w4a16", "W4A16", "gptq", calibrated=True),
}


class QuantizeError(ValueError):
    """The model or its calibration data cannot be quantized as asked."""


def scheme_for(name: str) -> Scheme:
    try:
        return SCHEMES[name]
    except KeyError as err:
        raise QuantizeError(
            f"unknown scheme {name!r}; choose one of {', '.join(sorted(SCHEMES))}"
        ) from err


def calibration_conversations(records: list[dict], count: int, seed: int) -> list[list[dict]]:
    """A reproducible sample of whole conversations: the prompt and the reply it should get."""
    if not records:
        raise QuantizeError("the training data holds no examples to calibrate on")
    chosen = records if len(records) <= count else random.Random(seed).sample(records, count)
    return [[*record["prompt"], *record["completion"]] for record in chosen]


def quantized_card(
    card: dict[str, Any],
    scheme: Scheme,
    source: Path,
    *,
    samples: int,
    max_seq_length: int,
    ignore: tuple[str, ...],
) -> dict[str, Any]:
    entry: dict[str, Any] = {
        "scheme": scheme.name,
        "preset": scheme.preset,
        "method": scheme.method,
        "format": "compressed-tensors",
        "ignore": list(ignore),
        "source": str(source),
        "llmcompressor": _version("llmcompressor"),
    }
    if scheme.calibrated:
        entry["calibrationSamples"] = samples
        entry["maxSeqLength"] = max_seq_length
    return {**card, "quantization": entry}


def check_source(model_dir: Path, output: Path) -> dict[str, Any]:
    """The source's card, after refusing work that would be wasted or destructive."""
    try:
        card = read_card(model_dir)
    except ModelCardError as err:
        raise QuantizeError(str(err)) from err
    if "quantization" in card:
        scheme = card["quantization"].get("scheme", "unknown")
        raise QuantizeError(f"{model_dir} is already quantized ({scheme}); quantize its source")
    if output.resolve() == model_dir.resolve():
        raise QuantizeError("write the quantized model to a new directory")
    if output.exists() and any(output.iterdir()):
        raise FileExistsError(f"{output} already holds files")
    return card


def quantize(
    config: PipelineConfig,
    model_dir: Path,
    scheme_name: str,
    output: Path,
    *,
    data_dir: Path | None = None,
) -> dict[str, Any]:
    scheme = scheme_for(scheme_name)
    card = check_source(model_dir, output)
    conversations: list[list[dict]] = []
    if scheme.calibrated:
        if data_dir is None:
            raise QuantizeError(f"{scheme.name} calibrates on training data; pass --data")
        built.verify(data_dir)
        train, _ = built.load_sft(data_dir)
        conversations = calibration_conversations(
            train, config.quantize.calibration_samples, config.seed
        )

    require_cuda()
    from llmcompressor import oneshot

    staging = output.with_name(f".{output.name}.partial")
    if staging.exists():
        shutil.rmtree(staging)

    tokenizer = load_tokenizer(model_dir, config)
    model = load_causal_lm(model_dir, config)
    recipe = _recipe(scheme, config.quantize.ignore)
    if conversations:
        oneshot(
            model=model,
            dataset=_calibration_dataset(tokenizer, conversations, config.calibration_length),
            recipe=recipe,
            max_seq_length=config.calibration_length,
            num_calibration_samples=len(conversations),
        )
    else:
        oneshot(model=model, recipe=recipe)

    model.config.use_cache = True
    model.save_pretrained(str(staging), save_compressed=True)
    tokenizer.save_pretrained(str(staging))
    save_card(
        staging,
        quantized_card(
            card,
            scheme,
            model_dir,
            samples=len(conversations),
            max_seq_length=config.calibration_length,
            ignore=config.quantize.ignore,
        ),
    )
    staging.replace(output)
    return {"model": str(output), "scheme": scheme.name, "calibrationSamples": len(conversations)}


def _recipe(scheme: Scheme, ignore: tuple[str, ...]) -> Any:
    if scheme.method == "gptq":
        from llmcompressor.modifiers.gptq import GPTQModifier

        return GPTQModifier(targets="Linear", scheme=scheme.preset, ignore=list(ignore))
    from llmcompressor.modifiers.quantization import QuantizationModifier

    return QuantizationModifier(targets="Linear", scheme=scheme.preset, ignore=list(ignore))


def _calibration_dataset(tokenizer: Any, conversations: list[list[dict]], length: int) -> Any:
    from datasets import Dataset

    texts = [
        tokenizer.apply_chat_template(conversation, tokenize=False)
        for conversation in conversations
    ]
    encoded = tokenizer(
        texts,
        padding=False,
        truncation=True,
        max_length=length,
        add_special_tokens=False,
    )
    return Dataset.from_dict(
        {"input_ids": encoded["input_ids"], "attention_mask": encoded["attention_mask"]}
    )


def _version(package: str) -> str | None:
    try:
        return metadata.version(package)
    except metadata.PackageNotFoundError:
        return None
