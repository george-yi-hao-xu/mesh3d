@echo off
setlocal enabledelayedexpansion

echo ============================================
echo  Building mesh3d (Desktop)
echo ============================================

if not exist "build\desktop" mkdir "build\desktop"

echo [Configure]
cmake -B build/desktop -S . -G "MinGW Makefiles"
if errorlevel 1 goto error

echo [Build]
cmake --build build/desktop
if errorlevel 1 goto error

echo.
echo ============================================
echo  BUILD SUCCESSFUL!
echo ============================================
echo Running...
build\desktop\mesh3d.exe
goto end

:error
echo.
echo ============================================
echo  BUILD FAILED
echo ============================================
exit /b 1

:end
pause
