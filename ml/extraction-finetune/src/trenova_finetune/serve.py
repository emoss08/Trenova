"""Serve a model this pipeline produced with vLLM's OpenAI-compatible server.

The command is built from the `serve` section of a config, so every setting that
changes speed is written down, reviewed, and measured with `bench` rather than
passed by hand. The API key is read by vLLM from `VLLM_API_KEY` and never appears
on the command line, where any user on the machine could read it.
"""

from __future__ import annotations

import json
import os
import shlex
import shutil
from collections.abc import Sequence
from pathlib import Path
from typing import Any

from .config import PipelineConfig, ServeSettings
from .pipeline import CARD_FILE, CARD_FORMAT

API_KEY_ENV = "VLLM_API_KEY"
VLLM_EXECUTABLE = "vllm"
RESERVED_FLAGS = ("--api-key",)


class ServeError(ValueError):
    """The model, the environment or the arguments cannot be served."""


def read_card(model_dir: Path) -> dict[str, Any]:
    """Return the model card, refusing a directory this pipeline did not produce."""
    path = model_dir / CARD_FILE
    try:
        card = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as err:
        raise ServeError(
            f"{model_dir} is not a model this pipeline produced (no {CARD_FILE})"
        ) from err
    except (OSError, json.JSONDecodeError) as err:
        raise ServeError(f"cannot read {path}: {err}") from err
    if not isinstance(card, dict) or card.get("format") != CARD_FORMAT:
        raise ServeError(f"{path} is not a {CARD_FORMAT} model card")
    return card


def serve_arguments(settings: ServeSettings) -> list[str]:
    """The `vllm serve` flags a serve section stands for."""
    args = [
        "--host",
        settings.host,
        "--port",
        str(settings.port),
        "--max-model-len",
        str(settings.max_model_len),
        "--tensor-parallel-size",
        str(settings.tensor_parallel_size),
        "--gpu-memory-utilization",
        str(settings.gpu_memory_utilization),
        _toggle("enable-prefix-caching", settings.enable_prefix_caching),
        _toggle("enable-chunked-prefill", settings.enable_chunked_prefill),
        "--kv-cache-dtype",
        settings.kv_cache_dtype,
        "--structured-outputs-config",
        json.dumps({"backend": settings.structured_outputs_backend}, separators=(",", ":")),
    ]
    if settings.max_num_seqs is not None:
        args += ["--max-num-seqs", str(settings.max_num_seqs)]
    if settings.max_num_batched_tokens is not None:
        args += ["--max-num-batched-tokens", str(settings.max_num_batched_tokens)]
    if settings.quantization is not None:
        args += ["--quantization", settings.quantization]
    if settings.speculative is not None:
        args += [
            "--speculative-config",
            json.dumps(settings.speculative.as_vllm(), separators=(",", ":"), sort_keys=True),
        ]
    if settings.report_cached_tokens:
        args.append("--enable-prompt-tokens-details")
    return args


def serve_command(
    config: PipelineConfig,
    model_dir: Path,
    served_name: str,
    extra: Sequence[str] = (),
) -> list[str]:
    """The full command line, with generation defaults left to the request as in production."""
    if not served_name.strip():
        raise ServeError("the served model name is empty")
    for flag in extra:
        if flag.split("=", 1)[0] in RESERVED_FLAGS:
            raise ServeError(f"pass the key in {API_KEY_ENV}, not {flag.split('=', 1)[0]}")
    command = [
        VLLM_EXECUTABLE,
        "serve",
        str(model_dir),
        "--served-model-name",
        served_name,
        "--generation-config",
        "vllm",
        "--seed",
        str(config.seed),
        *serve_arguments(config.serve),
    ]
    if config.trust_remote_code:
        command.append("--trust-remote-code")
    command.extend(extra)
    return command


def serve(
    config: PipelineConfig,
    model_dir: Path,
    served_name: str,
    extra: Sequence[str] = (),
    *,
    dry_run: bool = False,
) -> str:
    """Replace this process with `vllm serve`, or return the command when `dry_run` is set."""
    read_card(model_dir)
    command = serve_command(config, model_dir, served_name, extra)
    if dry_run:
        return shlex.join(command)
    if not os.environ.get(API_KEY_ENV):
        raise ServeError(f"set {API_KEY_ENV}; it is the key Trenova's provider sends")
    executable = shutil.which(VLLM_EXECUTABLE)
    if executable is None:
        raise ServeError("vllm is not installed; run uv sync --extra predict")
    os.execv(executable, command)


def _toggle(flag: str, enabled: bool) -> str:
    return f"--{flag}" if enabled else f"--no-{flag}"
