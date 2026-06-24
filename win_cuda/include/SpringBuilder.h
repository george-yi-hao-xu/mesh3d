#pragma once

#include "Particle.h"
#include "Spring.h"

#include <cstddef>
#include <utility>
#include <vector>

namespace mesh3d {
    using Candidate = std::pair<size_t, float>;
    using CandidateList = std::vector<Candidate>;

    struct SpringBuildProfile {
        double gridBuildMs = 0.0;
        double candidateSearchMs = 0.0;
        double candidateBucketMs = 0.0;
        double shuffleMs = 0.0;
        double connectMs = 0.0;
        size_t candidateCount = 0;
        size_t springCount = 0;
    };

    std::vector<CandidateList> BuildLimitedSpringCandidatesSpatialGrid(
        const std::vector<Particle>& particles,
        float maxDist,
        int maxPerParticle,
        SpringBuildProfile* profile = nullptr
    );

    void BuildRandomSprings(
        std::vector<Particle>& particles,
        std::vector<Spring>& springs,
        float springStiffness,
        unsigned int seed,
        float maxDist,
        int maxPerParticle,
        float prob
    );
}
