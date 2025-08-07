# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
import logging
import os
from dataclasses import dataclass
from typing import Dict, List, Optional

import numpy as np
import pandas as pd
import torch
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim
from PIL import Image
from torch.utils.data import DataLoader, Dataset, Sampler
from torchvision import models, transforms

logger = logging.getLogger(__name__)


class AdaptiveBatchNorm1d(nn.Module):
    """BatchNorm that handles batch size 1 gracefully"""

    def __init__(self, num_features):
        super().__init__()
        self.bn = nn.BatchNorm1d(num_features)

    def forward(self, x):
        if self.training and x.size(0) == 1:
            # Use eval mode for single sample during training
            self.bn.eval()
            output = self.bn(x)
            self.bn.train()
            return output
        return self.bn(x)


@dataclass
class ModelConfig:
    """Configuration for the document quality model"""

    backbone: str = "efficientnet_b0"  # More modern than MobileNet
    num_quality_classes: int = 5  # Very Poor, Poor, Moderate, Good, High
    num_issue_classes: int = 10  # Different types of quality issues
    hidden_dim: int = 256
    dropout_rate: float = 0.3
    use_attention: bool = True
    freeze_backbone_layers: int = 0  # Number of layers to freeze from start


# ==================== Dataset Class ====================
class DocumentQualityDataset(Dataset):
    def __init__(self, csv_file, root_dir, transform=None, return_metadata=False):
        self.metadata = pd.read_csv(csv_file)
        self.root_dir = root_dir
        self.transform = transform
        self.return_metadata = return_metadata

        # Add support for multi-task labels if available
        self.has_quality_classes = "quality_class" in self.metadata.columns
        self.has_issue_labels = any("issue_" in col for col in self.metadata.columns)

    def __len__(self):
        return len(self.metadata)

    def __getitem__(self, idx):
        img_path = os.path.join(self.root_dir, self.metadata.iloc[idx, 0])
        image = Image.open(img_path).convert("RGB")

        # Get quality score (regression target)
        quality_score = self.metadata.iloc[idx, 1]

        # Prepare targets dict
        targets = {"quality_scores": torch.tensor(quality_score, dtype=torch.float32)}

        # Add quality class if available
        if self.has_quality_classes:
            quality_class = self.metadata.iloc[idx]["quality_class"]
            targets["quality_classes"] = torch.tensor(quality_class, dtype=torch.long)

        # Add issue labels if available
        if self.has_issue_labels:
            issue_cols = [col for col in self.metadata.columns if "issue_" in col]
            issue_labels = self.metadata.iloc[idx][issue_cols].values.astype(np.float32)
            targets["issue_labels"] = torch.tensor(issue_labels, dtype=torch.float32)

        if self.transform:
            image = self.transform(image)

        if self.return_metadata:
            metadata = {"filepath": self.metadata.iloc[idx, 0], "index": idx}
            return image, targets, metadata

        return image, targets


