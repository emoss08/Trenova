#!/usr/bin/env python3
#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md

"""
Production Training Script for Document Quality Assessment

This is the unified, production-ready training script with all best practices:
- Automatic class balancing
- MLflow experiment tracking
- Model checkpointing with best model selection
- Early stopping
- Learning rate scheduling
- Gradient clipping
- Comprehensive logging

Usage:
    # Train with default config
    python scripts/train_production.py

    # Train with custom config
    python scripts/train_production.py --config config/my_config.yaml

    # Resume from checkpoint
    python scripts/train_production.py --resume checkpoints/checkpoint_epoch_10.pth

    # Train with custom experiment name
    python scripts/train_production.py --experiment my_experiment --run-name trial_01
"""

import argparse
import logging
import sys
from datetime import datetime
from pathlib import Path

import mlflow
import mlflow.pytorch
import torch
import yaml
from torch.utils.data import DataLoader

# Add project root to path
project_root = Path(__file__).parent.parent
sys.path.insert(0, str(project_root))

from src.data.dataset import DocumentDataset, create_dataset_from_folder
from src.models.model import BalancedBatchSampler, DocumentQualityModel, ModelConfig
from src.training.trainer import (
    Trainer,
    calculate_class_weights,
    create_weighted_criterion,
)
from src.utils.config import get_config

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


def setup_mlflow(config: dict, args):
    """Setup MLflow experiment tracking"""
    # Set tracking URI
    mlflow_uri = config.get("mlflow", {}).get("tracking_uri", "mlruns")
    mlflow.set_tracking_uri(mlflow_uri)

    # Set experiment
    experiment_name = (
        args.experiment
        if args.experiment
        else config.get("mlflow", {}).get("experiment_name", "document-quality-training")
    )
    mlflow.set_experiment(experiment_name)

    # Start run
    run_name = (
        args.run_name
        if args.run_name
        else config.get("mlflow", {}).get("run_name")
        or f"run_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
    )

    mlflow.start_run(run_name=run_name)

    # Log parameters
    mlflow.log_params(
        {
            "model.backbone": config.get("model", {}).get("backbone"),
            "model.hidden_dim": config.get("model", {}).get("hidden_dim"),
            "model.dropout_rate": config.get("model", {}).get("dropout_rate"),
            "training.batch_size": config.get("training", {}).get("batch_size"),
            "training.num_epochs": config.get("training", {}).get("num_epochs"),
            "training.learning_rate": config.get("training", {}).get("learning_rate"),
            "training.weight_decay": config.get("training", {}).get("weight_decay"),
            "training.use_balanced_sampling": config.get("training", {}).get(
                "use_balanced_sampling"
            ),
        }
    )

    # Log config file
    config_path = args.config if args.config else "config/default.yaml"
    mlflow.log_artifact(config_path)

    logger.info(f"✓ MLflow tracking initialized")
    logger.info(f"  Experiment: {experiment_name}")
    logger.info(f"  Run: {run_name}")


def load_datasets(config: dict, args) -> tuple:
    """Load or create datasets"""
    logger.info("Loading datasets...")

    # Check if data directory provided
    if args.data_dir:
        logger.info(f"Creating dataset from: {args.data_dir}")
        train_dataset, val_dataset, test_dataset = create_dataset_from_folder(
            folder_path=Path(args.data_dir),
            train_ratio=config.get("dataset", {}).get("train_ratio", 0.7),
            val_ratio=config.get("dataset", {}).get("val_ratio", 0.2),
            test_ratio=config.get("dataset", {}).get("test_ratio", 0.1),
        )
    else:
        # Load existing datasets
        datasets_dir = Path(config.get("paths", {}).get("datasets_dir", "datasets"))

        # Try to find metadata files
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

        # Find existing metadata
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

        if not all([train_metadata, val_metadata, test_metadata]):
            logger.error("Dataset not found. Please provide --data-dir or create dataset first.")
            logger.info("\nTo create a dataset:")
            logger.info("  python -m src.data.dataset --input-dir documents --output-dir datasets")
            sys.exit(1)

        # Load datasets
        train_dataset = DocumentDataset(
            metadata_file=str(train_metadata),
            transform="train",
            use_advanced_augmentations=config.get("dataset", {}).get(
                "augmentation", {}
            ).get("use_domain_augmentations", True),
        )

        val_dataset = DocumentDataset(
            metadata_file=str(val_metadata),
            transform="val",
            use_advanced_augmentations=False,
        )

        test_dataset = DocumentDataset(
            metadata_file=str(test_metadata),
            transform="test",
            use_advanced_augmentations=False,
        )

    logger.info(f"✓ Datasets loaded")
    logger.info(f"  Train: {len(train_dataset)} images")
    logger.info(f"  Val:   {len(val_dataset)} images")
    logger.info(f"  Test:  {len(test_dataset)} images")

    return train_dataset, val_dataset, test_dataset


