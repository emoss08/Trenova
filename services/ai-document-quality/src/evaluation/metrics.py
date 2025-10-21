import logging
from typing import Dict, List, Optional, Any

import numpy as np
import torch
import torch.nn as nn
from sklearn.metrics import (
    accuracy_score,
    auc,
    balanced_accuracy_score,
    classification_report,
    confusion_matrix,
    f1_score,
    mean_absolute_error,
    mean_squared_error,
    precision_recall_curve,
    precision_recall_fscore_support,
    r2_score,
    roc_auc_score,
    roc_curve,
)
from torch.utils.data import DataLoader

logger = logging.getLogger(__name__)


def calculate_regression_metrics(
    predictions: np.ndarray, targets: np.ndarray
) -> Dict[str, float]:
    mae = mean_absolute_error(targets, predictions)
    mse = mean_squared_error(targets, predictions)
    rmse = np.sqrt(mse)
    r2 = r2_score(targets, predictions)

    residuals = predictions - targets
    mean_residual = np.mean(residuals)
    std_residual = np.std(residuals)

    tolerance_5pct = np.mean(np.abs(residuals) < 0.05)
    tolerance_10pct = np.mean(np.abs(residuals) < 0.10)
    tolerance_20pct = np.mean(np.abs(residuals) < 0.20)

    mape = np.mean(np.abs((targets - predictions) / (targets + 1e-8))) * 100

    return {
        "mae": float(mae),
        "mse": float(mse),
        "rmse": float(rmse),
        "r2": float(r2),
        "mean_residual": float(mean_residual),
        "std_residual": float(std_residual),
        "within_5pct": float(tolerance_5pct),
        "within_10pct": float(tolerance_10pct),
        "within_20pct": float(tolerance_20pct),
        "mape": float(mape),
        "pred_mean": float(np.mean(predictions)),
        "pred_std": float(np.std(predictions)),
        "pred_min": float(np.min(predictions)),
        "pred_max": float(np.max(predictions)),
    }


def calculate_classification_metrics(
    predictions: np.ndarray,
    targets: np.ndarray,
    class_names: Optional[List[str]] = None,
) -> Dict[str, Any]:
    if class_names is None:
        class_names = ["High", "Good", "Moderate", "Poor", "Very Poor"]

    accuracy = accuracy_score(targets, predictions)
    balanced_acc = balanced_accuracy_score(targets, predictions)

    precision, recall, f1, support = precision_recall_fscore_support(
        targets, predictions, average=None, zero_division=0
    )

    weighted_precision = np.average(precision, weights=support)
    weighted_recall = np.average(recall, weights=support)
    weighted_f1 = f1_score(targets, predictions, average="weighted")

    macro_precision = np.mean(precision)
    macro_recall = np.mean(recall)
    macro_f1 = f1_score(targets, predictions, average="macro")

    cm = confusion_matrix(targets, predictions)

    per_class_metrics = {}
    for i, name in enumerate(class_names[: len(precision)]):
        per_class_metrics[name] = {
            "precision": float(precision[i]),
            "recall": float(recall[i]),
            "f1": float(f1[i]),
            "support": int(support[i]),
        }

    report = classification_report(
        targets,
        predictions,
        target_names=class_names[: len(precision)],
        zero_division=0,
    )

    return {
        "accuracy": float(accuracy),
        "balanced_accuracy": float(balanced_acc),
        "weighted_precision": float(weighted_precision),
        "weighted_recall": float(weighted_recall),
        "weighted_f1": float(weighted_f1),
        "macro_precision": float(macro_precision),
        "macro_recall": float(macro_recall),
        "macro_f1": float(macro_f1),
        "per_class": per_class_metrics,
        "confusion_matrix": cm.tolist(),
        "classification_report": report,
    }


def calculate_calibration_metrics(
    predictions: np.ndarray, targets: np.ndarray, n_bins: int = 10
) -> Dict[str, Any]:
    predictions = np.clip(predictions, 0, 1)
    bin_boundaries = np.linspace(0, 1, n_bins + 1)
    bin_lowers = bin_boundaries[:-1]
    bin_uppers = bin_boundaries[1:]
    ece = 0.0
    mce = 0.0
    bin_accs = []
    bin_confs = []
    bin_counts = []

    for bin_lower, bin_upper in zip(bin_lowers, bin_uppers):
        in_bin = (predictions > bin_lower) & (predictions <= bin_upper)
        prop_in_bin = np.mean(in_bin)

        if prop_in_bin > 0:
            accuracy_in_bin = np.mean(targets[in_bin])
            avg_confidence_in_bin = np.mean(predictions[in_bin])
            count_in_bin = np.sum(in_bin)

            ece += np.abs(avg_confidence_in_bin - accuracy_in_bin) * prop_in_bin

            mce = max(mce, np.abs(avg_confidence_in_bin - accuracy_in_bin))

            bin_accs.append(float(accuracy_in_bin))
            bin_confs.append(float(avg_confidence_in_bin))
            bin_counts.append(int(count_in_bin))
        else:
            bin_accs.append(0.0)
            bin_confs.append(0.0)
            bin_counts.append(0)

    return {
        "ece": float(ece),  # Expected Calibration Error
        "mce": float(mce),  # Maximum Calibration Error
        "bin_accuracies": bin_accs,
        "bin_confidences": bin_confs,
        "bin_counts": bin_counts,
        "n_bins": n_bins,
    }


