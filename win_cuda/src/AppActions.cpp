#include "AppActions.h"

#include "raylib.h"

void HandleKeyboardShortcuts(AppState& app, mesh3d::Mesh& cloth) {
    if (IsKeyPressed(KEY_SPACE) || IsKeyPressed(KEY_ENTER)) {
        if (!app.isRunning) {
            app.hasStarted = true;
            app.isRunning = true;
        } else {
            app.isRunning = false;
        }
    }

    if (IsKeyPressed(KEY_R)) {
        cloth = mesh3d::Mesh(app.currConfig, app.ptFileName);
        app.msg = "Restarted!";
        app.isRunning = false;
        app.hasStarted = false;
    }

    if (!app.hasStarted) {
        if (IsKeyPressed(KEY_UP)) {
            app.animationSpeed += ANIMATION_SPEED_STEP;
        }
        if (IsKeyPressed(KEY_DOWN) && app.animationSpeed > ANIMATION_SPEED_STEP) {
            app.animationSpeed -= ANIMATION_SPEED_STEP;
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
            app.currConfig.gravity += 0.5f;
        }
        if (IsKeyPressed(KEY_F)) {
            app.currConfig.gravity -= 0.5f;
        }
    }

    if (IsWindowResized() || !IsWindowFocused()) {
        app.isRunning = false;
    }
}

void UpdateSimulation(AppState& app, mesh3d::Mesh& cloth) {
    if (!app.isRunning) {
        return;
    }

    float dt = GetFrameTime() * app.animationSpeed;
    if (!cloth.Update(dt)) {
        app.isRunning = false;
        app.msg = "Simulation failed!";
    }
}
