#include "SpringBuilderCudaKernels.h"

#include <cuda_runtime.h>

#include <algorithm>
#include <chrono>
#include <cmath>
#include <iostream>

namespace mesh3d {
    namespace {
        // 每个粒子最多保留多少个候选弹簧连接。
        // 这里是编译期常量，因为 CUDA kernel 里的局部数组需要固定上限。
        constexpr int CUDA_MAX_CANDIDATES_PER_PARTICLE = 64;

        // 只把 position 传到 GPU。
        // Particle 里还有 velocity/force/mass/isFixed，但 build spring 候选只需要位置。
        struct DevicePosition {
            float x;
            float y;
            float z;
        };

        // 最小 CUDA 探针 kernel：写一个固定值，证明主程序可以真正运行 CUDA kernel。
        __global__ void ProbeKernel(int* output) {
            *output = 1234;
        }

        // 第一版真实候选搜索 kernel。
        // 设计很直接：一个 CUDA thread 负责一个粒子 i，扫描 j > i 的所有粒子，
        // 找出距离不超过 maxDist 的最近 top-K 个候选。
        //
        // 注意：这还是 O(n^2) 搜索，只是把大量距离计算搬到了 GPU。
        // 后续如果需要更快，再把 CPU spatial grid 思路搬到 GPU。
        __global__ void BuildTopKCandidatesKernel(
            const DevicePosition* positions,
            int particleCount,
            float maxDistSq,
            int perParticleLimit,
            int* outIndices,
            float* outDistances
        ) {
            const int i = blockIdx.x * blockDim.x + threadIdx.x;
            if (i >= particleCount) {
                return;
            }

            // 每个 thread 自己维护一个小的 top-K 表。
            // bestDistSq[k] 和 bestIndex[k] 是一一对应的。
            float bestDistSq[CUDA_MAX_CANDIDATES_PER_PARTICLE];
            int bestIndex[CUDA_MAX_CANDIDATES_PER_PARTICLE];

            // 遍历-初始状态
            for (int k = 0; k < perParticleLimit; ++k) {
                bestDistSq[k] = maxDistSq;
                bestIndex[k] = -1;
            }

            const DevicePosition a = positions[i];
            for (int j = i + 1; j < particleCount; ++j) {
                const DevicePosition b = positions[j];
                const float dx = a.x - b.x;
                const float dy = a.y - b.y;
                const float dz = a.z - b.z;
                const float distSq = dx * dx + dy * dy + dz * dz;

                if (distSq <= 0.00000001f || distSq > maxDistSq) {
                    continue;
                }

                // 找出当前 top-K 表里最差的 slot。
                // 如果新候选比最差 slot 更近，就替换它。
                int worstSlot = 0;
                float worstDistSq = bestDistSq[0];
                for (int k = 1; k < perParticleLimit; ++k) {
                    if (bestDistSq[k] > worstDistSq) {
                        worstDistSq = bestDistSq[k];
                        worstSlot = k;
                    }
                }

                if (bestIndex[worstSlot] < 0 || distSq < worstDistSq) {
                    bestDistSq[worstSlot] = distSq;
                    bestIndex[worstSlot] = j;
                }
            }

            // 输出是一个固定大小的二维表，逻辑上是：
            // out[i][k] = 第 i 个粒子的第 k 个候选。
            // 用一维数组存储，所以 index = i * perParticleLimit + k。
            const int outBase = i * perParticleLimit;
            for (int k = 0; k < perParticleLimit; ++k) {
                outIndices[outBase + k] = bestIndex[k];
                outDistances[outBase + k] = bestIndex[k] >= 0 ? sqrtf(bestDistSq[k]) : 0.0f;
            }
        }

        bool Check(cudaError_t result) {
            return result == cudaSuccess;
        }
    }

    bool RunCudaSpringBuilderProbe() {
        int hostValue = 0;
        int* deviceValue = nullptr;

        if (!Check(cudaMalloc(&deviceValue, sizeof(int)))) {
            return false;
        }

        ProbeKernel<<<1, 1>>>(deviceValue);
        const bool ok =
            Check(cudaGetLastError()) &&
            Check(cudaDeviceSynchronize()) &&
            Check(cudaMemcpy(&hostValue, deviceValue, sizeof(int), cudaMemcpyDeviceToHost));

        cudaFree(deviceValue);
        return ok && hostValue == 1234;
    }

    bool BuildSpringCandidatesCudaProbe(
        const std::vector<Particle>&,
        std::vector<SpringCandidateGpu>& candidates
    ) {
        candidates.clear();
        return RunCudaSpringBuilderProbe();
    }