def calculate_binary_classification_metrics(
    predictions: np.ndarray,
    targets: np.ndarray,
    threshold: float = 0.5,
    pos_label: int = 1,
) -> Dict[str, float]:
    predictions = np.clip(predictions, 0, 1)
    binary_preds = (predictions >= threshold).astype(int)

    accuracy = accuracy_score(targets, binary_preds)
    precision, recall, f1, _ = precision_recall_fscore_support(
        targets, binary_preds, average="binary", pos_label=pos_label, zero_division=0
    )

    try:
        roc_auc = roc_auc_score(targets, predictions)
    except Exception:
        roc_auc = 0.0

    try:
        precision_curve, recall_curve, _ = precision_recall_curve(targets, predictions)
        pr_auc = auc(recall_curve, precision_curve)
    except Exception:
        pr_auc = 0.0

    cm = confusion_matrix(targets, binary_preds)
    tn = cm[0, 0] if cm.shape[0] > 0 else 0
    fp = cm[0, 1] if cm.shape[0] > 0 and cm.shape[1] > 1 else 0
    fn = cm[1, 0] if cm.shape[0] > 1 else 0
    tp = cm[1, 1] if cm.shape[0] > 1 and cm.shape[1] > 1 else 0

    specificity = tn / (tn + fp) if (tn + fp) > 0 else 0
    npv = tn / (tn + fn) if (tn + fn) > 0 else 0
    fpr = fp / (fp + tn) if (fp + tn) > 0 else 0
    fnr = fn / (fn + tp) if (fn + tp) > 0 else 0

    return {
        "threshold": float(threshold),
        "accuracy": float(accuracy),
        "precision": float(precision),
        "recall": float(recall),
        "f1": float(f1),
        "specificity": float(specificity),
        "npv": float(npv),
        "fpr": float(fpr),
        "fnr": float(fnr),
        "roc_auc": float(roc_auc),
        "pr_auc": float(pr_auc),
        "true_positives": int(tp),
        "true_negatives": int(tn),
        "false_positives": int(fp),
        "false_negatives": int(fn),
    }


def calculate_issue_detection_metrics(
    predictions: np.ndarray,
    targets: np.ndarray,
    issue_names: Optional[List[str]] = None,
    threshold: float = 0.5,
) -> Dict[str, Any]:
    predictions = np.clip(predictions, 0, 1)
    if issue_names is None:
        issue_names = [
            "blur",
            "noise",
            "lighting",
            "shadow",
            "physical_damage",
            "skew",
            "partial",
            "glare",
            "compression",
            "overall_poor",
        ]

    binary_preds = (predictions >= threshold).astype(int)

    per_issue_metrics = {}
    for i, issue_name in enumerate(issue_names[: predictions.shape[1]]):
        issue_preds = binary_preds[:, i]
        issue_targets = targets[:, i].astype(int)

        accuracy = accuracy_score(issue_targets, issue_preds)

        precision, recall, f1, _ = precision_recall_fscore_support(
            issue_targets, issue_preds, average="binary", zero_division=0
        )

        try:
            auc_score = roc_auc_score(issue_targets, predictions[:, i])
        except Exception:
            auc_score = 0.0

        per_issue_metrics[issue_name] = {
            "accuracy": float(accuracy),
            "precision": float(precision),
            "recall": float(recall),
            "f1": float(f1),
            "auc": float(auc_score),
            "support": int(np.sum(issue_targets)),
            "predicted_positive": int(np.sum(issue_preds)),
        }

    micro_precision = precision_recall_fscore_support(
        targets.flatten(), binary_preds.flatten(), average="micro", zero_division=0
    )[0]
    micro_recall = precision_recall_fscore_support(
        targets.flatten(), binary_preds.flatten(), average="micro", zero_division=0
    )[1]
    micro_f1 = precision_recall_fscore_support(
        targets.flatten(), binary_preds.flatten(), average="micro", zero_division=0
    )[2]

    macro_precision = np.mean([m["precision"] for m in per_issue_metrics.values()])
    macro_recall = np.mean([m["recall"] for m in per_issue_metrics.values()])
    macro_f1 = np.mean([m["f1"] for m in per_issue_metrics.values()])

    hamming_loss = np.mean(binary_preds != targets)

    return {
        "per_issue": per_issue_metrics,
        "micro_precision": float(micro_precision),
        "micro_recall": float(micro_recall),
        "micro_f1": float(micro_f1),
        "macro_precision": float(macro_precision),
        "macro_recall": float(macro_recall),
        "macro_f1": float(macro_f1),
        "hamming_loss": float(hamming_loss),
        "threshold": float(threshold),
    }


