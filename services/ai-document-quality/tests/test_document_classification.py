"""
Unit tests for document type classification system.
"""

import pytest
import torch
import numpy as np
from PIL import Image

from models.document_classifier import (
    DocumentTypeEncoder,
    DocumentTypeClassifier,
    CustomerTemplateBank,
    DocumentType,
    STANDARD_DOCUMENT_TYPES
)
from api.document_classification import DocumentClassificationService


@pytest.mark.model
class TestDocumentTypeEncoder:
    """Tests for DocumentTypeEncoder."""

    def test_initialization(self):
        """Test encoder initialization."""
        encoder = DocumentTypeEncoder(
            backbone="efficientnet_b0",
            feature_dim=512,
            pretrained=False
        )

        assert encoder is not None
        assert encoder.feature_projection is not None

    def test_forward_shape(self, device):
        """Test encoder forward pass output shape."""
        encoder = DocumentTypeEncoder(
            backbone="efficientnet_b0",
            feature_dim=512,
            pretrained=False
        ).to(device)

        x = torch.randn(2, 3, 224, 224, device=device)
        features = encoder(x)

        assert features.shape == (2, 512)

    def test_feature_normalization(self, device):
        """Test that features are L2 normalized."""
        encoder = DocumentTypeEncoder(
            backbone="efficientnet_b0",
            feature_dim=512,
            pretrained=False
        ).to(device)
        encoder.normalize = True

        x = torch.randn(1, 3, 224, 224, device=device)
        features = encoder(x)

        # L2 norm should be 1
        norm = torch.norm(features, p=2, dim=1)
        assert torch.allclose(norm, torch.ones_like(norm), atol=1e-5)

    def test_batch_processing(self, device):
        """Test encoder with different batch sizes."""
        encoder = DocumentTypeEncoder(pretrained=False).to(device)

        for batch_size in [1, 4, 8]:
            x = torch.randn(batch_size, 3, 224, 224, device=device)
            features = encoder(x)
            assert features.shape == (batch_size, 512)


@pytest.mark.model
class TestCustomerTemplateBank:
    """Tests for CustomerTemplateBank."""

    def test_initialization(self):
        """Test template bank initialization."""
        bank = CustomerTemplateBank(feature_dim=512)

        assert bank.feature_dim == 512
        assert len(bank.templates) == 0

    def test_add_template(self):
        """Test adding a template."""
        bank = CustomerTemplateBank(feature_dim=512)

        features = torch.randn(512)
        bank.add_template(
            customer_id="customer_1",
            doc_type="BOL",
            features=features,
            metadata={"template_id": "v1"}
        )

        assert "customer_1" in bank.templates
        assert "BOL" in bank.templates["customer_1"]
        assert len(bank.templates["customer_1"]["BOL"]) == 1

    def test_add_multiple_templates(self):
        """Test adding multiple templates for same customer."""
        bank = CustomerTemplateBank(feature_dim=512)

        for i in range(5):
            features = torch.randn(512)
            bank.add_template(
                customer_id="customer_1",
                doc_type="BOL",
                features=features
            )

        assert bank.template_counts["customer_1"]["BOL"] == 5

    def test_find_similar_templates(self):
        """Test finding similar templates."""
        bank = CustomerTemplateBank(feature_dim=512)

        # Add templates
        template_features = torch.randn(512)
        bank.add_template(
            customer_id="customer_1",
            doc_type="BOL",
            features=template_features
        )

        # Query with similar features
        query = template_features + torch.randn(512) * 0.1  # Add small noise
        results = bank.find_similar_templates(
            customer_id="customer_1",
            features=query,
            threshold=0.5
        )

        assert len(results) > 0
        assert results[0][0] == "BOL"  # doc_type
        assert results[0][1] > 0.5  # similarity

    def test_find_similar_templates_no_match(self):
        """Test finding templates with no customer."""
        bank = CustomerTemplateBank(feature_dim=512)

        query = torch.randn(512)
        results = bank.find_similar_templates(
            customer_id="nonexistent",
            features=query
        )

        assert len(results) == 0

    def test_get_customer_stats(self):
        """Test getting customer statistics."""
        bank = CustomerTemplateBank(feature_dim=512)

        bank.add_template("customer_1", "BOL", torch.randn(512))
        bank.add_template("customer_1", "BOL", torch.randn(512))
        bank.add_template("customer_1", "INVOICE", torch.randn(512))

        stats = bank.get_customer_stats("customer_1")

        assert stats["BOL"] == 2
        assert stats["INVOICE"] == 1

    def test_save_and_load(self, temp_checkpoint_dir):
        """Test saving and loading template bank."""
        bank = CustomerTemplateBank(feature_dim=512)

        # Add some templates
        bank.add_template("customer_1", "BOL", torch.randn(512))
        bank.add_template("customer_2", "INVOICE", torch.randn(512))

        # Save
        save_path = temp_checkpoint_dir / "templates.pth"
        bank.save(str(save_path))

        # Load into new bank
        new_bank = CustomerTemplateBank(feature_dim=512)
        new_bank.load(str(save_path))

        assert new_bank.feature_dim == 512
        assert "customer_1" in new_bank.templates
        assert "customer_2" in new_bank.templates


