#include "ParticleRenderer.h"

#include "raylib.h"
#include "raymath.h"

namespace mesh3d {
    namespace {
        constexpr float PARTICLE_RADIUS = 0.1f;
        constexpr int PARTICLE_RINGS = 4;
        constexpr int PARTICLE_SLICES = 4;

        // Vertex shader for particle instancing. Each instance reuses the same
        // sphere mesh, while instanceTransform places it at one particle position.
        const char* PARTICLE_INSTANCING_VS = R"(
#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in mat4 instanceTransform;

uniform mat4 mvp;

out vec2 fragTexCoord;

void main()
{
    fragTexCoord = vertexTexCoord;
    gl_Position = mvp*instanceTransform*vec4(vertexPosition, 1.0);
}
)";

        // Fragment shader for particle instancing. The default white texture is
        // multiplied by colDiffuse, which we set to green or red per draw batch.
        const char* PARTICLE_INSTANCING_FS = R"(
#version 330
in vec2 fragTexCoord;

uniform sampler2D texture0;
uniform vec4 colDiffuse;

out vec4 finalColor;

void main()
{
    finalColor = texture(texture0, fragTexCoord)*colDiffuse;
}
)";

        struct ParticleInstancingRenderer {
            bool initialized = false;
            bool available = false;
            Model model = {};
            Material material = {};
        };

        ParticleInstancingRenderer& GetParticleInstancingRenderer() {
            static ParticleInstancingRenderer renderer;
            if (renderer.initialized) {
                return renderer;
            }

            renderer.initialized = true;
            renderer.model = LoadModelFromMesh(GenMeshSphere(PARTICLE_RADIUS, PARTICLE_RINGS, PARTICLE_SLICES));

            Shader shader = LoadShaderFromMemory(PARTICLE_INSTANCING_VS, PARTICLE_INSTANCING_FS);
            // DrawMeshInstanced needs these locations to bind mesh vertices,
            // camera MVP, material color, and the per-instance transform matrix.
            shader.locs[SHADER_LOC_VERTEX_POSITION] = GetShaderLocationAttrib(shader, "vertexPosition");
            shader.locs[SHADER_LOC_VERTEX_TEXCOORD01] = GetShaderLocationAttrib(shader, "vertexTexCoord");
            shader.locs[SHADER_LOC_MATRIX_MVP] = GetShaderLocation(shader, "mvp");
            shader.locs[SHADER_LOC_COLOR_DIFFUSE] = GetShaderLocation(shader, "colDiffuse");
            shader.locs[SHADER_LOC_MAP_DIFFUSE] = GetShaderLocation(shader, "texture0");
            shader.locs[SHADER_LOC_MATRIX_MODEL] = GetShaderLocationAttrib(shader, "instanceTransform");

            renderer.material = LoadMaterialDefault();
            renderer.material.shader = shader;
            // If any required shader binding is missing, keep the old safe
            // DrawSphereEx path so a shader issue cannot make particles vanish.
            renderer.available = renderer.model.meshCount > 0 &&
                shader.id > 0 &&
                shader.locs[SHADER_LOC_VERTEX_POSITION] >= 0 &&
                shader.locs[SHADER_LOC_MATRIX_MVP] >= 0 &&
                shader.locs[SHADER_LOC_COLOR_DIFFUSE] >= 0 &&
                shader.locs[SHADER_LOC_MATRIX_MODEL] >= 0;

            return renderer;
        }

        void DrawParticlesFallback(const std::vector<Particle>& particles) {
            for (const auto& particle : particles) {
                DrawSphereEx(particle.position, PARTICLE_RADIUS, PARTICLE_RINGS, PARTICLE_SLICES, particle.isFixed ? RED : GREEN);
            }
        }
    }

    void DrawParticlesInstancedOrFallback(const std::vector<Particle>& particles) {
        ParticleInstancingRenderer& renderer = GetParticleInstancingRenderer();
        if (!renderer.available) {
            DrawParticlesFallback(particles);
            return;
        }

        // Split fixed and free particles so each color can be drawn in one
        // instanced batch instead of one draw call per particle.
        std::vector<Matrix> fixedTransforms;
        std::vector<Matrix> freeTransforms;
        fixedTransforms.reserve(particles.size());
        freeTransforms.reserve(particles.size());

        for (const auto& particle : particles) {
            Matrix transform = MatrixTranslate(particle.position.x, particle.position.y, particle.position.z);
            if (particle.isFixed) {
                fixedTransforms.push_back(transform);
            } else {
                freeTransforms.push_back(transform);
            }
        }

        if (!freeTransforms.empty()) {
            renderer.material.maps[MATERIAL_MAP_DIFFUSE].color = GREEN;
            DrawMeshInstanced(renderer.model.meshes[0], renderer.material, freeTransforms.data(), static_cast<int>(freeTransforms.size()));
        }

        if (!fixedTransforms.empty()) {
            renderer.material.maps[MATERIAL_MAP_DIFFUSE].color = RED;
            DrawMeshInstanced(renderer.model.meshes[0], renderer.material, fixedTransforms.data(), static_cast<int>(fixedTransforms.size()));
        }
    }
}
