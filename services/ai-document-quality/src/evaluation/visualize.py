import logging
from pathlib import Path
from typing import Dict, List, Optional

import matplotlib.pyplot as plt
import numpy as np
import seaborn as sns
from matplotlib.figure import Figure
from sklearn.metrics import (
    auc,
    precision_recall_curve,
    roc_curve,
)

logger = logging.getLogger(__name__)

sns.set_style("whitegrid")
plt.rcParams["figure.figsize"] = (12, 8)
plt.rcParams["font.size"] = 10


def plot_calibration_curve(
    calibration_metrics: Dict,
    title: str = "Calibration Curve",
    save_path: Optional[Path] = None,
) -> Figure:
    fig, ax = plt.subplots(figsize=(10, 8))

    bin_accs = calibration_metrics["bin_accuracies"]
    bin_confs = calibration_metrics["bin_confidences"]
    bin_counts = calibration_metrics["bin_counts"]
    ece = calibration_metrics["ece"]

    ax.plot([0, 1], [0, 1], "k--", label="Perfect Calibration", linewidth=2)
    sizes = [count * 10 for count in bin_counts]
    scatter = ax.scatter(
        bin_confs,
        bin_accs,
        s=sizes,
        alpha=0.6,
        c=range(len(bin_confs)),
        cmap="viridis",
        edgecolors="black",
        linewidth=1.5,
    )

    ax.plot(bin_confs, bin_accs, "o-", alpha=0.5, color="blue", linewidth=2)

    ax.set_title(f"{title}\nExpected Calibration Error (ECE): {ece:.4f}", fontsize=14)
    ax.set_xlabel("Predicted Confidence", fontsize=12)
    ax.set_ylabel("Actual Accuracy", fontsize=12)
    ax.set_xlim([0, 1])
    ax.set_ylim([0, 1])
    ax.legend(fontsize=12)
    ax.grid(True, alpha=0.3)

    cbar = plt.colorbar(scatter, ax=ax)
    cbar.set_label("Bin Index", fontsize=12)

    plt.tight_layout()

    if save_path:
        fig.savefig(save_path, dpi=300, bbox_inches="tight")
        logger.info(f"Saved calibration curve to {save_path}")

    return fig


def plot_confusion_matrix(
    confusion_matrix: np.ndarray,
    class_names: Optional[List[str]] = None,
    title: str = "Confusion Matrix",
    save_path: Optional[Path] = None,
    normalize: bool = True,
) -> Figure:
    if class_names is None:
        class_names = ["High", "Good", "Moderate", "Poor", "Very Poor"]

    class_names = class_names[: confusion_matrix.shape[0]]

    fig, ax = plt.subplots(figsize=(10, 8))

    cm = confusion_matrix
    if normalize:
        cm = cm.astype("float") / (cm.sum(axis=1, keepdims=True) + 1e-8)
        fmt = ".2%"
        vmax = 1.0
    else:
        fmt = "d"
        vmax = None

    sns.heatmap(
        cm,
        annot=True,
        fmt=fmt,
        cmap="Blues",
        xticklabels=class_names,
        yticklabels=class_names,
        cbar_kws={"label": "Percentage" if normalize else "Count"},
        ax=ax,
        vmin=0,
        vmax=vmax,
    )

    ax.set_title(title, fontsize=14)
    ax.set_ylabel("True Label", fontsize=12)
    ax.set_xlabel("Predicted Label", fontsize=12)

    plt.xticks(rotation=45, ha="right")
    plt.yticks(rotation=0)

    plt.tight_layout()

    if save_path:
        fig.savefig(save_path, dpi=300, bbox_inches="tight")
        logger.info(f"Saved confusion matrix to {save_path}")

    return fig


