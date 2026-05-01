#define RAYGUI_IMPLEMENTATION
#include "raygui.h"
#include "Mesh.h"
#include <string>
#include <iostream>

const float ANIMATION_SPEED_STEP = 0.05f;
const int GRID_SIZE = 31;
const float MASS = 1.0f;

// Camera control constants (matching raylib internals)
const float CAMERA_MOUSE_MOVE_SENSITIVITY = 0.003f;
const float CAMERA_MOVE_SPEED = 5.4f;
const float CAMERA_ROTATION_SPEED = 0.03f;

// Forward declarations for camera manipulation functions (from rcamera.h)
extern "C" {
void CameraYaw(Camera *camera, float angle, bool rotateAroundTarget);
void CameraPitch(Camera *camera, float angle, bool lockView, bool rotateAroundTarget, bool rotateUp);
void CameraRoll(Camera *camera, float angle);
void CameraMoveForward(Camera *camera, float distance, bool moveInWorldPlane);
void CameraMoveRight(Camera *camera, float distance, bool moveInWorldPlane);
void CameraMoveToTarget(Camera *camera, float delta);
}

void DrawCoordSystem(void);

std::string formateFloat(float f) {
	char buffer[20];
	std::sprintf(buffer, "%.3f", f);
	return std::string(buffer);
}

