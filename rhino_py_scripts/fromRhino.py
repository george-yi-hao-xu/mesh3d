"""Export selected Rhino point objects to a mesh3d .msh point cloud file.

Workflow:
1. Select free/unfixed Rhino points.
2. Select fixed/pinned Rhino points.
3. Pick a save path.

The mesh3d app uses Y as the up axis, while Rhino commonly uses Z as up.
Keep USE_RHINO_Z_UP_MAPPING enabled to preserve the same visual orientation
between Rhino and mesh3d.
"""

from __future__ import print_function

import os

import rhinoscriptsyntax as rs


USE_RHINO_Z_UP_MAPPING = True
DEFAULT_MASS = 1.0


def rhino_to_mesh_point(point):
    if USE_RHINO_Z_UP_MAPPING:
        return point.X, point.Z, point.Y
    return point.X, point.Y, point.Z


def format_float(value):
    text = "{:.6f}".format(float(value))
    return text.rstrip("0").rstrip(".") if "." in text else text


def unique_ids(ids):
    seen = set()
    result = []
    for object_id in ids or []:
        key = str(object_id)
        if key in seen:
            continue
        seen.add(key)
        result.append(object_id)
    return result


def collect_rows(unfixed_ids, fixed_ids):
    fixed_keys = set(str(object_id) for object_id in fixed_ids)
    rows = []

    for object_id in unfixed_ids:
        if str(object_id) in fixed_keys:
            continue

        point = rs.PointCoordinates(object_id)
        if point is None:
            continue

        x, y, z = rhino_to_mesh_point(point)
        rows.append((x, y, z, 0, DEFAULT_MASS))

    for object_id in fixed_ids:
        point = rs.PointCoordinates(object_id)
        if point is None:
            continue

        x, y, z = rhino_to_mesh_point(point)
        rows.append((x, y, z, 1, DEFAULT_MASS))

    return rows


def write_msh(path, rows):
    with open(path, "w") as file:
        file.write("# Exported from Rhino for mesh3d\n")
        file.write("# Format: x y z fixed mass\n")
        if USE_RHINO_Z_UP_MAPPING:
            file.write("# Axis mapping: Rhino XYZ -> mesh3d XZY\n")
        else:
            file.write("# Axis mapping: Rhino XYZ -> mesh3d XYZ\n")

        for x, y, z, fixed, mass in rows:
            file.write(
                "{} {} {} {} {}\n".format(
                    format_float(x),
                    format_float(y),
                    format_float(z),
                    fixed,
                    format_float(mass),
                )
            )


def main():
    unfixed_ids = rs.GetObjects(
        "Select unfixed/free point objects",
        rs.filter.point,
        preselect=True,
        select=False,
    )
    if unfixed_ids is None:
        return

    fixed_ids = rs.GetObjects(
        "Select fixed/pinned point objects",
        rs.filter.point,
        preselect=False,
        select=False,
    )
    if fixed_ids is None:
        return

    unfixed_ids = unique_ids(unfixed_ids)
    fixed_ids = unique_ids(fixed_ids)

    default_name = "rhino_cloud.msh"
    path = rs.SaveFileName(
        "Save mesh3d point cloud",
        "Mesh point cloud (*.msh)|*.msh||",
        None,
        default_name,
        ".msh",
    )
    if not path:
        return

    if os.path.splitext(path)[1] == "":
        path += ".msh"

    rows = collect_rows(unfixed_ids, fixed_ids)
    if not rows:
        rs.MessageBox("No valid point objects were selected.", 0, "mesh3d export")
        return

    write_msh(path, rows)
    rs.MessageBox(
        "Exported {} points to:\n{}".format(len(rows), path),
        0,
        "Success: mesh3d exported",
    )


if __name__ == "__main__":
    main()