def plot_roc_curves(
    predictions: np.ndarray,
    targets: np.ndarray,
    title: str = "ROC Curve - Accept/Reject Decision",
    save_path: Optional[Path] = None,
) -> Figure:
    fig, ax = plt.subplots(figsize=(10, 8))

    fpr, tpr, thresholds = roc_curve(targets, predictions)
    roc_auc = auc(fpr, tpr)

    ax.plot(
        fpr, tpr, color="darkorange", lw=2, label=f"ROC curve (AUC = {roc_auc:.3f})"
    )

    ax.plot(
        [0, 1], [0, 1], color="navy", lw=2, linestyle="--", label="Random Classifier"
    )

    distances = np.sqrt((fpr - 0) ** 2 + (tpr - 1) ** 2)
    optimal_idx = np.argmin(distances)
    optimal_threshold = thresholds[optimal_idx]
    optimal_fpr = fpr[optimal_idx]
    optimal_tpr = tpr[optimal_idx]

    ax.plot(
        optimal_fpr,
        optimal_tpr,
        "ro",
        markersize=10,
        label=f"Optimal Threshold = {optimal_threshold:.3f}",
    )

    ax.set_xlim([0.0, 1.0])
    ax.set_ylim([0.0, 1.05])
    ax.set_xlabel("False Positive Rate", fontsize=12)
    ax.set_ylabel("True Positive Rate", fontsize=12)
    ax.set_title(title, fontsize=14)
    ax.legend(loc="lower right", fontsize=12)
    ax.grid(True, alpha=0.3)

    plt.tight_layout()

    if save_path:
        fig.savefig(save_path, dpi=300, bbox_inches="tight")
        logger.info(f"Saved ROC curve to {save_path}")

    return fig


def plot_precision_recall_curve(
    predictions: np.ndarray,
    targets: np.ndarray,
    title: str = "Precision-Recall Curve",
    save_path: Optional[Path] = None,
) -> Figure:
    fig, ax = plt.subplots(figsize=(10, 8))

    precision, recall, thresholds = precision_recall_curve(targets, predictions)
    pr_auc = auc(recall, precision)

    ax.plot(
        recall, precision, color="blue", lw=2, label=f"PR curve (AUC = {pr_auc:.3f})"
    )

    baseline = np.sum(targets) / len(targets)
    ax.plot(
        [0, 1],
        [baseline, baseline],
        color="red",
        linestyle="--",
        label="Random Classifier",
    )

    f1_scores = (
        2 * (precision[:-1] * recall[:-1]) / (precision[:-1] + recall[:-1] + 1e-8)
    )
    best_f1_idx = np.argmax(f1_scores)
    best_threshold = thresholds[best_f1_idx]
    best_precision = precision[best_f1_idx]
    best_recall = recall[best_f1_idx]
    best_f1 = f1_scores[best_f1_idx]

    ax.plot(
        best_recall,
        best_precision,
        "go",
        markersize=10,
        label=f"Best F1 = {best_f1:.3f} (threshold = {best_threshold:.3f})",
    )

    ax.set_xlim([0.0, 1.0])
    ax.set_ylim([0.0, 1.05])
    ax.set_xlabel("Recall", fontsize=12)
    ax.set_ylabel("Precision", fontsize=12)
    ax.set_title(title, fontsize=14)
    ax.legend(loc="lower left", fontsize=12)
    ax.grid(True, alpha=0.3)

    plt.tight_layout()

    if save_path:
        fig.savefig(save_path, dpi=300, bbox_inches="tight")
        logger.info(f"Saved PR curve to {save_path}")

    return fig


