package mesh

type ParticleNode struct {
	Position Vec3
	Velocity Vec3
	Force    Vec3
	Mass     float64
	Fixed    bool
}

// ApplyForce accumulates force on a particle unless it is fixed.
func (p *ParticleNode) ApplyForce(force Vec3) {
	if !p.Fixed {
		p.Force = p.Force.Add(force)
	}
}

// Update integrates velocity and position with a simple explicit Euler step.
func (p *ParticleNode) Update(dt float64) {
	if p.Fixed || dt <= 0 {
		return
	}

	acceleration := p.Force.Mul(1.0 / p.Mass)
	p.Velocity = p.Velocity.Add(acceleration.Mul(dt))
	p.Position = p.Position.Add(p.Velocity.Mul(dt))
	p.Force = Vec3{}
}
