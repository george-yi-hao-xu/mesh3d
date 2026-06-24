#include "Mesh.h"
#include "ParticleRenderer.h"

#include "raylib.h"
#include "rlgl.h"
#include <iostream>
#include <future>
#include <vector>
#include <fstream>
#include <sstream>
#include <string>
#include <cmath>
#include <random>
#include <algorithm>
#include <iomanip>
#include <unordered_map>

const float HEIGHT = 0.0f;

namespace mesh3d {
    namespace {
        // 记录粒子a b的数组下标
        struct SpringCandidate {
            size_t a = 0;
            size_t b = 0;
            float distance = 0.0f;
        };

        struct SpringBuildProfile {
            double gridBuildMs = 0.0;
            double candidateSearchMs = 0.0;
            double candidateBucketMs = 0.0;
            double shuffleMs = 0.0;
            double connectMs = 0.0;
            size_t candidateCount = 0;
            size_t springCount = 0;
        };

        struct GridCell {
            int x = 0;
            int y = 0;
            int z = 0;

            bool operator==(const GridCell& other) const {
                return x == other.x && y == other.y && z == other.z;
            }
        };

        // define hash
        struct GridCellHash {
            size_t operator()(const GridCell& cell) const {
                size_t h = 1469598103934665603ull;
                auto mix = [&h](int value) {
                    h ^= static_cast<size_t>(value);
                    h *= 1099511628211ull;
                };
                mix(cell.x);
                mix(cell.y);
                mix(cell.z);
                return h;
            }
        };

        GridCell PositionToCell(const Vector3& position, float cellSize) {
            return {
                static_cast<int>(std::floor(position.x / cellSize)),
                static_cast<int>(std::floor(position.y / cellSize)),
                static_cast<int>(std::floor(position.z / cellSize))
            };
        }

        std::vector<SpringCandidate> BuildSpringCandidatesSpatialGrid(
            const std::vector<Particle>& particles,
            float maxDist,
            SpringBuildProfile* profile = nullptr
        ) {
            std::vector<SpringCandidate> candidates;
            if (particles.size() < 2 || maxDist <= 0.0f) {
                return candidates;
            }

            const float maxDistSq = maxDist * maxDist;
            const float cellSize = maxDist * 0.5f;
            const int neighborRange = static_cast<int>(std::ceil(maxDist / cellSize));

            // record
            std::unordered_map<GridCell, std::vector<size_t>, GridCellHash> grid;
            grid.reserve(particles.size());

            const double gridStart = GetTime();
            for (size_t i = 0; i < particles.size(); ++i) {
                grid[PositionToCell(particles[i].position, cellSize)].push_back(i);
            }
            if (profile != nullptr) {
                profile->gridBuildMs = (GetTime() - gridStart) * 1000.0;
            }

            const double searchStart = GetTime();
            for (size_t i = 0; i < particles.size(); ++i) {
                const GridCell baseCell = PositionToCell(particles[i].position, cellSize);

                for (int dz = -neighborRange; dz <= neighborRange; ++dz) {
                    for (int dy = -neighborRange; dy <= neighborRange; ++dy) {
                        for (int dx = -neighborRange; dx <= neighborRange; ++dx) {
                            const GridCell neighborCell = { baseCell.x + dx, baseCell.y + dy, baseCell.z + dz };
                            auto found = grid.find(neighborCell);
                            if (found == grid.end()) {
                                continue;
                            }

                            for (size_t j : found->second) {
                                if (j <= i) {
                                    continue;
                                }

                                const float diffX = particles[i].position.x - particles[j].position.x;
                                const float diffY = particles[i].position.y - particles[j].position.y;
                                const float diffZ = particles[i].position.z - particles[j].position.z;
                                const float distSq = diffX * diffX + diffY * diffY + diffZ * diffZ;

                                if (distSq <= maxDistSq && distSq > 0.00000001f) {
                                    candidates.push_back({ i, j, std::sqrt(distSq) });
                                }
                            }
                        }
                    }
                }
            }
            if (profile != nullptr) {
                profile->candidateSearchMs = (GetTime() - searchStart) * 1000.0;
                profile->candidateCount = candidates.size();
            }

            return candidates;
        }

        using Candidate = std::pair<size_t, float>;
        using CandidateList = std::vector<Candidate>;

