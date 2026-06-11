from fastapi.testclient import TestClient

from app import app


client = TestClient(app)


def test_train_and_recommend():
    rows = []
    for index in range(24):
        score = 5 if index % 2 == 0 else 2
        rows.append(
            {
                "jobId": f"job_{index}",
                "features": {
                    "mesh_points": 10 + index,
                    "mesh_edges": 20 + index,
                    "config_stiffness": 8 if score == 5 else 30,
                    "config_dampingFactor": 0.2 if score == 5 else 0.02,
                },
                "score": score,
                "tags": ["stable"] if score == 5 else ["too-slow"],
            }
        )

    train_res = client.post("/train", json={"clusterId": "trc_test", "rows": rows})
    assert train_res.status_code == 200, train_res.text
    artifact = train_res.json()["modelArtifact"]
    assert train_res.json()["metrics"]["rows"] == 24

    recommend_res = client.post(
        "/recommend",
        json={
            "modelArtifact": artifact,
            "meshFeatures": {"mesh_points": 16, "mesh_edges": 28},
            "candidateCount": 32,
            "returnCount": 3,
            "bounds": {
                "stiffness": [1, 40],
                "dampingFactor": [0.01, 1],
                "gravity": [-10, 0],
                "maxSteps": [1000, 2000],
            },
        },
    )
    assert recommend_res.status_code == 200, recommend_res.text
    body = recommend_res.json()
    assert len(body["recommendations"]) == 3
    assert body["recommendations"][0]["rank"] == 1
    assert 1 <= body["recommendations"][0]["predictedScore"] <= 5


def test_recommend_missing_artifact():
    res = client.post(
        "/recommend",
        json={
            "modelArtifact": "missing.pt",
            "meshFeatures": {},
            "bounds": {"stiffness": [1, 2]},
        },
    )
    assert res.status_code == 404


def test_train_rejects_wrong_service_key(monkeypatch):
    monkeypatch.setenv("MESH3D_ML_API_KEY", "secret")
    res = client.post("/train", json={"clusterId": "trc_test", "rows": []}, headers={"X-Mesh3D-ML-Key": "wrong"})
    assert res.status_code == 401