@pytest.mark.model
class TestDocumentTypeClassifier:
    """Tests for DocumentTypeClassifier."""

    def test_initialization(self):
        """Test classifier initialization."""
        classifier = DocumentTypeClassifier(
            num_base_types=10,
            feature_dim=512,
            pretrained=False
        )

        assert classifier.encoder is not None
        assert classifier.base_classifier is not None
        assert classifier.template_bank is not None

    def test_forward_shape(self, device):
        """Test classifier forward pass output shapes."""
        classifier = DocumentTypeClassifier(
            num_base_types=10,
            feature_dim=512,
            pretrained=False
        ).to(device)

        x = torch.randn(2, 3, 224, 224, device=device)
        outputs = classifier(x)

        assert "features" in outputs
        assert "base_logits" in outputs
        assert "base_probs" in outputs

        assert outputs["features"].shape == (2, 512)
        assert outputs["base_logits"].shape == (2, 10)
        assert outputs["base_probs"].shape == (2, 10)

    def test_base_probs_sum_to_one(self, device):
        """Test that base probabilities sum to 1."""
        classifier = DocumentTypeClassifier(pretrained=False).to(device)

        x = torch.randn(1, 3, 224, 224, device=device)
        outputs = classifier(x)

        probs_sum = outputs["base_probs"].sum(dim=1)
        assert torch.allclose(probs_sum, torch.ones_like(probs_sum), atol=1e-5)

    def test_classify_without_customer(self, device):
        """Test classification without customer ID."""
        classifier = DocumentTypeClassifier(pretrained=False).to(device)

        x = torch.randn(1, 3, 224, 224, device=device)
        predictions = classifier.classify(x, customer_id=None, return_top_k=3)

        assert isinstance(predictions, list)
        assert len(predictions) <= 3

        for pred in predictions:
            assert "document_type" in pred
            assert "confidence" in pred
            assert "source" in pred
            assert pred["source"] == "base_classifier"

    def test_classify_with_customer(self, device):
        """Test classification with customer ID."""
        classifier = DocumentTypeClassifier(pretrained=False).to(device)

        # Learn a template first
        x = torch.randn(1, 3, 224, 224, device=device)
        classifier.learn_customer_template(
            image=x,
            customer_id="customer_1",
            doc_type="BOL",
            template_id="v1"
        )

        # Classify same image
        predictions = classifier.classify(
            x,
            customer_id="customer_1",
            return_top_k=5,
            confidence_threshold=0.5
        )

        # Should have both base and customer predictions
        sources = [p["source"] for p in predictions]
        assert "base_classifier" in sources

        # May have customer_template if similarity is high enough
        customer_preds = [p for p in predictions if p["source"] == "customer_template"]
        if customer_preds:
            assert customer_preds[0]["customer_id"] == "customer_1"

    def test_learn_customer_template(self, device):
        """Test learning customer template."""
        classifier = DocumentTypeClassifier(pretrained=False).to(device)

        x = torch.randn(1, 3, 224, 224, device=device)
        classifier.learn_customer_template(
            image=x,
            customer_id="fedex",
            doc_type="BOL",
            template_id="standard"
        )

        # Check template was added
        stats = classifier.template_bank.get_customer_stats("fedex")
        assert stats["BOL"] == 1

    def test_get_customer_info(self, device):
        """Test getting customer information."""
        classifier = DocumentTypeClassifier(pretrained=False).to(device)

        # Add templates
        for i in range(3):
            x = torch.randn(1, 3, 224, 224, device=device)
            classifier.learn_customer_template(
                image=x,
                customer_id="customer_1",
                doc_type="BOL"
            )

        info = classifier.get_customer_info("customer_1")

        assert info["customer_id"] == "customer_1"
        assert info["total_templates"] == 3
        assert "BOL" in info["document_types"]


