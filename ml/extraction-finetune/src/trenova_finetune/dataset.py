"""Read a dataset rendered by `trenova ai training-export render`.

Every file is checked against the checksums in the dataset manifest before it is
used, so training never runs on a file that was edited or truncated after it was
rendered. The dataset holds the production prompt and the raw confirmed and
predicted values; turning those into training targets is `targets.py`'s job.
"""

from __future__ import annotations

import hashlib
import json
from collections.abc import Iterator
from dataclasses import dataclass
from pathlib import Path
from typing import Any

DATASET_FORMAT = "trenova.extraction-dataset/v2"
MANIFEST_FILE = "dataset-manifest.json"
SCHEMA_FILE = "schema.json"
TRAIN_FILE = "examples-train.jsonl"
VALIDATION_FILE = "examples-validation.jsonl"
EVALUATION_FILE = "eval-validation.jsonl"
REQUIRED_FILES = (TRAIN_FILE, VALIDATION_FILE, EVALUATION_FILE, SCHEMA_FILE)
STRUCTURED_OUTPUT_MODES = ("JSONSchema", "JSONMode", "Prompted")
SPLIT_TRAIN = "train"
SPLIT_VALIDATION = "validation"
PROMPT_ROLES = ("system", "user")
REPLY_ROLES = ("assistant",)
_CHUNK = 1 << 20


class DatasetError(ValueError):
    """The dataset is incomplete, altered or malformed."""


@dataclass(frozen=True)
class DatasetFile:
    name: str
    records: int
    bytes: int
    sha256: str


@dataclass(frozen=True)
class DatasetManifest:
    export_id: str
    export_manifest_sha256: str
    prompt_sha256: str
    structured_output_mode: str
    schema_name: str
    temperature: float | None
    top_p: float | None
    page_limit: int
    field_keys: tuple[str, ...]
    counts: dict[str, int]
    files: dict[str, DatasetFile]
    sha256: str


@dataclass(frozen=True)
class Page:
    number: int
    text: str


@dataclass(frozen=True)
class Stop:
    role: str
    name: str = ""
    address_line1: str = ""
    address_line2: str = ""
    city: str = ""
    state: str = ""
    postal_code: str = ""
    date: str = ""
    time_window: str = ""
    appointment_required: bool = False


@dataclass(frozen=True)
class Snapshot:
    fields: dict[str, str]
    stops: tuple[Stop, ...]


@dataclass(frozen=True)
class Example:
    """One rendered example: the prompt, what the model sees, and the values to learn from."""

    id: str
    split: str
    document_kind: str
    prompt: list[dict[str, str]]
    visible_pages: tuple[Page, ...]
    target: Snapshot
    prediction: Snapshot | None
    outcomes: dict[str, str]


def load_manifest(directory: str | Path) -> DatasetManifest:
    """Read the dataset manifest and check it is one this pipeline understands."""
    path = Path(directory) / MANIFEST_FILE
    try:
        raw_bytes = path.read_bytes()
    except OSError as err:
        raise DatasetError(f"cannot read {path}: {err}") from err
    try:
        raw = json.loads(raw_bytes)
    except json.JSONDecodeError as err:
        raise DatasetError(f"{path} is not valid JSON: {err}") from err

    if raw.get("format") != DATASET_FORMAT:
        raise DatasetError(f"{path} is not a {DATASET_FORMAT} manifest; render the export again")
    mode = raw.get("structuredOutputMode")
    if mode not in STRUCTURED_OUTPUT_MODES:
        raise DatasetError(f"{path} names an unknown structured output mode {mode!r}")
    page_limit = raw.get("pageLimit")
    if not isinstance(page_limit, int) or page_limit <= 0:
        raise DatasetError(f"{path} does not say how much of each page the model sees")
    field_keys = raw.get("fieldKeys")
    if not isinstance(field_keys, list) or not field_keys:
        raise DatasetError(f"{path} does not list the reply's field keys")

    files: dict[str, DatasetFile] = {}
    for entry in raw.get("files") or []:
        try:
            item = DatasetFile(
                name=str(entry["name"]),
                records=int(entry["records"]),
                bytes=int(entry["bytes"]),
                sha256=str(entry["sha256"]),
            )
        except (KeyError, TypeError, ValueError) as err:
            raise DatasetError(f"{path} lists a file without its size or checksum") from err
        if Path(item.name).name != item.name:
            raise DatasetError(f"{path} lists a file outside the dataset: {item.name!r}")
        files[item.name] = item

    missing = [name for name in REQUIRED_FILES if name not in files]
    if missing:
        raise DatasetError(f"{path} does not list {', '.join(missing)}")

    return DatasetManifest(
        export_id=str(raw.get("exportId", "")),
        export_manifest_sha256=str(raw.get("exportManifestSha256", "")),
        prompt_sha256=str(raw.get("promptSha256", "")),
        structured_output_mode=mode,
        schema_name=str(raw.get("schemaName", "")),
        temperature=_optional_float(raw.get("temperature")),
        top_p=_optional_float(raw.get("topP")),
        page_limit=page_limit,
        field_keys=tuple(str(key) for key in field_keys),
        counts={key: int(value) for key, value in (raw.get("counts") or {}).items()},
        files=files,
        sha256=hashlib.sha256(raw_bytes).hexdigest(),
    )


