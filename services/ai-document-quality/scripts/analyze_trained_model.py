#!/usr/bin/env python3
#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
"""
Analyze a trained model to understand its performance and behavior
"""

import argparse
import json
import os
from pathlib import Path
from typing import Any, Dict, Optional

import matplotlib.pyplot as plt
import numpy as np
import seaborn as sns
import torch
import torch.nn as nn
import yaml
from sklearn.metrics import classification_report, confusion_matrix, r2_score
from torch.utils.data import DataLoader

from src.data.dataset import DocumentDataset
from src.models.model import DocumentQualityModel, ModelConfig


def load_config(config_path: str) -> Dict[str, Any]:
    """Load configuration from YAML file"""
    with open(config_path, "r") as f:
        return yaml.safe_load(f)


def load_model_from_checkpoint(
    checkpoint_path: str, config: Dict[str, Any]
) -> nn.Module:
    """Load model from checkpoint"""
    # Create model config
    model_config = ModelConfig(
        backbone=config.get("model.backbone", "efficientnet_b0"),
        num_quality_classes=config.get("model.num_quality_classes", 5),
        num_issue_classes=config.get("model.num_issue_classes", 10),
        hidden_dim=config.get("model.hidden_dim", 256),
        dropout_rate=config.get("model.dropout_rate", 0.5),
        use_attention=config.get("model.use_attention", True),
        freeze_backbone_layers=config.get("model.freeze_backbone_layers", 5),
    )

    # Create model
    model = DocumentQualityModel(model_config)

    # Load checkpoint
    checkpoint = torch.load(checkpoint_path, map_location="cpu")

    # Handle different checkpoint formats
    if isinstance(checkpoint, dict):
        if "model_state_dict" in checkpoint:
            model.load_state_dict(checkpoint["model_state_dict"])
        elif "state_dict" in checkpoint:
            model.load_state_dict(checkpoint["state_dict"])
        else:
            # Assume the checkpoint is the state dict itself
            model.load_state_dict(checkpoint)
    else:
        model.load_state_dict(checkpoint)

    model.eval()
    return model


def analyze_predictions(
    model: nn.Module, dataloader: DataLoader, device: torch.device
) -> Dict[str, Any]:
    """Analyze model predictions on a dataset"""
    model.eval()
    model = model.to(device)

    all_quality_scores = []
    all_quality_targets = []
    all_quality_preds = []
    all_quality_targets_class = []
    all_issue_preds = []
    all_issue_targets = []

    with torch.no_grad():
        for batch in dataloader:
            images = batch["image"].to(device)
            quality_targets = batch["quality_score"].to(device)
            quality_class_targets = batch["quality_class"].to(device)
            issue_targets = batch["issues"].to(device)

            outputs = model(images)

            # Collect predictions
            quality_scores = outputs["quality_score"].cpu().numpy()
            quality_preds = (
                torch.argmax(outputs["quality_class_logits"], dim=1).cpu().numpy()
            )
            issue_preds = torch.sigmoid(outputs["issue_logits"]).cpu().numpy()

            all_quality_scores.extend(quality_scores)
            all_quality_targets.extend(quality_targets.cpu().numpy())
            all_quality_preds.extend(quality_preds)
            all_quality_targets_class.extend(quality_class_targets.cpu().numpy())
            all_issue_preds.append(issue_preds)
            all_issue_targets.append(issue_targets.cpu().numpy())

    all_quality_scores = np.array(all_quality_scores)
    all_quality_targets = np.array(all_quality_targets)
    all_quality_preds = np.array(all_quality_preds)
    all_quality_targets_class = np.array(all_quality_targets_class)
    all_issue_preds = np.vstack(all_issue_preds)
    all_issue_targets = np.vstack(all_issue_targets)

    # Calculate metrics
    results = {
        "regression": {
            "predictions": all_quality_scores,
            "targets": all_quality_targets,
            "mae": np.mean(np.abs(all_quality_scores - all_quality_targets)),
            "rmse": np.sqrt(np.mean((all_quality_scores - all_quality_targets) ** 2)),
            "r2": r2_score(all_quality_targets, all_quality_scores),
            "mean_pred": np.mean(all_quality_scores),
            "std_pred": np.std(all_quality_scores),
            "min_pred": np.min(all_quality_scores),
            "max_pred": np.max(all_quality_scores),
        },
        "classification": {
            "predictions": all_quality_preds,
            "targets": all_quality_targets_class,
            "accuracy": np.mean(all_quality_preds == all_quality_targets_class),
            "confusion_matrix": confusion_matrix(
                all_quality_targets_class, all_quality_preds
            ),
            "class_distribution": np.bincount(all_quality_preds, minlength=5),
        },
        "issues": {
            "predictions": all_issue_preds,
            "targets": all_issue_targets,
            "per_class_accuracy": np.mean(
                (all_issue_preds > 0.5) == all_issue_targets, axis=0
            ),
        },
    }

    return results


