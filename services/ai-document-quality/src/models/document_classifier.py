import torch
import torch.nn as nn
import torch.nn.functional as F
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass
from collections import defaultdict


@dataclass
class DocumentType:

    base_type: str  # BOL, Invoice, Receipt, POD, etc.
    customer_id: Optional[str] = None
    template_id: Optional[str] = None
    confidence_threshold: float = 0.7

    @property
    def full_type(self) -> str:
        """Get full type identifier."""
        if self.customer_id and self.template_id:
            return f"{self.customer_id}_{self.base_type}_{self.template_id}"
        return self.base_type

    @property
    def is_customer_specific(self) -> bool:
        """Check if this is a customer-specific type."""
        return self.customer_id is not None


# Standard document types in transportation/logistics
STANDARD_DOCUMENT_TYPES = {
    "BOL": "Bill of Lading - Primary shipping document",
    "INVOICE": "Freight Invoice - Payment document",
    "RECEIPT": "Delivery Receipt - Proof of goods received",
    "POD": "Proof of Delivery - Signed delivery confirmation",
    "RATE_CONF": "Rate Confirmation - Agreed shipping rates",
    "LUMPER": "Lumper Receipt - Unloading service receipt",
    "FUEL": "Fuel Receipt - Fuel purchase receipt",
    "SCALE": "Scale Ticket - Weight verification",
    "INSPECTION": "Inspection Report - Vehicle/cargo inspection",
    "OTHER": "Other Document - Unclassified",
}


class DocumentTypeEncoder(nn.Module):
    def __init__(
        self,
        backbone: str = "efficientnet_b0",
        feature_dim: int = 512,
        pretrained: bool = True,
    ):
        super().__init__()

        if backbone == "efficientnet_b0":
            from torchvision.models import efficientnet_b0, EfficientNet_B0_Weights

            weights = EfficientNet_B0_Weights.DEFAULT if pretrained else None
            self.backbone = efficientnet_b0(weights=weights)
            backbone_features = 1280
        else:
            raise ValueError(f"Unsupported backbone: {backbone}")

        self.backbone.classifier = nn.Identity()

        self.feature_projection = nn.Sequential(
            nn.Linear(backbone_features, feature_dim),
            nn.ReLU(inplace=True),
            nn.Dropout(0.3),
            nn.Linear(feature_dim, feature_dim),
        )

        self.normalize = True

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        features = self.backbone(x)  # [batch_size, backbone_features]

        features = self.feature_projection(features)  # [batch_size, feature_dim]

        if self.normalize:
            features = F.normalize(features, p=2, dim=1)

        return features


class CustomerTemplateBank:
    def __init__(self, feature_dim: int = 512):
        self.feature_dim = feature_dim

        self.templates: Dict[str, Dict[str, List[torch.Tensor]]] = defaultdict(
            lambda: defaultdict(list)
        )

        self.template_metadata: Dict[str, Dict[str, List[Dict]]] = defaultdict(
            lambda: defaultdict(list)
        )

        self.template_counts: Dict[str, Dict[str, int]] = defaultdict(
            lambda: defaultdict(int)
        )

    def add_template(
        self,
        customer_id: str,
        doc_type: str,
        features: torch.Tensor,
        metadata: Optional[Dict] = None,
    ):
        if features.dim() == 1:
            features = features.unsqueeze(0)

        self.templates[customer_id][doc_type].append(features.cpu())
        self.template_metadata[customer_id][doc_type].append(metadata or {})
        self.template_counts[customer_id][doc_type] += 1

    def find_similar_templates(
        self,
        customer_id: str,
        features: torch.Tensor,
        top_k: int = 3,
        threshold: float = 0.7,
    ) -> List[Tuple[str, float, Dict]]:
        if customer_id not in self.templates:
            return []

        results = []

        for doc_type, template_list in self.templates[customer_id].items():
            if not template_list:
                continue

            template_features = torch.cat(
                template_list, dim=0
            )  # [num_templates, feature_dim]

            if features.dim() == 1:
                features = features.unsqueeze(0)

            similarities = F.cosine_similarity(
                features.unsqueeze(1),  # [1, 1, feature_dim]
                template_features.unsqueeze(0),  # [1, num_templates, feature_dim]
                dim=2,
            ).squeeze(
                0
            )  # [num_templates]

            max_sim, max_idx = similarities.max(dim=0)

            if max_sim.item() >= threshold:
                metadata = self.template_metadata[customer_id][doc_type][max_idx.item()]
                results.append((doc_type, max_sim.item(), metadata))

        results.sort(key=lambda x: x[1], reverse=True)

        return results[:top_k]

    def get_customer_stats(self, customer_id: str) -> Dict[str, int]:
        return dict(self.template_counts.get(customer_id, {}))

    def get_all_customers(self) -> List[str]:
        return list(self.templates.keys())

    def save(self, path: str):
        state = {
            "templates": {
                cust_id: {
                    doc_type: [t.cpu() for t in templates]
                    for doc_type, templates in doc_types.items()
                }
                for cust_id, doc_types in self.templates.items()
            },
            "metadata": dict(self.template_metadata),
            "counts": dict(self.template_counts),
            "feature_dim": self.feature_dim,
        }
        torch.save(state, path)

    def load(self, path: str):
        state = torch.load(path, map_location="cpu")

        self.templates = defaultdict(lambda: defaultdict(list))
        for cust_id, doc_types in state["templates"].items():
            for doc_type, templates in doc_types.items():
                self.templates[cust_id][doc_type] = templates

        self.template_metadata = defaultdict(
            lambda: defaultdict(list), state["metadata"]
        )
        self.template_counts = defaultdict(lambda: defaultdict(int), state["counts"])
        self.feature_dim = state["feature_dim"]


