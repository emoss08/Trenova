"""
Unit tests for document quality assessment model architecture.
"""

import pytest
import torch
import torch.nn as nn

from models.model import (
    EnhancedDocumentQualityModel,
    ChannelAttention,
    SpatialAttention,
    CBAM
)


@pytest.mark.model
class TestChannelAttention:
    """Tests for Channel Attention module."""

    def test_initialization(self):
        """Test ChannelAttention initialization."""
        ca = ChannelAttention(in_channels=64, reduction=16)
        assert ca.avg_pool is not None
        assert ca.max_pool is not None
        assert isinstance(ca.fc, nn.Sequential)

    def test_forward_shape(self, device):
        """Test ChannelAttention forward pass output shape."""
        ca = ChannelAttention(in_channels=64, reduction=16).to(device)
        x = torch.randn(2, 64, 28, 28, device=device)
        out = ca(x)

        assert out.shape == (2, 64, 1, 1)

    def test_forward_values(self, device):
        """Test ChannelAttention output values are valid."""
        ca = ChannelAttention(in_channels=64, reduction=16).to(device)
        x = torch.randn(2, 64, 28, 28, device=device)
        out = ca(x)

        # Output should be sigmoid activated (between 0 and 1)
        assert torch.all(out >= 0) and torch.all(out <= 1)

    @pytest.mark.parametrize("in_channels,reduction", [
        (32, 8),
        (64, 16),
        (128, 16),
        (256, 32)
    ])
    def test_different_configurations(self, in_channels, reduction, device):
        """Test ChannelAttention with different configurations."""
        ca = ChannelAttention(in_channels=in_channels, reduction=reduction).to(device)
        x = torch.randn(1, in_channels, 14, 14, device=device)
        out = ca(x)

        assert out.shape == (1, in_channels, 1, 1)


@pytest.mark.model
class TestSpatialAttention:
    """Tests for Spatial Attention module."""

    def test_initialization(self):
        """Test SpatialAttention initialization."""
        sa = SpatialAttention(kernel_size=7)
        assert isinstance(sa.conv, nn.Conv2d)
        assert sa.conv.kernel_size == (7, 7)

    def test_forward_shape(self, device):
        """Test SpatialAttention forward pass output shape."""
        sa = SpatialAttention(kernel_size=7).to(device)
        x = torch.randn(2, 64, 28, 28, device=device)
        out = sa(x)

        assert out.shape == (2, 1, 28, 28)

    def test_forward_values(self, device):
        """Test SpatialAttention output values are valid."""
        sa = SpatialAttention(kernel_size=7).to(device)
        x = torch.randn(2, 64, 28, 28, device=device)
        out = sa(x)

        # Output should be sigmoid activated (between 0 and 1)
        assert torch.all(out >= 0) and torch.all(out <= 1)

    @pytest.mark.parametrize("kernel_size", [3, 5, 7])
    def test_different_kernel_sizes(self, kernel_size, device):
        """Test SpatialAttention with different kernel sizes."""
        sa = SpatialAttention(kernel_size=kernel_size).to(device)
        x = torch.randn(1, 64, 14, 14, device=device)
        out = sa(x)

        assert out.shape == (1, 1, 14, 14)


