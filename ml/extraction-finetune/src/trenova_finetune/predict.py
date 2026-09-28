"""Answer the validation prompts with vLLM, asking for JSON the way Trenova will."""

from __future__ import annotations

from pathlib import Path
from typing import Any, Literal

from . import dataset
from .config import PipelineConfig
from .files import write_jsonl

StructuredKind = Literal["json_schema", "json_object", "none"]

ERROR_TRUNCATED = "the reply was cut off at max_tokens"
ERROR_EMPTY = "the model returned no text"


def structured_kind(mode: str) -> StructuredKind:
    """How a provider registered with this structured output mode is asked for JSON."""
    if mode == "JSONSchema":
        return "json_schema"
    if mode == "JSONMode":
        return "json_object"
    if mode == "Prompted":
        return "none"
    raise dataset.DatasetError(f"unknown structured output mode {mode!r}")


def sampling_settings(manifest: dataset.DatasetManifest, config: PipelineConfig) -> dict[str, Any]:
    """The sampling Trenova uses for document extraction, read from the dataset."""
    settings: dict[str, Any] = {"max_tokens": config.predict.max_tokens, "seed": config.seed}
    if manifest.temperature is not None:
        settings["temperature"] = manifest.temperature
    if manifest.top_p is not None:
        settings["top_p"] = manifest.top_p
    return settings


def prediction_record(identifier: str, output: Any) -> dict[str, str]:
    """Turn one vLLM request output into a line the scorer reads."""
    completions = getattr(output, "outputs", None) or []
    if not completions:
        return {"id": identifier, "reply": "", "error": ERROR_EMPTY}
    completion = completions[0]
    text = (completion.text or "").strip()
    if getattr(completion, "finish_reason", None) == "length":
        return {"id": identifier, "reply": text, "error": ERROR_TRUNCATED}
    if not text:
        return {"id": identifier, "reply": "", "error": ERROR_EMPTY}
    return {"id": identifier, "reply": text}


def _structured_params(kind: StructuredKind, schema: dict[str, Any]) -> dict[str, Any]:
    if kind == "none":
        return {}
    from vllm.sampling_params import StructuredOutputsParams

    if kind == "json_schema":
        return {"structured_outputs": StructuredOutputsParams(json=schema)}
    return {"structured_outputs": StructuredOutputsParams(json_object=True)}


def write_predictions(path: Path, records: list[dict[str, str]]) -> None:
    write_jsonl(path, records)


def predict(
    config: PipelineConfig,
    model_dir: Path,
    dataset_dir: Path,
    output: Path,
) -> dict[str, Any]:
    from vllm import LLM, SamplingParams

    manifest = dataset.load_manifest(dataset_dir)
    prompts = dataset.load_evaluation_prompts(dataset_dir)
    kind = structured_kind(manifest.structured_output_mode)
    params = SamplingParams(
        **sampling_settings(manifest, config),
        **_structured_params(kind, dataset.load_schema(dataset_dir)),
    )

    llm = LLM(
        model=str(model_dir),
        max_model_len=config.predict.max_model_len,
        tensor_parallel_size=config.predict.tensor_parallel_size,
        gpu_memory_utilization=config.predict.gpu_memory_utilization,
        seed=config.seed,
        trust_remote_code=config.trust_remote_code,
    )
    outputs = llm.chat([prompt for _, prompt in prompts], params, use_tqdm=True)
    records = [
        prediction_record(identifier, result)
        for (identifier, _), result in zip(prompts, outputs, strict=True)
    ]
    write_predictions(output, records)

    failed = sum(1 for record in records if "error" in record)
    return {"predictions": len(records), "failed": failed, "structured_output": kind}
