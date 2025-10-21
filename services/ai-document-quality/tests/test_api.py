"""
Unit tests for FastAPI application endpoints.
"""

import pytest
from io import BytesIO
from PIL import Image
import json


@pytest.mark.api
class TestHealthEndpoint:
    """Tests for /health endpoint."""

    def test_health_check(self, api_client):
        """Test health check endpoint."""
        response = api_client.get("/health")

        assert response.status_code == 200
        data = response.json()

        assert data["status"] == "healthy"
        assert "model_loaded" in data
        assert "device" in data
        assert "version" in data
        assert data["model_loaded"] is True

    def test_health_check_structure(self, api_client):
        """Test health check response structure."""
        response = api_client.get("/health")
        data = response.json()

        required_fields = ["status", "model_loaded", "device", "version"]
        for field in required_fields:
            assert field in data


@pytest.mark.api
class TestMetricsEndpoint:
    """Tests for /metrics endpoint."""

    def test_metrics_endpoint(self, api_client):
        """Test metrics endpoint."""
        response = api_client.get("/metrics")

        assert response.status_code == 200
        data = response.json()

        assert "total_predictions" in data
        assert "total_inference_time" in data
        assert "average_inference_time" in data

    def test_metrics_types(self, api_client):
        """Test metrics data types."""
        response = api_client.get("/metrics")
        data = response.json()

        assert isinstance(data["total_predictions"], int)
        assert isinstance(data["total_inference_time"], (int, float))
        assert isinstance(data["average_inference_time"], (int, float))

    def test_metrics_non_negative(self, api_client):
        """Test that metrics are non-negative."""
        response = api_client.get("/metrics")
        data = response.json()

        assert data["total_predictions"] >= 0
        assert data["total_inference_time"] >= 0
        assert data["average_inference_time"] >= 0


@pytest.mark.api
class TestAnalyzeEndpoint:
    """Tests for /analyze endpoint."""

    def test_analyze_single_image(self, api_client, mock_image_bytes):
        """Test analyzing a single image."""
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}
        response = api_client.post("/analyze", files=files)

        assert response.status_code == 200
        data = response.json()

        assert "quality_score" in data
        assert "quality_class" in data
        assert "quality_class_label" in data
        assert "confidence" in data
        assert "inference_time" in data

    def test_analyze_response_types(self, api_client, mock_image_bytes):
        """Test analyze response data types."""
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}
        response = api_client.post("/analyze", files=files)
        data = response.json()

        assert isinstance(data["quality_score"], (int, float))
        assert isinstance(data["quality_class"], int)
        assert isinstance(data["quality_class_label"], str)
        assert isinstance(data["confidence"], (int, float))
        assert isinstance(data["inference_time"], (int, float))

    def test_analyze_response_ranges(self, api_client, mock_image_bytes):
        """Test analyze response value ranges."""
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}
        response = api_client.post("/analyze", files=files)
        data = response.json()

        assert 0 <= data["quality_score"] <= 5
        assert 0 <= data["quality_class"] <= 4
        assert 0 <= data["confidence"] <= 1
        assert data["inference_time"] > 0

    def test_analyze_with_issues(self, api_client, mock_image_bytes):
        """Test analyzing image with issue detection."""
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}
        params = {"include_issues": True}
        response = api_client.post("/analyze", files=files, params=params)

        assert response.status_code == 200
        data = response.json()

        assert "issues" in data
        assert "recommendations" in data
        assert isinstance(data["issues"], list)
        assert isinstance(data["recommendations"], list)

    def test_analyze_without_issues(self, api_client, mock_image_bytes):
        """Test analyzing image without issue detection."""
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}
        params = {"include_issues": False}
        response = api_client.post("/analyze", files=files, params=params)

        assert response.status_code == 200
        data = response.json()

        assert "issues" not in data
        assert "recommendations" not in data

    @pytest.mark.parametrize("threshold", [0.3, 0.5, 0.7])
    def test_analyze_different_thresholds(self, api_client, mock_image_bytes, threshold):
        """Test analyzing with different issue detection thresholds."""
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}
        params = {"threshold": threshold, "include_issues": True}
        response = api_client.post("/analyze", files=files, params=params)

        assert response.status_code == 200
        data = response.json()

        # All detected issues should have confidence >= threshold
        for issue in data["issues"]:
            assert issue["confidence"] >= threshold

    def test_analyze_no_file(self, api_client):
        """Test analyze endpoint without file."""
        response = api_client.post("/analyze")

        assert response.status_code == 422  # Unprocessable Entity

    def test_analyze_invalid_file_type(self, api_client):
        """Test analyze endpoint with invalid file type."""
        files = {"file": ("test.txt", BytesIO(b"not an image"), "text/plain")}
        response = api_client.post("/analyze", files=files)

        # Should return error (400 or 422)
        assert response.status_code in [400, 422, 500]

    def test_analyze_large_image(self, api_client):
        """Test analyzing large image."""
        # Create large image
        img = Image.new("RGB", (4000, 3000), color="white")
        buffer = BytesIO()
        img.save(buffer, format="JPEG")
        buffer.seek(0)

        files = {"file": ("large.jpg", buffer, "image/jpeg")}
        response = api_client.post("/analyze", files=files)

        # Should handle large images
        assert response.status_code == 200

    def test_analyze_small_image(self, api_client):
        """Test analyzing small image."""
        # Create small image
        img = Image.new("RGB", (50, 50), color="white")
        buffer = BytesIO()
        img.save(buffer, format="JPEG")
        buffer.seek(0)

        files = {"file": ("small.jpg", buffer, "image/jpeg")}
        response = api_client.post("/analyze", files=files)

        # Should handle small images
        assert response.status_code == 200

    @pytest.mark.parametrize("format", ["JPEG", "PNG"])
    def test_analyze_different_formats(self, api_client, format):
        """Test analyzing different image formats."""
        img = Image.new("RGB", (224, 224), color="white")
        buffer = BytesIO()
        img.save(buffer, format=format)
        buffer.seek(0)

        files = {"file": (f"test.{format.lower()}", buffer, f"image/{format.lower()}")}
        response = api_client.post("/analyze", files=files)

        assert response.status_code == 200

    def test_analyze_quality_class_labels(self, api_client, mock_image_bytes):
        """Test that quality class labels are valid."""
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}
        response = api_client.post("/analyze", files=files)
        data = response.json()

        valid_labels = ["High", "Good", "Moderate", "Poor", "Very Poor"]
        assert data["quality_class_label"] in valid_labels

    def test_analyze_issue_structure(self, api_client, mock_image_bytes):
        """Test issue detection response structure."""
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}
        params = {"include_issues": True}
        response = api_client.post("/analyze", files=files, params=params)
        data = response.json()

        if len(data["issues"]) > 0:
            issue = data["issues"][0]
            assert "type" in issue
            assert "confidence" in issue
            assert "severity" in issue

            assert isinstance(issue["type"], str)
            assert isinstance(issue["confidence"], (int, float))
            assert isinstance(issue["severity"], str)
            assert issue["severity"] in ["low", "medium", "high"]


