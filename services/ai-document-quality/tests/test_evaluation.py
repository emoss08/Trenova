"""
Unit tests for evaluation metrics and explainability.
"""

import pytest
import torch
import numpy as np
from sklearn.metrics import accuracy_score, f1_score


@pytest.mark.evaluation
class TestMetricsCalculation:
    """Tests for metric calculation functions."""

    def test_mae_calculation(self):
        """Test Mean Absolute Error calculation."""
        y_true = np.array([1.0, 2.0, 3.0, 4.0, 5.0])
        y_pred = np.array([1.1, 2.1, 2.9, 4.2, 4.8])

        mae = np.mean(np.abs(y_true - y_pred))

        assert isinstance(mae, (float, np.floating))
        assert mae >= 0
        assert mae < 1.0  # Should be small for close predictions

    def test_rmse_calculation(self):
        """Test Root Mean Squared Error calculation."""
        y_true = np.array([1.0, 2.0, 3.0, 4.0, 5.0])
        y_pred = np.array([1.1, 2.1, 2.9, 4.2, 4.8])

        rmse = np.sqrt(np.mean((y_true - y_pred) ** 2))

        assert isinstance(rmse, (float, np.floating))
        assert rmse >= 0
        assert rmse < 1.0  # Should be small for close predictions

    def test_r2_score_calculation(self):
        """Test R² Score calculation."""
        y_true = np.array([1.0, 2.0, 3.0, 4.0, 5.0])
        y_pred = np.array([1.1, 2.1, 2.9, 4.2, 4.8])

        ss_res = np.sum((y_true - y_pred) ** 2)
        ss_tot = np.sum((y_true - np.mean(y_true)) ** 2)
        r2 = 1 - (ss_res / ss_tot)

        assert isinstance(r2, (float, np.floating))
        assert r2 <= 1.0
        assert r2 > 0.8  # Should be high for good predictions

    def test_accuracy_calculation(self):
        """Test classification accuracy calculation."""
        y_true = np.array([0, 1, 2, 3, 4])
        y_pred = np.array([0, 1, 2, 3, 4])

        accuracy = accuracy_score(y_true, y_pred)

        assert accuracy == 1.0

    def test_accuracy_with_errors(self):
        """Test accuracy with prediction errors."""
        y_true = np.array([0, 1, 2, 3, 4])
        y_pred = np.array([0, 1, 1, 3, 4])  # One error

        accuracy = accuracy_score(y_true, y_pred)

        assert accuracy == 0.8

    def test_f1_score_calculation(self):
        """Test F1 score calculation."""
        y_true = np.array([0, 1, 0, 1, 0, 1])
        y_pred = np.array([0, 1, 0, 1, 0, 1])

        f1 = f1_score(y_true, y_pred, average='binary')

        assert f1 == 1.0

    def test_f1_score_multiclass(self):
        """Test F1 score for multiclass."""
        y_true = np.array([0, 1, 2, 0, 1, 2])
        y_pred = np.array([0, 1, 2, 0, 1, 2])

        f1_macro = f1_score(y_true, y_pred, average='macro')
        f1_micro = f1_score(y_true, y_pred, average='micro')

        assert f1_macro == 1.0
        assert f1_micro == 1.0


@pytest.mark.evaluation
class TestCalibrationMetrics:
    """Tests for calibration metrics."""

    def test_ece_calculation(self):
        """Test Expected Calibration Error calculation."""
        # Perfect calibration
        confidences = np.array([0.9, 0.8, 0.7, 0.6, 0.5])
        predictions = np.array([1, 1, 1, 0, 0])
        targets = np.array([1, 1, 1, 0, 0])

        # Simple ECE calculation
        n_bins = 10
        bins = np.linspace(0, 1, n_bins + 1)
        bin_indices = np.digitize(confidences, bins) - 1

        ece = 0.0
        for i in range(n_bins):
            mask = bin_indices == i
            if mask.sum() > 0:
                bin_acc = (predictions[mask] == targets[mask]).mean()
                bin_conf = confidences[mask].mean()
                bin_weight = mask.sum() / len(confidences)
                ece += bin_weight * abs(bin_acc - bin_conf)

        assert isinstance(ece, (float, np.floating))
        assert 0 <= ece <= 1

    def test_mce_calculation(self):
        """Test Maximum Calibration Error calculation."""
        confidences = np.array([0.9, 0.8, 0.7, 0.6, 0.5])
        predictions = np.array([1, 1, 1, 0, 0])
        targets = np.array([1, 1, 1, 0, 0])

        # Simple MCE calculation
        n_bins = 10
        bins = np.linspace(0, 1, n_bins + 1)
        bin_indices = np.digitize(confidences, bins) - 1

        mce = 0.0
        for i in range(n_bins):
            mask = bin_indices == i
            if mask.sum() > 0:
                bin_acc = (predictions[mask] == targets[mask]).mean()
                bin_conf = confidences[mask].mean()
                mce = max(mce, abs(bin_acc - bin_conf))

        assert isinstance(mce, (float, np.floating))
        assert 0 <= mce <= 1

    def test_brier_score_calculation(self):
        """Test Brier score calculation."""
        probs = np.array([0.9, 0.8, 0.2, 0.1])
        targets = np.array([1, 1, 0, 0])

        brier = np.mean((probs - targets) ** 2)

        assert isinstance(brier, (float, np.floating))
        assert 0 <= brier <= 1
        assert brier < 0.2  # Should be low for good predictions


