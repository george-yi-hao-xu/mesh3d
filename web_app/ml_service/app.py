from __future__ import annotations

import os
from pathlib import Path
from typing import Dict

from fastapi import Depends, FastAPI, Header, HTTPException

from recommendation import recommend_configs
from schemas import RecommendRequest, RecommendResponse, TrainRequest, TrainResponse
from training import train_tabular_model


def load_local_env(path: Path) -> None:
    if not path.exists():
        return
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("export "):
            line = line.removeprefix("export ").strip()
        key, sep, value = line.partition("=")
        if not sep:
            continue
        key = key.strip()
        value = value.strip().strip("'\"")
        if key and key not in os.environ:
            os.environ[key] = value


load_local_env(Path(__file__).resolve().parent / ".env")

app = FastAPI(title="Mesh3D ML Service")


@app.get("/health")
def health() -> Dict[str, str]:
    return {"status": "ok"}


def require_ml_key(x_mesh3d_ml_key: str = Header(default="")) -> None:
    expected = os.getenv("MESH3D_ML_API_KEY", "").strip()
    if expected and x_mesh3d_ml_key != expected:
        raise HTTPException(status_code=401, detail="invalid ML service key")


@app.post("/train", response_model=TrainResponse)
def train(req: TrainRequest, _: None = Depends(require_ml_key)) -> TrainResponse:
    return train_tabular_model(req)


@app.post("/recommend", response_model=RecommendResponse)
def recommend(req: RecommendRequest, _: None = Depends(require_ml_key)) -> RecommendResponse:
    return recommend_configs(req)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="127.0.0.1", port=8090)
