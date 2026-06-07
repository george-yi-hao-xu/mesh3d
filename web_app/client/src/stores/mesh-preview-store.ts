import { makeAutoObservable, runInAction } from "mobx";
import { defaultJobName, sanitizeDownloadStem } from "../lib/format";
import { parseMeshData } from "../lib/mesh-parser";
import { generateSprings, serializeMeshV1 } from "../lib/mesh-topology";
import type { PreparedMesh, SolverConfig } from "../types";
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
  file: File | null = null;
  jobName = "";
  jobNameEdited = false;
  config: SolverConfig = { ...defaultConfig };
  status = "Choose a point cloud to preview springs.";
  preparedMesh: PreparedMesh | null = null;
  previewRequestId = 0;

  constructor(root: RootStore) {
    this.root = root;
    makeAutoObservable(this, { root: false });
  }

  setFile(file: File | null): void {
    this.file = file;
    if (!this.jobNameEdited || !this.jobName.trim()) {
      this.jobName = defaultJobName(file?.name);
      this.jobNameEdited = false;
    }
    void this.previewInput();
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

  ensureJobName(): void {
    if (!this.jobName.trim()) {
      this.jobName = defaultJobName(this.file?.name);
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
        this.status = `${preview.mesh.edges.length} springs generated from ${preview.mesh.points.length} points.`;
      });
      this.root.jobs.showPreparedMeshPreview(preview);
    } catch (error) {
      if (requestId !== this.previewRequestId) return;
      runInAction(() => {
        this.preparedMesh = null;
        this.status = error instanceof Error ? error.message : "Could not preview springs.";
      });
      this.root.viewer.clear(this.status);
    }
  }

  async prepareMeshPreview(): Promise<PreparedMesh> {
    const file = this.file;
    if (!file) {
      throw new Error("Choose a point cloud to preview springs.");
    }

    const sourceText = await file.text();
    const parsed = parseMeshData(sourceText);
    const edges = generateSprings(parsed.points, this.config);
    if (edges.length === 0) {
      throw new Error("No springs generated. Increase max distance, max springs, or connect probability.");
    }

    const mesh = {
      points: parsed.points,
      edges,
      metadata: {
        source: file.name,
        springs: String(edges.length),
      },
    };
    const text = serializeMeshV1(mesh);
    return { mesh, text, sourceName: file.name };
  }

  generatedMeshFileName(): string {
    return `${sanitizeDownloadStem(this.file?.name || "mesh")}_springs.mesh`;
  }

  reset(): void {
    this.file = null;
    this.jobName = "";
    this.jobNameEdited = false;
    this.config = { ...defaultConfig };
    this.status = "Choose a point cloud to preview springs.";
    this.preparedMesh = null;
    this.previewRequestId++;
  }
}
