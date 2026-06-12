package app

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
	if _, err := app.store.db.Exec(`update jobs set result_mesh_text = $2, status = 'done' where id = $1`, job.ID, "mesh result"); err != nil {
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

func TestCreateJobRejectsUploadWithoutSprings(t *testing.T) {
	app, handler := newTestApp(t)
	alice, err := app.store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	upload := saveTestUploadWithoutSprings(t, app.store, alice.ID)

	res := postJSON(handler, "/api/jobs", `{"uploadId":"`+upload.ID+`","name":"springless job","config":{"maxSimTime":1}}`, authCookie(t, app, alice))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("create job status = %d, want %d; body: %s", res.Code, http.StatusBadRequest, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "springs") {
		t.Fatalf("create job body = %q, want springs error", res.Body.String())
	}

	jobs := app.store.ListJobs(alice.ID)
	if len(jobs) != 0 {
		t.Fatalf("jobs after rejected create = %+v, want none", jobs)
	}
}

func TestJobReviewIsOwnedAndValidated(t *testing.T) {
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

	invalid := request(handler, http.MethodPut, "/api/jobs/"+job.ID+"/review", strings.NewReader(`{"score":6}`), authCookie(t, app, alice))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid review status = %d, want %d; body: %s", invalid.Code, http.StatusBadRequest, invalid.Body.String())
	}

	bobReview := request(handler, http.MethodPut, "/api/jobs/"+job.ID+"/review", strings.NewReader(`{"score":4}`), authCookie(t, app, bob))
	if bobReview.Code != http.StatusNotFound {
		t.Fatalf("bob review status = %d, want %d; body: %s", bobReview.Code, http.StatusNotFound, bobReview.Body.String())
	}

	aliceReview := request(handler, http.MethodPut, "/api/jobs/"+job.ID+"/review", strings.NewReader(`{"score":5,"tags":["Stable","good result","stable"],"note":"works well"}`), authCookie(t, app, alice))
	if aliceReview.Code != http.StatusOK {
		t.Fatalf("alice review status = %d, want %d; body: %s", aliceReview.Code, http.StatusOK, aliceReview.Body.String())
	}

	fetched := request(handler, http.MethodGet, "/api/jobs/"+job.ID, nil, authCookie(t, app, alice))
	if fetched.Code != http.StatusOK {
		t.Fatalf("fetch reviewed job status = %d, want %d; body: %s", fetched.Code, http.StatusOK, fetched.Body.String())
	}
	var reviewed Job
	if err := json.Unmarshal(fetched.Body.Bytes(), &reviewed); err != nil {
		t.Fatalf("decode reviewed job: %v", err)
	}
	if reviewed.Review == nil || reviewed.Review.Score != 5 || reviewed.Review.Note != "works well" {
		t.Fatalf("reviewed job review = %+v, want saved review", reviewed.Review)
	}
	if len(reviewed.Review.Tags) != 2 || reviewed.Review.Tags[0] != "stable" || reviewed.Review.Tags[1] != "good-result" {
		t.Fatalf("review tags = %+v, want normalized unique tags", reviewed.Review.Tags)
	}
}

