import logging
from pathlib import Path
from typing import Dict, List, Optional, Tuple, Any

import cv2
import matplotlib.pyplot as plt
import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F
from PIL import Image

logger = logging.getLogger(__name__)


class GradCAM:
    def __init__(self, model: nn.Module, target_layer: nn.Module):
        self.model = model
        self.model = model
        self.target_layer = target_layer
        self.gradients = None
        self.activations = None
        self.target_layer.register_forward_hook(self._save_activation)
        self.target_layer.register_backward_hook(self._save_gradient)

    def _save_activation(self, module, input, output):
        """Hook to save forward activation"""
        self.activations = output.detach()

    def _save_gradient(self, module, grad_input, grad_output):
        """Hook to save backward gradient"""
        self.gradients = grad_output[0].detach()

    def generate_cam(
        self, input_image: torch.Tensor, target_score: Optional[torch.Tensor] = None
    ) -> np.ndarray:
        self.model.eval()

        output = self.model(input_image)

        if target_score is None:
            target_score = output["quality_score"]

        self.model.zero_grad()

        target_score.backward(retain_graph=True)

        gradients = self.gradients  # [1, C, H, W]
        activations = self.activations  # [1, C, H, W]

        weights = torch.mean(gradients, dim=(2, 3), keepdim=True)  # [1, C, 1, 1]

        cam = torch.sum(weights * activations, dim=1, keepdim=True)  # [1, 1, H, W]
        cam = F.relu(cam)
        cam = cam.squeeze().cpu().numpy()
        cam = (cam - cam.min()) / (cam.max() - cam.min() + 1e-8)

        return cam

    def generate_multi_cam(self, input_image: torch.Tensor) -> Dict[str, np.ndarray]:
        cams = {}

        cam_quality = self.generate_cam(input_image)
        cams["quality_score"] = cam_quality

        output = self.model(input_image)
        issue_probs = torch.sigmoid(output["issue_logits"])
        top_issue_idx = torch.argmax(issue_probs)
        top_issue_score = issue_probs[0, top_issue_idx]

        self.model.zero_grad()
        cam_issue = self.generate_cam(input_image, target_score=top_issue_score)
        cams["top_issue"] = cam_issue

        return cams


def overlay_heatmap(
    image: np.ndarray,
    heatmap: np.ndarray,
    alpha: float = 0.4,
    colormap: int = cv2.COLORMAP_JET,
) -> np.ndarray:
    heatmap_resized = cv2.resize(heatmap, (image.shape[1], image.shape[0]))

    heatmap_uint8 = (heatmap_resized * 255).astype(np.uint8)

    heatmap_colored = cv2.applyColorMap(heatmap_uint8, colormap)

    heatmap_colored = cv2.cvtColor(heatmap_colored, cv2.COLOR_BGR2RGB)

    if image.dtype != np.uint8:
        image = (image * 255).astype(np.uint8)

    overlaid = cv2.addWeighted(image, 1 - alpha, heatmap_colored, alpha, 0)

    return overlaid


