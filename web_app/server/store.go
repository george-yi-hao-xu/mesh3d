package main

import (
	"errors"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"mesh3d/web_app/server/solver"
)

// NewStore creates the in-memory indexes around the configured storage root.
func NewStore(storageDir string) *Store {
	return &Store{
		storageDir:  storageDir,
		uploads:     make(map[string]Upload),
		jobs:        make(map[string]*Job),
		subscribers: make(map[string]map[chan Event]struct{}),
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
	return s.loadMetadata()
}

// SaveUpload persists an uploaded point-cloud file and its metadata.
func (s *Store) SaveUpload(file multipart.File, header *multipart.FileHeader) (Upload, error) {
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
func (s *Store) CreateJob(uploadID string, config map[string]interface{}) (*Job, error) {
	s.mu.Lock()
	upload, ok := s.uploads[uploadID]
	s.mu.Unlock()
	if !ok {
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
	job := &Job{
		ID:        id,
		UploadID:  upload.ID,
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

// ListJobs returns cloned job records so callers cannot mutate store state.
func (s *Store) ListJobs() []*Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, cloneJob(job))
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

// SetJobStatus updates a job state and broadcasts the change.
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
	eventType := status
	if status == "running" {
		eventType = "job"
	}
	s.Publish(id, Event{Type: eventType, JobID: id, Job: cloned, Error: msg})
}

// AddSnapshot records a checkpoint artifact and notifies subscribed clients.
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
	s.Publish(jobID, Event{Type: "snapshot", JobID: jobID, Snapshot: &snapshot})
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
	s.Publish(jobID, Event{Type: "done", JobID: jobID, Job: cloned})
}

// Subscribe registers a buffered event channel for a job.
func (s *Store) Subscribe(jobID string) chan Event {
	ch := make(chan Event, 16)
	s.mu.Lock()
	if s.subscribers[jobID] == nil {
		s.subscribers[jobID] = make(map[chan Event]struct{})
	}
	s.subscribers[jobID][ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

// Unsubscribe removes and closes a job event channel.
func (s *Store) Unsubscribe(jobID string, ch chan Event) {
	s.mu.Lock()
	if subs := s.subscribers[jobID]; subs != nil {
		delete(subs, ch)
	}
	close(ch)
	s.mu.Unlock()
}

// Publish sends an event to every active subscriber for a job.
func (s *Store) Publish(jobID string, event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subscribers[jobID] {
		select {
		case ch <- event:
		default:
		}
	}
}

// saveJobMetadata writes the latest job metadata to disk.
func (s *Store) saveJobMetadata(job *Job) error {
	return writeJSONFile(filepath.Join(s.storageDir, "jobs", job.ID, "job.json"), job)
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
