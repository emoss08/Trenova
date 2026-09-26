import json
from pathlib import Path

import pytest

from trenova_finetune import dataset

from .conftest import write_dataset


def test_a_rendered_dataset_verifies(dataset_dir: Path) -> None:
    manifest = dataset.load_manifest(dataset_dir)
    dataset.verify(dataset_dir, manifest)

    assert manifest.export_id == "aitx_1"
    assert manifest.structured_output_mode == "JSONSchema"
    assert manifest.temperature == pytest.approx(0.1)
    assert len(manifest.sha256) == 64


def test_an_edited_file_fails_verification(dataset_dir: Path) -> None:
    manifest = dataset.load_manifest(dataset_dir)
    path = dataset_dir / dataset.SFT_TRAIN_FILE
    path.write_bytes(path.read_bytes().replace(b"Load 1", b"Load 2"))
    with pytest.raises(dataset.DatasetError, match="has been changed"):
        dataset.verify(dataset_dir, manifest)


def test_a_missing_file_fails_verification(dataset_dir: Path) -> None:
    manifest = dataset.load_manifest(dataset_dir)
    (dataset_dir / dataset.EVALUATION_FILE).unlink()
    with pytest.raises(dataset.DatasetError, match="cannot read"):
        dataset.verify(dataset_dir, manifest)


def test_manifest_problems_are_reported(dataset_dir: Path) -> None:
    path = dataset_dir / dataset.MANIFEST_FILE
    raw = json.loads(path.read_text())

    for mutate, message in [
        (lambda m: m.update(format="other"), "not a"),
        (lambda m: m.update(structuredOutputMode="Telepathy"), "structured output"),
        (lambda m: m.update(files=m["files"][1:]), "does not list"),
        (lambda m: m["files"][0].update(name="../escape.jsonl"), "outside the dataset"),
        (lambda m: m["files"][0].pop("sha256"), "checksum"),
    ]:
        broken = json.loads(json.dumps(raw))
        mutate(broken)
        path.write_text(json.dumps(broken))
        with pytest.raises(dataset.DatasetError, match=message):
            dataset.load_manifest(dataset_dir)

    path.write_text("{not json")
    with pytest.raises(dataset.DatasetError, match="not valid JSON"):
        dataset.load_manifest(dataset_dir)


def test_records_load_by_split(dataset_dir: Path) -> None:
    train, validation = dataset.load_sft(dataset_dir)
    assert [record["id"] for record in train] == ["t0", "t1", "t2"]
    assert len(validation) == 2
    assert train[0]["prompt"][0]["role"] == "system"

    pairs = dataset.load_preferences(dataset_dir)
    assert len(pairs) == 2
    prompts = dataset.load_evaluation_prompts(dataset_dir)
    assert [identifier for identifier, _ in prompts] == ["v0", "v1"]
    assert dataset.load_schema(dataset_dir) == {"type": "object"}


@pytest.mark.parametrize(
    ("record", "message"),
    [
        ({"prompt": [], "completion": []}, "no id|must hold"),
        (
            {
                "id": "x",
                "prompt": [{"role": "user", "content": "a"}, {"role": "system", "content": "b"}],
                "completion": [{"role": "assistant", "content": "{}"}],
            },
            "in order",
        ),
        (
            {
                "id": "x",
                "prompt": [{"role": "system", "content": "a"}, {"role": "user", "content": "b"}],
                "completion": [{"role": "assistant", "content": "not json"}],
            },
            "not JSON",
        ),
        (
            {
                "id": "x",
                "prompt": [{"role": "system", "content": ""}, {"role": "user", "content": "b"}],
                "completion": [{"role": "assistant", "content": "{}"}],
            },
            "empty",
        ),
    ],
)
def test_malformed_sft_records_are_rejected(tmp_path: Path, record: dict, message: str) -> None:
    root = write_dataset(tmp_path / "d")
    (root / dataset.SFT_TRAIN_FILE).write_text(json.dumps(record) + "\n")
    with pytest.raises(dataset.DatasetError, match=message):
        dataset.load_sft(root)


def test_duplicate_evaluation_ids_are_rejected(tmp_path: Path) -> None:
    root = write_dataset(tmp_path / "d")
    path = root / dataset.EVALUATION_FILE
    first = path.read_text().splitlines()[0]
    path.write_text(first + "\n" + first + "\n")
    with pytest.raises(dataset.DatasetError, match="appears twice"):
        dataset.load_evaluation_prompts(root)


def test_invalid_json_lines_name_their_line(tmp_path: Path) -> None:
    path = tmp_path / "x.jsonl"
    path.write_text('{"id": "a"}\n\n[1]\n')
    with pytest.raises(dataset.DatasetError, match=r"x.jsonl:3"):
        list(dataset.read_jsonl(path))