def visualize_explanation(
    image: Image.Image,
    model: nn.Module,
    transform: Any,
    target_layer: nn.Module,
    save_path: Optional[Path] = None,
    quality_threshold: float = 0.5,
) -> Tuple[Dict, Image.Image]:
    device = next(model.parameters()).device
    model.eval()

    original_array = np.array(image)
    input_tensor = transform(image).unsqueeze(0).to(device)

    with torch.no_grad():
        outputs = model(input_tensor)

    quality_score = outputs["quality_score"].item()
    quality_class_logits = outputs["quality_class_logits"][0]
    quality_class = torch.argmax(quality_class_logits).item()
    issue_logits = outputs["issue_logits"][0]
    issue_probs = torch.sigmoid(issue_logits).cpu().numpy()

    class_names = ["High", "Good", "Moderate", "Poor", "Very Poor"]
    issue_names = [
        "Blur",
        "Noise",
        "Lighting",
        "Shadow",
        "Physical Damage",
        "Skew",
        "Partial",
        "Glare",
        "Compression",
        "Overall Poor",
    ]

    is_acceptable = quality_score >= quality_threshold

    grad_cam = GradCAM(model, target_layer)
    cam_quality = grad_cam.generate_cam(input_tensor)

    fig = plt.figure(figsize=(16, 10))
    gs = fig.add_gridspec(3, 3, hspace=0.3, wspace=0.3)

    ax1 = fig.add_subplot(gs[0, 0])
    ax1.imshow(original_array)
    ax1.set_title("Original Document", fontsize=12, fontweight="bold")
    ax1.axis("off")

    ax2 = fig.add_subplot(gs[0, 1])
    overlaid = overlay_heatmap(original_array, cam_quality)
    ax2.imshow(overlaid)
    ax2.set_title(
        "Quality Assessment Focus\n(Grad-CAM)", fontsize=12, fontweight="bold"
    )
    ax2.axis("off")

    ax3 = fig.add_subplot(gs[0, 2])
    im = ax3.imshow(cam_quality, cmap="jet")
    ax3.set_title("Attention Heatmap", fontsize=12, fontweight="bold")
    ax3.axis("off")
    plt.colorbar(im, ax=ax3, fraction=0.046)

    ax4 = fig.add_subplot(gs[1, 0])
    ax4.text(
        0.5,
        0.7,
        f"Quality Score",
        ha="center",
        va="center",
        fontsize=14,
        fontweight="bold",
    )
    ax4.text(
        0.5,
        0.5,
        f"{quality_score:.3f}",
        ha="center",
        va="center",
        fontsize=48,
        fontweight="bold",
        color="green" if is_acceptable else "red",
    )
    ax4.text(
        0.5,
        0.3,
        f"Class: {class_names[quality_class]}",
        ha="center",
        va="center",
        fontsize=12,
    )
    ax4.text(
        0.5,
        0.2,
        "✓ ACCEPTABLE" if is_acceptable else "✗ REJECTED",
        ha="center",
        va="center",
        fontsize=14,
        fontweight="bold",
        color="green" if is_acceptable else "red",
    )
    ax4.set_xlim([0, 1])
    ax4.set_ylim([0, 1])
    ax4.axis("off")

    ax5 = fig.add_subplot(gs[1, 1])
    class_probs = torch.softmax(quality_class_logits, dim=0).cpu().numpy()
    y_pos = np.arange(len(class_names))
    colors = ["#1a9850", "#91cf60", "#fee08b", "#fc8d59", "#d73027"]
    bars = ax5.barh(y_pos, class_probs, color=colors, alpha=0.7)

    bars[quality_class].set_edgecolor("black")
    bars[quality_class].set_linewidth(3)

    ax5.set_yticks(y_pos)
    ax5.set_yticklabels(class_names)
    ax5.set_xlabel("Probability", fontsize=11)
    ax5.set_title("Quality Class Probabilities", fontsize=12, fontweight="bold")
    ax5.set_xlim([0, 1])
    ax5.grid(axis="x", alpha=0.3)

    ax6 = fig.add_subplot(gs[1, 2])

    significant_issues = [
        (name, prob) for name, prob in zip(issue_names, issue_probs) if prob > 0.3
    ]
    significant_issues.sort(key=lambda x: x[1], reverse=True)

    if significant_issues:
        issue_names_sig, issue_probs_sig = zip(*significant_issues)
        y_pos_issues = np.arange(len(issue_names_sig))
        bars_issues = ax6.barh(y_pos_issues, issue_probs_sig, color="coral", alpha=0.7)

        ax6.set_yticks(y_pos_issues)
        ax6.set_yticklabels(issue_names_sig)
        ax6.set_xlabel("Probability", fontsize=11)
        ax6.set_title("Detected Issues (prob > 0.3)", fontsize=12, fontweight="bold")
        ax6.set_xlim([0, 1])
        ax6.grid(axis="x", alpha=0.3)

        for i, (bar, prob) in enumerate(zip(bars_issues, issue_probs_sig)):
            ax6.text(prob + 0.02, i, f"{prob:.2f}", va="center", fontsize=9)
    else:
        ax6.text(
            0.5,
            0.5,
            "No Significant Issues\nDetected",
            ha="center",
            va="center",
            fontsize=14,
            color="green",
        )
        ax6.set_xlim([0, 1])
        ax6.set_ylim([0, 1])
        ax6.axis("off")

    ax7 = fig.add_subplot(gs[2, :])

    recommendations = []

    if not is_acceptable:
        recommendations.append(
            f"⚠ Document quality score ({quality_score:.3f}) is below threshold ({quality_threshold})"
        )

        if issue_probs[0] > 0.5:  # Blur
            recommendations.append(
                "• Image appears blurry - ensure camera is focused before capture"
            )
        if issue_probs[6] > 0.5:  # Partial
            recommendations.append(
                "• Document appears partially cut off - capture the entire document"
            )
        if issue_probs[2] > 0.5:  # Lighting
            recommendations.append(
                "• Lighting issues detected - ensure even, adequate lighting"
            )
        if issue_probs[3] > 0.5:  # Shadow
            recommendations.append("• Shadows detected - avoid shadows on the document")
        if issue_probs[7] > 0.5:  # Glare
            recommendations.append(
                "• Glare detected - avoid reflections and direct light sources"
            )
        if issue_probs[5] > 0.5:  # Skew
            recommendations.append(
                "• Document appears skewed - hold device parallel to document"
            )

        if not recommendations[1:]:  # Only has the quality warning
            recommendations.append("• Retake the photo following best practices")
    else:
        recommendations.append(
            f"✓ Document quality is acceptable (score: {quality_score:.3f})"
        )
        recommendations.append("• This document is suitable for processing")

    rec_text = "\n".join(recommendations)
    ax7.text(
        0.02,
        0.95,
        "ASSESSMENT & RECOMMENDATIONS:",
        fontsize=13,
        fontweight="bold",
        va="top",
        transform=ax7.transAxes,
    )
    ax7.text(
        0.02,
        0.80,
        rec_text,
        fontsize=11,
        va="top",
        transform=ax7.transAxes,
        family="monospace",
    )
    ax7.axis("off")

    plt.suptitle(
        "Document Quality Assessment - Explanation Report",
        fontsize=16,
        fontweight="bold",
        y=0.98,
    )

    if save_path:
        plt.savefig(save_path, dpi=300, bbox_inches="tight")
        logger.info(f"Saved explanation visualization to {save_path}")

    fig.canvas.draw()
    vis_image = Image.frombytes(
        "RGB", fig.canvas.get_width_height(), fig.canvas.tostring_rgb()
    )
    plt.close(fig)

    predictions = {
        "quality_score": float(quality_score),
        "quality_class": class_names[quality_class],
        "quality_class_idx": int(quality_class),
        "is_acceptable": bool(is_acceptable),
        "class_probabilities": {
            name: float(prob) for name, prob in zip(class_names, class_probs)
        },
        "issues": {
            name: float(prob)
            for name, prob in zip(issue_names, issue_probs)
            if prob > 0.3
        },
        "recommendations": recommendations,
    }

    return predictions, vis_image


