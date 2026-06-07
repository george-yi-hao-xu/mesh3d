import { AuthStore } from "./auth-store";
import { JobStore } from "./job-store";
import { MeshPreviewStore } from "./mesh-preview-store";
import { ViewerStore } from "./viewer-store";

export class RootStore {
  readonly auth: AuthStore;
  readonly preview: MeshPreviewStore;
  readonly viewer: ViewerStore;
  readonly jobs: JobStore;

  constructor() {
    this.viewer = new ViewerStore();
    this.preview = new MeshPreviewStore(this);
    this.auth = new AuthStore(this);
    this.jobs = new JobStore(this);
  }
}