class DocumentTypeClassifier(nn.Module):
    def __init__(
        self,
        num_base_types: int = len(STANDARD_DOCUMENT_TYPES),
        feature_dim: int = 512,
        backbone: str = "efficientnet_b0",
        pretrained: bool = True,
    ):
        super().__init__()

        self.num_base_types = num_base_types
        self.feature_dim = feature_dim

        self.encoder = DocumentTypeEncoder(
            backbone=backbone, feature_dim=feature_dim, pretrained=pretrained
        )

        self.base_classifier = nn.Sequential(
            nn.Linear(feature_dim, 256),
            nn.ReLU(inplace=True),
            nn.Dropout(0.3),
            nn.Linear(256, num_base_types),
        )

        self.template_bank = CustomerTemplateBank(feature_dim=feature_dim)

        self.base_type_labels = list(STANDARD_DOCUMENT_TYPES.keys())

    def forward(self, x: torch.Tensor) -> Dict[str, torch.Tensor]:
        features = self.encoder(x)

        base_logits = self.base_classifier(features)
        base_probs = F.softmax(base_logits, dim=1)

        return {
            "features": features,
            "base_logits": base_logits,
            "base_probs": base_probs,
        }

    def classify(
        self,
        x: torch.Tensor,
        customer_id: Optional[str] = None,
        return_top_k: int = 3,
        confidence_threshold: float = 0.6,
    ) -> List[Dict]:
        if x.dim() == 3:
            x = x.unsqueeze(0)

        self.eval()
        with torch.no_grad():
            outputs = self.forward(x)
            features = outputs["features"][0]  # [feature_dim]
            base_probs = outputs["base_probs"][0]  # [num_base_types]

        base_confidences, base_indices = base_probs.topk(return_top_k)

        predictions = []

        for idx, conf in zip(base_indices, base_confidences):
            doc_type = self.base_type_labels[idx.item()]
            predictions.append(
                {
                    "document_type": doc_type,
                    "base_type": doc_type,
                    "customer_id": None,
                    "confidence": conf.item(),
                    "source": "base_classifier",
                    "description": STANDARD_DOCUMENT_TYPES[doc_type],
                }
            )

        if customer_id:
            customer_matches = self.template_bank.find_similar_templates(
                customer_id=customer_id,
                features=features,
                top_k=return_top_k,
                threshold=confidence_threshold,
            )

            for doc_type, similarity, metadata in customer_matches:
                predictions.append(
                    {
                        "document_type": f"{customer_id}_{doc_type}_{metadata.get('template_id', 'default')}",
                        "base_type": doc_type,
                        "customer_id": customer_id,
                        "confidence": similarity,
                        "source": "customer_template",
                        "template_metadata": metadata,
                        "description": f"Customer-specific {doc_type}",
                    }
                )

        predictions.sort(key=lambda x: x["confidence"], reverse=True)

        predictions = [
            p for p in predictions if p["confidence"] >= confidence_threshold
        ]

        return predictions[:return_top_k]

    def learn_customer_template(
        self,
        image: torch.Tensor,
        customer_id: str,
        doc_type: str,
        template_id: Optional[str] = None,
        metadata: Optional[Dict] = None,
    ):
        if image.dim() == 3:
            image = image.unsqueeze(0)

        self.eval()
        with torch.no_grad():
            outputs = self.forward(image)
            features = outputs["features"][0]  # [feature_dim]

        meta = metadata or {}
        meta["template_id"] = template_id or "default"

        self.template_bank.add_template(
            customer_id=customer_id, doc_type=doc_type, features=features, metadata=meta
        )

    def get_customer_info(self, customer_id: str) -> Dict:
        stats = self.template_bank.get_customer_stats(customer_id)
        return {
            "customer_id": customer_id,
            "template_counts": stats,
            "total_templates": sum(stats.values()),
            "document_types": list(stats.keys()),
        }
