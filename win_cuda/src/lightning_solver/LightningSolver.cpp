#include "lightning_solver/LightningSolver.h"
#include "lightning_solver/LightningSolverInternal.h"

#include <algorithm>
#include <cmath>
#include <iostream>

namespace mesh3d::lightning {
    namespace {
        struct GridPoint {
            int index = -1;
            bool inside = false;
        };

        float DistanceXZ(const Vector3& a, const Vector3& b) {
            const float dx = a.x - b.x;
            const float dz = a.z - b.z;
            return std::sqrt(dx * dx + dz * dz);
        }

        bool IsInsidePolygonXZ(const std::vector<Vector3>& polygon, float x, float z) {
            bool inside = false;
            const size_t n = polygon.size();
            for (size_t i = 0, j = n - 1; i < n; j = i++) {
                const Vector3& pi = polygon[i];
                const Vector3& pj = polygon[j];
                const bool crosses = ((pi.z > z) != (pj.z > z)) &&
                    (x < (pj.x - pi.x) * (z - pi.z) / ((pj.z - pi.z) + 1.0e-12f) + pi.x);
                if (crosses) {
                    inside = !inside;
                }
            }
            return inside;
        }

        // add inter pts between each boundary pts
        std::vector<Vector3> ResampleBoundary(const std::vector<Vector3>& boundary, float spacing) {
            std::vector<Vector3> sampled;
            if (boundary.size() < 3 || spacing <= 0.0f) {
                return sampled;
            }

            for (size_t i = 0; i < boundary.size(); ++i) {
                const Vector3& a = boundary[i];
                const Vector3& b = boundary[(i + 1) % boundary.size()];
                const float length = DistanceXZ(a, b);
                const int segments = std::max(1, static_cast<int>(std::ceil(length / spacing)));

                for (int s = 0; s < segments; ++s) {
                    const float t = static_cast<float>(s) / static_cast<float>(segments);
                    sampled.push_back({
                        a.x + (b.x - a.x) * t,
                        a.y + (b.y - a.y) * t,
                        a.z + (b.z - a.z) * t
                    });
                }
            }
            return sampled;
        }

        // add bars based on the calced a b neighbors
        void AddBar(
            int a,
            int b,
            int maxNeighbors,
            std::vector<std::pair<int, int>>& bars,
            std::vector<int>& degree
        ) {
            if (a < 0 || b < 0 || a == b) {
                return;
            }
            if (a > b) {
                std::swap(a, b);
            }
            if (degree[static_cast<size_t>(a)] >= maxNeighbors ||
                degree[static_cast<size_t>(b)] >= maxNeighbors) {
                return;
            }
            if (std::find(bars.begin(), bars.end(), std::make_pair(a, b)) != bars.end()) {
                return;
            }

            bars.push_back({ a, b });
            degree[static_cast<size_t>(a)]++;
            degree[static_cast<size_t>(b)]++;
        }

        void BuildDirectedNeighbors(
            const std::vector<Particle>& particles,
            const std::vector<std::pair<int, int>>& bars,
            int maxNeighbors,
            std::vector<DirectedNeighbor>& neighbors,
            std::vector<int>& neighborCounts
        ) {
            neighbors.assign(particles.size() * static_cast<size_t>(maxNeighbors), DirectedNeighbor{});
            neighborCounts.assign(particles.size(), 0);

            for (const auto& bar : bars) {
                const int a = bar.first;
                const int b = bar.second;
                if (a < 0 || b < 0 ||
                    a >= static_cast<int>(particles.size()) ||
                    b >= static_cast<int>(particles.size())) {
                    continue;
                }

                const float restLength = DistanceXZ(particles[static_cast<size_t>(a)].position, particles[static_cast<size_t>(b)].position);
                const int countA = neighborCounts[static_cast<size_t>(a)];
                const int countB = neighborCounts[static_cast<size_t>(b)];

                if (countA < maxNeighbors) {
                    neighbors[static_cast<size_t>(a) * maxNeighbors + countA] = { b, restLength };
                    neighborCounts[static_cast<size_t>(a)]++;
                }
                if (countB < maxNeighbors) {
                    neighbors[static_cast<size_t>(b) * maxNeighbors + countB] = { a, restLength };
                    neighborCounts[static_cast<size_t>(b)]++;
                }
            }
        }
    }

