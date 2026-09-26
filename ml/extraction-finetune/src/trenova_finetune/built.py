"""Read the training data `targets.build` wrote, after checking it against its manifest."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from .dataset import (
    PROMPT_ROLES,
    REPLY_ROLES,
    DatasetError,
    DatasetFile,
    messages,
    read_jsonl,
    record_id,
    verify_files,
)
from .targets import (
    BUILT_FILES,
    BUILT_FORMAT,
    BUILT_MANIFEST_FILE,
    PREFERENCE_FILE,
    SFT_TRAIN_FILE,
    SFT_VALIDATION_FILE,
)


def verify(directory: str | Path) -> dict[str, Any]:
    path = Path(directory) / BUILT_MANIFEST_FILE
    try:
        manifest = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as err:
        raise DatasetError(f"cannot read {path}: {err}") from err
    if manifest.get("format") != BUILT_FORMAT:
        raise DatasetError(f"{path} is not a {BUILT_FORMAT} manifest")
    try:
        files = {
            entry["name"]: DatasetFile(
                name=str(entry["name"]),
                records=int(entry["records"]),
                bytes=int(entry["bytes"]),
                sha256=str(entry["sha256"]),
            )
            for entry in manifest.get("files") or []
        }
    except (KeyError, TypeError, ValueError) as err:
        raise DatasetError(f"{path} lists a file without its size or checksum") from err
    if set(files) != set(BUILT_FILES):
        raise DatasetError(f"{path} must list exactly {', '.join(BUILT_FILES)}")
    verify_files(directory, files)
    return manifest


def _json_reply(value: list, where: str) -> None:
    try:
        json.loads(value[0]["content"])
    except json.JSONDecodeError as err:
        raise DatasetError(f"{where}: the assistant message is not JSON") from err


def load_sft(directory: str | Path) -> tuple[list[dict], list[dict]]:
    """Return prompt-completion records for training and validation."""
    root = Path(directory)
    splits = []
    for name in (SFT_TRAIN_FILE, SFT_VALIDATION_FILE):
        records = []
        for index, record in enumerate(read_jsonl(root / name), start=1):
            where = f"{name}:{index}"
            completion = messages(record, "completion", REPLY_ROLES, where)
            _json_reply(completion, where)
            records.append(
                {
                    "id": record_id(record, where),
                    "prompt": messages(record, "prompt", PROMPT_ROLES, where),
                    "completion": completion,
                }
            )
        splits.append(records)
    return splits[0], splits[1]


def load_preferences(directory: str | Path) -> list[dict]:
    """Return preference pairs: the confirmed reply over the production one."""
    records = []
    for index, record in enumerate(read_jsonl(Path(directory) / PREFERENCE_FILE), start=1):
        where = f"{PREFERENCE_FILE}:{index}"
        chosen = messages(record, "chosen", REPLY_ROLES, where)
        rejected = messages(record, "rejected", REPLY_ROLES, where)
        _json_reply(chosen, where)
        _json_reply(rejected, where)
        records.append(
            {
                "id": record_id(record, where),
                "prompt": messages(record, "prompt", PROMPT_ROLES, where),
                "chosen": chosen,
                "rejected": rejected,
            }
        )
    return records
