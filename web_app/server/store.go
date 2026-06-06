package main

import (
	"errors"
	"io"
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

func (s *Store) uploadMeshPath(uploadID string) string {
	return filepath.Join(s.storageDir, "uploads", uploadID+".mesh")
}

func (s *Store) jobDir(jobID string) string {
	return filepath.Join(s.storageDir, "jobs", jobID)
}

func (s *Store) jobInputPath(jobID string) string {
	return filepath.Join(s.jobDir(jobID), "input.mesh")
}

func (s *Store) jobSnapshotDir(jobID string) string {
	return filepath.Join(s.jobDir(jobID), "snapshots")
}

func (s *Store) jobSnapshotPath(jobID, fileName string) string {
	return filepath.Join(s.jobSnapshotDir(jobID), fileName)
}

func (s *Store) jobResultPath(jobID string) string {
	return filepath.Join(s.jobDir(jobID), "final.mesh")
}

func (s *Store) jobArtifactPath(jobID, relPath string) string {
	return filepath.Join(s.jobDir(jobID), relPath)
}

// Init creates local artifact folders and applies the Postgres schema.
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
	return s.initPostgres()
}

// CreateUser validates and stores a new user with a bcrypt password hash.
func (s *Store) CreateUser(username, password string) (*User, error) {
	return s.createUserPostgres(username, password)
}

// AuthenticateUser verifies username/password credentials.
func (s *Store) AuthenticateUser(username, password string) (*User, error) {
	return s.authenticateUserPostgres(username, password)
}

// GetUser returns a user by id.
func (s *Store) GetUser(id string) (*User, bool) {
	return s.getUserPostgres(id)
}

// SaveUpload persists an uploaded point-cloud file and its metadata.
func (s *Store) SaveUpload(userID string, file multipart.File, header *multipart.FileHeader) (Upload, error) {
	id := newID("upl")
	fileName := filepath.Base(header.Filename)
	if fileName == "." || fileName == string(filepath.Separator) || fileName == "" {
		fileName = "mesh.mesh"
	}

	path := s.uploadMeshPath(id)
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
	if err := s.saveUploadPostgres(upload); err != nil {
		return Upload{}, err
	}
	return upload, nil
}

// CreateJob creates a job folder with input copied from a prior upload.
func (s *Store) CreateJob(userID, uploadID, name string, config map[string]interface{}) (*Job, error) {
	upload, ok := s.uploadForUser(uploadID, userID)
	if !ok {
		return nil, errors.New("upload not found")
	}

	id := newID("job")
	if err := os.MkdirAll(s.jobSnapshotDir(id), 0755); err != nil {
		return nil, err
	}
	if err := copyFile(upload.Path, s.jobInputPath(id)); err != nil {
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
	if err := s.insertJobPostgres(job); err != nil {
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

// ListJobs returns user-owned job records.
func (s *Store) ListJobs(userID string) []*Job {
	return s.listJobsPostgres(userID)
}

// GetJob returns a job record by id.
func (s *Store) GetJob(id string) (*Job, bool) {
	return s.getJobPostgres(id)
}

// GetJobForUser returns a job only when it belongs to the requested user.
func (s *Store) GetJobForUser(userID, id string) (*Job, bool) {
	return s.getJobForUserPostgres(userID, id)
}

// DeleteJobForUser removes a finished job and its stored artifacts when it belongs to the requested user.
func (s *Store) DeleteJobForUser(userID, id string) error {
	return s.deleteJobForUserPostgres(userID, id)
}

// SetJobStatus updates a job state.
func (s *Store) SetJobStatus(id, status, msg string) {
	s.setJobStatusPostgres(id, status, msg)
}

// AddSnapshot records a checkpoint artifact.
func (s *Store) AddSnapshot(jobID string, snapshot Snapshot) {
	s.addSnapshotPostgres(jobID, snapshot)
}

// SetResult marks a job finished and stores the solver outcome.
func (s *Store) SetResult(jobID string, result solver.SolverResult) {
	s.setResultPostgres(jobID, result)
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

func cloneUser(user *User) *User {
	if user == nil {
		return nil
	}
	cp := *user
	return &cp
}

func bcryptPasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func comparePasswordHash(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
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
