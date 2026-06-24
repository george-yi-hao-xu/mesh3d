#include <cuda_runtime.h>

#include <iostream>

namespace {
    __global__ void AddKernel(const float* a, const float* b, float* out, int count) {
        const int i = blockIdx.x * blockDim.x + threadIdx.x;
        if (i < count) {
            out[i] = a[i] + b[i];
        }
    }

    bool CheckCuda(cudaError_t result, const char* label) {
        if (result == cudaSuccess) {
            return true;
        }

        std::cerr << label << " failed: " << cudaGetErrorString(result) << std::endl;
        return false;
    }
}

int main() {
    constexpr int count = 256;
    constexpr int bytes = count * sizeof(float);

    float hostA[count] = {};
    float hostB[count] = {};
    float hostOut[count] = {};

    for (int i = 0; i < count; ++i) {
        hostA[i] = static_cast<float>(i);
        hostB[i] = static_cast<float>(count - i);
    }

    float* deviceA = nullptr;
    float* deviceB = nullptr;
    float* deviceOut = nullptr;

    if (!CheckCuda(cudaMalloc(&deviceA, bytes), "cudaMalloc deviceA")) return 1;
    if (!CheckCuda(cudaMalloc(&deviceB, bytes), "cudaMalloc deviceB")) return 1;
    if (!CheckCuda(cudaMalloc(&deviceOut, bytes), "cudaMalloc deviceOut")) return 1;

    if (!CheckCuda(cudaMemcpy(deviceA, hostA, bytes, cudaMemcpyHostToDevice), "copy A")) return 1;
    if (!CheckCuda(cudaMemcpy(deviceB, hostB, bytes, cudaMemcpyHostToDevice), "copy B")) return 1;

    AddKernel<<<1, count>>>(deviceA, deviceB, deviceOut, count);
    if (!CheckCuda(cudaGetLastError(), "AddKernel launch")) return 1;
    if (!CheckCuda(cudaDeviceSynchronize(), "AddKernel sync")) return 1;

    if (!CheckCuda(cudaMemcpy(hostOut, deviceOut, bytes, cudaMemcpyDeviceToHost), "copy output")) return 1;

    cudaFree(deviceA);
    cudaFree(deviceB);
    cudaFree(deviceOut);

    for (int i = 0; i < count; ++i) {
        if (hostOut[i] != static_cast<float>(count)) {
            std::cerr << "unexpected result at " << i << ": " << hostOut[i] << std::endl;
            return 1;
        }
    }

    std::cout << "CUDA smoke test passed." << std::endl;
    return 0;
}
