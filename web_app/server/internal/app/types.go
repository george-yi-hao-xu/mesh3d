package app

import (
	"database/sql"
	"time"
)

type Upload struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	FileName   string    `json:"fileName"`
	Size       int64     `json:"size"`
	MeshKind   string    `json:"meshKind"`
	PointCount int       `json:"pointCount"`
	EdgeCount  int       `json:"edgeCount"`
	MeshText   string    `json:"-"`
	CreatedAt  time.Time `json:"createdAt"`
}

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"passwordHash"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Snapshot struct {
	Label     string    `json:"label"`
	SimTime   float64   `json:"simTime"`
	Step      int       `json:"step"`
	FileName  string    `json:"-"`
	MeshText  string    `json:"-"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
}

type JobReview struct {
	JobID     string    `json:"jobId"`
	UserID    string    `json:"userId,omitempty"`
	Score     int       `json:"score"`
	Tags      []string  `json:"tags"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Job struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"userId"`
	UploadID   string                 `json:"uploadId"`
	Name       string                 `json:"name,omitempty"`
	InputName  string                 `json:"inputName"`
	InputText  string                 `json:"-"`
	Status     string                 `json:"status"`
	Config     map[string]interface{} `json:"config"`
	Snapshots  []Snapshot             `json:"snapshots"`
	ResultURL  string                 `json:"resultUrl,omitempty"`
	ResultText string                 `json:"-"`
	Converged  bool                   `json:"converged"`
	Reason     string                 `json:"reason,omitempty"`
	FinalTime  float64                `json:"finalTime,omitempty"`
	FinalStep  int                    `json:"finalStep,omitempty"`
	Error      string                 `json:"error,omitempty"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	FinishedAt *time.Time             `json:"finishedAt,omitempty"`
	Review     *JobReview             `json:"review,omitempty"`
}

type JobFrame struct {
	Label   string  `json:"label"`
	URL     string  `json:"url"`
	Text    string  `json:"text"`
	IsFinal bool    `json:"isFinal"`
	SimTime float64 `json:"simTime,omitempty"`
	Step    int     `json:"step,omitempty"`
}

type TrainingCluster struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"userId,omitempty"`
	Name            string                 `json:"name"`
	Status          string                 `json:"status"`
	Jobs            []TrainingClusterJob   `json:"jobs"`
	LatestRun       *TrainingRun           `json:"latestRun,omitempty"`
	Recommendations []ConfigRecommendation `json:"recommendations,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

type TrainingClusterJob struct {
	ClusterID string    `json:"clusterId,omitempty"`
	Job       *Job      `json:"job"`
	AddedAt   time.Time `json:"addedAt"`
}

type TrainingRun struct {
	ID            string                 `json:"id"`
	ClusterID     string                 `json:"clusterId"`
	Status        string                 `json:"status"`
	Metrics       map[string]interface{} `json:"metrics"`
	ModelArtifact string                 `json:"modelArtifact,omitempty"`
	Error         string                 `json:"error,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
	FinishedAt    *time.Time             `json:"finishedAt,omitempty"`
}

type ConfigRecommendation struct {
	RunID          string                 `json:"runId,omitempty"`
	Rank           int                    `json:"rank"`
	Config         map[string]interface{} `json:"config"`
	PredictedScore float64                `json:"predictedScore"`
	PredictedTags  []string               `json:"predictedTags"`
	CreatedAt      time.Time              `json:"createdAt"`
}

type JobCreateResponse struct {
	Job    *Job       `json:"job"`
	Frames []JobFrame `json:"frames"`
}

type Store struct {
	db *sql.DB
}

type App struct {
	store     *Store
	clientDir string
	jwtSecret []byte
}
