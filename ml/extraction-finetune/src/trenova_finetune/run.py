"""The run record: what a model was trained from, how, and what each stage produced.

`run.json` is rewritten atomically after every stage, so a crash never leaves it
half-written and a resumed run can see exactly which stages finished.
"""

from __future__ import annotations

import json
import os
import platform
import tempfile
import time
from dataclasses import asdict, dataclass, field
from importlib import metadata
from pathlib import Path
from typing import Any

from . import __version__
from .config import PipelineConfig
from .dataset import DatasetManifest

RUN_FORMAT = "trenova.extraction-finetune-run/v1"
RUN_FILE = "run.json"
TRACKED_PACKAGES = (
    "torch",
    "transformers",
    "trl",
    "peft",
    "datasets",
    "accelerate",
    "bitsandbytes",
    "vllm",
)

STATUS_RUNNING = "running"
STATUS_COMPLETED = "completed"
STATUS_SKIPPED = "skipped"
STATUS_FAILED = "failed"


class RunError(RuntimeError):
    """The run directory cannot be used as asked."""


@dataclass
class StageRecord:
    status: str
    started_at: float
    finished_at: float | None = None
    output: str | None = None
    metrics: dict[str, Any] = field(default_factory=dict)
    note: str | None = None


@dataclass
class RunRecord:
    format: str
    pipeline_version: str
    created_at: float
    base_model: str
    revision: str | None
    config: dict[str, Any]
    dataset: dict[str, Any]
    environment: dict[str, Any]
    stages: dict[str, StageRecord] = field(default_factory=dict)
    final_model: str | None = None


def installed_versions() -> dict[str, str]:
    versions = {}
    for package in TRACKED_PACKAGES:
        try:
            versions[package] = metadata.version(package)
        except metadata.PackageNotFoundError:
            continue
    return versions


def new_record(config: PipelineConfig, dataset: DatasetManifest) -> RunRecord:
    return RunRecord(
        format=RUN_FORMAT,
        pipeline_version=__version__,
        created_at=time.time(),
        base_model=config.base_model,
        revision=config.revision,
        config=config.model_dump(mode="json"),
        dataset={
            "exportId": dataset.export_id,
            "exportManifestSha256": dataset.export_manifest_sha256,
            "datasetManifestSha256": dataset.sha256,
            "promptSha256": dataset.prompt_sha256,
            "structuredOutputMode": dataset.structured_output_mode,
            "schemaName": dataset.schema_name,
            "pageLimit": dataset.page_limit,
            "counts": dataset.counts,
        },
        environment={"python": platform.python_version(), "packages": installed_versions()},
    )


class Run:
    """A run directory and its record."""

    def __init__(self, directory: Path, record: RunRecord) -> None:
        self.directory = directory
        self.record = record

    @classmethod
    def create(
        cls,
        directory: str | Path,
        config: PipelineConfig,
        dataset: DatasetManifest,
    ) -> Run:
        root = Path(directory)
        if root.exists() and any(root.iterdir()):
            raise RunError(f"{root} already holds files; choose a new directory or pass --resume")
        root.mkdir(parents=True, exist_ok=True)
        run = cls(root, new_record(config, dataset))
        run.save()
        return run

    @classmethod
    def resume(
        cls,
        directory: str | Path,
        config: PipelineConfig,
        dataset: DatasetManifest,
    ) -> Run:
        root = Path(directory)
        path = root / RUN_FILE
        try:
            raw = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as err:
            raise RunError(f"cannot resume from {path}: {err}") from err
        if raw.get("format") != RUN_FORMAT:
            raise RunError(f"{path} is not a {RUN_FORMAT} record")

        stages = {name: StageRecord(**stage) for name, stage in raw.pop("stages", {}).items()}
        record = RunRecord(**raw, stages=stages)
        expected = new_record(config, dataset)
        if record.config != expected.config:
            raise RunError("the configuration differs from the one this run started with")
        if record.dataset["datasetManifestSha256"] != expected.dataset["datasetManifestSha256"]:
            raise RunError("the dataset differs from the one this run started with")
        return cls(root, record)

    def path(self, *parts: str) -> Path:
        return self.directory.joinpath(*parts)

    def completed(self, stage: str) -> bool:
        record = self.record.stages.get(stage)
        if record is None or record.status not in (STATUS_COMPLETED, STATUS_SKIPPED):
            return False
        return record.output is None or Path(record.output).exists()

    def start(self, stage: str) -> None:
        self.record.stages[stage] = StageRecord(status=STATUS_RUNNING, started_at=time.time())
        self.save()

    def finish(
        self,
        stage: str,
        output: Path | None,
        metrics: dict[str, Any] | None = None,
        note: str | None = None,
        status: str = STATUS_COMPLETED,
    ) -> None:
        record = self.record.stages[stage]
        record.status = status
        record.finished_at = time.time()
        record.output = str(output) if output is not None else None
        record.metrics = metrics or {}
        record.note = note
        self.record.environment["packages"] = installed_versions()
        self.save()

    def fail(self, stage: str, error: BaseException) -> None:
        record = self.record.stages.get(stage)
        if record is None:
            return
        record.status = STATUS_FAILED
        record.finished_at = time.time()
        record.note = f"{type(error).__name__}: {error}"
        self.save()

    def save(self) -> None:
        payload = json.dumps(asdict(self.record), indent=2, sort_keys=True)
        handle, temporary = tempfile.mkstemp(prefix=".run-", suffix=".json", dir=self.directory)
        try:
            with os.fdopen(handle, "w", encoding="utf-8") as stream:
                stream.write(payload)
                stream.write("\n")
            os.replace(temporary, self.directory / RUN_FILE)
        except BaseException:
            Path(temporary).unlink(missing_ok=True)
            raise
