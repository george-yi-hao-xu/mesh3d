@echo off
cd build
premake5.exe gmake2 || pause
cd ..
mingw32-make config=debug_x64 || pause
bin\Debug\Mesh3D.exe
pause
