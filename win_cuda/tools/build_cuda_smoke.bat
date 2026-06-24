@echo off
setlocal

echo ============================================
echo  Building mesh3d CUDA smoke test
echo ============================================
echo.

set "BUILD_DIR=build\cuda_smoke"

cmake -B "%BUILD_DIR%" -S "tools\cuda_smoke" -G "NMake Makefiles"
if errorlevel 1 goto error

cmake --build "%BUILD_DIR%"
if errorlevel 1 goto error

"%BUILD_DIR%\cuda_smoke.exe"
if errorlevel 1 goto error

echo.
echo CUDA smoke test completed successfully.
goto end

:error
echo.
echo CUDA smoke test failed.
echo Make sure this runs inside an MSVC Developer Command Prompt with CUDA Toolkit installed.
exit /b 1

:end