def plot_prediction_distribution(
    predictions: np.ndarray,
    targets: np.ndarray,
    title: str = "Quality Score Distribution",
    save_path: Optional[Path] = None,
) -> Figure:
    fig, axes = plt.subplots(2, 2, figsize=(14, 10))

    ax = axes[0, 0]
    ax.scatter(targets, predictions, alpha=0.5, s=20)
    ax.plot([0, 1], [0, 1], "r--", lw=2, label="Perfect Prediction")
    ax.set_xlabel("Ground Truth Quality Score", fontsize=11)
    ax.set_ylabel("Predicted Quality Score", fontsize=11)
    ax.set_title("Predictions vs Ground Truth", fontsize=12)
    ax.legend()
    ax.grid(True, alpha=0.3)

    ax = axes[0, 1]
    ax.hist(
        targets,
        bins=30,
        alpha=0.5,
        label="Ground Truth",
        color="blue",
        edgecolor="black",
    )
    ax.hist(
        predictions,
        bins=30,
        alpha=0.5,
        label="Predictions",
        color="orange",
        edgecolor="black",
    )
    ax.set_xlabel("Quality Score", fontsize=11)
    ax.set_ylabel("Frequency", fontsize=11)
    ax.set_title("Score Distributions", fontsize=12)
    ax.legend()
    ax.grid(True, alpha=0.3)

    ax = axes[1, 0]
    residuals = predictions - targets
    ax.scatter(predictions, residuals, alpha=0.5, s=20)
    ax.axhline(y=0, color="r", linestyle="--", lw=2)
    ax.set_xlabel("Predicted Quality Score", fontsize=11)
    ax.set_ylabel("Residual (Predicted - Actual)", fontsize=11)
    ax.set_title("Residuals Plot", fontsize=12)
    ax.grid(True, alpha=0.3)

    ax = axes[1, 1]
    categories = np.digitize(targets, bins=[0.2, 0.4, 0.6, 0.8]) - 1
    category_names = ["Very Poor", "Poor", "Moderate", "Good", "High"]

    data_by_category = [predictions[categories == i] for i in range(5)]
    positions = range(1, 6)

    bp = ax.boxplot(
        data_by_category,
        positions=positions,
        labels=category_names,
        patch_artist=True,
        showmeans=True,
    )

    colors = ["#d73027", "#fc8d59", "#fee08b", "#91cf60", "#1a9850"]
    for patch, color in zip(bp["boxes"], colors):
        patch.set_facecolor(color)
        patch.set_alpha(0.7)

    ax.set_ylabel("Predicted Quality Score", fontsize=11)
    ax.set_xlabel("Ground Truth Category", fontsize=11)
    ax.set_title("Predictions by True Quality Category", fontsize=12)
    ax.grid(True, alpha=0.3, axis="y")
    plt.xticks(rotation=45)

    fig.suptitle(title, fontsize=16, y=0.995)
    plt.tight_layout()

    if save_path:
        fig.savefig(save_path, dpi=300, bbox_inches="tight")
        logger.info(f"Saved prediction distribution to {save_path}")

    return fig


def plot_issue_analysis(
    issue_metrics: Dict,
    title: str = "Issue Detection Performance",
    save_path: Optional[Path] = None,
) -> Figure:
    per_issue = issue_metrics["per_issue"]
    issue_names = list(per_issue.keys())

    precisions = [per_issue[name]["precision"] for name in issue_names]
    recalls = [per_issue[name]["recall"] for name in issue_names]
    f1_scores = [per_issue[name]["f1"] for name in issue_names]
    supports = [per_issue[name]["support"] for name in issue_names]

    fig, axes = plt.subplots(2, 1, figsize=(14, 10))

    ax = axes[0]
    x = np.arange(len(issue_names))
    width = 0.25

    bars1 = ax.bar(x - width, precisions, width, label="Precision", alpha=0.8)
    bars2 = ax.bar(x, recalls, width, label="Recall", alpha=0.8)
    bars3 = ax.bar(x + width, f1_scores, width, label="F1 Score", alpha=0.8)

    ax.set_xlabel("Issue Type", fontsize=12)
    ax.set_ylabel("Score", fontsize=12)
    ax.set_title("Per-Issue Detection Metrics", fontsize=13)
    ax.set_xticks(x)
    ax.set_xticklabels(
        [name.replace("_", " ").title() for name in issue_names],
        rotation=45,
        ha="right",
    )
    ax.legend()
    ax.grid(True, alpha=0.3, axis="y")
    ax.set_ylim([0, 1.05])

    ax = axes[1]
    colors = plt.cm.viridis(np.linspace(0, 1, len(issue_names)))
    bars = ax.barh(range(len(issue_names)), supports, color=colors, alpha=0.8)

    ax.set_yticks(range(len(issue_names)))
    ax.set_yticklabels([name.replace("_", " ").title() for name in issue_names])
    ax.set_xlabel("Number of Samples with Issue", fontsize=12)
    ax.set_title("Issue Frequency in Dataset", fontsize=13)
    ax.grid(True, alpha=0.3, axis="x")

    for i, (bar, support) in enumerate(zip(bars, supports)):
        ax.text(
            support + max(supports) * 0.01,
            i,
            f"{support}",
            va="center",
            fontsize=10,
        )

    fig.suptitle(title, fontsize=16, y=0.995)
    plt.tight_layout()

    if save_path:
        fig.savefig(save_path, dpi=300, bbox_inches="tight")
        logger.info(f"Saved issue analysis to {save_path}")

    return fig


