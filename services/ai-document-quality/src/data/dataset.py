# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
import io
import json
import logging
import random
from dataclasses import asdict, dataclass
from multiprocessing import Pool, cpu_count
from pathlib import Path
from typing import Dict, List, Optional, Tuple

import cv2
import numpy as np
import pandas as pd
import torch
import torchvision.transforms as transforms
from PIL import Image, ImageDraw, ImageEnhance, ImageFilter
from torch.utils.data import Dataset
from tqdm import tqdm


@dataclass
class DocumentDegradation:
    """Enhanced degradation parameters for realistic ELD document issues"""

    blur_radius: float = 0.0
    noise_level: float = 0.0
    compression_quality: int = 100
    brightness_factor: float = 1.0
    contrast_factor: float = 1.0
    shadow_intensity: float = 0.0
    fold_intensity: float = 0.0
    smear_intensity: float = 0.0
    skew_angle: float = 0.0
    coffee_stain: bool = False
    crumple_factor: float = 0.0
    jpeg_artifacts: float = 0.0
    water_damage: bool = False
    highlighting: float = 0.0
    # New ELD-specific degradations
    motion_blur: float = 0.0  # From camera shake
    lens_distortion: float = 0.0  # From wide-angle lens
    glare_spots: int = 0  # From dashboard reflections
    low_light_noise: float = 0.0  # From night captures
    perspective_warp: float = 0.0  # From angled shots
    finger_shadow: bool = False  # From holding document
    partial_capture: float = 0.0  # Document cut off
    over_exposure: float = 0.0  # From direct sunlight

    def get_quality_score(self) -> float:
        """Calculate expected quality score based on degradations with improved weighting"""
        score = 1.0

        # Critical factors that severely impact readability
        critical_factors = [
            # (value, threshold, max_penalty)
            (self.blur_radius, 1.5, 0.7),  # Focus blur is critical
            (self.motion_blur, 3.0, 0.75),  # Motion blur is very bad
            (self.partial_capture, 0.1, 0.8),  # Missing content is critical
        ]

        for value, threshold, max_penalty in critical_factors:
            if value > threshold:
                penalty = min(max_penalty, (value - threshold) / (10.0 - threshold))
                score *= 1 - penalty

        # Major factors - significant but not critical
        major_factors = [
            (self.low_light_noise, 20.0, 0.5),
            (self.over_exposure, 0.3, 0.5),
            (abs(self.skew_angle), 10.0, 0.4),
            (self.perspective_warp, 0.2, 0.4),
        ]

        for value, threshold, max_penalty in major_factors:
            if value > threshold:
                penalty = min(max_penalty, (value - threshold) / (1.0 - threshold))
                score *= 1 - penalty

        # Moderate factors
        if self.noise_level > 20:
            score *= max(0.5, 1 - (self.noise_level / 200.0))
        if self.compression_quality < 70:
            score *= max(0.5, self.compression_quality / 100.0)
        if abs(self.brightness_factor - 1.0) > 0.2:
            score *= max(0.6, 1 - abs(self.brightness_factor - 1.0) * 0.5)
        if abs(self.contrast_factor - 1.0) > 0.2:
            score *= max(0.6, 1 - abs(self.contrast_factor - 1.0) * 0.5)

        # Physical document issues (less critical for digital assessment)
        if self.shadow_intensity > 0.3:
            score *= max(0.6, 1 - self.shadow_intensity * 0.5)
        if self.finger_shadow:
            score *= 0.9  # Minor penalty
        if self.fold_intensity > 0.3:
            score *= max(0.7, 1 - self.fold_intensity * 0.4)
        if self.crumple_factor > 0.3:
            score *= max(0.6, 1 - self.crumple_factor * 0.5)

        # Environmental damage
        if self.coffee_stain:
            score *= 0.75
        if self.water_damage:
            score *= 0.65
        if self.glare_spots > 1:
            score *= max(0.6, 1 - (self.glare_spots * 0.1))

        # Apply non-linear transformation for more realistic distribution
        # This creates a more natural quality distribution
        score = np.power(score, 0.8)

        return max(0.05, min(1.0, score))  # Clamp between 0.05 and 1.0

    def get_issue_labels(self) -> Dict[str, bool]:
        """Get binary labels for different quality issues"""
        return {
            "issue_blur": self.blur_radius > 2.0 or self.motion_blur > 5.0,
            "issue_noise": self.noise_level > 30 or self.low_light_noise > 40,
            "issue_lighting": abs(self.brightness_factor - 1.0) > 0.3
            or abs(self.contrast_factor - 1.0) > 0.3
            or self.over_exposure > 0.3,
            "issue_shadow": self.shadow_intensity > 0.3 or self.finger_shadow,
            "issue_physical_damage": self.fold_intensity > 0.3
            or self.crumple_factor > 0.3
            or self.coffee_stain
            or self.water_damage,
            "issue_skew": self.skew_angle > 5.0 or self.perspective_warp > 0.3,
            "issue_partial": self.partial_capture > 0.1,
            "issue_glare": self.glare_spots > 0,
            "issue_compression": self.compression_quality < 70
            or self.jpeg_artifacts > 0.5,
            "issue_overall_poor": self.get_quality_score() < 0.4,
        }


