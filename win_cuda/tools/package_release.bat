@echo off
setlocal

set "ROOT_DIR=%~dp0.."
pushd "%ROOT_DIR%" >nul

set "DIST_DIR=dist"
set "CPU_DIR=%DIST_DIR%\mesh3d_windows_cpu"
set "CUDA_DIR=%DIST_DIR%\mesh3d_windows_cuda"
set "CPU_EXE=build\desktop\mesh3d.exe"
set "CUDA_EXE=build\cuda_desktop_msvc\mesh3d_cuda.exe"

echo ============================================
echo  Packaging mesh3d release folders
echo ============================================
echo.

if not exist "%CPU_EXE%" (
    echo Missing CPU exe: %CPU_EXE%
    echo Build the CPU version first with build.bat.
    goto error
)

if not exist "%CUDA_EXE%" (
    echo Missing CUDA exe: %CUDA_EXE%
    echo Build the CUDA version first with tools\build_cuda_desktop.bat.
    goto error
)

if not exist "%DIST_DIR%" mkdir "%DIST_DIR%"
if exist "%CPU_DIR%" rmdir /s /q "%CPU_DIR%"
if exist "%CUDA_DIR%" rmdir /s /q "%CUDA_DIR%"
mkdir "%CPU_DIR%"
mkdir "%CUDA_DIR%"

copy "%CPU_EXE%" "%CPU_DIR%\" >nul
copy "%CUDA_EXE%" "%CUDA_DIR%\" >nul

call :copy_common "%CPU_DIR%"
call :copy_common "%CUDA_DIR%"

call :write_cpu_readme "%CPU_DIR%\README.txt"
call :write_cuda_readme "%CUDA_DIR%\README.txt"

echo CPU package:  %CPU_DIR%
echo CUDA package: %CUDA_DIR%
echo.
echo Package folders created. DLL dependency copying is not handled yet.
goto end

:copy_common
set "TARGET_DIR=%~1"
copy "default_config.txt" "%TARGET_DIR%\" >nul
copy "example_cloud.msh" "%TARGET_DIR%\" >nul
copy "example_cloud_2.msh" "%TARGET_DIR%\" >nul
if exist "resources" xcopy "resources" "%TARGET_DIR%\resources\" /e /i /y >nul
exit /b 0

:write_cpu_readme
(
echo mesh3d Windows CPU version
echo.
echo Run:
echo   mesh3d.exe
echo.
echo This version is recommended for most users.
echo It does not require an NVIDIA GPU.
echo.
echo Included files:
echo   default_config.txt
echo   example_cloud.msh
echo   example_cloud_2.msh
echo   resources\
echo.
echo If the app does not start, install the latest Microsoft Visual C++ Redistributable
echo or contact the creator with a screenshot of the error message.
) > "%~1"
exit /b 0

:write_cuda_readme
(
echo mesh3d Windows CUDA version
echo.
echo Run:
echo   mesh3d_cuda.exe
echo.
echo Requirements:
echo   Windows 10/11 64-bit
echo   NVIDIA GPU
echo   Recent NVIDIA driver
echo.
echo If this version does not start, try the CPU version first.
echo.
echo Included files:
echo   default_config.txt
echo   example_cloud.msh
echo   example_cloud_2.msh
echo   resources\
echo.
echo If the app reports missing DLLs, send the exact DLL name to the creator.
) > "%~1"
exit /b 0

:error
popd >nul
exit /b 1

:end
popd >nul
