package app

import (
	"net/http"
	"path/filepath"
)

// handleStatic serves the frontend client from the configured client directory.
func (a *App) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)
	if path == "." || path == string(filepath.Separator) {
		http.ServeFile(w, r, filepath.Join(a.clientDir, "index.html"))
		return
	}
	http.FileServer(http.Dir(a.clientDir)).ServeHTTP(w, r)
}