func TestTrainingClustersAreOwnedAndRequireReviewedJobs(t *testing.T) {
	app, handler := newTestApp(t)
	alice, err := app.store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	bob, err := app.store.CreateUser("bob", "password123")
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}

	createRes := postJSON(handler, "/api/training/clusters", `{"name":"cloth tuning"}`, authCookie(t, app, alice))
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create cluster status = %d, want %d; body: %s", createRes.Code, http.StatusCreated, createRes.Body.String())
	}
	var cluster TrainingCluster
	if err := json.Unmarshal(createRes.Body.Bytes(), &cluster); err != nil {
		t.Fatalf("decode cluster: %v", err)
	}

	bobRead := request(handler, http.MethodGet, "/api/training/clusters/"+cluster.ID, nil, authCookie(t, app, bob))
	if bobRead.Code != http.StatusNotFound {
		t.Fatalf("bob read cluster status = %d, want %d", bobRead.Code, http.StatusNotFound)
	}

	upload := saveTestUpload(t, app.store, alice.ID)
	unreviewedJob, err := app.store.CreateJob(alice.ID, upload.ID, "unreviewed", map[string]interface{}{"maxSimTime": 1})
	if err != nil {
		t.Fatalf("create unreviewed job: %v", err)
	}
	addUnreviewed := postJSON(handler, "/api/training/clusters/"+cluster.ID+"/jobs", `{"jobId":"`+unreviewedJob.ID+`"}`, authCookie(t, app, alice))
	if addUnreviewed.Code != http.StatusNotFound {
		t.Fatalf("add unreviewed job status = %d, want %d; body: %s", addUnreviewed.Code, http.StatusNotFound, addUnreviewed.Body.String())
	}

	reviewedJob := createReviewedTestJob(t, app.store, alice.ID, 5)
	addReviewed := postJSON(handler, "/api/training/clusters/"+cluster.ID+"/jobs", `{"jobId":"`+reviewedJob.ID+`"}`, authCookie(t, app, alice))
	if addReviewed.Code != http.StatusOK {
		t.Fatalf("add reviewed job status = %d, want %d; body: %s", addReviewed.Code, http.StatusOK, addReviewed.Body.String())
	}
	if err := json.Unmarshal(addReviewed.Body.Bytes(), &cluster); err != nil {
		t.Fatalf("decode updated cluster: %v", err)
	}
	if len(cluster.Jobs) != 1 || cluster.Jobs[0].Job.ID != reviewedJob.ID {
		t.Fatalf("cluster jobs = %+v, want reviewed job %s", cluster.Jobs, reviewedJob.ID)
	}
}

func TestTrainingRejectsSmallClusterBeforeCallingML(t *testing.T) {
	app, handler := newTestApp(t)
	alice, err := app.store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	cluster, err := app.store.CreateTrainingCluster(alice.ID, "small")
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}
	job := createReviewedTestJob(t, app.store, alice.ID, 4)
	if _, err := app.store.AddJobToTrainingCluster(alice.ID, cluster.ID, job.ID); err != nil {
		t.Fatalf("add job: %v", err)
	}

	res := postJSON(handler, "/api/training/clusters/"+cluster.ID+"/train", `{}`, authCookie(t, app, alice))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("train small status = %d, want %d; body: %s", res.Code, http.StatusBadRequest, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "20 reviewed jobs") {
		t.Fatalf("train small body = %q, want minimum review message", res.Body.String())
	}
}

func TestTrainingRecordsSidecarFailure(t *testing.T) {
	t.Setenv("MESH3D_ML_URL", "")
	app, handler := newTestApp(t)
	alice, err := app.store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	cluster, err := app.store.CreateTrainingCluster(alice.ID, "ready")
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}
	for i := 0; i < minTrainingReviews; i++ {
		job := createReviewedTestJob(t, app.store, alice.ID, 3+(i%3))
		if _, err := app.store.AddJobToTrainingCluster(alice.ID, cluster.ID, job.ID); err != nil {
			t.Fatalf("add job %d: %v", i, err)
		}
	}

	res := postJSON(handler, "/api/training/clusters/"+cluster.ID+"/train", `{}`, authCookie(t, app, alice))
	if res.Code != http.StatusBadGateway {
		t.Fatalf("train sidecar failure status = %d, want %d; body: %s", res.Code, http.StatusBadGateway, res.Body.String())
	}
	var run TrainingRun
	if err := json.Unmarshal(res.Body.Bytes(), &run); err != nil {
		t.Fatalf("decode failed run: %v", err)
	}
	if run.Status != "failed" || !strings.Contains(run.Error, "MESH3D_ML_URL") {
		t.Fatalf("failed run = %+v, want sidecar configuration error", run)
	}
}

