#define RAYGUI_IMPLEMENTATION
#include "raygui.h"
#include "Mesh.h"
#include <string>
#include <iostream>
#include <ctime>
#include <cstring>

#ifdef __EMSCRIPTEN__
#include <emscripten.h>
#endif

#include "FileDialog.h"


const float ANIMATION_SPEED_STEP = 0.05f;
const int GRID_SIZE = 31;


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
void DrawAxisLabels(const Camera& camera);

void SetDefaultSaveFilename(char* buffer, size_t size) {
	time_t now = time(NULL);
	struct tm* timeinfo = localtime(&now);
	strftime(buffer, size, "config_%Y%m%d_%H%M%S.txt", timeinfo);
}

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
	const int panelWidth = 260;
	const int panelX = screenWidth - panelWidth;

	bool isRunning = false;
	bool hasStarted = false;
	bool showSaveDialog = false;
	char saveFilename[256] = "config.txt";

	char cloudFilename[256] = "";
	if (loadedConfig.pointCloudFile.length() < sizeof(cloudFilename)) {
		std::strcpy(cloudFilename, loadedConfig.pointCloudFile.c_str());
	}

    InitWindow(screenWidth, screenHeight, "3D Cloth Simulation");

    Camera camera = { { 15.0f, 15.0f, 15.0f }, { 0.0f, -2.5f, 0.0f }, { 0.0f, 1.0f, 0.0f }, 60.0f, CAMERA_PERSPECTIVE };

	mesh3d::Mesh cloth = mesh3d::Mesh(loadedConfig);

    while (!WindowShouldClose()) {
#ifdef __EMSCRIPTEN__
        // Check if a point cloud file was uploaded from the web UI
        bool cloudReady = EM_ASM_INT({ return Module.cloudFileReady ? 1 : 0; });
        if (cloudReady) {
            loadedConfig.pointCloudFile = "/user_cloud.msh";
            cloth = mesh3d::Mesh(loadedConfig);
            msg = "Cloud loaded from web!";
            isRunning = false;
            hasStarted = false;
            EM_ASM({ Module.cloudFileReady = false; });
        }
#endif

        if (IsKeyPressed(KEY_SPACE) || IsKeyPressed(KEY_ENTER)) {
			if (!isRunning) { hasStarted = true; isRunning = true; }
			else { isRunning = false; }
		};

		// Restart simulation with 'r'
		if (IsKeyPressed(KEY_R)) {
			cloth = mesh3d::Mesh(loadedConfig);
			msg = "Restarted!";
			isRunning = false;
			hasStarted = false;
		}

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

		// Draw axis labels (in screen space, after EndMode3D)
		DrawAxisLabels(camera);

		// ---- GUI Control Panel ----
		float cx = panelX + 16;
		float cw = panelWidth - 32;
		float cy = 12;
		float ch = 24;
		float gap = 32;

		GuiGroupBox({ (float)(panelX + 8), 8, (float)(panelWidth - 16), (float)(screenHeight - 16) }, "Control Panel");

		// Status label
		const char* statusText = isRunning ? "Status: Running" : (hasStarted ? "Status: Paused (Locked)" : "Status: Ready");
		GuiLabel({ cx, cy, cw, 18 }, statusText);
		cy += 24;

		// Play / Pause Toggle
		bool prevRunning = isRunning;
		GuiToggle({ cx, cy, cw, ch }, isRunning ? "Pause" : "Play", &isRunning);
		if (isRunning != prevRunning) {
			if (isRunning) { hasStarted = true; }
			msg = isRunning ? "Running!" : "Paused!";
		}
		cy += gap;

		// Restart Button
		if (hasStarted){
			auto pressResult = GuiButton({ cx, cy, cw, ch }, "Restart");
			if (pressResult) {
				cloth = mesh3d::Mesh(loadedConfig);
				msg = "Reseted!";
				isRunning = false;
				hasStarted = false;
			}
		} else {
			// disabled restart button (drawn but not interactive)
			int prevState = GuiGetState();
			GuiSetState(STATE_DISABLED);
			GuiButton({ cx, cy, cw, ch }, "Restart");
			GuiSetState(prevState);
		}

		cy += gap;

		// Save Config Button
		if (GuiButton({ cx, cy, cw, ch }, "Save Config")) {
			SetDefaultSaveFilename(saveFilename, sizeof(saveFilename));
			showSaveDialog = true;
		}
		cy += gap + 8;

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

		// Particle Mass Slider
		GuiSlider({ cx, cy, cw, ch }, "Mass", TextFormat("%.2f", loadedConfig.particleMass), &loadedConfig.particleMass, 0.1f, 10.0f);
		cy += gap;

		GuiSetStyle(SLIDER, BASE_COLOR_PRESSED, prevThumbColor);
		
		// ---- Point Cloud Controls ----
		GuiLine({ cx, cy, cw, 1 }, NULL);
		cy += 8;
		GuiLabel({ cx, cy, cw, 18 }, "Point Cloud");
		cy += 20;

		// Display current cloud file name
		const char* cloudDisplay = loadedConfig.pointCloudFile.empty() ? "(default grid)" : loadedConfig.pointCloudFile.c_str();
		GuiLabel({ cx, cy, cw, ch }, TextFormat("Cloud: %s", cloudDisplay));
		cy += gap;

		// Select File button: opens a dialog (Win32) or triggers HTML upload (Web)
		if (GuiButton({ cx, cy, cw, ch }, "Select File")) {
#ifdef __EMSCRIPTEN__
			EM_ASM({ document.getElementById('cloudFileInput').click(); });
#elif defined(_WIN32)
			if (OpenFileDialog(cloudFilename, sizeof(cloudFilename))) {
				loadedConfig.pointCloudFile = cloudFilename;
				std::strcpy(cloudFilename, loadedConfig.pointCloudFile.c_str());
				cloth = mesh3d::Mesh(loadedConfig);
				msg = "Cloud loaded!";
				isRunning = false;
				hasStarted = false;
			}
#else
			// Linux/Mac fallback: load from the stored path
			loadedConfig.pointCloudFile = cloudFilename;
			cloth = mesh3d::Mesh(loadedConfig);
			msg = "Cloud loaded!";
			isRunning = false;
			hasStarted = false;
#endif
		}
		cy += gap;

		// Spring generation seed
		float seedFloat = static_cast<float>(loadedConfig.springSeed);
		GuiSlider({ cx, cy, cw, ch }, "Seed", TextFormat("%.0f", seedFloat), &seedFloat, 0.0f, 999.0f);
		loadedConfig.springSeed = static_cast<unsigned int>(seedFloat);
		cy += gap;

		// Max spring distance
		GuiSlider({ cx, cy, cw, ch }, "Max Dist", TextFormat("%.2f", loadedConfig.maxSpringDist), &loadedConfig.maxSpringDist, 0.1f, 5.0f);
		cy += gap;

		// Max springs per particle
		float maxSpringFloat = static_cast<float>(loadedConfig.maxSpringsPerParticle);
		GuiSlider({ cx, cy, cw, ch }, "Max Conn", TextFormat("%.0f", maxSpringFloat), &maxSpringFloat, 1.0f, 12.0f);
		loadedConfig.maxSpringsPerParticle = static_cast<int>(maxSpringFloat);
		cy += gap;

		// Connection probability
		GuiSlider({ cx, cy, cw, ch }, "Conn Prob", TextFormat("%.2f", loadedConfig.springConnectProb), &loadedConfig.springConnectProb, 0.0f, 1.0f);
		cy += gap;

		GuiUnlock();
		#pragma endregion

		// FPS display
		GuiLabel({ cx, cy, cw, 20 }, TextFormat("FPS: %.0f", GetFPS()));

        // Text overlay in 3D area
		DrawText(msg.c_str(), 20, 20*2, 20, BLACK);
		DrawText("Keep pressing right-mouse and drag to rotate camera", 20, 20 * 4, 20, BLACK);

		// Save Config Dialog
		if (showSaveDialog) {
			DrawRectangle(0, 0, screenWidth, screenHeight, Fade(BLACK, 0.5f));
			Rectangle dialogRec = { (float)(screenWidth / 2 - 200), (float)(screenHeight / 2 - 100), 400, 200 };
			int result = GuiTextInputBox(dialogRec, "Save Configuration", "Enter filename:", "OK;Cancel", saveFilename, 256, NULL);
			if (result == 1) {
				mesh3d::WriteConfig(saveFilename, loadedConfig);
				msg = "Config saved!";
				showSaveDialog = false;
#ifdef __EMSCRIPTEN__
				EM_ASM({
					var filename = UTF8ToString($0);
					var data = FS.readFile(filename);
					var blob = new Blob([data.buffer], {type: "text/plain"});
					var url = URL.createObjectURL(blob);
					var a = document.createElement("a");
					a.href = url;
					a.download = filename;
					document.body.appendChild(a);
					a.click();
					document.body.removeChild(a);
					URL.revokeObjectURL(url);
				}, saveFilename);
#endif
			} else if (result >= 0) {
				showSaveDialog = false;
			}
		}

		EndDrawing();
		#pragma endregion
    }

    CloseWindow();
    return 0;
}

