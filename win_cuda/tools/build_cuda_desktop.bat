@echo off
setlocal

echo ============================================
echo  Building mesh3d desktop with CUDA flag
echo ============================================
echo.

set "BUILD_DIR=build\cuda_desktop_msvc"
set "CMAKE_EXE=cmake"
set "VS_CMAKE=C:\Program Files\Microsoft Visual Studio\18\Community\Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin\cmake.exe"

if exist "%VS_CMAKE%" set "CMAKE_EXE=%VS_CMAKE%"

"%CMAKE_EXE%" -B "%BUILD_DIR%" -S . -G "NMake Makefiles" -DMESH3D_ENABLE_CUDA=ON
if errorlevel 1 goto error

"%CMAKE_EXE%" --build "%BUILD_DIR%"
if errorlevel 1 goto error

echo.
echo CUDA desktop build completed successfully.
echo Output: %BUILD_DIR%\mesh3d_cuda.exe
goto end

:error
echo.
echo CUDA desktop build failed.
echo Make sure this runs inside an MSVC x64 environment with CUDA Toolkit available.
exit /b 1

:end