@pytest.mark.model
class TestCBAM:
    """Tests for CBAM (Convolutional Block Attention Module)."""

    def test_initialization(self):
        """Test CBAM initialization."""
        cbam = CBAM(in_channels=64, reduction=16, kernel_size=7)
        assert cbam.channel_attention is not None
        assert cbam.spatial_attention is not None

    def test_forward_shape(self, device):
        """Test CBAM forward pass output shape."""
        cbam = CBAM(in_channels=64, reduction=16, kernel_size=7).to(device)
        x = torch.randn(2, 64, 28, 28, device=device)
        out = cbam(x)

        assert out.shape == x.shape

    def test_attention_application(self, device):
        """Test that CBAM applies attention to input."""
        cbam = CBAM(in_channels=64, reduction=16, kernel_size=7).to(device)
        x = torch.randn(2, 64, 28, 28, device=device)
        out = cbam(x)

        # Output should be different from input (attention applied)
        assert not torch.allclose(x, out)

    def test_gradient_flow(self, device):
        """Test that gradients flow through CBAM."""
        cbam = CBAM(in_channels=64, reduction=16, kernel_size=7).to(device)
        x = torch.randn(2, 64, 28, 28, device=device, requires_grad=True)
        out = cbam(x)
        loss = out.sum()
        loss.backward()

        assert x.grad is not None
        assert not torch.all(x.grad == 0)


