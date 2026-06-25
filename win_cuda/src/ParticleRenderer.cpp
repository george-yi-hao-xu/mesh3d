#include "ParticleRenderer.h"

#include "raylib.h"
#include "raymath.h"

namespace mesh3d {
    namespace {
        constexpr float PARTICLE_RADIUS = 0.035f;
        constexpr int PARTICLE_RINGS = 4;
        constexpr int PARTICLE_SLICES = 4;

        // Vertex shader for particle instancing. Each instance reuses the same
        // sphere mesh, while instanceTransform places it at one particle position.
        const char* PARTICLE_INSTANCING_VS = R"(
#version 330
in vec3 vertexPosition;
in vec3 vertexNormal;
in vec2 vertexTexCoord;
in mat4 instanceTransform;

uniform mat4 mvp;

out vec2 fragTexCoord;
out vec3 fragNormal;

void main()
{
    fragTexCoord = vertexTexCoord;
    fragNormal = normalize(vertexNormal);
    gl_Position = mvp*instanceTransform*vec4(vertexPosition, 1.0);
}
)";

        // Fragment shader for particle instancing. A small ambient term keeps
        // back-facing particles visible, while directional diffuse light gives
        // dense point clouds enough shape cues to read as depth instead of a flat mass.
        const char* PARTICLE_INSTANCING_FS = R"(
#version 330
in vec2 fragTexCoord;
in vec3 fragNormal;

uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform vec3 lightDir;
uniform float ambientStrength;

out vec4 finalColor;

void main()
{
    vec3 normal = normalize(fragNormal);
    float diffuse = max(dot(normal, normalize(-lightDir)), 0.0);
    float light = clamp(ambientStrength + diffuse*(1.0 - ambientStrength), 0.0, 1.0);
    vec4 texel = texture(texture0, fragTexCoord);
    finalColor = vec4(texel.rgb*colDiffuse.rgb*light, texel.a*colDiffuse.a);
}
)";

        struct ParticleInstancingRenderer {
            bool initialized = false;
            bool available = false;
            Model model = {};
            Material material = {};
            int lightDirLoc = -1;
            int ambientStrengthLoc = -1;
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
            shader.locs[SHADER_LOC_VERTEX_NORMAL] = GetShaderLocationAttrib(shader, "vertexNormal");
            shader.locs[SHADER_LOC_VERTEX_TEXCOORD01] = GetShaderLocationAttrib(shader, "vertexTexCoord");
            shader.locs[SHADER_LOC_MATRIX_MVP] = GetShaderLocation(shader, "mvp");
            shader.locs[SHADER_LOC_COLOR_DIFFUSE] = GetShaderLocation(shader, "colDiffuse");
            shader.locs[SHADER_LOC_MAP_DIFFUSE] = GetShaderLocation(shader, "texture0");
            shader.locs[SHADER_LOC_MATRIX_MODEL] = GetShaderLocationAttrib(shader, "instanceTransform");
            renderer.lightDirLoc = GetShaderLocation(shader, "lightDir");
            renderer.ambientStrengthLoc = GetShaderLocation(shader, "ambientStrength");

            renderer.material = LoadMaterialDefault();
            renderer.material.shader = shader;
            // If any required shader binding is missing, keep the old safe
            // DrawSphereEx path so a shader issue cannot make particles vanish.
            renderer.available = renderer.model.meshCount > 0 &&
                shader.id > 0 &&
                shader.locs[SHADER_LOC_VERTEX_POSITION] >= 0 &&
                shader.locs[SHADER_LOC_VERTEX_NORMAL] >= 0 &&
                shader.locs[SHADER_LOC_MATRIX_MVP] >= 0 &&
                shader.locs[SHADER_LOC_COLOR_DIFFUSE] >= 0 &&
                shader.locs[SHADER_LOC_MATRIX_MODEL] >= 0 &&
                renderer.lightDirLoc >= 0 &&
                renderer.ambientStrengthLoc >= 0;

            return renderer;
        }

        void ApplyParticleLighting(const ParticleInstancingRenderer& renderer) {
            const float lightDir[3] = { -0.45f, -0.85f, -0.25f };
            const float ambientStrength = 0.38f;
            SetShaderValue(renderer.material.shader, renderer.lightDirLoc, lightDir, SHADER_UNIFORM_VEC3);
            SetShaderValue(renderer.material.shader, renderer.ambientStrengthLoc, &ambientStrength, SHADER_UNIFORM_FLOAT);
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

        ApplyParticleLighting(renderer);

        if (!freeTransforms.empty()) {
            renderer.material.maps[MATERIAL_MAP_DIFFUSE].color = Color{ 224, 222, 214, 255 };
            DrawMeshInstanced(renderer.model.meshes[0], renderer.material, freeTransforms.data(), static_cast<int>(freeTransforms.size()));
        }

        if (!fixedTransforms.empty()) {
            renderer.material.maps[MATERIAL_MAP_DIFFUSE].color = Color{ 196, 161, 86, 255 };
            DrawMeshInstanced(renderer.model.meshes[0], renderer.material, fixedTransforms.data(), static_cast<int>(fixedTransforms.size()));
        }
    }
}
