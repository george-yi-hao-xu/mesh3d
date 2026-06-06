/**
 * @typedef {{ x: number, y: number, z: number, fixed: boolean, mass: number }} Point3
 * @typedef {{ points: Point3[], metadata: Record<string, string> }} PointCloud
 * @typedef {{ points: Point3[], min: { x: number, y: number, z: number }, max: { x: number, y: number, z: number }, span: number }} NormalizedPointCloud
 */

/**
 * Parses the solver's text `.msh` point-cloud format.
 *
 * @param {string} text
 * @returns {PointCloud}
 * @throws {Error} When no valid point rows are found.
 */
export function parsePointCloud(text) {
  const points = [];
  const metadata = {};

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

  return { points, metadata };
}

/**
 * Scales points into a stable Three.js viewing range while preserving the true `(0, 0, 0)` origin.
 *
 * @param {Point3[]} points
 * @returns {NormalizedPointCloud}
 */
export function normalizePoints(points) {
  const bounds = points.reduce((acc, point) => ({
    minX: Math.min(acc.minX, point.x),
    minY: Math.min(acc.minY, point.y),
    minZ: Math.min(acc.minZ, point.z),
    maxX: Math.max(acc.maxX, point.x),
    maxY: Math.max(acc.maxY, point.y),
    maxZ: Math.max(acc.maxZ, point.z),
  }), {
    minX: Infinity,
    minY: Infinity,
    minZ: Infinity,
    maxX: -Infinity,
    maxY: -Infinity,
    maxZ: -Infinity,
  });

  const maxSpan = Math.max(
    Math.abs(bounds.minX),
    Math.abs(bounds.maxX),
    Math.abs(bounds.minY),
    Math.abs(bounds.maxY),
    Math.abs(bounds.minZ),
    Math.abs(bounds.maxZ),
    1,
  );
  const scale = 16 / maxSpan;

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

/**
 * Infers local line segments from nearest neighbors because solver `.msh` outputs store points but not spring endpoints.
 *
 * @param {Point3[]} points
 * @returns {Array<[number, number]>}
 */
export function inferEdges(points) {
  if (points.length < 2 || points.length > 2500) return [];

  const nearestDistances = [];
  for (let i = 0; i < points.length; i++) {
    let nearest = Infinity;
    for (let j = 0; j < points.length; j++) {
      if (i === j) continue;
      nearest = Math.min(nearest, squaredDistance(points[i], points[j]));
    }
    if (Number.isFinite(nearest)) nearestDistances.push(Math.sqrt(nearest));
  }

  nearestDistances.sort((a, b) => a - b);
  const medianNearest = nearestDistances[Math.floor(nearestDistances.length / 2)] || 1;
  const maxDistSq = (medianNearest * 3.5) ** 2;
  const edges = new Set();

  for (let i = 0; i < points.length; i++) {
    const candidates = [];
    for (let j = 0; j < points.length; j++) {
      if (i === j) continue;
      const distSq = squaredDistance(points[i], points[j]);
      if (distSq <= maxDistSq) candidates.push({ index: j, distSq });
    }

    candidates.sort((a, b) => a.distSq - b.distSq);
    for (const candidate of candidates.slice(0, 4)) {
      const a = Math.min(i, candidate.index);
      const b = Math.max(i, candidate.index);
      edges.add(`${a}:${b}`);
    }
  }

  return Array.from(edges, (edge) => edge.split(":").map(Number));
}

/**
 * Computes squared Euclidean distance without allocating temporary vectors.
 *
 * @param {Point3} a
 * @param {Point3} b
 * @returns {number}
 */
function squaredDistance(a, b) {
  return ((a.x - b.x) ** 2) + ((a.y - b.y) ** 2) + ((a.z - b.z) ** 2);
}
