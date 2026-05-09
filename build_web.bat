@echo off
setlocal enabledelayedexpansion

echo ============================================
echo  Building mesh3d for Web (Emscripten/WASM)
echo ============================================
echo.
echo  NOTE: Make sure emsdk is installed and activated first.
echo  https://emscripten.org/docs/getting_started/downloads.html
echo.
:: Verify emcc is available
where emcc >nul 2>&1
if errorlevel 1 (
    echo ERROR: emcc not found in PATH.
    echo Please install and activate emsdk first.
    pause
    exit /b 1
)

if not exist "build\web" mkdir "build\web"
if not exist "docs" mkdir "docs"

echo [Configure]
call emcmake cmake -B build/web -S . -G "MinGW Makefiles"
if errorlevel 1 goto error

echo [Build]
cmake --build build/web
if errorlevel 1 goto error

echo [Publish]
echo Web build artifacts are generated directly in docs\.

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
