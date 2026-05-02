@echo off
setlocal enabledelayedexpansion

echo ============================================
echo  Building mesh3d for Web (Emscripten/WASM)
echo ============================================
echo.
echo  NOTE: Make sure emsdk is installed and activated first on your local machine.
echo  https://emscripten.org/docs/getting_started/downloads.html#platform-notes-installation-instructions-sdk 
echo.
echo   1. cd build\emsdk
echo   2. .\emsdk.bat install latest
echo   3. .\emsdk.bat activate latest
echo   4. Open a NEW terminal window
echo   5. Run this script: .\build_web.bat
echo.
echo ============================================

:: Verify emcc is available
where emcc >nul 2>&1
if errorlevel 1 (
    echo ERROR: emcc not found in PATH.
    echo Please install and activate emsdk first ^(see instructions above^).
    pause
    exit /b 1
)

if not exist "build\web" mkdir build\web
if not exist "docs" mkdir docs

powershell -Command "(Get-Content 'build/external/raylib-master/src/shell.html') -replace 'raylib web game', '3D Cloth Simulation' | Set-Content 'docs/shell.html'"

set "RAYLIB_SRC=build/external/raylib-master/src"
set "INCLUDES=-I%RAYLIB_SRC% -I%RAYLIB_SRC%/external -I%RAYLIB_SRC%/external/glfw/include -Iinclude -Isrc -Ibuild/external"
set "DEFINES=-DPLATFORM_WEB -DGRAPHICS_API_OPENGL_ES2 -D_GNU_SOURCE"
set "CFLAGS=-O3 %DEFINES% %INCLUDES% -Wall -Wno-missing-braces"
set "CXXFLAGS=-O3 %DEFINES% %INCLUDES% -Wall -Wno-missing-braces -std=c++17"

echo [1/3] Compiling raylib...
emcc -c %RAYLIB_SRC%/rcore.c     -o build/web/rcore.o     %CFLAGS%
if errorlevel 1 goto error
emcc -c %RAYLIB_SRC%/rshapes.c   -o build/web/rshapes.o   %CFLAGS%
if errorlevel 1 goto error
emcc -c %RAYLIB_SRC%/rtextures.c -o build/web/rtextures.o %CFLAGS%
if errorlevel 1 goto error
emcc -c %RAYLIB_SRC%/rtext.c     -o build/web/rtext.o     %CFLAGS%
if errorlevel 1 goto error
emcc -c %RAYLIB_SRC%/utils.c     -o build/web/utils.o     %CFLAGS%
if errorlevel 1 goto error
emcc -c %RAYLIB_SRC%/rmodels.c   -o build/web/rmodels.o   %CFLAGS%
if errorlevel 1 goto error
emcc -c %RAYLIB_SRC%/raudio.c    -o build/web/raudio.o    %CFLAGS%
if errorlevel 1 goto error

echo [2/3] Compiling project sources...
emcc -c src/Mesh.cpp     -o build/web/Mesh.o     %CXXFLAGS%
if errorlevel 1 goto error
emcc -c src/Particle.cpp -o build/web/Particle.o %CXXFLAGS%
if errorlevel 1 goto error
emcc -c src/Spring.cpp   -o build/web/Spring.o   %CXXFLAGS%
if errorlevel 1 goto error
emcc -c src/main.cpp     -o build/web/main.o     %CXXFLAGS%
if errorlevel 1 goto error

echo [3/3] Linking...
emcc build/web/*.o -o docs/index.html -s USE_GLFW=3 -s ASYNCIFY=1 -s TOTAL_MEMORY=134217728 -s EXPORTED_RUNTIME_METHODS=[FS] --preload-file config.txt --shell-file docs/shell.html
if errorlevel 1 goto error

echo. > docs\.nojekyll

echo.
echo ============================================
echo  BUILD SUCCESSFUL!
echo ============================================
echo Output files in docs/:
dir docs\*.* /b
echo.
echo Next steps:
echo   1. git add docs
echo   2. git commit -m "Add web build"
echo   3. git push
echo   4. On GitHub: Settings ^> Pages ^> Source = Deploy from branch
echo      Select "main" and "/docs", then Save.
goto end

:error
echo.
echo ============================================
echo  BUILD FAILED
echo ============================================
exit /b 1

:end
pause
