#include "Spring.h"
#include <cmath>

Vector3 AddVector3(const Vector3& a, const Vector3& b) {
	return { a.x + b.x, a.y + b.y, a.z + b.z };
}

Vector3 SubstractVector3(const Vector3& a, const Vector3& b) {
	return { a.x - b.x, a.y - b.y, a.z - b.z };
}

Vector3 MultiplyVector3(float l, const Vector3& v) {
	return { l * v.x, l * v.y, l * v.z };
}

float LengthVector3(const Vector3& v) {
	return std::sqrt(v.x * v.x + v.y * v.y + v.z * v.z);
}

Spring::Spring(Particle* a, Particle* b, float k): 
	pA(a), 
	pB(b), 
	stiffness(k), 
	restLength(LengthVector3(*b - *a))
{};

void Spring::ApplySpringForce(float dampingFactor) {
	Vector3 diff = SubstractVector3(pB->position, pA->position);
	float currentLength = LengthVector3(diff);

	// Avoid division by zero
	if (currentLength < 0.0001f) return;

	// Unit direction: from pA to pB
	Vector3 dir = { diff.x / currentLength, diff.y / currentLength, diff.z / currentLength };

	// Hooke's Law: F = -k * (currentLength - restLength) * dir
	float displacement = currentLength - restLength;
	Vector3 springForce = MultiplyVector3(-stiffness * displacement, dir);

	// Damping only along the spring direction (prevents oscillation without adding artificial drag)
	Vector3 velocityDiff = SubstractVector3(pB->velocity, pA->velocity);
	float velocityAlongSpring = velocityDiff.x * dir.x + velocityDiff.y * dir.y + velocityDiff.z * dir.z;
	Vector3 dampingForce = MultiplyVector3(-dampingFactor * velocityAlongSpring, dir);

	Vector3 totalForce = AddVector3(springForce, dampingForce);

	pA->ApplyForce(MultiplyVector3(-1, totalForce));
	pB->ApplyForce(totalForce);
}
