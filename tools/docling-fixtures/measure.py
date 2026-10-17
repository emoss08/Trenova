"""Converts files through a running docling-serve and reports what each cost.

It sends the request the worker's Docling adapter sends
(services/tms/internal/infrastructure/docling/client.go): an async multipart
submit with to_formats=json, the OCR preset, languages and table mode, image
export off; then status polls and the result fetch. For every file it records
the page count, the wall time from submit to result, docling-serve's own
processing time, seconds per page, and the container's peak memory while that
file converted.

    python3 -I tools/docling-fixtures/measure.py \\
        --url http://127.0.0.1:5001 --api-key "$DOCLING_SERVE_API_KEY" \\
        --container docling --out results/ file1.pdf file2.jpg ...

--container names the docling-serve container; its cgroup's peak-memory
counter is reset before each file and read after it (cgroup v1
memory.max_usage_in_bytes, or v2 memory.peak), which needs root on the Docker
host. Without it, memory is not reported.

--out keeps each file's DoclingDocument JSON beside a summary.csv.
"""

from __future__ import annotations

import argparse
import csv
import json
import mimetypes
import subprocess
import sys
import time
from pathlib import Path

import requests

CONTENT_TYPES = {
    ".pdf": "application/pdf",
    ".jpg": "image/jpeg",
    ".jpeg": "image/jpeg",
    ".png": "image/png",
    ".tif": "image/tiff",
    ".tiff": "image/tiff",
    ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    ".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    ".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
    ".eml": "message/rfc822",
    ".html": "text/html",
}


def peak_memory_file(container: str) -> Path | None:
    container_id = subprocess.run(
        ["docker", "inspect", "-f", "{{.Id}}", container],
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
    for candidate in (
        Path(f"/sys/fs/cgroup/memory/docker/{container_id}/memory.max_usage_in_bytes"),
        Path(f"/sys/fs/cgroup/system.slice/docker-{container_id}.scope/memory.peak"),
        Path(f"/sys/fs/cgroup/docker/{container_id}/memory.peak"),
    ):
        if candidate.exists():
            return candidate
    return None


def reset_peak(path: Path | None) -> None:
    if path is None:
        return
    try:
        path.write_text("0")
    except OSError:
        pass


def read_peak(path: Path | None) -> int | None:
    if path is None:
        return None
    return int(path.read_text().split()[0])


def form_fields(args: argparse.Namespace, do_ocr: bool) -> list[tuple[str, str]]:
    fields = [
        ("to_formats", "json"),
        ("do_ocr", str(do_ocr).lower()),
        ("ocr_preset", args.ocr_preset),
        ("table_mode", args.table_mode),
        ("image_export_mode", "placeholder"),
        ("include_images", "false"),
        ("abort_on_error", "false"),
    ]
    fields += [("ocr_lang", language) for language in args.ocr_lang]
    return fields


def convert(session: requests.Session, args: argparse.Namespace, path: Path) -> tuple[dict, float]:
    content_type = CONTENT_TYPES.get(path.suffix.lower()) or mimetypes.guess_type(path.name)[0]
    started = time.monotonic()
    with path.open("rb") as handle:
        submitted = session.post(
            f"{args.url}/v1/convert/file/async",
            data=form_fields(args, not args.no_ocr),
            files={"files": (path.name, handle, content_type or "application/octet-stream")},
            timeout=args.request_timeout,
        )
    submitted.raise_for_status()
    task_id = submitted.json()["task_id"]

    while True:
        status = session.get(
            f"{args.url}/v1/status/poll/{task_id}",
            params={"wait": 5},
            timeout=args.request_timeout,
        )
        status.raise_for_status()
        state = status.json()["task_status"]
        if state in ("success", "partial_success", "failure", "skipped"):
            break

    result = session.get(f"{args.url}/v1/result/{task_id}", timeout=args.request_timeout)
    result.raise_for_status()
    return result.json(), time.monotonic() - started


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", default="http://127.0.0.1:5001")
    parser.add_argument("--api-key", default="")
    parser.add_argument("--container")
    parser.add_argument("--out", type=Path)
    parser.add_argument("--ocr-preset", default="rapidocr")
    parser.add_argument("--ocr-lang", action="append", default=None)
    parser.add_argument("--table-mode", default="accurate")
    parser.add_argument("--no-ocr", action="store_true")
    parser.add_argument("--request-timeout", type=float, default=60)
    parser.add_argument("files", nargs="+", type=Path)
    args = parser.parse_args()
    args.ocr_lang = args.ocr_lang or ["en"]

    session = requests.Session()
    session.trust_env = False
    if args.api_key:
        session.headers["X-Api-Key"] = args.api_key

    memory = peak_memory_file(args.container) if args.container else None
    if args.out:
        args.out.mkdir(parents=True, exist_ok=True)

    rows = []
    for path in args.files:
        reset_peak(memory)
        body, wall = convert(session, args, path)
        peak = read_peak(memory)
        document = (body.get("document") or {}).get("json_content") or {}
        pages = len(document.get("pages") or {}) or 1
        row = {
            "file": path.name,
            "status": body.get("status"),
            "pages": pages,
            "wall_s": round(wall, 2),
            "processing_s": round(body.get("processing_time") or 0, 2),
            "s_per_page": round(wall / pages, 2),
            "peak_mib": round(peak / 2**20) if peak is not None else "",
            "errors": len(body.get("errors") or []),
        }
        rows.append(row)
        print(json.dumps(row), flush=True)
        if args.out:
            (args.out / f"{path.name}.json").write_text(json.dumps(document, indent=1, sort_keys=True))

    if args.out:
        with (args.out / "summary.csv").open("w", newline="") as handle:
            writer = csv.DictWriter(handle, fieldnames=list(rows[0].keys()))
            writer.writeheader()
            writer.writerows(rows)


if __name__ == "__main__":
    sys.exit(main())
