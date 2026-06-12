package mesh

import (
	"bufio"
	"fmt"
	"io"
	"math"
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
	Stiffness           float64
	DampingFactor       float64
	AirResistanceFactor float64
	Gravity             float64
}

// NewMeshModelFromMeshFile loads particles and explicit spring topology from a mesh-v1 file.
func NewMeshModelFromMeshFile(path string, cfg Config) (*MeshModel, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return NewMeshModelFromReader(file, cfg)
}

// NewMeshModelFromReader loads particles and explicit spring topology from mesh-v1 text.
func NewMeshModelFromReader(r io.Reader, cfg Config) (*MeshModel, error) {
	particles, springs, err := loadMeshV1(r, cfg.Stiffness)
	if err != nil {
		return nil, err
	}
	if len(particles) == 0 {
		return nil, fmt.Errorf("mesh contains no valid particles")
	}
	if len(springs) == 0 {
		return nil, fmt.Errorf("mesh contains no valid springs")
	}

	return &MeshModel{
		Particles:       particles,
		Springs:         springs,
		SpringStiffness: cfg.Stiffness,
		DampingFactor:   cfg.DampingFactor,
		AirResistance:   cfg.AirResistanceFactor,
		Gravity:         cfg.Gravity,
	}, nil
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

// WriteMeshSnapshot writes particle positions and spring topology in the mesh-v1 format.
func (m *MeshModel) WriteMeshSnapshot(path string, simTime float64, step int, final bool) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return m.WriteMeshSnapshotTo(out, simTime, step, final)
}

// WriteMeshSnapshotTo writes particle positions and spring topology in the mesh-v1 format.
func (m *MeshModel) WriteMeshSnapshotTo(out io.Writer, simTime float64, step int, final bool) error {
	kind := "checkpoint"
	if final {
		kind = "final"
	}
	particleIndex := make(map[*ParticleNode]int, len(m.Particles))
	for i := range m.Particles {
		particleIndex[&m.Particles[i]] = i
	}

	fmt.Fprintf(out, "# Mesh3D mesh snapshot\n")
	fmt.Fprintf(out, "# Format: mesh-v1\n")
	fmt.Fprintf(out, "# Kind: %s\n", kind)
	fmt.Fprintf(out, "# Simulated time: %.6fs\n", simTime)
	fmt.Fprintf(out, "# Step: %d\n", step)
	fmt.Fprintf(out, "# Vertices: %d\n", len(m.Particles))
	fmt.Fprintf(out, "# Edges: %d\n", len(m.Springs))
	fmt.Fprintf(out, "\nvertices\n")
	fmt.Fprintf(out, "# index x y z fixed mass\n")
	for i, p := range m.Particles {
		fixed := 0
		if p.Fixed {
			fixed = 1
		}
		fmt.Fprintf(out, "%d %.6f %.6f %.6f %d %.6f\n", i, p.Position.X, p.Position.Y, p.Position.Z, fixed, p.Mass)
	}

	fmt.Fprintf(out, "\nedges\n")
	fmt.Fprintf(out, "# a_index b_index rest_length stiffness\n")
	for _, spring := range m.Springs {
		a, ok := particleIndex[spring.A]
		if !ok {
			return fmt.Errorf("spring endpoint A is not part of mesh particles")
		}
		b, ok := particleIndex[spring.B]
		if !ok {
			return fmt.Errorf("spring endpoint B is not part of mesh particles")
		}
		fmt.Fprintf(out, "%d %d %.6f %.6f\n", a, b, spring.RestLength, spring.Stiffness)
	}
	return nil
}

func loadMeshV1(r io.Reader, defaultStiffness float64) ([]ParticleNode, []SpringEdge, error) {
	var particles []ParticleNode
	var edgeRows []struct {
		a          int
		b          int
		restLength float64
		stiffness  float64
		lineNumber int
	}
	section := ""
	hasMeshFormat := false
	scanner := bufio.NewScanner(r)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "# Format:")), "mesh-v1") {
				hasMeshFormat = true
			}
			continue
		}

		lower := strings.ToLower(line)
		if lower == "vertices" || lower == "edges" {
			section = lower
			continue
		}

		fields := strings.Fields(line)
		switch section {
		case "vertices":
			if len(fields) < 6 {
				return nil, nil, fmt.Errorf("invalid mesh vertex line %d: expected index x y z fixed mass", lineNumber)
			}
			index, err := strconv.Atoi(fields[0])
			if err != nil || index != len(particles) {
				return nil, nil, fmt.Errorf("invalid mesh vertex index on line %d", lineNumber)
			}
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid vertex x on line %d", lineNumber)
			}
			y, err := strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid vertex y on line %d", lineNumber)
			}
			z, err := strconv.ParseFloat(fields[3], 64)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid vertex z on line %d", lineNumber)
			}
			fixedInt, err := strconv.Atoi(fields[4])
			if err != nil {
				return nil, nil, fmt.Errorf("invalid fixed flag on line %d", lineNumber)
			}
			mass, err := strconv.ParseFloat(fields[5], 64)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid mass on line %d", lineNumber)
			}
			if mass <= 0 {
				return nil, nil, fmt.Errorf("mass must be positive on line %d", lineNumber)
			}
			particles = append(particles, ParticleNode{
				Position: Vec3{X: x, Y: y, Z: z},
				Mass:     mass,
				Fixed:    fixedInt != 0,
			})
		case "edges":
			if len(fields) < 4 {
				return nil, nil, fmt.Errorf("invalid mesh edge line %d: expected a_index b_index rest_length stiffness", lineNumber)
			}
			a, err := strconv.Atoi(fields[0])
			if err != nil {
				return nil, nil, fmt.Errorf("invalid edge start index on line %d", lineNumber)
			}
			b, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, nil, fmt.Errorf("invalid edge end index on line %d", lineNumber)
			}
			restLength, err := strconv.ParseFloat(fields[2], 64)
			if err != nil || restLength <= 0 {
				return nil, nil, fmt.Errorf("invalid edge rest length on line %d", lineNumber)
			}
			stiffness, err := strconv.ParseFloat(fields[3], 64)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid edge stiffness on line %d", lineNumber)
			}
			if stiffness <= 0 {
				stiffness = defaultStiffness
			}
			edgeRows = append(edgeRows, struct {
				a          int
				b          int
				restLength float64
				stiffness  float64
				lineNumber int
			}{a: a, b: b, restLength: restLength, stiffness: stiffness, lineNumber: lineNumber})
		default:
			return nil, nil, fmt.Errorf("mesh line %d appears before vertices or edges section", lineNumber)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	if !hasMeshFormat {
		return nil, nil, fmt.Errorf("mesh input must declare # Format: mesh-v1")
	}

	springs := make([]SpringEdge, 0, len(edgeRows))
	for _, edge := range edgeRows {
		if edge.a < 0 || edge.a >= len(particles) || edge.b < 0 || edge.b >= len(particles) || edge.a == edge.b {
			return nil, nil, fmt.Errorf("edge references invalid vertex on line %d", edge.lineNumber)
		}
		springs = append(springs, SpringEdge{
			A:          &particles[edge.a],
			B:          &particles[edge.b],
			RestLength: edge.restLength,
			Stiffness:  edge.stiffness,
		})
	}
	return particles, springs, nil
}
