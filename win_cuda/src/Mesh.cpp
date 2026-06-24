#include "Mesh.h"
#include "raylib.h"
#include <iostream>
#include <future>
#include <vector>
#include <fstream>
#include <sstream>
#include <string>
#include <cmath>
#include <random>
#include <algorithm>
#include <iomanip>

const float HEIGHT = 0.0f;

namespace mesh3d {
    Config LoadMeshConfig(const std::string& filename) {
        Config config;
        std::ifstream file(filename);
        std::string line;
        while (std::getline(file, line)) {
            std::istringstream iss(line);
            std::string key;
            if (std::getline(iss, key, '=')) {
                std::string value;
                if (std::getline(iss, value)) {
                    value.erase(value.find_last_not_of(" \t\r\n") + 1);
                    if (key == "width") config.width = std::stoi(value);
                    else if (key == "height") config.height = std::stoi(value);
                    else if (key == "spacing") config.spacing = std::stof(value);
                    else if (key == "stiffness") config.stiffness = std::stof(value);
                    else if (key == "particleMass") config.particleMass = std::stof(value);
                    else if (key == "dampingFactor") config.dampingFactor = std::stof(value);
                    else if (key == "airResistanceFactor") config.airResistanceFactor = std::stof(value);
                    else if (key == "gravity") config.gravity = std::stof(value);
                    else if (key == "pointCloudFile") config.pointCloudFile = value;
                    else if (key == "springSeed") config.springSeed = static_cast<unsigned int>(std::stoul(value));
                    else if (key == "maxSpringDist") config.maxSpringDist = std::stof(value);
                    else if (key == "maxSpringsPerParticle") config.maxSpringsPerParticle = std::stoi(value);
                    else if (key == "springConnectProb") config.springConnectProb = std::stof(value);
                }
            }
        }
        return config;
    }

    void WriteConfig(const std::string& filename, const Config& config) {
        std::ofstream file(filename);
        file << "width=" << config.width << "\n";
        file << "height=" << config.height << "\n";
        file << "spacing=" << config.spacing << "\n";
        file << "stiffness=" << config.stiffness << "\n";
        file << "particleMass=" << config.particleMass << "\n";
        file << "dampingFactor=" << config.dampingFactor << "\n";
        file << "airResistanceFactor=" << config.airResistanceFactor << "\n";
        file << "gravity=" << config.gravity << "\n";
        file << "pointCloudFile=" << config.pointCloudFile << "\n";
        file << "springSeed=" << config.springSeed << "\n";
        file << "maxSpringDist=" << config.maxSpringDist << "\n";
        file << "maxSpringsPerParticle=" << config.maxSpringsPerParticle << "\n";
        file << "springConnectProb=" << config.springConnectProb << "\n";
    }

    void Mesh::BuildRegularGrid(const Config& c) {
        const Vector3 ORIGIN = { (c.width - 1) * c.spacing / 2, HEIGHT, (c.height - 1) * c.spacing / 2 };

        particles.reserve(c.width * c.height);
        springs.reserve((c.width - 1) * c.height + (c.height - 1) * c.width);

        for (int y = 0; y < c.height; y++) {
            for (int x = 0; x < c.width; x++) {
                bool fixed = (y == 0 || x == 0 || y == c.height - 1 || x == c.width - 1);
                particles.emplace_back(Vector3{ x * c.spacing - ORIGIN.x, 0 - ORIGIN.y, y * c.spacing - ORIGIN.z }, fixed, c.particleMass);
            }
        }

        for (int y = 0; y < c.height; y++) {
            for (int x = 0; x < c.width; x++) {
                int idx = y * c.width + x;
                if (x < c.width - 1) springs.emplace_back(&particles[idx], &particles[idx + 1], c.stiffness);
                if (y < c.height - 1) springs.emplace_back(&particles[idx], &particles[idx + c.width], c.stiffness);
            }
        }
    }

