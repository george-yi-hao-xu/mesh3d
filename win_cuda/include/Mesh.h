#pragma once
#include <vector>
#include "Particle.h"
#include "Spring.h"
#include <string>
#include <ostream>

namespace mesh3d{
    struct Config {
        int width = 10;
        int height = 10;
        float spacing = 1.0f;
        float stiffness = 10.0f;
        float particleMass = 1.0f;
        float dampingFactor = 0.1f;
        float airResistanceFactor = 0.001f;
        float gravity = 9.8f;
        std::string pointCloudFile = "";
        unsigned int springSeed = 42;
        float maxSpringDist = 1.5f;
        int maxSpringsPerParticle = 4;
        float springConnectProb = 0.8f;
    };

    Config LoadMeshConfig(const std::string& filename);
    void WriteConfig(const std::string& filename, const Config& config);
    
    class Mesh {
    private:
        int width, height;
        std::vector<Particle> particles;
        std::vector<Spring> springs;

        // default
        float springStiffness = 20.0f;
        float dampingFactor = 10.0f;
        float airResistanceFactor = 0.001f;
        float gravity = 9.8f;

        void BuildRegularGrid(const Config& c);
        void BuildFromPointCloud(const Config& c, const char* ptFileName = nullptr);
        static std::vector<Particle> LoadPointCloud(const std::string& path);
        void GenerateRandomSprings(unsigned int seed, float maxDist, int maxPerParticle, float prob);
    public:
        Mesh(const Config& config, const char* ptFileName = nullptr);
        bool Update(float dt);
        void Draw();
        void WritePointCloud(std::ostream& out) const;
        size_t ParticleCount() const { return particles.size(); }
        size_t SpringCount() const { return springs.size(); }
    };
} // namespace mesh3d
