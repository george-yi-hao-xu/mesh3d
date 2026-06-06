import {
  createJob,
  deleteJob,
  fetchJob,
  fetchMeshData,
  listJobs,
  uploadPointCloud,
} from "./api.js";
import { escapeHTML, formatDate, formatSeconds, jobTitle } from "./format.js";
import { renderPointCloud, setMeshMessage } from "./mesh-viewer.js";

/**
 * @typedef {{
 *   file: HTMLInputElement,
 *   jobName: HTMLInputElement,
 *   stiffness: HTMLInputElement,
 *   damping: HTMLInputElement,
 *   snapshotInterval: HTMLInputElement,
 *   maxSimTime: HTMLInputElement,
 *   springSeed: HTMLInputElement,
 *   maxSpringDist: HTMLInputElement,
 *   maxSpringsPerParticle: HTMLInputElement,
 *   springConnectProb: HTMLInputElement,
 *   jobList: HTMLElement,
 *   activeJobTitle: HTMLElement,
 *   activeJobMeta: HTMLElement,
 *   activeInputName: HTMLElement,
 *   deleteJob: HTMLButtonElement,
 *   tabs: HTMLElement,
 *   preview: HTMLElement,
 *   download: HTMLAnchorElement,
 *   meshCanvas: HTMLElement,
 *   meshCanvasMessage: HTMLElement
 * }} JobElements
 *
 * @typedef {{ refreshJobs: () => Promise<void>, submitJob: () => Promise<void>, deleteActiveJob: (jobId?: string | null) => Promise<void> }} JobController
 */

/**
 * Creates the job-list and active-job controller around shared client state and DOM nodes.
 *
 * @param {import("./state.js").ClientState} state
 * @param {JobElements} els
 * @param {{ onAuthError?: (error: Error & { status?: number }) => boolean }} options
 * @returns {JobController}
 */
