from __future__ import annotations

import hashlib
import json
from pathlib import Path

import pytest

from trenova_finetune import dataset

FIXTURES = Path(__file__).resolve().parent / "fixtures"
SCHEMA = json.loads((FIXTURES / "extraction-schema.json").read_text())
FIELD_KEYS = SCHEMA["properties"]["fields"]["items"]["properties"]["key"]["enum"]

PAGE = (
    "Rate Confirmation Load # OUG-8393964\n"
    "Shipper: Juniper Manufacturing, 8862 Poplar Rd, Hartwell, CA 85055\n"
    "Pickup 03/14/2026 08:00-14:00\n"
    "Consignee: Granite Brands, Crestview, CO 52999\n"
    "Total: $2,563.12 USD"
)
PROMPT = [
    {"role": "system", "content": "Extract the fields."},
    {"role": "user", "content": f"## Document Pages\n<untrusted_data>\n{PAGE}\n</untrusted_data>"},
]


def target_snapshot() -> dict:
    return {
        "fields": {
            "referenceNumber": "OUG-8393964",
            "rate": "2563.12",
            "shipper": "Juniper Manufacturing",
            "consignee": "Granite Brands",
            "pickupWindow": "2026-03-14",
            "weight": "42000",
        },
        "stops": [
            {
                "role": "pickup",
                "sequence": 0,
                "name": "Juniper Manufacturing",
                "addressLine1": "8862 Poplar Rd",
                "city": "Hartwell",
                "state": "CA",
                "postalCode": "85055",
                "date": "2026-03-14",
                "appointmentRequired": False,
            },
            {
                "role": "delivery",
                "sequence": 0,
                "name": "Granite Brands",
                "city": "Crestview",
                "state": "CO",
                "postalCode": "52999",
                "date": "2026-03-16",
                "appointmentRequired": True,
            },
        ],
    }


def prediction_snapshot() -> dict:
    snapshot = target_snapshot()
    snapshot["fields"]["shipper"] = "Harbor Supply"
    snapshot["fields"]["loadNumber"] = "OUG-8393964"
    snapshot["stops"][0]["timeWindow"] = "08:00-14:00"
    return snapshot


def example(identifier: str, split: str, *, corrected: bool = True) -> dict:
    return {
        "id": identifier,
        "split": split,
        "documentKind": "RateConfirmation",
        "prompt": PROMPT,
        "visiblePages": [{"number": 1, "text": PAGE}],
        "target": target_snapshot(),
        "prediction": prediction_snapshot(),
        "outcomes": {
            "referenceNumber": "Correct",
            "shipper": "Corrected" if corrected else "Correct",
        },
    }


def _jsonl(records: list[dict]) -> bytes:
    return b"".join(json.dumps(record).encode() + b"\n" for record in records)


def write_dataset(
    root: Path,
    *,
    train: int = 3,
    validation: int = 2,
    corrected: bool = True,
    schema: dict | None = None,
    field_keys: list[str] | None = None,
) -> Path:
    root.mkdir(parents=True, exist_ok=True)
    contents = {
        dataset.TRAIN_FILE: _jsonl(
            [example(f"t{i}", "train", corrected=corrected) for i in range(train)]
        ),
        dataset.VALIDATION_FILE: _jsonl(
            [example(f"v{i}", "validation") for i in range(validation)]
        ),
        dataset.EVALUATION_FILE: _jsonl(
            [
                {
                    "id": f"v{i}",
                    "prompt": PROMPT,
                    "expected": target_snapshot(),
                    "baseline": prediction_snapshot(),
                }
                for i in range(validation)
            ]
        ),
        dataset.SCHEMA_FILE: json.dumps(schema or SCHEMA, indent=2).encode() + b"\n",
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
        "pageLimit": 2500,
        "fieldKeys": field_keys or FIELD_KEYS,
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