def plot_analysis(results: Dict[str, Any], output_dir: Path):
    """Create visualization plots for model analysis"""
    output_dir.mkdir(exist_ok=True)

    # 1. Regression scatter plot
    plt.figure(figsize=(10, 8))
    plt.scatter(
        results["regression"]["targets"],
        results["regression"]["predictions"],
        alpha=0.5,
        s=10,
    )
    plt.plot([0, 1], [0, 1], "r--", lw=2)
    plt.xlabel("True Quality Score")
    plt.ylabel("Predicted Quality Score")
    plt.title(f"Quality Score Predictions (R² = {results['regression']['r2']:.3f})")
    plt.grid(True, alpha=0.3)
    plt.savefig(output_dir / "regression_scatter.png", dpi=150, bbox_inches="tight")
    plt.close()

    # 2. Prediction distribution
    plt.figure(figsize=(12, 5))

    plt.subplot(1, 2, 1)
    plt.hist(
        results["regression"]["predictions"], bins=50, alpha=0.7, label="Predictions"
    )
    plt.hist(results["regression"]["targets"], bins=50, alpha=0.7, label="Targets")
    plt.xlabel("Quality Score")
    plt.ylabel("Frequency")
    plt.title("Distribution of Quality Scores")
    plt.legend()

    plt.subplot(1, 2, 2)
    plt.hist(
        results["regression"]["predictions"] - results["regression"]["targets"],
        bins=50,
        alpha=0.7,
    )
    plt.xlabel("Prediction Error")
    plt.ylabel("Frequency")
    plt.title("Distribution of Prediction Errors")

    plt.tight_layout()
    plt.savefig(output_dir / "distributions.png", dpi=150, bbox_inches="tight")
    plt.close()

    # 3. Confusion Matrix
    plt.figure(figsize=(8, 6))
    sns.heatmap(
        results["classification"]["confusion_matrix"],
        annot=True,
        fmt="d",
        cmap="Blues",
        xticklabels=["Very Poor", "Poor", "Moderate", "Good", "High"],
        yticklabels=["Very Poor", "Poor", "Moderate", "Good", "High"],
    )
    plt.xlabel("Predicted")
    plt.ylabel("True")
    plt.title(
        f"Classification Confusion Matrix (Acc = {results['classification']['accuracy']:.3f})"
    )
    plt.savefig(output_dir / "confusion_matrix.png", dpi=150, bbox_inches="tight")
    plt.close()

    # 4. Issue detection performance
    issue_names = [
        "Blur",
        "Noise",
        "Low Contrast",
        "Skew",
        "Border",
        "Shadow",
        "Glare",
        "Fold",
        "Stain",
        "Tear",
    ]

    plt.figure(figsize=(10, 6))
    plt.bar(range(len(issue_names)), results["issues"]["per_class_accuracy"])
    plt.xticks(range(len(issue_names)), issue_names, rotation=45)
    plt.ylabel("Accuracy")
    plt.title("Per-Issue Detection Accuracy")
    plt.grid(True, alpha=0.3, axis="y")
    plt.tight_layout()
    plt.savefig(output_dir / "issue_detection.png", dpi=150, bbox_inches="tight")
    plt.close()


