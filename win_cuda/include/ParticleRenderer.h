#pragma once

#include "Particle.h"

#include <vector>

namespace mesh3d {
    void DrawParticlesInstancedOrFallback(const std::vector<Particle>& particles);
}
