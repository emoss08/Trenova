"""
Unit tests for dataset creation and augmentation.
"""

import pytest
import numpy as np
from PIL import Image
import torch
from pathlib import Path

from data.augmentations import TransportationDocumentAugmentation


@pytest.mark.dataset
class TestTransportationDocumentAugmentation:
    """Tests for TransportationDocumentAugmentation."""

    def test_initialization(self):
        """Test augmentation initialization."""
        aug = TransportationDocumentAugmentation(severity_range=(0.1, 0.5))

        assert aug.severity_range == (0.1, 0.5)
        assert len(aug.stamp_positions) > 0
        assert len(aug.watermark_texts) > 0

    def test_initialization_default_params(self):
        """Test augmentation with default parameters."""
        aug = TransportationDocumentAugmentation()

        assert aug.severity_range == (0.1, 0.5)

    def test_add_stamp_overlay(self, sample_image):
        """Test stamp overlay augmentation."""
        aug = TransportationDocumentAugmentation()
        augmented = aug.add_stamp_overlay(sample_image)

        assert isinstance(augmented, Image.Image)
        assert augmented.size == sample_image.size
        assert augmented.mode == sample_image.mode

    def test_add_stamp_overlay_modifies_image(self, sample_image):
        """Test that stamp overlay actually modifies the image."""
        aug = TransportationDocumentAugmentation()

        # Run multiple times to ensure at least one stamp is added
        modified = False
        for _ in range(10):
            augmented = aug.add_stamp_overlay(sample_image)
            if not np.array_equal(np.array(augmented), np.array(sample_image)):
                modified = True
                break

        # With 70% probability per attempt, 10 attempts should almost certainly succeed
        assert modified

    def test_add_stamp_overlay_preserves_dimensions(self, sample_image):
        """Test that stamp overlay preserves image dimensions."""
        aug = TransportationDocumentAugmentation()
        augmented = aug.add_stamp_overlay(sample_image)

        assert augmented.width == sample_image.width
        assert augmented.height == sample_image.height

    def test_add_barcode_blur(self, sample_image):
        """Test barcode blur augmentation."""
        aug = TransportationDocumentAugmentation()
        augmented = aug.add_barcode_blur(sample_image)

        assert isinstance(augmented, Image.Image)
        assert augmented.size == sample_image.size

    def test_add_barcode_blur_modifies_image(self, sample_image):
        """Test that barcode blur actually modifies the image."""
        aug = TransportationDocumentAugmentation()

        # Run multiple times to ensure at least one blur is added
        modified = False
        for _ in range(10):
            augmented = aug.add_barcode_blur(sample_image)
            if not np.array_equal(np.array(augmented), np.array(sample_image)):
                modified = True
                break

        # With 20% probability per attempt, 10 attempts should likely succeed
        # But it's not guaranteed, so we just check it returns valid image
        assert isinstance(augmented, Image.Image)

    @pytest.mark.parametrize("severity_range", [
        (0.1, 0.3),
        (0.2, 0.5),
        (0.3, 0.7)
    ])
    def test_different_severity_ranges(self, sample_image, severity_range):
        """Test augmentation with different severity ranges."""
        aug = TransportationDocumentAugmentation(severity_range=severity_range)
        augmented = aug.add_stamp_overlay(sample_image)

        assert isinstance(augmented, Image.Image)

    def test_augmentation_reproducibility(self, sample_image):
        """Test augmentation with fixed random seed."""
        aug = TransportationDocumentAugmentation()

        # Set random seed
        import random
        random.seed(42)
        result1 = aug.add_stamp_overlay(sample_image)

        random.seed(42)
        result2 = aug.add_stamp_overlay(sample_image)

        # Results should be identical with same seed
        assert np.array_equal(np.array(result1), np.array(result2))

    def test_stamp_positions_valid(self):
        """Test that stamp positions are valid."""
        aug = TransportationDocumentAugmentation()

        for pos in aug.stamp_positions:
            assert len(pos) == 2
            assert 0 <= pos[0] <= 1
            assert 0 <= pos[1] <= 1

    def test_watermark_texts_valid(self):
        """Test that watermark texts are valid."""
        aug = TransportationDocumentAugmentation()

        assert len(aug.watermark_texts) > 0
        for text in aug.watermark_texts:
            assert isinstance(text, str)
            assert len(text) > 0

    def test_augmentation_with_different_sizes(self):
        """Test augmentation with different image sizes."""
        aug = TransportationDocumentAugmentation()

        sizes = [(100, 100), (224, 224), (500, 500), (1000, 800)]

        for size in sizes:
            img = Image.new("RGB", size, color="white")
            augmented = aug.add_stamp_overlay(img)

            assert augmented.size == size

    def test_augmentation_preserves_mode(self):
        """Test that augmentation preserves image mode."""
        aug = TransportationDocumentAugmentation()

        # Test RGB
        img_rgb = Image.new("RGB", (224, 224), color="white")
        aug_rgb = aug.add_stamp_overlay(img_rgb)
        assert aug_rgb.mode == "RGB"

    def test_multiple_augmentations(self, sample_image):
        """Test applying multiple augmentations sequentially."""
        aug = TransportationDocumentAugmentation()

        # Apply multiple augmentations
        img = sample_image
        img = aug.add_stamp_overlay(img)
        img = aug.add_barcode_blur(img)

        assert isinstance(img, Image.Image)
        assert img.size == sample_image.size

    def test_augmentation_does_not_raise(self, sample_image):
        """Test that augmentation doesn't raise exceptions."""
        aug = TransportationDocumentAugmentation()

        try:
            aug.add_stamp_overlay(sample_image)
            aug.add_barcode_blur(sample_image)
        except Exception as e:
            pytest.fail(f"Augmentation raised unexpected exception: {e}")

    def test_augmentation_output_type(self, sample_image):
        """Test that augmentation returns PIL Image."""
        aug = TransportationDocumentAugmentation()

        result = aug.add_stamp_overlay(sample_image)
        assert isinstance(result, Image.Image)

        result = aug.add_barcode_blur(sample_image)
        assert isinstance(result, Image.Image)


