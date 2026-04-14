#pragma once
#include "Particle.h"

class Spring {
public:
    Particle* pA, * pB;
    Vector3 initialDiff = { 0.0f, 0.0f, 0.0f };
    float stiffness; // the larger, the stronger

    Spring(Particle* a, Particle* b, float k);
    void ApplySpringForce(float dampingFactor);
};
