package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleHealth reports whether the server process is reachable.
func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleUploads accepts a point-cloud file and stores it as an upload artifact.
func (a *App) handleUploads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("pointCloud")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing pointCloud file")
		return
	}
	defer file.Close()

	upload, err := a.store.SaveUpload(file, header)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, upload)
}

// handleJobs lists existing jobs or creates a new solver job from an upload.
func (a *App) handleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.store.ListJobs())
	case http.MethodPost:
		var req struct {
			UploadID string                 `json:"uploadId"`
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

		job, err := a.store.CreateJob(req.UploadID, req.Config)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		go RunGoSolver(a.store, job.ID)
		writeJSON(w, http.StatusCreated, job)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleJobRoutes dispatches nested job routes such as events, snapshots, and result files.
func (a *App) handleJobRoutes(w http.ResponseWriter, r *http.Request) {
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
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		job, ok := a.store.GetJob(jobID)
		if !ok {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
		writeJSON(w, http.StatusOK, job)
		return
	}

	switch parts[1] {
	case "events":
		a.handleJobEvents(w, r, jobID)
	case "snapshots":
		if len(parts) != 3 {
			writeError(w, http.StatusNotFound, "snapshot not found")
			return
		}
		if !safePathPart(parts[2]) {
			writeError(w, http.StatusBadRequest, "invalid snapshot file")
			return
		}
		a.serveJobFile(w, r, jobID, filepath.Join("snapshots", parts[2]))
	case "result":
		a.serveJobFile(w, r, jobID, "final.msh")
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

// handleJobEvents streams sparse job updates with Server-Sent Events.
func (a *App) handleJobEvents(w http.ResponseWriter, r *http.Request, jobID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := a.store.GetJob(jobID); !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := a.store.Subscribe(jobID)
	defer a.store.Unsubscribe(jobID, ch)

	if job, ok := a.store.GetJob(jobID); ok {
		writeSSE(w, Event{Type: "job", JobID: jobID, Job: job})
		for _, snapshot := range job.Snapshots {
			s := snapshot
			writeSSE(w, Event{Type: "snapshot", JobID: jobID, Snapshot: &s})
		}
		flusher.Flush()
	}

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case event := <-ch:
			writeSSE(w, event)
			flusher.Flush()
			if event.Type == "done" || event.Type == "failed" {
				return
			}
		}
	}
}

// serveJobFile serves generated job artifacts from the job storage directory.
func (a *App) serveJobFile(w http.ResponseWriter, r *http.Request, jobID string, relPath string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := a.store.GetJob(jobID); !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	path := filepath.Join(a.store.storageDir, "jobs", jobID, relPath)
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeFile(w, r, path)
}

// handleStatic serves the vanilla JS client from the configured client directory.
func (a *App) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)
	if path == "." || path == string(filepath.Separator) {
		http.ServeFile(w, r, filepath.Join(a.clientDir, "index.html"))
		return
	}
	http.FileServer(http.Dir(a.clientDir)).ServeHTTP(w, r)
}
