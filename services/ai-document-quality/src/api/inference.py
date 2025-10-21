import logging
import time
from pathlib import Path
from typing import Dict, List, Optional, Tuple

import numpy as np
import torch
import torch.nn as nn
import torchvision.transforms as transforms
import yaml
from PIL import Image

from ..models.model import DocumentQualityModel, ModelConfig

logger = logging.getLogger(__name__)


class DocumentQualityPredictor:
    ISSUE_NAMES = [
        "blur",
        "noise",
        "lighting",
        "shadow",
        "physical_damage",
        "skew",
        "partial_capture",
        "glare",
        "compression",
        "overall_poor",
    ]

    CLASS_NAMES = ["High", "Good", "Moderate", "Poor", "Very Poor"]

    SEVERITY_THRESHOLDS = {
        "critical": 0.7,  # > 0.7 probability = critical
        "moderate": 0.5,  # 0.5-0.7 = moderate
        "minor": 0.3,  # 0.3-0.5 = minor
    }

    def __init__(
        self,
        model_path: str,
        config_path: Optional[str] = None,
        device: Optional[str] = None,
        enable_performance_monitoring: bool = True,
    ):
        self.model_path = Path(model_path)
        self.config_path = Path(config_path) if config_path else None
        self.enable_monitoring = enable_performance_monitoring

        if device is None or device == "auto":
            self.device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
        else:
            self.device = torch.device(device)

        logger.info(f"Initializing predictor on device: {self.device}")

        self.config = self._load_config()
        self.model = self._load_model()
        self.model.eval()
        self.transform = self._get_transform()
        self.request_count = 0
        self.processing_times = []
        self.quality_scores = []
        self.acceptance_decisions = []
        self.error_count = 0
        self.start_time = time.time()

        logger.info("✓ Predictor initialized successfully")

    def _load_config(self) -> Dict:
        """Load configuration"""
        if self.config_path and self.config_path.exists():
            with open(self.config_path, "r") as f:
                return yaml.safe_load(f)
        else:
            return {
                "model": {
                    "backbone": "efficientnet_b0",
                    "num_quality_classes": 5,
                    "num_issue_classes": 10,
                    "hidden_dim": 256,
                    "dropout_rate": 0.5,
                    "use_attention": True,
                }
            }

    def _load_model(self) -> nn.Module:
        """Load model from checkpoint"""
        logger.info(f"Loading model from {self.model_path}")

        model_config = ModelConfig(
            backbone=self.config.get("model", {}).get("backbone", "efficientnet_b0"),
            num_quality_classes=self.config.get("model", {}).get(
                "num_quality_classes", 5
            ),
            num_issue_classes=self.config.get("model", {}).get("num_issue_classes", 10),
            hidden_dim=self.config.get("model", {}).get("hidden_dim", 256),
            dropout_rate=self.config.get("model", {}).get("dropout_rate", 0.5),
            use_attention=self.config.get("model", {}).get("use_attention", True),
        )

        model = DocumentQualityModel(model_config)
        checkpoint = torch.load(self.model_path, map_location=self.device)

        if isinstance(checkpoint, dict):
            if "model_state_dict" in checkpoint:
                model.load_state_dict(checkpoint["model_state_dict"])
            elif "state_dict" in checkpoint:
                model.load_state_dict(checkpoint["state_dict"])
            else:
                model.load_state_dict(checkpoint)
        else:
            model.load_state_dict(checkpoint)

        model = model.to(self.device)
        logger.info("✓ Model loaded successfully")

        return model

    def _get_transform(self):
        """Get image transformation pipeline"""
        return transforms.Compose(
            [
                transforms.Resize((224, 224)),
                transforms.ToTensor(),
                transforms.Normalize(
                    mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]
                ),
            ]
        )

    def predict(
        self, image: Image.Image, threshold: float = 0.5, include_issues: bool = True
    ) -> Dict:
        start_time = time.time()

        try:
            if image.mode != "RGB":
                image = image.convert("RGB")

            input_tensor = self.transform(image).unsqueeze(0).to(self.device)

            with torch.no_grad():
                outputs = self.model(input_tensor)

            quality_score = outputs["quality_score"].item()
            quality_class_logits = outputs["quality_class_logits"][0]
            quality_class_probs = torch.softmax(quality_class_logits, dim=0)
            quality_class_idx = torch.argmax(quality_class_probs).item()
            quality_class_confidence = quality_class_probs[quality_class_idx].item()

            issue_probs = torch.sigmoid(outputs["issue_logits"][0]).cpu().numpy()
            is_acceptable = quality_score >= threshold

            result = {
                "quality_score": float(quality_score),
                "quality_class": self.CLASS_NAMES[quality_class_idx],
                "quality_class_index": int(quality_class_idx),
                "is_acceptable": bool(is_acceptable),
                "confidence": float(quality_class_confidence),
                "class_probabilities": {
                    name: float(prob)
                    for name, prob in zip(
                        self.CLASS_NAMES, quality_class_probs.cpu().numpy()
                    )
                },
            }

            if include_issues:
                result["issues"] = self._extract_issues(issue_probs)
                result["recommendations"] = self._generate_recommendations(
                    quality_score, is_acceptable, issue_probs, threshold
                )

            processing_time = (time.time() - start_time) * 1000
            if self.enable_monitoring:
                self._update_metrics(quality_score, is_acceptable, processing_time)

            result["processing_time_ms"] = processing_time

            return result

        except Exception as e:
            logger.error(f"Prediction error: {str(e)}")
            if self.enable_monitoring:
                self.error_count += 1
            raise

    def predict_batch(
        self,
        images: List[Image.Image],
        threshold: float = 0.5,
        include_issues: bool = True,
    ) -> List[Dict]:
        start_time = time.time()

        try:
            image_tensors = []
            for image in images:
                if image.mode != "RGB":
                    image = image.convert("RGB")
                image_tensors.append(self.transform(image))

            batch_tensor = torch.stack(image_tensors).to(self.device)

            with torch.no_grad():
                outputs = self.model(batch_tensor)

            results = []
            for i in range(len(images)):
                quality_score = outputs["quality_score"][i].item()
                quality_class_logits = outputs["quality_class_logits"][i]
                quality_class_probs = torch.softmax(quality_class_logits, dim=0)
                quality_class_idx = torch.argmax(quality_class_probs).item()
                quality_class_confidence = quality_class_probs[quality_class_idx].item()

                issue_probs = torch.sigmoid(outputs["issue_logits"][i]).cpu().numpy()

                is_acceptable = quality_score >= threshold

                result = {
                    "quality_score": float(quality_score),
                    "quality_class": self.CLASS_NAMES[quality_class_idx],
                    "quality_class_index": int(quality_class_idx),
                    "is_acceptable": bool(is_acceptable),
                    "confidence": float(quality_class_confidence),
                }

                if include_issues:
                    result["issues"] = self._extract_issues(issue_probs)
                    result["recommendations"] = self._generate_recommendations(
                        quality_score, is_acceptable, issue_probs, threshold
                    )

                results.append(result)

            processing_time = (time.time() - start_time) * 1000
            for result in results:
                if self.enable_monitoring:
                    self._update_metrics(
                        result["quality_score"],
                        result["is_acceptable"],
                        processing_time / len(images),
                    )

            return results

        except Exception as e:
            logger.error(f"Batch prediction error: {str(e)}")
            if self.enable_monitoring:
                self.error_count += 1
            raise

    def _extract_issues(
        self, issue_probs: np.ndarray, min_probability: float = 0.3
    ) -> List[Dict]:
        """Extract detected issues with severity classification"""
        issues = []

        for issue_name, prob in zip(self.ISSUE_NAMES, issue_probs):
            if prob >= min_probability:
                if prob >= self.SEVERITY_THRESHOLDS["critical"]:
                    severity = "critical"
                elif prob >= self.SEVERITY_THRESHOLDS["moderate"]:
                    severity = "moderate"
                else:
                    severity = "minor"

                issues.append(
                    {
                        "issue_type": issue_name,
                        "probability": float(prob),
                        "severity": severity,
                    }
                )

        issues.sort(key=lambda x: x["probability"], reverse=True)

        return issues

    def _generate_recommendations(
        self,
        quality_score: float,
        is_acceptable: bool,
        issue_probs: np.ndarray,
        threshold: float,
    ) -> List[str]:
        recommendations = []

        if is_acceptable:
            recommendations.append(
                f"✓ Document quality is acceptable (score: {quality_score:.3f})"
            )
            recommendations.append("Document is suitable for processing")
        else:
            recommendations.append(
                f"⚠ Document quality score ({quality_score:.3f}) is below threshold ({threshold})"
            )

            issue_dict = dict(zip(self.ISSUE_NAMES, issue_probs))

            if issue_dict.get("blur", 0) > 0.5:
                recommendations.append(
                    "• Image appears blurry - ensure camera is focused before capture"
                )
            if issue_dict.get("partial_capture", 0) > 0.5:
                recommendations.append(
                    "• Document appears partially cut off - capture the entire document"
                )
            if issue_dict.get("lighting", 0) > 0.5:
                recommendations.append(
                    "• Lighting issues detected - ensure even, adequate lighting"
                )
            if issue_dict.get("shadow", 0) > 0.5:
                recommendations.append(
                    "• Shadows detected - avoid shadows on the document"
                )
            if issue_dict.get("glare", 0) > 0.5:
                recommendations.append(
                    "• Glare detected - avoid reflections and direct light sources"
                )
            if issue_dict.get("skew", 0) > 0.5:
                recommendations.append(
                    "• Document appears skewed - hold device parallel to document"
                )
            if issue_dict.get("noise", 0) > 0.5:
                recommendations.append(
                    "• Image noise detected - improve lighting conditions"
                )
            if issue_dict.get("physical_damage", 0) > 0.5:
                recommendations.append(
                    "• Physical damage detected - use undamaged original if possible"
                )

            if len(recommendations) == 1:
                recommendations.append(
                    "• Please retake the photo following capture guidelines"
                )

        return recommendations

    def _update_metrics(
        self, quality_score: float, is_acceptable: bool, processing_time: float
    ):
        self.request_count += 1
        self.processing_times.append(processing_time)
        self.quality_scores.append(quality_score)
        self.acceptance_decisions.append(is_acceptable)

        if len(self.processing_times) > 10000:
            self.processing_times = self.processing_times[-10000:]
            self.quality_scores = self.quality_scores[-10000:]
            self.acceptance_decisions = self.acceptance_decisions[-10000:]

    def get_metrics(self) -> Dict:
        if not self.processing_times:
            return {
                "total_requests": 0,
                "average_processing_time_ms": 0.0,
                "p50_processing_time_ms": 0.0,
                "p95_processing_time_ms": 0.0,
                "p99_processing_time_ms": 0.0,
                "acceptance_rate": 0.0,
                "rejection_rate": 0.0,
                "average_quality_score": 0.0,
                "errors": self.error_count,
            }

        processing_times_arr = np.array(self.processing_times)
        quality_scores_arr = np.array(self.quality_scores)
        acceptance_arr = np.array(self.acceptance_decisions)

        return {
            "total_requests": self.request_count,
            "average_processing_time_ms": float(np.mean(processing_times_arr)),
            "p50_processing_time_ms": float(np.percentile(processing_times_arr, 50)),
            "p95_processing_time_ms": float(np.percentile(processing_times_arr, 95)),
            "p99_processing_time_ms": float(np.percentile(processing_times_arr, 99)),
            "acceptance_rate": float(np.mean(acceptance_arr)),
            "rejection_rate": float(1.0 - np.mean(acceptance_arr)),
            "average_quality_score": float(np.mean(quality_scores_arr)),
            "errors": self.error_count,
        }

    def get_uptime(self) -> float:
        return time.time() - self.start_time

    def reset_metrics(self):
        self.request_count = 0
        self.processing_times = []
        self.quality_scores = []
        self.acceptance_decisions = []
        self.error_count = 0
        logger.info("Metrics reset")
