import { parsePointCloud } from "./mesh-parser.js";

export async function getMe() {
  const res = await fetch("/api/auth/me");
  return readJSON(res);
}

export async function login(username, password) {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  return readJSON(res);
}

export async function register(username, password) {
  const res = await fetch("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  return readJSON(res);
}

export async function logout() {
  const res = await fetch("/api/auth/logout", {
    method: "POST",
  });
  return readJSON(res);
}

export async function checkHealth() {
  const res = await fetch("/api/health");
  if (!res.ok) throw new Error("server unavailable");
  return readJSON(res);
}

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

export async function createJob(uploadId, name, config) {
  const res = await fetch("/api/jobs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ uploadId, name, config }),
  });
  return readJSON(res);
}

export async function listJobs() {
  const res = await fetch("/api/jobs");
  return readJSON(res);
}

export async function fetchJob(jobId) {
  const res = await fetch(`/api/jobs/${jobId}`);
  return readJSON(res);
}

export async function getMeshData(url, cache, maxCachedMeshes) {
  if (cache.has(url)) {
    return cache.get(url);
  }

  const request = fetch(url).then(async (res) => {
    if (!res.ok) {
      throw new Error(`Could not load ${url}`);
    }

    const text = await res.text();
    return {
      text,
      pointCloud: parsePointCloud(text),
    };
  }).catch((error) => {
    cache.delete(url);
    throw error;
  });

  cache.set(url, request);
  trimCache(cache, maxCachedMeshes);
  return request;
}

async function readJSON(res) {
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const error = new Error(data.error || `Request failed: ${res.status}`);
    error.status = res.status;
    throw error;
  }
  return data;
}

function trimCache(cache, maxSize) {
  while (cache.size > maxSize) {
    const oldestUrl = cache.keys().next().value;
    cache.delete(oldestUrl);
  }
}