func TestUploadsAreUserScopedAndFetchable(t *testing.T) {
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

	aliceList := request(handler, http.MethodGet, "/api/uploads", nil, authCookie(t, app, alice))
	if aliceList.Code != http.StatusOK {
		t.Fatalf("alice uploads status = %d, want %d", aliceList.Code, http.StatusOK)
	}
	var uploads []Upload
	if err := json.Unmarshal(aliceList.Body.Bytes(), &uploads); err != nil {
		t.Fatalf("decode alice uploads: %v", err)
	}
	if len(uploads) != 1 || uploads[0].ID != upload.ID || uploads[0].PointCount != 2 || uploads[0].EdgeCount != 1 {
		t.Fatalf("alice uploads = %+v, want upload %s with metadata", uploads, upload.ID)
	}

	bobList := request(handler, http.MethodGet, "/api/uploads", nil, authCookie(t, app, bob))
	if bobList.Code != http.StatusOK {
		t.Fatalf("bob uploads status = %d, want %d", bobList.Code, http.StatusOK)
	}
	uploads = nil
	if err := json.Unmarshal(bobList.Body.Bytes(), &uploads); err != nil {
		t.Fatalf("decode bob uploads: %v", err)
	}
	if len(uploads) != 0 {
		t.Fatalf("bob uploads = %+v, want none", uploads)
	}

	aliceFetch := request(handler, http.MethodGet, "/api/uploads/"+upload.ID, nil, authCookie(t, app, alice))
	if aliceFetch.Code != http.StatusOK {
		t.Fatalf("alice fetch upload status = %d, want %d", aliceFetch.Code, http.StatusOK)
	}
	if !strings.Contains(aliceFetch.Body.String(), "mesh-v1") {
		t.Fatalf("alice fetch body did not include mesh text: %s", aliceFetch.Body.String())
	}

	bobFetch := request(handler, http.MethodGet, "/api/uploads/"+upload.ID, nil, authCookie(t, app, bob))
	if bobFetch.Code != http.StatusNotFound {
		t.Fatalf("bob fetch upload status = %d, want %d", bobFetch.Code, http.StatusNotFound)
	}
}

func TestDeleteUploadRequiresOwnershipAndNoJobs(t *testing.T) {
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

	bobDelete := request(handler, http.MethodDelete, "/api/uploads/"+upload.ID, nil, authCookie(t, app, bob))
	if bobDelete.Code != http.StatusNotFound {
		t.Fatalf("bob delete upload status = %d, want %d", bobDelete.Code, http.StatusNotFound)
	}
	if _, ok := app.store.GetUploadForUser(alice.ID, upload.ID); !ok {
		t.Fatalf("upload was removed by non-owner")
	}

	aliceDelete := request(handler, http.MethodDelete, "/api/uploads/"+upload.ID, nil, authCookie(t, app, alice))
	if aliceDelete.Code != http.StatusNoContent {
		t.Fatalf("alice delete upload status = %d, want %d; body: %s", aliceDelete.Code, http.StatusNoContent, aliceDelete.Body.String())
	}
	if _, ok := app.store.GetUploadForUser(alice.ID, upload.ID); ok {
		t.Fatalf("upload still exists after delete")
	}
}

