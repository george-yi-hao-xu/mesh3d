import type { Edge, MeshData, NormalizedPointCloud, Point3, PointBounds } from "../types";

export function parseMeshData(text: string): MeshData {
  return /^\s*#\s*Format:\s*mesh-v1\b/im.test(text)
    ? parseMeshSnapshot(text)
    : parsePointCloud(text);
}

export function parsePointCloud(text: string): MeshData {
  const points: Point3[] = [];
  const metadata: Record<string, string> = {};

  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line) continue;

    if (line.startsWith("#")) {
      const match = line.match(/^#\s*([^:]+):\s*(.+)$/);
      if (match) {
        metadata[match[1].trim().toLowerCase()] = match[2].trim();
      }
      continue;
    }

    const fields = line.split(/\s+/);
    if (fields.length < 3) continue;

    const x = Number(fields[0]);
    const y = Number(fields[1]);
    const z = Number(fields[2]);
    if (![x, y, z].every(Number.isFinite)) continue;

    points.push({
      x,
      y,
      z,
      fixed: fields[3] === "1",
      mass: fields.length >= 5 ? Number(fields[4]) : 1,
    });
  }

  if (points.length === 0) {
    throw new Error("no valid point rows found");
  }

  return { points, edges: [], metadata };
}

function parseMeshSnapshot(text: string): MeshData {
  const points: Point3[] = [];
  const edges: Edge[] = [];
  const metadata: Record<string, string> = {};
  let section = "";

  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line) continue;

    if (line.startsWith("#")) {
      const match = line.match(/^#\s*([^:]+):\s*(.+)$/);
      if (match) {
        metadata[match[1].trim().toLowerCase()] = match[2].trim();
      }
      continue;
    }

    const lower = line.toLowerCase();
    if (lower === "vertices" || lower === "edges") {
      section = lower;
      continue;
    }

    const fields = line.split(/\s+/);
    if (section === "vertices") {
      if (fields.length < 6) {
        throw new Error("invalid mesh vertex row");
      }
      const index = Number(fields[0]);
      const x = Number(fields[1]);
      const y = Number(fields[2]);
      const z = Number(fields[3]);
      const mass = Number(fields[5]);
      if (![index, x, y, z, mass].every(Number.isFinite) || !Number.isInteger(index)) {
        throw new Error("invalid mesh vertex value");
      }
      if (index !== points.length) {
        throw new Error("mesh vertices must be stored in index order");
      }
      points.push({
        x,
        y,
        z,
        fixed: fields[4] === "1",
        mass,
      });
      continue;
    }

    if (section === "edges") {
      if (fields.length < 2) {
        throw new Error("invalid mesh edge row");
      }
      const a = Number(fields[0]);
      const b = Number(fields[1]);
      const restLength = fields.length >= 3 ? Number(fields[2]) : 0;
      const stiffness = fields.length >= 4 ? Number(fields[3]) : 0;
      if (![a, b, restLength, stiffness].every(Number.isFinite) || !Number.isInteger(a) || !Number.isInteger(b)) {
        throw new Error("invalid mesh edge value");
      }
      edges.push({ a, b, restLength, stiffness });
    }
  }

  if (points.length === 0) {
    throw new Error("no valid mesh vertices found");
  }
  for (const edge of edges) {
    if (edge.a < 0 || edge.a >= points.length || edge.b < 0 || edge.b >= points.length) {
      throw new Error("mesh edge references an invalid vertex");
    }
  }

  return { points, edges, metadata };
}

export function computePointBounds(pointSets: Point3[][]): PointBounds | null {
  const bounds: PointBounds = {
    minX: Infinity,
    minY: Infinity,
    minZ: Infinity,
    maxX: -Infinity,
    maxY: -Infinity,
    maxZ: -Infinity,
  };
  let count = 0;

  for (const points of pointSets) {
    for (const point of points || []) {
      bounds.minX = Math.min(bounds.minX, point.x);
      bounds.minY = Math.min(bounds.minY, point.y);
      bounds.minZ = Math.min(bounds.minZ, point.z);
      bounds.maxX = Math.max(bounds.maxX, point.x);
      bounds.maxY = Math.max(bounds.maxY, point.y);
      bounds.maxZ = Math.max(bounds.maxZ, point.z);
      count += 1;
    }
  }

  return count > 0 ? bounds : null;
}

export function pointBoundsScale(bounds: PointBounds): number {
  const maxSpan = Math.max(
    Math.abs(bounds.minX),
    Math.abs(bounds.maxX),
    Math.abs(bounds.minY),
    Math.abs(bounds.maxY),
    Math.abs(bounds.minZ),
    Math.abs(bounds.maxZ),
    1,
  );
  return 16 / maxSpan;
}

export function normalizePoints(points: Point3[], referenceBounds: PointBounds | null = null): NormalizedPointCloud {
  const bounds = referenceBounds || computePointBounds([points]) || {
    minX: 0,
    minY: 0,
    minZ: 0,
    maxX: 0,
    maxY: 0,
    maxZ: 0,
  };

  const scale = pointBoundsScale(bounds);
  const normalizedPoints = points.map((point) => ({
    x: point.x * scale,
    y: point.y * scale,
    z: point.z * scale,
    fixed: point.fixed,
    mass: point.mass,
  }));

  const normalizedBounds = normalizedPoints.reduce((acc, point) => ({
    min: {
      x: Math.min(acc.min.x, point.x),
      y: Math.min(acc.min.y, point.y),
      z: Math.min(acc.min.z, point.z),
    },
    max: {
      x: Math.max(acc.max.x, point.x),
      y: Math.max(acc.max.y, point.y),
      z: Math.max(acc.max.z, point.z),
    },
  }), {
    min: { x: Infinity, y: Infinity, z: Infinity },
    max: { x: -Infinity, y: -Infinity, z: -Infinity },
  });

  return {
    points: normalizedPoints,
    min: {
      x: Math.min(normalizedBounds.min.x, 0),
      y: Math.min(normalizedBounds.min.y, 0),
      z: Math.min(normalizedBounds.min.z, 0),
    },
    max: {
      x: Math.max(normalizedBounds.max.x, 0),
      y: Math.max(normalizedBounds.max.y, 0),
      z: Math.max(normalizedBounds.max.z, 0),
    },
    span: 16,
  };
}
