package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"mesh3d/web_app/server/internal/app"
)

// main configures the data store, registers routes, and starts the HTTP server.
func main() {
	if err := app.LoadEnvFiles(".env"); err != nil {
		log.Printf("warning: %v", err)
	}

	addr := app.EnvOr("MESH3D_ADDR", ":8080")
	clientDir := app.EnvOr("MESH3D_CLIENT_DIR", "../client/dist")
	databaseURL := strings.TrimSpace(os.Getenv("MESH3D_DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("MESH3D_DATABASE_URL is required")
	}

	store, err := app.NewPostgresStore(databaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	if err := store.Init(); err != nil {
		log.Fatalf("init store: %v", err)
	}
	log.Printf("using postgres store")

	serverApp := app.New(store, clientDir, app.InitJWTSecret())

	log.Printf("mesh3d web app listening on %s", addr)
	log.Printf("serving client from %s", clientDir)
	log.Fatal(http.ListenAndServe(addr, app.LogRequests(serverApp.Routes())))
}
