"""The training recipe: turn rendered examples into the replies the model learns to give.

The rendered dataset holds what Trenova knows for certain: the exact prompt, the page
text the model sees, what a person confirmed and what the production model said. How
that becomes a supervised reply, and which examples become preference pairs, is a
choice of this recipe, set in the `targets` section of the configuration. Every reply
built here is checked against the production reply schema before it is written.
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import shutil
from collections.abc import Iterable
from dataclasses import dataclass
from decimal import Decimal, InvalidOperation
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator

from . import dataset
from .config import TargetSettings
from .dataset import DatasetError, DatasetManifest, Example, Page, Snapshot, Stop

BUILT_FORMAT = "trenova.extraction-training-data/v1"
BUILT_MANIFEST_FILE = "training-data.json"
SFT_TRAIN_FILE = "sft-train.jsonl"
SFT_VALIDATION_FILE = "sft-validation.jsonl"
PREFERENCE_FILE = "preference-train.jsonl"
BUILT_FILES = (SFT_TRAIN_FILE, SFT_VALIDATION_FILE, PREFERENCE_FILE)

_CAMEL_BOUNDARY = re.compile(r"(?<=[a-z0-9])(?=[A-Z])")
_MONEY_NOISE = str.maketrans("", "", "$,% " + chr(0xA0))


@dataclass(frozen=True)
class ReplyShape:
    """The parts of the reply schema a builder needs: key order, allowed keys and limits."""

    top_order: tuple[str, ...]
    field_order: tuple[str, ...]
    stop_order: tuple[str, ...]
    field_keys: tuple[str, ...]
    max_fields: int
    max_stops: int
    field_limits: dict[str, int]
    stop_limits: dict[str, int]


def reply_shape(schema: dict[str, Any], field_keys: Iterable[str]) -> ReplyShape:
    """Read the reply's structure from its schema and check it against the dataset's key list."""
    try:
        properties = schema["properties"]
        fields = properties["fields"]
        stops = properties["stops"]
        field_item = fields["items"]
        stop_item = stops["items"]
        enum = tuple(field_item["properties"]["key"]["enum"])
        shape = ReplyShape(
            top_order=tuple(schema["required"]),
            field_order=tuple(field_item["required"]),
            stop_order=tuple(stop_item["required"]),
            field_keys=enum,
            max_fields=int(fields["maxItems"]),
            max_stops=int(stops["maxItems"]),
            field_limits=_limits(field_item["properties"]),
            stop_limits=_limits(stop_item["properties"]),
        )
    except (KeyError, TypeError, ValueError) as err:
        raise DatasetError(f"the output schema is not the extraction reply schema: {err}") from err
    if enum != tuple(field_keys):
        raise DatasetError("the output schema's field keys differ from the dataset manifest's")
    return shape


def _limits(properties: dict[str, Any]) -> dict[str, int]:
    return {
        name: int(spec["maxLength"])
        for name, spec in properties.items()
        if isinstance(spec, dict) and "maxLength" in spec
    }


def humanize(key: str) -> str:
    """`loadNumber` → `Load Number`, the label a person would read."""
    words = _CAMEL_BOUNDARY.sub(" ", key).split()
    return " ".join(word[:1].upper() + word[1:] for word in words)


def evidence_needles(value: str) -> list[str]:
    """What to look for on the page: the value, and a money value with thousands separators."""
    text = value.strip().lower()
    if not text:
        return []
    needles = [text]
    try:
        amount = Decimal(text.translate(_MONEY_NOISE))
    except InvalidOperation:
        return needles
    if not amount.is_finite():
        return needles
    whole = int(amount)
    cents = abs(amount - whole)
    grouped = f"{whole:,}"
    if cents:
        grouped += "." + f"{cents:.2f}".split(".")[1]
    if grouped not in needles:
        needles.append(grouped)
    return needles


def locate_evidence(value: str, pages: Iterable[Page], context: int) -> tuple[int, str]:
    """The page a value appears on, and the words around it."""
    for needle in evidence_needles(value):
        for page in pages:
            haystack = page.text.lower()
            if len(haystack) != len(page.text):
                continue
            index = haystack.find(needle)
            if index < 0:
                continue
            start = max(0, index - context)
            end = min(len(page.text), index + len(needle) + context)
            return page.number, " ".join(page.text[start:end].split())
    return 0, ""


class ReplyBuilder:
    """Builds replies in the production wire format from a confirmed or predicted snapshot."""

    def __init__(self, shape: ReplyShape, recipe: TargetSettings) -> None:
        self.shape = shape
        self.recipe = recipe

    def build(
        self,
        primary: Snapshot | None,
        *,
        supplement: Snapshot | None,
        confidence: float,
        document_kind: str,
        pages: tuple[Page, ...],
    ) -> dict[str, Any]:
        fields: list[dict[str, Any]] = []
        if primary is not None:
            for key in self.shape.field_keys:
                if len(fields) >= self.shape.max_fields:
                    break
                value, field_confidence = self._field_value(key, primary, supplement, confidence)
                if value:
                    fields.append(self._field(key, value, field_confidence, pages))

        stops: list[dict[str, Any]] = []
        if primary is not None:
            positions: dict[str, int] = {}
            for index, stop in enumerate(primary.stops):
                if len(stops) >= self.shape.max_stops:
                    break
                position = positions.get(stop.role, 0)
                positions[stop.role] = position + 1
                time_window = stop.time_window or _supplemental_window(
                    supplement, stop.role, position
                )
                stops.append(self._stop(index + 1, stop, time_window, confidence, pages))

        top = {
            "documentKind": document_kind or self.recipe.default_document_kind,
            "overallConfidence": self.recipe.overall_confidence,
            "reviewStatus": self.recipe.review_status,
            "missingFields": [],
            "signals": [],
            "fields": fields,
            "stops": stops,
            "conflicts": [],
        }
        return _ordered(top, self.shape.top_order)

    def _field_value(
        self,
        key: str,
        primary: Snapshot,
        supplement: Snapshot | None,
        confidence: float,
    ) -> tuple[str, float]:
        value = primary.fields.get(key, "").strip()
        if value:
            return value, confidence
        if supplement is not None:
            value = supplement.fields.get(key, "").strip()
            if value:
                return value, self.recipe.unverified_confidence
        return "", confidence

    def _field(self, key: str, value: str, confidence: float, pages: tuple[Page, ...]) -> dict:
        page, excerpt = locate_evidence(value, pages, self.recipe.evidence_context_chars)
        field = {
            "key": key,
            "label": humanize(key),
            "value": value,
            "confidence": confidence,
            "evidenceExcerpt": excerpt,
            "pageNumber": page,
            "reviewRequired": False,
            "conflict": False,
            "source": self.recipe.source,
            "alternativeValues": [],
        }
        return _ordered(_bounded(field, self.shape.field_limits), self.shape.field_order)

    def _stop(
        self,
        sequence: int,
        stop: Stop,
        time_window: str,
        confidence: float,
        pages: tuple[Page, ...],
    ) -> dict:
        page, excerpt = locate_evidence(stop.name, pages, self.recipe.evidence_context_chars)
        reply = {
            "sequence": sequence,
            "role": stop.role,
            "name": stop.name,
            "addressLine1": stop.address_line1,
            "addressLine2": stop.address_line2,
            "city": stop.city,
            "state": stop.state,
            "postalCode": stop.postal_code,
            "date": stop.date,
            "timeWindow": time_window,
            "appointmentRequired": stop.appointment_required,
            "pageNumber": page,
            "evidenceExcerpt": excerpt,
            "confidence": confidence,
            "reviewRequired": False,
            "source": self.recipe.source,
        }
        return _ordered(_bounded(reply, self.shape.stop_limits), self.shape.stop_order)


def _supplemental_window(supplement: Snapshot | None, role: str, position: int) -> str:
    if supplement is None:
        return ""
    same_role = [stop for stop in supplement.stops if stop.role == role]
    return same_role[position].time_window if position < len(same_role) else ""


def _bounded(values: dict[str, Any], limits: dict[str, int]) -> dict[str, Any]:
    return {
        key: value[: limits[key]] if isinstance(value, str) and key in limits else value
        for key, value in values.items()
    }


def _ordered(values: dict[str, Any], order: tuple[str, ...]) -> dict[str, Any]:
    missing = [key for key in order if key not in values]
    if missing:
        raise DatasetError(f"the reply schema requires {', '.join(missing)}, which is not built")
    return {key: values[key] for key in order}


def reply_text(reply: dict[str, Any]) -> str:
    return json.dumps(reply, ensure_ascii=False, separators=(",", ":"))


def wants_preference(example: Example, recipe: TargetSettings) -> bool:
    return example.prediction is not None and any(
        outcome in recipe.preference_outcomes for outcome in example.outcomes.values()
    )


@dataclass(frozen=True)
class BuiltData:
    directory: Path
    counts: dict[str, int]
    sha256: str


def build(
    dataset_dir: Path,
    output: Path,
    manifest: DatasetManifest,
    recipe: TargetSettings,
) -> BuiltData:
    """Write SFT and preference files for one recipe; the directory appears whole or not at all."""
    if output.exists() and any(output.iterdir()):
        raise FileExistsError(f"{output} already holds files")
    schema = dataset.load_schema(dataset_dir)
    builder = ReplyBuilder(reply_shape(schema, manifest.field_keys), recipe)
    validator = Draft202012Validator(schema)

    staging = output.with_name(f".{output.name}.partial")
    if staging.exists():
        shutil.rmtree(staging)
    staging.mkdir(parents=True)

    counts = {"train": 0, "validation": 0, "preference": 0}
    files: list[dict[str, Any]] = []
    try:
        with (
            _Writer(staging / SFT_TRAIN_FILE) as train,
            _Writer(staging / SFT_VALIDATION_FILE) as validation,
            _Writer(staging / PREFERENCE_FILE) as preference,
        ):
            for split, sink in (
                (dataset.SPLIT_TRAIN, train),
                (dataset.SPLIT_VALIDATION, validation),
            ):
                for example in dataset.load_examples(dataset_dir, split):
                    chosen = _checked(
                        validator,
                        example.id,
                        builder.build(
                            example.target,
                            supplement=example.prediction if recipe.keep_unverified else None,
                            confidence=recipe.verified_confidence,
                            document_kind=example.document_kind,
                            pages=example.visible_pages,
                        ),
                    )
                    completion = [{"role": "assistant", "content": reply_text(chosen)}]
                    sink.write(
                        {"id": example.id, "prompt": example.prompt, "completion": completion}
                    )
                    counts[split] += 1

                    if split != dataset.SPLIT_TRAIN or not wants_preference(example, recipe):
                        continue
                    rejected = _checked(
                        validator,
                        example.id,
                        builder.build(
                            example.prediction,
                            supplement=None,
                            confidence=recipe.unverified_confidence,
                            document_kind=example.document_kind,
                            pages=example.visible_pages,
                        ),
                    )
                    if rejected == chosen:
                        continue
                    preference.write(
                        {
                            "id": example.id,
                            "prompt": example.prompt,
                            "chosen": completion,
                            "rejected": [{"role": "assistant", "content": reply_text(rejected)}],
                        }
                    )
                    counts["preference"] += 1
        for writer in (train, validation, preference):
            files.append(writer.entry())

        built_manifest = {
            "format": BUILT_FORMAT,
            "datasetManifestSha256": manifest.sha256,
            "recipe": recipe.model_dump(mode="json"),
            "counts": counts,
            "files": files,
        }
        encoded = json.dumps(built_manifest, indent=2, sort_keys=True).encode() + b"\n"
        (staging / BUILT_MANIFEST_FILE).write_bytes(encoded)
        os.replace(staging, output)
    except BaseException:
        shutil.rmtree(staging, ignore_errors=True)
        raise

    return BuiltData(directory=output, counts=counts, sha256=hashlib.sha256(encoded).hexdigest())


def _checked(validator: Draft202012Validator, identifier: str, reply: dict) -> dict:
    error = next(iter(validator.iter_errors(reply)), None)
    if error is not None:
        path = "/".join(str(part) for part in error.absolute_path) or "(root)"
        raise DatasetError(
            f"the reply built for example {identifier} breaks the output schema at {path}: "
            f"{error.message}"
        )
    return reply


class _Writer:
    def __init__(self, path: Path) -> None:
        self.path = path
        self.digest = hashlib.sha256()
        self.size = 0
        self.records = 0
        self.stream = None

    def __enter__(self) -> _Writer:
        self.stream = self.path.open("wb")
        return self

    def __exit__(self, *_: object) -> None:
        if self.stream is not None:
            self.stream.close()

    def write(self, record: dict) -> None:
        line = json.dumps(record, ensure_ascii=False).encode() + b"\n"
        assert self.stream is not None
        self.stream.write(line)
        self.digest.update(line)
        self.size += len(line)
        self.records += 1

    def entry(self) -> dict[str, Any]:
        return {
            "name": self.path.name,
            "records": self.records,
            "bytes": self.size,
            "sha256": self.digest.hexdigest(),
        }
