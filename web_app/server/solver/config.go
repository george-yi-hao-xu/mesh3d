package solver

import (
	"strconv"
	"strings"
)

type SolverConfig struct {
	Stiffness             float64
	ParticleMass          float64
	DampingFactor         float64
	AirResistanceFactor   float64
	Gravity               float64
	SpringSeed            int64
	MaxSpringDist         float64
	MaxSpringsPerParticle int
	SpringConnectProb     float64
	TimeStep              float64
	SnapshotInterval      float64
	MaxSimTime            float64
	MaxSteps              int
	VelocityEpsilon       float64
	PositionEpsilon       float64
	StableFrames          int
}

type SolverResult struct {
	Converged bool
	SimTime   float64
	Step      int
	Reason    string
}

// LoadSolverConfig maps loose JSON job config into typed solver settings.
func LoadSolverConfig(raw map[string]interface{}) SolverConfig {
	return SolverConfig{
		Stiffness:             configFloat(raw, "stiffness", 10.0),
		ParticleMass:          configFloat(raw, "particleMass", 1.0),
		DampingFactor:         configFloat(raw, "dampingFactor", configFloat(raw, "damping", 0.1)),
		AirResistanceFactor:   configFloat(raw, "airResistanceFactor", 0.001),
		Gravity:               configFloat(raw, "gravity", 9.8),
		SpringSeed:            int64(configInt(raw, "springSeed", 42)),
		MaxSpringDist:         configFloat(raw, "maxSpringDist", 1.5),
		MaxSpringsPerParticle: configInt(raw, "maxSpringsPerParticle", 4),
		SpringConnectProb:     configFloat(raw, "springConnectProb", 0.8),
		TimeStep:              configFloat(raw, "timeStep", 1.0/60.0),
		SnapshotInterval:      configFloat(raw, "snapshotInterval", 0.05),
		MaxSimTime:            configFloat(raw, "maxSimTime", 120.0),
		MaxSteps:              configInt(raw, "maxSteps", 200000),
		VelocityEpsilon:       configFloat(raw, "velocityEpsilon", 0.001),
		PositionEpsilon:       configFloat(raw, "positionEpsilon", 0.001),
		StableFrames:          configInt(raw, "stableFrames", 60),
	}
}

// configFloat reads a float config value from decoded JSON.
func configFloat(raw map[string]interface{}, key string, fallback float64) float64 {
	value, ok := raw[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

// configInt reads an integer config value from decoded JSON.
func configInt(raw map[string]interface{}, key string, fallback int) int {
	value, ok := raw[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}
