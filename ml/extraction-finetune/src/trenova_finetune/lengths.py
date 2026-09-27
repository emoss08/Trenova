"""Keep examples whose whole conversation fits the context used for training.

A target that is cut off teaches the model to stop mid-JSON, so an example that
does not fit is dropped rather than truncated, and too many drops stop the run.
"""

from __future__ import annotations

from collections.abc import Callable, Sequence
from dataclasses import dataclass
from typing import TypeVar

Record = TypeVar("Record")


class LengthBudgetError(ValueError):
    """Too many examples do not fit the configured context length."""


@dataclass(frozen=True)
class LengthPartition:
    kept: list
    dropped: int
    longest: int

    @property
    def dropped_fraction(self) -> float:
        total = len(self.kept) + self.dropped
        return self.dropped / total if total else 0.0


def partition_by_length(
    records: Sequence[Record],
    count_tokens: Callable[[Record], int],
    max_length: int,
) -> LengthPartition:
    kept: list = []
    dropped = 0
    longest = 0
    for record in records:
        length = count_tokens(record)
        longest = max(longest, length)
        if length <= max_length:
            kept.append(record)
        else:
            dropped += 1
    return LengthPartition(kept=kept, dropped=dropped, longest=longest)


def enforce_budget(partition: LengthPartition, max_fraction: float, label: str) -> None:
    if not partition.kept:
        raise LengthBudgetError(f"no {label} example fits the configured max_length")
    if partition.dropped_fraction > max_fraction:
        raise LengthBudgetError(
            f"{partition.dropped} {label} example(s) ({partition.dropped_fraction:.1%}) are longer "
            f"than max_length (the longest is {partition.longest} tokens); raise max_length or "
            f"max_dropped_fraction"
        )
