#!/usr/bin/env python3
#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
"""
Command-line interface for Document Quality Assessment

Usage:
    python cli.py analyze <file_path> [--output <output_path>]
    python cli.py batch <directory_path> [--output <output_path>]
    python cli.py train --dataset <dataset_path>
"""

import argparse
import json
import sys
from pathlib import Path
from typing import Optional

import torch
from torch.utils.data import DataLoader
from torchvision import transforms

from ..api.api_interface import DocumentQualityAPI
from ..data.dataset import create_enhanced_dataset
from ..models.analyzer import DocumentAnalyzer
from ..models.model import DocumentQualityDataset, create_model, train_model


def analyze_document(file_path: str, output_path: Optional[str] = None):
    """Analyze a single document and print results"""
    print(f"\n📄 Analyzing document: {file_path}")
    print("-" * 50)

    analyzer = DocumentAnalyzer("document_quality_model.pth")

    try:
        result = analyzer.analyze_document(file_path)

        # Print summary
        print(f"✅ Quality Score: {result.overall_quality_score:.2f}/1.00")
        print(f"📊 Confidence: {result.overall_confidence:.1%}")
        print(f"🏷️  Category: {result.quality_category}")
        print(
            f"{'✅' if result.is_acceptable else '❌'} Acceptable: {'Yes' if result.is_acceptable else 'No'}"
        )

        # Print transportation fields if found
        if result.transportation_fields:
            print("\n🚚 Transportation Fields Detected:")
            for field, detected in result.transportation_fields.items():
                if detected and field not in [
                    "is_transportation_doc",
                    "transportation_confidence",
                ]:
                    print(f"  • {field.upper()}")

        # Print issues
        if result.issues:
            print("\n⚠️  Issues Found:")
            critical = [i for i in result.issues if i.severity == "critical"]
            major = [i for i in result.issues if i.severity == "major"]

            if critical:
                print("  Critical:")
                for issue in critical:
                    print(f"    • {issue.description}")
            if major:
                print("  Major:")
                for issue in major:
                    print(f"    • {issue.description}")

        # Print recommendations
        print("\n💡 Recommendations:")
        for i, rec in enumerate(result.recommendations, 1):
            print(f"  {i}. {rec}")

        # Save detailed report if output path provided
        if output_path:
            report = analyzer.generate_report(result, Path(output_path))
            print(f"\n📝 Detailed report saved to: {output_path}")

    except Exception as e:
        print(f"❌ Error: {str(e)}")
        sys.exit(1)


def batch_analyze(directory_path: str, output_path: Optional[str] = None):
    """Analyze all documents in a directory"""
    dir_path = Path(directory_path)

    if not dir_path.exists():
        print(f"❌ Directory not found: {directory_path}")
        sys.exit(1)

    # Find all supported files
    supported_extensions = [".pdf", ".jpg", ".jpeg", ".png", ".tiff", ".bmp"]
    files: list[Path | str] = []
    for ext in supported_extensions:
        files.extend(dir_path.glob(f"*{ext}"))
        files.extend(dir_path.glob(f"*{ext.upper()}"))

    if not files:
        print(f"❌ No supported documents found in {directory_path}")
        sys.exit(1)

    print(f"\n📁 Found {len(files)} documents to analyze")
    print("-" * 50)

    analyzer = DocumentAnalyzer("document_quality_model.pth")

    # Analyze documents
    results_df = analyzer.batch_analyze(
        files, Path(output_path) if output_path else None
    )

    # Print summary statistics
    print("\n📊 Analysis Summary:")
    print(f"  Total documents: {len(results_df)}")

    if "is_acceptable" in results_df.columns:
        acceptable = results_df["is_acceptable"].sum()
        print(f"  Acceptable: {acceptable} ({acceptable / len(results_df):.1%})")
        print(
            f"  Rejected: {len(results_df) - acceptable} ({(len(results_df) - acceptable) / len(results_df):.1%})"
        )

    if "quality_category" in results_df.columns:
        print("\n  Quality Distribution:")
        for category, count in results_df["quality_category"].value_counts().items():
            print(f"    {category}: {count}")

    if output_path:
        print(f"\n📝 Results saved to: {output_path}")


