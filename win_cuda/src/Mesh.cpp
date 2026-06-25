#include "Mesh.h"
#include "ParticleRenderer.h"
#include "SpringBuilder.h"

#include "raylib.h"
#include "rlgl.h"
#include <iostream>
#include <future>
#include <vector>
#include <fstream>
#include <sstream>
#include <string>
#include <cmath>
#include <algorithm>
#include <iomanip>

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
                    value.erase(value.find_last_not_of(" \t\r\n") + 1);
                    if (key == "width") config.width = std::stoi(value);
                    else if (key == "height") config.height = std::stoi(value);
                    else if (key == "spacing") config.spacing = std::stof(value);
                    else if (key == "stiffness") config.stiffness = std::stof(value);
                    else if (key == "particleMass") config.particleMass = std::stof(value);
                    else if (key == "dampingFactor") config.dampingFactor = std::stof(value);
                    else if (key == "airResistanceFactor") config.airResistanceFactor = std::stof(value);
                    else if (key == "gravity") config.gravity = std::stof(value);
                    else if (key == "pointCloudFile") config.pointCloudFile = value;
                    else if (key == "springSeed") config.springSeed = static_cast<unsigned int>(std::stoul(value));
                    else if (key == "maxSpringDist") config.maxSpringDist = std::stof(value);
                    else if (key == "maxSpringsPerParticle") config.maxSpringsPerParticle = std::stoi(value);
                    else if (key == "springConnectProb") config.springConnectProb = std::stof(value);
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
        file << "gravity=" << config.gravity << "\n";
        file << "pointCloudFile=" << config.pointCloudFile << "\n";
        file << "springSeed=" << config.springSeed << "\n";
        file << "maxSpringDist=" << config.maxSpringDist << "\n";
        file << "maxSpringsPerParticle=" << config.maxSpringsPerParticle << "\n";
        file << "springConnectProb=" << config.springConnectProb << "\n";
    }

    void Mesh::BuildRegularGrid(const Config& c) {
        const Vector3 ORIGIN = { (c.width - 1) * c.spacing / 2, HEIGHT, (c.height - 1) * c.spacing / 2 };

        particles.reserve(c.width * c.height);
        springs.reserve((c.width - 1) * c.height + (c.height - 1) * c.width);

        for (int y = 0; y < c.height; y++) {
            for (int x = 0; x < c.width; x++) {
                bool fixed = (y == 0 || x == 0 || y == c.height - 1 || x == c.width - 1);
                particles.emplace_back(Vector3{ x * c.spacing - ORIGIN.x, 0 - ORIGIN.y, y * c.spacing - ORIGIN.z }, fixed, c.particleMass);
            }
        }

        for (int y = 0; y < c.height; y++) {
            for (int x = 0; x < c.width; x++) {
                int idx = y * c.width + x;
                if (x < c.width - 1) springs.emplace_back(&particles[idx], &particles[idx + 1], c.stiffness);
                if (y < c.height - 1) springs.emplace_back(&particles[idx], &particles[idx + c.width], c.stiffness);
            }
        }
    }

    std::vector<Particle> Mesh::LoadPointCloud(const std::string& path) {
        std::vector<Particle> loaded;
        std::ifstream file(path);
        if (!file.is_open()) {
            std::cerr << "Failed to open point cloud file: " << path << std::endl;
            return loaded;
        }

        std::string line;
        while (std::getline(file, line)) {
            line.erase(line.find_last_not_of(" \t\r\n") + 1);
            if (line.empty() || line[0] == '#') continue;

            std::istringstream iss(line);
            float x, y, z, mass;
            int fixed;
            if (iss >> x >> y >> z >> fixed >> mass) {
                loaded.emplace_back(Vector3{ x, y, z }, fixed != 0, mass);
            } else {
                std::cerr << "Invalid point cloud line: " << line << std::endl;
            }
        }
        return loaded;
    }

    void Mesh::GenerateRandomSprings(unsigned int seed, float maxDist, int maxPerParticle, float prob) {
        BuildRandomSprings(particles, springs, springStiffness, seed, maxDist, maxPerParticle, prob);
    }

    void Mesh::BuildFromPointCloud(const Config& c, const char* ptFileName) {
        if (ptFileName == nullptr) {
            BuildRegularGrid(c);
            return;
        }

        // gen points first
        particles = LoadPointCloud(ptFileName);
        if (particles.empty()) {
            std::cerr << "Point cloud file empty or failed to load. Falling back to regular grid." << std::endl;
            BuildRegularGrid(c);
            return;
        }
        // then gen springs
        GenerateRandomSprings(c.springSeed, c.maxSpringDist, c.maxSpringsPerParticle, c.springConnectProb);
    }

    Mesh::Mesh(const Config& c, const char* ptFileName) {
        springStiffness = c.stiffness;
        dampingFactor = c.dampingFactor;
        airResistanceFactor = c.airResistanceFactor;
        gravity = c.gravity;

        if (ptFileName != nullptr && ptFileName[0] != '\0') {
            BuildFromPointCloud(c, ptFileName);
        } else {
            BuildRegularGrid(c);
        }
    }

    bool Mesh::Update(float dt) {
        if (dt <= 0.0f) return true;

        for (auto& particle : particles) {
            particle.ApplyForce(Vector3{ 0, -gravity, 0 });
            particle.ApplyForce(Vector3{
                -airResistanceFactor * particle.velocity.x * std::abs(particle.velocity.x),
                -airResistanceFactor * particle.velocity.y * std::abs(particle.velocity.y),
                -airResistanceFactor * particle.velocity.z * std::abs(particle.velocity.z)
                });
        }

        for (auto& spring : springs) {
            spring.stiffness = springStiffness;
            spring.ApplySpringForce(dampingFactor);
        }

        for (auto& particle : particles) {
            particle.Update(dt);
        }

        for (auto& particle : particles) {
            if (std::isnan(particle.position.x) || std::isnan(particle.position.y) || std::isnan(particle.position.z)) {
                std::cerr << "Particle position invalid: " << particle.position.x << ", " << particle.position.y << ", " << particle.position.z << std::endl;
                return false;
            }
        }

        return true;
    }

    mesh3d::SpringStats Mesh::ComputeSpringStats() const {
        SpringStats stats;
        if (springs.empty()) {
            return stats;
        }

        double lengthSum = 0.0;
        double stretchSum = 0.0;

        for (const auto& spring : springs) {
            const Vector3& a = spring.pA->position;
            const Vector3& b = spring.pB->position;
            const double dx = static_cast<double>(b.x) - a.x;
            const double dy = static_cast<double>(b.y) - a.y;
            const double dz = static_cast<double>(b.z) - a.z;
            const double length = std::sqrt(dx * dx + dy * dy + dz * dz);
            const double stretch = length - spring.restLength;

            lengthSum += length;
            stretchSum += stretch;
        }

        const double count = static_cast<double>(springs.size());
        const double lengthMean = lengthSum / count;
        const double stretchMean = stretchSum / count;

        double lengthVarianceSum = 0.0;
        double stretchVarianceSum = 0.0;

        for (const auto& spring : springs) {
            const Vector3& a = spring.pA->position;
            const Vector3& b = spring.pB->position;
            const double dx = static_cast<double>(b.x) - a.x;
            const double dy = static_cast<double>(b.y) - a.y;
            const double dz = static_cast<double>(b.z) - a.z;
            const double length = std::sqrt(dx * dx + dy * dy + dz * dz);
            const double stretch = length - spring.restLength;
            const double lengthDelta = length - lengthMean;
            const double stretchDelta = stretch - stretchMean;

            lengthVarianceSum += lengthDelta * lengthDelta;
            stretchVarianceSum += stretchDelta * stretchDelta;
        }

        stats.lengthMean = static_cast<float>(lengthMean);
        stats.lengthVariance = static_cast<float>(lengthVarianceSum / count);
        stats.stretchMean = static_cast<float>(stretchMean);
        stats.stretchVariance = static_cast<float>(stretchVarianceSum / count);
        return stats;
    }

    mesh3d::PtStats Mesh::ComputePtStats() const {
        PtStats stats;
        if (particles.empty()) {
            return stats;
        }

        double forceMagSum = 0.0;

        for (const auto& particle : particles) {
            const Vector3& force = particle.lastFrameNetForce;
            const double forceMag = std::sqrt(
                static_cast<double>(force.x) * force.x +
                static_cast<double>(force.y) * force.y +
                static_cast<double>(force.z) * force.z
            );

            forceMagSum += forceMag;
        }

        const double count = static_cast<double>(particles.size());
        const double forceMagMean = forceMagSum / count;

        double forceMagVarianceSum = 0.0;

        for (const auto& particle : particles) {
            const Vector3& force = particle.lastFrameNetForce;
            const double forceMag = std::sqrt(
                static_cast<double>(force.x) * force.x +
                static_cast<double>(force.y) * force.y +
                static_cast<double>(force.z) * force.z
            );
            const double forceMagDelta = forceMag - forceMagMean;

            forceMagVarianceSum += forceMagDelta * forceMagDelta;
        }

        stats.forceValMean = static_cast<float>(forceMagMean);
        stats.forceValVar = static_cast<float>(forceMagVarianceSum / count);
        return stats;
    }

    void Mesh::Draw() {
        rlBegin(RL_LINES);
        rlColor4ub(BLUE.r, BLUE.g, BLUE.b, BLUE.a);
        for (auto& spring : springs) {
            const Vector3& a = spring.pA->position;
            const Vector3& b = spring.pB->position;
            rlVertex3f(a.x, a.y, a.z);
            rlVertex3f(b.x, b.y, b.z);
        }
        rlEnd();

        DrawParticlesInstancedOrFallback(particles);
    }

    void Mesh::WritePointCloud(std::ostream& out) const {
        const std::ios::fmtflags oldFlags = out.flags();
        const std::streamsize oldPrecision = out.precision();

        out << "# Exported point cloud from current mesh state\n";
        out << "# Format: x y z fixed mass\n";
        out << std::fixed << std::setprecision(6);

        for (const auto& particle : particles) {
            out << particle.position.x << ' '
                << particle.position.y << ' '
                << particle.position.z << ' '
                << (particle.isFixed ? 1 : 0) << ' '
                << particle.mass << '\n';
        }

        out.flags(oldFlags);
        out.precision(oldPrecision);
    }
}
