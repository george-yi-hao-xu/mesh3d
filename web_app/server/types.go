package main

import (
	"sync"
	"time"
)

type Upload struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	FileName  string    `json:"fileName"`
	Size      int64     `json:"size"`
	Path      string    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
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
}

type Event struct {
	Type     string      `json:"type"`
	JobID    string      `json:"jobId"`
	Snapshot *Snapshot   `json:"snapshot,omitempty"`
	Job      *Job        `json:"job,omitempty"`
	Error    string      `json:"error,omitempty"`
	Payload  interface{} `json:"payload,omitempty"`
}

type Store struct {
	mu          sync.Mutex
	storageDir  string
	users       map[string]*User
	usernames   map[string]string
	uploads     map[string]Upload
	jobs        map[string]*Job
	subscribers map[string]map[chan Event]struct{}
}

type App struct {
	store     *Store
	clientDir string
	jwtSecret []byte
}
