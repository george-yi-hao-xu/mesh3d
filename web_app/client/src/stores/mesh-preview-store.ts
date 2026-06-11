import { computed, makeAutoObservable, runInAction } from "mobx";
import { defaultJobName, sanitizeDownloadStem } from "../lib/format";
import { generateSprings, serializeMeshV1 } from "../lib/mesh-topology";
import type { MeshData, PreparedMesh, SolverConfig, Upload } from "../types";
import type { RootStore } from "./root-store";

export const defaultConfig: SolverConfig = {
  stiffness: 10,
  dampingFactor: 0.1,
  gravity: -4.9,
  airResistanceFactor: 0.001,
  timeStep: 0.01,
  snapshotInterval: 0.05,
  maxSimTime: 120,
  maxSteps: 20000,
  velocityEpsilon: 0.001,
  positionEpsilon: 0.001,
  stableFrames: 60,
  springSeed: 42,
  maxSpringDist: 1.5,
  maxSpringsPerParticle: 4,
  springConnectProb: 0.8,
};

export class MeshPreviewStore {
  readonly root: RootStore;
  sourceUpload: Upload | null = null;
  sourceText = "";
  sourceMesh: MeshData | null = null;
  jobName = "";
  jobNameEdited = false;
  config: SolverConfig = { ...defaultConfig };
  status = "Pick a mesh to preview springs.";
  preparedMesh: PreparedMesh | null = null;
  previewRequestId = 0;
  includeGeneratedSprings = true;

  constructor(root: RootStore) {
    this.root = root;
    makeAutoObservable(this, { root: false });
  }

  setWarehouseMesh(upload: Upload, text: string, mesh: MeshData): void {
    this.sourceUpload = upload;
    this.sourceText = text;
    this.sourceMesh = mesh;
    if (!this.jobNameEdited || !this.jobName.trim()) {
      this.jobName = defaultJobName(upload.fileName);
      this.jobNameEdited = false;
    }
    void this.previewInput();
  }

  setSavedGeneratedMesh(upload: Upload): void {
    if (!this.preparedMesh) return;
    this.sourceUpload = upload;
    this.sourceText = this.preparedMesh.text;
    this.sourceMesh = this.preparedMesh.mesh;
    this.preparedMesh = {
      ...this.preparedMesh,
      uploadId: upload.id,
      generated: false,
      sourceName: upload.fileName,
    };
    this.status = `${this.preparedMesh.mesh.points.length} points and ${this.preparedMesh.mesh.edges.length} springs saved to warehouse.`;
  }

  setJobName(value: string): void {
    this.jobName = value;
    this.jobNameEdited = true;
  }

  setConfigValue(key: keyof SolverConfig, value: number): void {
    this.config = {
      ...this.config,
      [key]: value,
    };
    if (["stiffness", "springSeed", "maxSpringDist", "maxSpringsPerParticle", "springConnectProb"].includes(key)) {
      void this.previewInput();
    }
  }

  setConfig(config: SolverConfig): void {
    this.config = { ...config };
    void this.previewInput();
  }

  setIncludeGeneratedSprings(value: boolean): void {
    this.includeGeneratedSprings = value;
    void this.previewInput();
  }

  ensureJobName(): void {
    if (!this.jobName.trim()) {
      this.jobName = defaultJobName(this.sourceUpload?.fileName);
      this.jobNameEdited = false;
    }
  }

  async previewInput(): Promise<void> {
    const requestId = ++this.previewRequestId;
    try {
      const preview = await this.prepareMeshPreview();
      if (requestId !== this.previewRequestId) return;
      runInAction(() => {
        this.preparedMesh = preview;
        this.status = springPreviewStatus(preview);
      });
      this.root.jobs.showPreparedMeshPreview(preview);
    } catch (error) {
      if (requestId !== this.previewRequestId) return;
      runInAction(() => {
        this.preparedMesh = null;
        this.status = error instanceof Error ? error.message : "Could not preview springs.";
      });
      this.root.jobs.clearPreparedMeshPreview(this.status);
    }
  }

  async prepareMeshPreview(): Promise<PreparedMesh> {
    const sourceMesh = this.sourceMesh;
    const sourceUpload = this.sourceUpload;
    if (!sourceMesh || !sourceUpload) {
      throw new Error("Pick a mesh first.");
    }

    const generatedEdges = generateSprings(sourceMesh.points, this.config);
    const existingEdges = sourceMesh.edges.map((edge) => ({ ...edge, origin: edge.origin || ("existing" as const) }));
    const existingPairs = new Set(existingEdges.map(edgeKey));
    const newEdges = this.includeGeneratedSprings
      ? generatedEdges.filter((edge) => !existingPairs.has(edgeKey(edge)))
      : [];

    if (existingEdges.length > 0 && newEdges.length === 0) {
      return {
        mesh: {
          ...sourceMesh,
          edges: existingEdges,
        },
        text: this.sourceText,
        sourceName: sourceUpload.fileName,
        uploadId: sourceUpload.id,
        generated: false,
      };
    }

    const edges = [...existingEdges, ...newEdges];
    const mesh = {
      points: sourceMesh.points,
      edges,
      metadata: {
        source: sourceUpload.fileName,
        springs: String(edges.length),
        existing_springs: String(existingEdges.length),
        generated_springs: String(newEdges.length),
      },
    };
    const text = serializeMeshV1(mesh);
    return { mesh, text, sourceName: sourceUpload.fileName, generated: true };
  }

  generatedMeshFileName(): string {
    return `${sanitizeDownloadStem(this.sourceUpload?.fileName || "mesh")}_springs.mesh`;
  }

  reset(): void {
    this.sourceUpload = null;
    this.sourceText = "";
    this.sourceMesh = null;
    this.jobName = "";
    this.jobNameEdited = false;
    this.config = { ...defaultConfig };
    this.status = "Pick a mesh to preview springs.";
    this.preparedMesh = null;
    this.previewRequestId++;
    this.includeGeneratedSprings = true;
  }

}

function edgeKey(edge: { a: number; b: number }): string {
  const a = Math.min(edge.a, edge.b);
  const b = Math.max(edge.a, edge.b);
  return `${a}:${b}`;
}

function springPreviewStatus(preview: PreparedMesh): string {
  const existingCount = preview.mesh.edges.filter((edge) => edge.origin !== "generated").length;
  const generatedCount = preview.mesh.edges.length - existingCount;
  const parts = [];
  if (existingCount > 0) parts.push(`${existingCount} existing`);
  if (generatedCount > 0) parts.push(`${generatedCount} generated`);
  const springText = parts.length > 0 ? parts.join(" and ") : "0";
  return `${springText} springs from ${preview.mesh.points.length} points.`;
}