    // GPU Calc; The result will be written into result
    bool SolveBoundaryCuda(
        const std::vector<Vector3>& boundaryPoints,
        const Config& config,
        Result& result
    ) {
        result = Result{};
        if (boundaryPoints.size() < 3 || config.spacing <= 0.0f ||
            config.steps <= 0 || config.dt <= 0.0f) {
            return false;
        }

        const int maxNeighbors = std::max(4, config.maxNeighborsPerParticle);
        const std::vector<Vector3> boundary = ResampleBoundary(boundaryPoints, config.spacing);

        if (boundary.size() < 3) {
            return false;
        }

        float minX = boundary[0].x;
        float maxX = boundary[0].x;
        float minZ = boundary[0].z;
        float maxZ = boundary[0].z;
        float avgY = 0.0f;
        for (const Vector3& p : boundary) {
            minX = std::min(minX, p.x);
            maxX = std::max(maxX, p.x);
            minZ = std::min(minZ, p.z);
            maxZ = std::max(maxZ, p.z);
            avgY += p.y;
        }
        avgY /= static_cast<float>(boundary.size());

        std::vector<Particle> particles;
        particles.reserve(boundary.size());
        
        for (const Vector3& p : boundary) {
            particles.emplace_back(p, true, config.particleMass);
        }

        const int cols = std::max(1, static_cast<int>(std::floor((maxX - minX) / config.spacing)) + 1);
        const int rows = std::max(1, static_cast<int>(std::floor((maxZ - minZ) / config.spacing)) + 1);
        std::vector<GridPoint> grid(static_cast<size_t>(cols) * static_cast<size_t>(rows));

        for (int row = 0; row < rows; ++row) {
            const float z = minZ + static_cast<float>(row) * config.spacing;
            for (int col = 0; col < cols; ++col) {
                const float x = minX + static_cast<float>(col) * config.spacing;
                GridPoint& gp = grid[static_cast<size_t>(row) * cols + col];
                gp.inside = IsInsidePolygonXZ(boundaryPoints, x, z);
                if (gp.inside) {
                    gp.index = static_cast<int>(particles.size());
                    particles.emplace_back(Vector3{ x, avgY, z }, false, config.particleMass);
                }
            }
        }

        std::vector<std::pair<int, int>> bars;
        std::vector<int> degree(particles.size(), 0);

        for (int i = 0; i < static_cast<int>(boundary.size()); ++i) {
            AddBar(i, (i + 1) % static_cast<int>(boundary.size()), maxNeighbors, bars, degree);
        }

        for (int row = 0; row < rows; ++row) {
            for (int col = 0; col < cols; ++col) {
                const GridPoint& gp = grid[static_cast<size_t>(row) * cols + col];
                if (!gp.inside) {
                    continue;
                }
                if (col + 1 < cols) {
                    AddBar(gp.index, grid[static_cast<size_t>(row) * cols + col + 1].index, maxNeighbors, bars, degree);
                }
                if (row + 1 < rows) {
                    AddBar(gp.index, grid[static_cast<size_t>(row + 1) * cols + col].index, maxNeighbors, bars, degree);
                }
            }
        }

        const float attachDist = config.spacing * 1.15f;
        for (int b = 0; b < static_cast<int>(boundary.size()); ++b) {
            int best = -1;
            float bestDist = attachDist;
            for (int i = static_cast<int>(boundary.size()); i < static_cast<int>(particles.size()); ++i) {
                const float d = DistanceXZ(boundary[static_cast<size_t>(b)], particles[static_cast<size_t>(i)].position);
                if (d < bestDist) {
                    bestDist = d;
                    best = i;
                }
            }
            AddBar(b, best, maxNeighbors, bars, degree);
        }

        std::vector<DirectedNeighbor> neighbors;
        std::vector<int> neighborCounts;
        BuildDirectedNeighbors(particles, bars, maxNeighbors, neighbors, neighborCounts);

        SolverParams params;
        params.stiffness = config.stiffness;
        params.dampingFactor = config.dampingFactor;
        params.airResistanceFactor = config.airResistanceFactor;
        params.gravity = config.gravity;
        params.steps = config.steps;
        params.dt = config.dt;
        params.maxNeighborsPerParticle = maxNeighbors;

        int stepsRun = 0;
        if (!RunLightningCuda(particles, neighbors, neighborCounts, params, stepsRun)) {
            std::cerr << "Lightning solver CUDA path unavailable or failed." << std::endl;
            return false;
        }

        result.particles = std::move(particles);
        result.bars = std::move(bars);
        result.stepsRun = stepsRun;
        result.ok = true;
        return true;
    }

#ifndef MESH3D_ENABLE_CUDA
    bool RunLightningCuda(
        std::vector<Particle>&,
        const std::vector<DirectedNeighbor>&,
        const std::vector<int>&,
        const SolverParams&,
        int&
    ) {
        return false;
    }
#endif
}
