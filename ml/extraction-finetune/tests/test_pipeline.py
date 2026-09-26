import json
from pathlib import Path
from typing import Any

import pytest

from trenova_finetune import dataset
from trenova_finetune.card import CARD_FILE
from trenova_finetune.config import load_config
from trenova_finetune.pipeline import Stages, execute, open_run
from trenova_finetune.training import StageResult


class FakeStages:
    def __init__(self, *, dpo_skipped: bool = False, fail_at: str | None = None) -> None:
        self.calls: list[str] = []
        self.data_dirs: list[Path] = []
        self.dpo_skipped = dpo_skipped
        self.fail_at = fail_at

    def _maybe_fail(self, name: str) -> None:
        self.calls.append(name)
        if self.fail_at == name:
            raise RuntimeError(f"{name} broke")

    def train_sft(self, config: Any, data_dir: Path, out: Path) -> StageResult:
        self._maybe_fail("sft")
        self.data_dirs.append(data_dir)
        (out / "adapter").mkdir(parents=True)
        return StageResult(output=out / "adapter", metrics={"train_loss": 0.4})

    def merge(self, config: Any, base: Any, adapter: Path, out: Path, **_: Any) -> Path:
        self._maybe_fail(f"merge:{adapter.parent.name}")
        out.mkdir(parents=True)
        return out

    def train_dpo(self, config: Any, data_dir: Path, model: Path, out: Path) -> StageResult:
        self._maybe_fail("dpo")
        self.data_dirs.append(data_dir)
        if self.dpo_skipped:
            return StageResult(output=None, metrics={"pairs": 0}, note="too few", skipped=True)
        (out / "adapter").mkdir(parents=True)
        return StageResult(output=out / "adapter", metrics={"pairs": 2})

    def predict(self, config: Any, model: Path, dataset_dir: Path, out: Path) -> dict:
        self._maybe_fail("predict")
        out.write_text("")
        return {"predictions": 2, "failed": 0}

    def stages(self) -> Stages:
        return Stages(
            train_sft=self.train_sft,
            merge=self.merge,
            train_dpo=self.train_dpo,
            predict=self.predict,
        )


def _open(tmp_path: Path, dataset_dir: Path, config_path: Path, *, resume: bool = False):
    config = load_config(config_path)
    run, manifest = open_run(config, dataset_dir, tmp_path / "run", resume=resume)
    return config, run, manifest


def test_the_full_pipeline_ends_on_the_preference_model(
    tmp_path: Path, dataset_dir: Path, config_path: Path
) -> None:
    config, run, manifest = _open(tmp_path, dataset_dir, config_path)
    fake = FakeStages()
    final = execute(config, dataset_dir, run, manifest, fake.stages())

    assert fake.calls == ["sft", "merge:sft", "dpo", "merge:dpo", "predict"]
    assert fake.data_dirs == [run.path("data"), run.path("data")]
    assert run.record.stages["targets"].metrics == {"train": 3, "validation": 2, "preference": 3}
    assert final == run.path("dpo", "model")
    card = json.loads((final / CARD_FILE).read_text())
    assert card["provider"]["structuredOutputMode"] == "JSONSchema"
    assert card["provider"]["temperature"] == pytest.approx(0.1)
    assert card["dataset"]["exportId"] == "aitx_1"
    assert run.record.final_model == str(final)


def test_skipped_preference_training_serves_the_sft_model(
    tmp_path: Path, dataset_dir: Path, config_path: Path
) -> None:
    config, run, manifest = _open(tmp_path, dataset_dir, config_path)
    fake = FakeStages(dpo_skipped=True)
    final = execute(config, dataset_dir, run, manifest, fake.stages(), skip_predict=True)

    assert fake.calls == ["sft", "merge:sft", "dpo"]
    assert final == run.path("sft", "model")
    assert run.record.stages["merge-dpo"].status == "skipped"
    assert "predict" not in run.record.stages


def test_a_failed_run_resumes_where_it_stopped(
    tmp_path: Path, dataset_dir: Path, config_path: Path
) -> None:
    config, run, manifest = _open(tmp_path, dataset_dir, config_path)
    with pytest.raises(RuntimeError, match="dpo broke"):
        execute(config, dataset_dir, run, manifest, FakeStages(fail_at="dpo").stages())
    assert run.record.stages["dpo"].status == "failed"

    config, resumed, manifest = _open(tmp_path, dataset_dir, config_path, resume=True)
    fake = FakeStages()
    execute(config, dataset_dir, resumed, manifest, fake.stages())
    assert fake.calls == ["dpo", "merge:dpo", "predict"]


def test_a_tampered_dataset_stops_before_training(
    tmp_path: Path, dataset_dir: Path, config_path: Path
) -> None:
    config, run, manifest = _open(tmp_path, dataset_dir, config_path)
    path = dataset_dir / dataset.VALIDATION_FILE
    path.write_bytes(path.read_bytes() + b"\n")
    fake = FakeStages()
    with pytest.raises(dataset.DatasetError):
        execute(config, dataset_dir, run, manifest, fake.stages())
    assert fake.calls == []
