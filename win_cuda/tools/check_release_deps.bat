@echo off
setlocal

set "ROOT_DIR=%~dp0.."
pushd "%ROOT_DIR%" >nul

set "CPU_EXE=build\desktop\mesh3d.exe"
set "CUDA_EXE=build\cuda_desktop_msvc\mesh3d_cuda.exe"

echo ============================================
echo  Checking mesh3d release DLL dependencies
echo ============================================
echo.

where objdump >nul 2>nul
if errorlevel 1 (
    echo objdump was not found in PATH.
    echo Install or add MinGW bin to PATH, or run from an environment that has objdump.
    goto error
)

if exist "%CPU_EXE%" (
    call :print_deps "CPU" "%CPU_EXE%"
) else (
    echo CPU exe missing: %CPU_EXE%
)

echo.

if exist "%CUDA_EXE%" (
    call :print_deps "CUDA" "%CUDA_EXE%"
) else (
    echo CUDA exe missing: %CUDA_EXE%
)

echo.
echo Notes:
echo   Windows system DLLs do not need to be copied.
echo   cudart*.dll is needed for CUDA builds if it is not already on the user's PATH.
echo   MSVC runtime DLLs are usually handled by installing Microsoft Visual C++ Redistributable.
goto end

:print_deps
echo [%~1] %~2
objdump -p "%~2" | findstr /c:"DLL Name"
exit /b 0

:error
popd >nul
exit /b 1

:end
popd >nul
