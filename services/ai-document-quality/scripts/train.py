#!/usr/bin/env python3
#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
"""
Training script for Document Quality Assessment Model

This script demonstrates how to train the document quality assessment model
using your transportation document dataset.

Usage:
    python train.py --data-dir /path/to/documents --epochs 20
    python train.py --config config/custom.yaml
    python train.py --resume checkpoint.pth
"""

import argparse
import logging
from pathlib import Path

import mlflow
import mlflow.pytorch
import numpy as np
import torch
from sklearn.metrics import (
    accuracy_score,
    mean_absolute_error,
    mean_squared_error,
    r2_score,
)
from torch.utils.data import DataLoader
from tqdm import tqdm

from src.data.dataset import DocumentDataset, create_dataset_from_folder
from src.models.model import DocumentQualityModel, ModelConfig, train_epoch
from src.utils.config import get_config
from src.training.advanced_strategies import (
    AdversarialTraining,
    MixupLoss,
    MixupTrainer,
    create_curriculum_dataloader,
    warm_restart_scheduler,
)
from src.data.augmentations import MixupAugmentation

# Configure logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


def setup_training(args):
    """Setup training configuration and datasets"""
    # Load configuration
    config = get_config(args.config)

    # Create model configuration
    model_config = ModelConfig(
        backbone=config.get("model.backbone", "efficientnet_b0"),
        num_quality_classes=config.get("model.num_quality_classes", 5),
        num_issue_classes=config.get("model.num_issue_classes", 10),
        hidden_dim=config.get("model.hidden_dim", 256),
        dropout_rate=config.get("model.dropout_rate", 0.3),
        use_attention=config.get("model.use_attention", True),
    )

    # Setup device
    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    logger.info(f"Using device: {device}")

    # Create or load datasets
    if args.data_dir:
        logger.info(f"Creating dataset from folder: {args.data_dir}")
        train_dataset, val_dataset, test_dataset = create_dataset_from_folder(
            folder_path=args.data_dir,
            train_ratio=config.get("dataset.train_ratio", 0.7),
            val_ratio=config.get("dataset.val_ratio", 0.2),
            test_ratio=config.get("dataset.test_ratio", 0.1),
        )
    else:
        # Try to load from default dataset paths
        logger.info("Loading existing datasets from default paths")
        datasets_dir = Path(config.get("paths.datasets_dir", "datasets"))

        # Check if metadata files exist - try multiple possible locations
        train_metadata_candidates = [
            datasets_dir / "default" / "train" / "train_metadata.csv",
            datasets_dir / "train" / "metadata.csv",
            datasets_dir / "train" / "train_metadata.csv",
        ]
        val_metadata_candidates = [
            datasets_dir / "default" / "val" / "val_metadata.csv",
            datasets_dir / "val" / "metadata.csv",
            datasets_dir / "val" / "val_metadata.csv",
        ]
        test_metadata_candidates = [
            datasets_dir / "default" / "test" / "test_metadata.csv",
            datasets_dir / "test" / "metadata.csv",
            datasets_dir / "test" / "test_metadata.csv",
        ]

        # Find existing metadata files
        train_metadata = None
        val_metadata = None
        test_metadata = None

        for candidate in train_metadata_candidates:
            if candidate.exists():
                train_metadata = candidate
                break
        for candidate in val_metadata_candidates:
            if candidate.exists():
                val_metadata = candidate
                break
        for candidate in test_metadata_candidates:
            if candidate.exists():
                test_metadata = candidate
                break

        if train_metadata and val_metadata and test_metadata:
            train_dataset = DocumentDataset(
                metadata_file=str(train_metadata), transform="train"
            )
            val_dataset = DocumentDataset(
                metadata_file=str(val_metadata), transform="val"
            )
            test_dataset = DocumentDataset(
                metadata_file=str(test_metadata), transform="test"
            )
        else:
            logger.error(
                "No dataset found. Please provide --data-dir with document images or create a dataset first."
            )
            logger.info("Example: python train.py --data-dir /path/to/documents")
            logger.info(
                "Or create a dataset first: python -m src.data.dataset --input-dir /path/to/documents"
            )
            import sys

            sys.exit(1)

    # Create data loaders
    use_curriculum = config.get("training.use_curriculum_learning", False)
    use_balanced_sampling = config.get("training.use_balanced_sampling", True)

    if use_curriculum and hasattr(train_dataset, "metadata"):
        # Use curriculum learning
        train_loader = create_curriculum_dataloader(
            dataset=train_dataset,
            metadata=train_dataset.metadata,
            batch_size=config.get("training.batch_size", 32),
            num_epochs=config.get("training.num_epochs", 50),
            current_epoch=0,
        )
        logger.info("Using curriculum learning for training")
    elif use_balanced_sampling and hasattr(train_dataset, "metadata"):
        # Use balanced batch sampling based on quality classes
        from src.models.model import BalancedBatchSampler

        sampler = BalancedBatchSampler(
            dataset=train_dataset,
            batch_size=config.get("training.batch_size", 32),
            num_classes=config.get("model.num_quality_classes", 5),
        )

        train_loader = DataLoader(
            train_dataset,
            batch_sampler=sampler,  # Use batch_sampler instead of sampler
            num_workers=config.get("dataset.num_workers", 4),
            pin_memory=True,
        )
        logger.info("Using balanced batch sampling for training")
    else:
        # Standard dataloader
        train_loader = DataLoader(
            train_dataset,
            batch_size=config.get("training.batch_size", 32),
            shuffle=True,
            num_workers=config.get("dataset.num_workers", 4),
            pin_memory=True,
        )

    val_loader = DataLoader(
        val_dataset,
        batch_size=config.get("training.batch_size", 32),
        shuffle=False,
        num_workers=config.get("dataset.num_workers", 4),
        pin_memory=True,
    )

    test_loader = DataLoader(
        test_dataset,
        batch_size=config.get("training.batch_size", 32),
        shuffle=False,
        num_workers=config.get("dataset.num_workers", 4),
        pin_memory=True,
    )

    return model_config, device, train_loader, val_loader, test_loader, config


