cmake_minimum_required(VERSION 3.16)
project(mesh3d LANGUAGES CXX C)

include(FetchContent)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

option(MESH3D_ENABLE_CUDA "Enable CUDA spring builder backend" OFF)

# ============================================
# Source files
# ============================================
set(PROJECT_SOURCES
    src/AppActions.cpp
    src/AppUi.cpp
    src/CameraControls.cpp
    src/main.cpp
    src/Mesh.cpp
    src/Particle.cpp
    src/ParticleRenderer.cpp
    src/raygui_impl.c
    src/SceneDraw.cpp
    src/Spring.cpp
    src/SpringBuilder.cpp
    src/SpringBuilderCuda.cpp
    src/FileDialog.cpp
)

set(PROJECT_HEADERS
    include/AppActions.h
    include/AppState.h
    include/AppUi.h
    include/CameraControls.h
    include/FileDialog.h
    include/Mesh.h
    include/Particle.h
    include/Particle.hpp
    include/ParticleRenderer.h
    include/SceneDraw.h
    include/Spring.h
    include/SpringBuilder.h
    include/SpringBuilderCuda.h
    include/resource_dir.h
)

if(MESH3D_ENABLE_CUDA)
    enable_language(CUDA)
    list(APPEND PROJECT_SOURCES src/SpringBuilderCudaKernels.cu)
    list(APPEND PROJECT_HEADERS src/SpringBuilderCudaKernels.h)
endif()

# ============================================
# Platform defines (applied globally so raylib also gets them)
# ============================================
if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    add_compile_definitions(PLATFORM_DESKTOP _GLFW_X11 _GNU_SOURCE)
else()
    add_compile_definitions(PLATFORM_DESKTOP)
endif()

# Disable examples to speed up build
set(BUILD_EXAMPLES OFF CACHE BOOL "" FORCE)

FetchContent_Declare(
    raylib
    GIT_REPOSITORY https://github.com/raysan5/raylib.git
    GIT_TAG 5.5
    GIT_SHALLOW TRUE
)

FetchContent_Declare(
    raygui
    GIT_REPOSITORY https://github.com/raysan5/raygui.git
    GIT_TAG master
    GIT_SHALLOW TRUE
)

FetchContent_MakeAvailable(raylib raygui)

# ============================================
# Executable
# ============================================
add_executable(${PROJECT_NAME} ${PROJECT_SOURCES} ${PROJECT_HEADERS})

if(MESH3D_ENABLE_CUDA)
    target_compile_definitions(${PROJECT_NAME} PRIVATE MESH3D_ENABLE_CUDA)
    set_target_properties(${PROJECT_NAME} PROPERTIES
        CUDA_ARCHITECTURES 86
        OUTPUT_NAME mesh3d_cuda
    )
endif()

target_include_directories(${PROJECT_NAME} PRIVATE
    include
    src
    ${raygui_SOURCE_DIR}/src
    ${raylib_SOURCE_DIR}/examples/shapes
    ${raylib_SOURCE_DIR}/src
    ${raylib_SOURCE_DIR}/src/external/glfw/include
)

target_link_libraries(${PROJECT_NAME} PRIVATE raylib)

# ============================================
# Platform link libraries
# ============================================
if(WIN32)
    target_link_libraries(${PROJECT_NAME} PRIVATE winmm gdi32 opengl32 comdlg32)
    # Keep as console app for now; set to WIN32 only if you want to hide console
    # set_property(TARGET ${PROJECT_NAME} PROPERTY WIN32_EXECUTABLE TRUE)
elseif(UNIX AND NOT APPLE)
    target_link_libraries(${PROJECT_NAME} PRIVATE pthread m dl rt X11)
elseif(APPLE)
    target_link_libraries(${PROJECT_NAME} PRIVATE
        "-framework OpenGL"
        "-framework Cocoa"
        "-framework IOKit"
        "-framework CoreFoundation"
        "-framework CoreAudio"
        "-framework CoreVideo"
        "-framework AudioToolbox"
    )
endif()
