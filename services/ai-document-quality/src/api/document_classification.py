import time
from typing import Dict, List, Optional
from pathlib import Path
import torch
from PIL import Image
import torchvision.transforms as T

from models.document_classifier import DocumentTypeClassifier, STANDARD_DOCUMENT_TYPES


class DocumentClassificationService:
    def __init__(
        self,
        model_path: Optional[str] = None,
        template_bank_path: Optional[str] = None,
        device: Optional[torch.device] = None,
    ):
        self.device = device or torch.device(
            "cuda" if torch.cuda.is_available() else "cpu"
        )

        self.model = DocumentTypeClassifier(
            num_base_types=len(STANDARD_DOCUMENT_TYPES),
            feature_dim=512,
            backbone="efficientnet_b0",
            pretrained=True,
        )

        if model_path and Path(model_path).exists():
            checkpoint = torch.load(model_path, map_location=self.device)
            if "model_state_dict" in checkpoint:
                self.model.load_state_dict(checkpoint["model_state_dict"])
            else:
                self.model.load_state_dict(checkpoint)

        if template_bank_path and Path(template_bank_path).exists():
            self.model.template_bank.load(template_bank_path)

        self.model.to(self.device)
        self.model.eval()

        self.transform = T.Compose(
            [
                T.Resize((224, 224)),
                T.ToTensor(),
                T.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),
            ]
        )

        self.total_classifications = 0
        self.total_template_learnings = 0
        self.total_inference_time = 0.0
        self.customer_classification_counts = {}

    def preprocess_image(self, image: Image.Image) -> torch.Tensor:
        if image.mode != "RGB":
            image = image.convert("RGB")

        tensor = self.transform(image)
        return tensor.unsqueeze(0).to(self.device)

    def classify_document(
        self,
        image: Image.Image,
        customer_id: Optional[str] = None,
        top_k: int = 3,
        confidence_threshold: float = 0.6,
        include_features: bool = False,
    ) -> Dict:
        start_time = time.time()

        tensor = self.preprocess_image(image)
        predictions = self.model.classify(
            x=tensor,
            customer_id=customer_id,
            return_top_k=top_k,
            confidence_threshold=confidence_threshold,
        )
        inference_time = time.time() - start_time

        self.total_classifications += 1
        self.total_inference_time += inference_time
        if customer_id:
            self.customer_classification_counts[customer_id] = (
                self.customer_classification_counts.get(customer_id, 0) + 1
            )

        result = {
            "predictions": predictions,
            "customer_id": customer_id,
            "num_predictions": len(predictions),
            "best_prediction": predictions[0] if predictions else None,
            "inference_time": inference_time,
            "has_customer_match": (
                any(p["source"] == "customer_template" for p in predictions)
                if customer_id
                else False
            ),
        }

        if include_features:
            with torch.no_grad():
                outputs = self.model.forward(tensor)
                features = outputs["features"][0].cpu().numpy().tolist()
                result["features"] = features

        return result

    def classify_batch(
        self,
        images: List[Image.Image],
        customer_ids: Optional[List[str]] = None,
        top_k: int = 3,
        confidence_threshold: float = 0.6,
    ) -> List[Dict]:
        if customer_ids is None:
            customer_ids = [None] * len(images)

        if len(customer_ids) != len(images):
            raise ValueError("customer_ids must have same length as images")

        results = []
        for image, customer_id in zip(images, customer_ids):
            result = self.classify_document(
                image=image,
                customer_id=customer_id,
                top_k=top_k,
                confidence_threshold=confidence_threshold,
            )
            results.append(result)

        return results

    def learn_customer_template(
        self,
        image: Image.Image,
        customer_id: str,
        document_type: str,
        template_id: Optional[str] = None,
        metadata: Optional[Dict] = None,
    ) -> Dict:
        start_time = time.time()

        if document_type not in STANDARD_DOCUMENT_TYPES:
            raise ValueError(
                f"Invalid document type: {document_type}. "
                f"Must be one of {list(STANDARD_DOCUMENT_TYPES.keys())}"
            )

        tensor = self.preprocess_image(image)
        self.model.learn_customer_template(
            image=tensor,
            customer_id=customer_id,
            doc_type=document_type,
            template_id=template_id,
            metadata=metadata,
        )

        learning_time = time.time() - start_time

        self.total_template_learnings += 1

        customer_info = self.model.get_customer_info(customer_id)

        return {
            "success": True,
            "customer_id": customer_id,
            "document_type": document_type,
            "template_id": template_id or "default",
            "learning_time": learning_time,
            "customer_templates": customer_info["template_counts"],
            "total_templates": customer_info["total_templates"],
        }

    def get_customer_templates(self, customer_id: str) -> Dict:
        info = self.model.get_customer_info(customer_id)

        return {
            "customer_id": customer_id,
            "document_types": info["document_types"],
            "template_counts": info["template_counts"],
            "total_templates": info["total_templates"],
            "has_templates": info["total_templates"] > 0,
            "classification_count": self.customer_classification_counts.get(
                customer_id, 0
            ),
        }

    def get_all_customers(self) -> List[Dict]:
        customers = self.model.template_bank.get_all_customers()
        return [self.get_customer_templates(customer_id) for customer_id in customers]

    def get_supported_document_types(self) -> Dict[str, str]:
        return dict(STANDARD_DOCUMENT_TYPES)

    def save_template_bank(self, path: str):
        self.model.template_bank.save(path)

    def get_metrics(self) -> Dict:
        return {
            "total_classifications": self.total_classifications,
            "total_template_learnings": self.total_template_learnings,
            "total_inference_time": self.total_inference_time,
            "average_inference_time": (
                self.total_inference_time / self.total_classifications
                if self.total_classifications > 0
                else 0.0
            ),
            "unique_customers": len(self.customer_classification_counts),
            "customer_classification_counts": dict(self.customer_classification_counts),
            "total_customer_templates": sum(
                info["total_templates"] for info in self.get_all_customers()
            ),
        }

    def reset_metrics(self):
        self.total_classifications = 0
        self.total_template_learnings = 0
        self.total_inference_time = 0.0
        self.customer_classification_counts = {}


# Global service instance (initialized by FastAPI app)
classification_service: Optional[DocumentClassificationService] = None


def get_classification_service() -> DocumentClassificationService:
    if classification_service is None:
        raise RuntimeError("Classification service not initialized")
    return classification_service


def initialize_classification_service(
    model_path: Optional[str] = None,
    template_bank_path: Optional[str] = None,
    device: Optional[torch.device] = None,
) -> DocumentClassificationService:
    global classification_service
    classification_service = DocumentClassificationService(
        model_path=model_path, template_bank_path=template_bank_path, device=device
    )
    return classification_service
