package solver

import (
	"math"

	"mesh3d/web_app/server/solver/mesh"
)

type ConvergenceChecker struct {
	previous         []mesh.Vec3
	velocityEps      float64
	positionEps      float64
	requiredStable   int
	stableFrameCount int
}

// NewConvergenceChecker captures initial positions for stability checks.
func NewConvergenceChecker(particles []mesh.ParticleNode, cfg SolverConfig) *ConvergenceChecker {
	previous := make([]mesh.Vec3, len(particles))
	for i := range particles {
		previous[i] = particles[i].Position
	}

	requiredStable := cfg.StableFrames
	if requiredStable <= 0 {
		requiredStable = 1
	}

	return &ConvergenceChecker{
		previous:       previous,
		velocityEps:    cfg.VelocityEpsilon,
		positionEps:    cfg.PositionEpsilon,
		requiredStable: requiredStable,
	}
}

// Update records movement metrics and reports sustained convergence.
func (c *ConvergenceChecker) Update(particles []mesh.ParticleNode) bool {
	maxVelocity := 0.0
	maxMove := 0.0

	for i := range particles {
		if particles[i].Fixed {
			c.previous[i] = particles[i].Position
			continue
		}
		maxVelocity = math.Max(maxVelocity, particles[i].Velocity.Length())
		maxMove = math.Max(maxMove, particles[i].Position.Sub(c.previous[i]).Length())
		c.previous[i] = particles[i].Position
	}

	if maxVelocity < c.velocityEps && maxMove < c.positionEps {
		c.stableFrameCount++
	} else {
		c.stableFrameCount = 0
	}
	return c.stableFrameCount >= c.requiredStable
}