export function createJobController(state, els, options) {
  const { onAuthError = () => false } = options;
  let playbackTimer = null;
  let playbackButton = null;

  /**
   * Loads the current user's jobs and seeds the metadata cache.
   *
   * @returns {Promise<void>}
   */
  async function refreshJobs() {
    state.jobs = await listJobs();
    state.jobs.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
    for (const job of state.jobs) {
      cacheJob(job);
    }
    renderJobs();
  }

  /**
   * Uploads the selected point cloud, submits a synchronous solver job, and displays bundled frames.
   *
   * @returns {Promise<void>}
   */
  async function submitJob() {
    const upload = await uploadPointCloud(els.file.files[0]);
    const response = await createJob(upload.id, els.jobName.value.trim(), getConfig());
    const job = response.job;
    state.activeJobId = job.id;
    state.activeFrameUrl = null;
    state.activeFrames = response.frames || [];
    cacheJob(job);
    upsertJob(job);
    state.jobs.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
    renderJobs();
    renderActiveJob(job, { autoSelectFrame: true });
  }

  /**
   * Deletes the selected finished job and clears the active-job view.
   *
   * @param {string | null} [jobId]
   * @returns {Promise<void>}
   */
  async function deleteActiveJob(jobId = state.activeJobId) {
    const job = getJobById(jobId);
    if (!job) return;

    els.deleteJob.disabled = true;
    try {
      await deleteJob(job.id);
    } catch (error) {
      if (onAuthError(error)) return;
      throw error;
    } finally {
      els.deleteJob.disabled = false;
    }

    removeJob(job.id);
    clearActiveJobView();
    renderJobs();
  }

  /**
   * Renders the left-column job list from `state.jobs`.
   *
   * @returns {void}
   */
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

  /**
   * Selects an existing job. Completed cached jobs can render immediately; older jobs may lazy-load frame files.
   *
   * @param {string} jobId
   * @returns {Promise<void>}
   */
  async function selectJob(jobId) {
    stopPlayback();
    state.activeJobId = jobId;
    state.activeFrameUrl = null;
    state.activeFrames = [];

    let job;
    try {
      job = await getJob(jobId);
    } catch (error) {
      if (onAuthError(error)) return;
      throw error;
    }
    upsertJob(job);
    renderJobs();
    renderActiveJob(job, { autoSelectFrame: true });
  }

  /**
   * Renders active-job metadata, timeline slider, and optional auto-selected frame.
   *
   * @param {import("./api.js").Job} job
   * @param {{ autoSelectFrame?: boolean }} [options]
   * @returns {void}
   */
  function renderActiveJob(job, options = {}) {
    stopPlayback();
    if (!job) return;
    const { autoSelectFrame = false } = options;

    els.activeJobTitle.textContent = jobTitle(job);
    const outcome = job.status === "done"
      ? ` - ${job.converged ? "converged" : "limit reached"} at ${formatSeconds(job.finalTime || 0)}`
      : "";
    els.activeJobMeta.textContent = `${job.status}${outcome} - ${job.id}`;
    els.activeInputName.textContent = job.inputName ? `Input file: ${job.inputName}` : "";
    els.activeInputName.classList.toggle("hidden", !job.inputName);
    els.deleteJob.classList.toggle("hidden", !canDeleteJob(job));
    els.deleteJob.disabled = false;
    els.tabs.innerHTML = "";
    els.tabs.classList.remove("frame-control");

    const frames = getFrames(job);

    if (frames.length === 0) {
      els.download.classList.add("hidden");
      els.download.removeAttribute("href");
      els.preview.textContent = job.status === "failed"
        ? `Job failed: ${job.error || "unknown error"}`
        : "Waiting for first checkpoint.";
      setMeshMessage(els, els.preview.textContent);
    } else {
      const selectedFrame = chooseFrame(frames, autoSelectFrame);
      renderFrameSlider(frames, selectedFrame);
      preloadMissingFrames(state.activeJobId);
      if (autoSelectFrame || !state.activeFrameUrl) {
        selectFrame(selectedFrame.url);
      } else {
        setMeshMessage(els, "", true);
      }
    }
  }

  /**
   * Builds the timeline slider from available frame labels.
   *
   * @param {import("./api.js").MeshFrame[]} frames
   * @param {import("./api.js").MeshFrame} selectedFrame
   * @returns {void}
   */
  function renderFrameSlider(frames, selectedFrame) {
    els.tabs.classList.add("frame-control");
    els.tabs.innerHTML = `
      <div class="frame-head">
        <div class="frame-tools">
          <button class="playback-button" type="button">Play</button>
          <span class="frame-title">Time frame</span>
        </div>
        <span class="frame-label"></span>
      </div>
      <input class="frame-slider" type="range" min="0" max="${frames.length - 1}" step="1" />
      <div class="frame-scale">
        <span>${escapeHTML(frames[0].label)}</span>
        <span>${escapeHTML(frames[frames.length - 1].label)}</span>
      </div>
    `;

    const slider = els.tabs.querySelector(".frame-slider");
    const label = els.tabs.querySelector(".frame-label");
    const playButton = els.tabs.querySelector(".playback-button");
    playbackButton = playButton;
    const selectedIndex = Math.max(0, frames.findIndex((frame) => frame.url === selectedFrame.url));
    slider.value = String(selectedIndex);
    updateFrameLabel(label, frames[selectedIndex], selectedIndex, frames.length);
    playButton.disabled = frames.length < 2;
    playButton.classList.toggle("hidden", frames.length < 2);
    playButton.addEventListener("click", () => {
      if (playbackTimer) {
        stopPlayback();
        return;
      }
      startPlayback(frames, slider, label, playButton);
    });

    slider.addEventListener("input", () => {
      stopPlayback();
      const index = Number(slider.value);
      updateFrameLabel(label, frames[index], index, frames.length);
      selectFrame(frames[index].url);
    });
  }

  /**
   * Displays one frame, loading it first only when the selected job was not submitted with bundled frame text.
   *
   * @param {string} url
   * @returns {Promise<void>}
   */
  async function selectFrame(url) {
    const jobIdAtRequest = state.activeJobId;
    const job = getActiveJob();
    state.activeFrameUrl = url;
    const frame = state.activeFrames.find((item) => item.url === url);
    if (!frame) return;

    if (frame.error) {
      if (onAuthError(frame.error)) return;
      els.preview.textContent = frame.error.message;
      setMeshMessage(els, frame.error.message);
      return;
    }

    if (!frame.loaded) {
      setMeshMessage(els, `Loading ${frame.label}.`);
      await loadFrame(frame);
    }
    if (state.activeJobId !== jobIdAtRequest || state.activeFrameUrl !== url) return;

    if (frame.error) {
      if (onAuthError(frame.error)) return;
      els.preview.textContent = frame.error.message;
      setMeshMessage(els, frame.error.message);
      return;
    }

    els.preview.textContent = frame.text;
    els.download.href = url;
    els.download.download = buildDownloadName(job, url);
    els.download.classList.remove("hidden");

    try {
      await renderPointCloud(state, els, frame.pointCloud, {
        jobId: state.activeJobId,
        frames: state.activeFrames,
      });
    } catch (error) {
      setMeshMessage(els, `Could not visualize this mesh: ${error.message}`);
    }
  }

  /**
   * Reads solver config values from form controls.
   *
   * @returns {Record<string, number>}
   */
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

  /**
   * Inserts or replaces one job in the local job list.
   *
   * @param {import("./api.js").Job} job
   * @returns {void}
   */
  function upsertJob(job) {
    const index = state.jobs.findIndex((item) => item.id === job.id);
    if (index >= 0) {
      state.jobs[index] = job;
    } else {
      state.jobs.unshift(job);
    }
  }

  function removeJob(jobId) {
    state.jobs = state.jobs.filter((job) => job.id !== jobId);
    state.jobCache.delete(jobId);
  }

  /**
   * Returns a cached finished job or fetches fresh job metadata.
   *
   * @param {string} jobId
   * @returns {Promise<import("./api.js").Job>}
   */
  async function getJob(jobId) {
    const cached = state.jobCache.get(jobId);
    if (cached && isFinishedJob(cached)) {
      return cached;
    }

    const job = await fetchJob(jobId);
    cacheJob(job);
    return job;
  }

  /**
   * Stores job metadata by id for later selection.
   *
   * @param {import("./api.js").Job | null | undefined} job
   * @returns {void}
   */
  function cacheJob(job) {
    if (job?.id) {
      state.jobCache.set(job.id, job);
    }
  }

  function getActiveJob() {
    return getJobById(state.activeJobId);
  }

  function getJobById(jobId) {
    if (!jobId) return null;
    return state.jobs.find((job) => job.id === jobId) || state.jobCache.get(jobId) || null;
  }

  function clearActiveJobView() {
    stopPlayback();
    state.activeJobId = null;
    state.activeFrameUrl = null;
    state.activeFrames = [];
    els.activeJobTitle.textContent = "No job selected";
    els.activeJobMeta.textContent = "Run a solve to see checkpoints.";
    els.activeInputName.textContent = "";
    els.activeInputName.classList.add("hidden");
    els.deleteJob.classList.add("hidden");
    els.tabs.innerHTML = "";
    els.tabs.classList.remove("frame-control");
    els.download.classList.add("hidden");
    els.download.removeAttribute("href");
    els.preview.textContent = "No job selected.";
    setMeshMessage(els, "No job selected.");
  }

  function canDeleteJob(job) {
    return job.status === "done" || job.status === "failed";
  }

  function buildDownloadName(job, url) {
    const prefix = sanitizeDownloadStem(jobTitle(job) || "mesh");
    const suffix = url.endsWith("/result") ? "final.msh" : url.split("/").pop() || "mesh.msh";
    return `${prefix}_${suffix}`;
  }

  function sanitizeDownloadStem(value) {
    return String(value)
      .trim()
      .replace(/\.[^.]+$/, "")
      .replace(/[^A-Za-z0-9_-]+/g, "_")
      .replace(/_+/g, "_")
      .replace(/^_+|_+$/g, "") || "mesh";
  }

  /**
   * Completed jobs are safe to reuse from metadata cache.
   *
   * @param {import("./api.js").Job} job
   * @returns {boolean}
   */
  function isFinishedJob(job) {
    return job.status === "done" || job.status === "failed";
  }

  /**
   * Converts job metadata into active frame records and preserves any already loaded frame text.
   *
   * @param {import("./api.js").Job} job
   * @returns {import("./api.js").MeshFrame[]}
   */
  function getFrames(job) {
    const frames = (job.snapshots || []).map((snapshot) => ({
      label: snapshot.label,
      url: snapshot.url,
    }));
    if (job.resultUrl) {
      frames.push({ label: "Final", url: job.resultUrl });
    }
    return syncActiveFrames(frames);
  }

  /**
   * Synchronizes `state.activeFrames` with the current job's frame URLs.
   *
   * @param {Array<Partial<import("./api.js").MeshFrame> & { label: string, url: string }>} frames
   * @returns {import("./api.js").MeshFrame[]}
   */
  function syncActiveFrames(frames) {
    const currentByUrl = new Map(state.activeFrames.map((frame) => [frame.url, frame]));
    state.activeFrames = frames.map((frame) => {
      const current = currentByUrl.get(frame.url);
      if (current) {
        current.label = frame.label;
        return current;
      }
      return {
        ...frame,
        text: frame.text || "",
        pointCloud: frame.pointCloud || null,
        loaded: Boolean(frame.loaded),
        loading: Boolean(frame.loading),
        error: frame.error || null,
        request: frame.request || null,
      };
    });
    return state.activeFrames;
  }

  /**
   * Selects the existing active frame when possible, otherwise defaults to the latest/final frame.
   *
   * @param {import("./api.js").MeshFrame[]} frames
   * @param {boolean} autoSelectFrame
   * @returns {import("./api.js").MeshFrame}
   */
  function chooseFrame(frames, autoSelectFrame) {
    if (!autoSelectFrame && state.activeFrameUrl) {
      const currentFrame = frames.find((frame) => frame.url === state.activeFrameUrl);
      if (currentFrame) return currentFrame;
    }
    return frames[frames.length - 1];
  }

  /**
   * Updates the visible slider label.
   *
   * @param {HTMLElement} label
   * @param {import("./api.js").MeshFrame} frame
   * @param {number} index
   * @param {number} total
   * @returns {void}
   */
  function updateFrameLabel(label, frame, index, total) {
    label.textContent = `${frame.label} (${index + 1}/${total})`;
  }

  /**
   * Advances the selected timeline frame at a fixed playback cadence.
   *
   * @param {import("./api.js").MeshFrame[]} frames
   * @param {HTMLInputElement} slider
   * @param {HTMLElement} label
   * @param {HTMLButtonElement} button
   * @returns {void}
   */
  function startPlayback(frames, slider, label, button) {
    if (frames.length < 2) return;
    if (Number(slider.value) >= frames.length - 1) {
      slider.value = "0";
      updateFrameLabel(label, frames[0], 0, frames.length);
      selectFrame(frames[0].url);
    }

    button.textContent = "Pause";
    playbackButton = button;
    playbackTimer = window.setInterval(() => {
      const currentIndex = Number(slider.value);
      if (currentIndex >= frames.length - 1) {
        stopPlayback();
        return;
      }

      const nextIndex = currentIndex + 1;
      slider.value = String(nextIndex);
      updateFrameLabel(label, frames[nextIndex], nextIndex, frames.length);
      selectFrame(frames[nextIndex].url);

      if (nextIndex >= frames.length - 1) {
        stopPlayback();
      }
    }, 50);
  }

  /**
   * Stops active timeline playback and resets the play button label.
   *
   * @returns {void}
   */
  function stopPlayback() {
    if (playbackTimer) {
      window.clearInterval(playbackTimer);
      playbackTimer = null;
    }
    if (playbackButton) {
      playbackButton.textContent = "Play";
    }
  }

  /**
   * Starts background loading for frames that were not bundled in the job creation response.
   *
   * @param {string | null} jobId
   * @returns {void}
   */
  function preloadMissingFrames(jobId) {
    const frames = state.activeFrames.filter((frame) => !frame.loaded && !frame.loading && !frame.error);
    if (frames.length === 0) return;

    let completed = state.activeFrames.filter((frame) => frame.loaded || frame.error).length;
    if (!selectedFrameIsLoaded()) {
      setMeshMessage(els, `Loading frames ${completed}/${state.activeFrames.length}.`);
    }

    for (const frame of frames) {
      loadFrame(frame).finally(() => {
        if (state.activeJobId !== jobId) return;
        completed += 1;
        const allDone = completed >= state.activeFrames.length;
        if (!allDone) {
          if (!selectedFrameIsLoaded()) {
            setMeshMessage(els, `Loading frames ${completed}/${state.activeFrames.length}.`);
          }
          return;
        }
        if (!state.activeFrameUrl) {
          selectFrame(state.activeFrames[state.activeFrames.length - 1].url);
        } else if (selectedFrameIsLoaded()) {
          setMeshMessage(els, "", true);
        }
      });
    }
  }

  /**
   * Loads and parses a single active frame if it is not already available.
   *
   * @param {import("./api.js").MeshFrame} frame
   * @returns {Promise<import("./api.js").MeshFrame>}
   */
  async function loadFrame(frame) {
    if (frame.loaded || frame.error) return frame;
    if (!frame.request) {
      frame.loading = true;
      frame.request = fetchMeshData(frame.url).then((mesh) => {
        frame.text = mesh.text;
        frame.pointCloud = mesh.pointCloud;
        frame.loaded = true;
        return frame;
      }).catch((error) => {
        frame.error = error;
        throw error;
      }).finally(() => {
        frame.loading = false;
      });
    }

    try {
      await frame.request;
    } catch {
      // The frame stores its own error; callers decide whether to surface it.
    }
    return frame;
  }

  /**
   * Reports whether the current slider-selected frame has render-ready point-cloud data.
   *
   * @returns {boolean}
   */
  function selectedFrameIsLoaded() {
    const selected = state.activeFrames.find((frame) => frame.url === state.activeFrameUrl);
    return Boolean(selected?.loaded);
  }

  return {
    refreshJobs,
    submitJob,
    deleteActiveJob,
  };
}
