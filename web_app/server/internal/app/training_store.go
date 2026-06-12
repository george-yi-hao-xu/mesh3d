package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const minTrainingReviews = 20

var (
	errTrainingClusterNotFound = errors.New("training cluster not found")
	errTrainingJobNotFound     = errors.New("reviewed job not found")
	errTrainingDataTooSmall    = errors.New("training cluster needs at least 20 reviewed jobs")
)

func (s *Store) ListTrainingClusters(userID string) ([]*TrainingCluster, error) {
	rows, err := s.db.Query(`
		select id, user_id, name, status, created_at, updated_at
		from training_clusters
		where user_id = $1
		order by updated_at desc, created_at desc`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clusters := make([]*TrainingCluster, 0)
	for rows.Next() {
		cluster, err := scanTrainingCluster(rows)
		if err != nil {
			return nil, err
		}
		if err := s.loadTrainingClusterDetails(cluster); err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}
	return clusters, rows.Err()
}

func (s *Store) CreateTrainingCluster(userID, name string) (*TrainingCluster, error) {
	name = normalizeClusterName(name)
	now := time.Now().UTC()
	cluster := &TrainingCluster{
		ID:        newID("trc"),
		UserID:    userID,
		Name:      name,
		Status:    "ready",
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.Exec(
		`insert into training_clusters (id, user_id, name, status, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, $6)`,
		cluster.ID, cluster.UserID, cluster.Name, cluster.Status, cluster.CreatedAt, cluster.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (s *Store) GetTrainingClusterForUser(userID, clusterID string) (*TrainingCluster, error) {
	cluster, err := scanTrainingCluster(s.db.QueryRow(`
		select id, user_id, name, status, created_at, updated_at
		from training_clusters
		where id = $1 and user_id = $2`, clusterID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errTrainingClusterNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.loadTrainingClusterDetails(cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func (s *Store) RenameTrainingCluster(userID, clusterID, name string) (*TrainingCluster, error) {
	name = normalizeClusterName(name)
	result, err := s.db.Exec(
		`update training_clusters set name = $3, updated_at = $4 where id = $1 and user_id = $2`,
		clusterID, userID, name, time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, errTrainingClusterNotFound
	}
	return s.GetTrainingClusterForUser(userID, clusterID)
}

func (s *Store) DeleteTrainingCluster(userID, clusterID string) error {
	result, err := s.db.Exec(`delete from training_clusters where id = $1 and user_id = $2`, clusterID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errTrainingClusterNotFound
	}
	return nil
}

func (s *Store) AddJobToTrainingCluster(userID, clusterID, jobID string) (*TrainingCluster, error) {
	if _, err := s.GetTrainingClusterForUser(userID, clusterID); err != nil {
		return nil, err
	}
	job, ok := s.GetJobForUser(userID, jobID)
	if !ok || job.Review == nil {
		return nil, errTrainingJobNotFound
	}
	now := time.Now().UTC()
	_, err := s.db.Exec(
		`insert into training_cluster_jobs (cluster_id, job_id, added_at)
		 values ($1, $2, $3)
		 on conflict (cluster_id, job_id) do nothing`,
		clusterID, jobID, now,
	)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.Exec(`update training_clusters set updated_at = $2 where id = $1`, clusterID, now)
	return s.GetTrainingClusterForUser(userID, clusterID)
}

func (s *Store) RemoveJobFromTrainingCluster(userID, clusterID, jobID string) (*TrainingCluster, error) {
	if _, err := s.GetTrainingClusterForUser(userID, clusterID); err != nil {
		return nil, err
	}
	_, err := s.db.Exec(`delete from training_cluster_jobs where cluster_id = $1 and job_id = $2`, clusterID, jobID)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.Exec(`update training_clusters set updated_at = $2 where id = $1`, clusterID, time.Now().UTC())
	return s.GetTrainingClusterForUser(userID, clusterID)
}

func (s *Store) ReviewedJobsForTrainingCluster(userID, clusterID string) ([]*Job, error) {
	if _, err := s.GetTrainingClusterForUser(userID, clusterID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`
		select j.id
		from training_cluster_jobs tcj
		join jobs j on j.id = tcj.job_id
		join job_reviews jr on jr.job_id = j.id
		where tcj.cluster_id = $1 and j.user_id = $2
		order by tcj.added_at asc`, clusterID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]*Job, 0)
	for rows.Next() {
		var jobID string
		if err := rows.Scan(&jobID); err != nil {
			return nil, err
		}
		job, ok := s.GetJobForUser(userID, jobID)
		if ok && job.Review != nil {
			jobs = append(jobs, job)
		}
	}
	return jobs, rows.Err()
}

func (s *Store) CreateTrainingRun(clusterID string) (*TrainingRun, error) {
	now := time.Now().UTC()
	run := &TrainingRun{
		ID:        newID("trn"),
		ClusterID: clusterID,
		Status:    "running",
		Metrics:   map[string]interface{}{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.Exec(
		`insert into training_runs (id, cluster_id, status, metrics, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, $6)`,
		run.ID, run.ClusterID, run.Status, []byte(`{}`), run.CreatedAt, run.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.Exec(`update training_clusters set status = 'training', updated_at = $2 where id = $1`, clusterID, now)
	return run, nil
}

func (s *Store) FinishTrainingRun(runID, status string, metrics map[string]interface{}, modelArtifact, errorText string) (*TrainingRun, error) {
	now := time.Now().UTC()
	if metrics == nil {
		metrics = map[string]interface{}{}
	}
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec(
		`update training_runs
		 set status = $2, metrics = $3, model_artifact = $4, error = $5, updated_at = $6, finished_at = $6
		 where id = $1`,
		runID, status, metricsJSON, nullableString(modelArtifact), nullableString(errorText), now,
	)
	if err != nil {
		return nil, err
	}
	run, err := s.trainingRun(runID)
	if err == nil {
		clusterStatus := "ready"
		if status == "trained" {
			clusterStatus = "trained"
		}
		_, _ = s.db.Exec(`update training_clusters set status = $2, updated_at = $3 where id = $1`, run.ClusterID, clusterStatus, now)
	}
	return run, err
}

func (s *Store) SaveConfigRecommendations(runID string, recommendations []ConfigRecommendation) error {
	_, err := s.db.Exec(`delete from config_recommendations where run_id = $1`, runID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, rec := range recommendations {
		configJSON, err := json.Marshal(rec.Config)
		if err != nil {
			return err
		}
		tagsJSON, err := json.Marshal(rec.PredictedTags)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(
			`insert into config_recommendations (run_id, rank, config, predicted_score, predicted_tags, created_at)
			 values ($1, $2, $3, $4, $5, $6)`,
			runID, rec.Rank, configJSON, rec.PredictedScore, tagsJSON, now,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadTrainingClusterDetails(cluster *TrainingCluster) error {
	jobs, err := s.trainingClusterJobs(cluster.ID)
	if err != nil {
		return err
	}
	cluster.Jobs = jobs
	run, err := s.latestTrainingRun(cluster.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	cluster.LatestRun = run
	recommendations, err := s.configRecommendations(run.ID)
	if err != nil {
		return err
	}
	cluster.Recommendations = recommendations
	return nil
}

func (s *Store) trainingClusterJobs(clusterID string) ([]TrainingClusterJob, error) {
	rows, err := s.db.Query(`
		select job_id, added_at
		from training_cluster_jobs
		where cluster_id = $1
		order by added_at asc`, clusterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TrainingClusterJob, 0)
	for rows.Next() {
		var jobID string
		item := TrainingClusterJob{ClusterID: clusterID}
		if err := rows.Scan(&jobID, &item.AddedAt); err != nil {
			return nil, err
		}
		job, ok := s.GetJob(jobID)
		if ok {
			item.Job = job
			items = append(items, item)
		}
	}
	return items, rows.Err()
}

func (s *Store) latestTrainingRun(clusterID string) (*TrainingRun, error) {
	return scanTrainingRun(s.db.QueryRow(`
		select id, cluster_id, status, metrics, model_artifact, error, created_at, updated_at, finished_at
		from training_runs
		where cluster_id = $1
		order by created_at desc
		limit 1`, clusterID))
}

func (s *Store) trainingRun(runID string) (*TrainingRun, error) {
	return scanTrainingRun(s.db.QueryRow(`
		select id, cluster_id, status, metrics, model_artifact, error, created_at, updated_at, finished_at
		from training_runs
		where id = $1`, runID))
}

func (s *Store) configRecommendations(runID string) ([]ConfigRecommendation, error) {
	rows, err := s.db.Query(`
		select run_id, rank, config, predicted_score, predicted_tags, created_at
		from config_recommendations
		where run_id = $1
		order by rank asc`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recommendations := make([]ConfigRecommendation, 0)
	for rows.Next() {
		rec, err := scanConfigRecommendation(rows)
		if err != nil {
			return nil, err
		}
		recommendations = append(recommendations, *rec)
	}
	return recommendations, rows.Err()
}

func scanTrainingCluster(row rowScanner) (*TrainingCluster, error) {
	var cluster TrainingCluster
	if err := row.Scan(&cluster.ID, &cluster.UserID, &cluster.Name, &cluster.Status, &cluster.CreatedAt, &cluster.UpdatedAt); err != nil {
		return nil, err
	}
	return &cluster, nil
}

func scanTrainingRun(row rowScanner) (*TrainingRun, error) {
	var run TrainingRun
	var metricsJSON []byte
	var modelArtifact, errorText sql.NullString
	var finishedAt sql.NullTime
	if err := row.Scan(
		&run.ID, &run.ClusterID, &run.Status, &metricsJSON, &modelArtifact, &errorText,
		&run.CreatedAt, &run.UpdatedAt, &finishedAt,
	); err != nil {
		return nil, err
	}
	run.Metrics = map[string]interface{}{}
	if len(metricsJSON) > 0 {
		if err := json.Unmarshal(metricsJSON, &run.Metrics); err != nil {
			return nil, err
		}
	}
	if modelArtifact.Valid {
		run.ModelArtifact = modelArtifact.String
	}
	if errorText.Valid {
		run.Error = errorText.String
	}
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}
	return &run, nil
}

func scanConfigRecommendation(row rowScanner) (*ConfigRecommendation, error) {
	var rec ConfigRecommendation
	var configJSON, tagsJSON []byte
	if err := row.Scan(&rec.RunID, &rec.Rank, &configJSON, &rec.PredictedScore, &tagsJSON, &rec.CreatedAt); err != nil {
		return nil, err
	}
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &rec.Config); err != nil {
			return nil, err
		}
	}
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &rec.PredictedTags); err != nil {
			return nil, err
		}
	}
	return &rec, nil
}

func normalizeClusterName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Training cluster"
	}
	if len(name) > 80 {
		return name[:80]
	}
	return name
}
