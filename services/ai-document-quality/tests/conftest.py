"""
Pytest configuration and fixtures for document quality assessment tests.
"""

import os
import sys
from pathlib import Path
from typing import Dict, Any

import pytest
import torch
import numpy as np
from PIL import Image
from fastapi.testclient import TestClient

# Add src to path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from models.model import EnhancedDocumentQualityModel
from api.app import app
from api.inference import DocumentQualityPredictor


@pytest.fixture(scope="session")
def device():
    """Fixture for compute device (CPU for testing)."""
    return torch.device("cpu")


@pytest.fixture(scope="session")
def sample_image():
    """Create a sample RGB image for testing."""
    # Create 224x224 RGB image with random noise
    arr = np.random.randint(0, 255, (224, 224, 3), dtype=np.uint8)
    return Image.fromarray(arr, mode="RGB")


@pytest.fixture(scope="session")
def sample_image_batch():
    """Create a batch of sample images for testing."""
    images = []
    for _ in range(4):
        arr = np.random.randint(0, 255, (224, 224, 3), dtype=np.uint8)
        images.append(Image.fromarray(arr, mode="RGB"))
    return images


@pytest.fixture(scope="session")
def sample_tensor(device):
    """Create a sample tensor for testing."""
    return torch.randn(1, 3, 224, 224, device=device)


@pytest.fixture(scope="session")
def sample_batch_tensor(device):
    """Create a batch of sample tensors for testing."""
    return torch.randn(4, 3, 224, 224, device=device)


@pytest.fixture(scope="session")
def model(device):
    """Create a model instance for testing."""
    model = EnhancedDocumentQualityModel(
        num_quality_classes=5,
        num_issues=10,
        pretrained=False  # Don't download pretrained weights for tests
    )
    model.to(device)
    model.eval()
    return model


@pytest.fixture(scope="session")
def model_state_dict(model):
    """Get model state dict for checkpoint testing."""
    return model.state_dict()


@pytest.fixture(scope="function")
def temp_checkpoint_dir(tmp_path):
    """Create a temporary directory for checkpoint files."""
    checkpoint_dir = tmp_path / "checkpoints"
    checkpoint_dir.mkdir()
    return checkpoint_dir


@pytest.fixture(scope="function")
def temp_output_dir(tmp_path):
    """Create a temporary directory for output files."""
    output_dir = tmp_path / "outputs"
    output_dir.mkdir()
    return output_dir


@pytest.fixture(scope="session")
def sample_checkpoint(model, tmp_path_factory):
    """Create a sample checkpoint file for testing."""
    checkpoint_path = tmp_path_factory.mktemp("checkpoints") / "test_checkpoint.pth"

    checkpoint = {
        "epoch": 10,
        "model_state_dict": model.state_dict(),
        "optimizer_state_dict": {},
        "scheduler_state_dict": {},
        "best_val_loss": 0.5,
        "config": {
            "num_quality_classes": 5,
            "num_issues": 10,
            "learning_rate": 0.001,
            "batch_size": 32
        },
        "metrics": {
            "train_loss": 0.6,
            "val_loss": 0.5,
            "val_mae": 0.3,
            "val_accuracy": 0.85
        }
    }

    torch.save(checkpoint, checkpoint_path)
    return checkpoint_path


@pytest.fixture(scope="session")
def predictor(sample_checkpoint, device):
    """Create a DocumentQualityPredictor instance for testing."""
    predictor = DocumentQualityPredictor(
        model_path=str(sample_checkpoint),
        device=device
    )
    return predictor


@pytest.fixture(scope="session")
def api_client():
    """Create a FastAPI test client."""
    return TestClient(app)


@pytest.fixture(scope="function")
def mock_image_bytes(sample_image):
    """Create mock image bytes for API testing."""
    from io import BytesIO
    buffer = BytesIO()
    sample_image.save(buffer, format="JPEG")
    buffer.seek(0)
    return buffer.getvalue()


@pytest.fixture(scope="session")
def sample_model_output():
    """Create sample model output for testing."""
    return {
        "quality_score": torch.tensor([3.5]),
        "quality_class": torch.tensor([2]),  # Moderate
        "quality_probs": torch.softmax(torch.randn(1, 5), dim=1),
        "issue_probs": torch.sigmoid(torch.randn(1, 10))
    }


@pytest.fixture(scope="session")
def sample_quality_labels():
    """Sample quality class labels."""
    return {
        0: "High",
        1: "Good",
        2: "Moderate",
        3: "Poor",
        4: "Very Poor"
    }


@pytest.fixture(scope="session")
def sample_issue_labels():
    """Sample issue type labels."""
    return {
        0: "Blur",
        1: "Noise",
        2: "Lighting",
        3: "Shadow",
        4: "Physical Damage",
        5: "Skew",
        6: "Partial",
        7: "Glare",
        8: "Compression",
        9: "Overall Poor"
    }


@pytest.fixture(scope="function")
def mock_dataset_metadata():
    """Create mock dataset metadata for testing."""
    return {
        "dataset_mode": "synthetic",
        "total_images": 1100,
        "source_documents": 100,
        "splits": {
            "train": 770,
            "val": 220,
            "test": 110
        },
        "quality_distribution": {
            "High": 200,
            "Good": 72,
            "Moderate": 98,
            "Poor": 314,
            "Very Poor": 416
        }
    }


@pytest.fixture(scope="session")
def sample_metrics():
    """Sample evaluation metrics for testing."""
    return {
        "regression": {
            "mae": 0.35,
            "rmse": 0.52,
            "r2": 0.78,
            "mape": 12.5
        },
        "classification": {
            "accuracy": 0.82,
            "balanced_accuracy": 0.76,
            "f1_macro": 0.74,
            "precision_macro": 0.75,
            "recall_macro": 0.73
        },
        "calibration": {
            "ece": 0.08,
            "mce": 0.15,
            "brier_score": 0.18
        },
        "issue_detection": {
            "f1_micro": 0.71,
            "f1_macro": 0.68,
            "precision_macro": 0.70,
            "recall_macro": 0.66
        }
    }


@pytest.fixture(autouse=True)
def reset_predictor_metrics(predictor):
    """Reset predictor metrics before each test."""
    predictor.total_predictions = 0
    predictor.total_inference_time = 0.0
    yield


@pytest.fixture(scope="function")
def cleanup_files():
    """Fixture to clean up test files after tests."""
    files_to_cleanup = []

    def register_file(filepath):
        files_to_cleanup.append(filepath)

    yield register_file

    # Cleanup
    for filepath in files_to_cleanup:
        if os.path.exists(filepath):
            os.remove(filepath)


# Pytest markers
def pytest_configure(config):
    """Configure custom pytest markers."""
    config.addinivalue_line(
        "markers", "slow: marks tests as slow (deselect with '-m \"not slow\"')"
    )
    config.addinivalue_line(
        "markers", "api: marks tests as API tests"
    )
    config.addinivalue_line(
        "markers", "model: marks tests as model tests"
    )
    config.addinivalue_line(
        "markers", "inference: marks tests as inference tests"
    )
    config.addinivalue_line(
        "markers", "dataset: marks tests as dataset tests"
    )
    config.addinivalue_line(
        "markers", "evaluation: marks tests as evaluation tests"
    )
    config.addinivalue_line(
        "markers", "integration: marks tests as integration tests"
    )
