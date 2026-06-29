#pragma once

#include "Particle.h"
#include "raylib.h"

#include <utility>
#include <vector>

namespace mesh3d::lightning {
    struct Config {
        float spacing = 0.1f;
        float particleMass = 1.0f;
        int maxNeighborsPerParticle = 8;

        float stiffness = 20.0f;
        float dampingFactor = 0.1f;
        float airResistanceFactor = 0.001f;
        float gravity = -0.2f;

        int steps = 50000;
        float dt = 1.0f / 120.0f;
    };

    struct Result {
        std::vector<Particle> particles;
        std::vector<std::pair<int, int>> bars;
        int stepsRun = 0;
        bool ok = false;
    };

    bool SolveBoundaryCuda(
        const std::vector<Vector3>& boundaryPoints,
        const Config& config,
        Result& result
    );
}
