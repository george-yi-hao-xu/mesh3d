#pragma once

#include "SpringBuilder.h"

#include <vector>

namespace mesh3d {
    struct SpringCandidateGpu {
        int a = -1;
        int b = -1;
        float distance = 0.0f;
    };

    bool IsCudaSpringBuilderAvailable();
    bool RunCudaSpringBuilderProbe();

    bool BuildSpringCandidatesCudaProbe(
        const std::vector<Particle>& particles,
        std::vector<SpringCandidateGpu>& candidates
    );

    bool BuildLimitedSpringCandidatesCuda(
        const std::vector<Particle>& particles,
        float maxDist,
        int maxPerParticle,
        std::vector<CandidateList>& candidates,
        SpringBuildProfile* profile = nullptr
    );
}