@pytest.mark.evaluation
class TestConfusionMatrix:
    """Tests for confusion matrix calculations."""

    def test_confusion_matrix_binary(self):
        """Test binary confusion matrix."""
        y_true = np.array([0, 1, 0, 1, 0, 1])
        y_pred = np.array([0, 1, 0, 1, 0, 0])

        from sklearn.metrics import confusion_matrix
        cm = confusion_matrix(y_true, y_pred)

        assert cm.shape == (2, 2)
        assert cm.sum() == len(y_true)

    def test_confusion_matrix_multiclass(self):
        """Test multiclass confusion matrix."""
        y_true = np.array([0, 1, 2, 3, 4, 0, 1, 2])
        y_pred = np.array([0, 1, 2, 3, 4, 0, 1, 1])

        from sklearn.metrics import confusion_matrix
        cm = confusion_matrix(y_true, y_pred, labels=[0, 1, 2, 3, 4])

        assert cm.shape == (5, 5)
        assert cm.sum() == len(y_true)

    def test_confusion_matrix_diagonal(self):
        """Test perfect predictions confusion matrix."""
        y_true = np.array([0, 1, 2, 3, 4])
        y_pred = np.array([0, 1, 2, 3, 4])

        from sklearn.metrics import confusion_matrix
        cm = confusion_matrix(y_true, y_pred)

        # All predictions on diagonal
        assert np.all(np.diag(cm) > 0)
        assert cm.sum() - np.trace(cm) == 0  # No off-diagonal elements


@pytest.mark.evaluation
class TestROCandPRCurves:
    """Tests for ROC and PR curve calculations."""

    def test_roc_auc_binary(self):
        """Test ROC AUC for binary classification."""
        y_true = np.array([0, 0, 1, 1])
        y_scores = np.array([0.1, 0.4, 0.35, 0.8])

        from sklearn.metrics import roc_auc_score
        auc = roc_auc_score(y_true, y_scores)

        assert 0 <= auc <= 1
        assert auc > 0.5  # Should be better than random

    def test_roc_curve_calculation(self):
        """Test ROC curve calculation."""
        y_true = np.array([0, 0, 1, 1, 1])
        y_scores = np.array([0.1, 0.3, 0.4, 0.7, 0.9])

        from sklearn.metrics import roc_curve
        fpr, tpr, thresholds = roc_curve(y_true, y_scores)

        assert len(fpr) == len(tpr) == len(thresholds)
        assert fpr[0] == 0.0
        assert tpr[-1] == 1.0

    def test_pr_auc_calculation(self):
        """Test Precision-Recall AUC calculation."""
        y_true = np.array([0, 0, 1, 1, 1])
        y_scores = np.array([0.1, 0.3, 0.4, 0.7, 0.9])

        from sklearn.metrics import average_precision_score
        pr_auc = average_precision_score(y_true, y_scores)

        assert 0 <= pr_auc <= 1

    def test_precision_recall_curve(self):
        """Test precision-recall curve calculation."""
        y_true = np.array([0, 0, 1, 1, 1])
        y_scores = np.array([0.1, 0.3, 0.4, 0.7, 0.9])

        from sklearn.metrics import precision_recall_curve
        precision, recall, thresholds = precision_recall_curve(y_true, y_scores)

        assert len(precision) == len(recall)
        assert len(thresholds) == len(precision) - 1