def create_dataloaders(
    train_dataset, val_dataset, config: dict
) -> tuple:
    """Create dataloaders with optional balanced sampling"""
    batch_size = config.get("training", {}).get("batch_size", 32)
    num_workers = config.get("dataset", {}).get("num_workers", 4)
    use_balanced_sampling = config.get("training", {}).get("use_balanced_sampling", True)

    if use_balanced_sampling and hasattr(train_dataset, "metadata"):
        logger.info("Using balanced batch sampling to address class imbalance")

        sampler = BalancedBatchSampler(
            dataset=train_dataset,
            batch_size=batch_size,
            num_classes=config.get("model", {}).get("num_quality_classes", 5),
        )

        train_loader = DataLoader(
            train_dataset,
            batch_sampler=sampler,
            num_workers=num_workers,
            pin_memory=True,
        )
    else:
        train_loader = DataLoader(
            train_dataset,
            batch_size=batch_size,
            shuffle=True,
            num_workers=num_workers,
            pin_memory=True,
        )

    val_loader = DataLoader(
        val_dataset,
        batch_size=batch_size,
        shuffle=False,
        num_workers=num_workers,
        pin_memory=True,
    )

    logger.info(f"✓ Dataloaders created")
    logger.info(f"  Batch size: {batch_size}")
    logger.info(f"  Train batches: {len(train_loader)}")
    logger.info(f"  Val batches: {len(val_loader)}")

    return train_loader, val_loader


def create_model(config: dict, device: torch.device) -> DocumentQualityModel:
    """Create model from config"""
    logger.info("Creating model...")

    model_config = ModelConfig(
        backbone=config.get("model", {}).get("backbone", "efficientnet_b0"),
        num_quality_classes=config.get("model", {}).get("num_quality_classes", 5),
        num_issue_classes=config.get("model", {}).get("num_issue_classes", 10),
        hidden_dim=config.get("model", {}).get("hidden_dim", 256),
        dropout_rate=config.get("model", {}).get("dropout_rate", 0.5),
        use_attention=config.get("model", {}).get("use_attention", True),
        freeze_backbone_layers=config.get("model", {}).get("freeze_backbone_layers", 5),
    )

    model = DocumentQualityModel(model_config)
    model = model.to(device)

    # Count parameters
    total_params = sum(p.numel() for p in model.parameters())
    trainable_params = sum(p.numel() for p in model.parameters() if p.requires_grad)

    logger.info(f"✓ Model created")
    logger.info(f"  Backbone: {model_config.backbone}")
    logger.info(f"  Total parameters: {total_params:,}")
    logger.info(f"  Trainable parameters: {trainable_params:,}")

    return model


def create_optimizer_and_scheduler(
    model: nn.Module, config: dict
) -> tuple:
    """Create optimizer and learning rate scheduler"""
    learning_rate = config.get("training", {}).get("learning_rate", 0.001)
    weight_decay = config.get("training", {}).get("weight_decay", 0.01)
    backbone_lr_factor = config.get("training", {}).get("backbone_lr_factor", 0.1)

    # Different learning rates for backbone and heads
    backbone_params = []
    head_params = []

    for name, param in model.named_parameters():
        if "backbone" in name:
            backbone_params.append(param)
        else:
            head_params.append(param)

    optimizer = torch.optim.AdamW(
        [
            {"params": backbone_params, "lr": learning_rate * backbone_lr_factor},
            {"params": head_params, "lr": learning_rate},
        ],
        weight_decay=weight_decay,
    )

    # Create scheduler
    scheduler_type = config.get("training", {}).get("scheduler_type", "cosine")
    num_epochs = config.get("training", {}).get("num_epochs", 50)

    if scheduler_type == "cosine":
        scheduler = torch.optim.lr_scheduler.CosineAnnealingLR(
            optimizer, T_max=num_epochs, eta_min=1e-7
        )
    elif scheduler_type == "cosine_warm_restarts":
        scheduler = torch.optim.lr_scheduler.CosineAnnealingWarmRestarts(
            optimizer,
            T_0=config.get("training", {}).get("warm_restart_t0", 10),
            T_mult=config.get("training", {}).get("warm_restart_tmult", 2),
            eta_min=config.get("training", {}).get("min_lr", 1e-7),
        )
    elif scheduler_type == "plateau":
        scheduler = torch.optim.lr_scheduler.ReduceLROnPlateau(
            optimizer, mode="min", factor=0.5, patience=5, verbose=True
        )
    else:
        scheduler = None

    logger.info(f"✓ Optimizer and scheduler created")
    logger.info(f"  Learning rate: {learning_rate}")
    logger.info(f"  Backbone LR factor: {backbone_lr_factor}")
    logger.info(f"  Weight decay: {weight_decay}")
    logger.info(f"  Scheduler: {scheduler_type}")

    return optimizer, scheduler


