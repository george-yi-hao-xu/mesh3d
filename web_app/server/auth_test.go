package main

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRegisterLoginLogout(t *testing.T) {
	app, handler := newTestApp(t)

	registerRes := postJSON(handler, "/api/auth/register", `{"username":"alice","password":"password123"}`, nil)
	if registerRes.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d; body: %s", registerRes.Code, http.StatusCreated, registerRes.Body.String())
	}
	if cookie := authCookieFromResponse(registerRes); cookie == nil || cookie.Value == "" {
		t.Fatalf("register did not set auth cookie")
	}
	var registerBody struct {
		User userResponse `json:"user"`
	}
	if err := json.Unmarshal(registerRes.Body.Bytes(), &registerBody); err != nil {
		t.Fatalf("decode register body: %v", err)
	}

	duplicateRes := postJSON(handler, "/api/auth/register", `{"username":"Alice","password":"password123"}`, nil)
	if duplicateRes.Code != http.StatusBadRequest {
		t.Fatalf("duplicate register status = %d, want %d", duplicateRes.Code, http.StatusBadRequest)
	}

	loginRes := postJSON(handler, "/api/auth/login", `{"username":"alice","password":"password123"}`, nil)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d; body: %s", loginRes.Code, http.StatusOK, loginRes.Body.String())
	}
	if cookie := authCookieFromResponse(loginRes); cookie == nil || cookie.Value == "" {
		t.Fatalf("login did not set auth cookie")
	}

	badLoginRes := postJSON(handler, "/api/auth/login", `{"username":"alice","password":"wrong-password"}`, nil)
	if badLoginRes.Code != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d, want %d", badLoginRes.Code, http.StatusUnauthorized)
	}

	user := &User{ID: registerBody.User.ID, Username: registerBody.User.Username}
	logoutRes := postJSON(handler, "/api/auth/logout", `{}`, authCookie(t, app, user))
	if logoutRes.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logoutRes.Code, http.StatusOK)
	}
	if cookie := authCookieFromResponse(logoutRes); cookie == nil || cookie.Value != "" {
		t.Fatalf("logout did not clear auth cookie")
	}
}