def get_target_layer(
    model: nn.Module, backbone_name: str = "efficientnet_b0"
) -> nn.Module:

    if "efficientnet" in backbone_name.lower():
        return model.backbone.features[-2]
    elif "resnet" in backbone_name.lower():
        return model.backbone.layer4
    elif "mobilenet" in backbone_name.lower():
        return model.backbone.features[-2]
    else:
        for name, module in reversed(list(model.named_modules())):
            if isinstance(module, nn.Conv2d):
                logger.info(f"Using layer {name} for Grad-CAM")
                return module

        raise ValueError(f"Could not find suitable target layer for {backbone_name}")


def batch_explain_predictions(
    model: nn.Module,
    image_paths: List[Path],
    transform: Any,
    output_dir: Path,
    target_layer: Optional[nn.Module] = None,
    max_images: int = 20,
) -> None:

    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    if target_layer is None:
        if hasattr(model, "config"):
            target_layer = get_target_layer(model, model.config.backbone)
        else:
            target_layer = get_target_layer(model)

    logger.info(
        f"Generating explanations for {min(len(image_paths), max_images)} images..."
    )

    for i, img_path in enumerate(image_paths[:max_images]):
        try:
            image = Image.open(img_path).convert("RGB")

            save_path = output_dir / f"explanation_{img_path.stem}.png"
            predictions, vis_image = visualize_explanation(
                image, model, transform, target_layer, save_path=save_path
            )

            logger.info(
                f"[{i+1}/{min(len(image_paths), max_images)}] Processed {img_path.name}: "
                f"Score={predictions['quality_score']:.3f}, "
                f"Class={predictions['quality_class']}, "
                f"Acceptable={predictions['is_acceptable']}"
            )

        except Exception as e:
            logger.error(f"Error processing {img_path}: {e}")
            continue

    logger.info(f"Explanations saved to {output_dir}")
