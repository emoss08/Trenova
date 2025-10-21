#!/usr/bin/env python3
#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md

"""
Comprehensive model evaluation script

This script evaluates a trained model on the test set and generates:
- Performance metrics (regression, classification, calibration)
- Visualization plots (ROC curves, confusion matrices, etc.)
- Explainability visualizations (Grad-CAM)
- Comprehensive HTML/text reports

Usage:
    python scripts/evaluate_model.py --model models/best_model.pth --config config/best_training.yaml
    python scripts/evaluate_model.py --model models/best_model.pth --output-dir evaluation_results
"""

import argparse
import json
import logging
from pathlib import Path

import torch
import yaml
from torch.utils.data import DataLoader

from src.data.dataset import DocumentDataset
from src.evaluation.explainability import batch_explain_predictions, get_target_layer
from src.evaluation.metrics import evaluate_model_comprehensive
from src.evaluation.visualize import create_evaluation_report
from src.models.model import DocumentQualityModel, ModelConfig

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


def load_config(config_path: Path) -> dict:
    """Load configuration from YAML file"""
    with open(config_path, "r") as f:
        config = yaml.safe_load(f)
    return config


def load_model(checkpoint_path: Path, config: dict, device: torch.device) -> DocumentQualityModel:
    """Load model from checkpoint"""
    logger.info(f"Loading model from {checkpoint_path}")

    # Create model config
    model_config = ModelConfig(
        backbone=config.get("model", {}).get("backbone", "efficientnet_b0"),
        num_quality_classes=config.get("model", {}).get("num_quality_classes", 5),
        num_issue_classes=config.get("model", {}).get("num_issue_classes", 10),
        hidden_dim=config.get("model", {}).get("hidden_dim", 256),
        dropout_rate=config.get("model", {}).get("dropout_rate", 0.5),
        use_attention=config.get("model", {}).get("use_attention", True),
        freeze_backbone_layers=config.get("model", {}).get("freeze_backbone_layers", 5),
    )

    # Create model
    model = DocumentQualityModel(model_config)

    # Load checkpoint
    checkpoint = torch.load(checkpoint_path, map_location=device)

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

    model = model.to(device)
    model.eval()

    logger.info("Model loaded successfully")
    return model


def get_test_dataloader(config: dict) -> DataLoader:
    """Get test dataset and dataloader"""
    datasets_dir = Path(config.get("paths", {}).get("datasets_dir", "datasets"))

    # Try to find test metadata
    test_metadata_candidates = [
        datasets_dir / "default" / "test" / "test_metadata.csv",
        datasets_dir / "test" / "metadata.csv",
        datasets_dir / "test" / "test_metadata.csv",
    ]

    test_metadata = None
    for candidate in test_metadata_candidates:
        if candidate.exists():
            test_metadata = candidate
            break

    if test_metadata is None:
        raise FileNotFoundError(f"Could not find test metadata in {datasets_dir}")

    logger.info(f"Loading test dataset from {test_metadata}")

    # Create dataset
    test_dataset = DocumentDataset(
        metadata_file=str(test_metadata),
        transform="test",
        use_advanced_augmentations=False,  # No augmentation for test
    )

    # Create dataloader
    test_loader = DataLoader(
        test_dataset,
        batch_size=config.get("training", {}).get("batch_size", 32),
        shuffle=False,
        num_workers=config.get("dataset", {}).get("num_workers", 4),
        pin_memory=True,
    )

    logger.info(f"Test dataset loaded: {len(test_dataset)} images")
    return test_loader, test_dataset


