import { makeAutoObservable, runInAction } from "mobx";
import {
  addJobToTrainingCluster,
  createTrainingCluster,
  deleteTrainingCluster,
  listTrainingClusters,
  recommendClusterConfig,
  removeJobFromTrainingCluster,
  trainCluster,
  updateTrainingCluster,
} from "../lib/api";
import type { AppError, SolverConfig, TrainingCluster } from "../types";
import type { RootStore } from "./root-store";

export class LearningStore {
  readonly root: RootStore;
  clusters: TrainingCluster[] = [];
  activeClusterId: string | null = null;
  loading = false;
  saving = false;
  training = false;
  recommending = false;
  error = "";

  constructor(root: RootStore) {
    this.root = root;
    makeAutoObservable(this, { root: false });
  }

  get activeCluster(): TrainingCluster | null {
    if (!this.activeClusterId) return this.clusters[0] || null;
    return this.clusters.find((cluster) => cluster.id === this.activeClusterId) || this.clusters[0] || null;
  }

  get reviewedJobs() {
    return this.root.jobs.jobs.filter((job) => Boolean(job.review));
  }

  get activeClusterJobIds(): Set<string> {
    return new Set((this.activeCluster?.jobs || []).map((item) => item.job.id));
  }

  async refreshClusters(): Promise<void> {
    this.loading = true;
    this.error = "";
    try {
      const clusters = await listTrainingClusters();
      runInAction(() => {
        this.clusters = clusters;
        if (!this.activeClusterId && clusters.length > 0) {
          this.activeClusterId = clusters[0].id;
        }
      });
    } catch (error) {
      if (this.root.auth.handleAuthError(error as AppError)) return;
      runInAction(() => {
        this.error = error instanceof Error ? error.message : "Could not load training clusters.";
      });
    } finally {
      runInAction(() => {
        this.loading = false;
      });
    }
  }

  selectCluster(clusterId: string): void {
    this.activeClusterId = clusterId;
  }

  async createCluster(name: string): Promise<void> {
    this.saving = true;
    this.error = "";
    try {
      const cluster = await createTrainingCluster(name);
      runInAction(() => {
        this.upsertCluster(cluster);
        this.activeClusterId = cluster.id;
      });
    } catch (error) {
      this.captureError(error, "Could not create training cluster.");
    } finally {
      runInAction(() => {
        this.saving = false;
      });
    }
  }

  async renameActive(name: string): Promise<void> {
    const cluster = this.activeCluster;
    if (!cluster) return;
    this.saving = true;
    this.error = "";
    try {
      const updated = await updateTrainingCluster(cluster.id, name);
      runInAction(() => this.upsertCluster(updated));
    } catch (error) {
      this.captureError(error, "Could not rename training cluster.");
    } finally {
      runInAction(() => {
        this.saving = false;
      });
    }
  }

  async deleteActive(): Promise<void> {
    const cluster = this.activeCluster;
    if (!cluster) return;
    this.saving = true;
    this.error = "";
    try {
      await deleteTrainingCluster(cluster.id);
      runInAction(() => {
        this.clusters = this.clusters.filter((item) => item.id !== cluster.id);
        this.activeClusterId = this.clusters[0]?.id || null;
      });
    } catch (error) {
      this.captureError(error, "Could not delete training cluster.");
    } finally {
      runInAction(() => {
        this.saving = false;
      });
    }
  }

  async toggleJob(jobId: string): Promise<void> {
    const cluster = this.activeCluster;
    if (!cluster) return;
    this.saving = true;
    this.error = "";
    try {
      const updated = this.activeClusterJobIds.has(jobId)
        ? await removeJobFromTrainingCluster(cluster.id, jobId)
        : await addJobToTrainingCluster(cluster.id, jobId);
      runInAction(() => this.upsertCluster(updated));
    } catch (error) {
      this.captureError(error, "Could not update cluster jobs.");
    } finally {
      runInAction(() => {
        this.saving = false;
      });
    }
  }

  async trainActive(): Promise<void> {
    const cluster = this.activeCluster;
    if (!cluster) return;
    this.training = true;
    this.error = "";
    try {
      await trainCluster(cluster.id);
      await this.refreshClusters();
    } catch (error) {
      this.captureError(error, "Could not train model.");
      await this.refreshClusters().catch(() => {});
    } finally {
      runInAction(() => {
        this.training = false;
      });
    }
  }

  async recommendForSelectedUpload(): Promise<void> {
    const cluster = this.activeCluster;
    const upload = this.root.warehouse.selectedUpload;
    if (!cluster || !upload) {
      this.error = "Select a mesh in the Solver tab before requesting recommendations.";
      return;
    }
    this.recommending = true;
    this.error = "";
    try {
      const updated = await recommendClusterConfig(cluster.id, upload.id);
      runInAction(() => this.upsertCluster(updated));
    } catch (error) {
      this.captureError(error, "Could not recommend configs.");
    } finally {
      runInAction(() => {
        this.recommending = false;
      });
    }
  }

  applyConfig(config: Record<string, unknown>): void {
    const next = { ...this.root.preview.config };
    for (const key of Object.keys(next) as Array<keyof SolverConfig>) {
      const value = config[key];
      if (typeof value === "number" && Number.isFinite(value)) {
        next[key] = value;
      }
    }
    this.root.preview.setConfig(next);
  }

  reset(): void {
    this.clusters = [];
    this.activeClusterId = null;
    this.loading = false;
    this.saving = false;
    this.training = false;
    this.recommending = false;
    this.error = "";
  }

  private upsertCluster(cluster: TrainingCluster): void {
    const index = this.clusters.findIndex((item) => item.id === cluster.id);
    if (index >= 0) {
      this.clusters[index] = cluster;
    } else {
      this.clusters = [cluster, ...this.clusters];
    }
  }

  private captureError(error: unknown, fallback: string): void {
    if (this.root.auth.handleAuthError(error as AppError)) return;
    runInAction(() => {
      this.error = error instanceof Error ? error.message : fallback;
    });
  }
}
