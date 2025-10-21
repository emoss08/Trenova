<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->

# Testing Guide

Comprehensive guide for testing the Document Quality Assessment model.

## Table of Contents

- [Overview](#overview)
- [Test Structure](#test-structure)
- [Setup](#setup)
- [Running Tests](#running-tests)
- [Test Coverage](#test-coverage)
- [Writing Tests](#writing-tests)
- [CI/CD Integration](#cicd-integration)
- [Troubleshooting](#troubleshooting)

## Overview

The test suite provides comprehensive coverage for all components of the document quality assessment system:

- **Model Architecture**: Tests for neural network components
- **Inference Engine**: Tests for prediction pipeline
- **API Endpoints**: Tests for FastAPI application
- **Dataset & Augmentation**: Tests for data processing
- **Evaluation Metrics**: Tests for metric calculations
- **Integration Tests**: End-to-end workflow tests

### Test Framework

- **pytest**: Primary testing framework
- **pytest-cov**: Code coverage reporting
- **pytest-asyncio**: Async/await support
- **FastAPI TestClient**: API endpoint testing

## Test Structure

```
tests/
├── __init__.py              # Test package initialization
├── conftest.py              # Pytest fixtures and configuration
├── test_models.py           # Model architecture tests
├── test_inference.py        # Inference engine tests
├── test_api.py             # API endpoint tests
├── test_dataset.py         # Dataset and augmentation tests
└── test_evaluation.py      # Evaluation metrics tests
```

### Test Organization

Tests are organized using pytest markers:

- `@pytest.mark.model` - Model architecture tests
- `@pytest.mark.inference` - Inference engine tests
- `@pytest.mark.api` - API endpoint tests
- `@pytest.mark.dataset` - Dataset tests
- `@pytest.mark.evaluation` - Evaluation tests
- `@pytest.mark.integration` - Integration tests
- `@pytest.mark.slow` - Slow-running tests

## Setup

### 1. Install Dependencies

```bash
# Install all requirements including test dependencies
pip install -r requirements.txt

# Or install test dependencies only
pip install pytest pytest-cov pytest-asyncio httpx
```

### 2. Verify Installation

```bash
pytest --version
```

### 3. Environment Setup

```bash
# Set Python path (if needed)
export PYTHONPATH="${PYTHONPATH}:$(pwd)/src"
```

## Running Tests

### Basic Usage

```bash
# Run all tests
pytest

# Run with verbose output
pytest -v

# Run specific test file
pytest tests/test_models.py

# Run specific test class
pytest tests/test_models.py::TestEnhancedDocumentQualityModel

# Run specific test function
pytest tests/test_models.py::TestEnhancedDocumentQualityModel::test_forward_shape
```

### Running by Marker

```bash
# Run only model tests
pytest -m model

# Run only API tests
pytest -m api

# Run all except slow tests
pytest -m "not slow"

# Run integration tests only
pytest -m integration

# Combine markers
pytest -m "api and not slow"
```

### Parallel Execution

```bash
# Install pytest-xdist for parallel execution
pip install pytest-xdist

# Run tests in parallel (4 workers)
pytest -n 4

# Run tests in parallel (auto-detect CPU count)
pytest -n auto
```

### Stop on First Failure

```bash
# Stop at first failure
pytest -x

# Stop after N failures
pytest --maxfail=3
```

### Verbose Output

```bash
# Show extra test details
pytest -v

# Show all output (including print statements)
pytest -s

# Show local variables on failure
pytest -l
```

## Test Coverage

### Generate Coverage Report

```bash
# Run tests with coverage
pytest --cov=src --cov-report=html

# Run with terminal report
pytest --cov=src --cov-report=term

# Run with both HTML and terminal reports
pytest --cov=src --cov-report=html --cov-report=term-missing

# Generate XML report (for CI/CD)
pytest --cov=src --cov-report=xml
```

### View Coverage Report

```bash
# Open HTML coverage report
open htmlcov/index.html  # macOS
xdg-open htmlcov/index.html  # Linux
start htmlcov/index.html  # Windows
```

### Coverage Thresholds

The project aims for:
- **Overall coverage**: ≥ 80%
- **Critical components** (model, inference): ≥ 90%
- **API endpoints**: ≥ 85%

### Check Coverage Thresholds

```bash
# Fail if coverage below 80%
pytest --cov=src --cov-fail-under=80
```

## Writing Tests

### Test Structure

```python
import pytest

@pytest.mark.model
class TestMyComponent:
    """Tests for MyComponent."""

    def test_initialization(self):
        """Test component initialization."""
        component = MyComponent()
        assert component is not None

    def test_forward_pass(self, sample_input):
        """Test forward pass with sample input."""
        component = MyComponent()
        output = component(sample_input)
        assert output.shape == (1, 10)

    @pytest.mark.parametrize("param1,param2", [
        (1, 2),
        (3, 4),
        (5, 6)
    ])
    def test_with_parameters(self, param1, param2):
        """Test with different parameters."""
        result = param1 + param2
        assert result > 0
```

### Using Fixtures

Fixtures are defined in [conftest.py](tests/conftest.py):

```python
def test_with_fixtures(model, sample_image, device):
    """Test using predefined fixtures."""
    # model, sample_image, device are automatically provided
    outputs = model(sample_image.to(device))
    assert outputs is not None
```

### Common Fixtures

- `device` - Compute device (CPU for testing)
- `sample_image` - Sample RGB image
- `sample_batch_tensor` - Batch of tensors
- `model` - Pre-initialized model
- `predictor` - Inference predictor
- `api_client` - FastAPI test client
- `temp_checkpoint_dir` - Temporary directory for files

### Mocking External Dependencies

```python
from unittest.mock import Mock, patch

def test_with_mock():
    """Test with mocked dependencies."""
    with patch('module.external_function') as mock_func:
        mock_func.return_value = "mocked value"
        result = function_that_calls_external()
        assert result == "expected result"
        mock_func.assert_called_once()
```

### Testing Async Functions

```python
import pytest

@pytest.mark.asyncio
async def test_async_function():
    """Test async function."""
    result = await async_function()
    assert result is not None
```

### Testing Exceptions

```python
def test_exception_handling():
    """Test exception handling."""
    with pytest.raises(ValueError):
        function_that_raises_error()

    with pytest.raises(ValueError, match="expected error message"):
        function_that_raises_error()
```

### Parametrized Tests

```python
@pytest.mark.parametrize("input,expected", [
    (1, 2),
    (2, 4),
    (3, 6)
])
def test_doubling(input, expected):
    """Test doubling function."""
    assert double(input) == expected
```

## Test Categories

### 1. Model Architecture Tests

Tests in [test_models.py](tests/test_models.py):

```bash
# Run all model tests
pytest -m model

# Test specific components
pytest tests/test_models.py::TestChannelAttention
pytest tests/test_models.py::TestCBAM
pytest tests/test_models.py::TestEnhancedDocumentQualityModel
```

**What's tested:**
- Component initialization
- Forward pass shapes
- Gradient flow
- Parameter counts
- State dict save/load

### 2. Inference Engine Tests

Tests in [test_inference.py](tests/test_inference.py):

```bash
# Run all inference tests
pytest -m inference

# Test specific functionality
pytest tests/test_inference.py::TestDocumentQualityPredictor::test_predict_single_image
pytest tests/test_inference.py::TestDocumentQualityPredictor::test_predict_batch
```

**What's tested:**
- Single image prediction
- Batch prediction
- Metrics tracking
- Preprocessing pipeline
- Issue detection
- Recommendation generation

### 3. API Endpoint Tests

Tests in [test_api.py](tests/test_api.py):

```bash
# Run all API tests
pytest -m api

# Test specific endpoints
pytest tests/test_api.py::TestHealthEndpoint
pytest tests/test_api.py::TestAnalyzeEndpoint
pytest tests/test_api.py::TestAnalyzeBatchEndpoint
```

**What's tested:**
- `/health` - Health check
- `/metrics` - Metrics retrieval
- `/analyze` - Single document analysis
- `/analyze/batch` - Batch document analysis
- Error handling
- Response validation

### 4. Dataset & Augmentation Tests

Tests in [test_dataset.py](tests/test_dataset.py):

```bash
# Run all dataset tests
pytest -m dataset

# Test specific functionality
pytest tests/test_dataset.py::TestTransportationDocumentAugmentation
pytest tests/test_dataset.py::TestDataTransforms
```

**What's tested:**
- Data augmentation
- Transformation pipelines
- Dataset metadata
- Quality score distributions

### 5. Evaluation Metrics Tests

Tests in [test_evaluation.py](tests/test_evaluation.py):

```bash
# Run all evaluation tests
pytest -m evaluation

# Test specific metrics
pytest tests/test_evaluation.py::TestMetricsCalculation
pytest tests/test_evaluation.py::TestCalibrationMetrics
pytest tests/test_evaluation.py::TestConfusionMatrix
```

**What's tested:**
- Regression metrics (MAE, RMSE, R²)
- Classification metrics (accuracy, F1, precision, recall)
- Calibration metrics (ECE, MCE, Brier score)
- Multi-label metrics
- Confusion matrices

### 6. Integration Tests

```bash
# Run integration tests
pytest -m integration
```

**What's tested:**
- End-to-end API workflows
- Model loading and inference
- Metrics accumulation
- Concurrent requests

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Set up Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.11'

      - name: Install dependencies
        run: |
          pip install -r requirements.txt

      - name: Run tests
        run: |
          pytest --cov=src --cov-report=xml --cov-report=term

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.xml
```

### GitLab CI Example

```yaml
test:
  image: python:3.11
  script:
    - pip install -r requirements.txt
    - pytest --cov=src --cov-report=xml --cov-report=term
  coverage: '/TOTAL.*\s+(\d+%)$/'
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml
```

## Performance Testing

### Timing Tests

```python
import time

def test_inference_performance(predictor, sample_image):
    """Test inference performance."""
    start = time.time()
    result = predictor.predict(sample_image)
    duration = time.time() - start

    # Should complete in < 5 seconds on CPU
    assert duration < 5.0
```

### Memory Testing

```python
import torch

def test_memory_usage(model, sample_tensor):
    """Test memory usage."""
    if torch.cuda.is_available():
        torch.cuda.reset_peak_memory_stats()
        output = model(sample_tensor)
        peak_memory = torch.cuda.max_memory_allocated()

        # Should use < 2GB
        assert peak_memory < 2 * 1024**3
```

## Troubleshooting

### Common Issues

#### 1. Import Errors

**Problem:**
```
ModuleNotFoundError: No module named 'src'
```

**Solution:**
```bash
# Add src to Python path
export PYTHONPATH="${PYTHONPATH}:$(pwd)/src"

# Or install in editable mode
pip install -e .
```

#### 2. Fixture Not Found

**Problem:**
```
fixture 'model' not found
```

**Solution:**
Ensure `conftest.py` is in the tests directory and pytest can discover it:
```bash
pytest --fixtures  # List all available fixtures
```

#### 3. CUDA Out of Memory

**Problem:**
```
RuntimeError: CUDA out of memory
```

**Solution:**
Tests run on CPU by default. If you enabled CUDA, reduce batch sizes:
```python
@pytest.fixture(scope="session")
def device():
    return torch.device("cpu")  # Force CPU for tests
```

#### 4. Slow Tests

**Problem:**
Tests take too long to run.

**Solution:**
```bash
# Skip slow tests
pytest -m "not slow"

# Run in parallel
pytest -n auto

# Use smaller models/batches in tests
```

#### 5. Failed to Load Model

**Problem:**
```
FileNotFoundError: model checkpoint not found
```

**Solution:**
Tests create temporary checkpoints. Ensure fixture `sample_checkpoint` is working:
```bash
pytest tests/conftest.py::test_sample_checkpoint -v
```

### Debugging Tests

#### Run with Debugging

```bash
# Drop into debugger on failure
pytest --pdb

# Drop into debugger on first failure
pytest --pdb -x
```

#### Show Print Statements

```bash
# Show all output
pytest -s

# Show output only for failed tests
pytest --capture=no
```

#### Increase Verbosity

```bash
# Very verbose
pytest -vv

# Show local variables on failure
pytest -l

# Show full diff on assertion errors
pytest -vv --tb=long
```

## Best Practices

### 1. Test Isolation

- Each test should be independent
- Use fixtures for setup/teardown
- Don't rely on test execution order

### 2. Descriptive Names

```python
# Good
def test_model_produces_correct_output_shape():
    pass

# Bad
def test1():
    pass
```

### 3. Test One Thing

```python
# Good
def test_initialization():
    model = Model()
    assert model is not None

def test_forward_pass():
    model = Model()
    output = model(input)
    assert output.shape == expected_shape

# Bad
def test_model():
    model = Model()
    assert model is not None
    output = model(input)
    assert output.shape == expected_shape
    # ... many more assertions
```

### 4. Use Assertions Effectively

```python
# Good - Clear assertion
assert result == expected, f"Expected {expected}, got {result}"

# Better - Use pytest helpers
assert result == pytest.approx(expected, rel=1e-5)

# Best - Use appropriate assertion
from pytest import approx
assert result == approx(expected)
```

### 5. Mark Slow Tests

```python
@pytest.mark.slow
def test_comprehensive_evaluation():
    """Long-running comprehensive test."""
    pass
```

## Quick Reference

### Common Commands

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=src --cov-report=html

# Run specific markers
pytest -m model
pytest -m "api and not slow"

# Run in parallel
pytest -n auto

# Stop on first failure
pytest -x

# Show output
pytest -s

# Debug on failure
pytest --pdb
```

### Test Statistics

```bash
# Show test durations
pytest --durations=10

# Show slowest tests
pytest --durations=0

# Generate JUnit XML report
pytest --junitxml=junit.xml
```

## Additional Resources

- [Pytest Documentation](https://docs.pytest.org/)
- [Pytest Best Practices](https://docs.pytest.org/en/stable/goodpractices.html)
- [FastAPI Testing](https://fastapi.tiangolo.com/tutorial/testing/)
- [Coverage.py Documentation](https://coverage.readthedocs.io/)

## Support

For issues or questions:
1. Check this guide first
2. Review test error messages carefully
3. Check fixture definitions in `conftest.py`
4. Open an issue on GitHub with test output
