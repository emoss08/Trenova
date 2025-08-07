#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
import logging
from typing import Dict, List, Tuple

import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F
from torch.utils.data import DataLoader, Sampler

logger = logging.getLogger(__name__)


class CurriculumSampler(Sampler):
    """Curriculum learning sampler that gradually increases training difficulty"""

    def __init__(
        self,
        dataset,
        quality_scores: List[float],
        start_percentile: float = 0.3,
        end_percentile: float = 1.0,
        num_epochs: int = 50,
        current_epoch: int = 0,
    ):
        """
        Args:
            dataset: The dataset to sample from
            quality_scores: List of quality scores for each sample
            start_percentile: Initial percentile of easiest samples to use
            end_percentile: Final percentile to use (usually 1.0)
            num_epochs: Total number of training epochs
            current_epoch: Current epoch number
        """
        self.dataset = dataset
        self.quality_scores = np.array(quality_scores)
        self.start_percentile = start_percentile
        self.end_percentile = end_percentile
        self.num_epochs = num_epochs
        self.current_epoch = current_epoch

        # Sort indices by quality (highest quality first - easiest)
        self.sorted_indices = np.argsort(self.quality_scores)[::-1]

    def set_epoch(self, epoch: int):
        """Update the current epoch"""
        self.current_epoch = epoch

    def get_current_percentile(self) -> float:
        """Calculate current percentile based on epoch"""
        if self.num_epochs <= 1:
            return self.end_percentile

        progress = self.current_epoch / (self.num_epochs - 1)
        return (
            self.start_percentile
            + (self.end_percentile - self.start_percentile) * progress
        )

    def __iter__(self):
        """Iterate over indices based on curriculum"""
        percentile = self.get_current_percentile()
        num_samples = int(len(self.dataset) * percentile)

        # Use easiest samples up to current percentile
        available_indices = self.sorted_indices[:num_samples]

        # Shuffle the selected indices
        shuffled = torch.randperm(len(available_indices)).numpy()

        return iter(available_indices[shuffled])

    def __len__(self):
        """Return number of samples in current curriculum"""
        percentile = self.get_current_percentile()
        return int(len(self.dataset) * percentile)