def main():
    parser = argparse.ArgumentParser(
        description="Comprehensive Model Evaluation",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Evaluate with custom config
  python scripts/evaluate_model.py --model models/best_model.pth --config config/best_training.yaml

  # Evaluate with custom output directory
  python scripts/evaluate_model.py --model models/best_model.pth --output-dir my_evaluation

  # Generate explanations for sample images
  python scripts/evaluate_model.py --model models/best_model.pth --explain --num-samples 10
        """,
    )

    parser.add_argument(
        "--model",
        type=str,
        required=True,
        help="Path to model checkpoint (e.g., models/best_model.pth)",
    )
    parser.add_argument(
        "--config",
        type=str,
        default="config/best_training.yaml",
        help="Path to config file (default: config/best_training.yaml)",
    )
    parser.add_argument(
        "--output-dir",
        type=str,
        default="evaluation_results",
        help="Output directory for evaluation results (default: evaluation_results)",
    )
    parser.add_argument(
        "--threshold",
        type=float,
        default=0.5,
        help="Quality score threshold for accept/reject (default: 0.5)",
    )
    parser.add_argument(
        "--explain",
        action="store_true",
        help="Generate explainability visualizations (Grad-CAM)",
    )
    parser.add_argument(
        "--num-samples",
        type=int,
        default=20,
        help="Number of samples to explain (default: 20)",
    )
    parser.add_argument(
        "--device",
        type=str,
        default="auto",
        choices=["auto", "cuda", "cpu"],
        help="Device to use (default: auto)",
    )

    args = parser.parse_args()

    # Setup device
    if args.device == "auto":
        device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    else:
        device = torch.device(args.device)

    logger.info(f"Using device: {device}")

    # Load config
    config_path = Path(args.config)
    if not config_path.exists():
        logger.error(f"Config file not found: {config_path}")
        return

    config = load_config(config_path)

    # Load model
    model_path = Path(args.model)
    if not model_path.exists():
        logger.error(f"Model file not found: {model_path}")
        return

    model = load_model(model_path, config, device)

    # Get test dataloader
    try:
        test_loader, test_dataset = get_test_dataloader(config)
    except FileNotFoundError as e:
        logger.error(str(e))
        return

    # Create output directory
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    logger.info(f"Output directory: {output_dir}")
    logger.info("=" * 80)

    # ===== STEP 1: Comprehensive Evaluation =====
    logger.info("Step 1/3: Running comprehensive evaluation...")
    logger.info("-" * 80)

    evaluation_results = evaluate_model_comprehensive(
        model=model,
        dataloader=test_loader,
        device=device,
        acceptance_threshold=args.threshold,
    )

    # Save evaluation results as JSON
    results_json_path = output_dir / "evaluation_results.json"
    with open(results_json_path, "w") as f:
        # Create a serializable copy (remove raw predictions for file size)
        serializable_results = {
            k: v for k, v in evaluation_results.items() if k != "raw_predictions"
        }
        json.dump(serializable_results, f, indent=2)

    logger.info(f"Evaluation results saved to {results_json_path}")
    logger.info("")

    # Print key metrics
    logger.info("KEY METRICS:")
    logger.info(f"  Quality Score MAE:    {evaluation_results['regression']['mae']:.4f}")
    logger.info(f"  Quality Score R²:     {evaluation_results['regression']['r2']:.4f}")
    logger.info(f"  Classification Acc:   {evaluation_results['classification']['accuracy']:.4f}")
    logger.info(f"  Balanced Accuracy:    {evaluation_results['classification']['balanced_accuracy']:.4f}")
    logger.info(f"  Binary F1 Score:      {evaluation_results['binary_classification']['f1']:.4f}")
    logger.info(f"  ROC AUC:              {evaluation_results['binary_classification']['roc_auc']:.4f}")
    logger.info(f"  Calibration ECE:      {evaluation_results['calibration']['ece']:.4f}")
    logger.info(f"  Issue Detection F1:   {evaluation_results['issue_detection']['macro_f1']:.4f}")
    logger.info("")

    # ===== STEP 2: Generate Visualizations =====
    logger.info("Step 2/3: Generating visualization reports...")
    logger.info("-" * 80)

    model_name = model_path.stem
    report_dir = create_evaluation_report(
        evaluation_results=evaluation_results,
        output_dir=output_dir,
        model_name=model_name,
    )

    logger.info(f"Evaluation report generated in {report_dir}")
    logger.info("")

    # ===== STEP 3: Generate Explainability Visualizations (Optional) =====
    if args.explain:
        logger.info("Step 3/3: Generating explainability visualizations...")
        logger.info("-" * 80)

        # Get sample images from test set
        sample_paths = []
        for i in range(min(args.num_samples, len(test_dataset))):
            sample_paths.append(Path(test_dataset.image_paths[i]))

        # Get target layer for Grad-CAM
        target_layer = get_target_layer(model, config.get("model", {}).get("backbone", "efficientnet_b0"))

        # Generate explanations
        explanations_dir = output_dir / "explanations"
        batch_explain_predictions(
            model=model,
            image_paths=sample_paths,
            transform=test_dataset.transform,
            output_dir=explanations_dir,
            target_layer=target_layer,
            max_images=args.num_samples,
        )

        logger.info(f"Explainability visualizations saved to {explanations_dir}")
        logger.info("")
    else:
        logger.info("Step 3/3: Skipped (use --explain to generate Grad-CAM visualizations)")
        logger.info("")

    # ===== SUMMARY =====
    logger.info("=" * 80)
    logger.info("EVALUATION COMPLETE!")
    logger.info("=" * 80)
    logger.info(f"All results saved to: {output_dir}")
    logger.info(f"  - Metrics JSON:     {results_json_path}")
    logger.info(f"  - Text Report:      {output_dir / f'{model_name}_evaluation_report.txt'}")
    logger.info(f"  - Visualizations:   {output_dir}/*.png")

    if args.explain:
        logger.info(f"  - Explanations:     {output_dir / 'explanations'}/*.png")

    logger.info("")
    logger.info("Next steps:")
    logger.info("  1. Review the evaluation report and visualizations")
    logger.info("  2. Check calibration (ECE should be < 0.1)")
    logger.info("  3. Analyze per-class and per-issue performance")
    logger.info("  4. If performance is inadequate, consider:")
    logger.info("     - Collecting more real-world training data")
    logger.info("     - Addressing class imbalance")
    logger.info("     - Hyperparameter tuning")
    logger.info("     - Model architecture changes")


if __name__ == "__main__":
    main()
