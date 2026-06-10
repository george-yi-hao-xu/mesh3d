package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// handleHealth reports whether the server process is reachable.
func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := a.store.CreateUser(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now().UTC()
	token, err := createJWT(a.jwtSecret, user, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	setAuthCookie(w, r, token, now.Add(tokenTTL))
	writeJSON(w, http.StatusCreated, map[string]userResponse{"user": userToResponse(user)})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := a.store.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	now := time.Now().UTC()
	token, err := createJWT(a.jwtSecret, user, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	setAuthCookie(w, r, token, now.Add(tokenTTL))
	writeJSON(w, http.StatusOK, map[string]userResponse{"user": userToResponse(user)})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clearAuthCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, ok := a.authenticateRequest(r)
	if !ok {
		clearAuthCookie(w, r)
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]userResponse{"user": userToResponse(user)})
}

func (a *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := a.authenticateRequest(r)
		if !ok {
			clearAuthCookie(w, r)
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

func (a *App) authenticateRequest(r *http.Request) (*User, bool) {
	cookie, err := r.Cookie(authCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return nil, false
	}

	claims, err := verifyJWT(a.jwtSecret, cookie.Value, time.Now().UTC())
	if err != nil {
		return nil, false
	}
	user, ok := a.store.GetUser(claims.Subject)
	if !ok || user.Username != claims.Username {
		return nil, false
	}
	return user, true
}

func currentUser(r *http.Request) *User {
	user, _ := r.Context().Value(userContextKey).(*User)
	return user
}

// handleUploads lists and stores user-owned warehouse mesh artifacts.
func (a *App) handleUploads(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, a.store.ListUploads(user.ID))
		return
	}
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

	upload, err := a.store.SaveUpload(user.ID, file, header, r.FormValue("meshKind"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, upload)
}

func (a *App) handleUploadRoutes(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/uploads/"))
	if len(parts) != 1 || !safePathPart(parts[0]) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if r.Method == http.MethodDelete {
		if err := a.store.DeleteUploadForUser(user.ID, parts[0]); err != nil {
			var inUse uploadInUseError
			switch err {
			case errUploadNotFound:
				writeError(w, http.StatusNotFound, err.Error())
			default:
				if errors.As(err, &inUse) {
					writeJSON(w, http.StatusBadRequest, map[string]interface{}{
						"error":         inUse.Error(),
						"relatedJobIds": inUse.jobIDs,
					})
				} else {
					writeError(w, http.StatusInternalServerError, err.Error())
				}
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	upload, ok := a.store.GetUploadForUser(user.ID, parts[0])
	if !ok {
		writeError(w, http.StatusNotFound, "upload not found")
		return
	}
	data, err := os.ReadFile(upload.Path)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"upload": upload,
		"text":   string(data),
	})
}

// handleJobs lists existing jobs or creates a new solver job from an upload.
func (a *App) handleJobs(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.store.ListJobs(user.ID))
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
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			job, ok := a.store.GetJobForUser(user.ID, jobID)
			if !ok {
				writeError(w, http.StatusNotFound, "job not found")
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
		writeError(w, http.StatusNotFound, "job not found")
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
		writeError(w, http.StatusNotFound, "job not found")
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

// handleStatic serves the frontend client from the configured client directory.
func (a *App) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)
	if path == "." || path == string(filepath.Separator) {
		http.ServeFile(w, r, filepath.Join(a.clientDir, "index.html"))
		return
	}
	http.FileServer(http.Dir(a.clientDir)).ServeHTTP(w, r)
}