@pytest.mark.evaluation
class TestIssueDetectionMetrics:
    """Tests for issue detection metrics."""

    def test_multilabel_accuracy(self):
        """Test multilabel accuracy calculation."""
        y_true = np.array([[1, 0, 1], [0, 1, 1], [1, 1, 0]])
        y_pred = np.array([[1, 0, 1], [0, 1, 1], [1, 0, 0]])

        # Exact match accuracy
        exact_match = (y_true == y_pred).all(axis=1).mean()

        assert 0 <= exact_match <= 1

    def test_multilabel_f1_micro(self):
        """Test multilabel F1 score (micro)."""
        y_true = np.array([[1, 0, 1], [0, 1, 1], [1, 1, 0]])
        y_pred = np.array([[1, 0, 1], [0, 1, 1], [1, 0, 0]])

        from sklearn.metrics import f1_score
        f1_micro = f1_score(y_true, y_pred, average='micro')

        assert 0 <= f1_micro <= 1

    def test_multilabel_f1_macro(self):
        """Test multilabel F1 score (macro)."""
        y_true = np.array([[1, 0, 1], [0, 1, 1], [1, 1, 0]])
        y_pred = np.array([[1, 0, 1], [0, 1, 1], [1, 0, 0]])

        from sklearn.metrics import f1_score
        f1_macro = f1_score(y_true, y_pred, average='macro')

        assert 0 <= f1_macro <= 1

    def test_per_issue_f1_scores(self):
        """Test per-issue F1 scores."""
        y_true = np.array([[1, 0, 1], [0, 1, 1], [1, 1, 0]])
        y_pred = np.array([[1, 0, 1], [0, 1, 1], [1, 0, 0]])

        from sklearn.metrics import f1_score
        f1_per_class = f1_score(y_true, y_pred, average=None)

        assert len(f1_per_class) == 3
        for score in f1_per_class:
            assert 0 <= score <= 1

    def test_hamming_loss(self):
        """Test Hamming loss for multilabel."""
        y_true = np.array([[1, 0, 1], [0, 1, 1], [1, 1, 0]])
        y_pred = np.array([[1, 0, 1], [0, 1, 1], [1, 0, 0]])

        from sklearn.metrics import hamming_loss
        loss = hamming_loss(y_true, y_pred)

        assert 0 <= loss <= 1


@pytest.mark.evaluation
class TestBalancedAccuracy:
    """Tests for balanced accuracy metrics."""

    def test_balanced_accuracy_perfect(self):
        """Test balanced accuracy with perfect predictions."""
        y_true = np.array([0, 0, 1, 1, 2, 2])
        y_pred = np.array([0, 0, 1, 1, 2, 2])

        from sklearn.metrics import balanced_accuracy_score
        bal_acc = balanced_accuracy_score(y_true, y_pred)

        assert bal_acc == 1.0

    def test_balanced_accuracy_imbalanced(self):
        """Test balanced accuracy with imbalanced classes."""
        # Imbalanced dataset
        y_true = np.array([0, 0, 0, 0, 0, 1, 1, 2])
        y_pred = np.array([0, 0, 0, 0, 0, 1, 0, 2])

        from sklearn.metrics import balanced_accuracy_score
        bal_acc = balanced_accuracy_score(y_true, y_pred)

        assert 0 <= bal_acc <= 1

    def test_balanced_accuracy_vs_regular(self):
        """Test that balanced accuracy differs from regular accuracy on imbalanced data."""
        # Heavily imbalanced
        y_true = np.array([0] * 90 + [1] * 10)
        y_pred = np.array([0] * 100)  # Predict all as majority class

        from sklearn.metrics import accuracy_score, balanced_accuracy_score
        regular_acc = accuracy_score(y_true, y_pred)
        balanced_acc = balanced_accuracy_score(y_true, y_pred)

        # Regular accuracy should be high (0.9)
        # Balanced accuracy should be low (0.5)
        assert regular_acc > balanced_acc


