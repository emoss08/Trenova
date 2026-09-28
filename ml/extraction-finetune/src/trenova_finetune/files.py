"""Writing files so a reader never sees one half-written."""

from __future__ import annotations

import json
import os
import tempfile
from collections.abc import Iterable
from pathlib import Path
from typing import Any


def write_text_atomically(path: Path, chunks: Iterable[str]) -> None:
    """Write `chunks` to a temporary file beside `path`, then rename it into place."""
    path.parent.mkdir(parents=True, exist_ok=True)
    handle, temporary = tempfile.mkstemp(
        prefix=f".{path.stem}-", suffix=path.suffix, dir=path.parent
    )
    try:
        with os.fdopen(handle, "w", encoding="utf-8") as stream:
            for chunk in chunks:
                stream.write(chunk)
        os.replace(temporary, path)
    except BaseException:
        Path(temporary).unlink(missing_ok=True)
        raise


def write_json(path: Path, value: Any) -> None:
    write_text_atomically(path, (json.dumps(value, indent=2, sort_keys=True), "\n"))


def write_jsonl(path: Path, records: Iterable[dict[str, Any]]) -> None:
    write_text_atomically(
        path, (json.dumps(record, ensure_ascii=False) + "\n" for record in records)
    )
