cmake_minimum_required(VERSION 3.18)
project(mesh3d_cuda_smoke LANGUAGES CUDA CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CUDA_STANDARD 17)
set(CMAKE_CUDA_STANDARD_REQUIRED ON)

add_executable(cuda_smoke main.cu)

set_target_properties(cuda_smoke PROPERTIES
    CUDA_ARCHITECTURES 86
)
