package main

import (
	"errors"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mesh3d/web_app/server/solver"

	"golang.org/x/crypto/bcrypt"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

var (
	errJobNotFound     = errors.New("job not found")
	errJobNotDeletable = errors.New("job is not finished")
)

// NewStore creates the in-memory indexes around the configured storage root.
func NewStore(storageDir string) *Store {
	return &Store{
		storageDir: storageDir,
		users:      make(map[string]*User),
		usernames:  make(map[string]string),
		uploads:    make(map[string]Upload),
		jobs:       make(map[string]*Job),
	}
}

// Init creates required storage folders and reloads persisted metadata.
func (s *Store) Init() error {
	for _, dir := range []string{
		s.storageDir,
		filepath.Join(s.storageDir, "uploads"),
		filepath.Join(s.storageDir, "jobs"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	if err := s.loadUsers(); err != nil {
		return err
	}
	return s.loadMetadata()
}

// CreateUser validates and stores a new user with a bcrypt password hash.
func (s *Store) CreateUser(username, password string) (*User, error) {
	username, key, err := normalizeUsername(username)
	if err != nil {
		return nil, err
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	s.mu.Lock()
	_, exists := s.usernames[key]
	s.mu.Unlock()
	if exists {
		return nil, errors.New("username already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           newID("usr"),
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().UTC(),
	}

	s.mu.Lock()
	if _, exists := s.usernames[key]; exists {
		s.mu.Unlock()
		return nil, errors.New("username already exists")
	}
	s.users[user.ID] = user
	s.usernames[key] = user.ID
	users := s.cloneUsersLocked()
	s.mu.Unlock()

	if err := s.saveUsers(users); err != nil {
		return nil, err
	}
	return cloneUser(user), nil
}

// AuthenticateUser verifies username/password credentials.
func (s *Store) AuthenticateUser(username, password string) (*User, error) {
	_, key, err := normalizeUsername(username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	s.mu.Lock()
	id, ok := s.usernames[key]
	user := s.users[id]
	s.mu.Unlock()
	if !ok || user == nil {
		return nil, errors.New("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid username or password")
	}
	return cloneUser(user), nil
}

// GetUser returns a user by id.
func (s *Store) GetUser(id string) (*User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[id]
	if !ok {
		return nil, false
	}
	return cloneUser(user), true
}

// SaveUpload persists an uploaded point-cloud file and its metadata.
func (s *Store) SaveUpload(userID string, file multipart.File, header *multipart.FileHeader) (Upload, error) {
	id := newID("upl")
	fileName := filepath.Base(header.Filename)
	if fileName == "." || fileName == string(filepath.Separator) || fileName == "" {
		fileName = "point_cloud.msh"
	}

	path := filepath.Join(s.storageDir, "uploads", id+".msh")
	out, err := os.Create(path)
	if err != nil {
		return Upload{}, err
	}
	defer out.Close()

	size, err := io.Copy(out, file)
	if err != nil {
		return Upload{}, err
	}

	upload := Upload{
		ID:        id,
		UserID:    userID,
		FileName:  fileName,
		Size:      size,
		Path:      path,
		CreatedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	s.uploads[id] = upload
	s.mu.Unlock()

	if err := writeJSONFile(filepath.Join(s.storageDir, "uploads", id+".json"), upload); err != nil {
		return Upload{}, err
	}

	return upload, nil
}

// CreateJob creates a job folder with input and config copied from a prior upload.
func (s *Store) CreateJob(userID, uploadID, name string, config map[string]interface{}) (*Job, error) {
	s.mu.Lock()
	upload, ok := s.uploads[uploadID]
	s.mu.Unlock()
	if !ok {
		return nil, errors.New("upload not found")
	}
	if upload.UserID != userID {
		return nil, errors.New("upload not found")
	}

	id := newID("job")
	jobDir := filepath.Join(s.storageDir, "jobs", id)
	snapshotDir := filepath.Join(jobDir, "snapshots")
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, err
	}
	if err := copyFile(upload.Path, filepath.Join(jobDir, "input.msh")); err != nil {
		return nil, err
	}
	if err := writeJSONFile(filepath.Join(jobDir, "config.json"), config); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultJobName(now, upload.FileName)
	}

	job := &Job{
		ID:        id,
		UserID:    userID,
		UploadID:  upload.ID,
		Name:      name,
		InputName: upload.FileName,
		Status:    "queued",
		Config:    config,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	s.jobs[id] = job
	s.mu.Unlock()

	if err := s.saveJobMetadata(job); err != nil {
		return nil, err
	}
	return cloneJob(job), nil
}

func defaultJobName(createdAt time.Time, inputName string) string {
	meshName := filepath.Base(inputName)
	if ext := filepath.Ext(meshName); ext != "" {
		meshName = strings.TrimSuffix(meshName, ext)
	}
	if meshName == "." || meshName == string(filepath.Separator) || meshName == "" {
		meshName = "mesh"
	}
	return createdAt.Format("2006-01-02_15-04-05") + "_" + meshName
}

// ListJobs returns cloned user-owned job records so callers cannot mutate store state.
func (s *Store) ListJobs(userID string) []*Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		if job.UserID == userID {
			jobs = append(jobs, cloneJob(job))
		}
	}
	return jobs
}

// GetJob returns a cloned job record by id.
func (s *Store) GetJob(id string) (*Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	return cloneJob(job), true
}

// GetJobForUser returns a cloned job only when it belongs to the requested user.
func (s *Store) GetJobForUser(userID, id string) (*Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok || job.UserID != userID {
		return nil, false
	}
	return cloneJob(job), true
}

// DeleteJobForUser removes a finished job and its stored artifacts when it belongs to the requested user.
func (s *Store) DeleteJobForUser(userID, id string) error {
	s.mu.Lock()
	job, ok := s.jobs[id]
	if !ok || job.UserID != userID {
		s.mu.Unlock()
		return errJobNotFound
	}
	if job.Status == "queued" || job.Status == "running" {
		s.mu.Unlock()
		return errJobNotDeletable
	}

	s.mu.Unlock()

	jobDir := filepath.Join(s.storageDir, "jobs", id)
	if err := os.RemoveAll(jobDir); err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.jobs, id)
	s.mu.Unlock()
	return nil
}

// SetJobStatus updates a job state.
func (s *Store) SetJobStatus(id, status, msg string) {
	s.mu.Lock()
	job, ok := s.jobs[id]
	if !ok {
		s.mu.Unlock()
		return
	}
	job.Status = status
	job.UpdatedAt = time.Now().UTC()
	if msg != "" {
		job.Error = msg
	}
	if status == "done" || status == "failed" {
		now := time.Now().UTC()
		job.FinishedAt = &now
	}
	cloned := cloneJob(job)
	s.mu.Unlock()

	_ = s.saveJobMetadata(cloned)
}

// AddSnapshot records a checkpoint artifact.
func (s *Store) AddSnapshot(jobID string, snapshot Snapshot) {
	s.mu.Lock()
	job, ok := s.jobs[jobID]
	if !ok {
		s.mu.Unlock()
		return
	}
	job.Snapshots = append(job.Snapshots, snapshot)
	job.UpdatedAt = time.Now().UTC()
	cloned := cloneJob(job)
	s.mu.Unlock()

	_ = s.saveJobMetadata(cloned)
}

// SetResult marks a job finished and stores the solver outcome.
func (s *Store) SetResult(jobID string, result solver.SolverResult) {
	s.mu.Lock()
	job, ok := s.jobs[jobID]
	if !ok {
		s.mu.Unlock()
		return
	}
	job.Status = "done"
	job.ResultURL = "/api/jobs/" + jobID + "/result"
	job.Converged = result.Converged
	job.Reason = result.Reason
	job.FinalTime = result.SimTime
	job.FinalStep = result.Step
	job.UpdatedAt = time.Now().UTC()
	now := time.Now().UTC()
	job.FinishedAt = &now
	cloned := cloneJob(job)
	s.mu.Unlock()

	_ = s.saveJobMetadata(cloned)
}

// saveJobMetadata writes the latest job metadata to disk.
func (s *Store) saveJobMetadata(job *Job) error {
	return writeJSONFile(filepath.Join(s.storageDir, "jobs", job.ID, "job.json"), job)
}

func (s *Store) saveUsers(users []*User) error {
	return writeJSONFile(filepath.Join(s.storageDir, "users.json"), users)
}

func (s *Store) loadUsers() error {
	path := filepath.Join(s.storageDir, "users.json")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var users []*User
	if err := readJSONFile(path, &users); err != nil {
		return err
	}
	for _, user := range users {
		if user == nil || user.ID == "" || user.Username == "" || user.PasswordHash == "" {
			continue
		}
		s.users[user.ID] = user
		s.usernames[strings.ToLower(user.Username)] = user.ID
	}
	return nil
}

// loadMetadata rebuilds in-memory upload and job indexes from disk.
func (s *Store) loadMetadata() error {
	uploadFiles, err := filepath.Glob(filepath.Join(s.storageDir, "uploads", "*.json"))
	if err != nil {
		return err
	}
	for _, path := range uploadFiles {
		var upload Upload
		if err := readJSONFile(path, &upload); err != nil {
			log.Printf("skip upload metadata %s: %v", path, err)
			continue
		}
		upload.Path = filepath.Join(s.storageDir, "uploads", upload.ID+".msh")
		if _, err := os.Stat(upload.Path); err != nil {
			continue
		}
		s.uploads[upload.ID] = upload
	}

	jobFiles, err := filepath.Glob(filepath.Join(s.storageDir, "jobs", "*", "job.json"))
	if err != nil {
		return err
	}
	for _, path := range jobFiles {
		var job Job
		if err := readJSONFile(path, &job); err != nil {
			log.Printf("skip job metadata %s: %v", path, err)
			continue
		}
		s.jobs[job.ID] = &job
	}
	return nil
}

func normalizeUsername(username string) (string, string, error) {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 32 {
		return "", "", errors.New("username must be 3 to 32 characters")
	}
	if !usernamePattern.MatchString(username) {
		return "", "", errors.New("username can only use letters, numbers, underscores, periods, and hyphens")
	}
	return username, strings.ToLower(username), nil
}

func (s *Store) cloneUsersLocked() []*User {
	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, cloneUser(user))
	}
	return users
}

func cloneUser(user *User) *User {
	if user == nil {
		return nil
	}
	cp := *user
	return &cp
}

// cloneJob copies a job deeply enough for safe read-only use outside Store.
func cloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	cp := *job
	cp.Snapshots = append([]Snapshot(nil), job.Snapshots...)
	if job.Config != nil {
		cp.Config = make(map[string]interface{}, len(job.Config))
		for k, v := range job.Config {
			cp.Config[k] = v
		}
	}
	return &cp
}