@pytest.mark.inference
class TestDocumentClassificationService:
    """Tests for DocumentClassificationService."""

    @pytest.fixture
    def classification_service(self, device):
        """Create classification service for testing."""
        service = DocumentClassificationService(
            model_path=None,
            template_bank_path=None,
            device=device
        )
        return service

    def test_initialization(self, classification_service):
        """Test service initialization."""
        assert classification_service.model is not None
        assert classification_service.transform is not None
        assert classification_service.total_classifications == 0

    def test_preprocess_image(self, classification_service, sample_image):
        """Test image preprocessing."""
        tensor = classification_service.preprocess_image(sample_image)

        assert isinstance(tensor, torch.Tensor)
        assert tensor.shape == (1, 3, 224, 224)

    def test_classify_document(self, classification_service, sample_image):
        """Test single document classification."""
        result = classification_service.classify_document(
            image=sample_image,
            customer_id=None,
            top_k=3
        )

        assert "predictions" in result
        assert "inference_time" in result
        assert "num_predictions" in result
        assert "best_prediction" in result

        assert len(result["predictions"]) <= 3

    def test_classify_with_customer_id(self, classification_service, sample_image):
        """Test classification with customer ID."""
        # First learn a template
        classification_service.learn_customer_template(
            image=sample_image,
            customer_id="fedex",
            document_type="BOL"
        )

        # Then classify
        result = classification_service.classify_document(
            image=sample_image,
            customer_id="fedex",
            top_k=5
        )

        assert result["customer_id"] == "fedex"
        assert "has_customer_match" in result

    def test_classify_batch(self, classification_service, sample_image_batch):
        """Test batch classification."""
        results = classification_service.classify_batch(
            images=sample_image_batch,
            customer_ids=None,
            top_k=3
        )

        assert isinstance(results, list)
        assert len(results) == len(sample_image_batch)

        for result in results:
            assert "predictions" in result
            assert "inference_time" in result

    def test_classify_batch_with_customer_ids(
        self, classification_service, sample_image_batch
    ):
        """Test batch classification with customer IDs."""
        customer_ids = ["customer_1", "customer_2", "customer_1", "customer_3"]

        results = classification_service.classify_batch(
            images=sample_image_batch,
            customer_ids=customer_ids
        )

        assert len(results) == len(sample_image_batch)

        for result, customer_id in zip(results, customer_ids):
            assert result["customer_id"] == customer_id

    def test_learn_customer_template(self, classification_service, sample_image):
        """Test learning customer template."""
        result = classification_service.learn_customer_template(
            image=sample_image,
            customer_id="ups",
            document_type="BOL",
            template_id="standard"
        )

        assert result["success"] is True
        assert result["customer_id"] == "ups"
        assert result["document_type"] == "BOL"
        assert result["total_templates"] == 1

    def test_learn_invalid_document_type(self, classification_service, sample_image):
        """Test learning with invalid document type."""
        with pytest.raises(ValueError):
            classification_service.learn_customer_template(
                image=sample_image,
                customer_id="customer_1",
                document_type="INVALID_TYPE"
            )

    def test_get_customer_templates(self, classification_service, sample_image):
        """Test getting customer template information."""
        # Learn some templates
        classification_service.learn_customer_template(
            image=sample_image,
            customer_id="customer_1",
            document_type="BOL"
        )
        classification_service.learn_customer_template(
            image=sample_image,
            customer_id="customer_1",
            document_type="INVOICE"
        )

        info = classification_service.get_customer_templates("customer_1")

        assert info["customer_id"] == "customer_1"
        assert info["total_templates"] == 2
        assert info["has_templates"] is True
        assert "BOL" in info["document_types"]
        assert "INVOICE" in info["document_types"]

    def test_get_all_customers(self, classification_service, sample_image):
        """Test getting all customers."""
        # Add templates for multiple customers
        classification_service.learn_customer_template(
            image=sample_image,
            customer_id="customer_1",
            document_type="BOL"
        )
        classification_service.learn_customer_template(
            image=sample_image,
            customer_id="customer_2",
            document_type="INVOICE"
        )

        customers = classification_service.get_all_customers()

        assert len(customers) == 2
        customer_ids = [c["customer_id"] for c in customers]
        assert "customer_1" in customer_ids
        assert "customer_2" in customer_ids

    def test_get_supported_document_types(self, classification_service):
        """Test getting supported document types."""
        doc_types = classification_service.get_supported_document_types()

        assert isinstance(doc_types, dict)
        assert "BOL" in doc_types
        assert "INVOICE" in doc_types
        assert len(doc_types) == len(STANDARD_DOCUMENT_TYPES)

    def test_metrics_tracking(self, classification_service, sample_image):
        """Test that metrics are tracked correctly."""
        initial_count = classification_service.total_classifications

        classification_service.classify_document(sample_image)

        assert classification_service.total_classifications == initial_count + 1
        assert classification_service.total_inference_time > 0

    def test_get_metrics(self, classification_service, sample_image):
        """Test getting service metrics."""
        classification_service.classify_document(sample_image)
        classification_service.learn_customer_template(
            image=sample_image,
            customer_id="customer_1",
            document_type="BOL"
        )

        metrics = classification_service.get_metrics()

        assert "total_classifications" in metrics
        assert "total_template_learnings" in metrics
        assert "average_inference_time" in metrics
        assert metrics["total_classifications"] > 0
        assert metrics["total_template_learnings"] > 0

    def test_save_template_bank(self, classification_service, sample_image, temp_checkpoint_dir):
        """Test saving template bank."""
        # Add some templates
        classification_service.learn_customer_template(
            image=sample_image,
            customer_id="customer_1",
            document_type="BOL"
        )

        # Save
        save_path = temp_checkpoint_dir / "templates.pth"
        classification_service.save_template_bank(str(save_path))

        assert save_path.exists()


