package main

import (
	"os"
	"time"

	"mesh3d/web_app/server/solver"
)

// RunGoSolver connects a stored job to the solver package and records checkpoint files.
func RunGoSolver(store *Store, jobID string) (*Job, error) {
	store.SetJobStatus(jobID, "running", "")

	job, ok := store.GetJob(jobID)
	if !ok {
		return nil, nil
	}

	cfg := solver.LoadSolverConfig(job.Config)
	model, err := solver.NewMeshModelFromPointCloud(store.jobInputPath(jobID), cfg)
	if err != nil {
		store.SetJobStatus(jobID, "failed", err.Error())
		return getJobAfterRun(store, jobID), err
	}

	result, err := solver.RunMesh(model, cfg, func(simTime float64, step int) error {
		fileName := solver.SnapshotFileName(simTime)
		path := store.jobSnapshotPath(jobID, fileName)
		if err := model.WriteMeshSnapshot(path, simTime, step, false); err != nil {
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
		return getJobAfterRun(store, jobID), err
	}

	if err := model.WriteMeshSnapshot(store.jobResultPath(jobID), result.SimTime, result.Step, true); err != nil {
		store.SetJobStatus(jobID, "failed", err.Error())
		return getJobAfterRun(store, jobID), err
	}
	store.SetResult(jobID, result)
	return getJobAfterRun(store, jobID), nil
}

// ReadJobFrames loads checkpoint and final mesh text for a completed job response.
func ReadJobFrames(store *Store, job *Job) ([]JobFrame, error) {
	if job == nil {
		return nil, nil
	}

	frames := make([]JobFrame, 0, len(job.Snapshots)+1)
	for _, snapshot := range job.Snapshots {
		text, err := os.ReadFile(snapshot.Path)
		if err != nil {
			return nil, err
		}
		frames = append(frames, JobFrame{
			Label:   snapshot.Label,
			URL:     snapshot.URL,
			Text:    string(text),
			SimTime: snapshot.SimTime,
			Step:    snapshot.Step,
		})
	}

	if job.ResultURL != "" {
		text, err := os.ReadFile(store.jobResultPath(job.ID))
		if err != nil {
			return nil, err
		}
		frames = append(frames, JobFrame{
			Label:   "Final",
			URL:     job.ResultURL,
			Text:    string(text),
			IsFinal: true,
			SimTime: job.FinalTime,
			Step:    job.FinalStep,
		})
	}

	return frames, nil
}

func getJobAfterRun(store *Store, jobID string) *Job {
	job, _ := store.GetJob(jobID)
	return job
}
