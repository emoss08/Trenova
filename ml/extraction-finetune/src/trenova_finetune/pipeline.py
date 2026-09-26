"""The end-to-end run: verify, fine-tune, merge, optionally prefer, merge, predict."""

from __future__ import annotations

import json
import logging
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from . import dataset, targets
from .config import PipelineConfig
from .run import STATUS_COMPLETED, STATUS_SKIPPED, Run

log = logging.getLogger(__name__)

STAGE_VERIFY = "verify"
STAGE_TARGETS = "targets"
STAGE_SFT = "sft"
STAGE_MERGE_SFT = "merge-sft"
STAGE_DPO = "dpo"
STAGE_MERGE_DPO = "merge-dpo"
STAGE_PREDICT = "predict"
PREDICTIONS_FILE = "predictions.jsonl"
CARD_FORMAT = "trenova.extraction-model/v1"
CARD_FILE = "trenova-model.json"


@dataclass(frozen=True)
class Stages:
    """The heavy work each stage does, injected so the ordering can be tested without a GPU."""

    train_sft: Callable[[PipelineConfig, Path, Path], Any]
    merge: Callable[..., Path]
    train_dpo: Callable[[PipelineConfig, Path, Path, Path], Any]
    predict: Callable[[PipelineConfig, Path, Path, Path], dict[str, Any]]


def default_stages() -> Stages:
    from .merge import merge_adapter
    from .predict import predict
    from .training import train_dpo, train_sft

    return Stages(train_sft=train_sft, merge=merge_adapter, train_dpo=train_dpo, predict=predict)


def open_run(
    config: PipelineConfig,
    dataset_dir: Path,
    out: Path,
    *,
    resume: bool,
) -> tuple[Run, dataset.DatasetManifest]:
    manifest = dataset.load_manifest(dataset_dir)
    run = Run.resume(out, config, manifest) if resume else Run.create(out, config, manifest)
    return run, manifest


def execute(
    config: PipelineConfig,
    dataset_dir: Path,
    run: Run,
    manifest: dataset.DatasetManifest,
    stages: Stages,
    *,
    skip_predict: bool = False,
) -> Path:
    def step(name: str, work: Callable[[], None]) -> None:
        if run.completed(name):
            log.info("%s: already done, skipping", name)
            return
        log.info("%s: starting", name)
        run.start(name)
        try:
            work()
        except BaseException as err:
            run.fail(name, err)
            raise
        log.info("%s: done", name)

    def verify() -> None:
        dataset.verify(dataset_dir, manifest)
        run.finish(STAGE_VERIFY, None, metrics=dict(manifest.counts))

    def build_targets() -> None:
        data = targets.build(dataset_dir, run.path("data"), manifest, config.targets)
        run.finish(STAGE_TARGETS, data.directory, metrics=dict(data.counts))

    def sft() -> None:
        result = stages.train_sft(config, run.path("data"), run.path("sft"))
        run.finish(STAGE_SFT, result.output, metrics=result.metrics)

    def merge_sft() -> None:
        output = stages.merge(
            config,
            config.base_model,
            run.path("sft", "adapter"),
            run.path("sft", "model"),
            revision=config.revision,
        )
        run.finish(STAGE_MERGE_SFT, output)

    def dpo() -> None:
        if not config.dpo.enabled:
            run.finish(STAGE_DPO, None, note="disabled in the configuration", status=STATUS_SKIPPED)
            return
        result = stages.train_dpo(
            config, run.path("data"), run.path("sft", "model"), run.path("dpo")
        )
        run.finish(
            STAGE_DPO,
            result.output,
            metrics=result.metrics,
            note=result.note,
            status=STATUS_SKIPPED if result.skipped else STATUS_COMPLETED,
        )

    def merge_dpo() -> None:
        if run.record.stages[STAGE_DPO].status == STATUS_SKIPPED:
            run.finish(STAGE_MERGE_DPO, None, note="no preference adapter", status=STATUS_SKIPPED)
            return
        output = stages.merge(
            config,
            run.path("sft", "model"),
            run.path("dpo", "adapter"),
            run.path("dpo", "model"),
        )
        run.finish(STAGE_MERGE_DPO, output)

    step(STAGE_VERIFY, verify)
    step(STAGE_TARGETS, build_targets)
    step(STAGE_SFT, sft)
    step(STAGE_MERGE_SFT, merge_sft)
    step(STAGE_DPO, dpo)
    step(STAGE_MERGE_DPO, merge_dpo)

    final = (
        run.path("dpo", "model")
        if run.record.stages[STAGE_MERGE_DPO].status != STATUS_SKIPPED
        else run.path("sft", "model")
    )
    write_card(final, run, manifest)
    run.record.final_model = str(final)
    run.save()

    if not skip_predict:

        def predict() -> None:
            output = run.path(PREDICTIONS_FILE)
            metrics = stages.predict(config, final, dataset_dir, output)
            run.finish(STAGE_PREDICT, output, metrics=metrics)

        step(STAGE_PREDICT, predict)

    return final


def model_card(run: Run, manifest: dataset.DatasetManifest) -> dict[str, Any]:
    return {
        "format": CARD_FORMAT,
        "baseModel": run.record.base_model,
        "revision": run.record.revision,
        "pipelineVersion": run.record.pipeline_version,
        "dataset": run.record.dataset,
        "provider": {
            "kind": "OpenAIChat",
            "structuredOutputMode": manifest.structured_output_mode,
            "task": "DocumentExtraction",
            "schemaName": manifest.schema_name,
            "temperature": manifest.temperature,
            "topP": manifest.top_p,
        },
        "stages": {
            name: {"status": stage.status, "metrics": stage.metrics, "note": stage.note}
            for name, stage in run.record.stages.items()
        },
        "environment": run.record.environment,
    }


def write_card(model_dir: Path, run: Run, manifest: dataset.DatasetManifest) -> None:
    card = model_card(run, manifest)
    (model_dir / CARD_FILE).write_text(json.dumps(card, indent=2, sort_keys=True) + "\n", "utf-8")