def plot_threshold_analysis(
    threshold_metrics: List[Dict],
    title: str = "Threshold Analysis",
    save_path: Optional[Path] = None,
) -> Figure:
    fig, ax = plt.subplots(figsize=(12, 8))

    thresholds = [m["threshold"] for m in threshold_metrics]
    f1_scores = [m["f1"] for m in threshold_metrics]
    precisions = [m["precision"] for m in threshold_metrics]
    recalls = [m["recall"] for m in threshold_metrics]
    accuracies = [m["accuracy"] for m in threshold_metrics]

    ax.plot(thresholds, f1_scores, "o-", label="F1 Score", linewidth=2, markersize=4)
    ax.plot(thresholds, precisions, "s-", label="Precision", linewidth=2, markersize=4)
    ax.plot(thresholds, recalls, "^-", label="Recall", linewidth=2, markersize=4)
    ax.plot(thresholds, accuracies, "d-", label="Accuracy", linewidth=2, markersize=4)

    best_f1_idx = np.argmax(f1_scores)
    best_threshold = thresholds[best_f1_idx]
    best_f1 = f1_scores[best_f1_idx]

    ax.axvline(
        x=best_threshold,
        color="red",
        linestyle="--",
        linewidth=2,
        alpha=0.7,
        label=f"Optimal Threshold = {best_threshold:.3f}",
    )
    ax.plot(best_threshold, best_f1, "r*", markersize=20)

    ax.set_xlabel("Decision Threshold", fontsize=12)
    ax.set_ylabel("Score", fontsize=12)
    ax.set_title(title, fontsize=14)
    ax.set_xlim([0, 1])
    ax.set_ylim([0, 1.05])
    ax.legend(loc="best", fontsize=11)
    ax.grid(True, alpha=0.3)

    plt.tight_layout()

    if save_path:
        fig.savefig(save_path, dpi=300, bbox_inches="tight")
        logger.info(f"Saved threshold analysis to {save_path}")

    return fig