    std::vector<Particle> Mesh::LoadPointCloud(const std::string& path) {
        std::vector<Particle> loaded;
        std::ifstream file(path);
        if (!file.is_open()) {
            std::cerr << "Failed to open point cloud file: " << path << std::endl;
            return loaded;
        }

        std::string line;
        while (std::getline(file, line)) {
            line.erase(line.find_last_not_of(" \t\r\n") + 1);
            if (line.empty() || line[0] == '#') continue;

            std::istringstream iss(line);
            float x, y, z, mass;
            int fixed;
            if (iss >> x >> y >> z >> fixed >> mass) {
                loaded.emplace_back(Vector3{ x, y, z }, fixed != 0, mass);
            } else {
                std::cerr << "Invalid point cloud line: " << line << std::endl;
            }
        }
        return loaded;
    }

    // ===================================================================
    // GenerateRandomSprings: 基于种子的随机弹簧生成算法
    //
    // 核心思想：我们不知道不规则点云里哪些点该连起来，所以让程序
    // 根据距离阈值 + 随机概率来自动决定连接关系。
    //
    // 参数说明：
    //   seed          — 随机种子。同样的种子 -> 同样的连接拓扑（可复现）
    //   maxDist       — 距离阈值。两个粒子离得太远就不考虑连弹簧
    //   maxPerParticle— 每个粒子最多连几根弹簧（防止某个点连太多）
    //   prob          — 连接概率。即使距离够近，也只有 prob 概率会真的连
    // ===================================================================
    void Mesh::GenerateRandomSprings(unsigned int seed, float maxDist, int maxPerParticle, float prob) {
        // 类型别名：让后面代码更易读
        // Candidate = 一个候选对 {对方粒子编号, 距离}
        using Candidate = std::pair<size_t, float>;
        // CandidateList = 某个粒子的所有候选列表
        using CandidateList = std::vector<Candidate>;
        // 粒子太少，连不起来，直接返回
        if (particles.size() < 2) return;

        size_t n = particles.size();

        // ---------------------------------------------------------------
        // Step 1: 统计每个粒子已经连了多少根弹簧
        // connectionCount[i] = 粒子 i 当前的弹簧数量
        // 初始都是 0
        // ---------------------------------------------------------------
        std::vector<int> connectionCount(n, 0);

        // ---------------------------------------------------------------
        // Step 2: 找出所有"候选"弹簧对
        //
        // 对于每一对粒子 (i, j)，只算一次（j > i，避免重复）。
        // 如果它们的距离 <= maxDist，就把 j 加入 i 的候选列表。
        //
        // candidates[i] 是一个列表，里面存的是所有和粒子 i 距离够近、
        // 可以考虑连弹簧的其他粒子编号。
        // ---------------------------------------------------------------
        std::vector<CandidateList> candidates(n);
        for (size_t i = 0; i < n; ++i) {
            for (size_t j = i + 1; j < n; ++j) {
                // 计算粒子 i 和 j 的欧几里得距离
                float dx = particles[i].position.x - particles[j].position.x;
                float dy = particles[i].position.y - particles[j].position.y;
                float dz = particles[i].position.z - particles[j].position.z;
                float dist = std::sqrt(dx * dx + dy * dy + dz * dz);

                // 距离在阈值内，且不是同一个点（dist > 0），就加入候选
                if (dist <= maxDist && dist > 0.0001f) {
                    candidates[i].push_back({ j, dist });
                }
            }
        }

        // ---------------------------------------------------------------
        // Step 3: 用种子打乱每个粒子的候选顺序
        //
        // 为什么要打乱？
        // 如果不打乱，程序总是按粒子编号从小到大尝试连接，
        // 那么编号小的粒子会优先占满 maxPerParticle 的名额，
        // 导致编号大的粒子几乎连不上。这会让连接分布不均匀。
        //
        // 每个粒子用不同的种子 (seed + i) 打乱，保证结果确定但又多样。
        // ---------------------------------------------------------------
        for (size_t i = 0; i < n; ++i) {
            std::mt19937 rng(static_cast<unsigned int>(seed + i));
            std::shuffle(candidates[i].begin(), candidates[i].end(), rng);
        }

        // ---------------------------------------------------------------
        // Step 4: 准备随机数生成器，用于"概率判定"
        //
        // dist01 会生成 [0.0, 1.0] 之间的均匀随机小数。
        // 如果生成的数 <= prob，就建立连接。
        // 例如 prob = 0.8，意味着 80% 的候选对会被连接。
        // ---------------------------------------------------------------
        std::mt19937 probRng(seed);
        std::uniform_real_distribution<float> dist01(0.0f, 1.0f);

        // ---------------------------------------------------------------
        // Step 5: 遍历所有候选，决定是否建立弹簧
        //
        // 规则：
        //   1. 如果 i 或 j 已经连满了（>= maxPerParticle），跳过
        //   2. 抽一个随机数，如果 <= prob，就连上
        //   3. 连上后，i 和 j 的 connectionCount 各 +1
        //
        // 注意：这里用的是 candidates[i] 里的 j，而 j > i，
        // 所以同一对粒子不会被处理两次。
        // ---------------------------------------------------------------
        for (size_t i = 0; i < n; ++i) {
            for (const auto& cand : candidates[i]) {
                size_t j = cand.first;  // cand = {粒子编号, 距离}

                // 任意一方已经连满了，就不连了
                if (connectionCount[i] >= maxPerParticle || connectionCount[j] >= maxPerParticle)
                    continue;

                // 概率判定：抽一个 0~1 的随机数，看运气
                if (dist01(probRng) <= prob) {
                    // 创建弹簧，连接粒子 i 和 j，劲度系数用当前的 springStiffness
                    springs.emplace_back(&particles[i], &particles[j], springStiffness);
                    connectionCount[i]++;
                    connectionCount[j]++;
                }
            }
        }
    }

