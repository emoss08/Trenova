# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
import logging
import random
from typing import Dict, Optional, Tuple

import cv2
import numpy as np
import torch
import torchvision.transforms as T
from PIL import Image, ImageDraw

logger = logging.getLogger(__name__)


class TransportationDocumentAugmentation:
    """Domain-specific augmentations for transportation documents"""

    def __init__(self, severity_range: Tuple[float, float] = (0.1, 0.5)):
        self.severity_range = severity_range

        # Common transportation document issues
        self.stamp_positions = [(0.7, 0.1), (0.8, 0.2), (0.1, 0.8), (0.2, 0.9)]
        self.watermark_texts = ["COPY", "DRAFT", "VOID", "DUPLICATE", "SAMPLE"]

    def add_stamp_overlay(self, image: Image.Image) -> Image.Image:
        """Add realistic stamp overlays common in shipping documents"""
        if random.random() > 0.3:  # 70% chance
            return image

        img = image.copy()
        draw = ImageDraw.Draw(img)

        # Random stamp position
        pos = random.choice(self.stamp_positions)
        x = int(pos[0] * img.width)
        y = int(pos[1] * img.height)

        # Create circular stamp effect
        stamp_size = random.randint(80, 150)
        stamp_color = random.choice(
            [(255, 0, 0, 128), (0, 0, 255, 128), (0, 128, 0, 128)]
        )

        # Draw stamp circle
        draw.ellipse(
            [
                x - stamp_size // 2,
                y - stamp_size // 2,
                x + stamp_size // 2,
                y + stamp_size // 2,
            ],
            outline=stamp_color[:3],
            width=3,
        )

        # Add stamp text
        stamp_text = random.choice(["RECEIVED", "PROCESSED", "VERIFIED", "APPROVED"])

        # Simple text without font (using default)
        text_bbox = draw.textbbox((x, y), stamp_text)
        text_width = text_bbox[2] - text_bbox[0]
        text_height = text_bbox[3] - text_bbox[1]
        draw.text(
            (x - text_width // 2, y - text_height // 2),
            stamp_text,
            fill=stamp_color[:3],
        )

        return img

    def add_barcode_blur(self, image: Image.Image) -> Image.Image:
        """Simulate barcode scanning issues"""
        if random.random() > 0.2:  # 80% chance
            return image

        img_array = np.array(image)
        h, w = img_array.shape[:2]

        # Typical barcode locations
        barcode_regions = [
            (0.6, 0.05, 0.35, 0.15),  # Top right
            (0.05, 0.05, 0.35, 0.15),  # Top left
            (0.3, 0.85, 0.4, 0.1),  # Bottom center
        ]

        region = random.choice(barcode_regions)
        x1 = int(region[0] * w)
        y1 = int(region[1] * h)
        x2 = x1 + int(region[2] * w)
        y2 = y1 + int(region[3] * h)

        # Apply motion blur to barcode region
        kernel_size = random.randint(5, 15)
        kernel = np.zeros((kernel_size, kernel_size))
        kernel[kernel_size // 2, :] = 1
        kernel = kernel / kernel_size

        roi = img_array[y1:y2, x1:x2]
        if roi.size > 0:
            blurred = cv2.filter2D(roi, -1, kernel)
            img_array[y1:y2, x1:x2] = blurred

        return Image.fromarray(img_array)

    def add_fold_marks(self, image: Image.Image) -> Image.Image:
        """Add fold marks common in folded shipping documents"""
        if random.random() > 0.25:  # 75% chance
            return image

        img = image.copy()
        img_array = np.array(img)
        h, w = img_array.shape[:2]

        # Common fold patterns
        fold_patterns = [
            [(0, h // 3), (w, h // 3)],  # Horizontal third
            [(0, 2 * h // 3), (w, 2 * h // 3)],  # Horizontal two-thirds
            [(w // 2, 0), (w // 2, h)],  # Vertical center
        ]

        num_folds = random.randint(1, 2)
        selected_folds = random.sample(fold_patterns, num_folds)

        for fold in selected_folds:
            # Create fold shadow
            thickness = random.randint(2, 5)
            darkness = random.uniform(0.7, 0.9)

            cv2.line(
                img_array,
                fold[0],
                fold[1],
                (int(255 * darkness), int(255 * darkness), int(255 * darkness)),
                thickness,
            )

        return Image.fromarray(img_array)

    def add_watermark(self, image: Image.Image) -> Image.Image:
        """Add watermarks typical in document copies"""
        if random.random() > 0.2:  # 80% chance
            return image

        img = image.copy().convert("RGBA")
        txt_layer = Image.new("RGBA", img.size, (255, 255, 255, 0))
        draw = ImageDraw.Draw(txt_layer)

        text = random.choice(self.watermark_texts)

        # Calculate text position (diagonal across image)
        angle = random.randint(-45, -30)

        # Create rotated text
        font_size = min(img.width, img.height) // 10

        # Draw text multiple times for watermark effect
        for i in range(3):
            for j in range(3):
                x = img.width * (i + 0.5) / 3
                y = img.height * (j + 0.5) / 3
                draw.text((x, y), text, fill=(128, 128, 128, 64))

        # Rotate the text layer
        txt_layer = txt_layer.rotate(angle, expand=1)

        # After rotation, the text layer might have different dimensions
        # Resize txt_layer to match img size or crop/pad as needed
        if txt_layer.size != img.size:
            # Create a new layer with the original image size
            new_txt_layer = Image.new("RGBA", img.size, (255, 255, 255, 0))

            # Calculate position to center the rotated text
            x_offset = (img.width - txt_layer.width) // 2
            y_offset = (img.height - txt_layer.height) // 2

            # Handle cases where rotated text is larger than original
            if x_offset < 0 or y_offset < 0:
                # Crop the rotated text to fit
                left = max(0, -x_offset)
                top = max(0, -y_offset)
                right = left + img.width
                bottom = top + img.height
                txt_layer = txt_layer.crop((left, top, right, bottom))
                x_offset = max(0, x_offset)
                y_offset = max(0, y_offset)

            # Paste the rotated text onto the new layer
            new_txt_layer.paste(txt_layer, (x_offset, y_offset))
            txt_layer = new_txt_layer

        # Composite the watermark
        img = Image.alpha_composite(img, txt_layer)

        return img.convert("RGB")

    def add_coffee_stain(self, image: Image.Image) -> Image.Image:
        """Add coffee stain effect common on desk documents"""
        if random.random() > 0.05:  # 95% chance (rare)
            return image

        img = image.copy()
        img_array = np.array(img)
        h, w = img_array.shape[:2]

        # Random stain position
        cx = random.randint(w // 4, 3 * w // 4)
        cy = random.randint(h // 4, 3 * h // 4)

        # Create stain mask
        radius = random.randint(30, 80)
        Y, X = np.ogrid[:h, :w]
        dist = np.sqrt((X - cx) ** 2 + (Y - cy) ** 2)

        # Irregular stain shape
        noise = np.random.randn(h, w) * 10
        mask = (dist + noise) <= radius

        # Apply brown tint
        stain_color = np.array([139, 90, 43])  # Brown
        alpha = 0.3

        for c in range(3):
            img_array[:, :, c][mask] = (
                alpha * stain_color[c] + (1 - alpha) * img_array[:, :, c][mask]
            ).astype(np.uint8)

        return Image.fromarray(img_array)

    def __call__(self, image: Image.Image) -> Image.Image:
        """Apply random transportation document augmentations"""
        augmentations = [
            self.add_stamp_overlay,
            self.add_barcode_blur,
            self.add_fold_marks,
            self.add_watermark,
            self.add_coffee_stain,
        ]

        # Apply 1-3 random augmentations
        num_augs = random.randint(1, 3)
        selected_augs = random.sample(augmentations, num_augs)

        for aug in selected_augs:
            image = aug(image)

        return image


class MixupAugmentation:
    """Mixup augmentation for better generalization"""

    def __init__(self, alpha: float = 0.2):
        self.alpha = alpha

    def __call__(
        self, images: torch.Tensor, targets: Dict[str, torch.Tensor]
    ) -> Tuple[torch.Tensor, Dict[str, torch.Tensor], float]:
        """Apply mixup to a batch of images and targets"""
        batch_size = images.size(0)

        if self.alpha > 0:
            lam = np.random.beta(self.alpha, self.alpha)
        else:
            lam = 1

        index = torch.randperm(batch_size)

        # Mix images
        mixed_images = lam * images + (1 - lam) * images[index]

        # Mix targets
        mixed_targets = {}
        for key, value in targets.items():
            if key in ["quality_score", "quality_scores"]:
                # For regression targets, interpolate
                mixed_targets[key] = lam * value + (1 - lam) * value[index]
            elif key == "quality_class":
                # For classification, keep the dominant class
                mixed_targets[key] = value if lam > 0.5 else value[index]
            elif key == "issues":
                # For multi-label, use OR operation
                mixed_targets[key] = torch.max(value, value[index])

        return mixed_images, mixed_targets, lam


class CutMixAugmentation:
    """CutMix augmentation for document robustness"""

    def __init__(self, alpha: float = 1.0):
        self.alpha = alpha

    def __call__(
        self, images: torch.Tensor, targets: Dict[str, torch.Tensor]
    ) -> Tuple[torch.Tensor, Dict[str, torch.Tensor], float]:
        """Apply CutMix augmentation"""
        batch_size, _, h, w = images.size()

        if self.alpha > 0:
            lam = np.random.beta(self.alpha, self.alpha)
        else:
            lam = 1

        index = torch.randperm(batch_size)

        # Generate random box
        bbx1, bby1, bbx2, bby2 = self.rand_bbox(images.size(), lam)

        # Apply CutMix
        images[:, :, bbx1:bbx2, bby1:bby2] = images[index, :, bbx1:bbx2, bby1:bby2]

        # Adjust lambda based on actual box area
        lam = 1 - ((bbx2 - bbx1) * (bby2 - bby1) / (h * w))

        # Mix targets proportionally
        mixed_targets = {}
        for key, value in targets.items():
            if key in ["quality_score", "quality_scores"]:
                mixed_targets[key] = lam * value + (1 - lam) * value[index]
            elif key == "quality_class":
                mixed_targets[key] = value if lam > 0.5 else value[index]
            elif key == "issues":
                mixed_targets[key] = torch.max(value, value[index])

        return images, mixed_targets, lam

    def rand_bbox(self, size, lam):
        """Generate random bounding box for CutMix"""
        _, _, h, w = size
        cut_rat = np.sqrt(1.0 - lam)
        cut_w = np.int32(w * cut_rat)
        cut_h = np.int32(h * cut_rat)

        # Uniform
        cx = np.random.randint(w)
        cy = np.random.randint(h)

        bbx1 = np.clip(cx - cut_w // 2, 0, w)
        bby1 = np.clip(cy - cut_h // 2, 0, h)
        bbx2 = np.clip(cx + cut_w // 2, 0, w)
        bby2 = np.clip(cy + cut_h // 2, 0, h)

        return bbx1, bby1, bbx2, bby2


class ClassAwareAugmentation:
    """Apply different augmentation strategies based on quality class"""

    def __init__(self):
        # Define augmentation strategies for different quality levels
        self.poor_quality_augs = T.Compose(
            [
                T.RandomAdjustSharpness(sharpness_factor=2.0, p=0.5),  # Try to enhance
                T.RandomAutocontrast(p=0.3),
                T.ColorJitter(brightness=0.2, contrast=0.3),
            ]
        )

        self.good_quality_augs = T.Compose(
            [
                T.GaussianBlur(kernel_size=3, sigma=(0.1, 0.5)),  # Add slight blur
                T.ColorJitter(brightness=0.1, contrast=0.1, saturation=0.1),
                T.RandomGrayscale(p=0.1),
            ]
        )

    def __call__(self, image: Image.Image, quality_class: int) -> Image.Image:
        """Apply augmentation based on quality class"""
        if quality_class <= 1:  # Very Poor or Poor
            return self.poor_quality_augs(image)
        elif quality_class >= 3:  # Good or High
            return self.good_quality_augs(image)
        else:  # Moderate
            # Apply balanced augmentation
            if random.random() > 0.5:
                return self.poor_quality_augs(image)
            else:
                return self.good_quality_augs(image)


class AdvancedTransforms:
    """Advanced transformation pipeline with domain-specific augmentations"""

    def __init__(
        self,
        mode: str = "train",
        use_domain_augmentations: bool = True,
        use_class_aware_aug: bool = True,
    ):
        self.mode = mode
        self.use_domain_augmentations = use_domain_augmentations
        self.use_class_aware_aug = use_class_aware_aug
        self.domain_augmentation: Optional[TransportationDocumentAugmentation] = None
        self.class_aware_aug: Optional[ClassAwareAugmentation] = None

        # Basic transforms
        if mode == "train":
            self.basic_transforms = T.Compose(
                [
                    T.Resize((256, 256)),
                    T.RandomCrop(224),
                    T.RandomHorizontalFlip(p=0.5),
                    T.RandomRotation(degrees=10),
                    T.ColorJitter(
                        brightness=0.3, contrast=0.3, saturation=0.2, hue=0.1
                    ),
                    T.RandomPerspective(distortion_scale=0.2, p=0.3),
                    T.RandomAffine(degrees=0, translate=(0.1, 0.1), scale=(0.9, 1.1)),
                    T.RandomGrayscale(p=0.1),  # Some documents are grayscale
                    T.RandomAdjustSharpness(sharpness_factor=2, p=0.2),
                ]
            )
        else:
            self.basic_transforms = T.Compose(
                [
                    T.Resize((224, 224)),
                ]
            )

        # Domain-specific augmentations
        if use_domain_augmentations and mode == "train":
            self.domain_augmentation = TransportationDocumentAugmentation()
        else:
            self.domain_augmentation = None

        # Class-aware augmentations
        if use_class_aware_aug and mode == "train":
            self.class_aware_aug = ClassAwareAugmentation()
        else:
            self.class_aware_aug = None

        # Normalization
        self.normalize = T.Compose(
            [
                T.ToTensor(),
                T.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),
            ]
        )

    def __call__(
        self, image: Image.Image, quality_class: Optional[int] = None
    ) -> torch.Tensor:
        """Apply transformations to image

        Args:
            image: Input PIL image
            quality_class: Optional quality class for class-aware augmentation
        """
        try:
            # Apply domain-specific augmentations first
            if self.domain_augmentation:
                try:
                    image = self.domain_augmentation(image)
                except Exception as e:
                    logger.warning(f"Domain augmentation failed: {e}. Skipping.")

            # Apply class-aware augmentations if quality class is provided
            if self.class_aware_aug and quality_class is not None:
                try:
                    image = self.class_aware_aug(image, quality_class)
                except Exception as e:
                    logger.warning(f"Class-aware augmentation failed: {e}. Skipping.")

            # Apply basic transforms
            image = self.basic_transforms(image)

            # Convert to tensor and normalize
            return self.normalize(image)

        except Exception as e:
            logger.error(
                f"Transform failed: {e}. Returning original image with basic transforms."
            )
            # Fallback: just resize and normalize the original image
            try:
                image = T.Resize((224, 224))(image)
                return self.normalize(image)
            except Exception as e2:
                logger.error(f"Even basic transform failed: {e2}")
                raise
