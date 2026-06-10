# Mesh3D Web App

This folder contains the server-backed web workflow for Mesh3D.

## Current Stack

- Frontend: React, TypeScript, MobX, Vite
- Backend: Go `net/http`
- Database: Postgres metadata
- Storage: local filesystem `.mesh` uploads and solver artifacts
- Solver: Go mass-spring solver ported from the C++ logic

## Local Run

Install Go. For the Postgres-backed metadata store, start a local database:

```bash
docker run --name mesh3d-postgres \
  --env-file server/.env \
  -p 5432:5432 \
  -v mesh3d_pgdata:/var/lib/postgresql/data \
  -d postgres:16
```

Start the frontend on the local machine:
```bash
cd web_app/client
pnpm i
pnpm dev
```

Build the frontend:

```bash
cd web_app/client
pnpm install
pnpm build
```

Then run the server. It loads `server/.env` automatically and serves the built frontend from `client/dist` unless `MESH3D_CLIENT_DIR` overrides it:

```bash
cd web_app/server
go mod tidy
go run .
```

On startup, the server applies `server/internal/app/schema/postgres.sql`. `MESH3D_DATABASE_URL` is required. Browser-generated `.mesh` uploads and generated `.mesh` solver artifacts remain on the local filesystem.

To inspect Postgres from the terminal, open `psql` inside the running container:

```bash
docker exec -it mesh3d-postgres psql -U mesh3d -d mesh3d
```

Useful `psql` commands:

```sql
\dt
\d jobs
SELECT * FROM jobs LIMIT 10;
SELECT * FROM uploads LIMIT 10;
\q
```

`psql` meta-commands like `\dt`, `\d`, and `\q` do not need semicolons. SQL queries like `SELECT ...` do need semicolons.

Open:

```text
http://localhost:8080
```

The server stores uploaded mesh topology files, checkpoint `.mesh` files, and final `.mesh` files under:

```text
web_app/server/storage/
```

Postgres stores users, upload metadata, job metadata, job config, and snapshot metadata.

## API

```text
GET  /api/uploads
POST /api/uploads
GET  /api/uploads/{id}
POST /api/jobs
GET  /api/jobs
GET  /api/jobs/{id}
DELETE /api/jobs/{id}
GET  /api/jobs/{id}/snapshots/{file}
GET  /api/jobs/{id}/result
```

Job creation runs the solver and returns the completed job plus all checkpoint/final frames in one response. Existing job files can still be downloaded through the snapshot and result endpoints.

## Request Flow

The Go backend handles three responsibilities:

```text
HTTP API
file/job storage
solver orchestration
```

When the browser talks to the backend, requests enter through `server/main.go`, where routes are registered:

```go
mux.HandleFunc("/api/health", app.handleHealth)
mux.HandleFunc("/api/uploads", app.handleUploads)
mux.HandleFunc("/api/jobs", app.handleJobs)
mux.HandleFunc("/api/jobs/", app.handleJobRoutes)
mux.HandleFunc("/", app.handleStatic)
```

### 1. Browser Opens the Page

```text
GET /
```

The server calls `handleStatic` in `server/internal/app/static_handlers.go`.

That serves files from `web_app/client`, including:

```text
index.html
assets/
styles.css
```

This is why `main.go` has `clientDir`: the backend currently serves the frontend too.

During frontend development, run the Go server on `:8080` and start Vite separately:

```bash
cd web_app/client
pnpm dev
```

Vite proxies `/api` requests to `http://127.0.0.1:8080`.

### 2. Browser Previews and Uploads Mesh Topology

```text
GET /api/uploads
```

Lists the signed-in user's mesh warehouse entries. Each entry includes the upload ID, file name, size, mesh kind, point count, edge count, and created timestamp.

```text
POST /api/uploads
```

Stores a raw uploaded `.msh`/`.mesh` file or a generated spring mesh in the signed-in user's mesh warehouse. The server validates that the file contains valid points and records point/spring counts.

```text
GET /api/uploads/{id}
```

Returns one user-owned warehouse mesh artifact plus its text content for previewing and job creation.

It:

```text
reads multipart file field "pointCloud"
calls store.SaveUpload(...)
writes storage/uploads/{uploadId}.mesh
records upload metadata in Postgres
returns upload JSON to the browser
```

### 3. Browser Creates a Solver Job

```text
POST /api/jobs
```

The server calls `handleJobs`.

It:

```text
reads JSON body: uploadId + config
calls store.CreateJob(...)
creates storage/jobs/{jobId}/
copies input.mesh into the job folder
records job metadata and config in Postgres
runs the solver from explicit mesh topology
reads checkpoint and final mesh files
returns job JSON plus frame mesh text
```

### 4. Solver Runs

The job handler calls `RunGoSolver` in `server/internal/app/job_runner.go`.

It:

```text
loads storage/jobs/{jobId}/input.mesh
loads solver settings from the Postgres-backed job config
creates the solver mesh
runs physics
writes checkpoint .mesh files
writes final.mesh
updates job metadata in Postgres
```

At each checkpoint, the server writes:

```text
storage/jobs/{jobId}/snapshots/{time}.mesh
```

Then it calls:

```go
store.AddSnapshot(...)
```

When finished, it writes:

```text
storage/jobs/{jobId}/final.mesh
```

Then it calls:

```go
store.SetResult(...)
```

After the solver finishes, `ReadJobFrames` loads the generated `.mesh` text and includes it in the `POST /api/jobs` response. The browser stores those frames on the selected job and shows the final frame by default.

### 5. Browser Fetches a Result File

For stored jobs or downloads, the browser can still request an existing `.mesh` file:

```text
GET /api/jobs/{jobId}/snapshots/{file}
```

For the final result:

```text
GET /api/jobs/{jobId}/result
```

The server calls `serveJobFile`, which reads the generated file from the job folder and returns it.

The full lifecycle is:

```text
browser upload
  -> Go handler
  -> Store saves file

browser create job
  -> Go handler
  -> Store creates job folder
  -> solver runs inside the request

solver creates checkpoints
  -> Store records snapshot
  -> server reads all generated frames
  -> browser receives job + frames
```

## Solver Notes

The browser turns uploaded point clouds into explicit `mesh-v1` vertices and springs before submitting a solve. The Go solver loads that explicit topology, then runs the mass-spring update loop until convergence or a configured limit.

Future mesh-quality judging notes live in:

```text
docs/mesh-quality-judgment.md
```

Solver orchestration code lives in:

```text
server/solver/
```

Mesh physics code lives in:

```text
server/solver/mesh/
```

The server-side job runner that connects solver output to stored snapshots and bundled frame responses lives in:

```text
server/internal/app/job_runner.go
```

Default convergence rule:

```text
max particle velocity < velocityEpsilon
and max per-step movement < positionEpsilon
for stableFrames consecutive frames
```

Important server solver config fields:

```text
stiffness
dampingFactor
airResistanceFactor
gravity
timeStep
snapshotInterval
maxSimTime
maxSteps
velocityEpsilon
positionEpsilon
stableFrames
```

Browser topology config fields are also saved with the job for reproducibility:

```text
springSeed
maxSpringDist
maxSpringsPerParticle
springConnectProb
```

## Google Cloud Direction

For the first deploy, use Cloud Run for the Go server. Local filesystem artifact storage is fine for local development, but production should move artifacts to Google Cloud Storage and metadata to Cloud SQL for PostgreSQL.

The intended production split is:

```text
Cloud Run: Go API/server and solver
Cloud Storage: uploads, snapshots, final .mesh files
Database: job metadata
Solver: Go solver first; external headless solver process remains possible later
```
