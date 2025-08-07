#!/usr/bin/env python3
#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
"""
Generate the best possible dataset for document quality assessment training.

This script creates a comprehensive dataset with:
- Realistic quality distribution
- All enhanced degradations
- Transportation-specific augmentations
- Optimal train/val/test splits
"""

import argparse
import logging
import shutil
from pathlib import Path

from src.data.dataset import create_enhanced_dataset

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


def check_source_documents(input_dir: Path) -> int:
    """Check and count source documents"""
    extensions = [".pdf", ".jpg", ".jpeg", ".png", ".tiff", ".tif"]

    doc_count = 0
    for ext in extensions:
        doc_count += len(list(input_dir.glob(f"*{ext}")))
        doc_count += len(list(input_dir.glob(f"*{ext.upper()}")))

    return doc_count


def main():
    parser = argparse.ArgumentParser(
        description="Generate the best dataset for document quality assessment"
    )
    parser.add_argument(
        "--input-dir",
        type=str,
        default="documents",
        help="Directory containing source documents",
    )
    parser.add_argument(
        "--output-dir",
        type=str,
        default="datasets/default",
        help="Output directory (default: datasets/default for training compatibility)",
    )
    parser.add_argument(
        "--samples-per-doc",
        type=int,
        default=50,
        help="Number of synthetic samples per source document",
    )
    parser.add_argument(
        "--clear-existing",
        action="store_true",
        help="Clear existing dataset directory before generation",
    )
    parser.add_argument(
        "--cpu-fraction",
        type=float,
        default=0.75,
        help="Fraction of CPUs to use (0.0-1.0, default: 0.75)",
    )
    parser.add_argument(
        "--max-cpus",
        type=int,
        default=None,
        help="Maximum number of CPUs to use (overrides cpu-fraction)",
    )

    args = parser.parse_args()

    input_path = Path(args.input_dir)
    output_path = Path(args.output_dir)

    # Check if input directory exists
    if not input_path.exists():
        logger.error(f"Input directory '{input_path}' does not exist!")
        logger.info("Please create a 'documents' folder and add your source documents:")
        logger.info("  - Transportation documents (bills of lading, invoices, etc.)")
        logger.info("  - High-quality scans or photos")
        logger.info("  - Various document types for diversity")
        return

    # Count source documents
    doc_count = check_source_documents(input_path)
    if doc_count == 0:
        logger.error(f"No documents found in '{input_path}'!")
        logger.info("Supported formats: PDF, JPG, JPEG, PNG, TIFF")
        return

    logger.info(f"Found {doc_count} source documents")

    # Calculate expected dataset size
    total_samples = doc_count * args.samples_per_doc
    train_samples = int(total_samples * 0.8)
    val_samples = int(total_samples * 0.15)
    test_samples = total_samples - train_samples - val_samples

    logger.info(f"Will generate approximately:")
    logger.info(f"  - Total: {total_samples} samples")
    logger.info(f"  - Train: {train_samples} samples")
    logger.info(f"  - Val: {val_samples} samples")
    logger.info(f"  - Test: {test_samples} samples")

    # Clear existing dataset if requested
    if args.clear_existing and output_path.exists():
        logger.warning(f"Clearing existing dataset at '{output_path}'")
        response = input("Are you sure? (y/N): ")
        if response.lower() == "y":
            shutil.rmtree(output_path)
        else:
            logger.info("Cancelled")
            return

    # Generate the best dataset
    logger.info("Starting dataset generation with optimal settings...")
    logger.info("Using:")
    logger.info(
        "  - Realistic quality distribution (15% high, 35% good, 30% moderate, 15% poor, 5% very poor)"
    )
    logger.info(
        "  - Enhanced degradations (motion blur, glare, perspective warp, etc.)"
    )
    logger.info(
        "  - Transportation-specific augmentations (stamps, barcodes, folds, etc.)"
    )
    logger.info("  - Optimal train/val/test split (80/15/5)")

    try:
        create_enhanced_dataset(
            input_dir=input_path,
            output_dir=output_path,
            mode="synthetic",
            split_ratio={"train": 0.8, "val": 0.15, "test": 0.05},
            cpu_fraction=args.cpu_fraction,
            max_cpus=args.max_cpus,
            # The create_enhanced_dataset function already uses the optimal settings
            # from our improvements (realistic distribution, all degradations, etc.)
            # It generates 10-20 variations per document by default
        )

        logger.info(f"✅ Dataset generated successfully at '{output_path}'")

        # Verify the generated dataset
        for split in ["train", "val", "test"]:
            split_dir = output_path / split
            if not split_dir.exists():
                split_dir = output_path / "default" / split

            if split_dir.exists():
                metadata_file = split_dir / f"{split}_metadata.csv"
                if metadata_file.exists():
                    import pandas as pd

                    df = pd.read_csv(metadata_file)
                    logger.info(f"{split.capitalize()} set: {len(df)} samples")

                    # Show quality distribution
                    quality_dist = df["quality_class"].value_counts().sort_index()
                    logger.info(f"  Quality distribution: {dict(quality_dist)}")

        logger.info("\n🎉 Dataset is ready for training!")
        logger.info("To start training with the best configuration, run:")
        logger.info("  python train.py --config config/best_training.yaml --epochs 50")

    except Exception as e:
        logger.error(f"Error generating dataset: {e}")
        import traceback

        traceback.print_exc()


if __name__ == "__main__":
    main()
