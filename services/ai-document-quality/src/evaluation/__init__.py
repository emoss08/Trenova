# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md

"""Evaluation module for document quality assessment model"""

from .metrics import (
    calculate_calibration_metrics,
    calculate_classification_metrics,
    calculate_regression_metrics,
)
from .visualize import (
    create_evaluation_report,
    plot_calibration_curve,
    plot_confusion_matrix,
    plot_issue_analysis,
    plot_prediction_distribution,
    plot_roc_curves,
)

__all__ = [
    "calculate_calibration_metrics",
    "calculate_classification_metrics",
    "calculate_regression_metrics",
    "plot_calibration_curve",
    "plot_confusion_matrix",
    "plot_roc_curves",
    "plot_issue_analysis",
    "plot_prediction_distribution",
    "create_evaluation_report",
]
