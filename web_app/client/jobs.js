import {
  createJob,
  fetchJob,
  getMeshData,
  listJobs,
  uploadPointCloud,
} from "./api.js";
import { escapeHTML, formatDate, formatSeconds, jobTitle } from "./format.js";
import { renderPointCloud, setMeshMessage } from "./mesh-viewer.js";

export function createJobController(state, els, options) {
  const { maxCachedMeshes, onAuthError = () => false } = options;

  async function refreshJobs() {
    state.jobs = await listJobs();
    state.jobs.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
    renderJobs();
  }

  async function submitJob() {
    const upload = await uploadPointCloud(els.file.files[0]);
    const job = await createJob(upload.id, els.jobName.value.trim(), getConfig());
    await refreshJobs();
    selectJob(job.id);
  }

  function renderJobs() {
    els.jobList.innerHTML = "";

    if (state.jobs.length === 0) {
      els.jobList.innerHTML = `<p class="job-meta">No jobs yet.</p>`;
      return;
    }

    for (const job of state.jobs) {
      const item = document.createElement("button");
      item.type = "button";
      item.className = `job-item ${job.id === state.activeJobId ? "active" : ""}`;
      item.innerHTML = `
        <span class="job-title">${escapeHTML(jobTitle(job))}</span>
        <span class="job-meta">${job.status} - ${job.snapshots?.length || 0} checkpoints - ${formatDate(job.createdAt)}</span>
      `;
      item.addEventListener("click", () => selectJob(job.id));
      els.jobList.appendChild(item);
    }
  }

  async function selectJob(jobId) {
    state.activeJobId = jobId;
    closeEvents();

    let job;
    try {
      job = await fetchJob(jobId);
    } catch (error) {
      if (onAuthError(error)) return;
      throw error;
    }
    upsertJob(job);
    renderJobs();
    renderActiveJob(job);

    if (job.status === "queued" || job.status === "running") {
      openEvents(jobId);
    }
  }

  function openEvents(jobId) {
    const events = new EventSource(`/api/jobs/${jobId}/events`);
    state.events = events;

    events.addEventListener("snapshot", (message) => {
      const event = JSON.parse(message.data);
      if (event.jobId !== state.activeJobId) return;
      fetchJob(event.jobId).then((job) => {
        upsertJob(job);
        renderActiveJob(job);
        renderJobs();
      }).catch((error) => {
        onAuthError(error);
      });
    });

    events.addEventListener("done", (message) => {
      const event = JSON.parse(message.data);
      if (event.job) upsertJob(event.job);
      if (event.jobId === state.activeJobId) renderActiveJob(event.job);
      renderJobs();
      closeEvents();
    });

    events.addEventListener("failed", (message) => {
      const event = JSON.parse(message.data);
      if (event.job) upsertJob(event.job);
      if (event.jobId === state.activeJobId) renderActiveJob(event.job);
      renderJobs();
      closeEvents();
    });
  }

  function closeEvents() {
    if (state.events) {
      state.events.close();
      state.events = null;
    }
  }

  function renderActiveJob(job) {
    if (!job) return;

    els.activeJobTitle.textContent = jobTitle(job);
    const outcome = job.status === "done"
      ? ` - ${job.converged ? "converged" : "limit reached"} at ${formatSeconds(job.finalTime || 0)}`
      : "";
    els.activeJobMeta.textContent = `${job.status}${outcome} - ${job.id}`;
    els.tabs.innerHTML = "";
    els.download.classList.add("hidden");
    els.download.removeAttribute("href");

    const snapshots = job.snapshots || [];
    for (const snapshot of snapshots) {
      addCheckpointButton(snapshot.label, snapshot.url);
    }

    if (job.resultUrl) {
      addCheckpointButton("Final", job.resultUrl);
    }

    if (snapshots.length === 0 && !job.resultUrl) {
      els.preview.textContent = job.status === "failed"
        ? `Job failed: ${job.error || "unknown error"}`
        : "Waiting for first checkpoint.";
      setMeshMessage(els, els.preview.textContent);
    } else {
      setMeshMessage(els, "Choose a checkpoint to load the 3D view.");
    }
  }

  function addCheckpointButton(label, url) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "tab";
    button.textContent = label;
    button.addEventListener("click", () => loadMesh(url, button));
    els.tabs.appendChild(button);
  }

  async function loadMesh(url, activeButton) {
    setMeshMessage(els, state.meshCache.has(url) ? "Loading cached mesh." : "Loading mesh.");

    let mesh;
    try {
      mesh = await getMeshData(url, state.meshCache, maxCachedMeshes);
    } catch (error) {
      if (onAuthError(error)) return;
      els.preview.textContent = error.message;
      setMeshMessage(els, error.message);
      return;
    }

    els.preview.textContent = mesh.text;
    els.download.href = url;
    els.download.download = url.endsWith("/result") ? "final.msh" : url.split("/").pop();
    els.download.classList.remove("hidden");

    for (const tab of els.tabs.querySelectorAll(".tab")) {
      tab.classList.toggle("active", tab === activeButton);
    }

    try {
      await renderPointCloud(state, els, mesh.pointCloud);
    } catch (error) {
      setMeshMessage(els, `Could not visualize this mesh: ${error.message}`);
    }
  }

  function getConfig() {
    return {
      stiffness: Number(els.stiffness.value),
      dampingFactor: Number(els.damping.value),
      snapshotInterval: Number(els.snapshotInterval.value),
      maxSimTime: Number(els.maxSimTime.value),
      springSeed: Number(els.springSeed.value),
      maxSpringDist: Number(els.maxSpringDist.value),
      maxSpringsPerParticle: Number(els.maxSpringsPerParticle.value),
      springConnectProb: Number(els.springConnectProb.value),
    };
  }

  function upsertJob(job) {
    const index = state.jobs.findIndex((item) => item.id === job.id);
    if (index >= 0) {
      state.jobs[index] = job;
    } else {
      state.jobs.unshift(job);
    }
  }

  return {
    refreshJobs,
    submitJob,
  };
}