class DocumentDatasetCreator:
    def __init__(
        self,
        output_dir: Path,
        mode: str = "synthetic",
        cpu_fraction: float = 0.75,
        max_cpus: Optional[int] = None,
    ):
        """
        Enhanced dataset creator with multiple modes

        Args:
            output_dir: Output directory for dataset
            mode: 'synthetic' for artificial degradations,
                  'real' for using real documents with labels,
                  'mixed' for combination
            cpu_fraction: Fraction of CPUs to use (0.0-1.0)
            max_cpus: Maximum number of CPUs to use (overrides cpu_fraction)
        """
        self.logger = logging.getLogger(__name__)
        self.output_dir = output_dir
        self.output_dir.mkdir(parents=True, exist_ok=True)
        self.mode = mode
        self.cpu_fraction = cpu_fraction
        self.max_cpus = max_cpus

        # Define quality categories
        self.quality_categories = {
            (0.8, 1.0): 0,  # High Quality
            (0.6, 0.8): 1,  # Good Quality
            (0.4, 0.6): 2,  # Moderate Quality
            (0.2, 0.4): 3,  # Poor Quality
            (0.0, 0.2): 4,  # Very Poor Quality
        }

    def create_dataset(
        self,
        source_docs: List[Path],
        category: str | None = None,
        split_ratio: Dict[str, float] | None = None,
    ) -> Dict[str, pd.DataFrame]:
        """
        Create enhanced dataset with train/val/test splits

        Args:
            source_docs: List of source document paths
            category: Optional category name
            split_ratio: Dict with 'train', 'val', 'test' ratios (default: 70/20/10)

        Returns:
            Dict with DataFrames for each split
        """
        if split_ratio is None:
            split_ratio = {"train": 0.7, "val": 0.2, "test": 0.1}

        output_dir = self.output_dir / (category or "default")
        output_dir.mkdir(exist_ok=True)

        # Shuffle documents for random split
        random.shuffle(source_docs)

        # Calculate split points
        n_docs = len(source_docs)
        train_end = int(n_docs * split_ratio["train"])
        val_end = train_end + int(n_docs * split_ratio["val"])

        splits = {
            "train": source_docs[:train_end],
            "val": source_docs[train_end:val_end],
            "test": source_docs[val_end:],
        }

        all_metadata = {}

        for split_name, split_docs in splits.items():
            split_dir = output_dir / split_name
            split_dir.mkdir(exist_ok=True)

            # Process documents for this split
            metadata_list = []

            # Prepare arguments for multiprocessing
            args = [(doc_path, split_dir, split_name) for doc_path in split_docs]

            # Use Pool to parallelize - leave some CPUs free for system
            num_cpus = cpu_count()

            if self.max_cpus is not None:
                # Use specified max CPUs
                num_workers = min(self.max_cpus, num_cpus)
            else:
                # Use fraction of CPUs, but leave at least 1 free
                num_workers = max(
                    1, min(int(num_cpus * self.cpu_fraction), num_cpus - 1)
                )

            self.logger.info(
                f"Using {num_workers} out of {num_cpus} CPUs for processing"
            )

            with Pool(processes=num_workers) as pool:
                results_iter = pool.imap(self._process_single_doc, args)
                for doc_metadata in tqdm(
                    results_iter,
                    total=len(split_docs),
                    desc=f"Processing {split_name} split",
                ):
                    metadata_list.extend(doc_metadata)

            # Create DataFrame for this split
            df = pd.DataFrame(metadata_list)
            all_metadata[split_name] = df

            # Save metadata CSV for this split
            csv_path = split_dir / f"{split_name}_metadata.csv"
            df.to_csv(csv_path, index=False)
            self.logger.info(f"Saved {split_name} metadata to {csv_path}")

        return all_metadata

    def _process_single_doc(self, args) -> List[Dict]:
        """Process a single document and return metadata list"""
        doc_path, output_dir, split_name = args
        metadata_list = []

        try:
            doc = Image.open(doc_path).convert("RGB")

            # Save the original with metadata
            original_path = output_dir / f"{doc_path.stem}_original.jpg"
            doc.save(original_path, format="JPEG", quality=100)

            # Get quality category
            quality_class = 0  # High quality for original

            metadata = {
                "filepath": str(original_path.relative_to(self.output_dir)),
                "quality_score": 1.0,
                "quality_class": quality_class,
                "degradation_type": "original",
                "split": split_name,
                "source_document": doc_path.name,
            }

            # Add issue labels (all False for original)
            issue_labels = {
                f"issue_{k}": False
                for k in [
                    "blur",
                    "noise",
                    "lighting",
                    "shadow",
                    "physical_damage",
                    "skew",
                    "partial",
                    "glare",
                    "compression",
                    "overall_poor",
                ]
            }
            metadata.update(issue_labels)
            metadata_list.append(metadata)

            # Generate variations
            variations = self._generate_variations(doc)
            for idx, (variant, degradation) in enumerate(variations):
                # Make sure filename is unique
                variant_path = (
                    output_dir
                    / f"{doc_path.stem}_variant_{idx}_{random.randint(0, 999999)}.jpg"
                )
                variant.save(variant_path, format="JPEG")

                # Get quality score and category
                quality_score = degradation.get_quality_score()
                quality_class = self._get_quality_class(quality_score)

                metadata = {
                    "filepath": str(variant_path.relative_to(self.output_dir)),
                    "quality_score": quality_score,
                    "quality_class": quality_class,
                    "degradation_type": "synthetic",
                    "split": split_name,
                    "source_document": doc_path.name,
                }

                # Add issue labels
                metadata.update(degradation.get_issue_labels())

                # Add degradation parameters for analysis
                metadata["degradation_params"] = json.dumps(asdict(degradation))

                metadata_list.append(metadata)

        except Exception as e:
            self.logger.error(f"Error processing {doc_path}: {str(e)}")

        return metadata_list

    def _get_quality_class(self, score: float) -> int:
        """Convert quality score to class label"""
        for (min_score, max_score), class_label in self.quality_categories.items():
            if min_score <= score < max_score:
                return class_label
        return 4  # Default to very poor if something goes wrong

    def _generate_variations(
        self, doc: Image.Image
    ) -> List[Tuple[Image.Image, DocumentDegradation]]:
        """Generate realistic document quality variations"""
        variations = []

        # Define quality distribution (more realistic bell curve)
        quality_distribution = [
            ("high", 0.15),  # 15% high quality
            ("good", 0.35),  # 35% good quality
            ("moderate", 0.30),  # 30% moderate quality
            ("poor", 0.15),  # 15% poor quality
            ("very_poor", 0.05),  # 5% very poor quality
        ]

        # Generate 10-20 variations per document
        num_variations = random.randint(10, 20)

        for _ in range(num_variations):
            # Sample quality level
            quality_level = np.random.choice(
                [q[0] for q in quality_distribution],
                p=[q[1] for q in quality_distribution],
            )

            # Generate degradation based on quality level
            degradation = self._generate_quality_based_degradation(quality_level)

            # Apply degradations
            variant = self._apply_degradations(doc.copy(), degradation)
            variations.append((variant, degradation))

        return variations

    def _generate_quality_based_degradation(
        self, quality_level: str
    ) -> DocumentDegradation:
        """Generate degradation parameters based on target quality level"""

        if quality_level == "high":
            # Minimal degradations - near perfect capture
            return DocumentDegradation(
                blur_radius=random.uniform(0, 0.5),
                compression_quality=random.randint(90, 95),
                skew_angle=random.uniform(-2, 2),
                brightness_factor=random.uniform(0.95, 1.05),
                contrast_factor=random.uniform(0.95, 1.05),
                noise_level=random.uniform(0, 5),
            )

        elif quality_level == "good":
            # Minor issues but still very readable
            deg = DocumentDegradation(
                blur_radius=random.uniform(0.5, 1.5),
                compression_quality=random.randint(80, 90),
                skew_angle=random.uniform(-5, 5),
                brightness_factor=random.uniform(0.85, 1.15),
                contrast_factor=random.uniform(0.85, 1.15),
                noise_level=random.uniform(5, 15),
            )

            # Add occasional minor issues
            if random.random() < 0.3:
                deg.glare_spots = 1
            if random.random() < 0.2:
                deg.shadow_intensity = random.uniform(0.1, 0.2)

            return deg

        elif quality_level == "moderate":
            # Noticeable issues but still usable
            deg = DocumentDegradation(
                blur_radius=random.uniform(1, 3),
                compression_quality=random.randint(65, 80),
                skew_angle=random.uniform(-10, 10),
                brightness_factor=random.uniform(0.7, 1.3),
                contrast_factor=random.uniform(0.7, 1.3),
                noise_level=random.uniform(10, 30),
                shadow_intensity=random.uniform(0.1, 0.3),
            )

            # Common moderate issues
            if random.random() < 0.4:
                deg.motion_blur = random.uniform(2, 5)
            if random.random() < 0.3:
                deg.glare_spots = random.randint(1, 2)
            if random.random() < 0.3:
                deg.perspective_warp = random.uniform(0.1, 0.2)
            if random.random() < 0.2:
                deg.finger_shadow = True

            return deg

        elif quality_level == "poor":
            # Significant issues affecting readability
            deg = DocumentDegradation(
                blur_radius=random.uniform(2, 5),
                compression_quality=random.randint(50, 70),
                skew_angle=random.uniform(-20, 20),
                brightness_factor=random.uniform(0.5, 1.5),
                contrast_factor=random.uniform(0.5, 1.5),
                noise_level=random.uniform(20, 50),
                shadow_intensity=random.uniform(0.2, 0.5),
                motion_blur=random.uniform(3, 8),
            )

            # Add multiple issues
            if random.random() < 0.5:
                deg.glare_spots = random.randint(2, 4)
            if random.random() < 0.4:
                deg.perspective_warp = random.uniform(0.2, 0.4)
            if random.random() < 0.3:
                deg.low_light_noise = random.uniform(30, 60)
            if random.random() < 0.3:
                deg.partial_capture = random.uniform(0.05, 0.15)
            if random.random() < 0.3:
                deg.fold_intensity = random.uniform(0.3, 0.6)

            return deg

        else:  # very_poor
            # Severe issues - barely readable
            deg = DocumentDegradation(
                blur_radius=random.uniform(4, 8),
                compression_quality=random.randint(30, 50),
                skew_angle=random.uniform(-30, 30),
                brightness_factor=random.choice(
                    [random.uniform(0.3, 0.5), random.uniform(1.5, 2.0)]
                ),
                contrast_factor=random.choice(
                    [random.uniform(0.3, 0.5), random.uniform(1.5, 2.0)]
                ),
                noise_level=random.uniform(40, 80),
                shadow_intensity=random.uniform(0.4, 0.7),
                motion_blur=random.uniform(5, 15),
                glare_spots=random.randint(3, 6),
                perspective_warp=random.uniform(0.3, 0.5),
                low_light_noise=random.uniform(50, 100),
            )

            # Multiple severe issues
            if random.random() < 0.5:
                deg.partial_capture = random.uniform(0.1, 0.3)
            if random.random() < 0.4:
                deg.water_damage = True
            if random.random() < 0.4:
                deg.crumple_factor = random.uniform(0.5, 0.8)
            if random.random() < 0.3:
                deg.over_exposure = random.uniform(0.5, 0.8)

            return deg

    def _apply_degradations(
        self, image: Image.Image, deg: DocumentDegradation
    ) -> Image.Image:
        """Apply all degradations to an image"""
        # Apply ELD-specific degradations first
        if deg.perspective_warp > 0:
            image = self._apply_perspective_warp(image, deg.perspective_warp)

        if deg.partial_capture > 0:
            image = self._apply_partial_capture(image, deg.partial_capture)

        if deg.finger_shadow:
            image = self._apply_finger_shadow(image)

        if deg.motion_blur > 0:
            image = self._apply_motion_blur(image, deg.motion_blur)

        if deg.glare_spots > 0:
            image = self._apply_glare_spots(image, deg.glare_spots)

        if deg.low_light_noise > 0:
            image = self._apply_low_light_noise(image, deg.low_light_noise)

        if deg.over_exposure > 0:
            image = self._apply_over_exposure(image, deg.over_exposure)

        # Apply standard degradations
        if deg.skew_angle != 0:
            image = self._apply_skew(image, deg.skew_angle)

        if deg.shadow_intensity > 0:
            image = self._apply_shadow(image, deg.shadow_intensity)

        if deg.fold_intensity > 0:
            image = self._apply_fold_marks(image, deg.fold_intensity)

        if deg.smear_intensity > 0:
            image = self._apply_smearing(image, deg.smear_intensity)

        if deg.crumple_factor > 0:
            image = self._apply_crumpling(image, deg.crumple_factor)

        if deg.coffee_stain:
            image = self._apply_stain(image)

        if deg.water_damage:
            image = self._apply_water_damage(image)

        if deg.highlighting > 0:
            image = self._apply_highlighting(image, deg.highlighting)

        # Basic image adjustments
        if deg.blur_radius > 0:
            image = image.filter(ImageFilter.GaussianBlur(deg.blur_radius))

        if deg.noise_level > 0:
            image = self._add_noise(image, deg.noise_level)

        if deg.brightness_factor != 1.0:
            image = ImageEnhance.Brightness(image).enhance(deg.brightness_factor)

        if deg.contrast_factor != 1.0:
            image = ImageEnhance.Contrast(image).enhance(deg.contrast_factor)

        # Apply JPEG compression last
        if deg.compression_quality < 100 or deg.jpeg_artifacts > 0:
            image = self._apply_jpeg_artifacts(
                image, deg.jpeg_artifacts, deg.compression_quality
            )

        return image

    # ==================== New ELD-specific degradation methods ====================

    def _apply_motion_blur(self, image: Image.Image, intensity: float) -> Image.Image:
        """Apply motion blur to simulate camera shake"""
        img_array = np.array(image)
        size = int(intensity)

        # Create motion blur kernel
        kernel = np.zeros((size, size))
        kernel[int((size - 1) / 2), :] = np.ones(size)
        kernel = kernel / size

        # Convert to OpenCV format and apply
        img_cv = cv2.cvtColor(img_array, cv2.COLOR_RGB2BGR)
        blurred = cv2.filter2D(img_cv, -1, kernel)

        # Add slight rotation to kernel for more realistic motion
        angle = random.uniform(-15, 15)
        M = cv2.getRotationMatrix2D((size / 2, size / 2), angle, 1)
        kernel = cv2.warpAffine(kernel, M, (size, size))

        blurred = cv2.filter2D(blurred, -1, kernel)
        result = cv2.cvtColor(blurred, cv2.COLOR_BGR2RGB)

        return Image.fromarray(result)

    def _apply_perspective_warp(
        self, image: Image.Image, intensity: float
    ) -> Image.Image:
        """Apply perspective transformation to simulate angled capture"""
        width, height = image.size
        img_array = np.array(image)

        # Define source points (corners of the image)
        src_points = np.array(
            [[0, 0], [width, 0], [width, height], [0, height]], dtype=np.float32
        )

        # Create destination points with perspective distortion
        offset = int(min(width, height) * intensity * 0.2)
        dst_points = np.array(
            [
                [random.randint(0, offset), random.randint(0, offset)],
                [width - random.randint(0, offset), random.randint(0, offset)],
                [width - random.randint(0, offset), height - random.randint(0, offset)],
                [random.randint(0, offset), height - random.randint(0, offset)],
            ],
            dtype=np.float32,
        )

        # Calculate perspective transform matrix
        matrix = cv2.getPerspectiveTransform(src_points, dst_points)

        # Apply transformation
        img_cv = cv2.cvtColor(img_array, cv2.COLOR_RGB2BGR)
        warped = cv2.warpPerspective(
            img_cv,
            matrix,
            (width, height),
            flags=cv2.INTER_LINEAR,
            borderMode=cv2.BORDER_CONSTANT,
            borderValue=(255, 255, 255),
        )

        result = cv2.cvtColor(warped, cv2.COLOR_BGR2RGB)
        return Image.fromarray(result)

    def _apply_glare_spots(self, image: Image.Image, num_spots: int) -> Image.Image:
        """Add glare spots to simulate dashboard reflections"""
        img = image.copy()
        draw = ImageDraw.Draw(img, "RGBA")
        width, height = img.size

        for _ in range(num_spots):
            # Random position
            x = random.randint(width // 4, 3 * width // 4)
            y = random.randint(height // 4, 3 * height // 4)

            # Random size
            radius = random.randint(30, 100)

            # Create gradient glare
            for r in range(radius, 0, -5):
                opacity = int(255 * (1 - (r / radius) ** 2))
                color = (255, 255, 240, opacity)  # Slightly yellow white
                draw.ellipse([x - r, y - r, x + r, y + r], fill=color)

        return img

    def _apply_finger_shadow(self, image: Image.Image) -> Image.Image:
        """Add finger shadow at edges to simulate holding document"""
        width, height = image.size
        shadow = Image.new("RGBA", (width, height), (0, 0, 0, 0))
        draw = ImageDraw.Draw(shadow)

        # Choose random edge
        edge = random.choice(["top", "bottom", "left", "right"])

        if edge == "top":
            # Finger shadows from top
            for i in range(random.randint(1, 3)):
                x = random.randint(width // 4, 3 * width // 4)
                draw.ellipse([x - 40, -20, x + 40, 60], fill=(0, 0, 0, 120))
        elif edge == "bottom":
            # Finger shadows from bottom
            for i in range(random.randint(1, 3)):
                x = random.randint(width // 4, 3 * width // 4)
                draw.ellipse(
                    [x - 40, height - 60, x + 40, height + 20], fill=(0, 0, 0, 120)
                )
        elif edge == "left":
            # Finger shadows from left
            for i in range(random.randint(1, 2)):
                y = random.randint(height // 4, 3 * height // 4)
                draw.ellipse([-20, y - 40, 60, y + 40], fill=(0, 0, 0, 120))
        else:
            # Finger shadows from right
            for i in range(random.randint(1, 2)):
                y = random.randint(height // 4, 3 * height // 4)
                draw.ellipse(
                    [width - 60, y - 40, width + 20, y + 40], fill=(0, 0, 0, 120)
                )

        # Apply gaussian blur to shadow
        shadow = shadow.filter(ImageFilter.GaussianBlur(radius=10))

        # Composite shadow onto image
        return Image.alpha_composite(image.convert("RGBA"), shadow).convert("RGB")

    def _apply_partial_capture(self, image: Image.Image, amount: float) -> Image.Image:
        """Crop image to simulate partial document capture"""
        width, height = image.size

        # Random side to crop
        side = random.choice(["top", "bottom", "left", "right"])

        if side == "top":
            crop_height = int(height * (1 - amount))
            image = image.crop((0, height - crop_height, width, height))
        elif side == "bottom":
            crop_height = int(height * (1 - amount))
            image = image.crop((0, 0, width, crop_height))
        elif side == "left":
            crop_width = int(width * (1 - amount))
            image = image.crop((width - crop_width, 0, width, height))
        else:
            crop_width = int(width * (1 - amount))
            image = image.crop((0, 0, crop_width, height))

        # Resize back to original dimensions with white padding
        new_img = Image.new("RGB", (width, height), "white")
        if side == "top":
            new_img.paste(image, (0, height - image.height))
        elif side == "bottom":
            new_img.paste(image, (0, 0))
        elif side == "left":
            new_img.paste(image, (width - image.width, 0))
        else:
            new_img.paste(image, (0, 0))

        return new_img

    def _apply_low_light_noise(
        self, image: Image.Image, intensity: float
    ) -> Image.Image:
        """Add noise typical of low-light captures"""
        img_array = np.array(image)

        # Add color noise with higher intensity in darker areas
        gray = cv2.cvtColor(img_array, cv2.COLOR_RGB2GRAY)
        dark_mask = (255 - gray) / 255.0

        # Generate noise
        noise = np.random.normal(0, intensity, img_array.shape)

        # Apply noise more strongly to dark areas
        for i in range(3):
            noise[:, :, i] *= dark_mask

        noisy = img_array + noise

        # Add slight color shift (blue/green tint common in low light)
        noisy[:, :, 1] += random.uniform(0, 5)  # Green channel
        noisy[:, :, 2] += random.uniform(0, 10)  # Blue channel

        return Image.fromarray(np.clip(noisy, 0, 255).astype(np.uint8))

    def _apply_over_exposure(self, image: Image.Image, intensity: float) -> Image.Image:
        """Simulate over-exposure from direct sunlight"""
        img_array = np.array(image, dtype=np.float32)

        # Create exposure mask (brighter areas get more exposed)
        gray = cv2.cvtColor(img_array, cv2.COLOR_RGB2GRAY)
        bright_mask = gray / 255.0

        # Apply exponential brightening to simulate overexposure
        exposure_factor = 1 + intensity * 2
        for i in range(3):
            img_array[:, :, i] = img_array[:, :, i] * (
                1 + bright_mask * (exposure_factor - 1)
            )

        # Clip values
        img_array = np.clip(img_array, 0, 255)

        # Add slight yellow tint (common in overexposed images)
        img_array[:, :, 0] = np.clip(img_array[:, :, 0] * 1.05, 0, 255)  # Red
        img_array[:, :, 1] = np.clip(img_array[:, :, 1] * 1.03, 0, 255)  # Green

        return Image.fromarray(img_array.astype(np.uint8))

    def _apply_shadow(self, image: Image.Image, intensity: float) -> Image.Image:
        width, height = image.size
        shadow = Image.new("RGB", (width, height), "white")
        draw = ImageDraw.Draw(shadow)

        # Create gradient shadow
        for i in range(width):
            shadow_value = int(255 * (1 - intensity * (i / width)))
            draw.line([(i, 0), (i, height)], fill=(shadow_value,) * 3)

        return Image.blend(image, shadow, intensity * 0.5)

    def _apply_fold_marks(self, image: Image.Image, intensity: float) -> Image.Image:
        img_array = np.array(image)
        height, width = img_array.shape[:2]

        # Create horizontal & vertical fold lines
        fold_positions = [
            (int(height * random.uniform(0.3, 0.7)), True),  # Horizontal
            (int(width * random.uniform(0.3, 0.7)), False),  # Vertical
        ]

        for pos, is_horizontal in fold_positions:
            if is_horizontal:
                fold_width = int(height * 0.01)
                img_array[pos - fold_width : pos + fold_width, :] = (
                    img_array[pos - fold_width : pos + fold_width, :]
                    * (1 - intensity * 0.3)
                ).astype(np.uint8)
            else:
                fold_width = int(width * 0.01)
                img_array[:, pos - fold_width : pos + fold_width] = (
                    img_array[:, pos - fold_width : pos + fold_width]
                    * (1 - intensity * 0.3)
                ).astype(np.uint8)

        return Image.fromarray(img_array)

    def _apply_smearing(self, image: Image.Image, intensity: float) -> Image.Image:
        img_array = np.array(image)

        # Create motion blur kernel
        size = int(10 * intensity)
        kernel = np.zeros((size, size))
        kernel[int((size - 1) / 2), :] = np.ones(size)
        kernel = kernel / size

        # Apply kernel
        channels = []
        for i in range(3):
            channels.append(
                np.convolve(
                    img_array[:, :, i].flatten(), kernel.flatten(), mode="same"
                ).reshape(img_array.shape[:2])
            )

        smeared = np.stack(channels, axis=-1).astype(np.uint8)
        return Image.fromarray(smeared)

    def _apply_jpeg_artifacts(
        self, image: Image.Image, intensity: float, quality: int | None = None
    ) -> Image.Image:
        if quality is None:
            quality = int(100 * (1 - intensity))

        # Apply multiple rounds of compression for stronger artifacts
        rounds = max(1, int(intensity * 3) + 1)
        for _ in range(rounds):
            buffer = io.BytesIO()
            image.save(buffer, format="JPEG", quality=quality)
            image = Image.open(buffer)
        return image

    def _apply_water_damage(self, image: Image.Image) -> Image.Image:
        width, height = image.size
        mask = Image.new("L", (width, height), 0)
        draw = ImageDraw.Draw(mask)

        # Create irregular water stains
        num_stains = random.randint(2, 4)
        for _ in range(num_stains):
            x = random.randint(0, width)
            y = random.randint(0, height)
            for r in range(3):
                size = random.randint(50, 200)
                opacity = random.randint(30, 70)
                draw.ellipse([x - size, y - size, x + size, y + size], fill=opacity)

        # apply the mask
        water_damage = Image.new(
            "RGBA", (width, height), (200, 200, 220)
        )  # slight blue tint cuz water stains do be blue
        return Image.composite(water_damage, image, mask)

    def _apply_highlighting(self, image: Image.Image, intensity: float) -> Image.Image:
        width, height = image.size
        highlight = Image.new("RGBA", (width, height), (0, 0, 0, 0))
        draw = ImageDraw.Draw(highlight)

        num_highlights = int(intensity * 5) + 1
        highlight_color = (255, 255, 0, int(100 * intensity))

        for _ in range(num_highlights):
            x = random.randint(0, width - 200)
            y = random.randint(0, height - 30)
            rect_width = random.randint(100, 200)
            rect_height = random.randint(20, 30)
            draw.rectangle(
                [x, y, x + rect_width, y + rect_height], fill=highlight_color
            )

        return Image.alpha_composite(image.convert("RGBA"), highlight).convert("RGB")

    def _apply_crumpling(self, image: Image.Image, intensity: float) -> Image.Image:
        width, height = image.size
        points = []

        # Create distortion points
        for _ in range(int(20 * intensity)):
            x = random.randint(0, width - 1)
            y = random.randint(0, height - 1)
            dx = random.randint(-10, 10)
            dy = random.randint(-10, 10)
            points.append((x, y, dx, dy))

        # Apply distortion
        pixels = image.load()
        result = Image.new("RGB", (width, height))
        result_pixels = result.load()

        for x in range(width):
            for y in range(height):
                dx = dy = 0
                for px, py, pdx, pdy in points:
                    dist = ((x - px) ** 2 + (y - py) ** 2) ** 0.5
                    if dist < 50:
                        factor = (50 - dist) / 50.0 * intensity
                        dx += pdx * factor
                        dy += pdy * factor

                sx = int(x + dx)
                sy = int(y + dy)
                if 0 <= sx < width and 0 <= sy < height:
                    result_pixels[x, y] = pixels[sx, sy]
                else:
                    result_pixels[x, y] = pixels[x, y]

        return result

    def _apply_stain(self, image: Image.Image) -> Image.Image:
        width, height = image.size
        stain = Image.new("RGBA", (width, height), (0, 0, 0, 0))
        draw = ImageDraw.Draw(stain)

        # Create random oval stains
        for _ in range(random.randint(1, 3)):
            x = random.randint(0, width)
            y = random.randint(0, height)
            size = random.randint(50, 150)
            color = (139, 69, 19, random.randint(50, 100))  # Brown with random opacity
            draw.ellipse([x - size, y - size, x + size, y + size], fill=color)

        return Image.alpha_composite(image.convert("RGBA"), stain).convert("RGB")

    def _apply_skew(self, image: Image.Image, angle: float) -> Image.Image:
        return image.rotate(angle, expand=True, fillcolor="white")

    def _add_noise(self, image: Image.Image, noise_level: float) -> Image.Image:
        img_array = np.array(image)
        noise = np.random.normal(0, noise_level, img_array.shape)
        noisy_array = img_array + noise
        return Image.fromarray(np.clip(noisy_array, 0, 255).astype(np.uint8))


def create_enhanced_dataset(
    input_dir: Path,
    output_dir: Path,
    mode: str = "synthetic",
    split_ratio: Optional[Dict[str, float]] = None,
    max_docs: Optional[int] = None,
    cpu_fraction: float = 0.75,
    max_cpus: Optional[int] = None,
) -> None:
    """
    Create enhanced dataset with proper splits and metadata

    Args:
        input_dir: Directory containing source documents
        output_dir: Output directory for dataset
        mode: Dataset creation mode ('synthetic', 'real', 'mixed')
        split_ratio: Train/val/test split ratios
        max_docs: Maximum number of documents to process (None for all)
        cpu_fraction: Fraction of CPUs to use (0.0-1.0)
        max_cpus: Maximum number of CPUs to use (overrides cpu_fraction)
    """
    logging.basicConfig(level=logging.INFO)

    creator = DocumentDatasetCreator(
        output_dir, mode=mode, cpu_fraction=cpu_fraction, max_cpus=max_cpus
    )

    # Get document files
    doc_files = []
    for ext in ["*.jpg", "*.jpeg", "*.png", "*.tiff", "*.bmp"]:
        doc_files.extend(list(input_dir.glob(ext)))
        doc_files.extend(list(input_dir.glob(ext.upper())))

    if not doc_files:
        print(f"No documents found in {input_dir}")
        return

    # Limit number of documents if specified
    if max_docs:
        doc_files = doc_files[:max_docs]

    print(f"Found {len(doc_files)} documents to process")

    # Create dataset with splits
    metadata_dfs = creator.create_dataset(doc_files, split_ratio=split_ratio)

    # Save combined metadata
    all_metadata = pd.concat(metadata_dfs.values(), ignore_index=True)
    metadata_path = output_dir / "full_dataset_metadata.csv"
    all_metadata.to_csv(metadata_path, index=False)

    # Generate summary report
    summary_path = output_dir / "dataset_summary.txt"
    with open(summary_path, "w") as f:
        f.write("Enhanced Document Quality Dataset Summary\n")
        f.write("=" * 50 + "\n\n")
        f.write(f"Dataset Mode: {mode}\n")
        f.write(f"Total images: {len(all_metadata)}\n")
        f.write(f"Source documents: {len(doc_files)}\n\n")

        # Split information
        f.write("Dataset Splits:\n")
        for split_name, df in metadata_dfs.items():
            f.write(f"  {split_name}: {len(df)} images\n")
        f.write("\n")

        # Quality distribution
        f.write("Quality Score Distribution:\n")
        quality_ranges = [
            (0.8, 1.0, "High Quality"),
            (0.6, 0.8, "Good Quality"),
            (0.4, 0.6, "Moderate Quality"),
            (0.2, 0.4, "Poor Quality"),
            (0.0, 0.2, "Very Poor Quality"),
        ]

        for min_score, max_score, label in quality_ranges:
            count = len(
                all_metadata[
                    (all_metadata["quality_score"] >= min_score)
                    & (all_metadata["quality_score"] < max_score)
                ]
            )
            percentage = (count / len(all_metadata)) * 100
            f.write(f"  {label}: {count} images ({percentage:.1f}%)\n")

        # Quality class distribution
        f.write("\nQuality Class Distribution:\n")
        class_counts = all_metadata["quality_class"].value_counts().sort_index()
        class_names = ["High", "Good", "Moderate", "Poor", "Very Poor"]
        for class_idx, count in class_counts.items():
            percentage = (count / len(all_metadata)) * 100
            f.write(f"  {class_names[class_idx]}: {count} images ({percentage:.1f}%)\n")

        # Issue type distribution
        f.write("\nIssue Type Distribution:\n")
        issue_columns = [
            col for col in all_metadata.columns if col.startswith("issue_")
        ]
        for issue_col in issue_columns:
            issue_count = all_metadata[issue_col].sum()
            percentage = (issue_count / len(all_metadata)) * 100
            issue_name = issue_col.replace("issue_", "").replace("_", " ").title()
            f.write(f"  {issue_name}: {issue_count} images ({percentage:.1f}%)\n")

    print(f"\nDataset creation complete!")
    print(f"Metadata saved to: {metadata_path}")
    print(f"Summary saved to: {summary_path}")


class DocumentDataset(Dataset):
    """PyTorch Dataset for document quality assessment"""

    def __init__(
        self,
        image_paths: List[str] | None = None,
        metadata_file: str | None = None,
        transform: str = "train",
        target_size: Tuple[int, int] = (224, 224),
        use_advanced_augmentations: bool = True,
    ):
        """
        Initialize DocumentDataset

        Args:
            image_paths: List of image paths (if provided directly)
            metadata_file: Path to metadata CSV file
            transform: Transform type ('train', 'val', 'test')
            target_size: Target image size for model input
            use_advanced_augmentations: Whether to use advanced document-specific augmentations
        """
        if metadata_file:
            self.metadata = pd.read_csv(metadata_file)
            metadata_path = Path(metadata_file)

            # Find the dataset root directory
            # The metadata file is in datasets/default/train/train_metadata.csv
            # We need to get to datasets/ directory
            if "default" in str(metadata_path):
                # Go up 3 levels: train_metadata.csv -> train -> default -> datasets
                dataset_root = metadata_path.parent.parent.parent
            else:
                # Go up 2 levels: metadata.csv -> train -> datasets
                dataset_root = metadata_path.parent.parent

            # Check for different possible column names
            if "filepath" in self.metadata.columns:
                rel_paths = self.metadata["filepath"].tolist()
            elif "path" in self.metadata.columns:
                rel_paths = self.metadata["path"].tolist()
            else:
                raise ValueError(
                    f"Could not find 'path' or 'filepath' column in metadata. Available columns: {list(self.metadata.columns)}"
                )

            # Convert relative paths to absolute paths
            self.image_paths = [str(dataset_root / path) for path in rel_paths]
        elif image_paths:
            self.image_paths = image_paths
            self.metadata = None
        else:
            raise ValueError("Either image_paths or metadata_file must be provided")

        self.transform_type = transform
        self.use_advanced_augmentations = use_advanced_augmentations
        self.transform = self._get_transform(transform, target_size)
        self.target_size = target_size

    def _get_transform(self, transform_type: str, target_size: Tuple[int, int]):
        """Get appropriate transforms based on dataset split"""
        # Try to use advanced augmentations if available
        if self.use_advanced_augmentations:
            try:
                from src.data.augmentations import AdvancedTransforms

                return AdvancedTransforms(
                    mode=transform_type, use_domain_augmentations=True
                )
            except ImportError:
                logger.warning(
                    "Advanced augmentations not available, using standard transforms"
                )

        # Fallback to standard transforms
        if transform_type == "train":
            return transforms.Compose(
                [
                    transforms.Resize(
                        (int(target_size[0] * 1.1), int(target_size[1] * 1.1))
                    ),
                    transforms.RandomCrop(target_size),
                    transforms.RandomHorizontalFlip(p=0.3),
                    transforms.RandomRotation(degrees=5),
                    transforms.ColorJitter(
                        brightness=0.2, contrast=0.2, saturation=0.1
                    ),
                    transforms.ToTensor(),
                    transforms.Normalize(
                        mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]
                    ),
                ]
            )
        else:  # val or test
            return transforms.Compose(
                [
                    transforms.Resize(target_size),
                    transforms.ToTensor(),
                    transforms.Normalize(
                        mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]
                    ),
                ]
            )

    def get_quality_class(self, idx: int) -> int:
        """Get quality class for an index without loading the image"""
        if self.metadata is not None:
            try:
                row = self.metadata.iloc[idx]
                if "quality_class" in row:
                    return int(row["quality_class"])
                elif "quality_score" in row:
                    # Derive from quality score if class not available
                    score = float(row["quality_score"])
                    return min(int(score * 5), 4)  # Assuming 5 classes (0-4)
            except Exception:
                pass
        # Default to middle class
        return 2

    def __len__(self):
        return len(self.image_paths)

    def __getitem__(self, idx):
        # Load image
        img_path = self.image_paths[idx]
        image = Image.open(img_path).convert("RGB")

        # Get labels from metadata if available
        quality_class = None
        if self.metadata is not None:
            row = self.metadata.iloc[idx]
            quality_class = int(row["quality_class"])

        # Apply transforms - pass quality class for class-aware augmentation
        if hasattr(self.transform, "__call__") and hasattr(self.transform, "__self__"):
            # This is an AdvancedTransforms instance
            image = self.transform(image, quality_class=quality_class)
        else:
            # Standard transforms
            image = self.transform(image)

        # Get labels from metadata if available
        if self.metadata is not None:
            row = self.metadata.iloc[idx]

            # Quality score (regression target)
            quality_score = torch.tensor(row["quality_score"], dtype=torch.float32)

            # Quality class (classification target)
            quality_class = torch.tensor(row["quality_class"], dtype=torch.long)

            # Issue flags (multi-label classification)
            issue_columns = [
                col for col in self.metadata.columns if col.startswith("issue_")
            ]
            issues = torch.tensor(
                [row[col] for col in issue_columns], dtype=torch.float32
            )

            return {
                "image": image,
                "quality_score": quality_score,
                "quality_class": quality_class,
                "issues": issues,
                "path": img_path,
            }
        else:
            # Return dummy labels if no metadata
            return {
                "image": image,
                "quality_score": torch.tensor(0.5, dtype=torch.float32),
                "quality_class": torch.tensor(2, dtype=torch.long),
                "issues": torch.zeros(10, dtype=torch.float32),
                "path": img_path,
            }


def create_dataset_from_folder(
    folder_path: Path,
    train_ratio: float = 0.7,
    val_ratio: float = 0.2,
    test_ratio: float = 0.1,
    output_dir: Path | None = None,
) -> Tuple[DocumentDataset, DocumentDataset, DocumentDataset]:
    """
    Create train/val/test datasets from a folder of documents

    Args:
        folder_path: Path to folder containing documents
        train_ratio: Ratio for training set
        val_ratio: Ratio for validation set
        test_ratio: Ratio for test set
        output_dir: Optional output directory for processed dataset

    Returns:
        Tuple of (train_dataset, val_dataset, test_dataset)
    """
    folder_path = Path(folder_path)
    if output_dir is None:
        output_dir = folder_path.parent / f"{folder_path.name}_processed"
    else:
        output_dir = Path(output_dir)

    # Create dataset using DocumentDatasetCreator
    create_enhanced_dataset(
        input_dir=folder_path,
        output_dir=output_dir,
        mode="synthetic",
        split_ratio={"train": train_ratio, "val": val_ratio, "test": test_ratio},
    )

    # Load the created datasets - check for different possible locations
    train_metadata_paths = [
        output_dir / "default" / "train" / "train_metadata.csv",
        output_dir / "train" / "metadata.csv",
        output_dir / "train" / "train_metadata.csv",
    ]
    val_metadata_paths = [
        output_dir / "default" / "val" / "val_metadata.csv",
        output_dir / "val" / "metadata.csv",
        output_dir / "val" / "val_metadata.csv",
    ]
    test_metadata_paths = [
        output_dir / "default" / "test" / "test_metadata.csv",
        output_dir / "test" / "metadata.csv",
        output_dir / "test" / "test_metadata.csv",
    ]

    # Find the actual metadata files
    train_metadata = None
    val_metadata = None
    test_metadata = None

    for path in train_metadata_paths:
        if path.exists():
            train_metadata = str(path)
            break
    for path in val_metadata_paths:
        if path.exists():
            val_metadata = str(path)
            break
    for path in test_metadata_paths:
        if path.exists():
            test_metadata = str(path)
            break

    if not all([train_metadata, val_metadata, test_metadata]):
        raise ValueError(f"Could not find metadata files in {output_dir}")

    train_dataset = DocumentDataset(metadata_file=train_metadata, transform="train")

    val_dataset = DocumentDataset(metadata_file=val_metadata, transform="val")

    test_dataset = DocumentDataset(metadata_file=test_metadata, transform="test")

    return train_dataset, val_dataset, test_dataset


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Create document quality dataset")
    parser.add_argument(
        "--input-dir",
        type=str,
        default="documents",
        help="Input directory containing documents",
    )
    parser.add_argument(
        "--output-dir", type=str, default="dataset", help="Output directory for dataset"
    )
    parser.add_argument(
        "--mode",
        type=str,
        default="synthetic",
        choices=["synthetic", "real", "mixed"],
        help="Dataset creation mode",
    )
    parser.add_argument(
        "--max-docs",
        type=int,
        default=None,
        help="Maximum number of documents to process",
    )
    parser.add_argument(
        "--train-ratio", type=float, default=0.7, help="Training set ratio"
    )
    parser.add_argument(
        "--val-ratio", type=float, default=0.2, help="Validation set ratio"
    )
    parser.add_argument("--test-ratio", type=float, default=0.1, help="Test set ratio")

    args = parser.parse_args()

    split_ratio = {
        "train": args.train_ratio,
        "val": args.val_ratio,
        "test": args.test_ratio,
    }

    create_enhanced_dataset(
        Path(args.input_dir),
        Path(args.output_dir),
        mode=args.mode,
        split_ratio=split_ratio,
        max_docs=args.max_docs,
    )
