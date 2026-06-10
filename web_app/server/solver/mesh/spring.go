package mesh

type SpringEdge struct {
	A          *ParticleNode
	B          *ParticleNode
	RestLength float64
	Stiffness  float64
}

// NewSpringEdge creates a spring and captures its initial rest length.
func NewSpringEdge(a, b *ParticleNode, stiffness float64) SpringEdge {
	return SpringEdge{
		A:          a,
		B:          b,
		RestLength: b.Position.Sub(a.Position).Length(),
		Stiffness:  stiffness,
	}
}

// ApplySpringForce applies Hooke and spring-axis damping forces to both particles.
func (s *SpringEdge) ApplySpringForce(dampingFactor float64) {
	diff := s.B.Position.Sub(s.A.Position)
	currentLength := diff.Length()
	if currentLength < 0.0001 {
		return
	}

	dir := diff.Mul(1.0 / currentLength)
	displacement := currentLength - s.RestLength
	springForce := dir.Mul(-s.Stiffness * displacement)

	velocityDiff := s.B.Velocity.Sub(s.A.Velocity)
	velocityAlongSpring := velocityDiff.Dot(dir)
	dampingForce := dir.Mul(-dampingFactor * velocityAlongSpring)

	totalForce := springForce.Add(dampingForce)
	s.A.ApplyForce(totalForce.Mul(-1))
	s.B.ApplyForce(totalForce)
}