def train_document_model(dataset_path: str):
    """Train a new document quality model"""
    dataset_dir = Path(dataset_path)

    if not dataset_dir.exists():
        print(f"❌ Dataset directory not found: {dataset_path}")
        sys.exit(1)

    print(f"\n🔧 Training document quality model")
    print(f"📁 Dataset: {dataset_path}")
    print("-" * 50)

    # Parameters
    batch_size = 32
    num_epochs = 10
    learning_rate = 0.001

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    print(f"🖥️  Device: {device}")

    # Data transforms
    data_transforms = {
        "train": transforms.Compose(
            [
                transforms.Resize((224, 224)),
                transforms.RandomHorizontalFlip(),
                transforms.ToTensor(),
                transforms.Normalize([0.485, 0.456, 0.406], [0.229, 0.224, 0.225]),
            ]
        ),
        "val": transforms.Compose(
            [
                transforms.Resize((224, 224)),
                transforms.ToTensor(),
                transforms.Normalize([0.485, 0.456, 0.406], [0.229, 0.224, 0.225]),
            ]
        ),
    }

    # Load dataset
    dataset_csv = dataset_dir / "dataset_metadata.csv"
    if not dataset_csv.exists():
        print(f"❌ Dataset metadata not found: {dataset_csv}")
        sys.exit(1)

    # Create datasets
    full_dataset = DocumentQualityDataset(
        str(dataset_csv), str(dataset_dir), transform=data_transforms["train"]
    )

    # Split into train and validation
    train_size = int(0.8 * len(full_dataset))
    val_size = len(full_dataset) - train_size
    train_dataset, val_dataset = torch.utils.data.random_split(
        full_dataset, [train_size, val_size]
    )

    # Create dataloaders
    dataloaders = {
        "train": DataLoader(train_dataset, batch_size=batch_size, shuffle=True),
        "val": DataLoader(val_dataset, batch_size=batch_size, shuffle=False),
    }

    print(f"📊 Dataset size: {len(full_dataset)} images")
    print(f"   Training: {train_size}")
    print(f"   Validation: {val_size}")

    # Create model
    model = create_model().to(device)
    criterion = torch.nn.MSELoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=learning_rate)

    # Train
    print("\n🚀 Starting training...")
    model = train_model(
        model,
        dataloaders,
        criterion,
        optimizer,
        num_epochs=num_epochs,
        patience=3,
        device=device,
    )

    # Save model
    torch.save(model.state_dict(), "document_quality_model.pth")
    print("\n✅ Model saved as document_quality_model.pth")


def create_dataset_from_docs(input_dir: str, output_dir: str):
    """Create a synthetic dataset from document images"""
    print(f"\n🎨 Creating synthetic dataset")
    print(f"📁 Input: {input_dir}")
    print(f"📁 Output: {output_dir}")
    print("-" * 50)

    create_enhanced_dataset(Path(input_dir), Path(output_dir))
    print("✅ Dataset creation complete!")


def main():
    parser = argparse.ArgumentParser(
        description="Document Quality Assessment CLI",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Analyze a single document
  python cli.py analyze invoice.pdf
  
  # Analyze with detailed report
  python cli.py analyze invoice.pdf --output report.txt
  
  # Batch analyze directory
  python cli.py batch ./documents --output results.csv
  
  # Train new model
  python cli.py train --dataset ./dataset
  
  # Create synthetic dataset
  python cli.py create-dataset --input ./clean_docs --output ./dataset
        """,
    )

    subparsers = parser.add_subparsers(dest="command", help="Commands")

    # Analyze command
    analyze_parser = subparsers.add_parser("analyze", help="Analyze a single document")
    analyze_parser.add_argument("file_path", help="Path to document file")
    analyze_parser.add_argument("--output", help="Output path for detailed report")

    # Batch command
    batch_parser = subparsers.add_parser("batch", help="Analyze multiple documents")
    batch_parser.add_argument("directory_path", help="Directory containing documents")
    batch_parser.add_argument("--output", help="Output CSV path")

    # Train command
    train_parser = subparsers.add_parser("train", help="Train quality assessment model")
    train_parser.add_argument("--dataset", required=True, help="Dataset directory path")

    # Create dataset command
    dataset_parser = subparsers.add_parser(
        "create-dataset", help="Create synthetic dataset"
    )
    dataset_parser.add_argument(
        "--input", required=True, help="Input documents directory"
    )
    dataset_parser.add_argument(
        "--output", required=True, help="Output dataset directory"
    )

    args = parser.parse_args()

    if args.command == "analyze":
        analyze_document(args.file_path, args.output)
    elif args.command == "batch":
        batch_analyze(args.directory_path, args.output)
    elif args.command == "train":
        train_document_model(args.dataset)
    elif args.command == "create-dataset":
        create_dataset_from_docs(args.input, args.output)
    else:
        parser.print_help()
        sys.exit(1)


if __name__ == "__main__":
    main()