def verify_files(directory: str | Path, files: dict[str, DatasetFile]) -> None:
    """Check every listed file's size, checksum and record count."""
    root = Path(directory)
    for item in files.values():
        path = root / item.name
        digest = hashlib.sha256()
        size = 0
        lines = 0
        try:
            with path.open("rb") as handle:
                while chunk := handle.read(_CHUNK):
                    digest.update(chunk)
                    size += len(chunk)
                    lines += chunk.count(b"\n")
        except OSError as err:
            raise DatasetError(f"cannot read {path}: {err}") from err

        if size != item.bytes or digest.hexdigest() != item.sha256:
            raise DatasetError(f"{path} does not match its manifest; it has been changed")
        if item.name.endswith(".jsonl") and lines != item.records:
            raise DatasetError(f"{path} holds {lines} records, the manifest says {item.records}")


def verify(directory: str | Path, manifest: DatasetManifest) -> None:
    verify_files(directory, manifest.files)


def read_jsonl(path: str | Path) -> Iterator[dict[str, Any]]:
    """Yield each record of a JSON Lines file."""
    source = Path(path)
    with source.open("r", encoding="utf-8") as handle:
        for number, line in enumerate(handle, start=1):
            text = line.strip()
            if not text:
                continue
            try:
                record = json.loads(text)
            except json.JSONDecodeError as err:
                raise DatasetError(f"{source}:{number} is not valid JSON: {err}") from err
            if not isinstance(record, dict):
                raise DatasetError(f"{source}:{number} is not a JSON object")
            yield record


def messages(record: dict[str, Any], key: str, roles: tuple[str, ...], where: str) -> list:
    """Return a record's chat messages after checking their roles and content."""
    value = record.get(key)
    if not isinstance(value, list) or len(value) != len(roles):
        raise DatasetError(f"{where}: {key} must hold {len(roles)} message(s)")
    for message, role in zip(value, roles, strict=True):
        if not isinstance(message, dict) or message.get("role") != role:
            raise DatasetError(f"{where}: {key} must be {', '.join(roles)} messages in order")
        if not isinstance(message.get("content"), str) or not message["content"]:
            raise DatasetError(f"{where}: a {role} message in {key} is empty")
    return value


def record_id(record: dict[str, Any], where: str) -> str:
    identifier = record.get("id")
    if not isinstance(identifier, str) or not identifier:
        raise DatasetError(f"{where}: the record has no id")
    return identifier


