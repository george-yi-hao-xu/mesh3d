# CUDA Spring Builder Notes

## Current Status

The project now has a CUDA-enabled desktop build path:

```bat
cd /d D:\yxu\mesh3d\win_cuda
call "C:\Program Files\Microsoft Visual Studio\18\Community\VC\Auxiliary\Build\vcvars64.bat"
set "PATH=C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v13.3\bin;%PATH%"
tools\build_cuda_desktop.bat
```

The CUDA build outputs:

```text
build\cuda_desktop_msvc\mesh3d_cuda.exe
```

The normal MinGW CPU build path remains separate.

## What CUDA Accelerates Now

Current spring build flow:

```text
GPU:
  For each particle, scan candidate particles.
  Compute distances.
  Keep the nearest top-K candidates.
  Return candidate indices and distances.

CPU:
  Convert GPU output into CandidateList.
  Shuffle candidates.
  Apply probability.
  Enforce max connection count.
  Create Spring objects.
```

So CUDA currently accelerates the heaviest part: candidate search and distance
calculation.

The final `Spring` objects are still created on the CPU.

## Why Spring Creation Still Runs On CPU

The current `Spring` representation uses CPU-side particle references/pointers.
That is convenient for the existing simulation code, but it is not a good fit
for GPU-side construction.

CUDA should not directly create CPU `Particle*` relationships. A safer GPU
output format is index based:

```cpp
struct SpringCandidateGpu {
    int a;
    int b;
    float distance;
};
```

The CPU can then convert those indices into the current `Spring` objects.

Moving full spring creation to GPU would be more natural after refactoring
`Spring` to store particle indices instead of pointers.

## Current Performance Observation

Before CUDA candidate search:

```text
48k particles: spring build around 4s
```

After first CUDA candidate backend:

```text
48k particles: spring build around 1s
```

This is already a useful improvement, but the current CUDA kernel is still a
simple all-pairs top-K search. Its complexity is still O(n^2); it is faster
because the distance work is parallelized on the GPU.

## Important Limitation

The current CUDA candidate kernel is intentionally simple:

```text
one CUDA thread = one particle
scan j > i
keep nearest top-K candidates
```

This is good for proving the CUDA data path and getting an early speedup.

It is not the final algorithm for very large particle counts. A future CUDA
spatial grid would reduce the number of candidate pairs that each particle has
to scan.

## Recommended Next Steps

1. Add CUDA build profiling:

```text
upload positions
kernel candidate search
download candidates
CPU candidate conversion
CPU shuffle/connect
```

2. Compare spring quality:

```text
spring count
spring length mean
spring length variance
visual stability
```

3. Decide whether CPU spring creation is a bottleneck.

If CPU connect time is still small, keep the current `Spring` structure.

If CPU connect becomes the bottleneck, consider an index-based spring refactor:

```cpp
struct Spring {
    int a;
    int b;
    float restLength;
    float stiffness;
};
```

4. Later, replace all-pairs CUDA search with CUDA spatial grid.

That would move the optimization from "parallel brute force" toward a better
algorithmic complexity.
