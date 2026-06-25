#include "AppUi.h"

#include "AppActions.h"
#include "FileDialog.h"
#include "raygui.h"

#include <cstring>
#include <ctime>
#include <fstream>

namespace {
void SetDefaultSaveFilename(char* buffer, size_t size) {
    time_t now = time(nullptr);
    struct tm* timeinfo = localtime(&now);
    strftime(buffer, size, "config_%Y%m%d_%H%M%S.txt", timeinfo);
}

void SetDefaultPointCloudFilename(char* buffer, size_t size) {
    time_t now = time(nullptr);
    struct tm* timeinfo = localtime(&now);
    strftime(buffer, size, "point_cloud_%Y%m%d_%H%M%S.msh", timeinfo);
}

const char* GetDisplayFileName(const char* path) {
    if (path == nullptr || path[0] == '\0') {
        return "Missing";
    }

    const char* slash = strrchr(path, '/');
    const char* backslash = strrchr(path, '\\');
    const char* separator = slash > backslash ? slash : backslash;
    return separator != nullptr ? separator + 1 : path;
}

int Mesh3dBtn(Rectangle pos, const char* label, bool active = true) {
    if (active) {
        return GuiButton(pos, label);
    }

    GuiLock();
    auto prevState = GuiGetState();
    GuiSetState(STATE_DISABLED);
    int result = GuiButton(pos, label);
    GuiSetState(prevState);
    GuiUnlock();
    return result;
}

int Mesh3dSlider(
    Rectangle pos,
    const char* textLeft,
    const char* textRight,
    float* value,
    float minValue,
    float maxValue,
    bool active = true
) {
    if (active) {
        return GuiSlider(pos, textLeft, textRight, value, minValue, maxValue);
    }

    GuiLock();
    auto prevSliderColor = GuiGetStyle(SLIDER, BASE_COLOR_PRESSED);
    GuiSetStyle(SLIDER, BASE_COLOR_PRESSED, ColorToInt(GRAY));
    int result = GuiSlider(pos, textLeft, textRight, value, minValue, maxValue);
    GuiSetStyle(SLIDER, BASE_COLOR_PRESSED, prevSliderColor);
    GuiUnlock();
    return result;
}

void DrawControlPanel(AppState& app, mesh3d::Mesh& cloth) {
    float cx = APP_PANEL_X + 16;
    float cw = APP_PANEL_WIDTH - 32;
    float cy = 12;
    float ch = 24;
    float gap = 32;

    GuiGroupBox({ (float)(APP_PANEL_X + 8), 8, (float)(APP_PANEL_WIDTH - 16), (float)(APP_SCREEN_HEIGHT - 16) }, "Control Panel");

    const char* statusText = app.isRunning ? "Status: Running" : (app.hasStarted ? "Status: Paused (Locked)" : "Status: Ready");
    GuiLabel({ cx, cy, cw, 18 }, statusText);
    cy += 24;
    GuiLabel({ cx, cy, cw, 18 }, TextFormat("Msg: %s", app.msg.c_str()));
    cy += 24;

    if (!app.hasStarted) {
        if (Mesh3dBtn({ cx, cy, cw, ch }, "Start Simulation")) {
            app.hasStarted = true;
            app.isRunning = true;
            app.msg = "Simulation started";
        }

        cy += gap;
        Mesh3dBtn({ cx, cy, cw, ch }, "Reset Simulation", false);
    } else if (app.isRunning) {
        if (Mesh3dBtn({ cx, cy, cw, ch }, "Pause Simulation")) {
            app.isRunning = false;
            app.msg = "Simulation paused";
        }

        cy += gap;

        if (Mesh3dBtn({ cx, cy, cw, ch }, "Reset Simulation")) {
            RebuildMeshTimed(app, cloth);
            app.msg = "Restarted!";
            app.isRunning = false;
            app.hasStarted = false;
        }
    } else {
        if (Mesh3dBtn({ cx, cy, cw, ch }, "Resume Simulation")) {
            app.isRunning = true;
            app.msg = "Simulation resumed";
        }

        cy += gap;

        if (Mesh3dBtn({ cx, cy, cw, ch }, "Reset Simulation")) {
            RebuildMeshTimed(app, cloth);
            app.msg = "Restarted!";
            app.isRunning = false;
            app.hasStarted = false;
        }
    }

    cy += gap;

    GuiLabel({ cx, cy, cw, 18 }, TextFormat("Config File: %s", app.configFileName));
    cy += gap;

    if (Mesh3dBtn({ cx, cy, cw / 2, ch }, "Load Config", !app.hasStarted)) {
#if defined(_WIN32)
        if (OpenFileDialog(app.configFileName, sizeof(app.configFileName))) {
            app.currConfig = mesh3d::LoadMeshConfig(app.configFileName);
            RebuildMeshTimed(app, cloth);
            app.msg = "Config loaded!";
            app.isRunning = false;
            app.hasStarted = false;
        }
#endif
    }

    if (Mesh3dBtn({ cx + cw / 2 + 8, cy, cw / 2 - 8, ch }, "Save Config", !app.hasStarted)) {
        SetDefaultSaveFilename(app.saveFilename, sizeof(app.saveFilename));
        app.showSaveDialog = true;
    }
    cy += gap + 8;

    constexpr int maxStepsPerFrame = 20;
    Mesh3dSlider({ cx, cy, cw, ch }, "Anim Speed", TextFormat("%.1f", app.animationSpeed), &app.animationSpeed, 0.5f, static_cast<float>(maxStepsPerFrame), !app.hasStarted);
    cy += gap;

    Mesh3dSlider({ cx, cy, cw, ch }, "Stiffness", TextFormat("%.1f", app.currConfig.stiffness), &app.currConfig.stiffness, 1.0f, 50.0f, !app.hasStarted);
    cy += gap;

    Mesh3dSlider({ cx, cy, cw, ch }, "Damping", TextFormat("%.2f", app.currConfig.dampingFactor), &app.currConfig.dampingFactor, 0.0f, 5.0f, !app.hasStarted);
    cy += gap;

    Mesh3dSlider({ cx, cy, cw, ch }, "Air Resist", TextFormat("%.3f", app.currConfig.airResistanceFactor), &app.currConfig.airResistanceFactor, 0.0f, 0.1f, !app.hasStarted);
    cy += gap;

    Mesh3dSlider({ cx, cy, cw, ch }, "Gravity", TextFormat("%.2f", app.currConfig.gravity), &app.currConfig.gravity, -20.0f, 20.0f, !app.hasStarted);
    cy += gap;

    Mesh3dSlider({ cx, cy, cw, ch }, "Mass", TextFormat("%.2f", app.currConfig.particleMass), &app.currConfig.particleMass, 0.1f, 10.0f, !app.hasStarted);
    cy += gap;

    GuiLine({ cx, cy, cw, 1 }, nullptr);
    cy += 8;
    GuiLabel({ cx, cy, cw, 18 }, "Point Cloud");
    cy += 20;

    GuiLabel({ cx, cy, cw, ch }, TextFormat("File Name: %s", GetDisplayFileName(app.ptFileName)));
    cy += gap;

    if (Mesh3dBtn({ cx, cy, cw / 2, ch }, "Load Pt Cloud", !app.hasStarted)) {
#if defined(_WIN32)
        if (OpenFileDialog(app.ptFileName, sizeof(app.ptFileName))) {
            RebuildMeshTimed(app, cloth);
            app.msg = "Cloud loaded!";
            app.isRunning = false;
            app.hasStarted = false;
        }
#else
        RebuildMeshTimed(app, cloth);
        app.msg = "Cloud loaded!";
        app.isRunning = false;
        app.hasStarted = false;
#endif
    }

    if (Mesh3dBtn({ cx + cw / 2 + 8, cy, cw / 2 - 8, ch }, "Regen Springs", !app.isRunning && !app.hasStarted)) {
        RebuildMeshTimed(app, cloth);
        app.msg = "Mesh regenerated!";
    }

    cy += gap;

    if (Mesh3dBtn({ cx, cy, cw, ch }, "Export Pt Cloud", !app.isRunning)) {
        SetDefaultPointCloudFilename(app.pointCloudSaveFilename, sizeof(app.pointCloudSaveFilename));
        app.showPointCloudSaveDialog = true;
        app.isRunning = false;
        app.msg = "Enter point cloud export filename";
    }

    cy += gap;

    float seedFloat = static_cast<float>(app.currConfig.springSeed);
    Mesh3dSlider({ cx, cy, cw, ch }, "Seed", TextFormat("%.0f", seedFloat), &seedFloat, 0.0f, 999.0f, !app.hasStarted);
    app.currConfig.springSeed = static_cast<unsigned int>(seedFloat);
    cy += gap;

    Mesh3dSlider({ cx, cy, cw, ch }, "Max Dist", TextFormat("%.2f", app.currConfig.maxSpringDist), &app.currConfig.maxSpringDist, 0.1f, 5.0f, !app.hasStarted);
    cy += gap;

    float maxSpringFloat = static_cast<float>(app.currConfig.maxSpringsPerParticle);
    Mesh3dSlider({ cx, cy, cw, ch }, "Max Conn", TextFormat("%.0f", maxSpringFloat), &maxSpringFloat, 1.0f, 12.0f, !app.hasStarted);
    app.currConfig.maxSpringsPerParticle = static_cast<int>(maxSpringFloat);
    cy += gap;

    Mesh3dSlider({ cx, cy, cw, ch }, "Conn Prob", TextFormat("%.2f", app.currConfig.springConnectProb), &app.currConfig.springConnectProb, 0.0f, 1.0f, !app.hasStarted);
    cy += gap;

    GuiLabel({ cx, cy, cw, 20 }, TextFormat("FPS: %d", app.displayedFps));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Update: %.2f ms", app.displayedUpdateMs));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Draw: %.2f ms", app.displayedDrawMs));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Build: %.2f ms", app.lastMeshBuildMs));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Particles: %zu", cloth.ParticleCount()));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Springs: %zu", cloth.SpringCount()));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Len Mean: %.4f", app.displayedSpringStats.lengthMean));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Len Var: %.4f", app.displayedSpringStats.lengthVariance));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Stretch Mean: %.4f", app.displayedSpringStats.stretchMean));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Stretch Var: %.4f", app.displayedSpringStats.stretchVariance));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Force Mean: %.4f", app.displayedPtStats.forceValMean));
    cy += 20;
    GuiLabel({ cx, cy, cw, 20 }, TextFormat("Force Var: %.4f", app.displayedPtStats.forceValVar));
}

