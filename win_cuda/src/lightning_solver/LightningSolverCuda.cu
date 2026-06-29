#include "lightning_solver/LightningSolverInternal.h"

#include <cuda_runtime.h>

#include <vector>

namespace mesh3d::lightning {
    namespace {
        // GPU 端只需要每个粒子的静态属性；位置、速度、受力会放在独立数组里连续存储。
        struct DeviceParticleStatic {
            float mass = 1.0f;
            int fixed = 0;
        };

        __device__ float3 Add(float3 a, float3 b) {
            return make_float3(a.x + b.x, a.y + b.y, a.z + b.z);
        }

        __device__ float3 Sub(float3 a, float3 b) {
            return make_float3(a.x - b.x, a.y - b.y, a.z - b.z);
        }

        __device__ float3 Mul(float scalar, float3 value) {
            return make_float3(scalar * value.x, scalar * value.y, scalar * value.z);
        }

        __device__ float Dot(float3 a, float3 b) {
            return a.x * b.x + a.y * b.y + a.z * b.z;
        }

        __global__ void LightningStepKernel(
            const float3* oldPositions,
            const float3* oldVelocities,
            float3* newPositions,
            float3* newVelocities,
            float3* lastForces,
            const DeviceParticleStatic* particleStatic,
            const DirectedNeighbor* neighbors,
            const int* neighborCounts,
            int particleCount,
            int maxNeighbors,
            float stiffness,
            float dampingFactor,
            float airResistanceFactor,
            float gravity,
            float dt
        ) {
            // 一个 CUDA thread 负责更新一个粒子。
            const int i = blockIdx.x * blockDim.x + threadIdx.x;
            if (i >= particleCount) {
                return;
            }

            // 本 step 只从 old* 读取，写入 new*，避免同一步内读到其他 thread 刚写的新状态。
            const float3 position = oldPositions[i];
            const float3 velocity = oldVelocities[i];
            const DeviceParticleStatic stat = particleStatic[i];

            // 基础外力：空气阻力 + 重力。弹簧力会在下面按邻居累加。
            float3 force = make_float3(
                -airResistanceFactor * velocity.x * fabsf(velocity.x),
                -gravity - airResistanceFactor * velocity.y * fabsf(velocity.y),
                -airResistanceFactor * velocity.z * fabsf(velocity.z)
            );

            // neighbors 是 CPU 端由 bars 转换来的定长邻接表：
            // 第 i 个粒子的邻居范围是 [i * maxNeighbors, i * maxNeighbors + count)。
            const int count = neighborCounts[i];
            const int base = i * maxNeighbors;
            for (int k = 0; k < count; ++k) {
                const DirectedNeighbor neighbor = neighbors[base + k];
                const int j = neighbor.other;
                if (j < 0 || j >= particleCount) {
                    continue;
                }

                const float3 otherPosition = oldPositions[j];
                const float3 diff = Sub(otherPosition, position);
                const float lengthSq = Dot(diff, diff);
                if (lengthSq <= 1.0e-8f) {
                    continue;
                }

                const float length = sqrtf(lengthSq);
                const float3 dir = Mul(1.0f / length, diff);
                const float displacement = length - neighbor.restLength;

                // Hooke 弹簧力 + 沿弹簧方向的阻尼。
                const float3 otherVelocity = oldVelocities[j];
                const float3 velocityDiff = Sub(otherVelocity, velocity);
                const float velocityAlongSpring = Dot(velocityDiff, dir);
                const float forceScale = stiffness * displacement + dampingFactor * velocityAlongSpring;
                force = Add(force, Mul(forceScale, dir));
            }

            lastForces[i] = force;

            // 固定粒子不积分，只保留当前位置，并把速度清零。
            if (stat.fixed != 0) {
                newPositions[i] = position;
                newVelocities[i] = make_float3(0.0f, 0.0f, 0.0f);
                return;
            }

            // 半隐式 Euler：先用力更新速度，再用新速度更新位置。
            const float invMass = stat.mass > 0.0f ? 1.0f / stat.mass : 1.0f;
            const float3 acceleration = Mul(invMass, force);
            const float3 nextVelocity = Add(velocity, Mul(dt, acceleration));
            const float3 nextPosition = Add(position, Mul(dt, nextVelocity));

            newPositions[i] = nextPosition;
            newVelocities[i] = nextVelocity;
        }

