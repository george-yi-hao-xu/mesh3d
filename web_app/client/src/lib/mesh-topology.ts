import type { Edge, MeshData, Point3, SolverConfig } from "../types";

export function generateSprings(points: Point3[], config: Pick<SolverConfig, "springSeed" | "maxSpringDist" | "maxSpringsPerParticle" | "springConnectProb" | "stiffness">): Edge[] {
  const maxDist = Number(config.maxSpringDist);
  const maxPerParticle = Math.floor(Number(config.maxSpringsPerParticle));
  const prob = Math.min(1, Math.max(0, Number(config.springConnectProb)));
  const stiffness = Number(config.stiffness);
  const seed = Math.floor(Number(config.springSeed) || 0);

  if (!Array.isArray(points) || points.length < 2 || maxDist <= 0 || maxPerParticle <= 0 || prob <= 0) {
    return [];
  }

  const maxDistSq = maxDist * maxDist;
  const candidates: Array<Array<{ index: number; dist: number }>> = Array.from({ length: points.length }, () => []);
  const connectionCount = Array(points.length).fill(0) as number[];

  for (let i = 0; i < points.length; i++) {
    for (let j = i + 1; j < points.length; j++) {
      const distSq = squaredDistance(points[i], points[j]);
      if (distSq <= maxDistSq && distSq > 1e-8) {
        candidates[i].push({ index: j, dist: Math.sqrt(distSq) });
      }
    }
  }

  for (let i = 0; i < candidates.length; i++) {
    shuffle(candidates[i], seededRandom(seed + i));
  }

  const random = seededRandom(seed);
  const edges: Edge[] = [];
  for (let i = 0; i < candidates.length; i++) {
    for (const candidate of candidates[i]) {
      const j = candidate.index;
      if (connectionCount[i] >= maxPerParticle || connectionCount[j] >= maxPerParticle) {
        continue;
      }
      if (random() <= prob) {
        edges.push({
          a: i,
          b: j,
          restLength: candidate.dist,
          stiffness,
        });
        connectionCount[i]++;
        connectionCount[j]++;
      }
    }
  }
  return edges;
}

export function serializeMeshV1(mesh: Pick<MeshData, "points" | "edges">): string {
  const lines = [
    "# Mesh3D browser generated mesh",
    "# Format: mesh-v1",
    `# Vertices: ${mesh.points.length}`,
    `# Edges: ${mesh.edges.length}`,
    "",
    "vertices",
    "# index x y z fixed mass",
  ];

  mesh.points.forEach((point, index) => {
    lines.push([
      index,
      formatNumber(point.x),
      formatNumber(point.y),
      formatNumber(point.z),
      point.fixed ? 1 : 0,
      formatNumber(Number.isFinite(point.mass) && point.mass > 0 ? point.mass : 1),
    ].join(" "));
  });

  lines.push("", "edges", "# a_index b_index rest_length stiffness");
  mesh.edges.forEach((edge) => {
    lines.push([
      edge.a,
      edge.b,
      formatNumber(edge.restLength),
      formatNumber(edge.stiffness),
    ].join(" "));
  });
  lines.push("");
  return lines.join("\n");
}

function formatNumber(value: number): string {
  return Number(value).toFixed(6);
}

function shuffle<T>(items: T[], random: () => number): void {
  for (let i = items.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1));
    [items[i], items[j]] = [items[j], items[i]];
  }
}

function seededRandom(seed: number): () => number {
  let state = seed >>> 0;
  return () => {
    state += 0x6d2b79f5;
    let value = state;
    value = Math.imul(value ^ (value >>> 15), value | 1);
    value ^= value + Math.imul(value ^ (value >>> 7), value | 61);
    return ((value ^ (value >>> 14)) >>> 0) / 4294967296;
  };
}

function squaredDistance(a: Point3, b: Point3): number {
  return ((a.x - b.x) ** 2) + ((a.y - b.y) ** 2) + ((a.z - b.z) ** 2);
}
