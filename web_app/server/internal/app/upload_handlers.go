package app

import (
	"errors"
	"net/http"
	"os"
	"strings"
)

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
