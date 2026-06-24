#pragma once

#include "SpringBuilderCuda.h"

#include <vector>

namespace mesh3d {
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
        SpringBuildProfile* profile
    );
}