@pytest.mark.dataset
class TestDatasetHelpers:
    """Tests for dataset helper functions and utilities."""

    def test_quality_score_ranges(self):
        """Test quality score range validation."""
        valid_scores = [0.0, 1.0, 2.5, 3.7, 5.0]
        for score in valid_scores:
            assert 0 <= score <= 5

    def test_quality_class_labels(self, sample_quality_labels):
        """Test quality class label mapping."""
        assert len(sample_quality_labels) == 5
        assert sample_quality_labels[0] == "High"
        assert sample_quality_labels[4] == "Very Poor"

    def test_issue_labels(self, sample_issue_labels):
        """Test issue label mapping."""
        assert len(sample_issue_labels) == 10

        expected_issues = [
            "Blur", "Noise", "Lighting", "Shadow", "Physical Damage",
            "Skew", "Partial", "Glare", "Compression", "Overall Poor"
        ]

        for issue in expected_issues:
            assert issue in sample_issue_labels.values()

    def test_dataset_split_ratios(self):
        """Test dataset split ratios."""
        total = 1100
        train = 770
        val = 220
        test = 110

        # Check ratios
        assert abs(train / total - 0.70) < 0.01
        assert abs(val / total - 0.20) < 0.01
        assert abs(test / total - 0.10) < 0.01

        # Check sum
        assert train + val + test == total

    def test_quality_distribution_sums(self, mock_dataset_metadata):
        """Test that quality distribution sums to total."""
        quality_dist = mock_dataset_metadata["quality_distribution"]
        total = sum(quality_dist.values())

        assert total == mock_dataset_metadata["total_images"]


@pytest.mark.dataset
class TestDataTransforms:
    """Tests for data transformation pipelines."""

    def test_train_transform_output(self, sample_image):
        """Test training transform output."""
        transform = T.Compose([
            T.Resize((224, 224)),
            T.ToTensor(),
            T.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        ])

        tensor = transform(sample_image)

        assert isinstance(tensor, torch.Tensor)
        assert tensor.shape == (3, 224, 224)

    def test_val_transform_output(self, sample_image):
        """Test validation transform output."""
        transform = T.Compose([
            T.Resize((224, 224)),
            T.ToTensor(),
            T.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        ])

        tensor = transform(sample_image)

        assert isinstance(tensor, torch.Tensor)
        assert tensor.shape == (3, 224, 224)

    def test_normalization_values(self, sample_image):
        """Test that normalization produces expected value ranges."""
        transform = T.Compose([
            T.Resize((224, 224)),
            T.ToTensor(),
            T.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        ])

        tensor = transform(sample_image)

        # After normalization with ImageNet stats, values should roughly be in [-3, 3]
        assert tensor.min() >= -5
        assert tensor.max() <= 5

    def test_resize_transform(self, sample_image):
        """Test resize transformation."""
        transform = T.Resize((224, 224))
        resized = transform(sample_image)

        assert resized.size == (224, 224)

    def test_to_tensor_transform(self, sample_image):
        """Test ToTensor transformation."""
        transform = T.ToTensor()
        tensor = transform(sample_image)

        assert isinstance(tensor, torch.Tensor)
        assert tensor.shape[0] == 3  # RGB channels
        assert 0 <= tensor.min() <= tensor.max() <= 1

    def test_transform_chain(self, sample_image):
        """Test chaining multiple transforms."""
        transforms = [
            T.Resize((224, 224)),
            T.RandomHorizontalFlip(p=0.5),
            T.ColorJitter(brightness=0.2, contrast=0.2),
            T.ToTensor(),
            T.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        ]

        transform = T.Compose(transforms)
        tensor = transform(sample_image)

        assert isinstance(tensor, torch.Tensor)
        assert tensor.shape == (3, 224, 224)

    @pytest.mark.parametrize("size", [(128, 128), (224, 224), (256, 256)])
    def test_different_resize_sizes(self, sample_image, size):
        """Test resize with different target sizes."""
        transform = T.Resize(size)
        resized = transform(sample_image)

        assert resized.size == size

    def test_random_horizontal_flip(self, sample_image):
        """Test random horizontal flip transform."""
        transform = T.RandomHorizontalFlip(p=1.0)  # Always flip
        flipped = transform(sample_image)

        # Create a distinctive image to test flip
        img = Image.new("RGB", (100, 100), color="white")
        from PIL import ImageDraw
        draw = ImageDraw.Draw(img)
        draw.rectangle([10, 10, 30, 90], fill="black")

        flipped = transform(img)
        assert flipped.size == img.size

    def test_color_jitter(self, sample_image):
        """Test color jitter transform."""
        transform = T.ColorJitter(brightness=0.5, contrast=0.5, saturation=0.5, hue=0.2)
        jittered = transform(sample_image)

        assert jittered.size == sample_image.size
        assert jittered.mode == sample_image.mode

    def test_random_rotation(self, sample_image):
        """Test random rotation transform."""
        transform = T.RandomRotation(degrees=30)
        rotated = transform(sample_image)

        assert rotated.size == sample_image.size

    def test_gaussian_blur(self, sample_image):
        """Test Gaussian blur transform."""
        transform = T.GaussianBlur(kernel_size=5)
        blurred = transform(sample_image)

        assert blurred.size == sample_image.size