class MixupTrainer:
    """Trainer with Mixup augmentation"""

    def __init__(self, alpha: float = 0.2, cutmix_prob: float = 0.5):
        """
        Args:
            alpha: Mixup interpolation strength
            cutmix_prob: Probability of using CutMix instead of Mixup
        """
        self.alpha = alpha
        self.cutmix_prob = cutmix_prob

    def mixup_data(
        self, x: torch.Tensor, y: Dict[str, torch.Tensor]
    ) -> Tuple[torch.Tensor, Dict[str, torch.Tensor], float]:
        """Apply Mixup augmentation"""
        batch_size = x.size(0)

        if self.alpha > 0:
            lam = np.random.beta(self.alpha, self.alpha)
        else:
            lam = 1

        index = torch.randperm(batch_size).to(x.device)

        # Mix inputs
        mixed_x = lam * x + (1 - lam) * x[index]

        # Mix targets appropriately
        mixed_y = {}
        for key, value in y.items():
            if key in ["quality_score", "quality_scores"]:
                # Interpolate regression targets
                mixed_y[key] = lam * value + (1 - lam) * value[index]
            elif key == "quality_class":
                # For classification, we'll need to handle this in the loss
                mixed_y[key] = value
                mixed_y[f"{key}_mixed"] = value[index]
                mixed_y[f"{key}_lam"] = lam
            elif key == "issues":
                # For multi-label, use maximum
                mixed_y[key] = torch.max(value, value[index])

        return mixed_x, mixed_y, lam

    def cutmix_data(
        self, x: torch.Tensor, y: Dict[str, torch.Tensor]
    ) -> Tuple[torch.Tensor, Dict[str, torch.Tensor], float]:
        """Apply CutMix augmentation"""
        batch_size = x.size(0)

        if self.alpha > 0:
            lam = np.random.beta(self.alpha, self.alpha)
        else:
            lam = 1

        index = torch.randperm(batch_size).to(x.device)

        # Generate random box
        bbx1, bby1, bbx2, bby2 = self.rand_bbox(x.size(), lam)

        # Apply CutMix to input
        x[:, :, bbx1:bbx2, bby1:bby2] = x[index, :, bbx1:bbx2, bby1:bby2]

        # Adjust lambda based on actual box area
        lam = 1 - ((bbx2 - bbx1) * (bby2 - bby1) / (x.size(-1) * x.size(-2)))

        # Mix targets based on area ratio
        mixed_y = {}
        for key, value in y.items():
            if key in ["quality_score", "quality_scores"]:
                mixed_y[key] = lam * value + (1 - lam) * value[index]
            elif key == "quality_class":
                mixed_y[key] = value
                mixed_y[f"{key}_mixed"] = value[index]
                mixed_y[f"{key}_lam"] = lam
            elif key == "issues":
                mixed_y[key] = torch.max(value, value[index])

        return x, mixed_y, lam

    def rand_bbox(self, size, lam):
        """Generate random bounding box for CutMix"""
        W = size[2]
        H = size[3]
        cut_rat = np.sqrt(1.0 - lam)
        cut_w = np.int32(W * cut_rat)
        cut_h = np.int32(H * cut_rat)

        # Uniform
        cx = np.random.randint(W)
        cy = np.random.randint(H)

        bbx1 = np.clip(cx - cut_w // 2, 0, W)
        bby1 = np.clip(cy - cut_h // 2, 0, H)
        bbx2 = np.clip(cx + cut_w // 2, 0, W)
        bby2 = np.clip(cy + cut_h // 2, 0, H)

        return bbx1, bby1, bbx2, bby2

    def __call__(
        self, x: torch.Tensor, y: Dict[str, torch.Tensor]
    ) -> Tuple[torch.Tensor, Dict[str, torch.Tensor], float]:
        """Apply either Mixup or CutMix"""
        if np.random.random() < self.cutmix_prob:
            return self.cutmix_data(x, y)
        else:
            return self.mixup_data(x, y)


class MixupLoss(nn.Module):
    """Loss function that handles mixed targets from Mixup/CutMix"""

    def __init__(self, base_loss):
        super().__init__()
        self.base_loss = base_loss

    def forward(self, predictions: Dict, targets: Dict) -> Dict[str, torch.Tensor]:
        """Calculate loss with mixed targets"""
        # Check if we have mixed classification targets
        if "quality_class_mixed" in targets:
            # Handle mixed classification loss
            lam = targets["quality_class_lam"]

            # Calculate loss for both original and mixed targets
            targets1 = {
                k: v
                for k, v in targets.items()
                if not k.endswith("_mixed") and not k.endswith("_lam")
            }
            losses1 = self.base_loss(predictions, targets1)

            # Create targets for mixed samples
            targets2 = targets1.copy()
            targets2["quality_class"] = targets["quality_class_mixed"]
            losses2 = self.base_loss(predictions, targets2)

            # Combine losses
            mixed_losses = {}
            for key in losses1:
                if key == "classification":
                    mixed_losses[key] = lam * losses1[key] + (1 - lam) * losses2[key]
                else:
                    mixed_losses[key] = losses1[key]  # Other losses are already mixed

            return mixed_losses
        else:
            # No mixing, use base loss
            return self.base_loss(predictions, targets)


class LabelSmoothing(nn.Module):
    """Label smoothing for classification tasks"""

    def __init__(self, smoothing: float = 0.1):
        super().__init__()
        self.smoothing = smoothing

    def forward(self, pred: torch.Tensor, target: torch.Tensor) -> torch.Tensor:
        """Apply label smoothing to cross-entropy loss"""
        n_classes = pred.size(-1)

        # Create smoothed target distribution
        smooth_target = torch.zeros_like(pred)
        smooth_target.fill_(self.smoothing / (n_classes - 1))
        smooth_target.scatter_(1, target.unsqueeze(1), 1 - self.smoothing)

        # Calculate loss
        log_probs = F.log_softmax(pred, dim=-1)
        loss = -(smooth_target * log_probs).sum(dim=-1).mean()

        return loss


class StochasticDepth:
    """Stochastic depth (layer dropout) for training"""

    def __init__(self, drop_rate: float = 0.1):
        self.drop_rate = drop_rate

    def __call__(self, x: torch.Tensor, training: bool = True) -> torch.Tensor:
        """Apply stochastic depth"""
        if not training or self.drop_rate == 0:
            return x

        keep_prob = 1 - self.drop_rate
        mask = torch.empty(x.shape[0], 1, 1, 1, device=x.device).bernoulli_(keep_prob)

        if keep_prob > 0:
            mask.div_(keep_prob)

        return x * mask


class AdversarialTraining:
    """Adversarial training for robustness"""

    def __init__(self, epsilon: float = 0.01, alpha: float = 0.003, num_steps: int = 3):
        """
        Args:
            epsilon: Maximum perturbation magnitude
            alpha: Step size for PGD
            num_steps: Number of PGD steps
        """
        self.epsilon = epsilon
        self.alpha = alpha
        self.num_steps = num_steps

    def pgd_attack(
        self,
        model: nn.Module,
        images: torch.Tensor,
        targets: Dict[str, torch.Tensor],
        loss_fn: nn.Module,
    ) -> torch.Tensor:
        """Generate adversarial examples using PGD"""
        # Initialize perturbation
        delta = torch.zeros_like(images, requires_grad=True)

        # PGD iterations
        for _ in range(self.num_steps):
            # Forward pass
            outputs = model(images + delta)
            loss = loss_fn(outputs, targets)["total"]

            # Backward pass
            loss.backward()

            # Update perturbation
            delta.data = delta + self.alpha * delta.grad.sign()
            delta.data = torch.clamp(delta.data, -self.epsilon, self.epsilon)
            delta.grad.zero_()

        return (images + delta).detach()


def create_curriculum_dataloader(
    dataset, metadata, batch_size: int, num_epochs: int, current_epoch: int = 0
) -> DataLoader:
    """Create a dataloader with curriculum learning"""
    # Extract quality scores from metadata
    quality_scores = metadata["quality_score"].values.tolist()

    # Create curriculum sampler
    sampler = CurriculumSampler(
        dataset=dataset,
        quality_scores=quality_scores,
        start_percentile=0.3,
        end_percentile=1.0,
        num_epochs=num_epochs,
        current_epoch=current_epoch,
    )

    # Create dataloader with curriculum sampler
    return DataLoader(
        dataset, batch_size=batch_size, sampler=sampler, num_workers=4, pin_memory=True
    )


def warm_restart_scheduler(
    optimizer, T_0: int = 10, T_mult: int = 2, eta_min: float = 1e-6
) -> torch.optim.lr_scheduler.CosineAnnealingWarmRestarts:
    """Create cosine annealing with warm restarts scheduler"""
    return torch.optim.lr_scheduler.CosineAnnealingWarmRestarts(
        optimizer, T_0=T_0, T_mult=T_mult, eta_min=eta_min
    )
