cmake_minimum_required(VERSION 3.16)
project(mesh3d LANGUAGES CXX C)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# ============================================
# Source files
# ============================================
set(PROJECT_SOURCES
    src/main.cpp
    src/Mesh.cpp
    src/Particle.cpp
    src/Spring.cpp
    src/FileDialog.cpp
)

set(PROJECT_HEADERS
    include/FileDialog.h
    include/Mesh.h
    include/Particle.h
    include/Particle.hpp
    include/Spring.h
    include/resource_dir.h
)

# ============================================
# Executable
# ============================================
add_executable(${PROJECT_NAME} ${PROJECT_SOURCES} ${PROJECT_HEADERS})

target_include_directories(${PROJECT_NAME} PRIVATE
    include
    src
    build/external
    build/external/raylib-master/examples/shapes
    build/external/raylib-master/src
    build/external/raylib-master/src/external/glfw/include
)

# ============================================
# Platform defines (applied globally so raylib also gets them)
# ============================================
if(EMSCRIPTEN)
    add_compile_definitions(PLATFORM_WEB GRAPHICS_API_OPENGL_ES2 _GNU_SOURCE)
elseif(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    add_compile_definitions(PLATFORM_DESKTOP _GLFW_X11 _GNU_SOURCE)
else()
    add_compile_definitions(PLATFORM_DESKTOP)
endif()

# ============================================
# raylib
# ============================================
if(EMSCRIPTEN)
    # Force raylib to build for Web platform when using Emscripten
    set(PLATFORM "Web" CACHE STRING "" FORCE)
    set(OPENGL_VERSION "OFF" CACHE STRING "" FORCE)
endif()

# Disable examples to speed up build
set(BUILD_EXAMPLES OFF CACHE BOOL "" FORCE)

add_subdirectory(build/external/raylib-master)
target_link_libraries(${PROJECT_NAME} PRIVATE raylib)

# ============================================
# Desktop-specific link libraries
# ============================================
if(NOT EMSCRIPTEN)
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
endif()

# ============================================
# Emscripten / Web build settings
# ============================================
if(EMSCRIPTEN)
    set(CMAKE_EXECUTABLE_SUFFIX ".html")

    # Ensure shell.html exists (copy from raylib template with title replacement)
    if(NOT EXISTS "${CMAKE_SOURCE_DIR}/docs/shell.html")
        file(READ "${CMAKE_SOURCE_DIR}/build/external/raylib-master/src/shell.html" SHELL_HTML_CONTENT)
        string(REPLACE "raylib web game" "3D Cloth Simulation" SHELL_HTML_CONTENT "${SHELL_HTML_CONTENT}")
        file(WRITE "${CMAKE_SOURCE_DIR}/docs/shell.html" "${SHELL_HTML_CONTENT}")
    endif()

    target_link_options(${PROJECT_NAME} PRIVATE
        "SHELL:-s USE_GLFW=3"
        "SHELL:-s ASYNCIFY=1"
        "SHELL:-s TOTAL_MEMORY=134217728"
        "SHELL:-s EXPORTED_RUNTIME_METHODS=['FS']"
        "SHELL:--preload-file ${CMAKE_SOURCE_DIR}/default_config.txt@default_config.txt"
        "SHELL:--preload-file ${CMAKE_SOURCE_DIR}/example_cloud.msh@example_cloud.msh"
        "SHELL:--shell-file ${CMAKE_SOURCE_DIR}/docs/shell.html"
    )

    set_target_properties(${PROJECT_NAME} PROPERTIES
        OUTPUT_NAME "index"
        RUNTIME_OUTPUT_DIRECTORY ${CMAKE_SOURCE_DIR}/docs
    )
endif()
