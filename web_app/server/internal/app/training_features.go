package app

import (
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
)

type mlTrainingRow struct {
	JobID    string             `json:"jobId"`
	Features map[string]float64 `json:"features"`
	Score    int                `json:"score"`
	Tags     []string           `json:"tags"`
	Note     string             `json:"note,omitempty"`
}

type meshFeaturePoint struct {
	x, y, z float64
	fixed   bool
}

func trainingRowsForJobs(store *Store, jobs []*Job) ([]mlTrainingRow, error) {
	rows := make([]mlTrainingRow, 0, len(jobs))
	for _, job := range jobs {
		if job == nil || job.Review == nil {
			continue
		}
		features, err := featureMapForMeshAndJob(store.jobInputPath(job.ID), job)
		if err != nil {
			return nil, err
		}
		rows = append(rows, mlTrainingRow{
			JobID:    job.ID,
			Features: features,
			Score:    job.Review.Score,
			Tags:     append([]string(nil), job.Review.Tags...),
			Note:     job.Review.Note,
		})
	}
	return rows, nil
}

func featureMapForMeshAndJob(meshPath string, job *Job) (map[string]float64, error) {
	features, err := meshFeatureMap(meshPath)
	if err != nil {
		return nil, err
	}
	if job != nil {
		addConfigFeatures(features, job.Config)
		if job.Converged {
			features["outcome_converged"] = 1
		}
		features["outcome_final_time"] = job.FinalTime
		features["outcome_final_step"] = float64(job.FinalStep)
		features["outcome_snapshot_count"] = float64(len(job.Snapshots))
	}
	return features, nil
}

func featureMapForUpload(upload Upload) (map[string]float64, error) {
	return meshFeatureMap(upload.Path)
}

func addConfigFeatures(features map[string]float64, config map[string]interface{}) {
	for _, key := range []string{
		"stiffness", "dampingFactor", "gravity", "airResistanceFactor", "timeStep",
		"snapshotInterval", "maxSimTime", "maxSteps", "velocityEpsilon", "positionEpsilon",
		"stableFrames", "springSeed", "maxSpringDist", "maxSpringsPerParticle", "springConnectProb",
	} {
		if value, ok := numericConfigValue(config, key); ok {
			features["config_"+key] = value
		}
	}
}

func meshFeatureMap(path string) (map[string]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	points := make([]meshFeaturePoint, 0)
	edges := make([][2]int, 0)
	lengths := make([]float64, 0)
	section := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lower := strings.ToLower(line)
		if lower == "vertices" || lower == "edges" {
			section = lower
			continue
		}
		fields := strings.Fields(line)
		switch section {
		case "vertices":
			if len(fields) < 6 {
				continue
			}
			x, errX := strconv.ParseFloat(fields[1], 64)
			y, errY := strconv.ParseFloat(fields[2], 64)
			z, errZ := strconv.ParseFloat(fields[3], 64)
			if errX == nil && errY == nil && errZ == nil {
				points = append(points, meshFeaturePoint{x: x, y: y, z: z, fixed: fields[4] == "1" || strings.EqualFold(fields[4], "true")})
			}
		case "edges":
			if len(fields) < 2 {
				continue
			}
			a, errA := strconv.Atoi(fields[0])
			b, errB := strconv.Atoi(fields[1])
			if errA != nil || errB != nil {
				continue
			}
			edges = append(edges, [2]int{a, b})
			if len(fields) >= 3 {
				if length, err := strconv.ParseFloat(fields[2], 64); err == nil && !math.IsNaN(length) && !math.IsInf(length, 0) && length > 0 {
					lengths = append(lengths, length)
				}
			}
		default:
			if len(fields) >= 3 {
				x, errX := strconv.ParseFloat(fields[0], 64)
				y, errY := strconv.ParseFloat(fields[1], 64)
				z, errZ := strconv.ParseFloat(fields[2], 64)
				if errX == nil && errY == nil && errZ == nil {
					points = append(points, meshFeaturePoint{x: x, y: y, z: z})
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	degrees := make([]int, len(points))
	for _, edge := range edges {
		if edge[0] >= 0 && edge[0] < len(degrees) {
			degrees[edge[0]]++
		}
		if edge[1] >= 0 && edge[1] < len(degrees) {
			degrees[edge[1]]++
		}
		if len(lengths) < len(edges) && edge[0] >= 0 && edge[0] < len(points) && edge[1] >= 0 && edge[1] < len(points) {
			a := points[edge[0]]
			b := points[edge[1]]
			lengths = append(lengths, math.Sqrt((a.x-b.x)*(a.x-b.x)+(a.y-b.y)*(a.y-b.y)+(a.z-b.z)*(a.z-b.z)))
		}
	}

	features := map[string]float64{
		"mesh_points":       float64(len(points)),
		"mesh_edges":        float64(len(edges)),
		"mesh_fixed_points": float64(countFixed(points)),
	}
	addDegreeFeatures(features, degrees)
	addLengthFeatures(features, lengths)
	return features, nil
}

func countFixed(points []meshFeaturePoint) int {
	count := 0
	for _, point := range points {
		if point.fixed {
			count++
		}
	}
	return count
}

func addDegreeFeatures(features map[string]float64, degrees []int) {
	if len(degrees) == 0 {
		return
	}
	minDegree := degrees[0]
	maxDegree := degrees[0]
	sum := 0
	isolated := 0
	for _, degree := range degrees {
		if degree < minDegree {
			minDegree = degree
		}
		if degree > maxDegree {
			maxDegree = degree
		}
		if degree == 0 {
			isolated++
		}
		sum += degree
	}
	features["mesh_degree_min"] = float64(minDegree)
	features["mesh_degree_max"] = float64(maxDegree)
	features["mesh_degree_mean"] = float64(sum) / float64(len(degrees))
	features["mesh_isolated_points"] = float64(isolated)
}

func addLengthFeatures(features map[string]float64, lengths []float64) {
	if len(lengths) == 0 {
		return
	}
	minLength := lengths[0]
	maxLength := lengths[0]
	sum := 0.0
	for _, length := range lengths {
		if length < minLength {
			minLength = length
		}
		if length > maxLength {
			maxLength = length
		}
		sum += length
	}
	mean := sum / float64(len(lengths))
	var variance float64
	for _, length := range lengths {
		delta := length - mean
		variance += delta * delta
	}
	variance /= float64(len(lengths))
	features["mesh_spring_length_min"] = minLength
	features["mesh_spring_length_max"] = maxLength
	features["mesh_spring_length_mean"] = mean
	features["mesh_spring_length_stddev"] = math.Sqrt(variance)
}

func numericConfigValue(config map[string]interface{}, key string) (float64, bool) {
	value, ok := config[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
