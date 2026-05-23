#include "CameraControls.h"

namespace {
constexpr float CAMERA_MOUSE_MOVE_SENSITIVITY = 0.003f;
constexpr float CAMERA_MOVE_SPEED = 5.4f;
constexpr float CAMERA_ROTATION_SPEED = 0.03f;
}

extern "C" {
void CameraYaw(Camera *camera, float angle, bool rotateAroundTarget);
void CameraPitch(Camera *camera, float angle, bool lockView, bool rotateAroundTarget, bool rotateUp);
void CameraRoll(Camera *camera, float angle);
void CameraMoveForward(Camera *camera, float distance, bool moveInWorldPlane);
void CameraMoveRight(Camera *camera, float distance, bool moveInWorldPlane);
void CameraMoveToTarget(Camera *camera, float delta);
}

void UpdateCameraControls(Camera& camera, Rectangle blockedArea, float dt) {
    if (CheckCollisionPointRec(GetMousePosition(), blockedArea)) {
        return;
    }

    float moveSpeed = CAMERA_MOVE_SPEED * dt;
    float rotSpeed = CAMERA_ROTATION_SPEED * dt;
    Vector2 mouseDelta = GetMouseDelta();

    CameraMoveToTarget(&camera, -GetMouseWheelMove());

    if (IsMouseButtonDown(MOUSE_BUTTON_RIGHT)) {
        CameraYaw(&camera, -mouseDelta.x * CAMERA_MOUSE_MOVE_SENSITIVITY, true);
        CameraPitch(&camera, -mouseDelta.y * CAMERA_MOUSE_MOVE_SENSITIVITY, true, true, false);
    }

    if (IsKeyDown(KEY_W)) CameraMoveForward(&camera, moveSpeed, true);
    if (IsKeyDown(KEY_S)) CameraMoveForward(&camera, -moveSpeed, true);
    if (IsKeyDown(KEY_A)) CameraMoveRight(&camera, -moveSpeed, true);
    if (IsKeyDown(KEY_D)) CameraMoveRight(&camera, moveSpeed, true);

    if (IsKeyDown(KEY_UP)) CameraPitch(&camera, rotSpeed, true, true, false);
    if (IsKeyDown(KEY_DOWN)) CameraPitch(&camera, -rotSpeed, true, true, false);
    if (IsKeyDown(KEY_LEFT)) CameraYaw(&camera, -rotSpeed, true);
    if (IsKeyDown(KEY_RIGHT)) CameraYaw(&camera, rotSpeed, true);

    if (IsKeyDown(KEY_Q)) CameraRoll(&camera, -rotSpeed);
    if (IsKeyDown(KEY_E)) CameraRoll(&camera, rotSpeed);
}