def load_examples(directory: str | Path, split: str) -> list[Example]:
    """Return the validated examples of one split."""
    name = TRAIN_FILE if split == SPLIT_TRAIN else VALIDATION_FILE
    examples = []
    seen: set[str] = set()
    for index, record in enumerate(read_jsonl(Path(directory) / name), start=1):
        where = f"{name}:{index}"
        identifier = record_id(record, where)
        if identifier in seen:
            raise DatasetError(f"{where}: example {identifier} appears twice")
        seen.add(identifier)
        if record.get("split") != split:
            raise DatasetError(f"{where}: example {identifier} belongs to another split")
        target = _snapshot(record.get("target"), where, "target")
        if target is None:
            raise DatasetError(f"{where}: example {identifier} has no confirmed values")
        examples.append(
            Example(
                id=identifier,
                split=split,
                document_kind=str(record.get("documentKind") or ""),
                prompt=messages(record, "prompt", PROMPT_ROLES, where),
                visible_pages=_pages(record.get("visiblePages"), where),
                target=target,
                prediction=_snapshot(record.get("prediction"), where, "prediction"),
                outcomes=_outcomes(record.get("outcomes"), where),
            )
        )
    return examples


def load_evaluation_prompts(directory: str | Path) -> list[tuple[str, list]]:
    """Return (id, prompt) for every validation example the scorer will compare."""
    prompts = []
    seen: set[str] = set()
    for index, record in enumerate(read_jsonl(Path(directory) / EVALUATION_FILE), start=1):
        where = f"{EVALUATION_FILE}:{index}"
        identifier = record_id(record, where)
        if identifier in seen:
            raise DatasetError(f"{where}: example {identifier} appears twice")
        seen.add(identifier)
        prompts.append((identifier, messages(record, "prompt", PROMPT_ROLES, where)))
    return prompts


def load_schema(directory: str | Path) -> dict[str, Any]:
    """Return the JSON Schema the served model must answer in."""
    path = Path(directory) / SCHEMA_FILE
    try:
        schema = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as err:
        raise DatasetError(f"cannot read the output schema {path}: {err}") from err
    if not isinstance(schema, dict) or schema.get("type") != "object":
        raise DatasetError(f"{path} is not an object schema")
    return schema


def _optional_float(value: Any) -> float | None:
    return None if value is None else float(value)


def _pages(value: Any, where: str) -> tuple[Page, ...]:
    if not isinstance(value, list) or not value:
        raise DatasetError(f"{where}: the example has no page text")
    pages = []
    for page in value:
        if not isinstance(page, dict) or not isinstance(page.get("text"), str):
            raise DatasetError(f"{where}: a page has no text")
        pages.append(Page(number=int(page.get("number") or 0), text=page["text"]))
    return tuple(pages)


def _snapshot(value: Any, where: str, label: str) -> Snapshot | None:
    if value is None:
        return None
    if not isinstance(value, dict):
        raise DatasetError(f"{where}: {label} is not an object")
    fields = value.get("fields") or {}
    if not isinstance(fields, dict) or not all(
        isinstance(key, str) and isinstance(text, str) for key, text in fields.items()
    ):
        raise DatasetError(f"{where}: {label}.fields must map names to text")
    stops = []
    for stop in value.get("stops") or []:
        if not isinstance(stop, dict) or not isinstance(stop.get("role"), str):
            raise DatasetError(f"{where}: a stop in {label} has no role")
        stops.append(
            Stop(
                role=stop["role"],
                name=str(stop.get("name") or ""),
                address_line1=str(stop.get("addressLine1") or ""),
                address_line2=str(stop.get("addressLine2") or ""),
                city=str(stop.get("city") or ""),
                state=str(stop.get("state") or ""),
                postal_code=str(stop.get("postalCode") or ""),
                date=str(stop.get("date") or ""),
                time_window=str(stop.get("timeWindow") or ""),
                appointment_required=bool(stop.get("appointmentRequired", False)),
            )
        )
    return Snapshot(fields=dict(fields), stops=tuple(stops))


def _outcomes(value: Any, where: str) -> dict[str, str]:
    if value is None:
        return {}
    if not isinstance(value, dict) or not all(
        isinstance(key, str) and isinstance(outcome, str) for key, outcome in value.items()
    ):
        raise DatasetError(f"{where}: outcomes must map field keys to outcomes")
    return dict(value)
