package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

func (a *App) handleTrainingClusters(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	switch r.Method {
	case http.MethodGet:
		clusters, err := a.store.ListTrainingClusters(user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, clusters)
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		cluster, err := a.store.CreateTrainingCluster(user.ID, req.Name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, cluster)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *App) handleTrainingRoutes(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/training/clusters/"))
	if len(parts) == 0 || !safePathPart(parts[0]) {
		writeError(w, http.StatusNotFound, "training cluster not found")
		return
	}
	clusterID := parts[0]
	if len(parts) == 1 {
		a.handleTrainingCluster(w, r, user.ID, clusterID)
		return
	}
	switch parts[1] {
	case "jobs":
		if len(parts) == 2 {
			a.handleTrainingClusterJobs(w, r, user.ID, clusterID)
			return
		}
		if len(parts) == 3 && r.Method == http.MethodDelete {
			cluster, err := a.store.RemoveJobFromTrainingCluster(user.ID, clusterID, parts[2])
			writeTrainingResult(w, cluster, err)
			return
		}
	case "train":
		if len(parts) == 2 {
			a.handleTrainingRun(w, r, user.ID, clusterID)
			return
		}
	case "recommend":
		if len(parts) == 2 {
			a.handleTrainingRecommend(w, r, user.ID, clusterID)
			return
		}
	}
	writeError(w, http.StatusNotFound, "not found")
}

func (a *App) handleTrainingCluster(w http.ResponseWriter, r *http.Request, userID, clusterID string) {
	switch r.Method {
	case http.MethodGet:
		cluster, err := a.store.GetTrainingClusterForUser(userID, clusterID)
		writeTrainingResult(w, cluster, err)
	case http.MethodPut:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		cluster, err := a.store.RenameTrainingCluster(userID, clusterID, req.Name)
		writeTrainingResult(w, cluster, err)
	case http.MethodDelete:
		if err := a.store.DeleteTrainingCluster(userID, clusterID); err != nil {
			writeTrainingResult(w, nil, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *App) handleTrainingClusterJobs(w http.ResponseWriter, r *http.Request, userID, clusterID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		JobID string `json:"jobId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cluster, err := a.store.AddJobToTrainingCluster(userID, clusterID, req.JobID)
	writeTrainingResult(w, cluster, err)
}

// #region: ML Train
func (a *App) handleTrainingRun(w http.ResponseWriter, r *http.Request, userID, clusterID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	jobs, err := a.store.ReviewedJobsForTrainingCluster(userID, clusterID)

	if err != nil {
		writeTrainingResult(w, nil, err)
		return
	}
	if len(jobs) < minTrainingReviews {
		writeError(w, http.StatusBadRequest, errTrainingDataTooSmall.Error())
		return
	}

	rows, err := trainingRowsForJobs(a.store, jobs)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	run, err := a.store.CreateTrainingRun(clusterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	mlRes, err := trainWithMLSidecar(mlTrainRequest{ClusterID: clusterID, Rows: rows})

	if err != nil {
		run, _ = a.store.FinishTrainingRun(run.ID, "failed", nil, "", err.Error())
		writeJSON(w, http.StatusBadGateway, run)
		return
	}

	run, err = a.store.FinishTrainingRun(run.ID, "trained", mlRes.Metrics, mlRes.ModelArtifact, "")

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, run)
}

func (a *App) handleTrainingRecommend(w http.ResponseWriter, r *http.Request, userID, clusterID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		UploadID       string `json:"uploadId"`
		CandidateCount int    `json:"candidateCount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cluster, err := a.store.GetTrainingClusterForUser(userID, clusterID)
	if err != nil {
		writeTrainingResult(w, nil, err)
		return
	}
	if cluster.LatestRun == nil || cluster.LatestRun.Status != "trained" || cluster.LatestRun.ModelArtifact == "" {
		writeError(w, http.StatusBadRequest, "cluster does not have a trained model")
		return
	}
	upload, ok := a.store.GetUploadForUser(userID, req.UploadID)
	if !ok {
		writeError(w, http.StatusNotFound, "upload not found")
		return
	}
	features, err := featureMapForUpload(upload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	candidateCount := req.CandidateCount
	if candidateCount <= 0 {
		candidateCount = 512
	}
	mlRes, err := recommendWithMLSidecar(mlRecommendRequest{
		ModelArtifact:  cluster.LatestRun.ModelArtifact,
		MeshFeatures:   features,
		CandidateCount: candidateCount,
		ReturnCount:    5,
		Bounds:         defaultRecommendationBounds(),
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	recommendations := make([]ConfigRecommendation, 0, len(mlRes.Recommendations))
	for i, rec := range mlRes.Recommendations {
		rank := rec.Rank
		if rank <= 0 {
			rank = i + 1
		}
		recommendations = append(recommendations, ConfigRecommendation{
			RunID:          cluster.LatestRun.ID,
			Rank:           rank,
			Config:         rec.Config,
			PredictedScore: rec.PredictedScore,
			PredictedTags:  rec.PredictedTags,
		})
	}
	if err := a.store.SaveConfigRecommendations(cluster.LatestRun.ID, recommendations); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cluster, err = a.store.GetTrainingClusterForUser(userID, clusterID)
	writeTrainingResult(w, cluster, err)
}

func writeTrainingResult(w http.ResponseWriter, value interface{}, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, value)
		return
	}
	switch {
	case errors.Is(err, errTrainingClusterNotFound), errors.Is(err, errTrainingJobNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errTrainingDataTooSmall):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