    void Mesh::BuildFromPointCloud(const Config& c, const char* ptFileName) {
        if (ptFileName == nullptr) {
            BuildRegularGrid(c);
            return;
        }

        // gen points first
        particles = LoadPointCloud(ptFileName);
        if (particles.empty()) {
            std::cerr << "Point cloud file empty or failed to load. Falling back to regular grid." << std::endl;
            BuildRegularGrid(c);
            return;
        }
        // then gen springs
        GenerateRandomSprings(c.springSeed, c.maxSpringDist, c.maxSpringsPerParticle, c.springConnectProb);
    }

    Mesh::Mesh(const Config& c, const char* ptFileName) {
        springStiffness = c.stiffness;
        dampingFactor = c.dampingFactor;
        airResistanceFactor = c.airResistanceFactor;
        gravity = c.gravity;

        if (ptFileName != nullptr && ptFileName[0] != '\0') {
            BuildFromPointCloud(c, ptFileName);
        } else {
            BuildRegularGrid(c);
        }
    }

    bool Mesh::Update(float dt) {
        if (dt <= 0.0f) return true;

        for (auto& particle : particles) {
            particle.ApplyForce(Vector3{ 0, -gravity, 0 });
            particle.ApplyForce(Vector3{
                -airResistanceFactor * particle.velocity.x * std::abs(particle.velocity.x),
                -airResistanceFactor * particle.velocity.y * std::abs(particle.velocity.y),
                -airResistanceFactor * particle.velocity.z * std::abs(particle.velocity.z)
                });
        }

        for (auto& spring : springs) {
            spring.stiffness = springStiffness;
            spring.ApplySpringForce(dampingFactor);
        }

        for (auto& particle : particles) {
            particle.Update(dt);
        }

        for (auto& particle : particles) {
            if (std::isnan(particle.position.x) || std::isnan(particle.position.y) || std::isnan(particle.position.z)) {
                std::cerr << "Particle position invalid: " << particle.position.x << ", " << particle.position.y << ", " << particle.position.z << std::endl;
                return false;
            }
        }

        return true;
    }

    void Mesh::Draw() {
        for (auto& spring : springs) {
            DrawLine3D(spring.pA->position, spring.pB->position, BLUE);
        }

        for (auto& particle : particles) {
            DrawSphereEx(particle.position, 0.1f, 4, 4, particle.isFixed ? RED : GREEN);
        }
    }

    void Mesh::WritePointCloud(std::ostream& out) const {
        const std::ios::fmtflags oldFlags = out.flags();
        const std::streamsize oldPrecision = out.precision();

        out << "# Exported point cloud from current mesh state\n";
        out << "# Format: x y z fixed mass\n";
        out << std::fixed << std::setprecision(6);

        for (const auto& particle : particles) {
            out << particle.position.x << ' '
                << particle.position.y << ' '
                << particle.position.z << ' '
                << (particle.isFixed ? 1 : 0) << ' '
                << particle.mass << '\n';
        }

        out.flags(oldFlags);
        out.precision(oldPrecision);
    }
}
