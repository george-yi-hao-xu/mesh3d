from __future__ import annotations

from typing import Dict, List, Tuple

from pydantic import BaseModel, Field


class TrainingRow(BaseModel):
    jobId: str
    features: Dict[str, float]
    score: int = Field(ge=1, le=5)
    tags: List[str] = Field(default_factory=list)
    note: str = ""


class TrainRequest(BaseModel):
    clusterId: str
    rows: List[TrainingRow]


class TrainResponse(BaseModel):
    metrics: Dict[str, float | int | List[str]]
    modelArtifact: str


class RecommendRequest(BaseModel):
    modelArtifact: str
    meshFeatures: Dict[str, float]
    candidateCount: int = 512
    returnCount: int = 5
    bounds: Dict[str, Tuple[float, float]]


class Recommendation(BaseModel):
    rank: int
    config: Dict[str, float | int]
    predictedScore: float
    predictedTags: List[str]


class RecommendResponse(BaseModel):
    recommendations: List[Recommendation]
