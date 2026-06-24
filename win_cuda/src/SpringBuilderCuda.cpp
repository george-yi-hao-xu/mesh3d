#include "SpringBuilderCuda.h"

#ifndef MESH3D_ENABLE_CUDA
namespace mesh3d {
    bool RunCudaSpringBuilderProbe() {
        return false;
    }

    bool BuildSpringCandidatesCudaProbe(
        const std::vector<Particle>&,
        std::vector<SpringCandidateGpu>& candidates
    ) {
        candidates.clear();
        return false;
    }

    bool BuildLimitedSpringCandidatesCuda(
        const std::vector<Particle>&,
        float,
        int,
        std::vector<CandidateList>& candidates,
        SpringBuildProfile*
    ) {
        candidates.clear();
        return false;
    }
}
#endif

namespace mesh3d {
    bool IsCudaSpringBuilderAvailable() {
#ifdef MESH3D_ENABLE_CUDA
        return true;
#else
        return false;
#endif
    }
}
