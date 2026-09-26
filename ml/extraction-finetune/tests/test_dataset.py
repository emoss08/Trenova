import json
from pathlib import Path

import pytest

from trenova_finetune import dataset

from .conftest import FIELD_KEYS, write_dataset


def test_a_rendered_dataset_verifies(dataset_dir: Path) -> None:
    manifest = dataset.load_manifest(dataset_dir)
    dataset.verify(dataset_dir, manifest)

    assert manifest.export_id == "aitx_1"
    assert manifest.structured_output_mode == "JSONSchema"
    assert manifest.temperature == pytest.approx(0.1)
    assert manifest.page_limit == 2500
    assert manifest.field_keys == tuple(FIELD_KEYS)
    assert len(manifest.sha256) == 64


def test_an_edited_file_fails_verification(dataset_dir: Path) -> None:
    manifest = dataset.load_manifest(dataset_dir)
    path = dataset_dir / dataset.TRAIN_FILE
    path.write_bytes(path.read_bytes().replace(b"Hartwell", b"Hartwall"))
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
        (lambda m: m.update(format="trenova.extraction-dataset/v1"), "render the export again"),
        (lambda m: m.update(structuredOutputMode="Telepathy"), "structured output"),
        (lambda m: m.pop("pageLimit"), "how much of each page"),
        (lambda m: m.update(fieldKeys=[]), "field keys"),
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


def test_examples_load_by_split(dataset_dir: Path) -> None:
    train = dataset.load_examples(dataset_dir, dataset.SPLIT_TRAIN)
    validation = dataset.load_examples(dataset_dir, dataset.SPLIT_VALIDATION)

    assert [example.id for example in train] == ["t0", "t1", "t2"]
    assert len(validation) == 2
    first = train[0]
    assert first.target.fields["shipper"] == "Juniper Manufacturing"
    assert first.prediction is not None
    assert first.prediction.fields["shipper"] == "Harbor Supply"
    assert first.target.stops[1].appointment_required
    assert first.visible_pages[0].number == 1
    assert first.outcomes["shipper"] == "Corrected"

    prompts = dataset.load_evaluation_prompts(dataset_dir)
    assert [identifier for identifier, _ in prompts] == ["v0", "v1"]
    assert dataset.load_schema(dataset_dir)["type"] == "object"


@pytest.mark.parametrize(
    ("mutate", "message"),
    [
        (lambda r: r.pop("id"), "no id"),
        (lambda r: r.update(split="validation"), "another split"),
        (lambda r: r.update(target=None), "no confirmed values"),
        (lambda r: r.update(visiblePages=[]), "no page text"),
        (lambda r: r["target"].update(fields={"rate": 12}), "map names to text"),
        (lambda r: r["target"]["stops"][0].pop("role"), "no role"),
        (lambda r: r.update(outcomes=["Correct"]), "outcomes"),
        (lambda r: r["prompt"].reverse(), "in order"),
    ],
)
def test_malformed_examples_are_rejected(tmp_path: Path, mutate, message: str) -> None:
    root = write_dataset(tmp_path / "d")
    path = root / dataset.TRAIN_FILE
    record = json.loads(path.read_text().splitlines()[0])
    mutate(record)
    path.write_text(json.dumps(record) + "\n")
    with pytest.raises(dataset.DatasetError, match=message):
        dataset.load_examples(root, dataset.SPLIT_TRAIN)


def test_duplicate_ids_are_rejected(tmp_path: Path) -> None:
    root = write_dataset(tmp_path / "d")
    for name, loader in (
        (dataset.EVALUATION_FILE, dataset.load_evaluation_prompts),
        (dataset.TRAIN_FILE, lambda d: dataset.load_examples(d, dataset.SPLIT_TRAIN)),
    ):
        path = root / name
        first = path.read_text().splitlines()[0]
        path.write_text(first + "\n" + first + "\n")
        with pytest.raises(dataset.DatasetError, match="appears twice"):
            loader(root)


def test_invalid_json_lines_name_their_line(tmp_path: Path) -> None:
    path = tmp_path / "x.jsonl"
    path.write_text('{"id": "a"}\n\n[1]\n')
    with pytest.raises(dataset.DatasetError, match=r"x.jsonl:3"):
        list(dataset.read_jsonl(path))