def main():
    parser = argparse.ArgumentParser(
        description="Production Training Script for Document Quality Assessment",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Train with default config
  python scripts/train_production.py

  # Train with custom config
  python scripts/train_production.py --config config/my_config.yaml

  # Resume from checkpoint
  python scripts/train_production.py --resume output/checkpoints/checkpoint_epoch_10.pth

  # Train with custom experiment tracking
  python scripts/train_production.py --experiment my_exp --run-name trial_01

  # Create dataset and train
  python scripts/train_production.py --data-dir documents/
        """,
    )

    parser.add_argument(
        "--config",
        type=str,
        default=None,
        help="Path to config YAML file (default: config/best_training.yaml)",
    )
    parser.add_argument(
        "--data-dir",
        type=str,
        default=None,
        help="Directory containing source documents (will create dataset if needed)",
    )
    parser.add_argument(
        "--output-dir",
        type=str,
        default=None,
        help="Output directory for checkpoints and logs",
    )
    parser.add_argument(
        "--resume",
        type=str,
        default=None,
        help="Path to checkpoint to resume training from",
    )
    parser.add_argument(
        "--experiment",
        type=str,
        default=None,
        help="MLflow experiment name",
    )
    parser.add_argument(
        "--run-name",
        type=str,
        default=None,
        help="MLflow run name",
    )
    parser.add_argument(
        "--no-mlflow",
        action="store_true",
        help="Disable MLflow tracking",
    )
    parser.add_argument(
        "--device",
        type=str,
        default="auto",
        choices=["auto", "cuda", "cpu"],
        help="Device to use (default: auto)",
    )
    parser.add_argument(
        "--epochs",
        type=int,
        default=None,
        help="Override number of epochs from config",
    )

    args = parser.parse_args()

    # Print header
    logger.info("=" * 80)
    logger.info("DOCUMENT QUALITY ASSESSMENT - PRODUCTION TRAINING")
    logger.info("=" * 80)
    logger.info("")

    # Load config
    if args.config:
        config_path = Path(args.config)
    else:
        # Use best_training.yaml by default
        config_path = Path("config/best_training.yaml")
        if not config_path.exists():
            config_path = Path("config/default.yaml")

    if not config_path.exists():
        logger.error(f"Config file not found: {config_path}")
        sys.exit(1)

    logger.info(f"Loading config from: {config_path}")
    config = get_config(str(config_path))

    # Override epochs if specified
    if args.epochs:
        if "training" not in config:
            config["training"] = {}
        config["training"]["num_epochs"] = args.epochs

    # Setup device
    if args.device == "auto":
        device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    else:
        device = torch.device(args.device)

    logger.info(f"Using device: {device}")
    logger.info("")

    # Setup MLflow
    if not args.no_mlflow:
        setup_mlflow(config, args)
        logger.info("")

    # Load datasets
    train_dataset, val_dataset, test_dataset = load_datasets(config, args)
    logger.info("")

    # Create dataloaders
    train_loader, val_loader = create_dataloaders(train_dataset, val_dataset, config)
    logger.info("")

    # Create model
    model = create_model(config, device)
    logger.info("")

    # Calculate class weights for handling imbalance
    logger.info("Calculating class weights...")
    class_weights = calculate_class_weights(
        train_dataset, num_classes=config.get("model", {}).get("num_quality_classes", 5)
    )
    logger.info("")

    # Create criterion with class weights
    criterion = create_weighted_criterion(config, class_weights, device)

    # Create optimizer and scheduler
    optimizer, scheduler = create_optimizer_and_scheduler(model, config)
    logger.info("")

    # Setup output directory
    if args.output_dir:
        output_dir = Path(args.output_dir)
    else:
        output_dir = Path(config.get("paths", {}).get("models_dir", "models")) / datetime.now().strftime(
            "%Y%m%d_%H%M%S"
        )

    output_dir.mkdir(parents=True, exist_ok=True)
    logger.info(f"Output directory: {output_dir}")
    logger.info("")

    # Create trainer
    trainer = Trainer(
        model=model,
        train_loader=train_loader,
        val_loader=val_loader,
        optimizer=optimizer,
        criterion=criterion,
        device=device,
        config=config,
        output_dir=output_dir,
        scheduler=scheduler,
        use_mlflow=not args.no_mlflow,
    )

    # Resume from checkpoint if specified
    if args.resume:
        trainer.load_checkpoint(Path(args.resume))
        logger.info("")

    # Train
    num_epochs = config.get("training", {}).get("num_epochs", 50)
    patience = config.get("training", {}).get("patience", 15)

    try:
        history = trainer.train(num_epochs=num_epochs, patience=patience, save_frequency=5)

        # Log final model to MLflow
        if not args.no_mlflow:
            mlflow.pytorch.log_model(model, "model")
            mlflow.log_artifact(str(output_dir / "best_model.pth"))

        logger.info("✓ Training completed successfully!")
        logger.info(f"  Best model saved to: {output_dir / 'best_model.pth'}")

    except KeyboardInterrupt:
        logger.warning("\n⚠ Training interrupted by user")
        # Save checkpoint
        trainer.save_checkpoint(is_best=False, filename="interrupted.pth")
        logger.info(f"  Checkpoint saved to: {output_dir / 'checkpoints' / 'interrupted.pth'}")

    except Exception as e:
        logger.error(f"❌ Training failed with error: {str(e)}")
        raise

    finally:
        if not args.no_mlflow:
            mlflow.end_run()


if __name__ == "__main__":
    main()
