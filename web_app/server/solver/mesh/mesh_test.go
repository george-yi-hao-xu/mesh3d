package mesh

import (
	"os"
	"strings"
	"testing"
)

func TestNewMeshModelFromMeshFileLoadsExplicitTopology(t *testing.T) {
	path := writeTestMesh(t, `# Format: mesh-v1

vertices
# index x y z fixed mass
0 0 0 0 1 1
1 1 0 0 0 2

edges
# a_index b_index rest_length stiffness
0 1 1 10
`)

	model, err := NewMeshModelFromMeshFile(path, Config{
		Stiffness:           10,
		DampingFactor:       0.1,
		AirResistanceFactor: 0.001,
		Gravity:             9.8,
	})
	if err != nil {
		t.Fatalf("load mesh: %v", err)
	}
	if len(model.Particles) != 2 {
		t.Fatalf("particles = %d, want 2", len(model.Particles))
	}
	if len(model.Springs) != 1 {
		t.Fatalf("springs = %d, want 1", len(model.Springs))
	}
	if model.Springs[0].A != &model.Particles[0] || model.Springs[0].B != &model.Particles[1] {
		t.Fatalf("spring endpoints were not wired to model particles")
	}
	if model.Springs[0].RestLength != 1 {
		t.Fatalf("rest length = %v, want 1", model.Springs[0].RestLength)
	}
}

func TestNewMeshModelFromMeshFileRejectsPointCloudInput(t *testing.T) {
	path := writeTestMesh(t, `# Format: x y z fixed mass
0 0 0 1 1
1 0 0 0 1
`)

	_, err := NewMeshModelFromMeshFile(path, Config{Stiffness: 10})
	if err == nil {
		t.Fatalf("point-cloud input was accepted")
	}
}

func TestNewMeshModelFromMeshFileRejectsMissingEdges(t *testing.T) {
	path := writeTestMesh(t, `# Format: mesh-v1

vertices
0 0 0 0 1 1
1 1 0 0 0 1
`)

	_, err := NewMeshModelFromMeshFile(path, Config{Stiffness: 10})
	if err == nil || !strings.Contains(err.Error(), "springs") {
		t.Fatalf("error = %v, want missing springs rejection", err)
	}
}

func TestNewMeshModelFromMeshFileRejectsInvalidEdgeIndex(t *testing.T) {
	path := writeTestMesh(t, `# Format: mesh-v1

vertices
0 0 0 0 1 1
1 1 0 0 0 1

edges
0 2 1 10
`)

	_, err := NewMeshModelFromMeshFile(path, Config{Stiffness: 10})
	if err == nil || !strings.Contains(err.Error(), "invalid vertex") {
		t.Fatalf("error = %v, want invalid edge rejection", err)
	}
}

func writeTestMesh(t *testing.T, text string) string {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "mesh-*.mesh")
	if err != nil {
		t.Fatalf("create temp mesh: %v", err)
	}
	if _, err := file.WriteString(text); err != nil {
		t.Fatalf("write temp mesh: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close temp mesh: %v", err)
	}
	return file.Name()
}
