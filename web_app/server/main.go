package main

import (
	"log"
	"net/http"
)

// main configures storage, registers routes, and starts the HTTP server.
func main() {
	addr := envOr("MESH3D_ADDR", ":8080")
	storageDir := envOr("MESH3D_STORAGE_DIR", "storage")
	clientDir := envOr("MESH3D_CLIENT_DIR", "../client")

	store := NewStore(storageDir)
	if err := store.Init(); err != nil {
		log.Fatalf("init storage: %v", err)
	}

	app := &App{store: store, clientDir: clientDir, jwtSecret: initJWTSecret()}

	mux := http.NewServeMux()

	// Browser health check used to show whether the backend is reachable.
	mux.HandleFunc("/api/health", app.handleHealth)

	mux.HandleFunc("/api/auth/register", app.handleRegister)
	mux.HandleFunc("/api/auth/login", app.handleLogin)
	mux.HandleFunc("/api/auth/logout", app.handleLogout)
	mux.HandleFunc("/api/auth/me", app.handleMe)

	// Browser uploads the point-cloud file here first; the response upload ID is then used to create a solver job.
	mux.HandleFunc("/api/uploads", app.requireAuth(app.handleUploads))

	// Browser creates and lists solver jobs here after an upload has been stored.
	mux.HandleFunc("/api/jobs", app.requireAuth(app.handleJobs))

	// Browser reads one job, listens for job events, and downloads checkpoint/final .msh files here.
	mux.HandleFunc("/api/jobs/", app.requireAuth(app.handleJobRoutes))

	// Serve the static frontend files from clientDir.
	mux.HandleFunc("/", app.handleStatic)

	log.Printf("mesh3d web app listening on %s", addr)
	log.Printf("serving client from %s", clientDir)
	log.Fatal(http.ListenAndServe(addr, logRequests(mux)))
}
