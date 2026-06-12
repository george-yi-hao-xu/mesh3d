package app

import (
	"strings"
	"time"

	"mesh3d/web_app/server/solver"
)

// RunGoSolver connects a stored job to the solver package and records checkpoint text.
func RunGoSolver(store *Store, jobID string) (*Job, error) {
	store.SetJobStatus(jobID, "running", "")

	job, ok := store.GetJob(jobID)
	if !ok {
		return nil, nil
	}

	cfg := solver.LoadSolverConfig(job.Config)
	model, err := solver.NewMeshModelFromReader(strings.NewReader(job.InputText), cfg)
	if err != nil {
		store.SetJobStatus(jobID, "failed", err.Error())
		return getJobAfterRun(store, jobID), err
	}

	result, err := solver.RunMesh(model, cfg, func(simTime float64, step int) error {
		fileName := solver.SnapshotFileName(simTime)
		var text strings.Builder
		if err := model.WriteMeshSnapshotTo(&text, simTime, step, false); err != nil {
			return err
		}
		store.AddSnapshot(jobID, Snapshot{
			Label:     solver.SnapshotLabel(simTime),
			SimTime:   simTime,
			Step:      step,
			FileName:  fileName,
			MeshText:  text.String(),
			URL:       "/api/jobs/" + jobID + "/snapshots/" + fileName,
			CreatedAt: time.Now().UTC(),
		})
		return nil
	})
	if err != nil {
		store.SetJobStatus(jobID, "failed", err.Error())
		return getJobAfterRun(store, jobID), err
	}

	var finalText strings.Builder
	if err := model.WriteMeshSnapshotTo(&finalText, result.SimTime, result.Step, true); err != nil {
		store.SetJobStatus(jobID, "failed", err.Error())
		return getJobAfterRun(store, jobID), err
	}
	store.SetResult(jobID, result, finalText.String())
	return getJobAfterRun(store, jobID), nil
}

// ReadJobFrames loads checkpoint and final mesh text for a completed job response.
func ReadJobFrames(store *Store, job *Job) ([]JobFrame, error) {
	if job == nil {
		return nil, nil
	}

	frames := make([]JobFrame, 0, len(job.Snapshots)+1)
	for _, snapshot := range job.Snapshots {
		frames = append(frames, JobFrame{
			Label:   snapshot.Label,
			URL:     snapshot.URL,
			Text:    snapshot.MeshText,
			SimTime: snapshot.SimTime,
			Step:    snapshot.Step,
		})
	}

	if job.ResultURL != "" {
		frames = append(frames, JobFrame{
			Label:   "Final",
			URL:     job.ResultURL,
			Text:    job.ResultText,
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
