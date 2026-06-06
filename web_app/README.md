# Mesh3D Web App

This folder contains the server-backed web workflow for Mesh3D.

## Current Stack

- Frontend: vanilla HTML/CSS/JS
- Backend: Go `net/http`
- Storage: local filesystem first
- Solver: Go mass-spring solver ported from the C++ logic

## Local Run

Install Go, then run:

```powershell
cd web_app/server
go run .
```

Open:

```text
http://localhost:8080
```

The server stores uploaded point clouds, job metadata, checkpoint `.msh` files, and final `.msh` files under:

```text
web_app/server/storage/
```

## API

```text
POST /api/uploads
POST /api/jobs
GET  /api/jobs
GET  /api/jobs/{id}
GET  /api/jobs/{id}/events
GET  /api/jobs/{id}/snapshots/{file}
GET  /api/jobs/{id}/result
```

Progress uses sparse checkpoint events. The client does not stream every physics frame; it receives notifications when `5s`, `10s`, later checkpoints, and final results are ready.

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

The server calls `handleStatic` in `server/handlers.go`.

That serves files from `web_app/client`, including:

```text
index.html
app.js
styles.css
```

This is why `main.go` has `clientDir`: the backend currently serves the frontend too.

### 2. Browser Uploads a Point Cloud

```text
POST /api/uploads
```

The server calls `handleUploads`.

It:

```text
reads multipart file field "pointCloud"
calls store.SaveUpload(...)
writes storage/uploads/{uploadId}.msh
writes storage/uploads/{uploadId}.json
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
copies input.msh into the job folder
writes config.json
starts the solver in a background goroutine
returns job JSON immediately
```

The solver is started with:

```go
go RunGoSolver(a.store, job.ID)
```

That `go` keyword means the solver runs in the background, so the HTTP response does not wait for the whole simulation.

### 4. Browser Listens for Checkpoints

```text
GET /api/jobs/{jobId}/events
```

The server calls `handleJobEvents`.

This opens a Server-Sent Events stream. The connection stays open while the job runs, and the server sends messages such as:

```json
{"type":"snapshot","jobId":"job_xxx"}
{"type":"done","jobId":"job_xxx"}
```

### 5. Solver Runs

The background goroutine calls `RunGoSolver` in `server/job_runner.go`.

It:

```text
loads storage/jobs/{jobId}/input.msh
loads storage/jobs/{jobId}/config.json through the job config
creates the solver mesh
runs physics
writes checkpoint .msh files
writes final.msh
updates job metadata
publishes events to active browser subscribers
```

At each checkpoint, the server writes:

```text
storage/jobs/{jobId}/snapshots/{time}.msh
```

Then it calls:

```go
store.AddSnapshot(...)
```

When finished, it writes:

```text
storage/jobs/{jobId}/final.msh
```

Then it calls:

```go
store.SetResult(...)
```

### 6. Browser Fetches a Result File

After receiving a checkpoint event, the browser requests the actual `.msh` file:

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
  -> solver goroutine starts

browser opens event stream
  -> Go keeps connection open

solver creates checkpoints
  -> Store records snapshot
  -> Store publishes event
  -> browser receives event
  -> browser fetches .msh file
```

## Solver Notes

The Go solver loads uploaded `.msh` point clouds, generates random springs using the same seeded distance/probability strategy as the C++ app, then runs the mass-spring update loop until convergence or a configured limit.

Solver orchestration code lives in:

```text
server/solver/
```

Mesh physics code lives in:

```text
server/solver/mesh/
```

The server-side job runner that connects solver output to stored snapshots and SSE events lives in:

```text
server/job_runner.go
```

Default convergence rule:

```text
max particle velocity < velocityEpsilon
and max per-step movement < positionEpsilon
for stableFrames consecutive frames
```

Important job config fields:

```text
stiffness
dampingFactor
airResistanceFactor
gravity
springSeed
maxSpringDist
maxSpringsPerParticle
springConnectProb
timeStep
snapshotInterval
maxSimTime
velocityEpsilon
positionEpsilon
stableFrames
```

## Google Cloud Direction

For the first deploy, use Cloud Run for the Go server. Local filesystem storage is fine for local development, but production should move artifacts to Google Cloud Storage and metadata to Firestore, Cloud SQL, or SQLite on a persistent volume.

The intended production split is:

```text
Cloud Run: Go API/server and solver
Cloud Storage: uploads, snapshots, final .msh files
Database: job metadata
Solver: Go solver first; external headless solver process remains possible later
```
