package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// copyFile copies a file and syncs the destination before returning.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// writeJSON sends a JSON response with the provided status code.
func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// writeError sends a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// newID creates a random id with the requested prefix.
func newID(prefix string) string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(buf[:])
}

// splitPath trims and splits a URL path into non-empty parts.
func splitPath(path string) []string {
	raw := strings.Split(strings.Trim(path, "/"), "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

// safePathPart rejects path traversal tokens for URL-derived file names.
func safePathPart(part string) bool {
	if part == "" || part == "." || part == ".." {
		return false
	}
	return part == filepath.Base(part) && !strings.Contains(part, `\`)
}

// envOr reads an environment variable or returns a fallback.
func EnvOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// loadDotEnv reads KEY=value pairs from a local .env file without overriding
// variables already provided by the shell.
func LoadDotEnv(path string) error {
	return loadDotEnv(path, false)
}

// LoadDotEnvOverride reads KEY=value pairs and overwrites values previously
// loaded from another env file.
func LoadDotEnvOverride(path string) error {
	return loadDotEnv(path, true)
}

func loadDotEnv(path string, override bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}

	for lineNumber, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("parse %s:%d: expected KEY=value", path, lineNumber+1)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("parse %s:%d: empty key", path, lineNumber+1)
		}
		if len(value) >= 2 {
			quote := value[:1]
			if (quote == `"` || quote == `'`) && strings.HasSuffix(value, quote) {
				value = value[1 : len(value)-1]
			}
		}
		if override || os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set %s from %s:%d: %w", key, path, lineNumber+1, err)
			}
		}
	}

	return nil
}

// LoadEnvFiles loads base .env values, then overlays .env.<MESH3D_ENV>.
// Variables already provided by the shell win over both files.
func LoadEnvFiles(basePath string) error {
	initial := os.Environ()
	if err := LoadDotEnv(basePath); err != nil {
		return err
	}
	env := EnvOr("MESH3D_ENV", "development")
	if err := LoadDotEnvOverride(basePath + "." + env); err != nil {
		return err
	}
	for _, pair := range initial {
		key, value, ok := strings.Cut(pair, "=")
		if ok {
			_ = os.Setenv(key, value)
		}
	}
	return nil
}

// logRequests wraps an HTTP handler with simple method/path/duration logging.
func LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