def train_model(args):
    """Main training function"""
    # Setup training
    model_config, device, train_loader, val_loader, test_loader, config = (
        setup_training(args)
    )

    # Setup MLflow
    mlflow.set_tracking_uri(config.get("mlflow.tracking_uri", "mlruns"))
    mlflow.set_experiment(
        config.get("mlflow.experiment_name", "document-quality-assessment")
    )

    # Create model
    model = DocumentQualityModel(model_config)
    model = model.to(device)

    # Progressive unfreezing setup
    use_progressive_unfreezing = config.get("training.use_progressive_unfreezing", True)
    unfreeze_schedule = config.get("training.unfreeze_schedule", [5, 10, 15, 20])

    if use_progressive_unfreezing:
        # Initially freeze all backbone layers
        logger.info(
            "Using progressive unfreezing - initially freezing all backbone layers"
        )
        for name, param in model.named_parameters():
            if "backbone" in name:
                param.requires_grad = False
    else:
        # Standard freezing as specified
        if model_config.freeze_backbone_layers > 0:
            logger.info(
                f"Freezing first {model_config.freeze_backbone_layers} backbone layers"
            )
            frozen_count = 0
            for name, param in model.named_parameters():
                if (
                    "backbone" in name
                    and frozen_count < model_config.freeze_backbone_layers
                ):
                    param.requires_grad = False
                    frozen_count += 1

    # Resume from checkpoint if specified
    start_epoch = 0
    best_val_loss = float("inf")

    if args.resume:
        logger.info(f"Resuming from checkpoint: {args.resume}")
        checkpoint = torch.load(args.resume, map_location=device)
        model.load_state_dict(checkpoint["model_state_dict"])
        start_epoch = checkpoint.get("epoch", 0)
        best_val_loss = checkpoint.get("best_val_loss", float("inf"))

    # Setup optimizer with different learning rates for backbone and heads
    backbone_lr = config.get("training.learning_rate", 0.001) * config.get(
        "training.backbone_lr_factor", 0.1
    )
    head_lr = config.get("training.learning_rate", 0.001)

    # Separate parameters
    backbone_params = []
    head_params = []

    for name, param in model.named_parameters():
        if "backbone" in name:
            backbone_params.append(param)
        else:
            head_params.append(param)

    optimizer = torch.optim.AdamW(
        [
            {"params": backbone_params, "lr": backbone_lr},
            {"params": head_params, "lr": head_lr},
        ],
        weight_decay=config.get("training.weight_decay", 0.01),
    )

    # Setup scheduler
    scheduler_type = config.get("training.scheduler_type", "cosine")
    if scheduler_type == "cosine_warm_restarts":
        # Implement warm restart scheduler directly
        scheduler = torch.optim.lr_scheduler.CosineAnnealingWarmRestarts(
            optimizer,
            T_0=config.get("training.warm_restart_t0", 10),
            T_mult=config.get("training.warm_restart_tmult", 2),
            eta_min=config.get("training.min_lr", 1e-6),
        )
    elif scheduler_type == "cosine":
        scheduler = torch.optim.lr_scheduler.CosineAnnealingLR(
            optimizer, T_max=args.epochs - start_epoch
        )
    else:
        scheduler = torch.optim.lr_scheduler.ReduceLROnPlateau(
            optimizer,
            mode="min",
            patience=config.get("training.patience", 5),
            factor=0.5,
        )

    # Create persistent criterion for dynamic weighting
    from src.models.model import MultiTaskLoss

    criterion = MultiTaskLoss(
        regression_weight=config.get("training.regression_weight", 1.0),
        classification_weight=config.get("training.classification_weight", 0.5),
        issue_weight=config.get("training.issue_weight", 0.5),
        consistency_weight=config.get("training.consistency_weight", 0.3),
        use_focal_loss=config.get("training.use_focal_loss", True),
        use_uncertainty_weighting=config.get(
            "training.use_uncertainty_weighting", False
        ),
        use_dynamic_weighting=config.get("training.use_dynamic_task_weighting", False),
        use_ordinal_regression=config.get("training.use_ordinal_regression", True),
        num_quality_classes=model_config.num_quality_classes,
    )

    # Training loop
    logger.info("Starting training...")
    patience_counter = 0
    task_weight_update_freq = config.get("training.task_weight_update_freq", 5)

    # Start MLflow run
    with mlflow.start_run(run_name=config.get("mlflow.run_name", None)):
        # Log hyperparameters
        mlflow.log_params(
            {
                "backbone": model_config.backbone,
                "hidden_dim": model_config.hidden_dim,
                "dropout_rate": model_config.dropout_rate,
                "batch_size": config.get("training.batch_size", 32),
                "learning_rate": config.get("training.learning_rate", 0.001),
                "weight_decay": config.get("training.weight_decay", 0.01),
                "scheduler_type": scheduler_type,
                "regression_weight": config.get("training.regression_weight", 1.0),
                "classification_weight": config.get(
                    "training.classification_weight", 0.5
                ),
                "issue_weight": config.get("training.issue_weight", 0.5),
                "use_focal_loss": config.get("training.use_focal_loss", True),
                "num_quality_classes": model_config.num_quality_classes,
                "num_issue_classes": model_config.num_issue_classes,
                "use_attention": model_config.use_attention,
            }
        )

        # Log dataset info
        mlflow.log_params(
            {
                "train_size": len(train_loader.dataset),
                "val_size": len(val_loader.dataset),
                "test_size": len(test_loader.dataset),
            }
        )

        for epoch in range(start_epoch, args.epochs):
            logger.info(f"\nEpoch {epoch + 1}/{args.epochs}")

            # Progressive unfreezing
            if use_progressive_unfreezing and epoch in unfreeze_schedule:
                # Unfreeze next layer group
                layer_group = unfreeze_schedule.index(epoch)
                logger.info(f"Unfreezing backbone layer group {layer_group}")

                # Get all backbone layers
                backbone_layers = []
                for name, param in model.named_parameters():
                    if "backbone" in name:
                        backbone_layers.append((name, param))

                # Unfreeze proportionally
                layers_per_group = len(backbone_layers) // len(unfreeze_schedule)
                start_idx = layer_group * layers_per_group
                end_idx = (
                    (layer_group + 1) * layers_per_group
                    if layer_group < len(unfreeze_schedule) - 1
                    else len(backbone_layers)
                )

                for i in range(start_idx, end_idx):
                    backbone_layers[i][1].requires_grad = True
                    logger.info(f"Unfroze layer: {backbone_layers[i][0]}")

            # Update dynamic weights if enabled
            if (
                config.get("training.use_dynamic_task_weighting", False)
                and epoch > 0
                and epoch % task_weight_update_freq == 0
            ):
                criterion.update_dynamic_weights()

            # Train for one epoch
            train_metrics = train_epoch_with_metrics(
                model=model,
                train_loader=train_loader,
                optimizer=optimizer,
                device=device,
                regression_weight=config.get("training.regression_weight", 1.0),
                classification_weight=config.get("training.classification_weight", 0.5),
                issue_weight=config.get("training.issue_weight", 0.5),
                use_focal_loss=config.get("training.use_focal_loss", True),
                use_mixup=config.get("training.use_mixup", False),
                mixup_alpha=config.get("training.mixup_alpha", 0.2),
                current_epoch=epoch,
                criterion=criterion,
                gradient_clip_norm=config.get("training.gradient_clip_norm", 1.0),
            )
            train_loss = train_metrics["total_loss"]

            # Validate
            val_metrics = validate_with_metrics(model, val_loader, device, config)
            val_loss = val_metrics["total_loss"]

            # Update scheduler
            if scheduler_type in ["cosine", "cosine_warm_restarts"]:
                scheduler.step()
            else:
                scheduler.step(val_loss)

            # Log progress
            logger.info(f"Train Loss: {train_loss:.4f}, Val Loss: {val_loss:.4f}")
            logger.info(f"Learning Rate: {optimizer.param_groups[0]['lr']:.6f}")

            # Log metrics to MLflow
            mlflow.log_metrics(
                {
                    "train_loss": train_loss,
                    "train_regression_loss": train_metrics.get("regression_loss", 0),
                    "train_classification_loss": train_metrics.get(
                        "classification_loss", 0
                    ),
                    "train_issue_loss": train_metrics.get("issue_loss", 0),
                    "val_loss": val_loss,
                    "val_regression_loss": val_metrics.get("regression_loss", 0),
                    "val_classification_loss": val_metrics.get(
                        "classification_loss", 0
                    ),
                    "val_issue_loss": val_metrics.get("issue_loss", 0),
                    "val_mae": val_metrics.get("mae", 0),
                    "val_rmse": val_metrics.get("rmse", 0),
                    "val_accuracy": val_metrics.get("accuracy", 0),
                    "learning_rate": optimizer.param_groups[0]["lr"],
                },
                step=epoch,
            )

            # Save checkpoint
            if val_loss < best_val_loss:
                best_val_loss = val_loss
                patience_counter = 0

                checkpoint = {
                    "epoch": epoch + 1,
                    "model_state_dict": model.state_dict(),
                    "optimizer_state_dict": optimizer.state_dict(),
                    "scheduler_state_dict": scheduler.state_dict(),
                    "best_val_loss": best_val_loss,
                    "model_config": model_config,
                }

                save_path = Path(args.output_dir) / "best_model.pth"
                torch.save(checkpoint, save_path)
                logger.info(f"Saved best model to {save_path}")

                # Also save the model in the format expected by inference
                torch.save(
                    model.state_dict(),
                    Path(args.output_dir) / "document_quality_model.pth",
                )

                # Log model to MLflow with signature
                # Create a sample input for signature
                sample_batch = next(iter(val_loader))
                sample_input = sample_batch["image"][:1]  # Take first image

                # Get model output for signature
                model.eval()
                with torch.no_grad():
                    sample_input_device = sample_input.to(device)
                    sample_output = model(sample_input_device)

                # Create signature - handle dict output properly
                from mlflow.models import infer_signature

                output_sample = {}
                for k, v in sample_output.items():
                    if isinstance(v, torch.Tensor):
                        # Handle different tensor shapes
                        if v.dim() == 0:  # scalar
                            output_sample[k] = float(v.cpu().item())
                        else:
                            output_sample[k] = v.cpu().numpy()
                    else:
                        output_sample[k] = v

                signature = infer_signature(sample_input.numpy(), output_sample)

                # Log model with signature only (no input_example to avoid dict error)
                try:
                    mlflow.pytorch.log_model(
                        pytorch_model=model,
                        name="model",
                        registered_model_name="document-quality-model",
                        signature=signature,
                        pip_requirements=[
                            "torch",
                            "torchvision",
                            "pillow",
                            "numpy",
                            "opencv-python",
                        ],
                        code_paths=["src/"],
                    )
                    logger.info("Model successfully registered to MLflow")
                except Exception as e:
                    logger.warning(f"Failed to register model: {e}")
                    # Log without registration
                    mlflow.pytorch.log_model(
                        pytorch_model=model,
                        name="model",
                        signature=signature,
                        pip_requirements=[
                            "torch",
                            "torchvision",
                            "pillow",
                            "numpy",
                            "opencv-python",
                        ],
                        code_paths=["src/"],
                    )

                # Set model back to train mode
                model.train()
            else:
                patience_counter += 1

            # Early stopping
            if patience_counter >= config.get("training.patience", 5):
                logger.info(f"Early stopping triggered after {epoch + 1} epochs")
                mlflow.log_metric("early_stop_epoch", epoch + 1)
                break

        # Final evaluation on test set
        logger.info("\nEvaluating on test set...")
        test_metrics = evaluate(model, test_loader, device)
        logger.info(f"Test metrics: {test_metrics}")

        # Log test metrics to MLflow
        mlflow.log_metrics(
            {
                "test_mae": test_metrics["mae"],
                "test_rmse": test_metrics["rmse"],
                "test_r2": test_metrics["r2"],
                "test_mean_pred": test_metrics["mean_pred"],
                "test_std_pred": test_metrics["std_pred"],
            }
        )

        # Log final model artifacts
        mlflow.log_artifact(str(Path(args.output_dir) / "best_model.pth"))
        mlflow.log_artifact(str(Path(args.output_dir) / "document_quality_model.pth"))

        logger.info("Training completed!")


