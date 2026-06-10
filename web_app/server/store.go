package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mesh3d/web_app/server/solver"

	"golang.org/x/crypto/bcrypt"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

var (
	errJobNotFound     = errors.New("job not found")
	errJobNotDeletable = errors.New("job is not finished")
	errUploadNotFound  = errors.New("upload not found")
	errInvalidReview   = errors.New("review score must be between 1 and 5")
)

type uploadInUseError struct {
	jobIDs []string
}

func (e uploadInUseError) Error() string {
	return "mesh is used by existing jobs; delete those jobs first"
}

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

// SaveUpload persists an uploaded mesh file and its metadata.
func (s *Store) SaveUpload(userID string, file multipart.File, header *multipart.FileHeader, meshKind string) (Upload, error) {
	id := newID("upl")
	fileName := filepath.Base(header.Filename)
	if fileName == "." || fileName == string(filepath.Separator) || fileName == "" {
		fileName = "mesh.mesh"
	}
	if meshKind != "generated" {
		meshKind = "uploaded"
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
	if err := out.Close(); err != nil {
		return Upload{}, err
	}

	pointCount, edgeCount, err := inspectMeshFile(path)
	if err != nil {
		_ = os.Remove(path)
		return Upload{}, err
	}

	upload := Upload{
		ID:         id,
		UserID:     userID,
		FileName:   fileName,
		Size:       size,
		MeshKind:   meshKind,
		PointCount: pointCount,
		EdgeCount:  edgeCount,
		Path:       path,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.saveUploadPostgres(upload); err != nil {
		return Upload{}, err
	}
	return upload, nil
}

// ListUploads returns user-owned stored mesh artifacts.
func (s *Store) ListUploads(userID string) []Upload {
	return s.listUploadsPostgres(userID)
}

// GetUploadForUser returns a stored mesh artifact only when it belongs to the requested user.
func (s *Store) GetUploadForUser(userID, uploadID string) (Upload, bool) {
	return s.uploadForUser(uploadID, userID)
}

// DeleteUploadForUser removes an unused user-owned warehouse mesh artifact.
func (s *Store) DeleteUploadForUser(userID, uploadID string) error {
	upload, ok := s.uploadForUser(uploadID, userID)

	if !ok {
		return errUploadNotFound
	}

	if err := s.deleteUploadPostgres(userID, uploadID); err != nil {
		return err
	}

	if err := os.Remove(upload.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// CreateJob creates a job folder with input copied from a prior upload.
func (s *Store) CreateJob(userID, uploadID, name string, config map[string]interface{}) (*Job, error) {
	upload, ok := s.uploadForUser(uploadID, userID)
	if !ok {
		return nil, errors.New("upload not found")
	}
	if upload.EdgeCount <= 0 {
		return nil, errors.New("mesh contains no springs; enable generated springs or upload a mesh with existing springs")
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

func inspectMeshFile(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	section := ""
	hasMeshFormat := false
	pointCount := 0
	edgeCount := 0
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "# Format:")), "mesh-v1") {
				hasMeshFormat = true
			}
			continue
		}

		lower := strings.ToLower(line)
		if lower == "vertices" || lower == "edges" {
			section = lower
			continue
		}

		fields := strings.Fields(line)
		if hasMeshFormat {
			switch section {
			case "vertices":
				if len(fields) < 6 {
					return 0, 0, fmt.Errorf("invalid mesh vertex line %d", lineNumber)
				}
				if _, err := strconv.Atoi(fields[0]); err != nil {
					return 0, 0, fmt.Errorf("invalid mesh vertex index on line %d", lineNumber)
				}
				for _, raw := range []string{fields[1], fields[2], fields[3], fields[5]} {
					if _, err := strconv.ParseFloat(raw, 64); err != nil {
						return 0, 0, fmt.Errorf("invalid mesh vertex value on line %d", lineNumber)
					}
				}
				pointCount++
			case "edges":
				if len(fields) < 2 {
					return 0, 0, fmt.Errorf("invalid mesh edge line %d", lineNumber)
				}
				if _, err := strconv.Atoi(fields[0]); err != nil {
					return 0, 0, fmt.Errorf("invalid mesh edge value on line %d", lineNumber)
				}
				if _, err := strconv.Atoi(fields[1]); err != nil {
					return 0, 0, fmt.Errorf("invalid mesh edge value on line %d", lineNumber)
				}
				edgeCount++
			default:
				return 0, 0, fmt.Errorf("mesh line %d appears before vertices or edges section", lineNumber)
			}
			continue
		}

		if len(fields) < 3 {
			continue
		}
		x, errX := strconv.ParseFloat(fields[0], 64)
		y, errY := strconv.ParseFloat(fields[1], 64)
		z, errZ := strconv.ParseFloat(fields[2], 64)
		if errX == nil && errY == nil && errZ == nil && finite3(x, y, z) {
			pointCount++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}
	if pointCount == 0 {
		return 0, 0, errors.New("no valid mesh points found")
	}
	return pointCount, edgeCount, nil
}

func finite3(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
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

// SaveJobReviewForUser stores a user's label for one of their jobs.
func (s *Store) SaveJobReviewForUser(userID, jobID string, score int, tags []string, note string) (*JobReview, error) {
	if score < 1 || score > 5 {
		return nil, errInvalidReview
	}
	if _, ok := s.GetJobForUser(userID, jobID); !ok {
		return nil, errJobNotFound
	}
	review := JobReview{
		JobID:     jobID,
		UserID:    userID,
		Score:     score,
		Tags:      normalizeReviewTags(tags),
		Note:      strings.TrimSpace(note),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.upsertJobReviewPostgres(&review); err != nil {
		return nil, err
	}
	return &review, nil
}

func normalizeReviewTags(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		tag = strings.Join(strings.Fields(tag), "-")
		if tag == "" || seen[tag] {
			continue
		}
		if len(tag) > 32 {
			tag = tag[:32]
		}
		seen[tag] = true
		normalized = append(normalized, tag)
		if len(normalized) >= 12 {
			break
		}
	}
	return normalized
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
	if job.Review != nil {
		review := *job.Review
		review.Tags = append([]string(nil), job.Review.Tags...)
		cp.Review = &review
	}
	if job.Config != nil {
		cp.Config = make(map[string]interface{}, len(job.Config))
		for k, v := range job.Config {
			cp.Config[k] = v
		}
	}
	return &cp
}
