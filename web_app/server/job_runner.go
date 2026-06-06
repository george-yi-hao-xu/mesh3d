package main

import (
	"path/filepath"
	"time"

	"mesh3d/web_app/server/solver"
)

// RunGoSolver connects a stored job to the solver package and publishes checkpoint events.
func RunGoSolver(store *Store, jobID string) {
	store.SetJobStatus(jobID, "running", "")

	job, ok := store.GetJob(jobID)
	if !ok {
		return
	}

	jobDir := filepath.Join(store.storageDir, "jobs", jobID)
	inputPath := filepath.Join(jobDir, "input.msh")
	snapshotDir := filepath.Join(jobDir, "snapshots")
	finalPath := filepath.Join(jobDir, "final.msh")

	cfg := solver.LoadSolverConfig(job.Config)
	model, err := solver.NewMeshModelFromPointCloud(inputPath, cfg)
	if err != nil {
		store.SetJobStatus(jobID, "failed", err.Error())
		return
	}

	result, err := solver.RunMesh(model, cfg, func(simTime float64, step int) error {
		fileName := solver.SnapshotFileName(simTime)
		path := filepath.Join(snapshotDir, fileName)
		if err := model.WritePointCloud(path, simTime, step, false); err != nil {
			return err
		}
		store.AddSnapshot(jobID, Snapshot{
			Label:     solver.SnapshotLabel(simTime),
			SimTime:   simTime,
			Step:      step,
			Path:      path,
			URL:       "/api/jobs/" + jobID + "/snapshots/" + fileName,
			CreatedAt: time.Now().UTC(),
		})
		return nil
	})
	if err != nil {
		store.SetJobStatus(jobID, "failed", err.Error())
		return
	}

	if err := model.WritePointCloud(finalPath, result.SimTime, result.Step, true); err != nil {
		store.SetJobStatus(jobID, "failed", err.Error())
		return
	}
	store.SetResult(jobID, result)
}
