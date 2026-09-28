import json
from pathlib import Path

import pytest

from trenova_finetune import cli, dataset, targets
from trenova_finetune.card import CARD_FILE, CARD_FORMAT, save_card
from trenova_finetune.config import TargetSettings, load_config
from trenova_finetune.quantize import (
    SCHEMES,
    QuantizeError,
    calibration_conversations,
    check_source,
    quantize,
    quantized_card,
    scheme_for,
)
from trenova_finetune.serve import ServeError, serve


def _model(root: Path, card: dict | None = None) -> Path:
    root.mkdir(parents=True, exist_ok=True)
    save_card(root, card or {"format": CARD_FORMAT, "baseModel": "Qwen/Qwen2.5-7B-Instruct"})
    return root


def _records(count: int) -> list[dict]:
    return [
        {
            "id": f"t{i}",
            "prompt": [
                {"role": "system", "content": "Extract."},
                {"role": "user", "content": f"doc {i}"},
            ],
            "completion": [{"role": "assistant", "content": "{}"}],
        }
        for i in range(count)
    ]


def test_schemes_name_their_llmcompressor_preset() -> None:
    assert scheme_for("fp8-dynamic").preset == "FP8_DYNAMIC"
    assert not scheme_for("fp8-dynamic").calibrated
    assert scheme_for("w4a16").preset == "W4A16"
    assert scheme_for("w4a16").method == "gptq"
    assert scheme_for("w4a16").calibrated
    with pytest.raises(QuantizeError, match="fp8-dynamic, w4a16"):
        scheme_for("int3")


def test_calibration_uses_whole_conversations_reproducibly() -> None:
    records = _records(40)
    first = calibration_conversations(records, 16, seed=42)
    assert len(first) == 16
    assert first == calibration_conversations(records, 16, seed=42)
    assert first != calibration_conversations(records, 16, seed=7)
    assert [message["role"] for message in first[0]] == ["system", "user", "assistant"]

    assert len(calibration_conversations(records[:5], 16, seed=42)) == 5
    with pytest.raises(QuantizeError, match="no examples"):
        calibration_conversations([], 16, seed=42)


def test_the_card_records_how_the_model_was_quantized(tmp_path: Path) -> None:
    card = {"format": CARD_FORMAT, "provider": {"kind": "OpenAIChat"}}
    calibrated = quantized_card(
        card,
        SCHEMES["w4a16"],
        tmp_path,
        samples=512,
        max_seq_length=8192,
        ignore=("lm_head",),
    )
    assert calibrated["provider"] == {"kind": "OpenAIChat"}
    assert calibrated["quantization"]["scheme"] == "w4a16"
    assert calibrated["quantization"]["format"] == "compressed-tensors"
    assert calibrated["quantization"]["calibrationSamples"] == 512
    assert calibrated["quantization"]["source"] == str(tmp_path)
    assert "quantization" not in card

    dynamic = quantized_card(
        card, SCHEMES["fp8-dynamic"], tmp_path, samples=0, max_seq_length=8192, ignore=("lm_head",)
    )
    assert "calibrationSamples" not in dynamic["quantization"]


def test_the_source_must_be_an_unquantized_pipeline_model(tmp_path: Path) -> None:
    with pytest.raises(QuantizeError, match=CARD_FILE):
        check_source(tmp_path / "missing", tmp_path / "out")

    quantized = _model(
        tmp_path / "quantized", {"format": CARD_FORMAT, "quantization": {"scheme": "w4a16"}}
    )
    with pytest.raises(QuantizeError, match="already quantized"):
        check_source(quantized, tmp_path / "out")

    model = _model(tmp_path / "model")
    with pytest.raises(QuantizeError, match="new directory"):
        check_source(model, model)

    used = tmp_path / "used"
    used.mkdir()
    (used / "weights.bin").write_text("x")
    with pytest.raises(FileExistsError):
        check_source(model, used)

    assert check_source(model, tmp_path / "fresh")["format"] == CARD_FORMAT


def test_calibrated_schemes_need_training_data(tmp_path: Path, config_path: Path) -> None:
    config = load_config(config_path)
    model = _model(tmp_path / "model")
    with pytest.raises(QuantizeError, match="--data"):
        quantize(config, model, "w4a16", tmp_path / "out")
    with pytest.raises(dataset.DatasetError):
        quantize(config, model, "w4a16", tmp_path / "out", data_dir=tmp_path / "no-data")


def test_calibration_reads_the_built_training_data(
    tmp_path: Path, dataset_dir: Path, config_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    manifest = dataset.load_manifest(dataset_dir)
    data = targets.build(dataset_dir, tmp_path / "data", manifest, TargetSettings())
    reached: list[str] = []

    def stop_before_the_gpu() -> None:
        reached.append("gpu")
        raise RuntimeError("no GPU in tests")

    monkeypatch.setattr("trenova_finetune.quantize.require_cuda", stop_before_the_gpu)
    with pytest.raises(RuntimeError, match="no GPU"):
        quantize(
            load_config(config_path),
            _model(tmp_path / "model"),
            "w4a16",
            tmp_path / "out",
            data_dir=data.directory,
        )
    assert reached == ["gpu"]
    assert not (tmp_path / "out").exists()


def test_calibration_length_defaults_to_the_training_length(config_path: Path) -> None:
    config = load_config(config_path)
    assert config.calibration_length == config.sft.max_length
    tuned = config.model_copy(
        update={"quantize": config.quantize.model_copy(update={"max_seq_length": 2048})}
    )
    assert tuned.calibration_length == 2048


def test_a_quantized_model_is_not_quantized_again_when_served(
    tmp_path: Path, config_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setenv("VLLM_API_KEY", "sk")
    config = load_config(config_path)
    online = config.model_copy(
        update={"serve": config.serve.model_copy(update={"quantization": "fp8"})}
    )
    model = _model(
        tmp_path / "model", {"format": CARD_FORMAT, "quantization": {"scheme": "fp8-dynamic"}}
    )
    with pytest.raises(ServeError, match=r"serve\.quantization to null"):
        serve(online, model, "m", dry_run=True)
    assert serve(config, model, "m", dry_run=True).startswith("vllm serve")


def test_the_quantize_command_reports_user_errors(
    tmp_path: Path, config_path: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    model = _model(tmp_path / "model")
    code = cli.main(
        [
            "quantize",
            "--config",
            str(config_path),
            "--model",
            str(model),
            "--scheme",
            "w4a16",
            "--out",
            str(tmp_path / "out"),
        ]
    )
    assert code == cli.EXIT_USER_ERROR
    assert "--data" in capsys.readouterr().err

    with pytest.raises(SystemExit):
        cli.main(
            [
                "quantize",
                "--config",
                str(config_path),
                "--model",
                str(model),
                "--scheme",
                "int3",
                "--out",
                str(tmp_path / "out"),
            ]
        )


def test_saved_cards_are_readable_json(tmp_path: Path) -> None:
    save_card(tmp_path, {"format": CARD_FORMAT, "b": 1, "a": 2})
    assert json.loads((tmp_path / CARD_FILE).read_text()) == {"format": CARD_FORMAT, "a": 2, "b": 1}
