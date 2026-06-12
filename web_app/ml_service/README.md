# Mesh3D ML Service

Python sidecar for training tabular config recommendation models.

Run locally:

```bash
cd web_app/ml_service
python -m venv .venv
. .venv/bin/activate
pip install -r requirements.txt
uvicorn app:app --host 127.0.0.1 --port 8090
```

Development config is loaded from `.env.development` by default. Keep the Go
server and Python sidecar on the same `MESH3D_ML_API_KEY`.

Set the Go server environment variables:

```bash
MESH3D_ML_URL=http://127.0.0.1:8090
MESH3D_ML_API_KEY=mesh3d_ml_dev_key
```

Set the same `MESH3D_ML_API_KEY` in the Python service environment or in
`web_app/ml_service/.env.development`. `/train` and `/recommend` reject requests
with the wrong key when this variable is set.
