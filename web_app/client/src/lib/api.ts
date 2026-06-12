import { parseMeshData } from "./mesh-parser";
import type { AppError, Job, JobFrameResponse, JobReview, MeshFrame, TrainingCluster, TrainingRun, Upload, UploadArtifact, User } from "../types";

export async function getMe(): Promise<{ user: User }> {
  const res = await fetch("/api/auth/me");
  return readJSON(res);
}

export async function login(username: string, password: string): Promise<{ user: User }> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  return readJSON(res);
}

export async function register(username: string, password: string): Promise<{ user: User }> {
  const res = await fetch("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  return readJSON(res);
}

export async function logout(): Promise<{ status: string }> {
  const res = await fetch("/api/auth/logout", { method: "POST" });
  return readJSON(res);
}

export async function checkHealth(): Promise<{ status: string }> {
  const res = await fetch("/api/health");
  if (!res.ok) throw new Error("server unavailable");
  return readJSON(res);
}

export async function uploadMeshArtifact(file: File | Blob | undefined, fileName = "", meshKind: "uploaded" | "generated" = "uploaded"): Promise<Upload> {
  if (!file) {
    throw new Error("Choose a mesh file first.");
  }

  const body = new FormData();
  body.append("pointCloud", file, fileName || (file instanceof File ? file.name : "") || "mesh.mesh");
  body.append("meshKind", meshKind);

  const res = await fetch("/api/uploads", {
    method: "POST",
    body,
  });
  return readJSON(res);
}

// GET all uploads (initial meshes)
export async function listUploads(): Promise<Upload[]> {
  const res = await fetch("/api/uploads");
  return readJSON(res);
}

// GET a specific upload artifact
export async function fetchUploadArtifact(uploadId: string): Promise<UploadArtifact> {
  const res = await fetch(`/api/uploads/${uploadId}`);
  return readJSON(res);
}

export async function deleteUpload(uploadId: string): Promise<void> {
  const res = await fetch(`/api/uploads/${uploadId}`, { method: "DELETE" });
  if (!res.ok) {
    const error = await readError(res);
    if (error.status === 405) {
      throw Object.assign(new Error("Mesh delete is not available from the current backend. Restart the backend server and try again."), { status: 405 });
    }
    throw error;
  }
}

// run solver
export async function createJob(uploadId: string, name: string, config: Record<string, number>): Promise<{ job: Job; frames: MeshFrame[] }> {
  const res = await fetch("/api/jobs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ uploadId, name, config }),
  });
  const data = await readJSON<{ job: Job; frames?: JobFrameResponse[] }>(res);
  return {
    ...data,
    frames: parseFrameData(data.frames || []),
  };
}

export async function listJobs(): Promise<Job[]> {
  const res = await fetch("/api/jobs");
  return readJSON(res);
}

export async function fetchJob(jobId: string): Promise<Job> {
  const res = await fetch(`/api/jobs/${jobId}`);
  return readJSON(res);
}

export async function deleteJob(jobId: string): Promise<void> {
  const res = await fetch(`/api/jobs/${jobId}`, { method: "DELETE" });
  if (!res.ok) {
    await readJSON(res);
  }
}

export async function saveJobReview(jobId: string, review: { score: number; tags: string[]; note: string }): Promise<JobReview> {
  const res = await fetch(`/api/jobs/${jobId}/review`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    // credentials: "same-origin",
    body: JSON.stringify(review),
  });
  return readJSON(res);
}

export async function listTrainingClusters(): Promise<TrainingCluster[]> {
  const res = await fetch("/api/training/clusters");
  return readJSON(res);
}

export async function createTrainingCluster(name: string): Promise<TrainingCluster> {
  const res = await fetch("/api/training/clusters", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
  return readJSON(res);
}

export async function updateTrainingCluster(clusterId: string, name: string): Promise<TrainingCluster> {
  const res = await fetch(`/api/training/clusters/${clusterId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
  return readJSON(res);
}

export async function deleteTrainingCluster(clusterId: string): Promise<void> {
  const res = await fetch(`/api/training/clusters/${clusterId}`, { method: "DELETE" });
  if (!res.ok) {
    await readJSON(res);
  }
}

export async function addJobToTrainingCluster(clusterId: string, jobId: string): Promise<TrainingCluster> {
  const res = await fetch(`/api/training/clusters/${clusterId}/jobs`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ jobId }),
  });
  return readJSON(res);
}

export async function removeJobFromTrainingCluster(clusterId: string, jobId: string): Promise<TrainingCluster> {
  const res = await fetch(`/api/training/clusters/${clusterId}/jobs/${jobId}`, { method: "DELETE" });
  return readJSON(res);
}

export async function trainCluster(clusterId: string): Promise<TrainingRun> {
  const res = await fetch(`/api/training/clusters/${clusterId}/train`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: "{}",
  });
  return readJSON(res);
}

export async function recommendClusterConfig(clusterId: string, uploadId: string): Promise<TrainingCluster> {
  const res = await fetch(`/api/training/clusters/${clusterId}/recommend`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ uploadId, candidateCount: 512 }),
  });
  return readJSON(res);
}

export async function fetchMeshData(url: string): Promise<{ text: string; pointCloud: ReturnType<typeof parseMeshData> }> {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`Could not load ${url}`);
  }

  const text = await res.text();
  return {
    text,
    pointCloud: parseMeshData(text),
  };
}

export function parseFrameData(frames: JobFrameResponse[]): MeshFrame[] {
  return (Array.isArray(frames) ? frames : []).map((frame) => ({
    ...frame,
    pointCloud: parseMeshData(frame.text),
    loaded: true,
    loading: false,
    error: null,
    request: null,
  }));
}

async function readJSON<T = unknown>(res: Response): Promise<T> {
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const error = new Error((data as { error?: string }).error || `Request failed: ${res.status}`) as AppError;
    error.status = res.status;
    error.relatedJobIds = parseRelatedJobIds(data);
    throw error;
  }
  return data as T;
}

async function readError(res: Response): Promise<AppError> {
  const data = await res.json().catch(() => ({}));
  const error = new Error((data as { error?: string }).error || `Request failed: ${res.status}`) as AppError;
  error.status = res.status;
  error.relatedJobIds = parseRelatedJobIds(data);
  return error;
}

function parseRelatedJobIds(data: unknown): string[] | undefined {
  const relatedJobIds = (data as { relatedJobIds?: unknown }).relatedJobIds;
  if (!Array.isArray(relatedJobIds)) return undefined;
  return relatedJobIds.filter((id): id is string => typeof id === "string");
}
