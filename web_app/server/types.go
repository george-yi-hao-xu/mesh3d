package main

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
	Path       string    `json:"-"`
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
	Path      string    `json:"-"`
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
	Status     string                 `json:"status"`
	Config     map[string]interface{} `json:"config"`
	Snapshots  []Snapshot             `json:"snapshots"`
	ResultURL  string                 `json:"resultUrl,omitempty"`
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

type JobCreateResponse struct {
	Job    *Job       `json:"job"`
	Frames []JobFrame `json:"frames"`
}

type Store struct {
	db         *sql.DB
	storageDir string
}

type App struct {
	store     *Store
	clientDir string
	jwtSecret []byte
}
