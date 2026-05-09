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
int Mesh3dBtn(Rectangle pos, const char* label, bool active = true);
int Mesh3dSlider(Rectangle pos, const char* textLeft, const char* textRight, float* value, float minValue, float maxValue, bool active = true);

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
	mesh3d::Config currConfig = mesh3d::LoadMeshConfig("default_config.txt");

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

	char ptFileName[256] = "example_cloud.msh";
	char configFileName[256] = "default_config.txt";


    InitWindow(screenWidth, screenHeight, "3D Cloth Simulation");

    Camera camera = { { 15.0f, 15.0f, 15.0f }, { 0.0f, -2.5f, 0.0f }, { 0.0f, 1.0f, 0.0f }, 60.0f, CAMERA_PERSPECTIVE };

	mesh3d::Mesh cloth = mesh3d::Mesh(currConfig, ptFileName);

    while (!WindowShouldClose()) {
#ifdef __EMSCRIPTEN__
        // Check if a point cloud file was uploaded from the web UI
        bool cloudReady = EM_ASM_INT({ return Module.cloudFileReady ? 1 : 0; });
        if (cloudReady) {
            std::strcpy(ptFileName, "/user_cloud.msh");
            cloth = mesh3d::Mesh(currConfig, ptFileName);
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
			cloth = mesh3d::Mesh(currConfig, ptFileName);
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
			if (IsKeyPressed(KEY_M)) { currConfig.stiffness += 1.0f; }
			if (IsKeyPressed(KEY_N)) {
				if (currConfig.stiffness > 1.0f) currConfig.stiffness -= 1.0f;
			}
			
			// increase or decrease damping factor
			if (IsKeyPressed(KEY_P)) { currConfig.dampingFactor += 0.1f; }
			if (IsKeyPressed(KEY_O)) {
				if (currConfig.dampingFactor >= 0.1f) currConfig.dampingFactor -= 0.1f;
			}

			// increase or decrease air resistance factor
			if (IsKeyPressed(KEY_K)) { currConfig.airResistanceFactor += 0.001f; }
			if (IsKeyPressed(KEY_J)) {
				if (currConfig.airResistanceFactor >= 0.001f) currConfig.airResistanceFactor -= 0.001f;
			}

			// increase or decrease gravity
			if (IsKeyPressed(KEY_G)) { currConfig.gravity += 0.5f; }
			if (IsKeyPressed(KEY_F)) { currConfig.gravity -= 0.5f; }
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

		// draw 2 buttons based on hasStarted and isRunning states
		if (!hasStarted) {
			// draw an active START button and a disabled RESET button
			if (Mesh3dBtn({ cx, cy, cw, ch }, "Start Simulation")) {
				cloth = mesh3d::Mesh(currConfig, ptFileName); // re-create cloth to apply any config param changes
				hasStarted = true;
				isRunning = true;
			}

			cy += gap;

			auto pressedReset = Mesh3dBtn({ cx, cy, cw, ch }, "Reset Simulation", false);
		} else if (hasStarted && isRunning) {
			// ok, started and is running, show an active PAUSE button and an active RESET button
			if(Mesh3dBtn({ cx, cy, cw, ch }, "Pause Simulation")) {
				isRunning = false;
			}

			cy += gap;

			if(Mesh3dBtn({ cx, cy, cw, ch }, "Reset Simulation")) {
				cloth = mesh3d::Mesh(currConfig, ptFileName);
				msg = "Restarted!";
				isRunning = false;
				hasStarted = false;
			}
		} else if (hasStarted && !isRunning) {
			// ok, started but paused, show an active RESUME button and an active RESET button
			if(Mesh3dBtn({ cx, cy, cw, ch }, "Resume Simulation")) {
				isRunning = true;
			}

			cy += gap;

			if(Mesh3dBtn({ cx, cy, cw, ch }, "Reset Simulation")) {
				cloth = mesh3d::Mesh(currConfig, ptFileName);
				msg = "Restarted!";
				isRunning = false;
				hasStarted = false;
			}
		}

		cy += gap;

		// Load/Save Config
		GuiLabel({ cx, cy, cw, 18 }, TextFormat("Config File: %s", configFileName));
		
		cy += gap;

		if (Mesh3dBtn({ cx, cy, cw / 2, ch }, "Load Config", !hasStarted)) {
#ifdef __EMSCRIPTEN__
			EM_ASM({ document.getElementById('configFileInput').click(); });
#elif defined(_WIN32)
			if (OpenFileDialog(configFileName, sizeof(configFileName))) {
				// open a window to pick config file
				currConfig = mesh3d::LoadMeshConfig(configFileName);
				cloth = mesh3d::Mesh(currConfig, ptFileName);
				msg = "Config loaded!";
				isRunning = false;
				hasStarted = false;
			}
#else
#endif
		}

		if (Mesh3dBtn({ cx + cw / 2 + 8, cy, cw / 2 - 8, ch }, "Save Config", !hasStarted)) {
			SetDefaultSaveFilename(saveFilename, sizeof(saveFilename));
			showSaveDialog = true;
		}
		cy += gap + 8;

		// Lock sliders after first play (drawn normally but not interactive)
		if (hasStarted) GuiLock();

		#pragma region Cfg_Slider

		// Animation Speed Slider
		Mesh3dSlider({ cx, cy, cw, ch }, "Anim Speed", TextFormat("%.2f", animationSpeed), &animationSpeed, 0.05f, 3.0f, !hasStarted);
		cy += gap;

		// Stiffness Slider
		Mesh3dSlider({ cx, cy, cw, ch }, "Stiffness", TextFormat("%.1f", currConfig.stiffness), &currConfig.stiffness, 1.0f, 50.0f, !hasStarted);
		cy += gap;

		// Damping Slider
		Mesh3dSlider({ cx, cy, cw, ch }, "Damping", TextFormat("%.2f", currConfig.dampingFactor), &currConfig.dampingFactor, 0.0f, 5.0f, !hasStarted);
		cy += gap;

		// Air Resistance Slider
		Mesh3dSlider({ cx, cy, cw, ch }, "Air Resist", TextFormat("%.3f", currConfig.airResistanceFactor), &currConfig.airResistanceFactor, 0.0f, 0.1f, !hasStarted);
		cy += gap;

		// Gravity Slider
		Mesh3dSlider({ cx, cy, cw, ch }, "Gravity", TextFormat("%.2f", currConfig.gravity), &currConfig.gravity, -20.0f, 20.0f, !hasStarted);
		cy += gap;

		// Particle Mass Slider
		Mesh3dSlider({ cx, cy, cw, ch }, "Mass", TextFormat("%.2f", currConfig.particleMass), &currConfig.particleMass, 0.1f, 10.0f, !hasStarted);
		cy += gap;
		
		#pragma endregion

		#pragma region Pt_Cld
		// ---- Point Cloud Controls ----
		GuiLine({ cx, cy, cw, 1 }, NULL);
		cy += 8;
		GuiLabel({ cx, cy, cw, 18 }, "Point Cloud");
		cy += 20;

		// Display current cloud file name
		GuiLabel({ cx, cy, cw, ch }, TextFormat("File Name: %s", ptFileName[0] != '\0' ? ptFileName : "Missing"));
		cy += gap;

		// Select File button: opens a dialog (Win32) or triggers HTML upload (Web)
		if (Mesh3dBtn({ cx, cy, cw / 2, ch }, "Select Pts File", !hasStarted)) {
#ifdef __EMSCRIPTEN__
			EM_ASM({ document.getElementById('cloudFileInput').click(); });
#elif defined(_WIN32)
			if (OpenFileDialog(ptFileName, sizeof(ptFileName))) {
				// Keep the current in-memory config and rebuild using the newly selected point cloud.
				cloth = mesh3d::Mesh(currConfig, ptFileName);
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

		// regen-mesh btn
		if (Mesh3dBtn({ cx + cw / 2 + 8, cy, cw / 2 - 8, ch }, "Regen Springs", !isRunning && !hasStarted)) {
			cloth = mesh3d::Mesh(currConfig, ptFileName);
			msg = "Mesh regenerated!";
		}

		cy += gap;

		// Spring generation seed
		float seedFloat = static_cast<float>(currConfig.springSeed);
		Mesh3dSlider({ cx, cy, cw, ch }, "Seed", TextFormat("%.0f", seedFloat), &seedFloat, 0.0f, 999.0f, !hasStarted);
		currConfig.springSeed = static_cast<unsigned int>(seedFloat);
		cy += gap;

		// Max spring distance
		Mesh3dSlider({ cx, cy, cw, ch }, "Max Dist", TextFormat("%.2f", currConfig.maxSpringDist), &currConfig.maxSpringDist, 0.1f, 5.0f, !hasStarted);
		cy += gap;

		// Max springs per particle
		float maxSpringFloat = static_cast<float>(currConfig.maxSpringsPerParticle);
		Mesh3dSlider({ cx, cy, cw, ch }, "Max Conn", TextFormat("%.0f", maxSpringFloat), &maxSpringFloat, 1.0f, 12.0f, !hasStarted);
		currConfig.maxSpringsPerParticle = static_cast<int>(maxSpringFloat);
		cy += gap;

		// Connection probability
		Mesh3dSlider({ cx, cy, cw, ch }, "Conn Prob", TextFormat("%.2f", currConfig.springConnectProb), &currConfig.springConnectProb, 0.0f, 1.0f, !hasStarted);
		cy += gap;

		// FPS display
		GuiLabel({ cx, cy, cw, 20 }, TextFormat("FPS: %d", GetFPS()));

        // Text overlay in 3D area
		DrawText(msg.c_str(), 20, 20*2, 20, BLACK);
		DrawText("Keep pressing right-mouse to rotate camera.", 20, 20 * 4, 20, BLACK);
		#pragma endregion Pt_Cld

		// Save Config Dialog
		if (showSaveDialog) {
			DrawRectangle(0, 0, screenWidth, screenHeight, Fade(BLACK, 0.5f));
			Rectangle dialogRec = { (float)(screenWidth / 2 - 200), (float)(screenHeight / 2 - 100), 400, 200 };
			int result = GuiTextInputBox(dialogRec, "Save Configuration", "Enter filename:", "OK;Cancel", saveFilename, 256, NULL);
			if (result == 1) {
				mesh3d::WriteConfig(saveFilename, currConfig);
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

int Mesh3dBtn(Rectangle pos, const char* label, bool active) {
	if (active) {
		return GuiButton(pos, label);
	} else {
		GuiLock();
		auto prevState = GuiGetState();
		GuiSetState(STATE_DISABLED);
		int result = GuiButton(pos, label);
		GuiSetState(prevState);
		GuiUnlock();
		return result;
	}
}

int Mesh3dSlider(Rectangle pos, const char* textLeft, const char* textRight, float* value, float minValue, float maxValue, bool active) {
	if (active) {
		return GuiSlider(pos, textLeft, textRight, value, minValue, maxValue);
	} else {
		GuiLock();
		// auto prevState = GuiGetState();
		// GuiSetState(STATE_DISABLED);
		auto prevSliderColor = GuiGetStyle(SLIDER, BASE_COLOR_PRESSED);
		GuiSetStyle(SLIDER, BASE_COLOR_PRESSED, ColorToInt(GRAY));
		
		int result = GuiSlider(pos, textLeft, textRight, value, minValue, maxValue);
		
		// GuiSetState(prevState);
		GuiSetStyle(SLIDER, BASE_COLOR_PRESSED, prevSliderColor);
		GuiUnlock();
		return result;
	}
}