func TestProtectedAPIRequiresAuth(t *testing.T) {
	_, handler := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/jobs status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestJobsAndResultsAreUserScoped(t *testing.T) {
	app, handler := newTestApp(t)
	alice, err := app.store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	bob, err := app.store.CreateUser("bob", "password123")
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}

	upload := saveTestUpload(t, app.store, alice.ID)
	job, err := app.store.CreateJob(alice.ID, upload.ID, "alice job", map[string]interface{}{"maxSimTime": 1})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	resultPath := app.store.jobResultPath(job.ID)
	if err := os.WriteFile(resultPath, []byte("mesh result"), 0644); err != nil {
		t.Fatalf("write result: %v", err)
	}

	aliceList := request(handler, http.MethodGet, "/api/jobs", nil, authCookie(t, app, alice))
	if aliceList.Code != http.StatusOK {
		t.Fatalf("alice list status = %d, want %d", aliceList.Code, http.StatusOK)
	}
	var jobs []*Job
	if err := json.Unmarshal(aliceList.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("decode alice jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != job.ID {
		t.Fatalf("alice jobs = %+v, want only %s", jobs, job.ID)
	}

	bobList := request(handler, http.MethodGet, "/api/jobs", nil, authCookie(t, app, bob))
	if bobList.Code != http.StatusOK {
		t.Fatalf("bob list status = %d, want %d", bobList.Code, http.StatusOK)
	}
	jobs = nil
	if err := json.Unmarshal(bobList.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("decode bob jobs: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("bob jobs = %+v, want none", jobs)
	}

	bobRead := request(handler, http.MethodGet, "/api/jobs/"+job.ID, nil, authCookie(t, app, bob))
	if bobRead.Code != http.StatusNotFound {
		t.Fatalf("bob read status = %d, want %d", bobRead.Code, http.StatusNotFound)
	}

	aliceResult := request(handler, http.MethodGet, "/api/jobs/"+job.ID+"/result", nil, authCookie(t, app, alice))
	if aliceResult.Code != http.StatusOK {
		t.Fatalf("alice result status = %d, want %d", aliceResult.Code, http.StatusOK)
	}

	bobResult := request(handler, http.MethodGet, "/api/jobs/"+job.ID+"/result", nil, authCookie(t, app, bob))
	if bobResult.Code != http.StatusNotFound {
		t.Fatalf("bob result status = %d, want %d", bobResult.Code, http.StatusNotFound)
	}

}

func TestDeleteJobIsOwnedAndOnlyForFinishedJobs(t *testing.T) {
	app, handler := newTestApp(t)
	alice, err := app.store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	bob, err := app.store.CreateUser("bob", "password123")
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}

	upload := saveTestUpload(t, app.store, alice.ID)
	job, err := app.store.CreateJob(alice.ID, upload.ID, "alice job", map[string]interface{}{"maxSimTime": 1})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	jobDir := app.store.jobDir(job.ID)

	bobDelete := request(handler, http.MethodDelete, "/api/jobs/"+job.ID, nil, authCookie(t, app, bob))
	if bobDelete.Code != http.StatusNotFound {
		t.Fatalf("bob delete status = %d, want %d", bobDelete.Code, http.StatusNotFound)
	}

	queuedDelete := request(handler, http.MethodDelete, "/api/jobs/"+job.ID, nil, authCookie(t, app, alice))
	if queuedDelete.Code != http.StatusConflict {
		t.Fatalf("queued delete status = %d, want %d", queuedDelete.Code, http.StatusConflict)
	}

	app.store.SetJobStatus(job.ID, "failed", "simulated failure")

	doneDelete := request(handler, http.MethodDelete, "/api/jobs/"+job.ID, nil, authCookie(t, app, alice))
	if doneDelete.Code != http.StatusNoContent {
		t.Fatalf("done delete status = %d, want %d", doneDelete.Code, http.StatusNoContent)
	}

	if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
		t.Fatalf("job directory still exists after delete: %v", err)
	}

	listRes := request(handler, http.MethodGet, "/api/jobs", nil, authCookie(t, app, alice))
	if listRes.Code != http.StatusOK {
		t.Fatalf("alice list after delete status = %d, want %d", listRes.Code, http.StatusOK)
	}
	var jobs []*Job
	if err := json.Unmarshal(listRes.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("decode list after delete: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("alice jobs after delete = %+v, want none", jobs)
	}
}

func TestJWTRejectsExpiredAndInvalidSignature(t *testing.T) {
	user := &User{ID: "usr_test", Username: "alice"}
	token, err := createJWT([]byte("secret-a"), user, time.Unix(1000, 0))
	if err != nil {
		t.Fatalf("create jwt: %v", err)
	}

	if _, err := verifyJWT([]byte("secret-b"), token, time.Unix(1001, 0)); err == nil {
		t.Fatalf("verifyJWT accepted invalid signature")
	}
	if _, err := verifyJWT([]byte("secret-a"), token, time.Unix(1000, 0).Add(tokenTTL+time.Second)); err == nil {
		t.Fatalf("verifyJWT accepted expired token")
	}
}

func newTestApp(t *testing.T) (*App, http.Handler) {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("MESH3D_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("MESH3D_TEST_DATABASE_URL is required for database-backed server tests")
	}

	store, err := NewPostgresStore(t.TempDir(), databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(func() {
		_ = store.db.Close()
	})
	if err := store.Init(); err != nil {
		t.Fatalf("init store: %v", err)
	}
	resetTestDB(t, store)

	app := &App{store: store, clientDir: "../client", jwtSecret: []byte("test-secret")}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", app.handleHealth)
	mux.HandleFunc("/api/auth/register", app.handleRegister)
	mux.HandleFunc("/api/auth/login", app.handleLogin)
	mux.HandleFunc("/api/auth/logout", app.handleLogout)
	mux.HandleFunc("/api/auth/me", app.handleMe)
	mux.HandleFunc("/api/uploads", app.requireAuth(app.handleUploads))
	mux.HandleFunc("/api/jobs", app.requireAuth(app.handleJobs))
	mux.HandleFunc("/api/jobs/", app.requireAuth(app.handleJobRoutes))
	return app, mux
}

func resetTestDB(t *testing.T, store *Store) {
	t.Helper()

	_, err := store.db.Exec(`truncate table job_snapshots, jobs, uploads, users restart identity cascade`)
	if err != nil {
		t.Fatalf("reset test database: %v", err)
	}
}

func saveTestUpload(t *testing.T, store *Store, userID string) Upload {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "input-*.mesh")
	if err != nil {
		t.Fatalf("create temp upload: %v", err)
	}
	defer file.Close()
	if _, err := file.WriteString(`# Format: mesh-v1

vertices
0 0 0 0 1 1
1 1 0 0 0 1

edges
0 1 1 10
`); err != nil {
		t.Fatalf("write temp upload: %v", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("seek temp upload: %v", err)
	}

	upload, err := store.SaveUpload(userID, file, &multipart.FileHeader{Filename: "input.mesh"})
	if err != nil {
		t.Fatalf("save upload: %v", err)
	}
	return upload
}

func postJSON(handler http.Handler, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	return request(handler, http.MethodPost, path, strings.NewReader(body), cookie)
}

func request(handler http.Handler, method, path string, body *strings.Reader, cookie *http.Cookie) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		reader = body
	}
	req := httptest.NewRequest(method, path, reader)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func authCookie(t *testing.T, app *App, user *User) *http.Cookie {
	t.Helper()

	token, err := createJWT(app.jwtSecret, user, time.Now().UTC())
	if err != nil {
		t.Fatalf("create auth cookie: %v", err)
	}
	return &http.Cookie{Name: authCookieName, Value: token}
}

func authCookieFromResponse(res *httptest.ResponseRecorder) *http.Cookie {
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == authCookieName {
			return cookie
		}
	}
	return nil
}
