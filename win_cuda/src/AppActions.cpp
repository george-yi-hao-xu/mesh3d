#include "AppActions.h"

#include "raylib.h"

#include <algorithm>

namespace {
    constexpr int maxStepsPerFrame = 20;
    constexpr float fixedDt = 1.0f / 120.0f;
    float simAccumulator = 0.0f;

    void ResetSimAccumulator() {
        simAccumulator = 0.0f;
    }
}

void RunLightningSolveTimed(AppState& app, mesh3d::Mesh& cloth) {
    app.isRunning = false;
    app.hasStarted = false;
    app.isLightningSolving = true;
    app.lightningStepsRun = 0;
    app.lastLightningBatchMs = 0.0f;
    app.lastMeshBuildMs = 0.0f;
    app.msg = TextFormat("Lightning solving: 0/%d steps", app.lightningMaxSteps);
    ResetSimAccumulator();
}

void RebuildMeshTimed(AppState& app, mesh3d::Mesh& cloth) {
    double start = GetTime();
    cloth = mesh3d::Mesh(app.currConfig, app.ptFileName);
    app.lastMeshBuildMs = static_cast<float>((GetTime() - start) * 1000.0);
    ResetSimAccumulator();
}

void HandleKeyboardShortcuts(AppState& app, mesh3d::Mesh& cloth) {
    if (app.isLightningSolving) {
        return;
    }

    if (IsKeyPressed(KEY_SPACE) || IsKeyPressed(KEY_ENTER)) {
        if (!app.isRunning) {
            app.hasStarted = true;
            app.isRunning = true;
            app.msg = "Simulation resumed";
        } else {
            app.isRunning = false;
            ResetSimAccumulator();
            app.msg = "Simulation paused";
        }
    }

    if (IsKeyPressed(KEY_R)) {
        RebuildMeshTimed(app, cloth);
        app.msg = "Restarted!";
        app.isRunning = false;
        app.hasStarted = false;
    }

    if (IsKeyPressed(KEY_L) && !app.isRunning && !app.isLightningSolving) {
        RunLightningSolveTimed(app, cloth);
    }

    if (!app.hasStarted) {
        if (IsKeyPressed(KEY_UP) && app.animationSpeed < static_cast<float>(maxStepsPerFrame)) {
            app.animationSpeed += 0.5f;
        }
        if (IsKeyPressed(KEY_DOWN) && app.animationSpeed > 0.5f) {
            app.animationSpeed -= 0.5f;
        }

        if (IsKeyPressed(KEY_M)) {
            app.currConfig.stiffness += 1.0f;
        }
        if (IsKeyPressed(KEY_N) && app.currConfig.stiffness > 1.0f) {
            app.currConfig.stiffness -= 1.0f;
        }

        if (IsKeyPressed(KEY_P)) {
            app.currConfig.dampingFactor += 0.1f;
        }
        if (IsKeyPressed(KEY_O) && app.currConfig.dampingFactor >= 0.1f) {
            app.currConfig.dampingFactor -= 0.1f;
        }

        if (IsKeyPressed(KEY_K)) {
            app.currConfig.airResistanceFactor += 0.001f;
        }
        if (IsKeyPressed(KEY_J) && app.currConfig.airResistanceFactor >= 0.001f) {
            app.currConfig.airResistanceFactor -= 0.001f;
        }

        if (IsKeyPressed(KEY_G)) {
            app.currConfig.gravity += 0.001f;
        }
        if (IsKeyPressed(KEY_F)) {
            app.currConfig.gravity -= 0.001f;
        }
    }

    if (IsWindowResized() || !IsWindowFocused()) {
        app.isRunning = false;
        ResetSimAccumulator();
    }
}

void UpdateSimulation(AppState& app, mesh3d::Mesh& cloth) {
    cloth.ApplyRuntimeConfig(app.currConfig);

    if (app.isLightningSolving) {
        const int remainingSteps = app.lightningMaxSteps - app.lightningStepsRun;
        if (remainingSteps <= 0) {
            app.isLightningSolving = false;
            app.msg = TextFormat("Lightning reached max steps: %d", app.lightningStepsRun);
            ResetSimAccumulator();
            return;
        }

        const int batchSteps = std::min(app.lightningBatchSteps, remainingSteps);
        int stepsRun = 0;
        const double start = GetTime();
        const bool ok = cloth.RunLightningRelaxation(app.currConfig, batchSteps, fixedDt, stepsRun);
        const float batchMs = static_cast<float>((GetTime() - start) * 1000.0);
        app.lastLightningBatchMs = batchMs;
        app.lastMeshBuildMs += batchMs;

        if (!ok) {
            app.isLightningSolving = false;
            app.msg = "Lightning solve failed or CUDA unavailable";
            ResetSimAccumulator();
            return;
        }

        app.lightningStepsRun += stepsRun;
        app.msg = TextFormat("Lightning solving: %d/%d steps", app.lightningStepsRun, app.lightningMaxSteps);

        if (stepsRun < batchSteps) {
            app.isLightningSolving = false;
            app.msg = TextFormat("Lightning converged: %d/%d steps", app.lightningStepsRun, app.lightningMaxSteps);
        } else if (app.lightningStepsRun >= app.lightningMaxSteps) {
            app.isLightningSolving = false;
            app.msg = TextFormat("Lightning reached max steps: %d", app.lightningStepsRun);
        }

        ResetSimAccumulator();
        return;
    }

    if (!app.isRunning) {
        ResetSimAccumulator();
        return;
    }

    simAccumulator += GetFrameTime() * app.animationSpeed;

    int steps = 0;
    while (simAccumulator >= fixedDt && steps < maxStepsPerFrame) {
        if (!cloth.Update(fixedDt)) {
            app.isRunning = false;
            ResetSimAccumulator();
            app.msg = "Simulation failed!";
            return;
        }

        simAccumulator -= fixedDt;
        steps++;
    }

    if (steps == maxStepsPerFrame) {
        ResetSimAccumulator();
    }
}
