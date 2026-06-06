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

    mesh3d::Mesh cloth(app.currConfig, app.ptFileName);
    Rectangle panelRec = { (float)APP_PANEL_X, 0, (float)APP_PANEL_WIDTH, (float)APP_SCREEN_HEIGHT };

    while (!WindowShouldClose()) {
        HandleWebUploads(app, cloth);
        HandleKeyboardShortcuts(app, cloth);
        UpdateCameraControls(camera, panelRec, GetFrameTime());
        UpdateSimulation(app, cloth);

        BeginDrawing();
        ClearBackground(RAYWHITE);

        BeginMode3D(camera);
        DrawCoordSystem();
        cloth.Draw();
        EndMode3D();

        DrawAxisLabels(camera);
        DrawAppUi(app, cloth);

        EndDrawing();
    }

    CloseWindow();
    return 0;
}
