# Mesh3D

![demo](./demo/demo.gif)

![demo-png](./demo/demo.png)

A simple project using raylib to simulate a 3D mesh.

## Features

- Adjustable mesh sizes
- Adjustable stiffness
- Adjustable damping factor
- Adjustable animation speed
- Adjustable camera position

## Physics Principles

This project is a cloth simulation based on the **Mass-Spring System**. Each grid intersection is a **Particle**, and particles are connected by **Springs**. The core physics comes from the combination of two forces: elastic restoring force and damping force.

### Hooke's Law and Stiffness

**Stiffness** corresponds to the spring constant **k** in Hooke's Law:

```
F_spring = -k · Δx
```

Where:
- **k** is the `stiffness` value in this project.
- **Δx** is the difference between the spring's current length and its rest length (deformation).

**Intuition:**
- **Higher k**: The spring is "stiffer." The same amount of stretch or compression produces a larger restoring force. The cloth behaves more taut and resists deformation (like canvas or a tightly stretched rubber band).
- **Lower k**: The spring is "softer." The cloth stretches and sags more easily, feeling more elastic (like a loose silk scarf or an elastic bandage).

> In the code, `stiffness` is passed directly as Hooke's Law's `k` into `Spring::ApplySpringForce()`, determining how sensitive each spring is to deformation.

### Damping (Dampness)

Damping is **not** part of Hooke's Law. It is an additional dissipative force proportional to **velocity**:

```
F_damping = -c · v
```

Where:
- **c** is the `dampingFactor` in this project.
- **v** is the relative velocity of the two particles along the spring's direction.

**Intuition:**
- Damping simulates "resistance" or "energy dissipation" — it does not make the cloth harder or softer, but causes oscillations and vibrations to **decay over time**.
- **Higher c**: Energy dissipates faster. After a disturbance (e.g., dragging and releasing with the mouse), the cloth quickly settles down, like fabric underwater.
- **Lower c**: Energy dissipates slower. The cloth will oscillate and swing for a long time, like silk in a vacuum or smooth airflow.
- **c = 0**: Almost no energy loss. The cloth would oscillate forever (ideal case).

### Relationship Between Stiffness and Damping

| Parameter | Physical Quantity | Affected Phenomenon |
|-----------|-------------------|---------------------|
| **Stiffness (k)** | Spring constant in Hooke's Law | Cloth **hardness** and **resistance to deformation** |
| **Damping (c)** | Damping coefficient | Cloth oscillation **decay rate** and **settling time** |

**Analogy**: Imagine a piece of cloth hanging on a clothesline —
- **Stiffness** determines whether the cloth is stiff like thick canvas (high k) or soft and saggy like gauze (low k).
- **Damping** determines whether, after you flick the cloth, it stops quickly like it's in water (high c), or keeps fluttering for a long time like it's in the wind (low c).

They are independently adjustable: you can make the cloth both stiff and oscillating (high k, low c), or soft but quickly stable (low k, high c).

## Controls

### Keyboard
- `mouse` to control the camera to rotate, to zoom in/out
- `r` to restart the simulation
- `space`/`enter` to pause the simulation
- `up`/`down` to increase/decrease the animation speed
- `n`/`m` to increase/decrease the stiffness
- `d`/`f` to increase/decrease the damping factor

### GUI Panel (Right Side)
A control panel on the right side provides buttons and sliders for all actions:
- **Play / Pause** — toggle simulation state
- **Restart** — reset the cloth mesh
- **Camera: Free / Locked** — enable or disable mouse camera control
- **Save Config** — write current settings to `config.txt`
- **Anim Speed** — slider to adjust animation speed (0.05 ~ 3.0)
- **Stiffness** — slider to adjust stiffness (1.0 ~ 50.0), **disabled while simulation is running**
- **Damping** — slider to adjust damping factor (0.0 ~ 5.0)
- **Air Resist** — slider to adjust air resistance (0.0 ~ 0.1)

> **Tip:** Mouse over the right panel automatically disables camera rotation so you can interact with the controls safely.

## Download exe file

[Download](./download/Mesh3D.exe)

## Build from source

1. Clone the repository
2. Run `build.bat` to build the project