void DrawStatusOverlay(const AppState& app) {
    DrawText(app.msg.c_str(), 20, 20 * 2, 20, BLACK);
    DrawText("Keep pressing right-mouse to rotate camera.", 20, 20 * 4, 20, BLACK);
}

void DrawSaveDialogs(AppState& app, mesh3d::Mesh& cloth) {
    if (app.showSaveDialog) {
        DrawRectangle(0, 0, APP_SCREEN_WIDTH, APP_SCREEN_HEIGHT, Fade(BLACK, 0.5f));
        Rectangle dialogRec = { (float)(APP_SCREEN_WIDTH / 2 - 200), (float)(APP_SCREEN_HEIGHT / 2 - 100), 400, 200 };
        int result = GuiTextInputBox(dialogRec, "Save Configuration", "Enter filename:", "OK;Cancel", app.saveFilename, 256, nullptr);
        if (result == 1) {
            mesh3d::WriteConfig(app.saveFilename, app.currConfig);
            app.msg = "Config saved!";
            app.showSaveDialog = false;
        } else if (result >= 0) {
            app.showSaveDialog = false;
        }
    }

    if (app.showPointCloudSaveDialog) {
        DrawRectangle(0, 0, APP_SCREEN_WIDTH, APP_SCREEN_HEIGHT, Fade(BLACK, 0.5f));
        Rectangle dialogRec = { (float)(APP_SCREEN_WIDTH / 2 - 200), (float)(APP_SCREEN_HEIGHT / 2 - 100), 400, 200 };
        int result = GuiTextInputBox(dialogRec, "Export Point Cloud", "Enter filename:", "OK;Cancel", app.pointCloudSaveFilename, 256, nullptr);
        if (result == 1) {
            std::ofstream file(app.pointCloudSaveFilename);
            if (file.is_open()) {
                cloth.WritePointCloud(file);
                file.close();
                app.msg = "Point cloud saved!";
            } else {
                app.msg = "Point cloud save failed!";
            }
            app.showPointCloudSaveDialog = false;
        } else if (result >= 0) {
            app.showPointCloudSaveDialog = false;
        }
    }
}
}

void DrawAppUi(AppState& app, mesh3d::Mesh& cloth) {
    DrawControlPanel(app, cloth);
    DrawStatusOverlay(app);
    DrawSaveDialogs(app, cloth);
}