int main() {
	mesh3d::Config loadedConfig = mesh3d::LoadMeshConfig("config.txt");
	float animationSpeed = 1.0f;
	std::string msg = "Press r to restart simulation";

	const int screenWidth = 1024;
	const int screenHeight = 768;
	const int panelWidth = 240;
	const int panelX = screenWidth - panelWidth;

	bool isRunning = false;
	bool hasStarted = false;

    InitWindow(screenWidth, screenHeight, "3D Cloth Simulation");

    Camera camera = { { 15.0f, 15.0f, 15.0f }, { 0.0f, -2.5f, 0.0f }, { 0.0f, 1.0f, 0.0f }, 60.0f, CAMERA_PERSPECTIVE };

	mesh3d::Mesh cloth = mesh3d::Mesh(loadedConfig);

    while (!WindowShouldClose()) {
        if (IsKeyPressed(KEY_SPACE) || IsKeyPressed(KEY_ENTER)) {
			if (!isRunning) { hasStarted = true; isRunning = true; }
			else { isRunning = false; }
		};

		// Parameter adjustments only allowed before first play
		if (!hasStarted) {
			// increase or decrease animation speed
			if (IsKeyPressed(KEY_UP)) { animationSpeed += ANIMATION_SPEED_STEP; }
			if (IsKeyPressed(KEY_DOWN)) {
				if (animationSpeed > ANIMATION_SPEED_STEP) animationSpeed -= ANIMATION_SPEED_STEP;
			}

			// increase or decrease stiffness
			if (IsKeyPressed(KEY_M)) { loadedConfig.stiffness += 1.0f; }
			if (IsKeyPressed(KEY_N)) {
				if (loadedConfig.stiffness > 1.0f) loadedConfig.stiffness -= 1.0f;
			}
			
			// increase or decrease damping factor
			if (IsKeyPressed(KEY_P)) { loadedConfig.dampingFactor += 0.1f; }
			if (IsKeyPressed(KEY_O)) {
				if (loadedConfig.dampingFactor >= 0.1f) loadedConfig.dampingFactor -= 0.1f;
			}

			// increase or decrease air resistance factor
			if (IsKeyPressed(KEY_K)) { loadedConfig.airResistanceFactor += 0.001f; }
			if (IsKeyPressed(KEY_J)) {
				if (loadedConfig.airResistanceFactor >= 0.001f) loadedConfig.airResistanceFactor -= 0.001f;
			}
		}

		// if window moves, then stop simulation
		if (IsWindowResized() || !IsWindowFocused) { isRunning = false; }

		// save config to the file
		if (IsKeyPressed(KEY_S)) {
			mesh3d::WriteConfig("config.txt", loadedConfig);
			msg = "Config saved!";
		}

		// restart simulation
		if (IsKeyPressed(KEY_R)) { 
			cloth = mesh3d::Mesh(loadedConfig);
			msg = "Reseted!";
			isRunning = false;
			hasStarted = false;
		}

		// Mouse over panel detection
		Rectangle panelRec = { (float)panelX, 0, (float)panelWidth, (float)screenHeight };
		bool isMouseOverPanel = CheckCollisionPointRec(GetMousePosition(), panelRec);

		// Update camera only when mouse is not over the GUI panel
		if (!isMouseOverPanel) {
			float dt = GetFrameTime();
			float moveSpeed = CAMERA_MOVE_SPEED * dt;
			float rotSpeed = CAMERA_ROTATION_SPEED * dt;
			Vector2 mouseDelta = GetMouseDelta();

			// Mouse wheel zoom (always active)
			CameraMoveToTarget(&camera, -GetMouseWheelMove());

			// Right-click drag to rotate
			if (IsMouseButtonDown(MOUSE_BUTTON_RIGHT)) {
				CameraYaw(&camera, -mouseDelta.x * CAMERA_MOUSE_MOVE_SENSITIVITY, true);
				CameraPitch(&camera, -mouseDelta.y * CAMERA_MOUSE_MOVE_SENSITIVITY, true, true, false);
			}

			// Keyboard movement (WASD)
			if (IsKeyDown(KEY_W)) CameraMoveForward(&camera, moveSpeed, true);
			if (IsKeyDown(KEY_S)) CameraMoveForward(&camera, -moveSpeed, true);
			if (IsKeyDown(KEY_A)) CameraMoveRight(&camera, -moveSpeed, true);
			if (IsKeyDown(KEY_D)) CameraMoveRight(&camera, moveSpeed, true);

			// Keyboard rotation (arrow keys)
			if (IsKeyDown(KEY_UP)) CameraPitch(&camera, rotSpeed, true, true, false);
			if (IsKeyDown(KEY_DOWN)) CameraPitch(&camera, -rotSpeed, true, true, false);
			if (IsKeyDown(KEY_LEFT)) CameraYaw(&camera, -rotSpeed, true);
			if (IsKeyDown(KEY_RIGHT)) CameraYaw(&camera, rotSpeed, true);

			// Keyboard roll (Q/E)
			if (IsKeyDown(KEY_Q)) CameraRoll(&camera, -rotSpeed);
			if (IsKeyDown(KEY_E)) CameraRoll(&camera, rotSpeed);
		}

		if (isRunning) {
			float dt = GetFrameTime() * animationSpeed;
			if (!cloth.Update(dt)) {
				isRunning = false;
				msg = "Simulation failed!";
			}
		}

		#pragma region Draw
        BeginDrawing();

        ClearBackground(RAYWHITE);

        BeginMode3D(camera);
		DrawCoordSystem();
        cloth.Draw();
        EndMode3D();

		// ---- GUI Control Panel ----
		float cx = panelX + 20;
		float cw = panelWidth - 40;
		float cy = 20;
		float ch = 28;
		float gap = 38;

		GuiGroupBox({ (float)(panelX + 10), 10, (float)(panelWidth - 20), (float)(screenHeight - 20) }, "Control Panel");

		// Status label
		const char* statusText = isRunning ? "Status: Running" : (hasStarted ? "Status: Paused (Locked)" : "Status: Ready");
		GuiLabel({ cx, cy, cw, 20 }, statusText);
		cy += 28;

		// Play / Pause Toggle
		bool prevRunning = isRunning;
		GuiToggle({ cx, cy, cw, ch }, isRunning ? "Pause" : "Play", &isRunning);
		if (isRunning != prevRunning) {
			if (isRunning) { hasStarted = true; }
			msg = isRunning ? "Running!" : "Paused!";
		}
		cy += gap;

		// Restart Button
		if (GuiButton({ cx, cy, cw, ch }, "Restart (R)")) {
			cloth = mesh3d::Mesh(loadedConfig);
			msg = "Reseted!";
			isRunning = false;
			hasStarted = false;
		}
		cy += gap;

		// Save Config Button
		if (GuiButton({ cx, cy, cw, ch }, "Save Config (S) to config.txt file")) {
			mesh3d::WriteConfig("config.txt", loadedConfig);
			msg = "Config saved!";
		}
		cy += gap + 15;

		// Lock sliders after first play (drawn normally but not interactive)
		if (hasStarted) GuiLock();

		#pragma region Sliders (only interactive when paused)
		// Make thumb bright red when running for better visibility
		int prevThumbColor = GuiGetStyle(SLIDER, BASE_COLOR_PRESSED);
		if (hasStarted) GuiSetStyle(SLIDER, BASE_COLOR_PRESSED, ColorToInt(GRAY));

		// Animation Speed Slider
		GuiSlider({ cx, cy, cw, ch }, "Anim Speed", TextFormat("%.2f", animationSpeed), &animationSpeed, 0.05f, 3.0f);
		cy += gap;

		// Stiffness Slider
		GuiSlider({ cx, cy, cw, ch }, "Stiffness", TextFormat("%.1f", loadedConfig.stiffness), &loadedConfig.stiffness, 1.0f, 50.0f);
		cy += gap;

		// Damping Slider
		GuiSlider({ cx, cy, cw, ch }, "Damping", TextFormat("%.2f", loadedConfig.dampingFactor), &loadedConfig.dampingFactor, 0.0f, 5.0f);
		cy += gap;

		// Air Resistance Slider
		GuiSlider({ cx, cy, cw, ch }, "Air Resist", TextFormat("%.3f", loadedConfig.airResistanceFactor), &loadedConfig.airResistanceFactor, 0.0f, 0.1f);
		cy += gap;

		GuiSetStyle(SLIDER, BASE_COLOR_PRESSED, prevThumbColor);
		GuiUnlock();
		#pragma endregion
		
		// Mass display (read-only)
		GuiLabel({ cx, cy, cw, 20 }, TextFormat("Partial Mass: %.2f", MASS));
		cy += gap;

		// FPS display
		GuiLabel({ cx, cy, cw, 20 }, TextFormat("FPS: %.0f", GetFPS()));

        // Text overlay in 3D area
		// DrawText(isRunning ? "Running..." : "Paused (Press Enter/Space to continue)", 20, 20, 20, RED);
		DrawText(msg.c_str(), 20, 20*2, 20, BLACK);
		DrawText("Keep pressing right-mouse and drag to rotate camera", 20, 20 * 4, 20, BLACK);
		// DrawText("Keyboard shortcuts still work!", 20, 20 * 5, 20, DARKGRAY);

		EndDrawing();
		#pragma endregion
    }

    CloseWindow();
    return 0;
}

void DrawCoordSystem() {
    const float  AXIS_LENGTH = 10.0f;

	DrawGrid(10, 1.0f);

	DrawLine3D({ 0, 0, 0 }, { AXIS_LENGTH, 0, 0 }, MAROON); // x
    DrawLine3D({ 0, 0, 0 }, { -AXIS_LENGTH, 0, 0 }, LIGHTGRAY); // -x
	DrawLine3D({ 0, 0, 0 }, { 0, AXIS_LENGTH, 0 }, DARKGREEN); // y
	DrawLine3D({ 0, 0, 0 }, { 0, -AXIS_LENGTH, 0 }, LIGHTGRAY); // -y
	DrawLine3D({ 0, 0, 0 }, { 0, 0, AXIS_LENGTH }, DARKBLUE); // z
	DrawLine3D({ 0, 0, 0 }, { 0, 0, -AXIS_LENGTH }, LIGHTGRAY); // -z
}
