package mesh

import (
	"bufio"
	"fmt"
	"math"
	mathrand "math/rand"
	"os"
	"strconv"
	"strings"
)

type MeshModel struct {
	Particles       []ParticleNode
	Springs         []SpringEdge
	SpringStiffness float64
	DampingFactor   float64
	AirResistance   float64
	Gravity         float64
}

type Config struct {
	Stiffness             float64
	ParticleMass          float64
	DampingFactor         float64
	AirResistanceFactor   float64
	Gravity               float64
	SpringSeed            int64
	MaxSpringDist         float64
	MaxSpringsPerParticle int
	SpringConnectProb     float64
}

// NewMeshModelFromPointCloud loads particles and generates spring topology.
func NewMeshModelFromPointCloud(path string, cfg Config) (*MeshModel, error) {
	particles, err := loadPointCloud(path, cfg.ParticleMass)
	if err != nil {
		return nil, err
	}
	if len(particles) == 0 {
		return nil, fmt.Errorf("point cloud contains no valid particles")
	}

	mesh := &MeshModel{
		Particles:       particles,
		SpringStiffness: cfg.Stiffness,
		DampingFactor:   cfg.DampingFactor,
		AirResistance:   cfg.AirResistanceFactor,
		Gravity:         cfg.Gravity,
	}
	mesh.GenerateRandomSprings(cfg.SpringSeed, cfg.MaxSpringDist, cfg.MaxSpringsPerParticle, cfg.SpringConnectProb)
	return mesh, nil
}

// Update performs one simulation step and reports whether particle positions stayed valid.
func (m *MeshModel) Update(dt float64) bool {
	if dt <= 0 {
		return true
	}

	for i := range m.Particles {
		p := &m.Particles[i]
		p.ApplyForce(Vec3{Y: -m.Gravity})
		p.ApplyForce(Vec3{
			X: -m.AirResistance * p.Velocity.X * math.Abs(p.Velocity.X),
			Y: -m.AirResistance * p.Velocity.Y * math.Abs(p.Velocity.Y),
			Z: -m.AirResistance * p.Velocity.Z * math.Abs(p.Velocity.Z),
		})
	}

	for i := range m.Springs {
		m.Springs[i].Stiffness = m.SpringStiffness
		m.Springs[i].ApplySpringForce(m.DampingFactor)
	}

	for i := range m.Particles {
		m.Particles[i].Update(dt)
	}

	for i := range m.Particles {
		if m.Particles[i].Position.HasNaN() {
			return false
		}
	}
	return true
}

// GenerateRandomSprings connects nearby particles using a deterministic seeded shuffle.
func (m *MeshModel) GenerateRandomSprings(seed int64, maxDist float64, maxPerParticle int, prob float64) {
	if len(m.Particles) < 2 || maxDist <= 0 || maxPerParticle <= 0 || prob <= 0 {
		return
	}
	if prob > 1 {
		prob = 1
	}

	type candidate struct {
		Index int
		Dist  float64
	}

	n := len(m.Particles)
	candidates := make([][]candidate, n)
	connectionCount := make([]int, n)

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			dist := m.Particles[i].Position.Sub(m.Particles[j].Position).Length()
			if dist <= maxDist && dist > 0.0001 {
				candidates[i] = append(candidates[i], candidate{Index: j, Dist: dist})
			}
		}
	}

	for i := 0; i < n; i++ {
		rng := mathrand.New(mathrand.NewSource(seed + int64(i)))
		rng.Shuffle(len(candidates[i]), func(a, b int) {
			candidates[i][a], candidates[i][b] = candidates[i][b], candidates[i][a]
		})
	}

	probRng := mathrand.New(mathrand.NewSource(seed))
	for i := 0; i < n; i++ {
		for _, cand := range candidates[i] {
			j := cand.Index
			if connectionCount[i] >= maxPerParticle || connectionCount[j] >= maxPerParticle {
				continue
			}
			if probRng.Float64() <= prob {
				m.Springs = append(m.Springs, NewSpringEdge(&m.Particles[i], &m.Particles[j], m.SpringStiffness))
				connectionCount[i]++
				connectionCount[j]++
			}
		}
	}
}

// WritePointCloud writes current particle positions in the .msh point-cloud format.
func (m *MeshModel) WritePointCloud(path string, simTime float64, step int, final bool) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	kind := "checkpoint"
	if final {
		kind = "final"
	}
	fmt.Fprintf(out, "# Mesh3D Go solver %s point cloud\n", kind)
	fmt.Fprintf(out, "# Simulated time: %.6fs\n", simTime)
	fmt.Fprintf(out, "# Step: %d\n", step)
	fmt.Fprintf(out, "# Particles: %d\n", len(m.Particles))
	fmt.Fprintf(out, "# Springs: %d\n", len(m.Springs))
	fmt.Fprintf(out, "# Format: x y z fixed mass\n")

	for _, p := range m.Particles {
		fixed := 0
		if p.Fixed {
			fixed = 1
		}
		fmt.Fprintf(out, "%.6f %.6f %.6f %d %.6f\n", p.Position.X, p.Position.Y, p.Position.Z, fixed, p.Mass)
	}
	return nil
}

// loadPointCloud parses a .msh point-cloud file into solver particles.
func loadPointCloud(path string, defaultMass float64) ([]ParticleNode, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var particles []ParticleNode
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, fmt.Errorf("invalid point cloud line %d: expected at least x y z", lineNumber)
		}

		x, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid x value on line %d", lineNumber)
		}
		y, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid y value on line %d", lineNumber)
		}
		z, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid z value on line %d", lineNumber)
		}

		fixed := false
		if len(fields) >= 4 {
			fixedInt, err := strconv.Atoi(fields[3])
			if err != nil {
				return nil, fmt.Errorf("invalid fixed flag on line %d", lineNumber)
			}
			fixed = fixedInt != 0
		}

		mass := defaultMass
		if len(fields) >= 5 {
			mass, err = strconv.ParseFloat(fields[4], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid mass on line %d", lineNumber)
			}
		}
		if mass <= 0 {
			return nil, fmt.Errorf("mass must be positive on line %d", lineNumber)
		}

		particles = append(particles, ParticleNode{
			Position: Vec3{X: x, Y: y, Z: z},
			Mass:     mass,
			Fixed:    fixed,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return particles, nil
}
