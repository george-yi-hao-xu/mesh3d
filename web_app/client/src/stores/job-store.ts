import { makeAutoObservable, runInAction } from "mobx";
import { createJob, deleteJob, fetchJob, fetchMeshData, listJobs } from "../lib/api";
import { formatSeconds, jobTitle, sanitizeDownloadStem } from "../lib/format";
import type { AppError, Job, MeshData, MeshFrame, PreparedMesh } from "../types";
import type { RootStore } from "./root-store";

export enum ViewerMode {
  Empty = "empty",
  Preview = "preview",
  Job = "job",
}

export class JobStore {
  readonly root: RootStore;
  jobs: Job[] = [];
  jobCache = new Map<string, Job>();
  activeJobId: string | null = null;
  activeFrameUrl: string | null = null;
  activeFrames: MeshFrame[] = [];
  activePreview: PreparedMesh | null = null;
  rawPreviewText = "No mesh loaded.";
  submitting = false;
  deleting = false;
  deleteOverlayJobId: string | null = null;
  deleteError = "";
  playbackTimer: number | null = null;
  playback = false;

  constructor(root: RootStore) {
    this.root = root;
    makeAutoObservable(this, { root: false, jobCache: false });
  }

  get activeJob(): Job | null {
    if (!this.activeJobId) return null;
    return this.jobs.find((job) => job.id === this.activeJobId) || this.jobCache.get(this.activeJobId) || null;
  }

  get selectedFrame(): MeshFrame | null {
    if (!this.activeFrameUrl) return null;
    return this.activeFrames.find((frame) => frame.url === this.activeFrameUrl) || null;
  }

  get selectedFrameIndex(): number {
    return Math.max(0, this.activeFrames.findIndex((frame) => frame.url === this.activeFrameUrl));
  }

  get viewerMode(): ViewerMode {
    if (this.activePreview) return ViewerMode.Preview;
    if (this.activeJob) return ViewerMode.Job;
    return ViewerMode.Empty;
  }

  get canToggleGeneratedSprings(): boolean {
    return this.viewerMode === ViewerMode.Preview;
  }

  get reserveSpringDisplay(): boolean {
    return this.viewerMode !== ViewerMode.Empty;
  }

  get springLegendMesh(): MeshData | null {
    if (this.viewerMode === ViewerMode.Preview) {
      return this.activePreview?.mesh || null;
    }
    if (this.viewerMode === ViewerMode.Job) {
      return this.selectedFrame?.pointCloud || this.firstLoadedFrameMesh;
    }
    return null;
  }

  get activeTitle(): string {
    if (this.activePreview) return "Spring preview";
    if (this.activeJob) return jobTitle(this.activeJob);
    return "No job selected";
  }

  get activeMeta(): string {
    if (this.activePreview) {
      return `${this.activePreview.mesh.points.length} points - ${this.activePreview.mesh.edges.length} springs`;
    }
    const job = this.activeJob;
    if (!job) return "Run a solve to see checkpoints.";
    const outcome = job.status === "done"
      ? ` - ${job.converged ? "converged" : "limit reached"} at ${formatSeconds(job.finalTime || 0)}`
      : "";
    return `${job.status}${outcome} - ${job.id}`;
  }

  get activeInputName(): string {
    if (this.activePreview) return `Input file: ${this.activePreview.sourceName}`;
    return this.activeJob?.inputName ? `Input file: ${this.activeJob.inputName}` : "";
  }

  get canDeleteActiveJob(): boolean {
    const job = this.activeJob;
    return Boolean(job && (job.status === "done" || job.status === "failed"));
  }

  get downloadUrl(): string {
    return this.selectedFrame?.url || "";
  }

  get downloadName(): string {
    const job = this.activeJob;
    const frame = this.selectedFrame;
    if (!job || !frame) return "";
    const prefix = sanitizeDownloadStem(jobTitle(job) || "mesh");
    const suffix = frame.url.endsWith("/result") ? "final.mesh" : frame.url.split("/").pop() || "mesh.mesh";
    return `${prefix}_${suffix}`;
  }

  get firstLoadedFrameMesh(): MeshData | null {
    return this.activeFrames.find((frame) => frame.loaded && frame.pointCloud?.edges)?.pointCloud || null;
  }

  async refreshJobs(): Promise<void> {
    const jobs = await listJobs();
    runInAction(() => {
      this.jobs = jobs.slice().sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
      for (const job of this.jobs) {
        this.cacheJob(job);
      }
    });
  }

  async submitJob(): Promise<void> {
    this.submitting = true;
    try {
      this.root.preview.ensureJobName();
      const preview = await this.root.preview.prepareMeshPreview();
      if (preview.mesh.edges.length === 0) {
        throw new Error("Cannot run solver: this mesh has no springs. Enable generated springs or choose a mesh with existing springs.");
      }
      const uploadId = preview.uploadId || (await this.root.warehouse.savePreparedMesh(preview)).id;
      const response = await createJob(uploadId, this.root.preview.jobName.trim(), this.root.preview.config);
      runInAction(() => {
        this.activePreview = null;
        this.activeJobId = response.job.id;
        this.activeFrameUrl = null;
        this.activeFrames = response.frames || [];
        this.cacheJob(response.job);
        this.upsertJob(response.job);
        this.sortJobs();
      });
      this.renderActiveJob(response.job, true);
    } catch (error) {
      if (this.root.auth.handleAuthError(error as AppError)) return;
      throw error;
    } finally {
      runInAction(() => {
        this.submitting = false;
      });
    }
  }