def print_analysis_summary(results: Dict[str, Any]):
    """Print analysis summary to console"""
    print("\n" + "=" * 60)
    print("MODEL ANALYSIS SUMMARY")
    print("=" * 60)

    print("\n📊 REGRESSION METRICS (Quality Score 0-1)")
    print("-" * 40)
    print(f"MAE:              {results['regression']['mae']:.4f}")
    print(f"RMSE:             {results['regression']['rmse']:.4f}")
    print(f"R² Score:         {results['regression']['r2']:.4f}")
    print(f"Mean Prediction:  {results['regression']['mean_pred']:.4f}")
    print(f"Std Prediction:   {results['regression']['std_pred']:.4f}")
    print(
        f"Prediction Range: [{results['regression']['min_pred']:.4f}, {results['regression']['max_pred']:.4f}]"
    )

    print("\n🏷️  CLASSIFICATION METRICS (5 Quality Classes)")
    print("-" * 40)
    print(f"Accuracy:         {results['classification']['accuracy']:.4f}")
    print(f"Class Distribution: {results['classification']['class_distribution']}")

    print("\n⚠️  ISSUE DETECTION METRICS")
    print("-" * 40)
    issue_names = [
        "Blur",
        "Noise",
        "Low Contrast",
        "Skew",
        "Border",
        "Shadow",
        "Glare",
        "Fold",
        "Stain",
        "Tear",
    ]
    for i, (name, acc) in enumerate(
        zip(issue_names, results["issues"]["per_class_accuracy"])
    ):
        print(f"{name:15s}: {acc:.4f}")

    print("\n" + "=" * 60)

    # Diagnosis
    print("\n🔍 DIAGNOSIS:")
    print("-" * 40)

    if results["regression"]["std_pred"] < 0.01:
        print("⚠️  Model is predicting nearly constant values (collapsed)")
    elif results["regression"]["std_pred"] < 0.05:
        print("⚠️  Model has very low prediction variance")
    else:
        print("✅ Model shows reasonable prediction variance")

    if results["regression"]["r2"] < 0:
        print("❌ Negative R² - model is worse than baseline")
    elif results["regression"]["r2"] < 0.3:
        print("⚠️  Low R² - poor predictive performance")
    else:
        print("✅ Reasonable R² score")

    if abs(results["regression"]["mean_pred"] - 0.5) > 0.3:
        print("⚠️  Predictions are biased toward extreme values")

    # Check if model is overfitting based on class distribution
    max_class_ratio = (
        results["classification"]["class_distribution"].max()
        / results["classification"]["class_distribution"].sum()
    )
    if max_class_ratio > 0.8:
        print("⚠️  Model is predicting mostly one class")


def main():
    parser = argparse.ArgumentParser(
        description="Analyze trained document quality model"
    )
    parser.add_argument(
        "--checkpoint",
        type=str,
        help="Path to model checkpoint (default: latest mlflow model)",
    )
    parser.add_argument(
        "--config",
        type=str,
        default="config/best_training.yaml",
        help="Path to config file",
    )
    parser.add_argument("--data-path", type=str, default="data", help="Path to dataset")
    parser.add_argument(
        "--split",
        type=str,
        default="test",
        choices=["train", "val", "test"],
        help="Which data split to analyze",
    )
    parser.add_argument(
        "--output-dir",
        type=str,
        default="analysis_results",
        help="Directory to save analysis results",
    )

    args = parser.parse_args()

    # Load config
    config = load_config(args.config)

    # Find checkpoint
    if args.checkpoint:
        checkpoint_path = args.checkpoint
    else:
        # Find latest model
        mlflow_models = Path("mlruns").glob("*/*/artifacts/best_model.pth")
        checkpoint_path = max(mlflow_models, key=os.path.getmtime)
        print(f"Using latest model: {checkpoint_path}")

    # Load model
    print("Loading model...")
    model = load_model_from_checkpoint(str(checkpoint_path), config)

    # Create dataloader
    print(f"Loading {args.split} dataset...")
    dataset = DocumentDataset(
        metadata_file=f"{args.data_path}/{args.split}_metadata.csv",
        transform=args.split if args.split in ["train", "val"] else "test",
    )

    dataloader = DataLoader(dataset, batch_size=32, shuffle=False, num_workers=4)

    print(f"Dataset size: {len(dataset)} samples")

    # Analyze
    print("Analyzing model predictions...")
    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    results = analyze_predictions(model, dataloader, device)

    # Print summary
    print_analysis_summary(results)

    # Save detailed results
    output_dir = Path(args.output_dir)
    output_dir.mkdir(exist_ok=True)

    # Save numerical results
    with open(output_dir / "analysis_results.json", "w") as f:
        json_results = {
            "regression": {
                k: v.tolist() if isinstance(v, np.ndarray) else v
                for k, v in results["regression"].items()
                if k not in ["predictions", "targets"]
            },
            "classification": {
                k: v.tolist() if isinstance(v, np.ndarray) else v
                for k, v in results["classification"].items()
                if k not in ["predictions", "targets"]
            },
            "issues": {
                k: v.tolist() if isinstance(v, np.ndarray) else v
                for k, v in results["issues"].items()
                if k not in ["predictions", "targets"]
            },
        }
        json.dump(json_results, f, indent=2)

    # Create plots
    print(f"\nCreating visualizations in {output_dir}...")
    plot_analysis(results, output_dir)

    print(f"\n✅ Analysis complete! Results saved to {output_dir}/")


if __name__ == "__main__":
    main()
