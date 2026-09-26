import copy
import json
from pathlib import Path

import pytest
from jsonschema import Draft202012Validator

from trenova_finetune import built, dataset
from trenova_finetune.config import TargetSettings
from trenova_finetune.targets import (
    BUILT_MANIFEST_FILE,
    PREFERENCE_FILE,
    SFT_TRAIN_FILE,
    SFT_VALIDATION_FILE,
    ReplyBuilder,
    build,
    evidence_needles,
    humanize,
    locate_evidence,
    reply_shape,
    reply_text,
)

from .conftest import FIELD_KEYS, SCHEMA, write_dataset


def _builder(recipe: TargetSettings | None = None) -> ReplyBuilder:
    return ReplyBuilder(reply_shape(SCHEMA, FIELD_KEYS), recipe or TargetSettings())


def _example(dataset_dir: Path) -> dataset.Example:
    return dataset.load_examples(dataset_dir, dataset.SPLIT_TRAIN)[0]


def _fields(reply: dict) -> dict[str, dict]:
    return {field["key"]: field for field in reply["fields"]}


def test_labels_read_like_words() -> None:
    assert humanize("loadNumber") == "Load Number"
    assert humanize("referenceNumber") == "Reference Number"
    assert humanize("bol") == "Bol"


def test_money_is_found_with_or_without_separators() -> None:
    assert evidence_needles(" 2563.12 ") == ["2563.12", "2,563.12"]
    assert evidence_needles("2450.00") == ["2450.00", "2,450"]
    assert evidence_needles("Harbor Supply") == ["harbor supply"]
    assert evidence_needles("") == []

    pages = (dataset.Page(1, "intro"), dataset.Page(2, "Total due:   $2,563.12 USD by Friday"))
    page, excerpt = locate_evidence("2563.12", pages, 10)
    assert page == 2
    assert excerpt == "l due: $2,563.12 USD by Fr"
    assert locate_evidence("absent", pages, 10) == (0, "")


def test_the_target_is_the_confirmed_answer(dataset_dir: Path) -> None:
    example = _example(dataset_dir)
    recipe = TargetSettings()
    reply = _builder(recipe).build(
        example.target,
        supplement=example.prediction,
        confidence=recipe.verified_confidence,
        document_kind=example.document_kind,
        pages=example.visible_pages,
    )
    Draft202012Validator(SCHEMA).validate(reply)

    assert list(reply) == SCHEMA["required"]
    fields = _fields(reply)
    assert [field["key"] for field in reply["fields"]] == [
        key for key in FIELD_KEYS if key in fields
    ]
    assert list(reply["fields"][0]) == SCHEMA["properties"]["fields"]["items"]["required"]
    assert fields["shipper"]["value"] == "Juniper Manufacturing"
    assert fields["shipper"]["confidence"] == pytest.approx(0.95)
    assert fields["loadNumber"]["confidence"] == pytest.approx(0.7)
    assert fields["rate"]["pageNumber"] == 1
    assert "$2,563.12" in fields["rate"]["evidenceExcerpt"]
    assert "weight" not in fields

    pickup, delivery = reply["stops"]
    assert (pickup["sequence"], delivery["sequence"]) == (1, 2)
    assert pickup["timeWindow"] == "08:00-14:00"
    assert delivery["timeWindow"] == ""
    assert delivery["appointmentRequired"] is True
    assert pickup["evidenceExcerpt"]


def test_unverified_fields_can_be_left_out(dataset_dir: Path) -> None:
    example = _example(dataset_dir)
    reply = _builder().build(
        example.target,
        supplement=None,
        confidence=0.95,
        document_kind="",
        pages=example.visible_pages,
    )
    assert "loadNumber" not in _fields(reply)
    assert reply["documentKind"] == "RateConfirmation"
    assert reply["stops"][0]["timeWindow"] == ""


def test_values_are_held_to_the_schema_limits() -> None:
    shape = reply_shape(SCHEMA, FIELD_KEYS)
    snapshot = dataset.Snapshot(
        fields={key: "x" * 1000 for key in FIELD_KEYS},
        stops=tuple(dataset.Stop(role="pickup", name="n" * 1000) for _ in range(20)),
    )
    reply = ReplyBuilder(shape, TargetSettings()).build(
        snapshot, supplement=None, confidence=0.9, document_kind="", pages=()
    )
    Draft202012Validator(SCHEMA).validate(reply)
    assert len(reply["fields"]) == shape.max_fields
    assert len(reply["stops"]) == shape.max_stops
    assert len(reply["fields"][0]["value"]) == shape.field_limits["value"]


