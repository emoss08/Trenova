#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
"""Model components for document quality assessment"""

from .analyzer import DocumentAnalysisResult, DocumentAnalyzer, DocumentIssue
from .inference import (
    calculate_sharpness,
    calculate_text_density,
    categorize_quality,
    detect_transportation_fields,
    load_model,
    ocr_text_quality,
    predict_quality,
)
from .model import (
    DocumentQualityDataset,
    DocumentQualityModel,
    FocalLoss,
    ModelConfig,
    MultiTaskLoss,
    create_model,
    evaluate_model,
    train_model,
)
from .types import OCRQualityMetrics, QualityPrediction, QualityThreshold

__all__ = [
    # Model components
    "DocumentQualityModel",
    "ModelConfig",
    "create_model",
    "train_model",
    "evaluate_model",
    "MultiTaskLoss",
    "FocalLoss",
    "DocumentQualityDataset",
    # Inference
    "predict_quality",
    "load_model",
    "categorize_quality",
    "calculate_sharpness",
    "calculate_text_density",
    "detect_transportation_fields",
    "ocr_text_quality",
    # Analysis
    "DocumentAnalyzer",
    "DocumentAnalysisResult",
    "DocumentIssue",
    # Types
    "OCRQualityMetrics",
    "QualityPrediction",
    "QualityThreshold",
]