  showPreparedMeshPreview(preview: PreparedMesh): void {
    this.stopPlayback();
    this.activePreview = preview;
    this.activeJobId = null;
    this.activeFrameUrl = null;
    this.activeFrames = [];
    this.rawPreviewText = preview.text;
    this.root.viewer.render(preview.mesh, { jobId: "spring-preview", frames: [] });
  }

  clearPreparedMeshPreview(message: string): void {
    this.stopPlayback();
    this.activePreview = null;
    this.activeJobId = null;
    this.activeFrameUrl = null;
    this.activeFrames = [];
    this.rawPreviewText = message;
    this.root.viewer.clear(message);
  }

  async selectJob(jobId: string): Promise<void> {
    this.stopPlayback();
    runInAction(() => {
      this.activePreview = null;
      this.activeJobId = jobId;
      this.activeFrameUrl = null;
      this.activeFrames = [];
    });

    let job: Job;
    try {
      job = await this.getJob(jobId);
    } catch (error) {
      if (this.root.auth.handleAuthError(error as AppError)) return;
      throw error;
    }

    runInAction(() => {
      this.upsertJob(job);
    });
    this.renderActiveJob(job, true);
  }

  renderActiveJob(job: Job, autoSelectFrame = false): void {
    this.stopPlayback();
    if (!job) return;
    const frames = this.getFrames(job);

    if (frames.length === 0) {
      runInAction(() => {
        this.rawPreviewText = job.status === "failed"
          ? `Job failed: ${job.error || "unknown error"}`
          : "Waiting for first checkpoint.";
      });
      this.root.viewer.clear(this.rawPreviewText);
      return;
    }

    const selectedFrame = this.chooseFrame(frames, autoSelectFrame);
    if (autoSelectFrame || !this.activeFrameUrl) {
      void this.selectFrame(selectedFrame.url);
    } else {
      this.root.viewer.setMessage("", true);
    }
  }

  async selectFrame(url: string): Promise<void> {
    const jobIdAtRequest = this.activeJobId;
    const job = this.activeJob;
    runInAction(() => {
      this.activeFrameUrl = url;
    });
    const frame = this.activeFrames.find((item) => item.url === url);
    if (!frame) return;

    const loadedError = frame.error as AppError | null;
    if (loadedError) {
      if (this.root.auth.handleAuthError(loadedError)) return;
      this.root.viewer.setMessage(loadedError.message);
      runInAction(() => {
        this.rawPreviewText = loadedError.message;
      });
      return;
    }

    if (!frame.loaded) {
      this.root.viewer.setMessage(this.root.viewer.viewer ? "" : `Loading ${frame.label}.`, Boolean(this.root.viewer.viewer));
      await this.loadFrame(frame);
    }
    if (this.activeJobId !== jobIdAtRequest || this.activeFrameUrl !== url) return;

    if (frame.error) {
      if (this.root.auth.handleAuthError(frame.error)) return;
      this.root.viewer.setMessage(frame.error.message);
      runInAction(() => {
        this.rawPreviewText = frame.error?.message || "";
      });
      return;
    }

    runInAction(() => {
      this.rawPreviewText = frame.text;
    });

    if (frame.pointCloud) {
      this.root.viewer.render(frame.pointCloud, {
        jobId: this.activeJobId,
        frames: this.activeFrames,
      });
    } else if (job) {
      this.root.viewer.setMessage(`Could not visualize ${jobTitle(job)}.`);
    }
  }

  selectFrameAt(index: number): void {
    if (this.activeFrames.length === 0) return;
    const selectedIndex = Math.max(0, Math.min(this.activeFrames.length - 1, index));
    void this.selectFrame(this.activeFrames[selectedIndex].url);
  }

  startPlayback(): void {
    if (this.activeFrames.length < 2) return;
    if (this.selectedFrameIndex >= this.activeFrames.length - 1) {
      this.selectFrameAt(0);
    }

    this.playback = true;
    const tick = async () => {
      if (!this.playback) return;
      const currentIndex = this.selectedFrameIndex;
      if (currentIndex >= this.activeFrames.length - 1) {
        this.stopPlayback();
        return;
      }

      const nextIndex = currentIndex + 1;
      await this.selectFrame(this.activeFrames[nextIndex].url);

      if (nextIndex >= this.activeFrames.length - 1) {
        this.stopPlayback();
        return;
      }
      if (this.playback) {
        this.playbackTimer = window.setTimeout(tick, 50);
      }
    };
    this.playbackTimer = window.setTimeout(tick, 50);
  }

  stopPlayback(): void {
    if (this.playbackTimer !== null) {
      window.clearTimeout(this.playbackTimer);
      this.playbackTimer = null;
    }
    this.playback = false;
  }

