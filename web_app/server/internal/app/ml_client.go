package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type mlTrainRequest struct {
	ClusterID string          `json:"clusterId"`
	Rows      []mlTrainingRow `json:"rows"`
}

type mlTrainResponse struct {
	Metrics       map[string]interface{} `json:"metrics"`
	ModelArtifact string                 `json:"modelArtifact"`
}

type mlRecommendRequest struct {
	ModelArtifact  string                `json:"modelArtifact"`
	MeshFeatures   map[string]float64    `json:"meshFeatures"`
	CandidateCount int                   `json:"candidateCount"`
	ReturnCount    int                   `json:"returnCount"`
	Bounds         map[string][2]float64 `json:"bounds"`
}

type mlRecommendation struct {
	Rank           int                    `json:"rank"`
	Config         map[string]interface{} `json:"config"`
	PredictedScore float64                `json:"predictedScore"`
	PredictedTags  []string               `json:"predictedTags"`
}

type mlRecommendResponse struct {
	Recommendations []mlRecommendation `json:"recommendations"`
}

func trainWithMLSidecar(req mlTrainRequest) (*mlTrainResponse, error) {
	var res mlTrainResponse
	if err := postMLJSON("/train", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func recommendWithMLSidecar(req mlRecommendRequest) (*mlRecommendResponse, error) {
	var res mlRecommendResponse
	if err := postMLJSON("/recommend", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func postMLJSON(path string, req interface{}, out interface{}) error {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("MESH3D_ML_URL")), "/")
	if baseURL == "" {
		return errors.New("MESH3D_ML_URL is not configured")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	httpReq, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(os.Getenv("MESH3D_ML_API_KEY")); apiKey != "" {
		httpReq.Header.Set("X-Mesh3D-ML-Key", apiKey)
	}
	httpRes, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpRes.Body.Close()
	if httpRes.StatusCode < 200 || httpRes.StatusCode >= 300 {
		var errBody struct {
			Error  string `json:"error"`
			Detail string `json:"detail"`
		}
		_ = json.NewDecoder(httpRes.Body).Decode(&errBody)
		msg := errBody.Error
		if msg == "" {
			msg = errBody.Detail
		}
		if msg == "" {
			msg = httpRes.Status
		}
		return fmt.Errorf("ml sidecar: %s", msg)
	}
	return json.NewDecoder(httpRes.Body).Decode(out)
}

func defaultRecommendationBounds() map[string][2]float64 {
	return map[string][2]float64{
		"stiffness":             {1, 50},
		"dampingFactor":         {0.01, 1},
		"gravity":               {-20, 0},
		"airResistanceFactor":   {0, 0.05},
		"timeStep":              {0.001, 0.03},
		"snapshotInterval":      {0.05, 0.5},
		"maxSimTime":            {10, 180},
		"maxSteps":              {1000, 50000},
		"velocityEpsilon":       {0.0001, 0.01},
		"positionEpsilon":       {0.0001, 0.01},
		"stableFrames":          {10, 120},
		"springSeed":            {0, 1000},
		"maxSpringDist":         {0.1, 5},
		"maxSpringsPerParticle": {1, 12},
		"springConnectProb":     {0.1, 1},
	}
}
