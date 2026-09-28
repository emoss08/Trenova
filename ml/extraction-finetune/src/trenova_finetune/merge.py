"""Fold a LoRA adapter into its base model so vLLM can serve one set of weights."""

from __future__ import annotations

import shutil
from pathlib import Path

from .config import PipelineConfig
from .models import load_causal_lm, load_tokenizer


def merge_adapter(
    config: PipelineConfig,
    base: str | Path,
    adapter: Path,
    output: Path,
    *,
    revision: str | None = None,
) -> Path:
    from peft import PeftModel

    if output.exists() and any(output.iterdir()):
        raise FileExistsError(f"{output} already holds files")
    staging = output.with_name(f".{output.name}.partial")
    if staging.exists():
        shutil.rmtree(staging)

    model = load_causal_lm(base, config, revision=revision, quantize=False)
    model = PeftModel.from_pretrained(model, str(adapter)).merge_and_unload()
    model.config.use_cache = True
    model.save_pretrained(str(staging))
    load_tokenizer(adapter, config).save_pretrained(str(staging))

    staging.replace(output)
    return output
