import { parseMeshData } from "./mesh-parser.js";

/**
 * @typedef {{ id: string, username: string }} User
 * @typedef {{ id: string, fileName: string, size: number, createdAt: string }} Upload
 * @typedef {{
 *   id: string,
 *   name?: string,
 *   inputName?: string,
 *   status: string,
 *   snapshots?: Array<{ label: string, url: string, simTime?: number, step?: number }>,
 *   resultUrl?: string,
 *   converged?: boolean,
 *   finalTime?: number,
 *   finalStep?: number,
 *   error?: string,
 *   createdAt: string
 * }} Job
 * @typedef {{ text: string, pointCloud: import("./mesh-parser.js").MeshData }} MeshData
 * @typedef {{
 *   label: string,
 *   url: string,
 *   text: string,
 *   isFinal?: boolean,
 *   simTime?: number,
 *   step?: number,
 *   pointCloud: import("./mesh-parser.js").MeshData,
 *   loaded: boolean,
 *   loading: boolean,
 *   error: Error | null,
 *   request: Promise<unknown> | null
 * }} MeshFrame
 */

/**
 * Reads the current authenticated user from the session cookie.
 *
 * @returns {Promise<{ user: User }>}
 */
export async function getMe() {
  const res = await fetch("/api/auth/me");
  return readJSON(res);
}

/**
 * Logs in and lets the server set the auth cookie.
 *
 * @param {string} username
 * @param {string} password
 * @returns {Promise<{ user: User }>}
 */
export async function login(username, password) {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  return readJSON(res);
}

/**
 * Creates an account and starts an authenticated browser session.
 *
 * @param {string} username
 * @param {string} password
 * @returns {Promise<{ user: User }>}
 */
export async function register(username, password) {
  const res = await fetch("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  return readJSON(res);
}

/**
 * Ends the authenticated browser session.
 *
 * @returns {Promise<{ status: string }>}
 */
export async function logout() {
  const res = await fetch("/api/auth/logout", {
    method: "POST",
  });
  return readJSON(res);
}

/**
 * Checks whether the backend process is reachable.
 *
 * @returns {Promise<{ status: string }>}
 */
export async function checkHealth() {
  const res = await fetch("/api/health");
  if (!res.ok) throw new Error("server unavailable");
  return readJSON(res);
}

/**
 * Uploads a point-cloud file before creating a solver job.
 *
 * @param {File | undefined} file
 * @returns {Promise<Upload>}
 * @throws {Error} When no file has been selected.
 */
export async function uploadPointCloud(file) {
  if (!file) {
    throw new Error("Choose a point cloud file first.");
  }

  const body = new FormData();
  body.append("pointCloud", file);

  const res = await fetch("/api/uploads", {
    method: "POST",
    body,
  });
  return readJSON(res);
}

/**
 * Creates and runs a solver job. The current server response includes the completed job and bundled frame text.
 *
 * @param {string} uploadId
 * @param {string} name
 * @param {Record<string, number>} config
 * @returns {Promise<{ job: Job, frames: MeshFrame[] }>}
 */
export async function createJob(uploadId, name, config) {
  const res = await fetch("/api/jobs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ uploadId, name, config }),
  });
  const data = await readJSON(res);
  return {
    ...data,
    frames: parseFrameData(data.frames || []),
  };
}

/**
 * Lists jobs owned by the current user.
 *
 * @returns {Promise<Job[]>}
 */
export async function listJobs() {
  const res = await fetch("/api/jobs");
  return readJSON(res);
}

/**
 * Fetches one job metadata record.
 *
 * @param {string} jobId
 * @returns {Promise<Job>}
 */
export async function fetchJob(jobId) {
  const res = await fetch(`/api/jobs/${jobId}`);
  return readJSON(res);
}

/**
 * Deletes one owned job.
 *
 * @param {string} jobId
 * @returns {Promise<void>}
 */
export async function deleteJob(jobId) {
  const res = await fetch(`/api/jobs/${jobId}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    await readJSON(res);
  }
}

/**
 * Fetches and parses one mesh artifact. This is used for existing jobs whose frames were not bundled at submit time.
 *
 * @param {string} url
 * @returns {Promise<MeshData>}
 */
export async function fetchMeshData(url) {
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

/**
 * Converts server-bundled frame text into render-ready client frame objects.
 *
 * @param {Array<{ label: string, url: string, text: string, isFinal?: boolean, simTime?: number, step?: number }>} frames
 * @returns {MeshFrame[]}
 */
export function parseFrameData(frames) {
  return (Array.isArray(frames) ? frames : []).map((frame) => ({
    ...frame,
    pointCloud: parseMeshData(frame.text),
    loaded: true,
    loading: false,
    error: null,
    request: null,
  }));
}

/**
 * Decodes JSON responses and normalizes non-2xx responses into Error objects with status codes.
 *
 * @param {Response} res
 * @returns {Promise<any>}
 * @throws {Error & { status?: number }}
 */
async function readJSON(res) {
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const error = new Error(data.error || `Request failed: ${res.status}`);
    error.status = res.status;
    throw error;
  }
  return data;
}