def train_epoch_with_metrics(
    model,
    train_loader,
    optimizer,
    device,
    regression_weight=1.0,
    classification_weight=0.5,
    issue_weight=0.5,
    use_focal_loss=True,
    use_mixup=False,
    mixup_alpha=0.2,
    current_epoch=0,
    criterion=None,
    gradient_clip_norm=1.0,
):
    """Train for one epoch and return detailed metrics"""
    from src.models.model import MultiTaskLoss

    model.train()

    # Use provided criterion or create new one
    if criterion is None:
        base_criterion = MultiTaskLoss(
            regression_weight=regression_weight,
            classification_weight=classification_weight,
            issue_weight=issue_weight,
            use_focal_loss=use_focal_loss,
        )
    else:
        base_criterion = criterion

    # Setup mixup if enabled
    if use_mixup:
        mixup_augmentation = MixupAugmentation(alpha=mixup_alpha)
        # Keep using base_criterion as we'll handle mixup targets separately
        criterion = base_criterion
    else:
        criterion = base_criterion

    # Update curriculum sampler epoch if applicable
    if hasattr(train_loader.sampler, "set_epoch"):
        train_loader.sampler.set_epoch(current_epoch)

    total_loss = 0
    total_regression_loss = 0
    total_classification_loss = 0
    total_issue_loss = 0
    num_batches = 0

    for batch in tqdm(train_loader, desc="Training"):
        images = batch["image"].to(device)
        targets = {
            "quality_score": batch["quality_score"].to(device),
            "quality_class": batch["quality_class"].to(device),
            "issues": batch["issues"].to(device),
        }

        # Apply mixup if enabled
        if use_mixup:
            images, targets, lam = mixup_augmentation(images, targets)  # type: ignore

        optimizer.zero_grad()
        outputs = model(images)

        loss = criterion(outputs, targets)

        # Handle both dict and tensor loss
        if isinstance(loss, dict):
            loss_value = loss["total"]
            total_loss += loss["total"].item()
            total_regression_loss += loss.get("regression", 0).item()
            total_classification_loss += loss.get("classification", 0).item()
            total_issue_loss += loss.get("issues", 0).item()
        else:
            loss_value = loss
            total_loss += loss.item()

        loss_value.backward()

        # Gradient clipping
        if gradient_clip_norm > 0:
            torch.nn.utils.clip_grad_norm_(
                model.parameters(), max_norm=gradient_clip_norm
            )

        optimizer.step()
        num_batches += 1

        # Record losses for dynamic weighting if using MultiTaskLoss
        if hasattr(criterion, "record_losses") and isinstance(loss, dict):
            criterion.record_losses(loss)

    return {
        "total_loss": total_loss / num_batches,
        "regression_loss": total_regression_loss / num_batches,
        "classification_loss": total_classification_loss / num_batches,
        "issue_loss": total_issue_loss / num_batches,
    }


