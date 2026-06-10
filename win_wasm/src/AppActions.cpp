#include "AppActions.h"

#include "raylib.h"

#include <cstring>

#ifdef __EMSCRIPTEN__
#include <emscripten.h>
#endif

void HandleWebUploads(AppState& app, mesh3d::Mesh& cloth) {
#ifdef __EMSCRIPTEN__
    bool configCloudReady = EM_ASM_INT({ return Module.configCloudFileReady ? 1 : 0; });
    if (configCloudReady) {
        std::strcpy(app.configFileName, "/user_config.txt");
        app.currConfig = mesh3d::LoadMeshConfig(app.configFileName);
        cloth = mesh3d::Mesh(app.currConfig, app.ptFileName);
        app.msg = "Config loaded from web!";
        app.isRunning = false;
        app.hasStarted = false;
        EM_ASM({ Module.configCloudFileReady = false; });
    }

    bool cloudReady = EM_ASM_INT({ return Module.cloudFileReady ? 1 : 0; });
    if (cloudReady) {
        std::strcpy(app.ptFileName, "/user_cloud.msh");
        cloth = mesh3d::Mesh(app.currConfig, app.ptFileName);
        app.msg = "Cloud loaded from web!";
        app.isRunning = false;
        app.hasStarted = false;
        EM_ASM({ Module.cloudFileReady = false; });
    }
#else
    (void)app;
    (void)cloth;
#endif
}

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
