#include "SceneDraw.h"

namespace {
constexpr float AXIS_LENGTH = 10.0f;
}

void DrawCoordSystem() {
    DrawGrid(10, 1.0f);

    DrawLine3D({ 0, 0, 0 }, { AXIS_LENGTH, 0, 0 }, MAROON);
    DrawLine3D({ 0, 0, 0 }, { -AXIS_LENGTH, 0, 0 }, LIGHTGRAY);

    DrawLine3D({ 0, 0, 0 }, { 0, AXIS_LENGTH, 0 }, DARKGREEN);
    DrawLine3D({ 0, 0, 0 }, { 0, -AXIS_LENGTH, 0 }, LIGHTGRAY);

    DrawLine3D({ 0, 0, 0 }, { 0, 0, AXIS_LENGTH }, DARKBLUE);
    DrawLine3D({ 0, 0, 0 }, { 0, 0, -AXIS_LENGTH }, LIGHTGRAY);
}

void DrawAxisLabels(const Camera& camera) {
    Vector2 sx = GetWorldToScreen({ AXIS_LENGTH, 0, 0 }, camera);
    Vector2 sy = GetWorldToScreen({ 0, AXIS_LENGTH, 0 }, camera);
    Vector2 sz = GetWorldToScreen({ 0, 0, AXIS_LENGTH }, camera);

    DrawText("X", (int)(sx.x + 4), (int)(sx.y - 10), 20, MAROON);
    DrawText("Y", (int)(sy.x + 4), (int)(sy.y - 10), 20, DARKGREEN);
    DrawText("Z", (int)(sz.x + 4), (int)(sz.y - 10), 20, DARKBLUE);
}