        std::vector<CandidateList> BuildLimitedSpringCandidatesSpatialGrid(
            const std::vector<Particle>& particles,
            float maxDist,
            int maxPerParticle,
            SpringBuildProfile* profile = nullptr
        ) {
            std::vector<CandidateList> candidates(particles.size());
            if (particles.size() < 2 || maxDist <= 0.0f || maxPerParticle <= 0) {
                return candidates;
            }

            const float maxDistSq = maxDist * maxDist;
            const float cellSize = maxDist * 0.5f;
            const int neighborRange = static_cast<int>(std::ceil(maxDist / cellSize));
            // Keep more candidates than the final spring limit so shuffle/probability
            // still has room to vary topology, but avoid storing every nearby pair.
            const size_t perParticleLimit = static_cast<size_t>(std::max(1, maxPerParticle * 8));
            std::unordered_map<GridCell, std::vector<size_t>, GridCellHash> grid;
            grid.reserve(particles.size());

            const double gridStart = GetTime();
            for (size_t i = 0; i < particles.size(); ++i) {
                grid[PositionToCell(particles[i].position, cellSize)].push_back(i);
            }
            if (profile != nullptr) {
                profile->gridBuildMs = (GetTime() - gridStart) * 1000.0;
            }

            const double searchStart = GetTime();
            for (size_t i = 0; i < particles.size(); ++i) {
                const GridCell baseCell = PositionToCell(particles[i].position, cellSize);
                CandidateList localCandidates;

                for (int dz = -neighborRange; dz <= neighborRange; ++dz) {
                    for (int dy = -neighborRange; dy <= neighborRange; ++dy) {
                        for (int dx = -neighborRange; dx <= neighborRange; ++dx) {
                            const GridCell neighborCell = { baseCell.x + dx, baseCell.y + dy, baseCell.z + dz };
                            auto found = grid.find(neighborCell);
                            if (found == grid.end()) {
                                continue;
                            }

                            for (size_t j : found->second) {
                                if (j <= i) {
                                    continue;
                                }

                                const float diffX = particles[i].position.x - particles[j].position.x;
                                const float diffY = particles[i].position.y - particles[j].position.y;
                                const float diffZ = particles[i].position.z - particles[j].position.z;
                                const float distSq = diffX * diffX + diffY * diffY + diffZ * diffZ;

                                if (distSq <= maxDistSq && distSq > 0.00000001f) {
                                    localCandidates.push_back({ j, std::sqrt(distSq) });
                                }
                            }
                        }
                    }
                }

                if (localCandidates.size() > perParticleLimit) {
                    // nth_element partitions in linear time on average: after it runs,
                    // the first perParticleLimit entries are the nearest candidates,
                    // but they are not fully sorted. Full sorting is unnecessary here.
                    std::nth_element(
                        localCandidates.begin(),
                        localCandidates.begin() + static_cast<std::ptrdiff_t>(perParticleLimit),
                        localCandidates.end(),
                        [](const Candidate& a, const Candidate& b) {
                            return a.second < b.second;
                        }
                    );
                    localCandidates.resize(perParticleLimit);
                }

                if (profile != nullptr) {
                    profile->candidateCount += localCandidates.size();
                }
                candidates[i] = std::move(localCandidates);
            }
            if (profile != nullptr) {
                profile->candidateSearchMs = (GetTime() - searchStart) * 1000.0;
            }

            return candidates;
        }
    }

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
        if (particles.size() < 2) return;

        const size_t n = particles.size();
        std::vector<int> connectionCount(n, 0);
        SpringBuildProfile profile;

        // This returns already-bucketed per-particle candidate lists, capped to
        // nearby candidates only, so there is no large global candidate vector to split.
        std::vector<CandidateList> candidates = BuildLimitedSpringCandidatesSpatialGrid(particles, maxDist, maxPerParticle, &profile);
        profile.candidateBucketMs = 0.0;

        // Shuffle per-particle candidates deterministically so lower indices do not
        // always consume maxPerParticle connection slots first.
        const double shuffleStart = GetTime();
        for (size_t i = 0; i < n; ++i) {
            std::mt19937 rng(static_cast<unsigned int>(seed + i));
            std::shuffle(candidates[i].begin(), candidates[i].end(), rng);
        }
        profile.shuffleMs = (GetTime() - shuffleStart) * 1000.0;

        std::mt19937 probRng(seed);
        std::uniform_real_distribution<float> dist01(0.0f, 1.0f);

        // Convert candidates into springs while respecting probability and connection limits.
        const double connectStart = GetTime();
        for (size_t i = 0; i < n; ++i) {
            for (const auto& cand : candidates[i]) {
                size_t j = cand.first;

                if (connectionCount[i] >= maxPerParticle || connectionCount[j] >= maxPerParticle) {
                    continue;
                }

                if (dist01(probRng) <= prob) {
                    springs.emplace_back(&particles[i], &particles[j], springStiffness);
                    connectionCount[i]++;
                    connectionCount[j]++;
                }
            }
        }
        profile.connectMs = (GetTime() - connectStart) * 1000.0;
        profile.springCount = springs.size();

        std::cout << std::fixed << std::setprecision(2)
            << "Spring build profile: "
            << "grid=" << profile.gridBuildMs << "ms, "
            << "search=" << profile.candidateSearchMs << "ms, "
            << "bucket=" << profile.candidateBucketMs << "ms, "
            << "shuffle=" << profile.shuffleMs << "ms, "
            << "connect=" << profile.connectMs << "ms, "
            << "candidates=" << profile.candidateCount << ", "
            << "springs=" << profile.springCount
            << std::endl;
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

    SpringStats Mesh::ComputeSpringStats() const {
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
