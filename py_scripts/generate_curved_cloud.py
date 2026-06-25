
"""
Generate curved-boundary point clouds for the mesh3d simulator.

Output format:
    x y z fixed mass

The boundary points are pinned (fixed=1). Interior points are free (fixed=0).
The generated file can be loaded directly as a point cloud.

生成原理概览：
1. 在 XZ 平面用极坐标角度 theta 描出一圈闭合边界，边界点固定。
2. 在边界包围盒里铺近似六角网格，并给每个候选点加入少量随机扰动。
3. 用射线法判断候选点是否落在边界多边形内部，只保留内部自由点。
4. 可选地根据 x/z 坐标计算 y 高度，让点云初始时带有曲面起伏。
"""

from __future__ import annotations

import argparse
import math
import random
from dataclasses import dataclass
from pathlib import Path


# Default generation settings. The total point count is:
# DEFAULT_BOUNDARY_COUNT + DEFAULT_INTERIOR_COUNT.
POINT_COUNT_PRESETS = {
    "12k": (384, 11616),
    "48k": (768, 47232),
}
DEFAULT_POINT_PRESET = "12k"
DEFAULT_BOUNDARY_COUNT, DEFAULT_INTERIOR_COUNT = POINT_COUNT_PRESETS[DEFAULT_POINT_PRESET]
DEFAULT_OUTPUT = "curved_zaha_cloud_12k.msh"
DEFAULT_STYLE = "zaha"
DEFAULT_SCALE = 4.0
DEFAULT_ASPECT = 0.62
DEFAULT_HEIGHT = 0.0
DEFAULT_MASS = 1.0
DEFAULT_SEED = 42


@dataclass
class Point:
    x: float
    y: float
    z: float
    fixed: int
    mass: float


def clamp(value: float, low: float, high: float) -> float:
    return max(low, min(high, value))


def boundary_radius(theta: float, style: str) -> float:
    # 在极坐标中，半径 r(theta) 决定边界轮廓。
    # ellipse 使用常数半径，配合 aspect 得到椭圆；zaha/wave 叠加多组正弦波，
    # 让半径随角度平滑变化，从而得到连续的曲线边界。
    if style == "ellipse":
        return 1.0
    if style == "zaha":
        return (
            1.0
            + 0.18 * math.sin(3.0 * theta + 0.7)
            + 0.10 * math.cos(5.0 * theta - 0.4)
            + 0.06 * math.sin(2.0 * theta) * math.cos(theta)
        )
    if style == "wave":
        return 1.0 + 0.14 * math.sin(4.0 * theta) + 0.08 * math.cos(7.0 * theta)
    raise ValueError(f"Unknown style: {style}")


def boundary_point(theta: float, scale: float, aspect: float, style: str) -> tuple[float, float]:
    # 将极坐标边界点转换到 XZ 平面：
    # x = scale * r * cos(theta)，z = scale * aspect * r * sin(theta)。
    # aspect 只压缩/拉伸 Z 轴，不改变角度采样顺序。
    radius = boundary_radius(theta, style)
    x = scale * radius * math.cos(theta)
    z = scale * aspect * radius * math.sin(theta)

    if style == "zaha":
        # zaha 风格额外做一个平滑剪切和 Z 向偏移，
        # 让平面轮廓不只是径向起伏，而具有更强的流线形不对称感。
        shear = 0.24 * scale * math.sin(theta) * math.sin(theta + 0.8)
        x += shear
        z += 0.10 * scale * math.sin(2.0 * theta - 0.5)

    return x, z


def surface_height(x: float, z: float, scale: float, height: float, style: str) -> float:
    # y 高度不是用来决定边界内外的；边界和采样都发生在 XZ 平面。
    # 这里把 x/z 归一化后代入平滑三角函数，只负责给点云初始形态添加起伏。
    if height == 0.0:
        return 0.0

    nx = x / max(scale, 0.001)
    nz = z / max(scale, 0.001)

    if style == "zaha":
        return height * (
            0.55 * math.sin(1.4 * nx + 0.7 * nz)
            + 0.35 * math.cos(1.8 * nz - 0.5 * nx)
            + 0.25 * nx
        )
    if style == "wave":
        return height * (0.65 * math.sin(1.8 * nx) + 0.35 * math.cos(2.2 * nz))
    return height * 0.30 * math.cos(1.5 * nx) * math.sin(1.5 * nz)


def polygon_area(poly: list[tuple[float, float]]) -> float:
    # 鞋带公式：有向面积为正表示顶点大致按逆时针排列。
    # 后续会用它统一边界点方向，避免不同风格生成出反向多边形。
    area = 0.0
    for i, (x0, z0) in enumerate(poly):
        x1, z1 = poly[(i + 1) % len(poly)]
        area += x0 * z1 - x1 * z0
    return 0.5 * area


def point_in_polygon(x: float, z: float, poly: list[tuple[float, float]]) -> bool:
    # 射线法判断点是否在多边形内：
    # 从测试点向 +X 方向发一条水平射线，统计它与边界边的交叉次数。
    # 每穿过一次边界就翻转 inside；最终为 True 表示交叉次数为奇数，在内部。
    inside = False
    j = len(poly) - 1
    for i in range(len(poly)):
        xi, zi = poly[i]
        xj, zj = poly[j]
        crosses = (zi > z) != (zj > z)
        if crosses:
            x_at_z = (xj - xi) * (z - zi) / (zj - zi) + xi
            if x < x_at_z:
                inside = not inside
        j = i
    return inside


