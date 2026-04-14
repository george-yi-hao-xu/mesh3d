#include "Mesh.h"
#include "raylib.h"
#include <iostream>
#include <future>
#include <vector>
#include <fstream>
#include <sstream>
#include <string>
#include <cmath>

const float HEIGHT = 0.0f;

namespace mesh3d {
	Config LoadMeshConfig(const std::string& filename) {
		Config config;
		std::ifstream file(filename);
        std::string line;
		while (std::getline(file, line)) {
			std::istringstream iss(line);
            std::string key;
			if (std::getline(iss, key, '=')) {
				std::string value;
				if (std::getline(iss, value)) {
					if (key == "width") config.width = std::stoi(value);
					else if (key == "height") config.height = std::stoi(value);
					else if (key == "spacing") config.spacing = std::stof(value);
					else if (key == "stiffness") config.stiffness = std::stof(value);
					else if (key == "particleMass") config.particleMass = std::stof(value);
					else if (key == "dampingFactor") config.dampingFactor = std::stof(value);
					else if (key == "airResistanceFactor") config.airResistanceFactor = std::stof(value);
				}
			}
		}
		return config;
	}

    void WriteConfig(const std::string& filename, const Config& config) {
        std::ofstream file(filename);
		file << "width=" << config.width << "\n";
        file << "height=" << config.height << "\n";
        file << "spacing=" << config.spacing << "\n";
        file << "stiffness=" << config.stiffness << "\n";
        file << "particleMass=" << config.particleMass << "\n";
		file << "dampingFactor=" << config.dampingFactor << "\n";
		file << "airResistanceFactor=" << config.airResistanceFactor << "\n";
    }

    Mesh::Mesh(const Config& c): width(c.width), height(c.height), springStiffness(c.stiffness), dampingFactor(c.dampingFactor), airResistanceFactor(c.airResistanceFactor) {
        const Vector3 ORIGIN = { (c.width - 1) * c.spacing / 2, HEIGHT, (c.height - 1) * c.spacing / 2 };

		particles.reserve(c.width * c.height);
		springs.reserve((c.width - 1) * c.height + (c.height - 1) * c.width);

        // Create particles
        for (int y = 0; y < c.height; y++) {
            for (int x = 0; x < c.width; x++) {
                bool fixed = (y == 0 || x == 0 || y == c.height - 1 || x == c.width - 1);  // Fix top row
                particles.emplace_back(Vector3{ x * c.spacing - ORIGIN.x, 0 - ORIGIN.y, y * c.spacing - ORIGIN.z }, fixed, c.particleMass);
            }
        }

        // Create springs
        for (int y = 0; y < c.height; y++) {
            for (int x = 0; x < c.width; x++) {
                int idx = y * c.width + x;
                if (x < c.width - 1) springs.emplace_back(&particles[idx], &particles[idx + 1], c.stiffness); // Horizontal
                if (y < c.height - 1) springs.emplace_back(&particles[idx], &particles[idx + c.width], c.stiffness); // Vertical
            }
        }
    }

    bool Mesh::Update(float dt) {
        if (dt <= 0.0f) return true;

		for (auto& particle : particles) {
			particle.ApplyForce(Vector3{ 0, -9.8f, 0 });  // Gravity
			// air resistance
			particle.ApplyForce(Vector3{
				airResistanceFactor * particle.velocity.x * particle.velocity.x,
				airResistanceFactor * particle.velocity.y * particle.velocity.y,
				airResistanceFactor * particle.velocity.z * particle.velocity.z
				});
			particle.Update(dt);
		}
        

        for (auto& spring : springs) {
			spring.stiffness = springStiffness; // listen on the stiffness changes
            spring.ApplySpringForce(dampingFactor); // hook law
        }

        // debug
        for (auto& particle : particles) {
            if (std::isnan(particle.position.x) || std::isnan(particle.position.y) || std::isnan(particle.position.z)) {
                std::cerr << "Particle position invalid: " << particle.position.x << ", " << particle.position.y << ", " << particle.position.z << std::endl;
				return false;
            }
        }

		return true;
    }

	void Mesh::SetStiffness(float stiff) {
		springStiffness = stiff;
	}

	void Mesh::SetDampingFactor(float dFactor) {
		dampingFactor = dFactor;
	}

	void Mesh::SetAirResistanceFactor(float arFactor) {
		airResistanceFactor = arFactor;
	}

    void Mesh::Draw() {

        for (auto& spring : springs) {
            DrawLine3D(spring.pA->position, spring.pB->position, BLUE);
        }

		for (auto& particle : particles) {
			DrawSphere(particle.position, 0.1f, RED);
		}
    }
}
