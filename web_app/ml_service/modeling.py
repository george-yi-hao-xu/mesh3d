from __future__ import annotations

import math
from pathlib import Path
from typing import Dict, List

import torch
from torch import nn


MODEL_DIR = Path(__file__).resolve().parent / "models"
MODEL_DIR.mkdir(parents=True, exist_ok=True)
SEED = 42


class TabularNet(nn.Module):
    """Small feed-forward network for scoring configs and predicting tags."""

    def __init__(self, input_size: int, tag_count: int):
        super().__init__()
        self.shared = nn.Sequential(
            nn.Linear(input_size, 64),
            nn.ReLU(),
            nn.Linear(64, 32),
            nn.ReLU(),
        )
        self.score_head = nn.Linear(32, 1)
        self.tag_head = nn.Linear(32, tag_count)

    def forward(self, x: torch.Tensor) -> tuple[torch.Tensor, torch.Tensor]:
        """Run one batch through the shared layers and both prediction heads."""
        shared = self.shared(x)
        return self.score_head(shared).squeeze(-1), self.tag_head(shared)


def feature_vector(features: Dict[str, float], feature_names: List[str]) -> List[float]:
    """Build the model input vector using the trained feature order.

    `features` is a name-to-value mapping, for example:
    `{"layer_height": 0.2, "infill": 15}`.

    `feature_names` is the canonical order saved from training. The model only
    sees a plain list of numbers, so every request must use this same order.
    Missing features are filled with 0.0, and invalid numeric values are cleaned
    before they reach PyTorch.
    """
    return [clean_float(features.get(name, 0.0)) for name in feature_names]


def tag_vector(tags: List[str], tag_names: List[str]) -> List[float]:
    """Convert a list of tag names into a multi-hot vector in a fixed order."""
    tag_set = set(tags)
    return [1.0 if tag in tag_set else 0.0 for tag in tag_names]


def normalize(x: torch.Tensor) -> tuple[torch.Tensor, torch.Tensor, torch.Tensor]:
    """Standardize each feature column and return the stats needed later."""
    means = x.mean(dim=0)
    stds = x.std(dim=0)
    stds = torch.where(stds < 1e-6, torch.ones_like(stds), stds)
    return (x - means) / stds, means, stds


def clean_float(value: float) -> float:
    """Coerce a value to float and replace NaN/Infinity with a safe zero."""
    value = float(value)
    if math.isnan(value) or math.isinf(value):
        return 0.0
    return value
