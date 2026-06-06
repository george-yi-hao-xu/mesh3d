const state = {
  jobs: [],
  activeJobId: null,
  events: null,
};

const els = {
  status: document.querySelector("#serverStatus"),
  form: document.querySelector("#jobForm"),
  file: document.querySelector("#pointCloud"),
  stiffness: document.querySelector("#stiffness"),
  damping: document.querySelector("#damping"),
  snapshotInterval: document.querySelector("#snapshotInterval"),
  maxSimTime: document.querySelector("#maxSimTime"),
  springSeed: document.querySelector("#springSeed"),
  maxSpringDist: document.querySelector("#maxSpringDist"),
  maxSpringsPerParticle: document.querySelector("#maxSpringsPerParticle"),
  springConnectProb: document.querySelector("#springConnectProb"),
  jobList: document.querySelector("#jobList"),
  activeJobTitle: document.querySelector("#activeJobTitle"),
  activeJobMeta: document.querySelector("#activeJobMeta"),
  tabs: document.querySelector("#checkpointTabs"),
  preview: document.querySelector("#meshPreview"),
  download: document.querySelector("#downloadLink"),
};

els.form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const submit = els.form.querySelector("button");
  submit.disabled = true;
  submit.textContent = "Starting";

  try {
    const upload = await uploadPointCloud();
    const job = await createJob(upload.id);
    await refreshJobs();
    selectJob(job.id);
  } catch (error) {
    alert(error.message);
  } finally {
    submit.disabled = false;
    submit.textContent = "Run Solve";
  }
});

async function init() {
  await checkHealth();
  await refreshJobs();
}

async function checkHealth() {
  try {
    const res = await fetch("/api/health");
    if (!res.ok) throw new Error("server unavailable");
    els.status.textContent = "Server ready";
  } catch {
    els.status.textContent = "Server offline";
  }
}

async function uploadPointCloud() {
  const file = els.file.files[0];
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

async function createJob(uploadId) {
  const config = {
    stiffness: Number(els.stiffness.value),
    dampingFactor: Number(els.damping.value),
    snapshotInterval: Number(els.snapshotInterval.value),
    maxSimTime: Number(els.maxSimTime.value),
    springSeed: Number(els.springSeed.value),
    maxSpringDist: Number(els.maxSpringDist.value),
    maxSpringsPerParticle: Number(els.maxSpringsPerParticle.value),
    springConnectProb: Number(els.springConnectProb.value),
  };

  const res = await fetch("/api/jobs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ uploadId, config }),
  });
  return readJSON(res);
}

async function refreshJobs() {
  const res = await fetch("/api/jobs");
  state.jobs = await readJSON(res);
  state.jobs.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
  renderJobs();
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
      <span class="job-title">${escapeHTML(job.inputName || job.id)}</span>
      <span class="job-meta">${job.status} - ${job.snapshots?.length || 0} checkpoints - ${formatDate(job.createdAt)}</span>
    `;
    item.addEventListener("click", () => selectJob(job.id));
    els.jobList.appendChild(item);
  }
}

async function selectJob(jobId) {
  state.activeJobId = jobId;
  closeEvents();

  const job = await fetchJob(jobId);
  upsertJob(job);
  renderJobs();
  renderActiveJob(job);

  if (job.status === "queued" || job.status === "running") {
    openEvents(jobId);
  }
}

async function fetchJob(jobId) {
  const res = await fetch(`/api/jobs/${jobId}`);
  return readJSON(res);
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

  els.activeJobTitle.textContent = job.inputName || job.id;
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
  const res = await fetch(url);
  if (!res.ok) {
    els.preview.textContent = `Could not load ${url}`;
    return;
  }

  const text = await res.text();
  els.preview.textContent = text;
  els.download.href = url;
  els.download.download = url.endsWith("/result") ? "final.msh" : url.split("/").pop();
  els.download.classList.remove("hidden");

  for (const tab of els.tabs.querySelectorAll(".tab")) {
    tab.classList.toggle("active", tab === activeButton);
  }
}

function upsertJob(job) {
  const index = state.jobs.findIndex((item) => item.id === job.id);
  if (index >= 0) {
    state.jobs[index] = job;
  } else {
    state.jobs.unshift(job);
  }
}

async function readJSON(res) {
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `Request failed: ${res.status}`);
  }
  return data;
}

function formatDate(value) {
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(new Date(value));
}

function formatSeconds(value) {
  return `${Number(value).toFixed(2)}s`;
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

init();