        __global__ void CheckForceConvergenceKernel(
            const float3* forces,
            const DeviceParticleStatic* particleStatic,
            int particleCount,
            float thresholdSq,
            int* converged
        ) {
            const int i = blockIdx.x * blockDim.x + threadIdx.x;
            if (i >= particleCount || particleStatic[i].fixed != 0) {
                return;
            }

            const float3 force = forces[i];
            const float forceSq = Dot(force, force);
            if (forceSq > thresholdSq) {
                atomicExch(converged, 0);
            }
        }

        bool Check(cudaError_t result) {
            return result == cudaSuccess;
        }
    }

    bool RunLightningCuda(
        std::vector<Particle>& particles,
        const std::vector<DirectedNeighbor>& neighbors,
        const std::vector<int>& neighborCounts,
        const SolverParams& params,
        int& stepsRun
    ) {
        stepsRun = 0;
        // 基础参数非法时直接失败；上层会走错误提示。
        if (particles.empty() || params.steps <= 0 || params.dt <= 0.0f ||
            params.maxNeighborsPerParticle <= 0) {
            return false;
        }

        const int particleCount = static_cast<int>(particles.size());
        const size_t particleBytes = particles.size() * sizeof(float3);
        const size_t staticBytes = particles.size() * sizeof(DeviceParticleStatic);
        const size_t forceBytes = particles.size() * sizeof(float3);
        const size_t neighborBytes = neighbors.size() * sizeof(DirectedNeighbor);
        const size_t countBytes = neighborCounts.size() * sizeof(int);

        std::vector<float3> hostPositions(particles.size());
        std::vector<float3> hostVelocities(particles.size());
        std::vector<DeviceParticleStatic> hostStatic(particles.size());
        // 把 Particle 对象拆成 GPU 更适合处理的 SoA/连续数组。
        for (size_t i = 0; i < particles.size(); ++i) {
            hostPositions[i] = make_float3(particles[i].position.x, particles[i].position.y, particles[i].position.z);
            hostVelocities[i] = make_float3(particles[i].velocity.x, particles[i].velocity.y, particles[i].velocity.z);
            hostStatic[i] = { particles[i].mass, particles[i].isFixed ? 1 : 0 };
        }

        float3* devicePositionsA = nullptr;
        float3* devicePositionsB = nullptr;
        float3* deviceVelocitiesA = nullptr;
        float3* deviceVelocitiesB = nullptr;
        float3* deviceForces = nullptr;
        DeviceParticleStatic* deviceStatic = nullptr;
        DirectedNeighbor* deviceNeighbors = nullptr;
        int* deviceNeighborCounts = nullptr;
        int* deviceConverged = nullptr;

        // 位置和速度各分配两份 buffer，用于 step 之间 ping-pong 交换。
        bool ok =
            Check(cudaMalloc(&devicePositionsA, particleBytes)) &&
            Check(cudaMalloc(&devicePositionsB, particleBytes)) &&
            Check(cudaMalloc(&deviceVelocitiesA, particleBytes)) &&
            Check(cudaMalloc(&deviceVelocitiesB, particleBytes)) &&
            Check(cudaMalloc(&deviceForces, forceBytes)) &&
            Check(cudaMalloc(&deviceStatic, staticBytes)) &&
            Check(cudaMalloc(&deviceNeighbors, neighborBytes)) &&
            Check(cudaMalloc(&deviceNeighborCounts, countBytes)) &&
            Check(cudaMalloc(&deviceConverged, sizeof(int)));

        if (ok) {
            ok =
                Check(cudaMemcpy(devicePositionsA, hostPositions.data(), particleBytes, cudaMemcpyHostToDevice)) &&
                Check(cudaMemcpy(deviceVelocitiesA, hostVelocities.data(), particleBytes, cudaMemcpyHostToDevice)) &&
                Check(cudaMemcpy(deviceStatic, hostStatic.data(), staticBytes, cudaMemcpyHostToDevice)) &&
                Check(cudaMemcpy(deviceNeighbors, neighbors.data(), neighborBytes, cudaMemcpyHostToDevice)) &&
                Check(cudaMemcpy(deviceNeighborCounts, neighborCounts.data(), countBytes, cudaMemcpyHostToDevice));
        }

        // read* 是当前 step 的输入，write* 是当前 step 的输出。
        float3* readPositions = devicePositionsA;
        float3* writePositions = devicePositionsB;
        float3* readVelocities = deviceVelocitiesA;
        float3* writeVelocities = deviceVelocitiesB;

        if (ok) {
            const int threadsPerBlock = 128;
            const int blockCount = (particleCount + threadsPerBlock - 1) / threadsPerBlock;
            const float forceThresholdSq =
                params.forceConvergenceThreshold * params.forceConvergenceThreshold;
            for (int step = 0; step < params.steps; ++step) {
                // 每次 kernel launch 推进一个物理时间步。
                LightningStepKernel<<<blockCount, threadsPerBlock>>>(
                    readPositions,
                    readVelocities,
                    writePositions,
                    writeVelocities,
                    deviceForces,
                    deviceStatic,
                    deviceNeighbors,
                    deviceNeighborCounts,
                    particleCount,
                    params.maxNeighborsPerParticle,
                    params.stiffness,
                    params.dampingFactor,
                    params.airResistanceFactor,
                    params.gravity,
                    params.dt
                );
                ok = Check(cudaGetLastError());
                if (!ok) {
                    break;
                }

                int hostConverged = 1;
                if (params.forceConvergenceThreshold > 0.0f) {
                    ok = Check(cudaMemcpy(deviceConverged, &hostConverged, sizeof(int), cudaMemcpyHostToDevice));
                    if (!ok) {
                        break;
                    }

                    CheckForceConvergenceKernel<<<blockCount, threadsPerBlock>>>(
                        deviceForces,
                        deviceStatic,
                        particleCount,
                        forceThresholdSq,
                        deviceConverged
                    );
                    ok = Check(cudaGetLastError()) &&
                        Check(cudaMemcpy(&hostConverged, deviceConverged, sizeof(int), cudaMemcpyDeviceToHost));
                    if (!ok) {
                        break;
                    }
                }

                // 输出 buffer 变成下一步输入；不需要在 GPU 上拷贝整块位置/速度数据。
                std::swap(readPositions, writePositions);
                std::swap(readVelocities, writeVelocities);
                stepsRun++;
                if (params.forceConvergenceThreshold > 0.0f && hostConverged != 0) {
                    break;
                }
            }
            ok = ok && Check(cudaDeviceSynchronize());
        }

        if (ok) {
            // read* 指向最后一次完成交换后的最新状态。
            ok =
                Check(cudaMemcpy(hostPositions.data(), readPositions, particleBytes, cudaMemcpyDeviceToHost)) &&
                Check(cudaMemcpy(hostVelocities.data(), readVelocities, particleBytes, cudaMemcpyDeviceToHost));
        }

        std::vector<float3> hostForces(particles.size());
        if (ok) {
            ok = Check(cudaMemcpy(hostForces.data(), deviceForces, forceBytes, cudaMemcpyDeviceToHost));
        }

        cudaFree(devicePositionsA);
        cudaFree(devicePositionsB);
        cudaFree(deviceVelocitiesA);
        cudaFree(deviceVelocitiesB);
        cudaFree(deviceForces);
        cudaFree(deviceStatic);
        cudaFree(deviceNeighbors);
        cudaFree(deviceNeighborCounts);
        cudaFree(deviceConverged);

        if (!ok) {
            return false;
        }

        // 把 GPU 结果写回原来的 Particle 对象，供渲染和 CPU 侧 Mesh 逻辑继续使用。
        for (size_t i = 0; i < particles.size(); ++i) {
            particles[i].position = { hostPositions[i].x, hostPositions[i].y, hostPositions[i].z };
            particles[i].velocity = { hostVelocities[i].x, hostVelocities[i].y, hostVelocities[i].z };
            particles[i].lastFrameNetForce = { hostForces[i].x, hostForces[i].y, hostForces[i].z };
            particles[i].force = { 0.0f, 0.0f, 0.0f };
        }
        return true;
    }
}