    bool BuildLimitedSpringCandidatesCuda(
        const std::vector<Particle>& particles,
        float maxDist,
        int maxPerParticle,
        std::vector<CandidateList>& candidates,
        SpringBuildProfile* profile
    ) {
        candidates.assign(particles.size(), CandidateList{});
        if (particles.size() < 2 || maxDist <= 0.0f || maxPerParticle <= 0) {
            return false;
        }

        const auto start = std::chrono::steady_clock::now();
        const int particleCount = static_cast<int>(particles.size());

        // CPU 版本会保留 maxPerParticle * 8 个候选，给后面的 shuffle/probability 留空间。
        // CUDA 版本同样这样做，但再用 CUDA_MAX_CANDIDATES_PER_PARTICLE 限制上限，
        // 避免单个 thread 的局部数组过大。
        const int requestedLimit = std::max(1, maxPerParticle * 8);
        const int perParticleLimit = std::min(requestedLimit, CUDA_MAX_CANDIDATES_PER_PARTICLE);
        const size_t outputCount = static_cast<size_t>(particleCount) * static_cast<size_t>(perParticleLimit);

        // 把 Particle 转成紧凑的 position 数组，减少传给 GPU 的数据量。
        std::vector<DevicePosition> hostPositions(particles.size());
        for (size_t i = 0; i < particles.size(); ++i) {
            hostPositions[i] = {
                particles[i].position.x,
                particles[i].position.y,
                particles[i].position.z
            };
        }

        DevicePosition* devicePositions = nullptr;
        int* deviceIndices = nullptr;
        float* deviceDistances = nullptr;

        const size_t positionsBytes = hostPositions.size() * sizeof(DevicePosition);
        const size_t indicesBytes = outputCount * sizeof(int);
        const size_t distancesBytes = outputCount * sizeof(float);

        // 分配 GPU 内存：
        // devicePositions 输入粒子位置；
        // deviceIndices/deviceDistances 输出候选点下标和距离。
        bool ok =
            Check(cudaMalloc(&devicePositions, positionsBytes)) &&
            Check(cudaMalloc(&deviceIndices, indicesBytes)) &&
            Check(cudaMalloc(&deviceDistances, distancesBytes));

        // CPU -> GPU：上传粒子位置。
        if (ok) {
            ok = Check(cudaMemcpy(devicePositions, hostPositions.data(), positionsBytes, cudaMemcpyHostToDevice));
        }

        // 启动 kernel。128 threads/block 是一个保守默认值，后面可以根据 profiling 调整。
        if (ok) {
            const int threadsPerBlock = 128;
            const int blockCount = (particleCount + threadsPerBlock - 1) / threadsPerBlock;
            BuildTopKCandidatesKernel<<<blockCount, threadsPerBlock>>>(
                devicePositions,
                particleCount,
                maxDist * maxDist,
                perParticleLimit,
                deviceIndices,
                deviceDistances
            );
            ok = Check(cudaGetLastError()) && Check(cudaDeviceSynchronize());
        }

        // GPU -> CPU：下载候选结果。
        std::vector<int> hostIndices(outputCount, -1);
        std::vector<float> hostDistances(outputCount, 0.0f);
        if (ok) {
            ok =
                Check(cudaMemcpy(hostIndices.data(), deviceIndices, indicesBytes, cudaMemcpyDeviceToHost)) &&
                Check(cudaMemcpy(hostDistances.data(), deviceDistances, distancesBytes, cudaMemcpyDeviceToHost));
        }

        cudaFree(devicePositions);
        cudaFree(deviceIndices);
        cudaFree(deviceDistances);

        // CUDA 失败时返回 false，让调用方自动回退 CPU spatial grid。
        if (!ok) {
            candidates.clear();
            return false;
        }

        // 把固定大小的 GPU 输出表转回现有 CPU 数据结构：
        // candidates[i] 是第 i 个粒子的候选列表，元素是 {j, distance}。
        size_t candidateCount = 0;
        for (int i = 0; i < particleCount; ++i) {
            CandidateList& local = candidates[static_cast<size_t>(i)];
            const size_t base = static_cast<size_t>(i) * static_cast<size_t>(perParticleLimit);
            local.reserve(static_cast<size_t>(perParticleLimit));

            for (int k = 0; k < perParticleLimit; ++k) {
                const int j = hostIndices[base + static_cast<size_t>(k)];
                if (j > i) {
                    local.push_back({ static_cast<size_t>(j), hostDistances[base + static_cast<size_t>(k)] });
                }
            }
            candidateCount += local.size();
        }

        if (profile != nullptr) {
            const auto end = std::chrono::steady_clock::now();
            profile->gridBuildMs = 0.0;
            profile->candidateSearchMs = std::chrono::duration<double, std::milli>(end - start).count();
            profile->candidateCount = candidateCount;
        }

        std::cout << "CUDA spring candidate backend: particles=" << particles.size()
            << ", perParticleLimit=" << perParticleLimit
            << ", candidates=" << candidateCount
            << std::endl;
        return true;
    }
}