static const float AXIS_LENGTH = 10.0f;

void DrawCoordSystem() {
	DrawGrid(10, 1.0f);

	DrawLine3D({ 0, 0, 0 }, { AXIS_LENGTH, 0, 0 }, MAROON); // x
    DrawLine3D({ 0, 0, 0 }, { -AXIS_LENGTH, 0, 0 }, LIGHTGRAY); // -x

	DrawLine3D({ 0, 0, 0 }, { 0, AXIS_LENGTH, 0 }, DARKGREEN); // y
	DrawLine3D({ 0, 0, 0 }, { 0, -AXIS_LENGTH, 0 }, LIGHTGRAY); // -y

	DrawLine3D({ 0, 0, 0 }, { 0, 0, AXIS_LENGTH }, DARKBLUE); // z
	DrawLine3D({ 0, 0, 0 }, { 0, 0, -AXIS_LENGTH }, LIGHTGRAY); // -z
}

void DrawAxisLabels(const Camera& camera) {
	Vector2 sx = GetWorldToScreen({ AXIS_LENGTH, 0, 0 }, camera);
	Vector2 sy = GetWorldToScreen({ 0, AXIS_LENGTH, 0 }, camera);
	Vector2 sz = GetWorldToScreen({ 0, 0, AXIS_LENGTH }, camera);

	DrawText("X", (int)(sx.x + 4), (int)(sx.y - 10), 20, MAROON);
	DrawText("Y", (int)(sy.x + 4), (int)(sy.y - 10), 20, DARKGREEN);
	DrawText("Z", (int)(sz.x + 4), (int)(sz.y - 10), 20, DARKBLUE);
}
