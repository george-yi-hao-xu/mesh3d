package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mesh3d/web_app/server/solver"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed schema/postgres.sql
var schemaFS embed.FS

// NewPostgresStore creates a Store backed by Postgres metadata and local artifact files.
func NewPostgresStore(storageDir, databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, storageDir: storageDir}, nil
}

func (s *Store) initPostgres() error {
	schema, err := schemaFS.ReadFile("schema/postgres.sql")
	if err != nil {
		return err
	}
	_, err = s.db.Exec(string(schema))
	return err
}

func (s *Store) createUserPostgres(username, password string) (*User, error) {
	username, key, err := normalizeUsername(username)
	if err != nil {
		return nil, err
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	var existing string
	err = s.db.QueryRow(`select id from users where lower(username) = $1`, key).Scan(&existing)
	if err == nil {
		return nil, errors.New("username already exists")
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	hash, err := bcryptPasswordHash(password)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           newID("usr"),
		Username:     username,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	if _, err := s.db.Exec(
		`insert into users (id, username, password_hash, created_at) values ($1, $2, $3, $4)`,
		user.ID, user.Username, user.PasswordHash, user.CreatedAt,
	); err != nil {
		if strings.Contains(err.Error(), "users_username") || strings.Contains(err.Error(), "users_username_lower") {
			return nil, errors.New("username already exists")
		}
		return nil, err
	}
	return cloneUser(user), nil
}

func (s *Store) authenticateUserPostgres(username, password string) (*User, error) {
	_, key, err := normalizeUsername(username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	user, err := s.userByUsernameKey(key)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}
	if err := comparePasswordHash(user.PasswordHash, password); err != nil {
		return nil, errors.New("invalid username or password")
	}
	return user, nil
}

func (s *Store) userByUsernameKey(key string) (*User, error) {
	row := s.db.QueryRow(`select id, username, password_hash, created_at from users where lower(username) = $1`, key)
	return scanUser(row)
}

func (s *Store) getUserPostgres(id string) (*User, bool) {
	user, err := scanUser(s.db.QueryRow(`select id, username, password_hash, created_at from users where id = $1`, id))
	if err != nil {
		return nil, false
	}
	return user, true
}

func (s *Store) saveUploadPostgres(upload Upload) error {
	objectKey := filepath.ToSlash(filepath.Join("uploads", upload.ID+".mesh"))
	_, err := s.db.Exec(
		`insert into uploads (
			id, user_id, file_name, size_bytes, object_key, mesh_kind, point_count, edge_count, created_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		upload.ID, upload.UserID, upload.FileName, upload.Size, objectKey, upload.MeshKind,
		upload.PointCount, upload.EdgeCount, upload.CreatedAt,
	)
	return err
}

func (s *Store) uploadForUser(uploadID, userID string) (Upload, bool) {
	var upload Upload
	var objectKey string
	err := s.db.QueryRow(
		`select id, user_id, file_name, size_bytes, object_key, mesh_kind, coalesce(point_count, 0), coalesce(edge_count, 0), created_at
		 from uploads where id = $1 and user_id = $2`,
		uploadID, userID,
	).Scan(&upload.ID, &upload.UserID, &upload.FileName, &upload.Size, &objectKey, &upload.MeshKind, &upload.PointCount, &upload.EdgeCount, &upload.CreatedAt)
	if err != nil {
		return Upload{}, false
	}
	upload.Path = s.artifactPath(objectKey)
	return upload, true
}

func (s *Store) listUploadsPostgres(userID string) []Upload {
	rows, err := s.db.Query(`
		select id, user_id, file_name, size_bytes, object_key, mesh_kind,
		       coalesce(point_count, 0), coalesce(edge_count, 0), created_at
		from uploads
		where user_id = $1
		order by created_at desc`, userID)
	if err != nil {
		log.Printf("list uploads: %v", err)
		return []Upload{}
	}
	defer rows.Close()

	uploads := make([]Upload, 0)
	for rows.Next() {
		var upload Upload
		var objectKey string
		if err := rows.Scan(
			&upload.ID, &upload.UserID, &upload.FileName, &upload.Size, &objectKey, &upload.MeshKind,
			&upload.PointCount, &upload.EdgeCount, &upload.CreatedAt,
		); err != nil {
			log.Printf("scan upload: %v", err)
			continue
		}
		upload.Path = s.artifactPath(objectKey)
		uploads = append(uploads, upload)
	}
	return uploads
}

func (s *Store) deleteUploadPostgres(userID, uploadID string) error {
	rows, err := s.db.Query(`select id from jobs where upload_id = $1 and user_id = $2 order by created_at desc`, uploadID, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var jobIDs []string
	for rows.Next() {
		var jobID string
		if err := rows.Scan(&jobID); err != nil {
			return err
		}
		jobIDs = append(jobIDs, jobID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(jobIDs) > 0 {
		return uploadInUseError{jobIDs: jobIDs}
	}

	result, err := s.db.Exec(`delete from uploads where id = $1 and user_id = $2`, uploadID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errUploadNotFound
	}
	return nil
}

func (s *Store) insertJobPostgres(job *Job) error {
	configJSON, err := json.Marshal(job.Config)
	if err != nil {
		return err
	}
	inputObjectKey := filepath.ToSlash(filepath.Join("jobs", job.ID, "input.mesh"))
	_, err = s.db.Exec(
		`insert into jobs (
			id, user_id, upload_id, name, input_name, input_object_key, config, status,
			converged, created_at, updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		job.ID, job.UserID, job.UploadID, nullableString(job.Name), job.InputName, inputObjectKey,
		configJSON, job.Status, job.Converged, job.CreatedAt, job.UpdatedAt,
	)
	return err
}

func (s *Store) listJobsPostgres(userID string) []*Job {
	rows, err := s.db.Query(`
		select id, user_id, upload_id, name, input_name, input_object_key, config, status,
		       result_object_key, converged, reason, final_time, final_step, error,
		       created_at, updated_at, finished_at
		from jobs
		where user_id = $1
		order by created_at desc`, userID)
	if err != nil {
		log.Printf("list jobs: %v", err)
		return nil
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := scanJobRows(rows)
		if err != nil {
			log.Printf("scan job: %v", err)
			continue
		}
		if err := s.loadSnapshots(job); err != nil {
			log.Printf("load snapshots for %s: %v", job.ID, err)
		}
		jobs = append(jobs, job)
	}
	return jobs
}

func (s *Store) getJobPostgres(id string) (*Job, bool) {
	job, err := s.queryJob(`where id = $1`, id)
	if err != nil {
		return nil, false
	}
	return job, true
}

func (s *Store) getJobForUserPostgres(userID, id string) (*Job, bool) {
	job, err := s.queryJob(`where id = $1 and user_id = $2`, id, userID)
	if err != nil {
		return nil, false
	}
	return job, true
}

func (s *Store) queryJob(where string, args ...interface{}) (*Job, error) {
	query := fmt.Sprintf(`
		select id, user_id, upload_id, name, input_name, input_object_key, config, status,
		       result_object_key, converged, reason, final_time, final_step, error,
		       created_at, updated_at, finished_at
		from jobs %s`, where)
	job, err := scanJobRows(s.db.QueryRow(query, args...))
	if err != nil {
		return nil, err
	}
	if err := s.loadSnapshots(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Store) deleteJobForUserPostgres(userID, id string) error {
	job, ok := s.getJobForUserPostgres(userID, id)
	if !ok {
		return errJobNotFound
	}
	if job.Status == "queued" || job.Status == "running" {
		return errJobNotDeletable
	}
	if err := os.RemoveAll(s.jobDir(id)); err != nil {
		return err
	}
	_, err := s.db.Exec(`delete from jobs where id = $1 and user_id = $2`, id, userID)
	return err
}

func (s *Store) setJobStatusPostgres(id, status, msg string) {
	now := time.Now().UTC()
	var finishedAt interface{}
	if status == "done" || status == "failed" {
		finishedAt = now
	}
	_, err := s.db.Exec(
		`update jobs
		 set status = $2,
		     error = case when $3 <> '' then $3 else error end,
		     updated_at = $4,
		     finished_at = coalesce($5, finished_at)
		 where id = $1`,
		id, status, msg, now, finishedAt,
	)
	if err != nil {
		log.Printf("set job status %s: %v", id, err)
	}
}

func (s *Store) addSnapshotPostgres(jobID string, snapshot Snapshot) {
	fileName := filepath.Base(snapshot.Path)
	if fileName == "." || fileName == string(filepath.Separator) || fileName == "" {
		fileName = filepath.Base(snapshot.URL)
	}
	objectKey := filepath.ToSlash(filepath.Join("jobs", jobID, "snapshots", fileName))
	_, err := s.db.Exec(
		`insert into job_snapshots (job_id, label, sim_time, step, object_key, created_at)
		 values ($1, $2, $3, $4, $5, $6)`,
		jobID, snapshot.Label, snapshot.SimTime, snapshot.Step, objectKey, snapshot.CreatedAt,
	)
	if err != nil {
		log.Printf("add snapshot %s: %v", jobID, err)
		return
	}
	if _, err := s.db.Exec(`update jobs set updated_at = $2 where id = $1`, jobID, time.Now().UTC()); err != nil {
		log.Printf("touch job %s: %v", jobID, err)
	}
}

func (s *Store) setResultPostgres(jobID string, result solver.SolverResult) {
	now := time.Now().UTC()
	objectKey := filepath.ToSlash(filepath.Join("jobs", jobID, "final.mesh"))
	_, err := s.db.Exec(
		`update jobs
		 set status = 'done',
		     result_object_key = $2,
		     converged = $3,
		     reason = $4,
		     final_time = $5,
		     final_step = $6,
		     updated_at = $7,
		     finished_at = $7
		 where id = $1`,
		jobID, objectKey, result.Converged, nullableString(result.Reason), result.SimTime, result.Step, now,
	)
	if err != nil {
		log.Printf("set result %s: %v", jobID, err)
	}
}

func (s *Store) loadSnapshots(job *Job) error {
	rows, err := s.db.Query(
		`select label, sim_time, step, object_key, created_at
		 from job_snapshots
		 where job_id = $1
		 order by step asc, id asc`, job.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	job.Snapshots = nil
	for rows.Next() {
		var snapshot Snapshot
		var objectKey string
		if err := rows.Scan(&snapshot.Label, &snapshot.SimTime, &snapshot.Step, &objectKey, &snapshot.CreatedAt); err != nil {
			return err
		}
		snapshot.Path = s.artifactPath(objectKey)
		snapshot.URL = "/api/jobs/" + job.ID + "/snapshots/" + filepath.Base(objectKey)
		job.Snapshots = append(job.Snapshots, snapshot)
	}
	return rows.Err()
}

func (s *Store) artifactPath(objectKey string) string {
	return filepath.Join(s.storageDir, filepath.FromSlash(objectKey))
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanUser(row rowScanner) (*User, error) {
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func scanJobRows(row rowScanner) (*Job, error) {
	var job Job
	var name, resultObjectKey, reason, errorText sql.NullString
	var finalTime sql.NullFloat64
	var finalStep sql.NullInt64
	var finishedAt sql.NullTime
	var inputObjectKey string
	var configJSON []byte
	err := row.Scan(
		&job.ID, &job.UserID, &job.UploadID, &name, &job.InputName, &inputObjectKey, &configJSON, &job.Status,
		&resultObjectKey, &job.Converged, &reason, &finalTime, &finalStep, &errorText,
		&job.CreatedAt, &job.UpdatedAt, &finishedAt,
	)
	if err != nil {
		return nil, err
	}
	if name.Valid {
		job.Name = name.String
	}
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &job.Config); err != nil {
			return nil, err
		}
	}
	if resultObjectKey.Valid {
		job.ResultURL = "/api/jobs/" + job.ID + "/result"
	}
	if reason.Valid {
		job.Reason = reason.String
	}
	if finalTime.Valid {
		job.FinalTime = finalTime.Float64
	}
	if finalStep.Valid {
		job.FinalStep = int(finalStep.Int64)
	}
	if errorText.Valid {
		job.Error = errorText.String
	}
	if finishedAt.Valid {
		job.FinishedAt = &finishedAt.Time
	}
	return &job, nil
}

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