@pytest.mark.model
class TestEnhancedDocumentQualityModel:
    """Tests for EnhancedDocumentQualityModel."""

    def test_initialization(self):
        """Test model initialization."""
        model = EnhancedDocumentQualityModel(
            num_quality_classes=5,
            num_issues=10,
            pretrained=False
        )

        assert model.backbone is not None
        assert model.cbam is not None
        assert model.quality_score_head is not None
        assert model.quality_class_head is not None
        assert model.issue_detection_head is not None

    def test_forward_shape(self, model, sample_tensor):
        """Test model forward pass output shapes."""
        outputs = model(sample_tensor)

        assert "quality_score" in outputs
        assert "quality_class" in outputs
        assert "issue_probs" in outputs

        assert outputs["quality_score"].shape == (1, 1)
        assert outputs["quality_class"].shape == (1, 5)
        assert outputs["issue_probs"].shape == (1, 10)

    def test_forward_batch(self, model, sample_batch_tensor):
        """Test model forward pass with batch."""
        outputs = model(sample_batch_tensor)

        batch_size = sample_batch_tensor.shape[0]
        assert outputs["quality_score"].shape == (batch_size, 1)
        assert outputs["quality_class"].shape == (batch_size, 5)
        assert outputs["issue_probs"].shape == (batch_size, 10)

    def test_quality_score_range(self, model, sample_tensor):
        """Test that quality score is in valid range."""
        outputs = model(sample_tensor)
        quality_score = outputs["quality_score"]

        # Quality score should be between 0 and 5 (after sigmoid * 5)
        assert torch.all(quality_score >= 0) and torch.all(quality_score <= 5)

    def test_issue_probs_range(self, model, sample_tensor):
        """Test that issue probabilities are in valid range."""
        outputs = model(sample_tensor)
        issue_probs = outputs["issue_probs"]

        # Issue probabilities should be between 0 and 1
        assert torch.all(issue_probs >= 0) and torch.all(issue_probs <= 1)

    def test_gradient_flow(self, model, sample_tensor):
        """Test that gradients flow through the model."""
        sample_tensor.requires_grad = True
        outputs = model(sample_tensor)

        # Compute a simple loss
        loss = outputs["quality_score"].sum() + outputs["quality_class"].sum()
        loss.backward()

        # Check that gradients exist and are non-zero
        assert sample_tensor.grad is not None
        assert not torch.all(sample_tensor.grad == 0)

    def test_eval_mode(self, model, sample_tensor):
        """Test model in evaluation mode."""
        model.eval()
        with torch.no_grad():
            outputs1 = model(sample_tensor)
            outputs2 = model(sample_tensor)

        # Outputs should be deterministic in eval mode
        assert torch.allclose(outputs1["quality_score"], outputs2["quality_score"])
        assert torch.allclose(outputs1["quality_class"], outputs2["quality_class"])
        assert torch.allclose(outputs1["issue_probs"], outputs2["issue_probs"])

    def test_train_mode(self, model, sample_tensor, device):
        """Test model in training mode."""
        model.train()

        # Forward pass should work in train mode
        outputs = model(sample_tensor)
        assert outputs is not None

    @pytest.mark.parametrize("num_quality_classes,num_issues", [
        (3, 5),
        (5, 10),
        (7, 15)
    ])
    def test_different_configurations(self, num_quality_classes, num_issues, device):
        """Test model with different configurations."""
        model = EnhancedDocumentQualityModel(
            num_quality_classes=num_quality_classes,
            num_issues=num_issues,
            pretrained=False
        ).to(device)

        x = torch.randn(1, 3, 224, 224, device=device)
        outputs = model(x)

        assert outputs["quality_class"].shape == (1, num_quality_classes)
        assert outputs["issue_probs"].shape == (1, num_issues)

    def test_model_state_dict(self, model):
        """Test model state dict can be saved and loaded."""
        state_dict = model.state_dict()

        # Create new model and load state dict
        new_model = EnhancedDocumentQualityModel(
            num_quality_classes=5,
            num_issues=10,
            pretrained=False
        )
        new_model.load_state_dict(state_dict)

        # Models should have same parameters
        for (name1, param1), (name2, param2) in zip(
            model.named_parameters(),
            new_model.named_parameters()
        ):
            assert name1 == name2
            assert torch.allclose(param1, param2)

    def test_feature_extraction(self, model, sample_tensor):
        """Test that model extracts features correctly."""
        # Get features from backbone
        features = model.backbone.features(sample_tensor)

        assert features is not None
        assert features.dim() == 4  # (batch, channels, height, width)
        assert features.shape[0] == sample_tensor.shape[0]

    def test_multi_scale_fusion(self, model, sample_tensor):
        """Test multi-scale feature fusion."""
        outputs = model(sample_tensor)

        # All outputs should be computed from fused features
        assert outputs["quality_score"] is not None
        assert outputs["quality_class"] is not None
        assert outputs["issue_probs"] is not None

    def test_memory_efficiency(self, model, device):
        """Test model memory efficiency with different batch sizes."""
        batch_sizes = [1, 2, 4, 8]

        for batch_size in batch_sizes:
            x = torch.randn(batch_size, 3, 224, 224, device=device)
            outputs = model(x)

            assert outputs["quality_score"].shape[0] == batch_size
            assert outputs["quality_class"].shape[0] == batch_size
            assert outputs["issue_probs"].shape[0] == batch_size

    def test_no_nan_outputs(self, model, sample_tensor):
        """Test that model doesn't produce NaN outputs."""
        outputs = model(sample_tensor)

        assert not torch.any(torch.isnan(outputs["quality_score"]))
        assert not torch.any(torch.isnan(outputs["quality_class"]))
        assert not torch.any(torch.isnan(outputs["issue_probs"]))

    def test_model_parameters_count(self, model):
        """Test model has reasonable number of parameters."""
        total_params = sum(p.numel() for p in model.parameters())
        trainable_params = sum(p.numel() for p in model.parameters() if p.requires_grad)

        # Model should have reasonable number of parameters (< 20M)
        assert total_params < 20_000_000
        assert trainable_params > 0

    def test_checkpoint_compatibility(self, model, temp_checkpoint_dir):
        """Test model checkpoint save and load."""
        checkpoint_path = temp_checkpoint_dir / "test_model.pth"

        # Save checkpoint
        checkpoint = {
            "model_state_dict": model.state_dict(),
            "config": {
                "num_quality_classes": 5,
                "num_issues": 10
            }
        }
        torch.save(checkpoint, checkpoint_path)

        # Load checkpoint
        loaded_checkpoint = torch.load(checkpoint_path, map_location="cpu")

        # Create new model and load state
        new_model = EnhancedDocumentQualityModel(
            num_quality_classes=5,
            num_issues=10,
            pretrained=False
        )
        new_model.load_state_dict(loaded_checkpoint["model_state_dict"])

        # Verify loaded model works
        x = torch.randn(1, 3, 224, 224)
        outputs = new_model(x)
        assert outputs is not None
