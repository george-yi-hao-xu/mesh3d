package app

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleJobs lists existing jobs or creates a new solver job from an upload.
func (a *App) handleJobs(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	switch r.Method {
	// GET all jobs
	// GET /api/jobs
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.store.ListJobs(user.ID))
	
	// POST a new job
	// POST /api/jobs
	case http.MethodPost:
		var req struct {
			UploadID string                 `json:"uploadId"`
			Name     string                 `json:"name"`
			Config   map[string]interface{} `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.UploadID == "" {
			writeError(w, http.StatusBadRequest, "uploadId is required")
			return
		}
		if req.Config == nil {
			req.Config = make(map[string]interface{})
		}

		job, err := a.store.CreateJob(user.ID, req.UploadID, req.Name, req.Config)

		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		job, runErr := RunGoSolver(a.store, job.ID)

		if job == nil {
			writeError(w, http.StatusInternalServerError, "job disappeared while running")
			return
		}

		if runErr != nil {
			writeJSON(w, http.StatusCreated, JobCreateResponse{Job: job, Frames: nil})
			return
		}

		frames, err := ReadJobFrames(a.store, job)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, JobCreateResponse{Job: job, Frames: frames})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleJobRoutes dispatches nested job routes such as job metadata, snapshots, and result files.
func (a *App) handleJobRoutes(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/jobs/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	jobID := parts[0]
	if !safePathPart(jobID) {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	// GET&DELETE api jobs/:jobId
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			job, ok := a.store.GetJobForUser(user.ID, jobID)
			if !ok {
				log.Printf("job not found for user %s and job id %s", user.ID, jobID)
				writeError(w, http.StatusNotFound, "job not found when try to get this job")
				return
			}
			writeJSON(w, http.StatusOK, job)
		case http.MethodDelete:
			if err := a.store.DeleteJobForUser(user.ID, jobID); err != nil {
				switch err {
				case errJobNotFound:
					writeError(w, http.StatusNotFound, err.Error())
				case errJobNotDeletable:
					writeError(w, http.StatusConflict, err.Error())
				default:
					writeError(w, http.StatusInternalServerError, err.Error())
				}
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// api jobs/:jobId/???
	switch parts[1] {
	case "review":
		if len(parts) != 2 {
			writeError(w, http.StatusNotFound, "review not found")
			return
		}

		a.handleJobReview(w, r, user.ID, jobID)
	case "snapshots":
		if len(parts) != 3 {
			writeError(w, http.StatusNotFound, "snapshot not found")
			return
		}
		if !safePathPart(parts[2]) {
			writeError(w, http.StatusBadRequest, "invalid snapshot file")
			return
		}

		a.serveJobFile(w, r, user.ID, jobID, filepath.Join("snapshots", parts[2]))
	case "result":
		a.serveJobResultFile(w, r, user.ID, jobID)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (a *App) handleJobReview(w http.ResponseWriter, r *http.Request, userID, jobID string) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Score int      `json:"score"`
		Tags  []string `json:"tags"`
		Note  string   `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	review, err := a.store.SaveJobReviewForUser(userID, jobID, req.Score, req.Tags, req.Note)
	if err != nil {
		switch err {
		case errJobNotFound:
			writeError(w, http.StatusNotFound, err.Error())
		case errInvalidReview:
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, review)
}

func (a *App) serveJobResultFile(w http.ResponseWriter, r *http.Request, userID, jobID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := a.store.GetJobForUser(userID, jobID); !ok {
		writeError(w, http.StatusNotFound, "job not found when try to get result file for this job")
		return
	}
	if _, err := os.Stat(a.store.jobResultPath(jobID)); err == nil {
		serveFile(w, r, a.store.jobResultPath(jobID))
		return
	}
	writeError(w, http.StatusNotFound, "file not found")
}

// serveJobFile serves generated job artifacts from the job storage directory.
func (a *App) serveJobFile(w http.ResponseWriter, r *http.Request, userID, jobID string, relPath string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := a.store.GetJobForUser(userID, jobID); !ok {
		writeError(w, http.StatusNotFound, "job not found when try to serve this job file")
		return
	}

	path := a.store.jobArtifactPath(jobID, relPath)
	serveFile(w, r, path)
}

func serveFile(w http.ResponseWriter, r *http.Request, path string) {
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeFile(w, r, path)
}
