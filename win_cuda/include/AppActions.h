#pragma once

#include "AppState.h"
#include "Mesh.h"

void HandleKeyboardShortcuts(AppState& app, mesh3d::Mesh& cloth);
void UpdateSimulation(AppState& app, mesh3d::Mesh& cloth);
void RebuildMeshTimed(AppState& app, mesh3d::Mesh& cloth);
void RunLightningSolveTimed(AppState& app, mesh3d::Mesh& cloth);
