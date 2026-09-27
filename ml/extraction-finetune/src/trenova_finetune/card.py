"""The model card: `trenova-model.json`, written beside every model this pipeline produces.

It records what the model was trained from and how Trenova's provider must call it, and
marks the directory as one `serve` and `quantize` will accept.
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from .files import write_json

CARD_FORMAT = "trenova.extraction-model/v1"
CARD_FILE = "trenova-model.json"


class ModelCardError(ValueError):
    """The directory holds no model card, or one this pipeline did not write."""


def read_card(model_dir: Path) -> dict[str, Any]:
    """Return the model card, refusing a directory this pipeline did not produce."""
    path = model_dir / CARD_FILE
    try:
        card = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as err:
        raise ModelCardError(
            f"{model_dir} is not a model this pipeline produced (no {CARD_FILE})"
        ) from err
    except (OSError, json.JSONDecodeError) as err:
        raise ModelCardError(f"cannot read {path}: {err}") from err
    if not isinstance(card, dict) or card.get("format") != CARD_FORMAT:
        raise ModelCardError(f"{path} is not a {CARD_FORMAT} model card")
    return card


def save_card(model_dir: Path, card: dict[str, Any]) -> None:
    write_json(model_dir / CARD_FILE, card)
