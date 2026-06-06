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

	app := &App{store: store, clientDir: clientDir}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", app.handleHealth)
	mux.HandleFunc("/api/uploads", app.handleUploads)
	mux.HandleFunc("/api/jobs", app.handleJobs)
	mux.HandleFunc("/api/jobs/", app.handleJobRoutes)
	mux.HandleFunc("/", app.handleStatic)

	log.Printf("mesh3d web app listening on %s", addr)
	log.Printf("serving client from %s", clientDir)
	log.Fatal(http.ListenAndServe(addr, logRequests(mux)))
}
