#pragma once

#include "Mesh.h"

#include <string>

constexpr float ANIMATION_SPEED_STEP = 0.05f;
constexpr int APP_SCREEN_WIDTH = 1024;
constexpr int APP_SCREEN_HEIGHT = 768;
constexpr int APP_PANEL_WIDTH = 260;
constexpr int APP_PANEL_X = APP_SCREEN_WIDTH - APP_PANEL_WIDTH;

struct AppState {
    mesh3d::Config currConfig;
    float animationSpeed = 1.0f;
    std::string msg = "Press r to restart simulation";

    bool isRunning = false;
    bool hasStarted = false;
    bool showSaveDialog = false;
    bool showPointCloudSaveDialog = false;

    char saveFilename[256] = "config.txt";
    char pointCloudSaveFilename[256] = "point_cloud.msh";
    char ptFileName[256] = "example_cloud.msh";
    char configFileName[256] = "default_config.txt";
    float lastMeshBuildMs = 0.0f;
    float updateMs = 0.0f;
    float drawMs = 0.0f;
    float nextStatsRefreshTime = 0.0f;
    float displayedUpdateMs = 0.0f;
    float displayedDrawMs = 0.0f;
};
