from pathlib import Path

import pytest

from trenova_finetune import dataset
from trenova_finetune.cli import EXIT_USER_ERROR, build_parser, main


def test_verify_prints_the_counts(dataset_dir: Path, capsys: pytest.CaptureFixture[str]) -> None:
    assert main(["verify", "--dataset", str(dataset_dir)]) == 0
    assert '"exportId": "aitx_1"' in capsys.readouterr().out


def test_user_errors_exit_with_a_message(
    dataset_dir: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    (dataset_dir / dataset.SCHEMA_FILE).write_text("{}")
    assert main(["verify", "--dataset", str(dataset_dir)]) == EXIT_USER_ERROR
    assert "has been changed" in capsys.readouterr().err


def test_run_refuses_an_invalid_config_before_touching_the_gpu(
    tmp_path: Path, dataset_dir: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    config = tmp_path / "bad.yaml"
    config.write_text("base_model: x\nquantization: 3bit\n")
    code = main(
        [
            "run",
            "--config",
            str(config),
            "--dataset",
            str(dataset_dir),
            "--out",
            str(tmp_path / "r"),
        ]
    )
    assert code == EXIT_USER_ERROR
    assert "quantization" in capsys.readouterr().err
    assert not (tmp_path / "r").exists()


def test_targets_builds_training_data(
    tmp_path: Path, dataset_dir: Path, config_path: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    out = tmp_path / "data"
    code = main(
        ["targets", "--config", str(config_path), "--dataset", str(dataset_dir), "--out", str(out)]
    )
    assert code == 0
    assert '"preference": 3' in capsys.readouterr().out
    assert (out / "sft-train.jsonl").exists()


def test_every_command_parses() -> None:
    parser = build_parser()
    for argv in (
        ["verify", "--dataset", "d"],
        ["targets", "--config", "c", "--dataset", "d", "--out", "o"],
        ["run", "--config", "c", "--dataset", "d", "--out", "o", "--resume", "--skip-predict"],
        ["merge", "--config", "c", "--base", "b", "--adapter", "a", "--out", "o"],
        ["predict", "--config", "c", "--dataset", "d", "--model", "m", "--out", "o"],
    ):
        assert parser.parse_args(argv).command == argv[0]
