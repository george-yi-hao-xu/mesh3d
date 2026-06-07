import { makeAutoObservable } from "mobx";
import { ThreeMeshViewer } from "../lib/three-viewer";
import type { MeshData, MeshFrame } from "../types";

export class ViewerStore {
  viewer: ThreeMeshViewer | null = null;
  message = "No mesh loaded.";
  messageHidden = false;
  disabled = false;

  constructor() {
    makeAutoObservable(this, { viewer: false });
  }

  mount(container: HTMLElement): void {
    if (this.viewer) return;
    try {
      this.message = "Loading Three.js.";
      this.messageHidden = false;
      this.viewer = new ThreeMeshViewer(container);
      if (!this.message || this.message === "Loading Three.js.") {
        this.message = "No mesh loaded.";
      }
    } catch (error) {
      console.error(error);
      this.disabled = true;
      this.message = "Could not load Three.js. Raw mesh text is shown below.";
      this.messageHidden = false;
    }
  }

  unmount(): void {
    this.viewer?.dispose();
    this.viewer = null;
  }

  render(pointCloud: MeshData, options: { jobId?: string | null; frames?: MeshFrame[] } = {}): void {
    if (this.disabled) {
      this.message = "3D view unavailable. Raw mesh text is shown below.";
      this.messageHidden = false;
      return;
    }
    if (!this.viewer) return;

    try {
      this.viewer.renderPointCloud(pointCloud, options);
      this.message = "";
      this.messageHidden = true;
    } catch (error) {
      this.message = `Could not visualize this mesh: ${error instanceof Error ? error.message : "unknown error"}`;
      this.messageHidden = false;
    }
  }

  setMessage(message: string, hidden = false): void {
    this.message = message;
    this.messageHidden = hidden;
  }

  clear(message = "No mesh loaded."): void {
    this.viewer?.clear();
    this.message = message;
    this.messageHidden = false;
  }
}
