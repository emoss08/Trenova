import json
from pathlib import Path
from types import SimpleNamespace

import pytest

from trenova_finetune import dataset
from trenova_finetune.config import load_config
from trenova_finetune.predict import (
    ERROR_EMPTY,
    ERROR_TRUNCATED,
    prediction_record,
    sampling_settings,
    structured_kind,
    write_predictions,
)


def test_structured_output_follows_the_provider_mode() -> None:
    assert structured_kind("JSONSchema") == "json_schema"
    assert structured_kind("JSONMode") == "json_object"
    assert structured_kind("Prompted") == "none"
    with pytest.raises(dataset.DatasetError):
        structured_kind("Telepathy")


def test_sampling_matches_the_rendered_dataset(dataset_dir: Path, config_path: Path) -> None:
    settings = sampling_settings(dataset.load_manifest(dataset_dir), load_config(config_path))
    assert settings["temperature"] == pytest.approx(0.1)
    assert settings["top_p"] == pytest.approx(0.95)
    assert settings["max_tokens"] == 2048
    assert settings["seed"] == 42


def _output(text: str, finish: str = "stop") -> SimpleNamespace:
    return SimpleNamespace(outputs=[SimpleNamespace(text=text, finish_reason=finish)])


def test_prediction_records_flag_what_the_scorer_cannot_use() -> None:
    assert prediction_record("a", _output(' {"fields": []} ')) == {
        "id": "a",
        "reply": '{"fields": []}',
    }
    assert prediction_record("b", _output('{"fie', "length"))["error"] == ERROR_TRUNCATED
    assert prediction_record("c", _output("   "))["error"] == ERROR_EMPTY
    assert prediction_record("d", SimpleNamespace(outputs=[]))["error"] == ERROR_EMPTY


def test_predictions_are_written_whole(tmp_path: Path) -> None:
    path = tmp_path / "out" / "predictions.jsonl"
    write_predictions(path, [{"id": "a", "reply": "{}"}, {"id": "b", "reply": "", "error": "x"}])
    lines = [json.loads(line) for line in path.read_text().splitlines()]
    assert [line["id"] for line in lines] == ["a", "b"]
    assert not list(path.parent.glob(".predictions-*"))