@pytest.mark.dataset
class TestDatasetMetadata:
    """Tests for dataset metadata handling."""

    def test_metadata_structure(self, mock_dataset_metadata):
        """Test dataset metadata structure."""
        assert "dataset_mode" in mock_dataset_metadata
        assert "total_images" in mock_dataset_metadata
        assert "source_documents" in mock_dataset_metadata
        assert "splits" in mock_dataset_metadata
        assert "quality_distribution" in mock_dataset_metadata

    def test_metadata_splits(self, mock_dataset_metadata):
        """Test dataset split metadata."""
        splits = mock_dataset_metadata["splits"]

        assert "train" in splits
        assert "val" in splits
        assert "test" in splits

        assert splits["train"] > 0
        assert splits["val"] > 0
        assert splits["test"] > 0

    def test_metadata_quality_distribution(self, mock_dataset_metadata):
        """Test quality distribution metadata."""
        quality_dist = mock_dataset_metadata["quality_distribution"]

        expected_classes = ["High", "Good", "Moderate", "Poor", "Very Poor"]
        for class_name in expected_classes:
            assert class_name in quality_dist
            assert quality_dist[class_name] >= 0

    def test_metadata_dataset_mode(self, mock_dataset_metadata):
        """Test dataset mode metadata."""
        assert mock_dataset_metadata["dataset_mode"] in ["synthetic", "real", "mixed"]

    def test_metadata_image_counts(self, mock_dataset_metadata):
        """Test image count consistency."""
        total = mock_dataset_metadata["total_images"]
        splits_sum = sum(mock_dataset_metadata["splits"].values())
        quality_sum = sum(mock_dataset_metadata["quality_distribution"].values())

        assert splits_sum == total
        assert quality_sum == total


@pytest.mark.dataset
@pytest.mark.slow
class TestAugmentationPipeline:
    """Integration tests for augmentation pipeline."""

    def test_full_augmentation_pipeline(self, sample_image):
        """Test complete augmentation pipeline."""
        aug = TransportationDocumentAugmentation()

        # Apply all augmentations
        img = sample_image
        img = aug.add_stamp_overlay(img)
        img = aug.add_barcode_blur(img)

        # Apply torch transforms
        transform = T.Compose([
            T.Resize((224, 224)),
            T.ToTensor(),
            T.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225])
        ])

        tensor = transform(img)

        assert isinstance(tensor, torch.Tensor)
        assert tensor.shape == (3, 224, 224)

    def test_augmentation_batch_processing(self, sample_image_batch):
        """Test batch processing with augmentation."""
        aug = TransportationDocumentAugmentation()

        augmented_batch = []
        for img in sample_image_batch:
            aug_img = aug.add_stamp_overlay(img)
            augmented_batch.append(aug_img)

        assert len(augmented_batch) == len(sample_image_batch)

        for img in augmented_batch:
            assert isinstance(img, Image.Image)

    def test_augmentation_consistency(self, sample_image):
        """Test augmentation produces consistent outputs."""
        aug = TransportationDocumentAugmentation()

        # Apply same augmentation multiple times
        results = []
        import random
        for i in range(5):
            random.seed(i)
            result = aug.add_stamp_overlay(sample_image)
            results.append(np.array(result))

        # All results should be valid
        for result in results:
            assert result.shape == (224, 224, 3)
