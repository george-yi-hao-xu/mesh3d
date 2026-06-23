#pragma once
#include "Particle.h"

class Spring {
public:
    Particle* pA, * pB;
    float restLength = 0.0f;
    float stiffness; // the larger, the stronger

    Spring(Particle* a, Particle* b, float k);
    void ApplySpringForce(float dampingFactor);
};
