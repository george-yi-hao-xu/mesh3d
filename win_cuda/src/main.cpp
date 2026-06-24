#include "AppActions.h"
#include "AppState.h"
#include "AppUi.h"
#include "CameraControls.h"
#include "Mesh.h"
#include "SceneDraw.h"

#include "raylib.h"

int main() {
    AppState app;
    app.currConfig = mesh3d::LoadMeshConfig(app.configFileName);

    InitWindow(APP_SCREEN_WIDTH, APP_SCREEN_HEIGHT, "3D Cloth Simulation");

    Camera camera = {
        { 15.0f, 15.0f, 15.0f },
        { 0.0f, -2.5f, 0.0f },
        { 0.0f, 1.0f, 0.0f },
        60.0f,
        CAMERA_PERSPECTIVE
    };

    double initialBuildStart = GetTime();
    mesh3d::Mesh cloth(app.currConfig, app.ptFileName);
    app.lastMeshBuildMs = static_cast<float>((GetTime() - initialBuildStart) * 1000.0);
    Rectangle panelRec = { (float)APP_PANEL_X, 0, (float)APP_PANEL_WIDTH, (float)APP_SCREEN_HEIGHT };

    while (!WindowShouldClose()) {
        HandleKeyboardShortcuts(app, cloth);
        UpdateCameraControls(camera, panelRec, GetFrameTime());

        double updateStart = GetTime();
        UpdateSimulation(app, cloth);
        app.updateMs = static_cast<float>((GetTime() - updateStart) * 1000.0);

        BeginDrawing();
        ClearBackground(RAYWHITE);

        double drawStart = GetTime();
        BeginMode3D(camera);
        DrawCoordSystem();
        cloth.Draw();
        EndMode3D();
        app.drawMs = static_cast<float>((GetTime() - drawStart) * 1000.0);

        DrawAxisLabels(camera);
        double now = GetTime();
        if (now >= app.nextStatsRefreshTime) {
            app.displayedUpdateMs = app.updateMs;
            app.displayedDrawMs = app.drawMs;
            app.displayedSpringStats = cloth.ComputeSpringStats();
            app.displayedFps = GetFPS();
            app.nextStatsRefreshTime = now + 0.25;
        }
        DrawAppUi(app, cloth);

        EndDrawing();
    }

    CloseWindow();
    return 0;
}