def generate_points(
    style: str,
    boundary_count: int,
    interior_count: int,
    scale: float,
    aspect: float,
    height: float,
    mass: float,
    seed: int,
) -> list[Point]:
    rng = random.Random(seed)

    # 均匀采样角度生成固定边界点。边界点之后会写成 fixed=1，
    # 模拟器可将它们当作固定约束来撑住曲面外轮廓。
    boundary_2d = [
        boundary_point(2.0 * math.pi * i / boundary_count, scale, aspect, style)
        for i in range(boundary_count)
    ]
    if polygon_area(boundary_2d) < 0.0:
        boundary_2d.reverse()

    points: list[Point] = []
    for x, z in boundary_2d:
        y = surface_height(x, z, scale, height, style)
        points.append(Point(x, y, z, 1, mass))

    min_x = min(x for x, _ in boundary_2d)
    max_x = max(x for x, _ in boundary_2d)
    min_z = min(z for _, z in boundary_2d)
    max_z = max(z for _, z in boundary_2d)

    # 用多边形面积估算内部点的平均间距。面积越大或点越少，spacing 越大。
    # step 略小于理论间距，让后续裁剪后仍更容易达到目标 interior_count。
    area = abs(polygon_area(boundary_2d))
    spacing = math.sqrt(area / max(interior_count, 1))
    step = spacing * 0.92

    # 候选点按交错行排列，行距为 step * sqrt(3) / 2，
    # 这是近似六角/三角网格的常用间距，点分布比普通方格更均匀。
    # jitter 会打破完全规则的格点，减少模拟中出现方向性条纹。
    candidates: list[tuple[float, float]] = []
    z = min_z + step * 0.5
    row = 0
    while z <= max_z:
        x_offset = 0.5 * step if row % 2 else 0.0
        x = min_x + step * 0.5 + x_offset
        while x <= max_x:
            jitter_x = rng.uniform(-0.22, 0.22) * step
            jitter_z = rng.uniform(-0.22, 0.22) * step
            px = x + jitter_x
            pz = z + jitter_z
            # 包围盒里会有大量点落在曲线外侧，必须用多边形内外测试裁掉。
            if point_in_polygon(px, pz, boundary_2d):
                candidates.append((px, pz))
            x += step
        z += step * math.sqrt(3.0) * 0.5
        row += 1

    rng.shuffle(candidates)
    # 打乱候选列表后截取目标数量，避免总是偏向扫描顺序靠前的区域。
    for x, z in candidates[:interior_count]:
        y = surface_height(x, z, scale, height, style)
        points.append(Point(x, y, z, 0, mass))

    return points


def write_msh(path: Path, points: list[Point], args: argparse.Namespace) -> None:
    with path.open("w", encoding="utf-8", newline="\n") as file:
        file.write("# Curved point cloud generated by tools/generate_curved_cloud.py\n")
        file.write("# Format: x y z fixed mass\n")
        file.write("# fixed: 1 = pinned boundary, 0 = free interior\n")
        file.write(f"# style={args.style} boundary={args.boundary} interior={args.interior} seed={args.seed}\n\n")
        file.write("# Boundary points\n")
        for p in points:
            if p.fixed:
                file.write(f"{p.x: .6f} {p.y: .6f} {p.z: .6f} {p.fixed:d} {p.mass:.6f}\n")
        file.write("\n# Interior points\n")
        for p in points:
            if not p.fixed:
                file.write(f"{p.x: .6f} {p.y: .6f} {p.z: .6f} {p.fixed:d} {p.mass:.6f}\n")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate a curved-boundary .msh point cloud.")
    parser.add_argument("--output", "-o", default=DEFAULT_OUTPUT, help="Output .msh path.")
    parser.add_argument("--style", choices=("ellipse", "zaha", "wave"), default=DEFAULT_STYLE, help="Boundary style.")
    parser.add_argument(
        "--preset",
        choices=tuple(POINT_COUNT_PRESETS),
        default=DEFAULT_POINT_PRESET,
        help="Point-count preset.",
    )
    parser.add_argument("--boundary", type=int, default=None, help="Pinned boundary point count.")
    parser.add_argument("--interior", type=int, default=None, help="Free interior point count.")
    parser.add_argument("--scale", type=float, default=DEFAULT_SCALE, help="Overall footprint scale.")
    parser.add_argument("--aspect", type=float, default=DEFAULT_ASPECT, help="Z axis aspect ratio.")
    parser.add_argument("--height", type=float, default=DEFAULT_HEIGHT, help="Initial Y height wave amplitude.")
    parser.add_argument("--mass", type=float, default=DEFAULT_MASS, help="Particle mass.")
    parser.add_argument("--seed", type=int, default=DEFAULT_SEED, help="Random seed for interior jitter.")
    args = parser.parse_args()

    preset_boundary, preset_interior = POINT_COUNT_PRESETS[args.preset]
    if args.boundary is None:
        args.boundary = preset_boundary
    if args.interior is None:
        args.interior = preset_interior

    if args.boundary < 12:
        parser.error("--boundary must be at least 12")
    if args.interior < 0:
        parser.error("--interior must be non-negative")
    if args.scale <= 0.0:
        parser.error("--scale must be positive")
    if args.aspect <= 0.0:
        parser.error("--aspect must be positive")
    if args.mass <= 0.0:
        parser.error("--mass must be positive")
    return args


def main() -> None:
    args = parse_args()
    points = generate_points(
        style=args.style,
        boundary_count=args.boundary,
        interior_count=args.interior,
        scale=args.scale,
        aspect=args.aspect,
        height=args.height,
        mass=args.mass,
        seed=args.seed,
    )
    output = Path(args.output)
    write_msh(output, points, args)
    print(f"Wrote {len(points)} points to {output}")


if __name__ == "__main__":
    main()