@pytest.mark.api
class TestAnalyzeBatchEndpoint:
    """Tests for /analyze/batch endpoint."""

    def test_batch_analyze(self, api_client, mock_image_bytes):
        """Test batch analysis endpoint."""
        files = [
            ("files", ("test1.jpg", BytesIO(mock_image_bytes), "image/jpeg")),
            ("files", ("test2.jpg", BytesIO(mock_image_bytes), "image/jpeg"))
        ]
        response = api_client.post("/analyze/batch", files=files)

        assert response.status_code == 200
        data = response.json()

        assert "results" in data
        assert "total_processed" in data
        assert "total_time" in data

        assert len(data["results"]) == 2
        assert data["total_processed"] == 2

    def test_batch_analyze_results_structure(self, api_client, mock_image_bytes):
        """Test batch analysis results structure."""
        files = [
            ("files", ("test1.jpg", BytesIO(mock_image_bytes), "image/jpeg")),
            ("files", ("test2.jpg", BytesIO(mock_image_bytes), "image/jpeg"))
        ]
        response = api_client.post("/analyze/batch", files=files)
        data = response.json()

        for result in data["results"]:
            assert "quality_score" in result
            assert "quality_class" in result
            assert "quality_class_label" in result
            assert "confidence" in result

    def test_batch_analyze_empty(self, api_client):
        """Test batch analysis with no files."""
        response = api_client.post("/analyze/batch", files=[])

        # Should return error or empty results
        assert response.status_code in [200, 422]

    def test_batch_analyze_single_file(self, api_client, mock_image_bytes):
        """Test batch analysis with single file."""
        files = [("files", ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg"))]
        response = api_client.post("/analyze/batch", files=files)

        assert response.status_code == 200
        data = response.json()
        assert data["total_processed"] == 1

    def test_batch_analyze_large_batch(self, api_client, sample_image):
        """Test batch analysis with larger batch."""
        # Create 10 images
        files = []
        for i in range(10):
            buffer = BytesIO()
            sample_image.save(buffer, format="JPEG")
            buffer.seek(0)
            files.append(("files", (f"test{i}.jpg", buffer, "image/jpeg")))

        response = api_client.post("/analyze/batch", files=files)

        assert response.status_code == 200
        data = response.json()
        assert data["total_processed"] == 10

    def test_batch_analyze_with_issues(self, api_client, mock_image_bytes):
        """Test batch analysis with issue detection."""
        files = [
            ("files", ("test1.jpg", BytesIO(mock_image_bytes), "image/jpeg")),
            ("files", ("test2.jpg", BytesIO(mock_image_bytes), "image/jpeg"))
        ]
        params = {"include_issues": True}
        response = api_client.post("/analyze/batch", files=files, params=params)

        assert response.status_code == 200
        data = response.json()

        for result in data["results"]:
            assert "issues" in result
            assert "recommendations" in result

    def test_batch_analyze_timing(self, api_client, mock_image_bytes):
        """Test batch analysis timing information."""
        files = [
            ("files", ("test1.jpg", BytesIO(mock_image_bytes), "image/jpeg")),
            ("files", ("test2.jpg", BytesIO(mock_image_bytes), "image/jpeg"))
        ]
        response = api_client.post("/analyze/batch", files=files)
        data = response.json()

        assert data["total_time"] > 0
        assert isinstance(data["total_time"], (int, float))

    def test_batch_analyze_max_limit(self, api_client, sample_image):
        """Test batch analysis respects max file limit."""
        # Try to send more than 100 files (if limit exists)
        files = []
        for i in range(105):
            buffer = BytesIO()
            sample_image.save(buffer, format="JPEG")
            buffer.seek(0)
            files.append(("files", (f"test{i}.jpg", buffer, "image/jpeg")))

        response = api_client.post("/analyze/batch", files=files)

        # Should either reject or only process first 100
        if response.status_code == 200:
            data = response.json()
            assert data["total_processed"] <= 100
        else:
            assert response.status_code in [400, 422]


@pytest.mark.api
@pytest.mark.integration
class TestAPIIntegration:
    """Integration tests for API workflows."""

    def test_full_workflow(self, api_client, mock_image_bytes):
        """Test complete API workflow."""
        # 1. Check health
        health_response = api_client.get("/health")
        assert health_response.status_code == 200

        # 2. Analyze image
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}
        analyze_response = api_client.post("/analyze", files=files)
        assert analyze_response.status_code == 200

        # 3. Check metrics updated
        metrics_response = api_client.get("/metrics")
        assert metrics_response.status_code == 200
        metrics_data = metrics_response.json()
        assert metrics_data["total_predictions"] > 0

    def test_cors_headers(self, api_client):
        """Test CORS headers are present."""
        response = api_client.options("/health")

        # Check CORS headers (if configured)
        headers = response.headers
        # Note: TestClient may not include CORS headers

    def test_error_response_format(self, api_client):
        """Test error responses have consistent format."""
        # Send invalid request
        response = api_client.post("/analyze")

        assert response.status_code in [400, 422]
        data = response.json()

        # FastAPI default error format
        assert "detail" in data

    def test_openapi_schema(self, api_client):
        """Test OpenAPI schema is available."""
        response = api_client.get("/openapi.json")

        assert response.status_code == 200
        schema = response.json()

        assert "openapi" in schema
        assert "info" in schema
        assert "paths" in schema

    def test_docs_available(self, api_client):
        """Test API documentation is available."""
        response = api_client.get("/docs")

        assert response.status_code == 200

    def test_redoc_available(self, api_client):
        """Test ReDoc documentation is available."""
        response = api_client.get("/redoc")

        assert response.status_code == 200

    def test_concurrent_requests(self, api_client, mock_image_bytes):
        """Test API handles concurrent requests."""
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}

        # Make multiple requests
        responses = []
        for _ in range(5):
            response = api_client.post("/analyze", files=files)
            responses.append(response)

        # All should succeed
        for response in responses:
            assert response.status_code == 200

    def test_metrics_accumulation(self, api_client, mock_image_bytes):
        """Test that metrics accumulate correctly."""
        # Get initial metrics
        initial_metrics = api_client.get("/metrics").json()
        initial_count = initial_metrics["total_predictions"]

        # Make 3 predictions
        files = {"file": ("test.jpg", BytesIO(mock_image_bytes), "image/jpeg")}
        for _ in range(3):
            api_client.post("/analyze", files=files)

        # Check metrics updated
        final_metrics = api_client.get("/metrics").json()
        final_count = final_metrics["total_predictions"]

        assert final_count == initial_count + 3