def test_the_schema_and_manifest_must_agree_on_field_keys() -> None:
    with pytest.raises(dataset.DatasetError, match="field keys differ"):
        reply_shape(SCHEMA, [*FIELD_KEYS, "somethingNew"])
    with pytest.raises(dataset.DatasetError, match="not the extraction reply schema"):
        reply_shape({"type": "object"}, FIELD_KEYS)


def test_build_writes_verified_training_data(tmp_path: Path, dataset_dir: Path) -> None:
    manifest = dataset.load_manifest(dataset_dir)
    data = build(dataset_dir, tmp_path / "data", manifest, TargetSettings())

    assert data.counts == {"train": 3, "validation": 2, "preference": 3}
    written = built.verify(data.directory)
    assert written["datasetManifestSha256"] == manifest.sha256
    assert written["recipe"]["verified_confidence"] == pytest.approx(0.95)

    train, validation = built.load_sft(data.directory)
    assert [record["id"] for record in train] == ["t0", "t1", "t2"]
    assert len(validation) == 2
    reply = json.loads(train[0]["completion"][0]["content"])
    Draft202012Validator(SCHEMA).validate(reply)
    assert train[0]["completion"][0]["content"] == reply_text(reply)

    pairs = built.load_preferences(data.directory)
    assert len(pairs) == 3
    chosen = _fields(json.loads(pairs[0]["chosen"][0]["content"]))
    rejected = _fields(json.loads(pairs[0]["rejected"][0]["content"]))
    assert chosen["shipper"]["value"] == "Juniper Manufacturing"
    assert rejected["shipper"]["value"] == "Harbor Supply"
    assert rejected["shipper"]["confidence"] == pytest.approx(0.7)

    for name in (SFT_TRAIN_FILE, SFT_VALIDATION_FILE, PREFERENCE_FILE, BUILT_MANIFEST_FILE):
        assert (data.directory / name).exists()
    assert not list(tmp_path.glob(".data.partial"))


def test_preferences_follow_the_recipe(tmp_path: Path) -> None:
    uncorrected = write_dataset(tmp_path / "uncorrected", corrected=False)
    data = build(
        uncorrected,
        tmp_path / "a",
        dataset.load_manifest(uncorrected),
        TargetSettings(),
    )
    assert data.counts["preference"] == 0

    corrected = write_dataset(tmp_path / "corrected")
    data = build(
        corrected,
        tmp_path / "b",
        dataset.load_manifest(corrected),
        TargetSettings(preference_outcomes=("Missed",)),
    )
    assert data.counts["preference"] == 0


def test_a_reply_that_breaks_the_schema_stops_the_build(tmp_path: Path) -> None:
    strict = copy.deepcopy(SCHEMA)
    strict["properties"]["overallConfidence"] = {"type": "integer"}
    root = write_dataset(tmp_path / "d", schema=strict)
    with pytest.raises(dataset.DatasetError, match="breaks the output schema at overallConfidence"):
        build(root, tmp_path / "data", dataset.load_manifest(root), TargetSettings())
    assert not (tmp_path / "data").exists()
    assert not list(tmp_path.glob(".data.partial"))


def test_build_refuses_a_used_directory(tmp_path: Path, dataset_dir: Path) -> None:
    out = tmp_path / "data"
    out.mkdir()
    (out / "keep.txt").write_text("x")
    with pytest.raises(FileExistsError):
        build(dataset_dir, out, dataset.load_manifest(dataset_dir), TargetSettings())


def test_built_data_is_checked_before_training(tmp_path: Path, dataset_dir: Path) -> None:
    data = build(
        dataset_dir, tmp_path / "data", dataset.load_manifest(dataset_dir), TargetSettings()
    )
    path = data.directory / SFT_TRAIN_FILE
    path.write_bytes(path.read_bytes().replace(b"Juniper", b"Jupiter"))
    with pytest.raises(dataset.DatasetError, match="has been changed"):
        built.verify(data.directory)
