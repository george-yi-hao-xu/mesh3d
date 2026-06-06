package solver

import (
	"fmt"
	"math"

	"mesh3d/web_app/server/solver/mesh"
)

// NewMeshModelFromPointCloud adapts solver config to the mesh package constructor.
func NewMeshModelFromPointCloud(path string, cfg SolverConfig) (*mesh.MeshModel, error) {
	return mesh.NewMeshModelFromPointCloud(path, meshConfig(cfg))
}

// RunMesh advances a mesh until convergence or configured limits and emits checkpoints.
func RunMesh(model *mesh.MeshModel, cfg SolverConfig, onSnapshot func(simTime float64, step int) error) (SolverResult, error) {
	if cfg.TimeStep <= 0 {
		return SolverResult{}, fmt.Errorf("timeStep must be positive")
	}
	if cfg.SnapshotInterval <= 0 {
		return SolverResult{}, fmt.Errorf("snapshotInterval must be positive")
	}
	if cfg.MaxSteps <= 0 {
		return SolverResult{}, fmt.Errorf("maxSteps must be positive")
	}
	if cfg.MaxSimTime <= 0 {
		return SolverResult{}, fmt.Errorf("maxSimTime must be positive")
	}

	convergence := NewConvergenceChecker(model.Particles, cfg)
	nextSnapshot := cfg.SnapshotInterval
	simTime := 0.0

	for step := 1; step <= cfg.MaxSteps && simTime < cfg.MaxSimTime; step++ {
		if ok := model.Update(cfg.TimeStep); !ok {
			return SolverResult{}, fmt.Errorf("solver produced invalid particle positions")
		}
		simTime += cfg.TimeStep

		for simTime+1e-9 >= nextSnapshot {
			if err := onSnapshot(nextSnapshot, step); err != nil {
				return SolverResult{}, err
			}
			nextSnapshot += cfg.SnapshotInterval
		}

		if convergence.Update(model.Particles) {
			return SolverResult{
				Converged: true,
				SimTime:   simTime,
				Step:      step,
				Reason:    "velocity and position movement below thresholds",
			}, nil
		}
	}

	return SolverResult{
		Converged: false,
		SimTime:   simTime,
		Step:      int(math.Round(simTime / cfg.TimeStep)),
		Reason:    "max simulation time or max steps reached",
	}, nil
}

// meshConfig copies the mesh-owned subset of solver settings.
func meshConfig(cfg SolverConfig) mesh.Config {
	return mesh.Config{
		Stiffness:             cfg.Stiffness,
		ParticleMass:          cfg.ParticleMass,
		DampingFactor:         cfg.DampingFactor,
		AirResistanceFactor:   cfg.AirResistanceFactor,
		Gravity:               cfg.Gravity,
		SpringSeed:            cfg.SpringSeed,
		MaxSpringDist:         cfg.MaxSpringDist,
		MaxSpringsPerParticle: cfg.MaxSpringsPerParticle,
		SpringConnectProb:     cfg.SpringConnectProb,
	}
}
