#!/usr/bin/env python3
"""Generate random mesh3d .msh point-cloud files."""

from __future__ import annotations

import argparse
import math
import random
from pathlib import Path


DEFAULT_COUNT = 48000
DEFAULT_OUTPUT = "random_cloud_48k.msh"
DEFAULT_SEED = 42
DEFAULT_SIZE = 16.0
DEFAULT_JITTER = 0.18
DEFAULT_BOUNDARY_RATIO = 0.06
DEFAULT_MASS = 1.0


def positive_int(value: str) -> int:
    """Parse a CLI value as an integer greater than zero."""
    parsed = int(value)
    if parsed <= 0:
        raise argparse.ArgumentTypeError("must be greater than 0")
    return parsed


def non_negative_float(value: str) -> float:
    """Parse a CLI value as a float that can be zero but not negative."""
    parsed = float(value)
    if parsed < 0:
        raise argparse.ArgumentTypeError("must be greater than or equal to 0")
    return parsed


def positive_float(value: str) -> float:
    """Parse a CLI value as a float greater than zero."""
    parsed = float(value)
    if parsed <= 0:
        raise argparse.ArgumentTypeError("must be greater than 0")
    return parsed


def ratio_float(value: str) -> float:
    """Parse the fixed-boundary ratio and keep it below half the cloth size."""
    parsed = float(value)
    if parsed < 0 or parsed >= 0.5:
        raise argparse.ArgumentTypeError("must be in the range [0, 0.5)")
    return parsed


def format_float(value: float) -> str:
    """Format floats compactly while keeping enough precision for the mesh file."""
    text = f"{value:.6f}"
    return text.rstrip("0").rstrip(".") if "." in text else text


def grid_shape(count: int) -> tuple[int, int]:
    """Choose a near-square row/column layout that can hold the requested points."""
    columns = math.ceil(math.sqrt(count))
    rows = math.ceil(count / columns)
    return columns, rows


def generate_points(
    count: int,
    *,
    seed: int,
    size: float,
    jitter: float,
    boundary_ratio: float,
    mass: float,
) -> list[tuple[float, float, float, int, float]]:
    """Create a noisy flat XZ point cloud with pinned boundary points."""
    rng = random.Random(seed)
    columns, rows = grid_shape(count)
    step_x = size / max(columns - 1, 1)
    step_z = size / max(rows - 1, 1)
    jitter_x = step_x * jitter
    jitter_z = step_z * jitter
    half_size = size / 2.0
    boundary_width = max(step_x, step_z, size * boundary_ratio)
    points: list[tuple[float, float, float, int, float]] = []

    for row in range(rows):
        for column in range(columns):
            if len(points) >= count:
                return points

            base_x = -half_size + column * step_x
            base_z = -half_size + row * step_z
            is_edge = row == 0 or column == 0 or row == rows - 1 or column == columns - 1

            if is_edge:
                x = base_x
                z = base_z
            else:
                x = base_x + rng.uniform(-jitter_x, jitter_x)
                z = base_z + rng.uniform(-jitter_z, jitter_z)

            fixed = int(
                is_edge
                or abs(x + half_size) <= boundary_width
                or abs(x - half_size) <= boundary_width
                or abs(z + half_size) <= boundary_width
                or abs(z - half_size) <= boundary_width
            )
            points.append((x, 0.0, z, fixed, mass))

    return points


def write_msh(path: Path, points: list[tuple[float, float, float, int, float]], seed: int) -> None:
    """Write points to disk in the legacy mesh3d .msh point-cloud format."""
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8", newline="\n") as file:
        file.write("# Generated random point cloud for mesh3d\n")
        file.write("# Format: x y z fixed mass\n")
        file.write(f"# Count: {len(points)}\n")
        file.write(f"# Seed: {seed}\n")
        for x, y, z, fixed, mass in points:
            file.write(
                f"{format_float(x)} {format_float(y)} {format_float(z)} "
                f"{fixed} {format_float(mass)}\n"
            )


def parse_args() -> argparse.Namespace:
    """Define and parse command-line options for the generator."""
    parser = argparse.ArgumentParser(
        description="Generate a random flat cloth-style mesh3d .msh point cloud.",
    )
    parser.add_argument("--count", type=positive_int, default=DEFAULT_COUNT, help="number of points to generate")
    parser.add_argument("--output", type=Path, default=Path(DEFAULT_OUTPUT), help="output .msh path")
    parser.add_argument("--seed", type=int, default=DEFAULT_SEED, help="random seed for reproducible jitter")
    parser.add_argument("--size", type=positive_float, default=DEFAULT_SIZE, help="cloth side length in app units")
    parser.add_argument(
        "--jitter",
        type=non_negative_float,
        default=DEFAULT_JITTER,
        help="interior point jitter as a fraction of grid spacing",
    )
    parser.add_argument(
        "--boundary-ratio",
        type=ratio_float,
        default=DEFAULT_BOUNDARY_RATIO,
        help="fraction of side length treated as a fixed boundary band",
    )
    return parser.parse_args()


def main() -> None:
    """Generate the point cloud, write it to the requested file, and print a summary."""
    args = parse_args()
    points = generate_points(
        args.count,
        seed=args.seed,
        size=args.size,
        jitter=args.jitter,
        boundary_ratio=args.boundary_ratio,
        mass=DEFAULT_MASS,
    )
    write_msh(args.output, points, args.seed)
    fixed_count = sum(1 for point in points if point[3] == 1)
    print(f"Wrote {len(points)} points ({fixed_count} fixed) to {args.output}")


if __name__ == "__main__":
    main()
