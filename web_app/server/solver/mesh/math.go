package mesh

import "math"

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

// Add returns the component-wise sum of two vectors.
func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

// Sub returns the component-wise difference between two vectors.
func (v Vec3) Sub(other Vec3) Vec3 {
	return Vec3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

// Mul returns the vector scaled by a scalar.
func (v Vec3) Mul(scale float64) Vec3 {
	return Vec3{X: v.X * scale, Y: v.Y * scale, Z: v.Z * scale}
}

// Dot returns the vector dot product.
func (v Vec3) Dot(other Vec3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// Length returns the Euclidean vector length.
func (v Vec3) Length() float64 {
	return math.Sqrt(v.Dot(v))
}

// HasNaN reports whether any vector component is not a number.
func (v Vec3) HasNaN() bool {
	return math.IsNaN(v.X) || math.IsNaN(v.Y) || math.IsNaN(v.Z)
}
