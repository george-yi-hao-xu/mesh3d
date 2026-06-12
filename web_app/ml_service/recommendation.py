from __future__ import annotations

import random
from pathlib import Path
from typing import Dict, Tuple

import torch
from fastapi import HTTPException

from modeling import MODEL_DIR, SEED, TabularNet, feature_vector
from schemas import Recommendation, RecommendRequest, RecommendResponse


def recommend_configs(req: RecommendRequest) -> RecommendResponse:
    artifact_path = MODEL_DIR / Path(req.modelArtifact).name
    if not artifact_path.exists():
        raise HTTPException(status_code=404, detail="model artifact not found")

    payload = torch.load(artifact_path, map_location="cpu", weights_only=False)
    feature_names = payload["feature_names"]
    tag_names = payload["tag_names"]
    means = torch.tensor(payload["means"], dtype=torch.float32)
    stds = torch.tensor(payload["stds"], dtype=torch.float32)
    model = TabularNet(len(feature_names), len(tag_names))
    model.load_state_dict(payload["state_dict"])
    model.eval()

    candidate_count = max(1, min(req.candidateCount, 5000))
    return_count = max(1, min(req.returnCount, 20))
    rng = random.Random(SEED)
    scored: list[Recommendation] = []

    with torch.no_grad():
        for _ in range(candidate_count):
            config = sample_config(req.bounds, rng)
            features = dict(req.meshFeatures)
            for key, value in config.items():
                features[f"config_{key}"] = float(value)
            x = torch.tensor([feature_vector(features, feature_names)], dtype=torch.float32)
            x = (x - means) / stds
            pred_score, pred_tags = model(x)
            score = float(pred_score.sigmoid().item() * 4 + 1)
            tag_probs = torch.sigmoid(pred_tags).squeeze(0).tolist() if tag_names else []
            tags = [tag for tag, prob in zip(tag_names, tag_probs) if prob >= 0.5]
            scored.append(Recommendation(rank=0, config=config, predictedScore=round(score, 4), predictedTags=tags))

    scored.sort(key=lambda item: item.predictedScore, reverse=True)
    top = scored[:return_count]
    for index, item in enumerate(top, start=1):
        item.rank = index
    return RecommendResponse(recommendations=top)


def sample_config(bounds: Dict[str, Tuple[float, float]], rng: random.Random) -> Dict[str, float | int]:
    config: Dict[str, float | int] = {}
    integer_keys = {"maxSteps", "stableFrames", "springSeed", "maxSpringsPerParticle"}
    for key, pair in bounds.items():
        lo, hi = pair
        value = rng.uniform(float(lo), float(hi))
        if key in integer_keys:
            config[key] = int(round(value))
        else:
            config[key] = round(value, 6)
    return config
