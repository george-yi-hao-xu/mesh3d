package app

import "net/http"

// New creates the HTTP application with its storage and runtime settings.
func New(store *Store, clientDir string, jwtSecret []byte) *App {
	return &App{store: store, clientDir: clientDir, jwtSecret: jwtSecret}
}

// Routes registers the browser API and static frontend routes.
func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()

	// Browser health check used to show whether the backend is reachable.
	mux.HandleFunc("/api/health", a.handleHealth)

	mux.HandleFunc("/api/auth/register", a.handleRegister)
	mux.HandleFunc("/api/auth/login", a.handleLogin)
	mux.HandleFunc("/api/auth/logout", a.handleLogout)
	mux.HandleFunc("/api/auth/me", a.handleMe)

	// Browser stores and picks user-owned mesh warehouse artifacts here.
	mux.HandleFunc("/api/uploads", a.requireAuth(a.handleUploads))
	mux.HandleFunc("/api/uploads/", a.requireAuth(a.handleUploadRoutes))

	// Browser creates and lists solver jobs here after an upload has been stored.
	mux.HandleFunc("/api/jobs", a.requireAuth(a.handleJobs))

	// Browser reads, deletes, and downloads checkpoint/final files for one stored job here.
	mux.HandleFunc("/api/jobs/", a.requireAuth(a.handleJobRoutes))

	// Browser groups reviewed jobs and requests ML training/recommendations here.
	mux.HandleFunc("/api/training/clusters", a.requireAuth(a.handleTrainingClusters))
	mux.HandleFunc("/api/training/clusters/", a.requireAuth(a.handleTrainingRoutes))

	// Serve the static frontend files from clientDir.
	mux.HandleFunc("/", a.handleStatic)

	return mux
}
