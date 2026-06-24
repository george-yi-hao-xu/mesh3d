#include "SpringBuilder.h"
#include "SpringBuilderCuda.h"

#include "raylib.h"

#include <algorithm>
#include <cmath>
#include <iomanip>
#include <iostream>
#include <random>
#include <unordered_map>

namespace mesh3d {
    namespace {
        struct GridCell {
            int x = 0;
            int y = 0;
            int z = 0;

            bool operator==(const GridCell& other) const {
                return x == other.x && y == other.y && z == other.z;
            }
        };

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
    }

    std::vector<CandidateList> BuildLimitedSpringCandidatesSpatialGrid(
        const std::vector<Particle>& particles,
        float maxDist,
        int maxPerParticle,
        SpringBuildProfile* profile
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

    void BuildRandomSprings(
        std::vector<Particle>& particles,
        std::vector<Spring>& springs,
        float springStiffness,
        unsigned int seed,
        float maxDist,
        int maxPerParticle,
        float prob
    ) {
        if (particles.size() < 2) return;

        const size_t n = particles.size();
        std::vector<int> connectionCount(n, 0);
        SpringBuildProfile profile;

        // This returns already-bucketed per-particle candidate lists, capped to
        // nearby candidates only, so there is no large global candidate vector to split.
        std::vector<CandidateList> candidates;
#ifdef MESH3D_ENABLE_CUDA
        const bool usedCudaCandidates = BuildLimitedSpringCandidatesCuda(particles, maxDist, maxPerParticle, candidates, &profile);
        if (!usedCudaCandidates) {
            candidates = BuildLimitedSpringCandidatesSpatialGrid(particles, maxDist, maxPerParticle, &profile);
        }
#else
        candidates = BuildLimitedSpringCandidatesSpatialGrid(particles, maxDist, maxPerParticle, &profile);
#endif
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
}
