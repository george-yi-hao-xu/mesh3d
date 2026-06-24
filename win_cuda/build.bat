@echo off
setlocal enabledelayedexpansion

echo ============================================
echo  Building mesh3d (Desktop w CUDA)
echo ============================================

set "BUILD_DIR=build\desktop"
set "CACHE_FILE=%BUILD_DIR%\CMakeCache.txt"
set "CURRENT_SOURCE=%CD:\=/%"

if exist "%CACHE_FILE%" (
    for /f "tokens=2 delims==" %%A in ('findstr /b /c:"CMAKE_HOME_DIRECTORY:INTERNAL=" "%CACHE_FILE%"') do set "CACHE_SOURCE=%%A"
    if defined CACHE_SOURCE (
        if /i not "!CACHE_SOURCE!"=="!CURRENT_SOURCE!" (
            echo [Clean] Removing stale CMake cache from !CACHE_SOURCE!
            rmdir /s /q "%BUILD_DIR%"
        )
    )
)

if not exist "build\desktop" mkdir "build\desktop"

echo [Configure]
cmake -B "%BUILD_DIR%" -S . -G "MinGW Makefiles"
if errorlevel 1 goto error

echo [Build]
cmake --build "%BUILD_DIR%"
if errorlevel 1 goto error

echo.
echo ============================================
echo  BUILD SUCCESSFUL!
echo ============================================
if "%MESH3D_SKIP_RUN%"=="1" goto end
echo Running...
"%BUILD_DIR%\mesh3d.exe"
goto end

:error
echo.
echo ============================================
echo  BUILD FAILED
echo ============================================
exit /b 1

:end
pause