def validate_with_metrics(model, val_loader, device, config):
    """Validate the model and return detailed metrics"""
    model.eval()
    total_loss = 0
    total_regression_loss = 0
    total_classification_loss = 0
    total_issue_loss = 0
    num_batches = 0

    all_predictions = []
    all_targets = []
    all_class_preds = []
    all_class_targets = []

    # Setup loss function
    from src.models.model import MultiTaskLoss

    criterion = MultiTaskLoss(
        regression_weight=config.get("training.regression_weight", 1.0),
        classification_weight=config.get("training.classification_weight", 0.5),
        issue_weight=config.get("training.issue_weight", 0.5),
        use_focal_loss=config.get("training.use_focal_loss", True),
    )

    with torch.no_grad():
        for batch in tqdm(val_loader, desc="Validating"):
            images = batch["image"].to(device)

            # Forward pass
            outputs = model(images)

            # Calculate loss
            targets = {
                "quality_score": batch["quality_score"].to(device),
                "quality_class": batch["quality_class"].to(device),
                "issues": batch["issues"].to(device),
            }

            loss = criterion(outputs, targets)
            # Handle both dict and tensor loss
            if isinstance(loss, dict):
                total_loss += loss["total"].item()
                total_regression_loss += loss.get("regression", 0).item()
                total_classification_loss += loss.get("classification", 0).item()
                total_issue_loss += loss.get("issues", 0).item()
            else:
                total_loss += loss.item()
            num_batches += 1

            # Collect predictions for metrics
            all_predictions.extend(outputs["quality_score"].cpu().numpy())
            all_targets.extend(batch["quality_score"].numpy())

            # Classification predictions
            class_preds = outputs["quality_class_logits"].argmax(dim=1).cpu().numpy()
            all_class_preds.extend(class_preds)
            all_class_targets.extend(batch["quality_class"].cpu().numpy())

    # Calculate regression metrics
    predictions = np.array(all_predictions)
    targets = np.array(all_targets)

    return {
        "total_loss": total_loss / num_batches,
        "regression_loss": total_regression_loss / num_batches,
        "classification_loss": total_classification_loss / num_batches,
        "issue_loss": total_issue_loss / num_batches,
        "mae": mean_absolute_error(targets, predictions),
        "rmse": np.sqrt(mean_squared_error(targets, predictions)),
        "accuracy": accuracy_score(all_class_targets, all_class_preds),
    }


