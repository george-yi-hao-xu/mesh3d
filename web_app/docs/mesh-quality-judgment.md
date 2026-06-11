# Mesh Quality Judgment Notes

This note records the June 2026 discussion about adding mesh judging to the web app, so future sessions can recover the product intent quickly.

## Current Flow

The web app currently works like this:

- The user selects or uploads a mesh/point cloud in the browser.
- The browser prepares an explicit `mesh-v1` topology with nodes and springs.
- The job request sends the selected upload ID and solver/config values to the Go server.
- The server copies the exact input mesh into the job folder, runs the Go mass-spring solver, writes checkpoint/final `.mesh` files, and returns frames for animation.
- The client displays those frames with timeline controls.

Relevant code areas:

- Client mesh preparation: `web_app/client/src/stores/mesh-preview-store.ts`
- Client spring generation and mesh serialization: `web_app/client/src/lib/mesh-topology.ts`
- Client job/frame state: `web_app/client/src/stores/job-store.ts`
- Server job creation and storage: `web_app/server/internal/app/store.go`
- Server solver orchestration: `web_app/server/internal/app/job_runner.go`
- Go solver and mesh physics: `web_app/server/solver/`

## Product Direction

The proposed feature is to judge whether a mesh is likely to be good before and after the solve.

The chosen v1 should be an explainable deterministic score, not a trained deep-learning model yet. The app has many config values, and those can become model inputs later, but the first version should use transparent engineering metrics that users can understand.

Primary target for v1:

```text
Stable simulation and likely convergence
```

This means the score should help users catch meshes/configs that are likely to produce unstable, disconnected, hard-to-read, or slow-to-converge solver behavior.

## V1 Metrics

Compute a `MeshQualityReport` from the prepared input mesh and the submitted solver config.

Suggested report fields:

```text
score: 0-5
grade: good | caution | poor
metrics:
  points
  springs
  fixedPoints
  components
  isolatedPoints
  minDegree
  maxDegree
  meanDegree
  springLengthMin
  springLengthMax
  springLengthMean
  springLengthVariance
  springLengthStdDev
  springLengthCoefficientOfVariation
warnings: string[]
configRisks: string[]
```

Important v1 signals:

- Spring length 方差 / variance.
- Spring length coefficient of variation.
- Connectivity and number of connected components.
- Isolated or nearly isolated points.
- Degree distribution, especially very low average or minimum degree.
- Fixed/anchor point count.
- Solver config risk, such as high stiffness with large time step, zero/low damping, too few stable frames, or very large max run limits.

Suggested default scoring:

- Start at `100`.
- Penalize high spring length variation.
- Penalize disconnected components and isolated nodes heavily.
- Penalize very low average degree or many degree-1 nodes.
- Penalize missing or too few fixed points.
- Penalize risky solver config combinations.
- Grade thresholds: `good >= 80`, `caution >= 55`, `poor < 55`.

The quality report should inform the user, not block job creation. Existing validation can still reject invalid meshes, for example meshes with no springs.

## Implementation Shape

Recommended v1 implementation:

- Add a client-side helper, for example `mesh-quality.ts`, that computes live quality from `preview.preparedMesh.mesh` and the current config.
- Add a matching Go server helper that computes quality from the exact job input mesh and submitted config.
- Persist the server-computed report on the job, for example with a `mesh_quality jsonb` column exposed as `meshQuality` in job JSON.
- Show quality in preview mode before solve.
- Show the saved quality report in job mode after solve and when browsing job history.

The server-side report should be treated as the authoritative saved value because it uses the exact input mesh copied for the job.

## Future ML Direction

If deep learning becomes useful later, the app can start saving feature vectors and solver outcomes:

- Mesh metrics listed above.
- Solver config values.
- Solver result: converged, final time, final step, failure reason, frame count.
- Optional user labels, such as accepted/rejected mesh or visual quality rating.

That dataset can support a future model, but v1 should stay explainable and deterministic.

## Assumptions From The Discussion

- First version: explainable score.
- Timing: show both before solve and after solve.
- Quality target: stable simulation/convergence.
- Mechanical advantage / 力学优势 is a possible later extension, not part of the first scoring target.