def create_evaluation_report(
    evaluation_results: Dict, output_dir: Path, model_name: str = "model"
) -> Path:
    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    logger.info(f"Creating evaluation report in {output_dir}")

    quality_scores = np.array(evaluation_results["raw_predictions"]["quality_scores"])
    quality_targets = np.array(evaluation_results["raw_predictions"]["quality_targets"])
    quality_class_preds = np.array(
        evaluation_results["raw_predictions"]["quality_class_preds"]
    )
    quality_class_targets = np.array(
        evaluation_results["raw_predictions"]["quality_class_targets"]
    )

    binary_targets = (quality_targets >= 0.5).astype(int)

    plot_calibration_curve(
        evaluation_results["calibration"],
        save_path=output_dir / f"{model_name}_calibration.png",
    )
    plt.close()

    cm = np.array(evaluation_results["classification"]["confusion_matrix"])
    plot_confusion_matrix(
        cm, save_path=output_dir / f"{model_name}_confusion_matrix.png"
    )
    plt.close()

    plot_roc_curves(
        quality_scores,
        binary_targets,
        save_path=output_dir / f"{model_name}_roc_curve.png",
    )
    plt.close()

    plot_precision_recall_curve(
        quality_scores,
        binary_targets,
        save_path=output_dir / f"{model_name}_pr_curve.png",
    )
    plt.close()

    plot_prediction_distribution(
        quality_scores,
        quality_targets,
        save_path=output_dir / f"{model_name}_predictions.png",
    )
    plt.close()

    plot_issue_analysis(
        evaluation_results["issue_detection"],
        save_path=output_dir / f"{model_name}_issue_analysis.png",
    )
    plt.close()

    plot_threshold_analysis(
        evaluation_results["threshold_analysis"],
        save_path=output_dir / f"{model_name}_threshold_analysis.png",
    )
    plt.close()

    report_path = output_dir / f"{model_name}_evaluation_report.txt"
    with open(report_path, "w") as f:
        f.write("=" * 80 + "\n")
        f.write(f"EVALUATION REPORT: {model_name}\n")
        f.write("=" * 80 + "\n\n")

        f.write(f"Number of samples: {evaluation_results['n_samples']}\n\n")

        f.write("REGRESSION METRICS (Quality Scores)\n")
        f.write("-" * 80 + "\n")
        reg = evaluation_results["regression"]
        f.write(f"  MAE:  {reg['mae']:.4f}\n")
        f.write(f"  RMSE: {reg['rmse']:.4f}\n")
        f.write(f"  R²:   {reg['r2']:.4f}\n")
        f.write(f"  Predictions within 5%:  {reg['within_5pct']:.2%}\n")
        f.write(f"  Predictions within 10%: {reg['within_10pct']:.2%}\n")
        f.write(f"  Predictions within 20%: {reg['within_20pct']:.2%}\n\n")

        f.write("CLASSIFICATION METRICS (Quality Classes)\n")
        f.write("-" * 80 + "\n")
        cls = evaluation_results["classification"]
        f.write(f"  Accuracy:          {cls['accuracy']:.4f}\n")
        f.write(f"  Balanced Accuracy: {cls['balanced_accuracy']:.4f}\n")
        f.write(f"  Weighted F1:       {cls['weighted_f1']:.4f}\n")
        f.write(f"  Macro F1:          {cls['macro_f1']:.4f}\n\n")

        f.write("  Per-Class Metrics:\n")
        for class_name, metrics in cls["per_class"].items():
            f.write(f"    {class_name}:\n")
            f.write(f"      Precision: {metrics['precision']:.4f}\n")
            f.write(f"      Recall:    {metrics['recall']:.4f}\n")
            f.write(f"      F1 Score:  {metrics['f1']:.4f}\n")
            f.write(f"      Support:   {metrics['support']}\n")

        f.write("\n")

        f.write("BINARY CLASSIFICATION (Accept/Reject @ 0.5 threshold)\n")
        f.write("-" * 80 + "\n")
        binary = evaluation_results["binary_classification"]
        f.write(f"  Accuracy:   {binary['accuracy']:.4f}\n")
        f.write(f"  Precision:  {binary['precision']:.4f}\n")
        f.write(f"  Recall:     {binary['recall']:.4f}\n")
        f.write(f"  F1 Score:   {binary['f1']:.4f}\n")
        f.write(f"  ROC AUC:    {binary['roc_auc']:.4f}\n")
        f.write(f"  PR AUC:     {binary['pr_auc']:.4f}\n")
        f.write(f"  FPR (False Reject Rate): {binary['fpr']:.4f}\n")
        f.write(f"  FNR (False Accept Rate): {binary['fnr']:.4f}\n\n")

        f.write("CALIBRATION METRICS\n")
        f.write("-" * 80 + "\n")
        cal = evaluation_results["calibration"]
        f.write(f"  Expected Calibration Error (ECE): {cal['ece']:.4f}\n")
        f.write(f"  Maximum Calibration Error (MCE):  {cal['mce']:.4f}\n\n")

        f.write("OPTIMAL THRESHOLD ANALYSIS\n")
        f.write("-" * 80 + "\n")
        opt = evaluation_results["optimal_threshold"]
        f.write(f"  Threshold:  {opt['threshold']:.4f}\n")
        f.write(f"  F1 Score:   {opt['f1']:.4f}\n")
        f.write(f"  Precision:  {opt['precision']:.4f}\n")
        f.write(f"  Recall:     {opt['recall']:.4f}\n")
        f.write(f"  Accuracy:   {opt['accuracy']:.4f}\n\n")

        f.write("ISSUE DETECTION METRICS\n")
        f.write("-" * 80 + "\n")
        issues = evaluation_results["issue_detection"]
        f.write(f"  Micro F1:    {issues['micro_f1']:.4f}\n")
        f.write(f"  Macro F1:    {issues['macro_f1']:.4f}\n")
        f.write(f"  Hamming Loss: {issues['hamming_loss']:.4f}\n\n")

        f.write("  Per-Issue Metrics:\n")
        for issue_name, metrics in issues["per_issue"].items():
            f.write(f"    {issue_name.replace('_', ' ').title()}:\n")
            f.write(f"      F1 Score:  {metrics['f1']:.4f}\n")
            f.write(f"      Precision: {metrics['precision']:.4f}\n")
            f.write(f"      Recall:    {metrics['recall']:.4f}\n")
            f.write(f"      Support:   {metrics['support']}\n")

        f.write("\n" + "=" * 80 + "\n")

    logger.info(f"Evaluation report saved to {report_path}")
    logger.info(f"All plots saved to {output_dir}")

    return output_dir
