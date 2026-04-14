#pragma once
#include "raylib.h"

class Particle {
public:
    Vector3 position;
    Vector3 velocity = { 0.0f, 0.0f, 0.0f }; // same as below
	Vector3 force = { 0.0f, 0.0f, 0.0f }; // important to have defalut 0 0 0 values here, if not, weird things happen
	float mass = 1.0f; // default mass
    bool isFixed = false;  // Whether this particle is locked in place

    Particle(Vector3 pos, bool fixed = false, float mass = 1.0f);
    void ApplyForce(Vector3 f);
    void Update(float dt);

    // Overload operator- inside the class
    Vector3 operator-(const Particle& other) const {
        return { position.x - other.position.x,
                 position.y - other.position.y,
                 position.z - other.position.z };
    }
};