  togglePlayback(): void {
    if (this.playback) {
      this.stopPlayback();
    } else {
      this.startPlayback();
    }
  }

  openDeleteOverlay(jobId = this.activeJobId): void {
    if (!jobId) return;
    this.deleteOverlayJobId = jobId;
    this.deleteError = "";
  }

  closeDeleteOverlay(): void {
    this.deleteOverlayJobId = null;
    this.deleteError = "";
  }

  async confirmDelete(): Promise<void> {
    if (!this.deleteOverlayJobId) return;
    this.deleting = true;
    try {
      await deleteJob(this.deleteOverlayJobId);
      runInAction(() => {
        this.removeJob(this.deleteOverlayJobId);
        this.closeDeleteOverlay();
        this.clearActiveJobView();
      });
    } catch (error) {
      if (this.root.auth.handleAuthError(error as AppError)) {
        this.closeDeleteOverlay();
        return;
      }
      runInAction(() => {
        this.deleteError = error instanceof Error ? error.message : "Could not delete job.";
      });
    } finally {
      runInAction(() => {
        this.deleting = false;
      });
    }
  }

  reset(): void {
    this.stopPlayback();
    this.jobs = [];
    this.jobCache.clear();
    this.clearActiveJobView();
    this.deleteOverlayJobId = null;
    this.deleteError = "";
    this.submitting = false;
    this.deleting = false;
  }

  clearActiveJobView(): void {
    this.stopPlayback();
    this.activeJobId = null;
    this.activeFrameUrl = null;
    this.activeFrames = [];
    this.activePreview = null;
    this.rawPreviewText = "No job selected.";
    this.root.viewer.clear("No job selected.");
  }

  private upsertJob(job: Job): void {
    const index = this.jobs.findIndex((item) => item.id === job.id);
    if (index >= 0) {
      this.jobs[index] = job;
    } else {
      this.jobs.unshift(job);
    }
    this.cacheJob(job);
  }

  private removeJob(jobId: string | null): void {
    if (!jobId) return;
    this.jobs = this.jobs.filter((job) => job.id !== jobId);
    this.jobCache.delete(jobId);
  }

  private async getJob(jobId: string): Promise<Job> {
    const cached = this.jobCache.get(jobId);
    if (cached && isFinishedJob(cached)) {
      return cached;
    }

    const job = await fetchJob(jobId);
    runInAction(() => {
      this.cacheJob(job);
    });
    return job;
  }

  private cacheJob(job: Job | null | undefined): void {
    if (job?.id) {
      this.jobCache.set(job.id, job);
    }
  }

  private sortJobs(): void {
    this.jobs = this.jobs.slice().sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
  }

  private getFrames(job: Job): MeshFrame[] {
    const frames: MeshFrame[] = (job.snapshots || []).map((snapshot) => ({
      label: snapshot.label,
      url: snapshot.url,
      text: "",
      pointCloud: null,
      loaded: false,
      loading: false,
      error: null,
      request: null,
      simTime: snapshot.simTime,
      step: snapshot.step,
    }));
    if (job.resultUrl) {
      frames.push({
        label: "Final",
        url: job.resultUrl,
        text: "",
        isFinal: true,
        pointCloud: null,
        loaded: false,
        loading: false,
        error: null,
        request: null,
      });
    }
    return this.syncActiveFrames(frames);
  }

  private syncActiveFrames(frames: MeshFrame[]): MeshFrame[] {
    const currentByUrl = new Map(this.activeFrames.map((frame) => [frame.url, frame]));
    this.activeFrames = frames.map((frame) => {
      const current = currentByUrl.get(frame.url);
      if (current) {
        current.label = frame.label;
        current.simTime = frame.simTime;
        current.step = frame.step;
        return current;
      }
      return frame;
    });
    return this.activeFrames;
  }

  private chooseFrame(frames: MeshFrame[], autoSelectFrame: boolean): MeshFrame {
    if (!autoSelectFrame && this.activeFrameUrl) {
      const currentFrame = frames.find((frame) => frame.url === this.activeFrameUrl);
      if (currentFrame) return currentFrame;
    }
    return frames[frames.length - 1];
  }

  private async loadFrame(frame: MeshFrame): Promise<MeshFrame> {
    if (frame.loaded || frame.error) return frame;
    if (!frame.request) {
      runInAction(() => {
        frame.loading = true;
      });
      frame.request = fetchMeshData(frame.url).then((mesh) => {
        runInAction(() => {
          frame.text = mesh.text;
          frame.pointCloud = mesh.pointCloud;
          frame.loaded = true;
        });
        return frame;
      }).catch((error) => {
        runInAction(() => {
          frame.error = error as AppError;
        });
        throw error;
      }).finally(() => {
        runInAction(() => {
          frame.loading = false;
        });
      });
    }

    try {
      await frame.request;
    } catch {
      // The frame stores its own error; callers decide whether to surface it.
    }
    return frame;
  }
}

function isFinishedJob(job: Job): boolean {
  return job.status === "done" || job.status === "failed";
}
