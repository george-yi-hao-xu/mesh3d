@echo off
setlocal

echo ============================================
echo  mesh3d CUDA toolchain probe
echo ============================================
echo.

echo [CMake]
where cmake >nul 2>nul
if errorlevel 1 (
    echo cmake: NOT FOUND
) else (
    for /f "delims=" %%A in ('where cmake') do echo cmake: %%A
    cmake --version | findstr /b /c:"cmake version"
)
echo.

echo [CUDA nvcc]
where nvcc >nul 2>nul
if errorlevel 1 (
    echo nvcc: NOT FOUND
) else (
    for /f "delims=" %%A in ('where nvcc') do echo nvcc: %%A
    nvcc --version
)
echo.

echo [MSVC cl]
where cl >nul 2>nul
if errorlevel 1 (
    echo cl: NOT FOUND
    echo hint: CUDA on Windows usually needs MSVC Build Tools in the active shell.
) else (
    for /f "delims=" %%A in ('where cl') do echo cl: %%A
    cl 2>&1 | findstr /b /c:"Microsoft"
)
echo.

echo [NVIDIA driver]
where nvidia-smi >nul 2>nul
if errorlevel 1 (
    echo nvidia-smi: NOT FOUND
) else (
    for /f "delims=" %%A in ('where nvidia-smi') do echo nvidia-smi: %%A
    nvidia-smi --query-gpu=name,driver_version,compute_cap --format=csv,noheader
)
echo.

echo Probe finished.