@pytest.mark.model
class TestDocumentType:
    """Tests for DocumentType dataclass."""

    def test_base_type_only(self):
        """Test document type with base type only."""
        doc_type = DocumentType(base_type="BOL")

        assert doc_type.base_type == "BOL"
        assert doc_type.customer_id is None
        assert doc_type.full_type == "BOL"
        assert not doc_type.is_customer_specific

    def test_customer_specific_type(self):
        """Test customer-specific document type."""
        doc_type = DocumentType(
            base_type="BOL",
            customer_id="fedex",
            template_id="standard"
        )

        assert doc_type.is_customer_specific
        assert doc_type.full_type == "fedex_BOL_standard"


@pytest.mark.integration
class TestClassificationIntegration:
    """Integration tests for classification workflow."""

    def test_full_workflow(self, classification_service, sample_image):
        """Test complete classification workflow."""
        # 1. Classify without customer (cold start)
        result1 = classification_service.classify_document(sample_image)
        assert len(result1["predictions"]) > 0
        assert result1["has_customer_match"] is False

        # 2. Learn customer template
        learn_result = classification_service.learn_customer_template(
            image=sample_image,
            customer_id="fedex",
            document_type="BOL"
        )
        assert learn_result["success"]

        # 3. Classify same document with customer ID
        result2 = classification_service.classify_document(
            sample_image,
            customer_id="fedex"
        )
        # May have customer match if similarity is high
        assert result2["customer_id"] == "fedex"

        # 4. Get customer info
        customer_info = classification_service.get_customer_templates("fedex")
        assert customer_info["total_templates"] == 1

    def test_multiple_customers_workflow(self, classification_service, sample_image_batch):
        """Test workflow with multiple customers."""
        customers = ["fedex", "ups", "dhl"]

        # Learn templates for each customer
        for customer_id, image in zip(customers, sample_image_batch[:3]):
            classification_service.learn_customer_template(
                image=image,
                customer_id=customer_id,
                document_type="BOL"
            )

        # Get all customers
        all_customers = classification_service.get_all_customers()
        assert len(all_customers) == 3

        # Classify with each customer
        for customer_id in customers:
            result = classification_service.classify_document(
                sample_image_batch[0],
                customer_id=customer_id
            )
            assert result["customer_id"] == customer_id
