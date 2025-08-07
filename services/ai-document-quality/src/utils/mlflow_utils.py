#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
import json
import logging
from pathlib import Path
from typing import Any, Dict, Optional

import mlflow
import mlflow.pytorch
import torch
import yaml

logger = logging.getLogger(__name__)


def setup_mlflow(config: Dict[str, Any]) -> str:
    """
    Setup MLflow tracking with configuration

    Args:
        config: Configuration dictionary

    Returns:
        run_id: MLflow run ID
    """
    # Set tracking URI
    tracking_uri = config.get("mlflow.tracking_uri", "mlruns")
    mlflow.set_tracking_uri(tracking_uri)

    # Set experiment
    experiment_name = config.get(
        "mlflow.experiment_name", "document-quality-assessment"
    )
    mlflow.set_experiment(experiment_name)

    # Start run
    run_name = config.get("mlflow.run_name", None)
    run = mlflow.start_run(run_name=run_name)

    logger.info(f"Started MLflow run: {run.info.run_id}")
    logger.info(f"Tracking URI: {tracking_uri}")
    logger.info(f"Experiment: {experiment_name}")

    return run.info.run_id


def log_config(config: Dict[str, Any], config_path: Optional[str] = None):
    """Log configuration to MLflow"""

    # Log configuration as parameters (flattened)
    def flatten_dict(d, parent_key="", sep="."):
        items = []
        for k, v in d.items():
            new_key = f"{parent_key}{sep}{k}" if parent_key else k
            if isinstance(v, dict):
                items.extend(flatten_dict(v, new_key, sep=sep).items())
            else:
                items.append((new_key, v))
        return dict(items)

    flat_config = flatten_dict(config)

    # MLflow has a limit on number of parameters, so log important ones
    important_params = {
        k: v
        for k, v in flat_config.items()
        if any(key in k for key in ["model.", "training.", "dataset."])
    }

    mlflow.log_params(important_params)

    # Log full config as artifact
    if config_path:
        mlflow.log_artifact(config_path, "config")
    else:
        # Save config to temp file and log
        with open("temp_config.yaml", "w") as f:
            yaml.dump(config, f)
        mlflow.log_artifact("temp_config.yaml", "config")
        Path("temp_config.yaml").unlink()


def log_model_info(model: torch.nn.Module):
    """Log model architecture info"""
    # Count parameters
    total_params = sum(p.numel() for p in model.parameters())
    trainable_params = sum(p.numel() for p in model.parameters() if p.requires_grad)

    mlflow.log_params(
        {
            "total_parameters": total_params,
            "trainable_parameters": trainable_params,
        }
    )

    # Log model summary as text
    model_str = str(model)
    with open("model_architecture.txt", "w") as f:
        f.write(model_str)
    mlflow.log_artifact("model_architecture.txt", "model_info")
    Path("model_architecture.txt").unlink()


def log_dataset_info(
    train_size: int,
    val_size: int,
    test_size: int,
    dataset_stats: Optional[Dict[str, Any]] = None,
):
    """Log dataset information"""
    mlflow.log_params(
        {
            "train_size": train_size,
            "val_size": val_size,
            "test_size": test_size,
            "total_size": train_size + val_size + test_size,
        }
    )

    if dataset_stats:
        # Log dataset statistics as JSON artifact
        with open("dataset_stats.json", "w") as f:
            json.dump(dataset_stats, f, indent=2)
        mlflow.log_artifact("dataset_stats.json", "dataset_info")
        Path("dataset_stats.json").unlink()


def log_training_curves(history: Dict[str, list]):
    """Log training history curves"""
    import matplotlib.pyplot as plt

    # Create figure with subplots
    fig, axes = plt.subplots(2, 2, figsize=(12, 10))
    fig.suptitle("Training History")

    # Loss curves
    if "train_loss" in history and "val_loss" in history:
        ax = axes[0, 0]
        ax.plot(history["train_loss"], label="Train")
        ax.plot(history["val_loss"], label="Validation")
        ax.set_title("Total Loss")
        ax.set_xlabel("Epoch")
        ax.set_ylabel("Loss")
        ax.legend()
        ax.grid(True)

    # MAE curves
    if "val_mae" in history:
        ax = axes[0, 1]
        ax.plot(history["val_mae"])
        ax.set_title("Validation MAE")
        ax.set_xlabel("Epoch")
        ax.set_ylabel("MAE")
        ax.grid(True)

    # Accuracy curves
    if "val_accuracy" in history:
        ax = axes[1, 0]
        ax.plot(history["val_accuracy"])
        ax.set_title("Validation Accuracy")
        ax.set_xlabel("Epoch")
        ax.set_ylabel("Accuracy")
        ax.grid(True)

    # Learning rate
    if "learning_rate" in history:
        ax = axes[1, 1]
        ax.plot(history["learning_rate"])
        ax.set_title("Learning Rate")
        ax.set_xlabel("Epoch")
        ax.set_ylabel("LR")
        ax.grid(True)

    plt.tight_layout()
    plt.savefig("training_curves.png", dpi=150)
    mlflow.log_artifact("training_curves.png", "plots")
    Path("training_curves.png").unlink()
    plt.close()


def log_confusion_matrix(y_true, y_pred, class_names):
    """Log confusion matrix for classification"""
    import matplotlib.pyplot as plt
    import seaborn as sns
    from sklearn.metrics import confusion_matrix

    cm = confusion_matrix(y_true, y_pred)

    plt.figure(figsize=(8, 6))
    sns.heatmap(
        cm,
        annot=True,
        fmt="d",
        cmap="Blues",
        xticklabels=class_names,
        yticklabels=class_names,
    )
    plt.title("Confusion Matrix")
    plt.ylabel("True Label")
    plt.xlabel("Predicted Label")
    plt.tight_layout()

    plt.savefig("confusion_matrix.png", dpi=150)
    mlflow.log_artifact("confusion_matrix.png", "plots")
    Path("confusion_matrix.png").unlink()
    plt.close()


def log_predictions_scatter(y_true, y_pred):
    """Log scatter plot of predictions vs true values"""
    import matplotlib.pyplot as plt
    import numpy as np

    plt.figure(figsize=(8, 6))
    plt.scatter(y_true, y_pred, alpha=0.5)

    # Add diagonal line
    min_val = min(y_true.min(), y_pred.min())
    max_val = max(y_true.max(), y_pred.max())
    plt.plot([min_val, max_val], [min_val, max_val], "r--", lw=2)

    plt.xlabel("True Quality Score")
    plt.ylabel("Predicted Quality Score")
    plt.title("Predictions vs True Values")
    plt.grid(True, alpha=0.3)
    plt.tight_layout()

    plt.savefig("predictions_scatter.png", dpi=150)
    mlflow.log_artifact("predictions_scatter.png", "plots")
    Path("predictions_scatter.png").unlink()
    plt.close()


def save_model_for_production(model, model_config, input_shape=(1, 3, 224, 224)):
    """Save model in production-ready format with MLflow"""
    # Create sample input for signature
    sample_input = torch.randn(input_shape)

    # Log model with signature
    mlflow.pytorch.log_model(
        pytorch_model=model,
        name="model",
        signature=mlflow.models.infer_signature(sample_input.numpy()),
        registered_model_name="document-quality-model",
        pip_requirements=[
            "torch",
            "torchvision",
            "pillow",
            "numpy",
            "opencv-python",
        ],
        code_paths=["src/"],
        extra_files={
            "model_config.json": json.dumps(
                model_config.__dict__
                if hasattr(model_config, "__dict__")
                else model_config
            )
        },
    )
