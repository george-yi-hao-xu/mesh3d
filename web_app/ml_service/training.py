from __future__ import annotations

import random
import uuid

import torch
from fastapi import HTTPException
from torch import nn

from modeling import MODEL_DIR, SEED, TabularNet, feature_vector, normalize, tag_vector
from schemas import TrainRequest, TrainResponse


def train_tabular_model(req: TrainRequest) -> TrainResponse:
    if len(req.rows) < 2:
        raise HTTPException(status_code=400, detail="at least two training rows are required")

    torch.manual_seed(SEED)
    random.seed(SEED)

    feature_names = sorted({key for row in req.rows for key in row.features.keys()})
    tag_names = sorted({tag for row in req.rows for tag in row.tags})
    x = torch.tensor([feature_vector(row.features, feature_names) for row in req.rows], dtype=torch.float32)
    x, means, stds = normalize(x)
    y_score = torch.tensor([(row.score - 1) / 4 for row in req.rows], dtype=torch.float32)
    y_tags = torch.tensor([tag_vector(row.tags, tag_names) for row in req.rows], dtype=torch.float32)

    train_idx, val_idx = train_validation_indices(len(req.rows))
    model = TabularNet(len(feature_names), len(tag_names))
    fit_model(model, x, y_score, y_tags, train_idx, tag_names)
    mae, tag_accuracy = evaluate_model(model, x, y_score, y_tags, val_idx, tag_names)
    artifact = save_model_artifact(req.clusterId, model, feature_names, tag_names, means, stds)

    return TrainResponse(
        metrics={
            "rows": len(req.rows),
            "features": len(feature_names),
            "tags": tag_names,
            "validationScoreMae": round(mae, 4),
            "validationTagAccuracy": round(tag_accuracy, 4),
        },
        modelArtifact=artifact,
    )


def train_validation_indices(row_count: int) -> tuple[torch.Tensor, torch.Tensor]:
    indices = list(range(row_count))
    random.shuffle(indices)
    split = max(1, int(len(indices) * 0.8))
    train_idx = torch.tensor(indices[:split], dtype=torch.long)
    val_idx = torch.tensor(indices[split:] or indices[:1], dtype=torch.long)
    return train_idx, val_idx


def fit_model(
    model: TabularNet,
    x: torch.Tensor,
    y_score: torch.Tensor,
    y_tags: torch.Tensor,
    train_idx: torch.Tensor,
    tag_names: list[str],
) -> None:
    optimizer = torch.optim.Adam(model.parameters(), lr=0.01)
    score_loss = nn.MSELoss()
    tag_loss = nn.BCEWithLogitsLoss()

    for _ in range(160):
        model.train()
        optimizer.zero_grad()
        pred_score, pred_tags = model(x[train_idx])
        loss = score_loss(pred_score, y_score[train_idx])
        if tag_names:
            loss = loss + 0.35 * tag_loss(pred_tags, y_tags[train_idx])
        loss.backward()
        optimizer.step()


def evaluate_model(
    model: TabularNet,
    x: torch.Tensor,
    y_score: torch.Tensor,
    y_tags: torch.Tensor,
    val_idx: torch.Tensor,
    tag_names: list[str],
) -> tuple[float, float]:
    model.eval()
    with torch.no_grad():
        pred_score, pred_tags = model(x[val_idx])
        mae = torch.mean(torch.abs((pred_score.clamp(0, 1) * 4 + 1) - (y_score[val_idx] * 4 + 1))).item()
        tag_accuracy = 0.0
        if tag_names:
            tag_accuracy = ((torch.sigmoid(pred_tags) >= 0.5) == (y_tags[val_idx] >= 0.5)).float().mean().item()
    return float(mae), float(tag_accuracy)


def save_model_artifact(
    cluster_id: str,
    model: TabularNet,
    feature_names: list[str],
    tag_names: list[str],
    means: torch.Tensor,
    stds: torch.Tensor,
) -> str:
    artifact = f"{cluster_id}_{uuid.uuid4().hex}.pt"
    torch.save(
        {
            "state_dict": model.state_dict(),
            "feature_names": feature_names,
            "tag_names": tag_names,
            "means": means.tolist(),
            "stds": stds.tolist(),
        },
        MODEL_DIR / artifact,
    )
    return artifact
