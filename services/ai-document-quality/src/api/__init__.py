# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md

"""API module for document quality assessment service"""

from .inference import DocumentQualityPredictor
from .models import (
    DocumentAnalysisRequest,
    DocumentAnalysisResponse,
    HealthResponse,
    IssueDetection,
    QualityAssessment,
)

__all__ = [
    "DocumentQualityPredictor",
    "DocumentAnalysisRequest",
    "DocumentAnalysisResponse",
    "HealthResponse",
    "QualityAssessment",
    "IssueDetection",
]