@pytest.mark.evaluation
class TestMetricsEdgeCases:
    """Tests for edge cases in metrics calculation."""

    def test_all_same_predictions(self):
        """Test metrics when all predictions are the same."""
        y_true = np.array([0, 1, 2, 3, 4])
        y_pred = np.array([2, 2, 2, 2, 2])

        from sklearn.metrics import accuracy_score
        accuracy = accuracy_score(y_true, y_pred)

        assert 0 <= accuracy <= 1

    def test_all_correct_predictions(self):
        """Test metrics with all correct predictions."""
        y_true = np.array([0, 1, 2, 3, 4])
        y_pred = np.array([0, 1, 2, 3, 4])

        from sklearn.metrics import accuracy_score, f1_score
        accuracy = accuracy_score(y_true, y_pred)
        f1 = f1_score(y_true, y_pred, average='macro')

        assert accuracy == 1.0
        assert f1 == 1.0

    def test_all_wrong_predictions(self):
        """Test metrics with all wrong predictions."""
        y_true = np.array([0, 1, 2, 3, 4])
        y_pred = np.array([4, 3, 1, 0, 2])

        from sklearn.metrics import accuracy_score
        accuracy = accuracy_score(y_true, y_pred)

        assert accuracy == 0.0

    def test_empty_arrays(self):
        """Test that metrics handle empty arrays appropriately."""
        y_true = np.array([])
        y_pred = np.array([])

        # Should either raise error or return NaN
        try:
            from sklearn.metrics import accuracy_score
            result = accuracy_score(y_true, y_pred)
            assert np.isnan(result) or result == 0
        except ValueError:
            pass  # Expected behavior


@pytest.mark.evaluation
class TestMetricsIntegration:
    """Integration tests for metrics calculation."""

    def test_comprehensive_metrics_dict(self, sample_metrics):
        """Test comprehensive metrics dictionary structure."""
        assert "regression" in sample_metrics
        assert "classification" in sample_metrics
        assert "calibration" in sample_metrics
        assert "issue_detection" in sample_metrics

    def test_regression_metrics_complete(self, sample_metrics):
        """Test regression metrics are complete."""
        reg_metrics = sample_metrics["regression"]

        assert "mae" in reg_metrics
        assert "rmse" in reg_metrics
        assert "r2" in reg_metrics
        assert "mape" in reg_metrics

        # Check ranges
        assert reg_metrics["mae"] >= 0
        assert reg_metrics["rmse"] >= 0
        assert reg_metrics["r2"] <= 1.0

    def test_classification_metrics_complete(self, sample_metrics):
        """Test classification metrics are complete."""
        cls_metrics = sample_metrics["classification"]

        assert "accuracy" in cls_metrics
        assert "balanced_accuracy" in cls_metrics
        assert "f1_macro" in cls_metrics

        # Check ranges
        assert 0 <= cls_metrics["accuracy"] <= 1
        assert 0 <= cls_metrics["balanced_accuracy"] <= 1
        assert 0 <= cls_metrics["f1_macro"] <= 1

    def test_calibration_metrics_complete(self, sample_metrics):
        """Test calibration metrics are complete."""
        cal_metrics = sample_metrics["calibration"]

        assert "ece" in cal_metrics
        assert "mce" in cal_metrics
        assert "brier_score" in cal_metrics

        # Check ranges
        assert 0 <= cal_metrics["ece"] <= 1
        assert 0 <= cal_metrics["mce"] <= 1
        assert 0 <= cal_metrics["brier_score"] <= 1

    def test_issue_detection_metrics_complete(self, sample_metrics):
        """Test issue detection metrics are complete."""
        issue_metrics = sample_metrics["issue_detection"]

        assert "f1_micro" in issue_metrics
        assert "f1_macro" in issue_metrics

        # Check ranges
        assert 0 <= issue_metrics["f1_micro"] <= 1
        assert 0 <= issue_metrics["f1_macro"] <= 1

    def test_metrics_consistency(self):
        """Test that related metrics are consistent."""
        # For perfect predictions
        y_true = np.array([0, 1, 2, 3, 4])
        y_pred = np.array([0, 1, 2, 3, 4])

        from sklearn.metrics import accuracy_score, balanced_accuracy_score
        acc = accuracy_score(y_true, y_pred)
        bal_acc = balanced_accuracy_score(y_true, y_pred)

        # For perfect predictions, both should be 1.0
        assert acc == bal_acc == 1.0

    def test_metrics_reasonable_values(self, sample_metrics):
        """Test that all metrics have reasonable values."""
        def check_metric_range(metrics_dict):
            for key, value in metrics_dict.items():
                if isinstance(value, dict):
                    check_metric_range(value)
                elif isinstance(value, (int, float)):
                    # Most metrics should be between 0 and 1, or reasonable values
                    assert not np.isnan(value), f"Metric {key} is NaN"
                    assert not np.isinf(value), f"Metric {key} is infinite"

        check_metric_range(sample_metrics)
