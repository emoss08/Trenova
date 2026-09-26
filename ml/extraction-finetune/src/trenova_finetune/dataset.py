"""Read a dataset rendered by `trenova ai training-export render`.

Every file is checked against the checksums in the dataset manifest before it is
used, so training never runs on a file that was edited or truncated after it was
rendered.
"""

from __future__ import annotations

import hashlib
import json
from collections.abc import Iterator
from dataclasses import dataclass
from pathlib import Path
from typing import Any

DATASET_FORMAT = "trenova.extraction-dataset/v1"
MANIFEST_FILE = "dataset-manifest.json"
SCHEMA_FILE = "schema.json"
SFT_TRAIN_FILE = "sft-train.jsonl"
SFT_VALIDATION_FILE = "sft-validation.jsonl"
PREFERENCE_FILE = "preference-train.jsonl"
EVALUATION_FILE = "eval-validation.jsonl"
REQUIRED_FILES = (
    SFT_TRAIN_FILE,
    SFT_VALIDATION_FILE,
    PREFERENCE_FILE,
    EVALUATION_FILE,
    SCHEMA_FILE,
)
STRUCTURED_OUTPUT_MODES = ("JSONSchema", "JSONMode", "Prompted")
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
    keep_unverified: bool
    counts: dict[str, int]
    files: dict[str, DatasetFile]
    sha256: str


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
        raise DatasetError(f"{path} is not a {DATASET_FORMAT} manifest")
    mode = raw.get("structuredOutputMode")
    if mode not in STRUCTURED_OUTPUT_MODES:
        raise DatasetError(f"{path} names an unknown structured output mode {mode!r}")

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
        keep_unverified=bool(raw.get("keepUnverified", False)),
        counts={key: int(value) for key, value in (raw.get("counts") or {}).items()},
        files=files,
        sha256=hashlib.sha256(raw_bytes).hexdigest(),
    )


def verify(directory: str | Path, manifest: DatasetManifest) -> None:
    """Check every listed file's size, checksum and record count."""
    root = Path(directory)
    for item in manifest.files.values():
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
            raise DatasetError(f"{path} does not match the dataset manifest; it has been changed")
        if item.name.endswith(".jsonl") and lines != item.records:
            raise DatasetError(f"{path} holds {lines} records, the manifest says {item.records}")


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


def _optional_float(value: Any) -> float | None:
    return None if value is None else float(value)


def _messages(record: dict[str, Any], key: str, roles: tuple[str, ...], where: str) -> list:
    messages = record.get(key)
    if not isinstance(messages, list) or len(messages) != len(roles):
        raise DatasetError(f"{where}: {key} must hold {len(roles)} message(s)")
    for message, role in zip(messages, roles, strict=True):
        if not isinstance(message, dict) or message.get("role") != role:
            raise DatasetError(f"{where}: {key} must be {', '.join(roles)} messages in order")
        if not isinstance(message.get("content"), str) or not message["content"]:
            raise DatasetError(f"{where}: a {role} message in {key} is empty")
    return messages


def _record_id(record: dict[str, Any], where: str) -> str:
    identifier = record.get("id")
    if not isinstance(identifier, str) or not identifier:
        raise DatasetError(f"{where}: the record has no id")
    return identifier


def _assistant_json(messages: list, where: str) -> None:
    try:
        json.loads(messages[0]["content"])
    except json.JSONDecodeError as err:
        raise DatasetError(f"{where}: the assistant message is not JSON") from err


PROMPT_ROLES = ("system", "user")
REPLY_ROLES = ("assistant",)


def load_sft(directory: str | Path) -> tuple[list[dict], list[dict]]:
    """Return validated prompt-completion records for training and validation."""
    root = Path(directory)
    splits = []
    for name in (SFT_TRAIN_FILE, SFT_VALIDATION_FILE):
        records = []
        for index, record in enumerate(read_jsonl(root / name), start=1):
            where = f"{name}:{index}"
            records.append(
                {
                    "id": _record_id(record, where),
                    "prompt": _messages(record, "prompt", PROMPT_ROLES, where),
                    "completion": _assistant_checked(record, where),
                }
            )
        splits.append(records)
    return splits[0], splits[1]


def _assistant_checked(record: dict[str, Any], where: str) -> list:
    completion = _messages(record, "completion", REPLY_ROLES, where)
    _assistant_json(completion, where)
    return completion


def load_preferences(directory: str | Path) -> list[dict]:
    """Return validated preference pairs: the confirmed reply over the production one."""
    records = []
    for index, record in enumerate(read_jsonl(Path(directory) / PREFERENCE_FILE), start=1):
        where = f"{PREFERENCE_FILE}:{index}"
        chosen = _messages(record, "chosen", REPLY_ROLES, where)
        rejected = _messages(record, "rejected", REPLY_ROLES, where)
        _assistant_json(chosen, where)
        _assistant_json(rejected, where)
        records.append(
            {
                "id": _record_id(record, where),
                "prompt": _messages(record, "prompt", PROMPT_ROLES, where),
                "chosen": chosen,
                "rejected": rejected,
            }
        )
    return records


def load_evaluation_prompts(directory: str | Path) -> list[tuple[str, list]]:
    """Return (id, prompt) for every validation example the scorer will compare."""
    prompts = []
    seen: set[str] = set()
    for index, record in enumerate(read_jsonl(Path(directory) / EVALUATION_FILE), start=1):
        where = f"{EVALUATION_FILE}:{index}"
        identifier = _record_id(record, where)
        if identifier in seen:
            raise DatasetError(f"{where}: example {identifier} appears twice")
        seen.add(identifier)
        prompts.append((identifier, _messages(record, "prompt", PROMPT_ROLES, where)))
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
