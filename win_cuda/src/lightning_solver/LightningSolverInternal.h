#pragma once

#include "Particle.h"
#include "raylib.h"

#include <utility>
#include <vector>

namespace mesh3d::lightning {
    struct SolverParams {
        float stiffness = 20.0f;
        float dampingFactor = 0.1f;
        float airResistanceFactor = 0.001f;
        float gravity = -0.2f;
        int steps = 50000;
        float dt = 1.0f / 120.0f;
        int maxNeighborsPerParticle = 8;
        float forceConvergenceThreshold = 0.001f;
    };

    struct DirectedNeighbor {
        int other = -1;
        float restLength = 0.0f;
    };

    bool RunLightningCuda(
        std::vector<Particle>& particles,
        const std::vector<DirectedNeighbor>& neighbors,
        const std::vector<int>& neighborCounts,
        const SolverParams& params,
        int& stepsRun
    );
}
