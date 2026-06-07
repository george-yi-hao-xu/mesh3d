import { makeAutoObservable, runInAction } from "mobx";
import { fetchUploadArtifact, listUploads, uploadMeshArtifact } from "../lib/api";
import { parseMeshData } from "../lib/mesh-parser";
import { sanitizeDownloadStem } from "../lib/format";
import type { AppError, MeshData, PreparedMesh, Upload } from "../types";
import type { RootStore } from "./root-store";

export class MeshWarehouseStore {
  readonly root: RootStore;
  uploads: Upload[] = [];
  selectedUpload: Upload | null = null;
  selectedText = "";
  selectedMesh: MeshData | null = null;
  pickerOpen = false;
  loading = false;
  uploading = false;
  savingGenerated = false;
  error = "";

  constructor(root: RootStore) {
    this.root = root;
    makeAutoObservable(this, { root: false });
  }

  get selectedLabel(): string {
    return this.selectedUpload ? this.selectedUpload.fileName : "No mesh selected";
  }

  get canSaveGeneratedMesh(): boolean {
    const preview = this.root.preview.preparedMesh;
    return Boolean(preview?.generated && !preview.uploadId && preview.mesh.edges.length > 0);
  }

  // Refresh the list of uploads from the server, showing loading and error states as needed.
  async refreshUploads(): Promise<void> {
    this.loading = true;
    this.error = "";
    try {
      const uploads = await listUploads();
      runInAction(() => {
        const safeUploads = Array.isArray(uploads) ? uploads : [];
        this.uploads = safeUploads.slice().sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
      });
    } catch (error) {
      if (this.root.auth.handleAuthError(error as AppError)) return;
      runInAction(() => {
        this.error = error instanceof Error ? error.message : "Could not load mesh warehouse.";
      });
    } finally {
      runInAction(() => {
        this.loading = false;
      });
    }
  }

  openPicker(): void {
    this.pickerOpen = true;
    this.error = "";
    void this.refreshUploads();
  }

  closePicker(): void {
    this.pickerOpen = false;
    this.error = "";
  }

  // Upload a new mesh file to the server, then refresh the list and select the new upload if successful. Shows loading and error states as needed.
  async uploadNew(file: File | null | undefined): Promise<void> {
    if (!file) return;

    this.uploading = true;
    this.error = "";

    try {
      const upload = await uploadMeshArtifact(file, file.name, "uploaded");
      await this.refreshUploads();
      await this.selectUpload(upload);
    } catch (error) {
      if (this.root.auth.handleAuthError(error as AppError)) return;
      runInAction(() => {
        this.error = error instanceof Error ? error.message : "Could not upload mesh.";
      });
    } finally {
      runInAction(() => {
        this.uploading = false;
      });
    }
  }

  // Select an upload from the warehouse, fetching its full text and parsed mesh data. 
  // Shows loading and error states as needed.
  async selectUpload(upload: Upload): Promise<void> {
    this.loading = true;
    this.error = "";
    try {
      const artifact = await fetchUploadArtifact(upload.id);
      const mesh = parseMeshData(artifact.text);
      runInAction(() => {
        this.selectedUpload = artifact.upload;
        this.selectedText = artifact.text;
        this.selectedMesh = mesh;
      });
      this.root.preview.setWarehouseMesh(artifact.upload, artifact.text, mesh);
    } catch (error) {
      if (this.root.auth.handleAuthError(error as AppError)) return;
      runInAction(() => {
        this.error = error instanceof Error ? error.message : "Could not select mesh.";
      });
    } finally {
      runInAction(() => {
        this.loading = false;
      });
    }
  }

  async saveCurrentPreview(): Promise<Upload | null> {
    const preview = this.root.preview.preparedMesh;
    if (!preview || !preview.generated) return null;
    return this.savePreparedMesh(preview);
  }

  async savePreparedMesh(preview: PreparedMesh): Promise<Upload> {
    this.savingGenerated = true;
    this.error = "";
    try {
      const upload = await uploadMeshArtifact(
        new Blob([preview.text], { type: "text/plain" }),
        `${sanitizeDownloadStem(preview.sourceName || "mesh")}_springs.mesh`,
        "generated",
      );
      await this.refreshUploads();
      const artifact = await fetchUploadArtifact(upload.id);
      const mesh = parseMeshData(artifact.text);
      runInAction(() => {
        this.selectedUpload = artifact.upload;
        this.selectedText = artifact.text;
        this.selectedMesh = mesh;
      });
      this.root.preview.setSavedGeneratedMesh(artifact.upload);
      return artifact.upload;
    } catch (error) {
      if (this.root.auth.handleAuthError(error as AppError)) throw error;
      runInAction(() => {
        this.error = error instanceof Error ? error.message : "Could not save generated mesh.";
      });
      throw error;
    } finally {
      runInAction(() => {
        this.savingGenerated = false;
      });
    }
  }

  reset(): void {
    this.uploads = [];
    this.selectedUpload = null;
    this.selectedText = "";
    this.selectedMesh = null;
    this.pickerOpen = false;
    this.loading = false;
    this.uploading = false;
    this.savingGenerated = false;
    this.error = "";
  }
}