class EnhancedShadowDetection(nn.Module):
    """Enhanced module for detecting shadows and lighting issues in documents"""

    def __init__(self, in_channels: int, out_channels: int):
        super().__init__()
        self.conv1 = nn.Conv2d(in_channels, out_channels // 2, 3, padding=1)
        self.conv2 = nn.Conv2d(in_channels, out_channels // 2, 5, padding=2)
        self.bn = nn.BatchNorm2d(out_channels)
        self.relu = nn.ReLU(inplace=True)

        # Learnable threshold for shadow detection
        self.shadow_threshold = nn.Parameter(torch.tensor(0.3))

    def forward(self, x):
        # Multi-scale shadow detection
        shadow_feat1 = self.conv1(x)
        shadow_feat2 = self.conv2(x)

        # Combine features
        shadow_features = torch.cat([shadow_feat1, shadow_feat2], dim=1)
        shadow_features = self.relu(self.bn(shadow_features))

        # Create shadow mask
        shadow_mask = torch.sigmoid(
            (shadow_features.mean(dim=1, keepdim=True) - self.shadow_threshold) * 10
        )

        return shadow_features, shadow_mask


class AdvancedShadowDetection(nn.Module):
    """Specialized module for improved shadow detection with global context"""

    def __init__(self, in_channels: int = 3):
        super().__init__()
        self.shadow_conv = nn.Sequential(
            # Initial feature extraction
            nn.Conv2d(1, 32, 3, padding=1),
            nn.BatchNorm2d(32),
            nn.ReLU(inplace=True),
            # Edge-aware processing with dilation
            nn.Conv2d(32, 32, 3, padding=2, dilation=2),
            nn.BatchNorm2d(32),
            nn.ReLU(inplace=True),
            # Gradient detection
            nn.Conv2d(32, 16, 5, padding=2),
            nn.BatchNorm2d(16),
            nn.ReLU(inplace=True),
            # Residual connection
            nn.Conv2d(16, 16, 3, padding=1),
            nn.BatchNorm2d(16),
            nn.ReLU(inplace=True),
        )

        # Global context
        self.global_context = nn.Sequential(
            nn.AdaptiveAvgPool2d(1),
            nn.Conv2d(16, 8, 1),
            nn.ReLU(inplace=True),
            nn.Conv2d(8, 16, 1),
            nn.Sigmoid(),
        )

        # Final prediction
        self.predictor = nn.Sequential(
            nn.Conv2d(16, 8, 1), nn.ReLU(inplace=True), nn.Conv2d(8, 1, 1)
        )

    def forward(self, x):
        # Convert to grayscale for shadow detection
        if x.size(1) == 3:
            gray = 0.299 * x[:, 0] + 0.587 * x[:, 1] + 0.114 * x[:, 2]
            gray = gray.unsqueeze(1)
        else:
            gray = x

        # Extract shadow features
        features = self.shadow_conv(gray)

        # Apply global context
        global_weight = self.global_context(features)
        features = features * global_weight

        # Generate shadow map
        shadow_map = torch.sigmoid(self.predictor(features))

        # Global shadow score
        shadow_score = F.adaptive_avg_pool2d(shadow_map, 1).squeeze(-1).squeeze(-1)

        return shadow_score, shadow_map


class MultiScaleFeatureFusion(nn.Module):
    """Multi-scale feature fusion for robust document analysis"""

    def __init__(
        self, in_channels: int, out_channels: int, scales: List[int] = [1, 2, 4]
    ):
        super().__init__()
        self.scales = scales
        self.in_channels = in_channels
        self.out_channels = out_channels

        # Calculate channels for each scale to ensure proper concatenation
        # Each scale will produce a specific number of channels
        channels_per_scale = self._calculate_channels_per_scale(
            out_channels, len(scales)
        )

        self.scale_convs = nn.ModuleList(
            [
                nn.Conv2d(in_channels, channels, 3, padding=1)
                for channels in channels_per_scale
            ]
        )

        # Fusion conv expects the sum of all scale channels
        total_channels = sum(channels_per_scale)
        self.fusion_conv = nn.Conv2d(total_channels, out_channels, 1)
        self.bn = nn.BatchNorm2d(out_channels)
        self.relu = nn.ReLU(inplace=True)

    def _calculate_channels_per_scale(
        self, total_channels: int, num_scales: int
    ) -> List[int]:
        """Calculate channel distribution across scales to match total channels exactly"""
        base_channels = total_channels // num_scales
        remainder = total_channels % num_scales

        channels_per_scale = [base_channels] * num_scales
        # Distribute remainder channels to first scales
        for i in range(remainder):
            channels_per_scale[i] += 1

        return channels_per_scale

    def forward(self, x):
        multi_scale_features = []

        # Check if spatial dimensions are too small for multi-scale processing
        h, w = x.shape[2:]
        use_multiscale = h > 1 and w > 1

        for scale, conv in zip(self.scales, self.scale_convs):
            if scale > 1 and use_multiscale:
                # Ensure we don't downsample below 1x1
                effective_scale = min(scale, min(h, w))
                if effective_scale > 1:
                    # Downsample
                    scaled_x = F.avg_pool2d(x, effective_scale)
                    feat = conv(scaled_x)
                    # Upsample back
                    feat = F.interpolate(
                        feat, size=x.shape[2:], mode="bilinear", align_corners=False
                    )
                else:
                    # If can't downsample, just apply conv
                    feat = conv(x)
            else:
                feat = conv(x)
            multi_scale_features.append(feat)

        # Concatenate and fuse
        fused = torch.cat(multi_scale_features, dim=1)
        fused = self.fusion_conv(fused)
        return self.relu(self.bn(fused))


class DocumentSpecificFeatures(nn.Module):
    """Extract document-specific features optimized for quality assessment"""

    def __init__(self, in_channels: int, out_channels: int):
        super().__init__()
        # Calculate channels for each feature to ensure exact match
        # For 64 channels: 11, 11, 11, 11, 10, 10 = 64
        base_channels = out_channels // 6
        remainder = out_channels % 6

        channels = [base_channels] * 6
        for i in range(remainder):
            channels[i] += 1

        # Multi-scale edge detection for sharpness
        self.edge_conv1 = nn.Conv2d(in_channels, channels[0], kernel_size=3, padding=1)
        self.edge_conv2 = nn.Conv2d(in_channels, channels[1], kernel_size=5, padding=2)

        # Text region detection with different receptive fields
        self.text_conv1 = nn.Conv2d(in_channels, channels[2], kernel_size=5, padding=2)
        self.text_conv2 = nn.Conv2d(in_channels, channels[3], kernel_size=7, padding=3)

        # Blur/focus detection
        self.blur_conv = nn.Conv2d(in_channels, channels[4], kernel_size=7, padding=3)

        # Noise pattern detection
        self.noise_conv = nn.Conv2d(in_channels, channels[5], kernel_size=3, padding=1)

        self.bn = nn.BatchNorm2d(out_channels)
        self.relu = nn.ReLU(inplace=True)

        # Learnable importance weights
        self.feature_weights = nn.Parameter(torch.ones(6) / 6)

        # Channel attention for feature calibration
        self.channel_attention = nn.Sequential(
            nn.AdaptiveAvgPool2d(1),
            nn.Conv2d(out_channels, out_channels // 4, 1),
            nn.ReLU(inplace=True),
            nn.Conv2d(out_channels // 4, out_channels, 1),
            nn.Sigmoid(),
        )

    def forward(self, x):
        # Extract different feature types
        edge_feat1 = self.edge_conv1(x)
        edge_feat2 = self.edge_conv2(x)
        text_feat1 = self.text_conv1(x)
        text_feat2 = self.text_conv2(x)
        blur_feat = self.blur_conv(x)
        noise_feat = self.noise_conv(x)

        # Apply weighted combination
        weights = F.softmax(self.feature_weights, dim=0)

        # Concatenate all features
        features = torch.cat(
            [
                edge_feat1 * weights[0],
                edge_feat2 * weights[1],
                text_feat1 * weights[2],
                text_feat2 * weights[3],
                blur_feat * weights[4],
                noise_feat * weights[5],
            ],
            dim=1,
        )

        # Apply batch norm and activation
        features = self.relu(self.bn(features))

        # Apply channel attention
        attn_weights = self.channel_attention(features)
        features = features * attn_weights

        return features


class ChannelAttention(nn.Module):
    """Channel attention module using squeeze-and-excitation"""

    def __init__(self, in_channels: int, reduction: int = 16):
        super().__init__()
        self.avg_pool = nn.AdaptiveAvgPool2d(1)
        self.max_pool = nn.AdaptiveMaxPool2d(1)
        self.fc1 = nn.Conv2d(in_channels, in_channels // reduction, 1, bias=False)
        self.relu = nn.ReLU()
        self.fc2 = nn.Conv2d(in_channels // reduction, in_channels, 1, bias=False)
        self.sigmoid = nn.Sigmoid()

    def forward(self, x):
        # Average pool path
        avg_out = self.fc2(self.relu(self.fc1(self.avg_pool(x))))
        # Max pool path
        max_out = self.fc2(self.relu(self.fc1(self.max_pool(x))))
        # Combine and activate
        channel_attn = self.sigmoid(avg_out + max_out)
        return x * channel_attn


class SpatialAttention(nn.Module):
    """Spatial attention module to focus on important regions"""

    def __init__(self, in_channels: int):
        super().__init__()
        self.conv = nn.Conv2d(in_channels, 1, kernel_size=7, padding=3)
        self.sigmoid = nn.Sigmoid()

    def forward(self, x):
        attn = self.sigmoid(self.conv(x))
        return x * attn, attn


class CBAM(nn.Module):
    """Convolutional Block Attention Module"""

    def __init__(self, in_channels: int, reduction: int = 16):
        super().__init__()
        self.channel_attention = ChannelAttention(in_channels, reduction)
        self.spatial_attention = SpatialAttention(in_channels)

    def forward(self, x):
        x = self.channel_attention(x)
        x, attn_map = self.spatial_attention(x)
        return x, attn_map


class DocumentQualityModel(nn.Module):
    """Enhanced model for document quality assessment with multi-task learning"""

    def __init__(self, config: ModelConfig):
        super().__init__()
        self.config = config

        # Load backbone
        if config.backbone == "efficientnet_b0":
            self.backbone = models.efficientnet_b0(
                weights=models.EfficientNet_B0_Weights.DEFAULT
            )
            backbone_out_channels = 1280
            # Remove the final classification layer
            self.backbone.classifier = nn.Identity()
        elif config.backbone == "resnet50":
            self.backbone = models.resnet50(weights=models.ResNet50_Weights.DEFAULT)
            backbone_out_channels = 2048
            # Remove the final FC layer
            self.backbone.fc = nn.Identity()
        else:
            # Default to MobileNetV3
            self.backbone = models.mobilenet_v3_large(
                weights=models.MobileNet_V3_Large_Weights.DEFAULT
            )
            backbone_out_channels = 960
            self.backbone.classifier = nn.Identity()

        # Freeze early layers if specified
        if config.freeze_backbone_layers > 0:
            self._freeze_backbone_layers(config.freeze_backbone_layers)

        # Document-specific feature extractor
        self.doc_features = DocumentSpecificFeatures(3, 60)  # 60 is divisible by 6

        # Enhanced shadow detection
        self.shadow_detection = EnhancedShadowDetection(3, 32)

        # Multi-scale feature fusion - ensure output matches backbone channels
        self.multi_scale_fusion = MultiScaleFeatureFusion(
            backbone_out_channels, backbone_out_channels
        )

        # Attention module - use CBAM for better performance
        if config.use_attention:
            self.attention = CBAM(backbone_out_channels)

        # Task-specific heads
        self.quality_head = nn.Sequential(
            nn.Linear(backbone_out_channels, config.hidden_dim),
            nn.ReLU(),
            nn.Dropout(config.dropout_rate),
            nn.Linear(config.hidden_dim, config.hidden_dim // 2),
            nn.ReLU(),
            nn.Dropout(config.dropout_rate),
            nn.Linear(config.hidden_dim // 2, 1),  # Regression output for quality score
        )

        self.classification_head = nn.Sequential(
            nn.Linear(backbone_out_channels, config.hidden_dim),
            nn.ReLU(),
            nn.Dropout(config.dropout_rate),
            nn.Linear(config.hidden_dim, config.num_quality_classes),
        )

        self.issue_detection_head = nn.Sequential(
            nn.Linear(backbone_out_channels, config.hidden_dim),
            nn.ReLU(),
            nn.Dropout(config.dropout_rate),
            nn.Linear(config.hidden_dim, config.num_issue_classes),
        )

        # Enhanced feature fusion with document-specific features and shadow features
        fusion_input_dim = (
            backbone_out_channels + 60 + 32
        )  # backbone + doc_features + shadow
        self.feature_fusion = nn.Sequential(
            nn.Linear(fusion_input_dim, backbone_out_channels),
            AdaptiveBatchNorm1d(backbone_out_channels),
            nn.ReLU(),
            nn.Dropout(config.dropout_rate),
            nn.Linear(backbone_out_channels, backbone_out_channels),
            AdaptiveBatchNorm1d(backbone_out_channels),
            nn.ReLU(),
        )

        # Global quality embedding
        self.quality_embedding = nn.Sequential(
            nn.Linear(backbone_out_channels, 128), nn.ReLU(), nn.Linear(128, 64)
        )

        # Initialize heads properly to prevent collapse
        self._initialize_heads()

    def _initialize_heads(self):
        """Initialize network heads to prevent collapse"""
        # Initialize linear layers with smaller weights
        for module in [
            self.quality_head,
            self.classification_head,
            self.issue_detection_head,
        ]:
            for layer in module:
                if isinstance(layer, nn.Linear):
                    nn.init.xavier_normal_(layer.weight, gain=0.5)
                    if layer.bias is not None:
                        nn.init.constant_(layer.bias, 0.0)

        # Initialize final regression layer with very small weights
        # to ensure outputs start in a reasonable range
        final_layer = self.quality_head[-1]
        nn.init.uniform_(final_layer.weight, -0.01, 0.01)
        nn.init.constant_(final_layer.bias, 0.5)  # Start predictions around 0.5

    def _freeze_backbone_layers(self, num_layers: int):
        """Freeze the first num_layers of the backbone"""
        params = list(self.backbone.parameters())
        for param in params[:num_layers]:
            param.requires_grad = False

    def extract_features(self, x):
        """Extract features from backbone"""
        if hasattr(self.backbone, "features"):
            # For EfficientNet and MobileNet
            features = self.backbone.features(x)
            # Global average pooling
            features = F.adaptive_avg_pool2d(features, 1)
            features = features.flatten(1)
        else:
            # For ResNet
            features = self.backbone(x)

        return features

    def forward(self, x, return_features=False):
        batch_size = x.size(0)

        # Extract document-specific features from input
        doc_feat = self.doc_features(x)
        doc_feat_pooled = F.adaptive_avg_pool2d(doc_feat, 1).flatten(1)

        # Extract shadow features
        shadow_feat, shadow_mask = self.shadow_detection(x)
        shadow_feat_pooled = F.adaptive_avg_pool2d(shadow_feat, 1).flatten(1)

        # Extract backbone features
        if hasattr(self.backbone, "features"):
            # For EfficientNet and MobileNet
            backbone_feat_maps = self.backbone.features(x)
            # Apply multi-scale fusion before pooling
            backbone_feat_maps = self.multi_scale_fusion(backbone_feat_maps)
            # Apply attention if enabled
            if self.config.use_attention:
                backbone_feat_maps, attn_map = self.attention(backbone_feat_maps)
            # Global average pooling
            backbone_features = F.adaptive_avg_pool2d(backbone_feat_maps, 1).flatten(1)
        else:
            # For ResNet
            backbone_features = self.backbone(x)
            # ResNet already produces a 1D feature vector, no need for multi-scale fusion

        # Combine all features
        combined_features = torch.cat(
            [backbone_features, doc_feat_pooled, shadow_feat_pooled], dim=1
        )
        fused_features = self.feature_fusion(combined_features)

        # Store embeddings for contrastive loss
        embeddings = self.quality_embedding(fused_features)

        # Multi-task outputs
        quality_score = torch.sigmoid(self.quality_head(fused_features))  # 0-1 range
        quality_class = self.classification_head(fused_features)
        issue_logits = self.issue_detection_head(fused_features)

        outputs = {
            "quality_score": quality_score.squeeze(-1),  # Only squeeze last dimension
            "quality_class_logits": quality_class,
            "issue_logits": issue_logits,
            "issues": issue_logits,  # Alias for compatibility
            "embeddings": embeddings,
            "shadow_mask": shadow_mask if return_features else None,
        }

        # Remove None values
        outputs = {k: v for k, v in outputs.items() if v is not None}

        return outputs


# ==================== Model Definition ====================
def create_model(config: Optional[ModelConfig] = None):
    """Create an enhanced document quality model"""
    if config is None:
        # For backward compatibility, create simple model
        model = models.mobilenet_v3_large(
            weights=models.MobileNet_V3_Large_Weights.DEFAULT
        )
        model.classifier[-1] = nn.Linear(model.classifier[-1].in_features, 1)
        return model

    return DocumentQualityModel(config)


class EnsembleDocumentQualityModel(nn.Module):
    """Ensemble model that combines multiple backbones for robust predictions"""

    def __init__(self, configs: List[ModelConfig]):
        super().__init__()
        self.models = nn.ModuleList(
            [DocumentQualityModel(config) for config in configs]
        )
        self.ensemble_weights = nn.Parameter(torch.ones(len(configs)) / len(configs))

    def forward(self, x):
        outputs = []
        for model in self.models:
            outputs.append(model(x))

        # Weighted ensemble averaging
        weights = F.softmax(self.ensemble_weights, dim=0)

        # Combine outputs
        ensemble_output = {}
        for key in outputs[0].keys():
            if key == "attention_map":
                continue  # Skip attention maps

            stacked = torch.stack([out[key] for out in outputs])

            # Apply weighted average
            if stacked.dim() == 3:  # [num_models, batch, features]
                weighted = (stacked * weights.view(-1, 1, 1)).sum(dim=0)
            elif stacked.dim() == 2:  # [num_models, batch]
                weighted = (stacked * weights.view(-1, 1)).sum(dim=0)
            else:
                weighted = (stacked * weights.view(-1)).sum(dim=0)

            ensemble_output[key] = weighted

        return ensemble_output


class OrdinalRegressionLoss(nn.Module):
    """Ordinal regression loss for quality classification"""

    def __init__(self, num_classes: int):
        super().__init__()
        self.num_classes = num_classes

    def forward(self, logits: torch.Tensor, targets: torch.Tensor) -> torch.Tensor:
        """Calculate ordinal regression loss

        Args:
            logits: Class logits [B, num_classes]
            targets: Target classes [B]
        """
        batch_size = logits.size(0)

        # Create ordinal targets
        ordinal_targets = torch.zeros(
            batch_size, self.num_classes - 1, device=logits.device
        )
        for i in range(batch_size):
            ordinal_targets[i, : targets[i]] = 1

        # Convert logits to ordinal predictions
        ordinal_logits = logits[:, :-1] - logits[:, 1:]

        # Binary cross entropy for each threshold
        loss = F.binary_cross_entropy_with_logits(ordinal_logits, ordinal_targets)

        return loss


class ConsistencyLoss(nn.Module):
    """Consistency loss between quality score and class predictions"""

    def __init__(self, num_classes: int = 5, temperature: float = 3.0):
        super().__init__()
        self.num_classes = num_classes
        self.temperature = temperature

    def forward(
        self, quality_score: torch.Tensor, class_logits: torch.Tensor
    ) -> torch.Tensor:
        """Calculate consistency loss between continuous score and discrete classes

        Args:
            quality_score: Continuous quality scores [B] in range [0, 1]
            class_logits: Class logits [B, num_classes]
        """
        # Convert quality score to expected class distribution
        # Score 0-0.2 -> class 0, 0.2-0.4 -> class 1, etc.
        expected_class = (quality_score * self.num_classes).clamp(
            0, self.num_classes - 1
        )

        # Create soft targets based on proximity to class boundaries
        soft_targets = torch.zeros_like(class_logits)
        for i in range(len(quality_score)):
            score = quality_score[i].item()
            class_idx = int(expected_class[i].item())

            # Set main class probability
            soft_targets[i, class_idx] = 1.0

            # Add soft probabilities to adjacent classes based on distance
            class_center = (class_idx + 0.5) / self.num_classes
            distance = abs(score - class_center) * self.num_classes

            if class_idx > 0 and score < class_center:
                soft_targets[i, class_idx - 1] = distance
                soft_targets[i, class_idx] = 1 - distance
            elif class_idx < self.num_classes - 1 and score > class_center:
                soft_targets[i, class_idx + 1] = distance
                soft_targets[i, class_idx] = 1 - distance

        # Apply temperature scaling
        soft_targets = F.softmax(soft_targets / self.temperature, dim=-1)
        class_probs = F.log_softmax(class_logits / self.temperature, dim=-1)

        # KL divergence loss
        loss = F.kl_div(class_probs, soft_targets, reduction="batchmean")

        return loss * self.temperature * self.temperature


class MultiTaskLoss(nn.Module):
    """Combined loss for multi-task learning with dynamic task weighting"""

    def __init__(
        self,
        regression_weight: float = 1.0,
        classification_weight: float = 0.5,
        issue_weight: float = 0.5,
        consistency_weight: float = 0.3,
        use_focal_loss: bool = True,
        use_uncertainty_weighting: bool = True,
        use_dynamic_weighting: bool = False,
        use_ordinal_regression: bool = True,
        num_quality_classes: int = 5,
    ):
        super().__init__()
        self.regression_weight = regression_weight
        self.classification_weight = classification_weight
        self.issue_weight = issue_weight
        self.consistency_weight = consistency_weight
        self.use_uncertainty_weighting = use_uncertainty_weighting
        self.use_dynamic_weighting = use_dynamic_weighting
        self.use_ordinal_regression = use_ordinal_regression

        # Loss functions
        self.mse_loss = nn.MSELoss()
        self.smooth_l1_loss = nn.SmoothL1Loss()
        self.quality_aware_loss = QualityAwareLoss(
            critical_threshold=0.5, penalty_factor=2.0
        )

        # Classification loss
        if use_ordinal_regression:
            self.classification_loss = OrdinalRegressionLoss(num_quality_classes)
        else:
            self.ce_loss = nn.CrossEntropyLoss(label_smoothing=0.1)

        # Consistency loss
        self.consistency_loss = ConsistencyLoss(num_quality_classes)

        if use_focal_loss:
            self.issue_loss = FocalLoss(alpha=0.25, gamma=2.0)
        else:
            self.issue_loss = nn.BCEWithLogitsLoss()

        # Learnable task uncertainty parameters (log variance)
        if use_uncertainty_weighting:
            self.log_vars = nn.Parameter(torch.zeros(4))  # Added one for consistency

        # Dynamic task weighting
        if use_dynamic_weighting:
            self.task_loss_history = {
                "regression": [],
                "classification": [],
                "issues": [],
                "consistency": [],
            }
            self.dynamic_weights = {
                "regression": regression_weight,
                "classification": classification_weight,
                "issues": issue_weight,
                "consistency": consistency_weight,
            }

    def forward(self, predictions: Dict, targets: Dict) -> Dict[str, torch.Tensor]:
        losses = {}

        # Regression loss for quality score - handle both key names
        if "quality_score" in targets:
            # Use Quality-Aware Loss for better handling of critical boundaries
            regression_loss = self.quality_aware_loss(
                predictions["quality_score"], targets["quality_score"]
            )
            losses["regression"] = regression_loss
        elif "quality_scores" in targets:
            regression_loss = self.quality_aware_loss(
                predictions["quality_score"], targets["quality_scores"]
            )
            losses["regression"] = regression_loss

        # Classification loss for quality categories
        if "quality_class" in targets:
            if self.use_ordinal_regression:
                classification_loss = self.classification_loss(
                    predictions["quality_class_logits"], targets["quality_class"]
                )
            else:
                classification_loss = self.ce_loss(
                    predictions["quality_class_logits"], targets["quality_class"]
                )
            losses["classification"] = classification_loss
        elif "quality_classes" in targets:
            if self.use_ordinal_regression:
                classification_loss = self.classification_loss(
                    predictions["quality_class_logits"], targets["quality_classes"]
                )
            else:
                classification_loss = self.ce_loss(
                    predictions["quality_class_logits"], targets["quality_classes"]
                )
            losses["classification"] = classification_loss

        # Consistency loss between score and class predictions
        if "quality_score" in predictions and "quality_class_logits" in predictions:
            consistency_loss = self.consistency_loss(
                predictions["quality_score"], predictions["quality_class_logits"]
            )
            losses["consistency"] = consistency_loss

        # Multi-label classification for issues
        if "issues" in targets:
            issue_loss = self.issue_loss(predictions["issue_logits"], targets["issues"])
            losses["issues"] = issue_loss
        elif "issue_labels" in targets:
            issue_loss = self.issue_loss(
                predictions["issue_logits"], targets["issue_labels"]
            )
            losses["issues"] = issue_loss

        # Uncertainty-weighted loss combination
        if self.use_uncertainty_weighting and len(losses) > 0:
            total_loss = 0
            loss_keys = ["regression", "classification", "issues", "consistency"]

            for i, key in enumerate(loss_keys):
                if key in losses:
                    precision = torch.exp(-self.log_vars[i])
                    total_loss += precision * losses[key] + self.log_vars[i]

            losses["total"] = total_loss
        else:
            # Use dynamic weights if enabled, otherwise use static weights
            if self.use_dynamic_weighting:
                total_loss = (
                    self.dynamic_weights.get("regression", self.regression_weight)
                    * losses.get("regression", 0)
                    + self.dynamic_weights.get(
                        "classification", self.classification_weight
                    )
                    * losses.get("classification", 0)
                    + self.dynamic_weights.get("issues", self.issue_weight)
                    * losses.get("issues", 0)
                    + self.dynamic_weights.get("consistency", self.consistency_weight)
                    * losses.get("consistency", 0)
                )
            else:
                total_loss = (
                    self.regression_weight * losses.get("regression", 0)
                    + self.classification_weight * losses.get("classification", 0)
                    + self.issue_weight * losses.get("issues", 0)
                    + self.consistency_weight * losses.get("consistency", 0)
                )
            losses["total"] = total_loss

        return losses

    def update_dynamic_weights(self, window_size: int = 10):
        """Update dynamic task weights based on recent loss history"""
        if not self.use_dynamic_weighting:
            return

        # Calculate average recent losses for each task
        avg_losses = {}
        for task in self.task_loss_history:
            if len(self.task_loss_history[task]) >= window_size:
                recent_losses = self.task_loss_history[task][-window_size:]
                avg_losses[task] = np.mean(recent_losses)

        # Update weights inversely proportional to loss progress
        if len(avg_losses) > 0:
            # Normalize by the maximum average loss
            max_loss = max(avg_losses.values())
            if max_loss > 0:
                for task in avg_losses:
                    # Tasks with higher losses get higher weights
                    self.dynamic_weights[task] = (avg_losses[task] / max_loss) * 2.0

        logger.info(f"Updated dynamic weights: {self.dynamic_weights}")

    def record_losses(self, losses: Dict[str, torch.Tensor]):
        """Record loss values for dynamic weighting"""
        if self.use_dynamic_weighting:
            for task in ["regression", "classification", "issues", "consistency"]:
                if task in losses:
                    self.task_loss_history[task].append(losses[task].item())


class FocalLoss(nn.Module):
    """Focal loss for addressing class imbalance"""

    def __init__(self, alpha: float = 0.25, gamma: float = 2.0):
        super().__init__()
        self.alpha = alpha
        self.gamma = gamma

    def forward(self, inputs: torch.Tensor, targets: torch.Tensor) -> torch.Tensor:
        bce_loss = F.binary_cross_entropy_with_logits(inputs, targets, reduction="none")
        pt = torch.exp(-bce_loss)
        focal_loss = self.alpha * (1 - pt) ** self.gamma * bce_loss
        return focal_loss.mean()


class QualityAwareLoss(nn.Module):
    """Custom loss that emphasizes critical quality boundaries"""

    def __init__(self, critical_threshold: float = 0.5, penalty_factor: float = 2.0):
        super().__init__()
        self.critical_threshold = critical_threshold
        self.penalty_factor = penalty_factor
        self.base_loss = nn.SmoothL1Loss(reduction="none")

    def forward(self, predictions: torch.Tensor, targets: torch.Tensor) -> torch.Tensor:
        # Base loss
        base_loss = self.base_loss(predictions, targets)

        # Additional penalty for misclassifying good documents as bad or vice versa
        is_good_doc = targets >= self.critical_threshold
        pred_good_doc = predictions >= self.critical_threshold

        # Penalty when predictions cross the critical threshold incorrectly
        critical_errors = (is_good_doc != pred_good_doc).float()

        # Scale penalty based on how far the prediction is from the target
        error_magnitude = torch.abs(predictions - targets)

        # Apply heavier penalty for critical errors
        weighted_loss = base_loss * (
            1 + critical_errors * self.penalty_factor * error_magnitude
        )

        return weighted_loss.mean()


class ContrastiveLoss(nn.Module):
    """Contrastive loss to ensure quality embeddings are well-separated"""

    def __init__(self, margin: float = 1.0, temperature: float = 0.07):
        super().__init__()
        self.margin = margin
        self.temperature = temperature

    def forward(
        self, embeddings: torch.Tensor, quality_scores: torch.Tensor
    ) -> torch.Tensor:
        batch_size = embeddings.size(0)

        # Normalize embeddings
        embeddings = F.normalize(embeddings, dim=1)

        # Compute pairwise similarity
        sim_matrix = torch.matmul(embeddings, embeddings.T) / self.temperature

        # Create quality difference matrix
        quality_diff = torch.abs(
            quality_scores.unsqueeze(1) - quality_scores.unsqueeze(0)
        )

        # Similar documents (small quality difference) should have high similarity
        # Different documents (large quality difference) should have low similarity
        positive_mask = quality_diff < 0.1  # Similar quality
        negative_mask = quality_diff > 0.3  # Different quality

        # Contrastive loss
        positive_loss = (
            -torch.log(torch.sigmoid(sim_matrix[positive_mask])).mean()
            if positive_mask.any()
            else 0
        )
        negative_loss = (
            -torch.log(1 - torch.sigmoid(sim_matrix[negative_mask])).mean()
            if negative_mask.any()
            else 0
        )

        return positive_loss + negative_loss


# ==================== Training Function ====================
def train_epoch(
    model,
    train_loader,
    optimizer,
    device,
    regression_weight=1.0,
    classification_weight=0.5,
    issue_weight=0.5,
    use_focal_loss=True,
):
    """
    Train the model for one epoch

    Args:
        model: The model to train
        train_loader: DataLoader for training data
        optimizer: Optimizer
        device: Device to run on
        regression_weight: Weight for regression loss
        classification_weight: Weight for classification loss
        issue_weight: Weight for issue detection loss
        use_focal_loss: Whether to use focal loss for issues

    Returns:
        Average training loss for the epoch
    """
    model.train()
    total_loss = 0.0
    num_batches = 0

    # Setup loss function
    criterion = MultiTaskLoss(
        regression_weight=regression_weight,
        classification_weight=classification_weight,
        issue_weight=issue_weight,
        use_focal_loss=use_focal_loss,
    )

    for batch in train_loader:
        # Move data to device
        images = batch["image"].to(device)
        targets = {
            "quality_score": batch["quality_score"].to(device),
            "quality_class": batch["quality_class"].to(device),
            "issues": batch["issues"].to(device),
        }

        # Zero gradients
        optimizer.zero_grad()

        # Forward pass
        outputs = model(images)

        # Calculate loss
        loss_dict = criterion(outputs, targets)
        loss = loss_dict["total"]

        # Backward pass
        loss.backward()

        # Gradient clipping
        torch.nn.utils.clip_grad_norm_(model.parameters(), max_norm=1.0)

        # Update weights
        optimizer.step()

        # Track loss
        total_loss += loss.item()
        num_batches += 1

    return total_loss / num_batches


def train_model(
    model,
    dataloaders,
    criterion,
    optimizer,
    num_epochs=10,
    patience=3,
    device=None,
    scheduler=None,
):
    best_model_wts = model.state_dict()
    best_loss = float("inf")
    epochs_no_improve = 0

    # Check if using enhanced model
    is_enhanced = isinstance(model, DocumentQualityModel)

    for epoch in range(num_epochs):
        print(f"Epoch {epoch + 1}/{num_epochs}")
        print("-" * 10)

        for phase in ["train", "val"]:
            if phase == "train":
                model.train()
            else:
                model.eval()

            running_loss = 0.0
            running_losses = {
                "total": 0.0,
                "regression": 0.0,
                "classification": 0.0,
                "issues": 0.0,
            }

            for batch_data in dataloaders[phase]:
                # Handle both old and new dataset formats
                if len(batch_data) == 2:
                    inputs, targets = batch_data
                else:
                    inputs, targets, metadata = batch_data

                inputs = inputs.to(device)

                # Handle both single tensor and dict targets
                if isinstance(targets, torch.Tensor):
                    targets = {"quality_scores": targets.to(device)}
                else:
                    targets = {
                        k: v.to(device) if isinstance(v, torch.Tensor) else v
                        for k, v in targets.items()
                    }

                optimizer.zero_grad()

                with torch.set_grad_enabled(phase == "train"):
                    if is_enhanced:
                        outputs = model(inputs)
                        if isinstance(criterion, MultiTaskLoss):
                            losses = criterion(outputs, targets)
                            loss = losses["total"]

                            # Track individual losses
                            for k, v in losses.items():
                                if k in running_losses:
                                    running_losses[k] += v.item() * inputs.size(0)
                        else:
                            # Backward compatibility
                            loss = criterion(
                                outputs["quality_score"], targets["quality_scores"]
                            )
                    else:
                        # Original model
                        outputs = model(inputs).squeeze()
                        loss = criterion(outputs, targets["quality_scores"])

                    if phase == "train":
                        loss.backward()
                        # Gradient clipping
                        torch.nn.utils.clip_grad_norm_(model.parameters(), max_norm=1.0)
                        optimizer.step()

                running_loss += loss.item() * inputs.size(0)

            epoch_loss = running_loss / len(dataloaders[phase].dataset)

            # Print detailed losses if available
            if is_enhanced and isinstance(criterion, MultiTaskLoss):
                loss_str = f"{phase} Loss: {epoch_loss:.4f}"
                for k, v in running_losses.items():
                    if k != "total" and v > 0:
                        avg_loss = v / len(dataloaders[phase].dataset)
                        loss_str += f", {k}: {avg_loss:.4f}"
                print(loss_str)
            else:
                print(f"{phase} Loss: {epoch_loss:.4f}")

            if phase == "val":
                if scheduler is not None:
                    if isinstance(
                        scheduler, torch.optim.lr_scheduler.ReduceLROnPlateau
                    ):
                        scheduler.step(epoch_loss)
                    else:
                        scheduler.step()

                if epoch_loss < best_loss:
                    best_loss = epoch_loss
                    best_model_wts = model.state_dict()
                    epochs_no_improve = 0
                else:
                    epochs_no_improve += 1

                if epochs_no_improve >= patience:
                    print("Early stopping triggered")
                    model.load_state_dict(best_model_wts)
                    return model

    print("Training complete.")
    model.load_state_dict(best_model_wts)
    return model


# ==================== Expanded Evaluation Metrics ====================
def evaluate_model(model, dataloader, device, criterion=None):
    model.eval()
    total_loss = 0.0
    total_samples = 0
    all_preds = []
    all_labels = []
    all_class_preds = []
    all_class_labels = []

    # Check if using enhanced model
    is_enhanced = isinstance(model, DocumentQualityModel)

    with torch.no_grad():
        for batch_data in dataloader:
            # Handle both old and new dataset formats
            if len(batch_data) == 2:
                inputs, targets = batch_data
            else:
                inputs, targets, metadata = batch_data

            inputs = inputs.to(device)

            # Handle both single tensor and dict targets
            if isinstance(targets, torch.Tensor):
                targets = {"quality_scores": targets.to(device)}
            else:
                targets = {
                    k: v.to(device) if isinstance(v, torch.Tensor) else v
                    for k, v in targets.items()
                }

            if is_enhanced:
                outputs = model(inputs)
                scores = outputs["quality_score"]

                # Calculate loss if criterion provided
                if criterion and isinstance(criterion, MultiTaskLoss):
                    losses = criterion(outputs, targets)
                    loss = losses["total"]
                else:
                    loss = nn.MSELoss()(scores, targets["quality_scores"])

                # Store class predictions if available
                if "quality_classes" in targets:
                    class_preds = torch.argmax(outputs["quality_class_logits"], dim=1)
                    all_class_preds.extend(class_preds.cpu().numpy())
                    all_class_labels.extend(targets["quality_classes"].cpu().numpy())
            else:
                # Original model
                outputs = model(inputs).squeeze()
                scores = outputs
                loss = nn.MSELoss()(outputs, targets["quality_scores"])

            total_loss += loss.item() * inputs.size(0)
            total_samples += inputs.size(0)
            all_preds.extend(scores.cpu().numpy())
            all_labels.extend(targets["quality_scores"].cpu().numpy())

    # Calculate metrics
    mae = sum(abs(p - l) for p, l in zip(all_preds, all_labels)) / total_samples
    rmse = np.sqrt(np.mean((np.array(all_preds) - np.array(all_labels)) ** 2))

    print(f"Test Loss: {total_loss / total_samples:.4f}")
    print(f"Test MAE: {mae:.4f}")
    print(f"Test RMSE: {rmse:.4f}")

    # Calculate classification metrics if available
    if all_class_preds:
        from sklearn.metrics import accuracy_score, f1_score

        accuracy = accuracy_score(all_class_labels, all_class_preds)
        f1 = f1_score(all_class_labels, all_class_preds, average="weighted")
        print(f"Classification Accuracy: {accuracy:.4f}")
        print(f"Classification F1 Score: {f1:.4f}")

    return {
        "loss": total_loss / total_samples,
        "mae": mae,
        "rmse": rmse,
        "predictions": all_preds,
        "targets": all_labels,
    }


# ==================== Main Script ====================
if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Train Document Quality Model")
    parser.add_argument(
        "--enhanced",
        action="store_true",
        help="Use enhanced model with multi-task learning",
    )
    parser.add_argument(
        "--backbone",
        type=str,
        default="efficientnet_b0",
        choices=["efficientnet_b0", "resnet50", "mobilenet_v3"],
        help="Backbone architecture",
    )
    parser.add_argument("--batch-size", type=int, default=32, help="Batch size")
    parser.add_argument("--epochs", type=int, default=10, help="Number of epochs")
    parser.add_argument("--lr", type=float, default=0.001, help="Learning rate")
    parser.add_argument(
        "--patience", type=int, default=3, help="Early stopping patience"
    )
    parser.add_argument(
        "--dataset-dir", type=str, default="dataset", help="Dataset directory"
    )

    args = parser.parse_args()

    # Paths
    dataset_dir = args.dataset_dir
    dataset_csv = f"{dataset_dir}/dataset_metadata.csv"

    # Parameters
    batch_size = args.batch_size
    num_epochs = args.epochs
    learning_rate = args.lr
    patience = args.patience

    if not torch.cuda.is_available():
        print("CUDA is not available. Using CPU instead.")
        device = torch.device("cpu")
    else:
        device = torch.device("cuda")
        print(f"Using device: {torch.cuda.current_device()}")

    # Enhanced transforms for better augmentation
    if args.enhanced:
        data_transforms = {
            "train": transforms.Compose(
                [
                    transforms.Resize((256, 256)),
                    transforms.RandomCrop(224),
                    transforms.RandomHorizontalFlip(),
                    transforms.RandomRotation(5),
                    transforms.ColorJitter(
                        brightness=0.2, contrast=0.2, saturation=0.1
                    ),
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
    else:
        # Original transforms
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

    # Dataset and Dataloader
    datasets = {
        "train": DocumentQualityDataset(
            dataset_csv, dataset_dir, transform=data_transforms["train"]
        ),
        "val": DocumentQualityDataset(
            dataset_csv, dataset_dir, transform=data_transforms["val"]
        ),
    }

    dataloaders = {
        "train": DataLoader(
            datasets["train"],
            batch_size=batch_size,
            shuffle=True,
            num_workers=4,
            pin_memory=True,
        ),
        "val": DataLoader(
            datasets["val"],
            batch_size=batch_size,
            shuffle=False,
            num_workers=4,
            pin_memory=True,
        ),
    }

    # Model creation
    if args.enhanced:
        print(f"Using enhanced model with {args.backbone} backbone")
        config = ModelConfig(
            backbone=args.backbone, dropout_rate=0.3, use_attention=True
        )
        model = create_model(config).to(device)

        # Different learning rates for backbone and heads
        backbone_params = []
        head_params = []

        for name, param in model.named_parameters():
            if "backbone" in name:
                backbone_params.append(param)
            else:
                head_params.append(param)

        optimizer = optim.AdamW(
            [
                {"params": backbone_params, "lr": learning_rate * 0.1},
                {"params": head_params, "lr": learning_rate},
            ],
            weight_decay=0.01,
        )

        # Use multi-task loss
        criterion = MultiTaskLoss(
            regression_weight=1.0, classification_weight=0.5, issue_weight=0.5
        )

        # Learning rate scheduler
        scheduler = optim.lr_scheduler.CosineAnnealingLR(
            optimizer, T_max=num_epochs, eta_min=1e-6
        )
    else:
        print("Using original model")
        model = create_model().to(device)
        criterion = nn.MSELoss()
        optimizer = optim.Adam(model.parameters(), lr=learning_rate)
        scheduler = None

    # Train
    model = train_model(
        model,
        dataloaders,
        criterion,
        optimizer,
        num_epochs=num_epochs,
        patience=patience,
        device=device,
        scheduler=scheduler,
    )

    # Save the model
    model_name = (
        f"document_quality_model_{'enhanced' if args.enhanced else 'original'}.pth"
    )
    torch.save(
        {
            "model_state_dict": model.state_dict(),
            "config": config.__dict__ if args.enhanced else None,
            "enhanced": args.enhanced,
        },
        model_name,
    )
    print(f"Model saved as {model_name}")

    # Evaluate
    test_loader = DataLoader(datasets["val"], batch_size=batch_size, shuffle=False)
    evaluate_model(model, test_loader, device, criterion)


class BalancedBatchSampler(Sampler):
    """Sampler that ensures balanced class distribution in each batch"""

    def __init__(self, dataset, batch_size: int, num_classes: int = 5):
        self.dataset = dataset
        self.batch_size = batch_size
        self.num_classes = num_classes
        self.epoch = 0

        # Group indices by class
        self._organize_by_class()

    def _organize_by_class(self):
        """Organize dataset indices by class without loading images"""
        self.class_indices = [[] for _ in range(self.num_classes)]

        # Try to use get_quality_class method if available
        if hasattr(self.dataset, "get_quality_class"):
            # Use the dataset's method to get quality classes without loading images
            for idx in range(len(self.dataset)):
                try:
                    class_idx = self.dataset.get_quality_class(idx)
                    # Ensure class_idx is within valid range
                    class_idx = max(0, min(class_idx, self.num_classes - 1))
                    self.class_indices[class_idx].append(idx)
                except Exception as e:
                    logger.warning(f"Error getting class for index {idx}: {e}")
                    # Add to middle class as fallback
                    self.class_indices[self.num_classes // 2].append(idx)
        else:
            # Fallback: If no metadata, distribute indices evenly across classes
            # This avoids loading images during initialization
            logger.warning(
                "No metadata available for BalancedBatchSampler. Distributing indices evenly."
            )
            for idx in range(len(self.dataset)):
                class_idx = idx % self.num_classes
                self.class_indices[class_idx].append(idx)

        # Log class distribution
        for i, indices in enumerate(self.class_indices):
            logger.info(f"Class {i}: {len(indices)} samples")

    def set_epoch(self, epoch: int):
        """Set epoch for reproducibility"""
        self.epoch = epoch

    def __iter__(self):
        # Set random seed based on epoch
        np.random.seed(self.epoch)

        # Create a copy of indices for sampling
        available_indices = [indices.copy() for indices in self.class_indices]

        # Shuffle each class's indices
        for indices in available_indices:
            np.random.shuffle(indices)

        batches = []

        # Create balanced batches
        while sum(len(indices) for indices in available_indices) >= self.batch_size:
            batch = []

            # Calculate samples per class for this batch
            samples_per_class = self.batch_size // self.num_classes
            extra_samples = self.batch_size % self.num_classes

            for class_idx in range(self.num_classes):
                if len(available_indices[class_idx]) > 0:
                    # Determine number of samples from this class
                    n_samples = samples_per_class
                    if class_idx < extra_samples:
                        n_samples += 1

                    # Don't exceed available samples
                    n_samples = min(n_samples, len(available_indices[class_idx]))

                    # Add samples to batch
                    batch.extend(available_indices[class_idx][:n_samples])

                    # Remove used samples
                    available_indices[class_idx] = available_indices[class_idx][
                        n_samples:
                    ]

            if len(batch) >= self.batch_size * 0.8:  # Allow slightly smaller last batch
                batches.append(batch)  # Append as a batch, not extend

        return iter(batches)

    def __len__(self):
        return len(self.dataset)


def create_advanced_model(config: Optional[ModelConfig] = None) -> DocumentQualityModel:
    """Create an advanced document quality model with all improvements"""
    if config is None:
        config = ModelConfig(
            backbone="efficientnet_b0",
            num_quality_classes=5,
            num_issue_classes=10,
            hidden_dim=256,
            dropout_rate=0.3,
            use_attention=True,
            freeze_backbone_layers=0,
        )

    # Create the model with a flag to use advanced shadow detection
    model = DocumentQualityModel(config)

    # Replace the shadow detection with the advanced version
    model.shadow_detection = AdvancedShadowDetection(3)

    # Update the feature fusion to account for different shadow output
    # Advanced shadow detection outputs just shadow_score (scalar) instead of features
    backbone_channels = (
        1280
        if config.backbone == "efficientnet_b0"
        else 2048
        if config.backbone == "resnet50"
        else 960
    )
    fusion_input_dim = (
        backbone_channels + 60 + 1
    )  # backbone + doc_features + shadow_score

    model.feature_fusion = nn.Sequential(
        nn.Linear(fusion_input_dim, backbone_channels),
        AdaptiveBatchNorm1d(backbone_channels),
        nn.ReLU(),
        nn.Dropout(config.dropout_rate),
        nn.Linear(backbone_channels, backbone_channels),
        AdaptiveBatchNorm1d(backbone_channels),
        nn.ReLU(),
    )

    # Override the forward method to use advanced shadow detection
    original_forward = model.forward

    def advanced_forward(self, x, return_features=False):
        batch_size = x.size(0)

        # Extract document-specific features
        doc_feat = model.doc_features(x)
        doc_feat_pooled = F.adaptive_avg_pool2d(doc_feat, 1).flatten(1)

        # Extract shadow score using advanced detection
        shadow_score, shadow_map = model.shadow_detection(x)

        # Extract backbone features
        if hasattr(model.backbone, "features"):
            backbone_feat_maps = model.backbone.features(x)
            backbone_feat_maps = model.multi_scale_fusion(backbone_feat_maps)
            if model.config.use_attention:
                backbone_feat_maps, attn_map = model.attention(backbone_feat_maps)
            backbone_features = F.adaptive_avg_pool2d(backbone_feat_maps, 1).flatten(1)
        else:
            backbone_features = model.backbone(x)
            # ResNet already produces a 1D feature vector, no multi-scale fusion needed

        # Combine all features - shadow_score is already a scalar per batch
        combined_features = torch.cat(
            [
                backbone_features,
                doc_feat_pooled,
                shadow_score.unsqueeze(1) if shadow_score.dim() == 1 else shadow_score,
            ],
            dim=1,
        )

        fused_features = model.feature_fusion(combined_features)

        # Store embeddings
        embeddings = model.quality_embedding(fused_features)

        # Multi-task outputs
        quality_score = torch.sigmoid(model.quality_head(fused_features))
        quality_class = model.classification_head(fused_features)
        issue_logits = model.issue_detection_head(fused_features)

        # Add shadow score to shadow-related issue if applicable
        if issue_logits.size(1) >= 6:  # Assuming shadow is the 6th issue
            issue_logits[:, 5] = issue_logits[:, 5] + shadow_score * 2.0

        outputs = {
            "quality_score": quality_score.squeeze(-1),
            "quality_class_logits": quality_class,
            "issue_logits": issue_logits,
            "issues": issue_logits,
            "embeddings": embeddings,
            "shadow_mask": shadow_map if return_features else None,
        }

        outputs = {k: v for k, v in outputs.items() if v is not None}

        return outputs

    # Replace the forward method
    model.forward = advanced_forward

    return model
