package solver

import (
	"fmt"
	"math"
	"strings"
)

// SnapshotLabel formats simulation time for UI labels.
func SnapshotLabel(simTime float64) string {
	if math.Abs(simTime-math.Round(simTime)) < 1e-9 {
		return fmt.Sprintf("%.0fs", simTime)
	}
	return fmt.Sprintf("%.2fs", simTime)
}

// SnapshotFileName formats simulation time into a stable checkpoint file name.
func SnapshotFileName(simTime float64) string {
	label := SnapshotLabel(simTime)
	label = strings.ReplaceAll(label, ".", "p")
	if len(label) < 6 {
		label = strings.Repeat("0", 6-len(label)) + label
	}
	return label + ".msh"
}
