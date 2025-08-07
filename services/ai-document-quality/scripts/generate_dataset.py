#!/usr/bin/env python3
#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
"""
Generate enhanced document quality dataset with all improvements.

This script creates a high-quality dataset for training the document
quality assessment model with realistic degradations and augmentations.
"""

import argparse
import logging
from pathlib import Path

from src.data.dataset import create_enhanced_dataset

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def main():
    parser = argparse.ArgumentParser(
        description="Generate enhanced document quality dataset"
    )
    parser.add_argument(
        "--input-dir",
        type=str,
        required=True,
        help="Directory containing source documents (PDFs, images)",
    )
    parser.add_argument(
        "--output-dir",
        type=str,
        default="datasets/enhanced",
        help="Output directory for generated dataset",
    )
    parser.add_argument(
        "--num-samples",
        type=int,
        default=5000,
        help="Number of synthetic samples to generate per source document",
    )
    parser.add_argument(
        "--quality-distribution",
        type=str,
        default="realistic",
        choices=["realistic", "uniform", "challenging"],
        help="Quality distribution for synthetic data",
    )
    parser.add_argument(
        "--include-augmentations",
        action="store_true",
        help="Apply transportation-specific augmentations",
    )

    args = parser.parse_args()

    # Configure dataset creation based on distribution type
    if args.quality_distribution == "realistic":
        # Realistic distribution matching real-world documents
        quality_ranges = {
            "high": 0.15,  # 15% high quality
            "good": 0.35,  # 35% good quality
            "moderate": 0.30,  # 30% moderate quality
            "poor": 0.15,  # 15% poor quality
            "very_poor": 0.05,  # 5% very poor quality
        }
    elif args.quality_distribution == "uniform":
        # Uniform distribution across all quality levels
        quality_ranges = {
            "high": 0.20,
            "good": 0.20,
            "moderate": 0.20,
            "poor": 0.20,
            "very_poor": 0.20,
        }
    else:  # challenging
        # More challenging samples for robust training
        quality_ranges = {
            "high": 0.05,
            "good": 0.15,
            "moderate": 0.30,
            "poor": 0.35,
            "very_poor": 0.15,
        }

    logger.info(f"Generating dataset with {args.quality_distribution} distribution")
    logger.info(f"Quality ranges: {quality_ranges}")

    # Create enhanced dataset
    create_enhanced_dataset(
        input_dir=Path(args.input_dir),
        output_dir=Path(args.output_dir),
        mode="synthetic",
        max_docs=None,  # Process all documents
        split_ratio={"train": 0.8, "val": 0.15, "test": 0.05},
        num_synthetic_per_doc=args.num_samples // 100,  # Approximate samples per doc
    )

    logger.info(f"Dataset created successfully in {args.output_dir}")

    # Print dataset statistics
    output_path = Path(args.output_dir)
    for split in ["train", "val", "test"]:
        split_dir = output_path / "default" / split
        if split_dir.exists():
            num_images = len(list(split_dir.glob("*.jpg"))) + len(
                list(split_dir.glob("*.png"))
            )
            logger.info(f"{split.capitalize()} set: {num_images} images")


if __name__ == "__main__":
    main()
