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
