from __future__ import annotations

import hashlib
import json
from pathlib import Path

import pytest

from trenova_finetune import dataset

REPLY = json.dumps({"documentKind": "RateConfirmation", "fields": [], "stops": []})
PROMPT = [
    {"role": "system", "content": "Extract the fields."},
    {"role": "user", "content": "## Document Pages\n<untrusted_data>\nLoad 1\n</untrusted_data>"},
]


def _jsonl(records: list[dict]) -> bytes:
    return b"".join(json.dumps(record).encode() + b"\n" for record in records)


def write_dataset(root: Path, *, train: int = 3, validation: int = 2, pairs: int = 2) -> Path:
    root.mkdir(parents=True, exist_ok=True)
    completion = [{"role": "assistant", "content": REPLY}]
    contents = {
        dataset.SFT_TRAIN_FILE: _jsonl(
            [{"id": f"t{i}", "prompt": PROMPT, "completion": completion} for i in range(train)]
        ),
        dataset.SFT_VALIDATION_FILE: _jsonl(
            [{"id": f"v{i}", "prompt": PROMPT, "completion": completion} for i in range(validation)]
        ),
        dataset.PREFERENCE_FILE: _jsonl(
            [
                {"id": f"t{i}", "prompt": PROMPT, "chosen": completion, "rejected": completion}
                for i in range(pairs)
            ]
        ),
        dataset.EVALUATION_FILE: _jsonl(
            [
                {"id": f"v{i}", "prompt": PROMPT, "expected": {"fields": {}}, "baseline": None}
                for i in range(validation)
            ]
        ),
        dataset.SCHEMA_FILE: json.dumps({"type": "object"}).encode() + b"\n",
    }
    files = []
    for name, content in contents.items():
        (root / name).write_bytes(content)
        files.append(
            {
                "name": name,
                "records": content.count(b"\n") if name.endswith(".jsonl") else 0,
                "bytes": len(content),
                "sha256": hashlib.sha256(content).hexdigest(),
            }
        )
    manifest = {
        "format": dataset.DATASET_FORMAT,
        "exportId": "aitx_1",
        "exportManifestSha256": "e" * 64,
        "exampleFormat": "trenova.extraction-training/v1",
        "task": "ShipmentDraftExtraction",
        "structuredOutputMode": "JSONSchema",
        "schemaName": "rate_confirmation_extract",
        "promptSha256": "p" * 64,
        "temperature": 0.1,
        "topP": 0.95,
        "keepUnverified": True,
        "counts": {"examples": train + validation, "train": train, "validation": validation},
        "files": files,
        "renderedAt": 1790000000,
    }
    (root / dataset.MANIFEST_FILE).write_text(json.dumps(manifest, indent=2) + "\n")
    return root


@pytest.fixture
def dataset_dir(tmp_path: Path) -> Path:
    return write_dataset(tmp_path / "dataset")


CONFIG_YAML = """
base_model: Qwen/Qwen2.5-7B-Instruct
quantization: none
sft:
  max_length: 4096
dpo:
  min_pairs: 1
predict:
  max_model_len: 8192
"""


@pytest.fixture
def config_path(tmp_path: Path) -> Path:
    path = tmp_path / "config.yaml"
    path.write_text(CONFIG_YAML)
    return path
