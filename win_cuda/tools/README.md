# mesh3d tools

This folder contains small helper tools that are independent from the main
desktop build.

## CUDA smoke test on Windows

Use this when checking whether the machine can compile and run a minimal CUDA
program before wiring CUDA into the main spring builder.

### 1. Open a command prompt

You can use a normal `cmd.exe`, but it must load the Visual Studio x64 compiler
environment first.

```bat
cd /d D:\yxu\mesh3d\win_cuda
call "C:\Program Files\Microsoft Visual Studio\18\Community\VC\Auxiliary\Build\vcvars64.bat"
set "PATH=C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v13.3\bin;%PATH%"
```

The `set PATH=...` command only changes the current command prompt session. It
does not modify the system PATH.

### 2. Confirm the toolchain

```bat
where cl
cl
tools\cuda_probe.bat
```

The first `where cl` result should be the x64 compiler:

```text
C:\Program Files\Microsoft Visual Studio\18\Community\VC\Tools\MSVC\14.50.35717\bin\Hostx64\x64\cl.exe
```

The `cl` banner should include `for x64`.

`tools\cuda_probe.bat` should find:

```text
nvcc: C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v13.3\bin\nvcc.exe
cl: ...\Hostx64\x64\cl.exe
nvidia-smi: ...
```

### 3. Build and run the CUDA smoke test

```bat
tools\build_cuda_smoke.bat
```

Success output:

```text
CUDA smoke test passed.
```

If this fails, keep the full command output. The most useful lines are the first
CMake error and any compiler name or architecture shown near it.
