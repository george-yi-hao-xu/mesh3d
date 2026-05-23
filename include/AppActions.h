#pragma once

#include "AppState.h"
#include "Mesh.h"

void HandleWebUploads(AppState& app, mesh3d::Mesh& cloth);
void HandleKeyboardShortcuts(AppState& app, mesh3d::Mesh& cloth);
void UpdateSimulation(AppState& app, mesh3d::Mesh& cloth);
