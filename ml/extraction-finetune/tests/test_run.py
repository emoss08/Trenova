import json
from pathlib import Path

import pytest

from trenova_finetune import dataset
from trenova_finetune.config import load_config
from trenova_finetune.run import RUN_FILE, STATUS_COMPLETED, STATUS_FAILED, Run, RunError


def test_a_run_records_its_lineage(tmp_path: Path, dataset_dir: Path, config_path: Path) -> None:
    config = load_config(config_path)
    manifest = dataset.load_manifest(dataset_dir)
    run = Run.create(tmp_path / "run", config, manifest)

    run.start("sft")
    run.finish("sft", tmp_path, metrics={"loss": 0.5})
    saved = json.loads((tmp_path / "run" / RUN_FILE).read_text())

    assert saved["base_model"] == "Qwen/Qwen2.5-7B-Instruct"
    assert saved["dataset"]["exportId"] == "aitx_1"
    assert saved["dataset"]["datasetManifestSha256"] == manifest.sha256
    assert saved["stages"]["sft"]["status"] == STATUS_COMPLETED
    assert saved["stages"]["sft"]["metrics"] == {"loss": 0.5}
    assert run.completed("sft")


def test_a_run_will_not_reuse_a_directory(
    tmp_path: Path, dataset_dir: Path, config_path: Path
) -> None:
    config = load_config(config_path)
    manifest = dataset.load_manifest(dataset_dir)
    Run.create(tmp_path / "run", config, manifest)
    with pytest.raises(RunError, match="already holds files"):
        Run.create(tmp_path / "run", config, manifest)


def test_resume_refuses_a_changed_configuration(
    tmp_path: Path, dataset_dir: Path, config_path: Path
) -> None:
    config = load_config(config_path)
    manifest = dataset.load_manifest(dataset_dir)
    run = Run.create(tmp_path / "run", config, manifest)
    run.start("sft")
    run.fail("sft", RuntimeError("out of memory"))

    resumed = Run.resume(tmp_path / "run", config, manifest)
    assert resumed.record.stages["sft"].status == STATUS_FAILED
    assert not resumed.completed("sft")

    changed = config.model_copy(update={"seed": 7})
    with pytest.raises(RunError, match="configuration differs"):
        Run.resume(tmp_path / "run", changed, manifest)
    with pytest.raises(RunError, match="cannot resume"):
        Run.resume(tmp_path / "elsewhere", config, manifest)


def test_a_completed_stage_whose_output_vanished_runs_again(
    tmp_path: Path, dataset_dir: Path, config_path: Path
) -> None:
    config = load_config(config_path)
    manifest = dataset.load_manifest(dataset_dir)
    run = Run.create(tmp_path / "run", config, manifest)
    output = tmp_path / "adapter"
    output.mkdir()
    run.start("sft")
    run.finish("sft", output)
    assert run.completed("sft")
    output.rmdir()
    assert not run.completed("sft")
