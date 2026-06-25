#pragma once

#include "Mesh.h"

#include <string>

constexpr int APP_SCREEN_WIDTH = 1280;
constexpr int APP_SCREEN_HEIGHT = 900;
constexpr int APP_PANEL_WIDTH = 320;
constexpr int APP_PANEL_X = APP_SCREEN_WIDTH - APP_PANEL_WIDTH;
constexpr float AUTO_PAUSE_FORCE_MEAN_THRESHOLD = 0.001f;

struct AppState {
    mesh3d::Config currConfig;
    float animationSpeed = 1.0f;
    float updateMs = 0.0f;
    float drawMs = 0.0f;
    float lastMeshBuildMs = 0.0f;
    float displayedUpdateMs = 0.0f;
    float displayedDrawMs = 0.0f;
    mesh3d::SpringStats displayedSpringStats;
    mesh3d::PtStats displayedPtStats;
    int displayedFps = 0;
    double nextStatsRefreshTime = 0.0;
    std::string msg = "Press r to restart simulation";

    bool isRunning = false;
    bool hasStarted = false;
    bool showSaveDialog = false;
    bool showPointCloudSaveDialog = false;

    char saveFilename[256] = "config.txt";
    char pointCloudSaveFilename[256] = "point_cloud.msh";
    char ptFileName[256] = "example_cloud.msh";
    char configFileName[256] = "default_config.txt";
};
