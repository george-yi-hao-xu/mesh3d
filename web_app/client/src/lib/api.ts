import { parseMeshData } from "./mesh-parser";
import type { AppError, Job, JobFrameResponse, MeshFrame, Upload, User } from "../types";

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

export async function uploadMeshArtifact(file: File | Blob | undefined, fileName = ""): Promise<Upload> {
  if (!file) {
    throw new Error("Choose a point cloud file first.");
  }

  const body = new FormData();
  body.append("pointCloud", file, fileName || (file instanceof File ? file.name : "") || "mesh.mesh");

  const res = await fetch("/api/uploads", {
    method: "POST",
    body,
  });
  return readJSON(res);
}

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
    throw error;
  }
  return data as T;
}
