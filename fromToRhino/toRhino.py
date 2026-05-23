"""Import a mesh3d .msh point cloud file into Rhino point layers.

The script asks for:
1. The .msh file to load.
2. The layer for fixed/pinned points.
3. The layer for unfixed/free points.

The mesh3d app uses Y as the up axis, while Rhino commonly uses Z as up.
Keep USE_RHINO_Z_UP_MAPPING enabled to preserve the same visual orientation
between mesh3d and Rhino.
"""

from __future__ import print_function

import rhinoscriptsyntax as rs


USE_RHINO_Z_UP_MAPPING = True
DEFAULT_FIXED_LAYER = "mesh3d_fixed"
DEFAULT_UNFIXED_LAYER = "mesh3d_unfixed"


def mesh_to_rhino_point(x, y, z):
    if USE_RHINO_Z_UP_MAPPING:
        return x, z, y
    return x, y, z


def read_msh(path):
    rows = []

    with open(path, "r") as file:
        for line_number, raw_line in enumerate(file, 1):
            line = raw_line.strip()
            if not line or line.startswith("#"):
                continue

            parts = line.split()
            if len(parts) < 5:
                print("Skipping line {}: expected x y z fixed mass".format(line_number))
                continue

            try:
                x = float(parts[0])
                y = float(parts[1])
                z = float(parts[2])
                fixed = int(float(parts[3])) != 0
                mass = float(parts[4])
            except ValueError:
                print("Skipping line {}: could not parse numbers".format(line_number))
                continue

            rows.append((x, y, z, fixed, mass))

    return rows


def ensure_layer(layer_name, color):
    if not rs.IsLayer(layer_name):
        rs.AddLayer(layer_name, color)
    return layer_name


def add_points(rows, fixed_layer, unfixed_layer):
    created = []
    fixed_count = 0
    unfixed_count = 0

    rs.EnableRedraw(False)
    try:
        for x, y, z, fixed, mass in rows:
            point = mesh_to_rhino_point(x, y, z)
            object_id = rs.AddPoint(point)
            if object_id is None:
                continue

            layer = fixed_layer if fixed else unfixed_layer
            rs.ObjectLayer(object_id, layer)
            rs.SetUserText(object_id, "mesh3d_fixed", "1" if fixed else "0")
            rs.SetUserText(object_id, "mesh3d_mass", str(mass))
            created.append(object_id)

            if fixed:
                fixed_count += 1
            else:
                unfixed_count += 1
    finally:
        rs.EnableRedraw(True)

    if created:
        rs.SelectObjects(created)

    return fixed_count, unfixed_count


def main():
    path = rs.OpenFileName(
        "Open mesh3d point cloud",
        "Mesh point cloud (*.msh;*.txt)|*.msh;*.txt||",
    )
    if not path:
        return

    fixed_layer = rs.GetString("Layer for fixed/pinned points (do not include spaces)", DEFAULT_FIXED_LAYER)
    if not fixed_layer:
        return

    unfixed_layer = rs.GetString("Layer for unfixed/free points (do not include spaces)", DEFAULT_UNFIXED_LAYER)
    if not unfixed_layer:
        return

    fixed_layer = ensure_layer(fixed_layer, (220, 50, 47))
    unfixed_layer = ensure_layer(unfixed_layer, (45, 160, 75))

    rows = read_msh(path)
    if not rows:
        rs.MessageBox("No valid points were found in the file.", 0, "mesh3d import")
        return

    fixed_count, unfixed_count = add_points(rows, fixed_layer, unfixed_layer)
    rs.MessageBox(
        "Imported {} points.\nFixed: {}\nUnfixed: {}".format(
            fixed_count + unfixed_count,
            fixed_count,
            unfixed_count,
        ),
        0,
        "mesh3d import",
    )


if __name__ == "__main__":
    main()