func TestDeleteUploadRejectsReferencedUpload(t *testing.T) {
	app, handler := newTestApp(t)
	alice, err := app.store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	upload := saveTestUpload(t, app.store, alice.ID)
	job, err := app.store.CreateJob(alice.ID, upload.ID, "alice job", map[string]interface{}{"maxSimTime": 1})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	res := request(handler, http.MethodDelete, "/api/uploads/"+upload.ID, nil, authCookie(t, app, alice))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("delete referenced upload status = %d, want %d; body: %s", res.Code, http.StatusBadRequest, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "existing jobs") {
		t.Fatalf("delete referenced upload body = %q, want existing jobs error", res.Body.String())
	}
	var body struct {
		RelatedJobIDs []string `json:"relatedJobIds"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode referenced upload response: %v", err)
	}
	if len(body.RelatedJobIDs) != 1 || body.RelatedJobIDs[0] != job.ID {
		t.Fatalf("related job ids = %+v, want %s", body.RelatedJobIDs, job.ID)
	}
	if _, ok := app.store.GetUploadForUser(alice.ID, upload.ID); !ok {
		t.Fatalf("referenced upload was deleted")
	}
}

func TestInvalidUploadIsRejected(t *testing.T) {
	app, handler := newTestApp(t)
	alice, err := app.store.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}

	var body strings.Builder
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("pointCloud", "bad.mesh")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("not a mesh\n")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/uploads", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(authCookie(t, app, alice))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("invalid upload status = %d, want %d; body: %s", res.Code, http.StatusBadRequest, res.Body.String())
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

	store, err := NewPostgresStore(databaseURL)
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
	mux.HandleFunc("/api/uploads/", app.requireAuth(app.handleUploadRoutes))
	mux.HandleFunc("/api/jobs", app.requireAuth(app.handleJobs))
	mux.HandleFunc("/api/jobs/", app.requireAuth(app.handleJobRoutes))
	mux.HandleFunc("/api/training/clusters", app.requireAuth(app.handleTrainingClusters))
	mux.HandleFunc("/api/training/clusters/", app.requireAuth(app.handleTrainingRoutes))
	return app, mux
}

func resetTestDB(t *testing.T, store *Store) {
	t.Helper()

	_, err := store.db.Exec(`truncate table config_recommendations, training_runs, training_cluster_jobs, training_clusters, job_snapshots, job_reviews, jobs, uploads, users restart identity cascade`)
	if err != nil {
		t.Fatalf("reset test database: %v", err)
	}
}

func createReviewedTestJob(t *testing.T, store *Store, userID string, score int) *Job {
	t.Helper()
	upload := saveTestUpload(t, store, userID)
	job, err := store.CreateJob(userID, upload.ID, "reviewed job", map[string]interface{}{
		"stiffness":             10,
		"dampingFactor":         0.1,
		"gravity":               -4.9,
		"airResistanceFactor":   0.001,
		"timeStep":              0.01,
		"snapshotInterval":      0.05,
		"maxSimTime":            1,
		"maxSteps":              100,
		"velocityEpsilon":       0.001,
		"positionEpsilon":       0.001,
		"stableFrames":          10,
		"springSeed":            42,
		"maxSpringDist":         1.5,
		"maxSpringsPerParticle": 4,
		"springConnectProb":     0.8,
	})
	if err != nil {
		t.Fatalf("create reviewed job: %v", err)
	}
	tags := []string{"stable"}
	if score < 4 {
		tags = []string{"too-slow"}
	}
	if _, err := store.SaveJobReviewForUser(userID, job.ID, score, tags, "training label"); err != nil {
		t.Fatalf("save review: %v", err)
	}
	job, ok := store.GetJobForUser(userID, job.ID)
	if !ok {
		t.Fatalf("fetch reviewed job")
	}
	return job
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

	upload, err := store.SaveUpload(userID, file, &multipart.FileHeader{Filename: "input.mesh"}, "uploaded")
	if err != nil {
		t.Fatalf("save upload: %v", err)
	}
	return upload
}

func saveTestUploadWithoutSprings(t *testing.T, store *Store, userID string) Upload {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "input-springless-*.mesh")
	if err != nil {
		t.Fatalf("create temp upload: %v", err)
	}
	defer file.Close()
	if _, err := file.WriteString(`# Format: mesh-v1

vertices
0 0 0 0 1 1
1 1 0 0 0 1

edges
`); err != nil {
		t.Fatalf("write temp upload: %v", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("seek temp upload: %v", err)
	}

	upload, err := store.SaveUpload(userID, file, &multipart.FileHeader{Filename: "input-springless.mesh"}, "uploaded")
	if err != nil {
		t.Fatalf("save upload: %v", err)
	}
	if upload.EdgeCount != 0 {
		t.Fatalf("springless upload edge count = %d, want 0", upload.EdgeCount)
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