def evaluate_model_comprehensive(
    model: nn.Module,
    dataloader: DataLoader,
    device: torch.device,
    acceptance_threshold: float = 0.5,
) -> Dict[str, Any]:
    model.eval()
    model = model.to(device)
    model.eval()
    model = model.to(device)

    all_quality_scores = []
    all_quality_targets = []
    all_quality_class_preds = []
    all_quality_class_probs = []
    all_quality_class_targets = []
    all_issue_probs = []
    all_issue_targets = []

    logger.info("Running inference on evaluation dataset...")

    with torch.no_grad():
        for batch in dataloader:
            images = batch["image"].to(device)
            quality_targets = batch["quality_score"]
            quality_class_targets = batch["quality_class"]
            issue_targets = batch["issues"]

            outputs = model(images)

            quality_scores = outputs["quality_score"].cpu().numpy()
            quality_class_logits = outputs["quality_class_logits"].cpu()
            quality_class_probs = torch.softmax(quality_class_logits, dim=1).numpy()
            quality_class_preds = torch.argmax(quality_class_logits, dim=1).numpy()
            issue_probs = torch.sigmoid(outputs["issue_logits"]).cpu().numpy()

            all_quality_scores.extend(quality_scores)
            all_quality_targets.extend(quality_targets.numpy())
            all_quality_class_preds.extend(quality_class_preds)
            all_quality_class_probs.append(quality_class_probs)
            all_quality_class_targets.extend(quality_class_targets.numpy())
            all_issue_probs.append(issue_probs)
            all_issue_targets.append(issue_targets.numpy())

    all_quality_scores = np.array(all_quality_scores)
    all_quality_targets = np.array(all_quality_targets)
    all_quality_class_preds = np.array(all_quality_class_preds)
    all_quality_class_probs = np.vstack(all_quality_class_probs)
    all_quality_class_targets = np.array(all_quality_class_targets)
    all_issue_probs = np.vstack(all_issue_probs)
    all_issue_targets = np.vstack(all_issue_targets)

    logger.info("Calculating comprehensive metrics...")

    regression_metrics = calculate_regression_metrics(
        all_quality_scores, all_quality_targets
    )

    classification_metrics = calculate_classification_metrics(
        all_quality_class_preds, all_quality_class_targets
    )

    binary_targets = (all_quality_targets >= acceptance_threshold).astype(int)
    binary_metrics = calculate_binary_classification_metrics(
        all_quality_scores, binary_targets, threshold=acceptance_threshold
    )

    calibration_metrics = calculate_calibration_metrics(
        all_quality_scores, binary_targets
    )

    issue_metrics = calculate_issue_detection_metrics(
        all_issue_probs, all_issue_targets
    )

    thresholds = np.linspace(0, 1, 101)
    threshold_metrics = []
    for thresh in thresholds:
        metrics = calculate_binary_classification_metrics(
            all_quality_scores, binary_targets, threshold=thresh
        )
        threshold_metrics.append(
            {
                "threshold": thresh,
                "f1": metrics["f1"],
                "precision": metrics["precision"],
                "recall": metrics["recall"],
                "accuracy": metrics["accuracy"],
            }
        )

    best_threshold_idx = np.argmax([m["f1"] for m in threshold_metrics])
    optimal_threshold = threshold_metrics[best_threshold_idx]

    logger.info("Evaluation complete!")

    return {
        "regression": regression_metrics,
        "classification": classification_metrics,
        "binary_classification": binary_metrics,
        "calibration": calibration_metrics,
        "issue_detection": issue_metrics,
        "threshold_analysis": threshold_metrics,
        "optimal_threshold": optimal_threshold,
        "raw_predictions": {
            "quality_scores": all_quality_scores.tolist(),
            "quality_class_preds": all_quality_class_preds.tolist(),
            "quality_class_probs": all_quality_class_probs.tolist(),
            "quality_targets": all_quality_targets.tolist(),
            "quality_class_targets": all_quality_class_targets.tolist(),
        },
        "n_samples": len(all_quality_scores),
    }