def validate(model, val_loader, device, config):
    """Validate the model (legacy function for compatibility)"""
    metrics = validate_with_metrics(model, val_loader, device, config)
    return metrics["total_loss"]


def evaluate(model, test_loader, device):
    """Evaluate model on test set"""
    model.eval()

    all_predictions = []
    all_targets = []
    all_issues_pred = []
    all_issues_true = []

    with torch.no_grad():
        for batch in tqdm(test_loader, desc="Evaluating"):
            images = batch["image"].to(device)

            # Forward pass
            outputs = model(images)

            # Collect predictions
            all_predictions.extend(outputs["quality_score"].cpu().numpy())
            all_targets.extend(batch["quality_score"].numpy())

            # Issue predictions
            issue_probs = torch.sigmoid(outputs["issues"])
            issue_preds = (issue_probs > 0.5).cpu().numpy()
            all_issues_pred.extend(issue_preds)
            all_issues_true.extend(batch["issues"].numpy())

    # Calculate metrics
    import numpy as np
    from sklearn.metrics import mean_absolute_error, mean_squared_error, r2_score

    predictions = np.array(all_predictions)
    targets = np.array(all_targets)

    metrics = {
        "mae": mean_absolute_error(targets, predictions),
        "rmse": np.sqrt(mean_squared_error(targets, predictions)),
        "r2": r2_score(targets, predictions),
        "mean_pred": predictions.mean(),
        "std_pred": predictions.std(),
    }

    return metrics


def main():
    """Main function"""
    parser = argparse.ArgumentParser(
        description="Train Document Quality Assessment Model"
    )
    parser.add_argument(
        "--data-dir", type=str, help="Directory containing document images"
    )
    parser.add_argument("--config", type=str, help="Path to configuration file")
    parser.add_argument(
        "--epochs", type=int, default=20, help="Number of training epochs"
    )
    parser.add_argument(
        "--output-dir", type=str, default="models", help="Output directory for models"
    )
    parser.add_argument("--resume", type=str, help="Resume from checkpoint")
    parser.add_argument(
        "--device", type=str, choices=["cuda", "cpu"], help="Device to use"
    )

    args = parser.parse_args()

    # Create output directory
    Path(args.output_dir).mkdir(parents=True, exist_ok=True)

    # Start training
    train_model(args)


if __name__ == "__main__":
    main()
