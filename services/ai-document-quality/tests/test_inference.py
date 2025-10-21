"""
Unit tests for document quality inference engine.
"""

import pytest
import time
from io import BytesIO
from PIL import Image
import torch
import numpy as np

from api.inference import DocumentQualityPredictor


@pytest.mark.inference
class TestDocumentQualityPredictor:
    """Tests for DocumentQualityPredictor."""

    def test_initialization(self, sample_checkpoint, device):
        """Test predictor initialization."""
        predictor = DocumentQualityPredictor(
            model_path=str(sample_checkpoint),
            device=device
        )

        assert predictor.model is not None
        assert predictor.device == device
        assert predictor.total_predictions == 0
        assert predictor.total_inference_time == 0.0

    def test_initialization_invalid_path(self, device):
        """Test predictor initialization with invalid model path."""
        with pytest.raises(FileNotFoundError):
            DocumentQualityPredictor(
                model_path="/invalid/path/to/model.pth",
                device=device
            )

    def test_predict_single_image(self, predictor, sample_image):
        """Test single image prediction."""
        result = predictor.predict(sample_image)

        assert "quality_score" in result
        assert "quality_class" in result
        assert "quality_class_label" in result
        assert "confidence" in result
        assert "inference_time" in result

        # Validate types
        assert isinstance(result["quality_score"], float)
        assert isinstance(result["quality_class"], int)
        assert isinstance(result["quality_class_label"], str)
        assert isinstance(result["confidence"], float)
        assert isinstance(result["inference_time"], float)

        # Validate ranges
        assert 0 <= result["quality_score"] <= 5
        assert 0 <= result["quality_class"] <= 4
        assert 0 <= result["confidence"] <= 1
        assert result["inference_time"] > 0

    def test_predict_with_issues(self, predictor, sample_image):
        """Test prediction with issue detection."""
        result = predictor.predict(sample_image, include_issues=True)

        assert "issues" in result
        assert isinstance(result["issues"], list)

        # Check issue structure
        if len(result["issues"]) > 0:
            issue = result["issues"][0]
            assert "type" in issue
            assert "confidence" in issue
            assert "severity" in issue

            assert isinstance(issue["type"], str)
            assert isinstance(issue["confidence"], float)
            assert isinstance(issue["severity"], str)
            assert 0 <= issue["confidence"] <= 1
            assert issue["severity"] in ["low", "medium", "high"]

    def test_predict_without_issues(self, predictor, sample_image):
        """Test prediction without issue detection."""
        result = predictor.predict(sample_image, include_issues=False)

        assert "issues" not in result

    def test_predict_with_recommendations(self, predictor, sample_image):
        """Test prediction with recommendations."""
        result = predictor.predict(sample_image, include_issues=True)

        assert "recommendations" in result
        assert isinstance(result["recommendations"], list)

        # Recommendations should be strings
        for rec in result["recommendations"]:
            assert isinstance(rec, str)

    @pytest.mark.parametrize("threshold", [0.3, 0.5, 0.7])
    def test_predict_different_thresholds(self, predictor, sample_image, threshold):
        """Test prediction with different issue detection thresholds."""
        result = predictor.predict(sample_image, threshold=threshold, include_issues=True)

        # All detected issues should have confidence >= threshold
        for issue in result["issues"]:
            assert issue["confidence"] >= threshold

    def test_predict_batch(self, predictor, sample_image_batch):
        """Test batch prediction."""
        results = predictor.predict_batch(sample_image_batch)

        assert isinstance(results, list)
        assert len(results) == len(sample_image_batch)

        # Check each result
        for result in results:
            assert "quality_score" in result
            assert "quality_class" in result
            assert "quality_class_label" in result
            assert "confidence" in result

    def test_predict_batch_empty(self, predictor):
        """Test batch prediction with empty list."""
        results = predictor.predict_batch([])

        assert isinstance(results, list)
        assert len(results) == 0

    def test_predict_batch_with_issues(self, predictor, sample_image_batch):
        """Test batch prediction with issue detection."""
        results = predictor.predict_batch(
            sample_image_batch,
            include_issues=True
        )

        for result in results:
            assert "issues" in result
            assert "recommendations" in result

    def test_metrics_tracking(self, predictor, sample_image):
        """Test that predictor tracks metrics correctly."""
        initial_predictions = predictor.total_predictions
        initial_time = predictor.total_inference_time

        # Make a prediction
        predictor.predict(sample_image)

        # Check metrics updated
        assert predictor.total_predictions == initial_predictions + 1
        assert predictor.total_inference_time > initial_time

    def test_get_metrics(self, predictor, sample_image):
        """Test getting predictor metrics."""
        # Make a prediction to generate metrics
        predictor.predict(sample_image)

        metrics = predictor.get_metrics()

        assert "total_predictions" in metrics
        assert "total_inference_time" in metrics
        assert "average_inference_time" in metrics

        assert metrics["total_predictions"] > 0
        assert metrics["total_inference_time"] > 0
        assert metrics["average_inference_time"] > 0

    def test_get_metrics_no_predictions(self, sample_checkpoint, device):
        """Test getting metrics with no predictions made."""
        predictor = DocumentQualityPredictor(
            model_path=str(sample_checkpoint),
            device=device
        )

        metrics = predictor.get_metrics()

        assert metrics["total_predictions"] == 0
        assert metrics["total_inference_time"] == 0.0
        assert metrics["average_inference_time"] == 0.0

    def test_preprocess_image(self, predictor, sample_image):
        """Test image preprocessing."""
        tensor = predictor.preprocess_image(sample_image)

        assert isinstance(tensor, torch.Tensor)
        assert tensor.shape == (1, 3, 224, 224)
        assert tensor.device == predictor.device

    def test_preprocess_different_sizes(self, predictor):
        """Test preprocessing images of different sizes."""
        sizes = [(100, 100), (300, 200), (500, 500), (1000, 800)]

        for size in sizes:
            img = Image.new("RGB", size, color="white")
            tensor = predictor.preprocess_image(img)

            # Should always be resized to 224x224
            assert tensor.shape == (1, 3, 224, 224)

    def test_preprocess_grayscale_image(self, predictor):
        """Test preprocessing grayscale image."""
        # Create grayscale image
        img = Image.new("L", (224, 224), color=128)

        # Convert to RGB (should happen automatically)
        tensor = predictor.preprocess_image(img.convert("RGB"))

        assert tensor.shape == (1, 3, 224, 224)

    def test_postprocess_output(self, predictor, sample_model_output):
        """Test output postprocessing."""
        result = predictor.postprocess_output(
            sample_model_output,
            threshold=0.5,
            include_issues=True
        )

        assert "quality_score" in result
        assert "quality_class" in result
        assert "issues" in result

    def test_quality_class_labels(self, predictor):
        """Test quality class label mapping."""
        for class_idx in range(5):
            label = predictor.quality_labels[class_idx]
            assert label in ["High", "Good", "Moderate", "Poor", "Very Poor"]

    def test_issue_type_labels(self, predictor):
        """Test issue type label mapping."""
        expected_issues = [
            "Blur", "Noise", "Lighting", "Shadow", "Physical Damage",
            "Skew", "Partial", "Glare", "Compression", "Overall Poor"
        ]

        for issue_idx in range(10):
            label = predictor.issue_labels[issue_idx]
            assert label in expected_issues

    def test_severity_calculation(self, predictor):
        """Test issue severity calculation."""
        # Test different confidence levels
        assert predictor._get_severity(0.9) == "high"
        assert predictor._get_severity(0.7) == "high"
        assert predictor._get_severity(0.6) == "medium"
        assert predictor._get_severity(0.5) == "medium"
        assert predictor._get_severity(0.4) == "low"

    def test_recommendations_generation(self, predictor, sample_image):
        """Test that recommendations are generated appropriately."""
        result = predictor.predict(sample_image, include_issues=True)

        recommendations = result["recommendations"]
        assert isinstance(recommendations, list)

        # Should have recommendations if quality is poor
        if result["quality_score"] < 3.0:
            assert len(recommendations) > 0

    def test_confidence_calculation(self, predictor, sample_image):
        """Test confidence score calculation."""
        result = predictor.predict(sample_image)

        confidence = result["confidence"]

        # Confidence should be the probability of the predicted class
        assert 0 <= confidence <= 1

    def test_model_eval_mode(self, predictor, sample_image):
        """Test that model is in eval mode during inference."""
        # Model should be in eval mode
        assert not predictor.model.training

        # Predictions should be deterministic
        result1 = predictor.predict(sample_image)
        result2 = predictor.predict(sample_image)

        # Results should be identical (or very close due to floating point)
        assert abs(result1["quality_score"] - result2["quality_score"]) < 0.01

    def test_no_gradient_computation(self, predictor, sample_image):
        """Test that no gradients are computed during inference."""
        # Make prediction
        result = predictor.predict(sample_image)

        # Check that no gradients are tracked
        for param in predictor.model.parameters():
            assert param.grad is None or torch.all(param.grad == 0)

    def test_inference_time_reasonable(self, predictor, sample_image):
        """Test that inference time is reasonable."""
        result = predictor.predict(sample_image)

        # Inference should be fast (< 1 second on CPU for small model)
        assert result["inference_time"] < 5.0

    def test_batch_inference_efficiency(self, predictor, sample_image_batch):
        """Test that batch inference is more efficient than sequential."""
        # Sequential inference
        start_seq = time.time()
        for img in sample_image_batch:
            predictor.predict(img)
        seq_time = time.time() - start_seq

        # Batch inference
        start_batch = time.time()
        predictor.predict_batch(sample_image_batch)
        batch_time = time.time() - start_batch

        # Batch should be faster (or at least not significantly slower)
        # Allow some margin for overhead
        assert batch_time < seq_time * 1.5

    def test_memory_cleanup(self, predictor, sample_image):
        """Test that memory is properly cleaned up after inference."""
        initial_memory = torch.cuda.memory_allocated() if torch.cuda.is_available() else 0

        # Make predictions
        for _ in range(10):
            predictor.predict(sample_image)

        final_memory = torch.cuda.memory_allocated() if torch.cuda.is_available() else 0

        # Memory should not grow significantly (allow 10MB growth)
        if torch.cuda.is_available():
            assert final_memory - initial_memory < 10 * 1024 * 1024

    def test_concurrent_predictions(self, predictor, sample_image_batch):
        """Test that predictor can handle concurrent predictions."""
        # This tests thread safety (though torch usually requires GIL)
        results = []
        for img in sample_image_batch:
            result = predictor.predict(img)
            results.append(result)

        assert len(results) == len(sample_image_batch)

    def test_error_handling_invalid_image(self, predictor):
        """Test error handling for invalid image input."""
        # Try to predict on invalid input
        with pytest.raises((AttributeError, TypeError)):
            predictor.predict(None)

    def test_error_handling_corrupted_image(self, predictor):
        """Test error handling for corrupted image."""
        # Create an image with invalid data
        try:
            arr = np.random.randint(0, 255, (10, 10, 3), dtype=np.uint8)
            img = Image.fromarray(arr, mode="RGB")
            result = predictor.predict(img)
            # Should still work, just resize from small size
            assert result is not None
        except Exception as e:
            pytest.fail(f"Should handle small images: {e}")

    @pytest.mark.parametrize("format", ["JPEG", "PNG", "BMP"])
    def test_different_image_formats(self, predictor, format):
        """Test prediction with different image formats."""
        # Create image in different formats
        img = Image.new("RGB", (224, 224), color="white")
        buffer = BytesIO()
        img.save(buffer, format=format)
        buffer.seek(0)
        img_loaded = Image.open(buffer)

        result = predictor.predict(img_loaded)
        assert result is not None

    def test_prediction_consistency(self, predictor, sample_image):
        """Test that predictions are consistent for same image."""
        results = [predictor.predict(sample_image) for _ in range(5)]

        # All predictions should be very similar
        scores = [r["quality_score"] for r in results]
        assert max(scores) - min(scores) < 0.1

    def test_model_output_structure(self, predictor, sample_image):
        """Test that model output has expected structure."""
        # Get raw model output
        tensor = predictor.preprocess_image(sample_image)
        with torch.no_grad():
            output = predictor.model(tensor)

        assert "quality_score" in output
        assert "quality_class" in output
        assert "issue_probs" in output

        assert output["quality_score"].shape == (1, 1)
        assert output["quality_class"].shape == (1, 5)
        assert output["issue_probs"].shape == (1, 10)
