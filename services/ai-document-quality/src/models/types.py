#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
from dataclasses import dataclass
from typing import TypedDict


class OCRQualityMetrics(TypedDict):
    word_count: int
    char_count: int
    special_char_ratio: float
    word_to_char_ratio: float
    transporation_doc: bool
    transportation_fields: dict[str, bool]


class QualityPrediction(TypedDict):
    quality_score: float
    confidence: float
    category: str
    sharpness: float
    text_density: float
    ocr_quality: int
    ocr_detail: OCRQualityMetrics
    metrics_consistency: float


@dataclass
class QualityThreshold:
    min_score: float
    min_sharpness: float
    min_density: float
    min_words: int
    category: str
