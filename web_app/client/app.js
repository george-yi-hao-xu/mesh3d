import { checkHealth, getMe, login, logout, register } from "./api.js";
import { defaultJobName, jobTitle } from "./format.js";
import { createJobController } from "./jobs.js";
import { state } from "./state.js";

/**
 * @typedef {{
 *   status: HTMLElement,
 *   authPanel: HTMLElement,
 *   appLayout: HTMLElement,
 *   authForm: HTMLFormElement,
 *   authTitle: HTMLElement,
 *   authUsername: HTMLInputElement,
 *   authPassword: HTMLInputElement,
 *   authError: HTMLElement,
 *   authSubmit: HTMLButtonElement,
 *   authToggle: HTMLButtonElement,
 *   authUser: HTMLElement,
 *   logout: HTMLButtonElement,
 *   deleteJob: HTMLButtonElement,
 *   deleteOverlay: HTMLElement,
 *   deleteOverlayTitle: HTMLElement,
 *   deleteOverlayBody: HTMLElement,
 *   deleteOverlayError: HTMLElement,
 *   deleteOverlayCancel: HTMLButtonElement,
 *   deleteOverlayConfirm: HTMLButtonElement,
  *   form: HTMLFormElement,
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
 *   tabs: HTMLElement,
 *   meshCanvas: HTMLElement,
 *   meshCanvasMessage: HTMLElement,
 *   preview: HTMLElement,
 *   download: HTMLAnchorElement
 * }} AppElements
 */

/** @type {AppElements} */
const els = {
  status: document.querySelector("#serverStatus"),
  authPanel: document.querySelector("#authPanel"),
  appLayout: document.querySelector("#appLayout"),
  authForm: document.querySelector("#authForm"),
  authTitle: document.querySelector("#authTitle"),
  authUsername: document.querySelector("#authUsername"),
  authPassword: document.querySelector("#authPassword"),
  authError: document.querySelector("#authError"),
  authSubmit: document.querySelector("#authSubmit"),
  authToggle: document.querySelector("#authToggle"),
  authUser: document.querySelector("#authUser"),
  logout: document.querySelector("#logoutButton"),
  deleteJob: document.querySelector("#deleteJobButton"),
  deleteOverlay: document.querySelector("#deleteOverlay"),
  deleteOverlayTitle: document.querySelector("#deleteOverlayTitle"),
  deleteOverlayBody: document.querySelector("#deleteOverlayBody"),
  deleteOverlayError: document.querySelector("#deleteOverlayError"),
  deleteOverlayCancel: document.querySelector("#deleteOverlayCancel"),
  deleteOverlayConfirm: document.querySelector("#deleteOverlayConfirm"),
  form: document.querySelector("#jobForm"),
  file: document.querySelector("#pointCloud"),
  jobName: document.querySelector("#jobName"),
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
  activeInputName: document.querySelector("#activeInputName"),
  tabs: document.querySelector("#checkpointTabs"),
  meshCanvas: document.querySelector("#meshCanvas"),
  meshCanvasMessage: document.querySelector("#meshCanvasMessage"),
  preview: document.querySelector("#meshPreview"),
  download: document.querySelector("#downloadLink"),
};

const jobs = createJobController(state, els, { onAuthError: handleAuthError });
let authMode = "login";
let pendingDeleteJobId = null;

els.authForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  setAuthError("");
  els.authSubmit.disabled = true;
  els.authSubmit.textContent = authMode === "login" ? "Logging in" : "Registering";

  try {
    const username = els.authUsername.value.trim();
    const password = els.authPassword.value;
    const data = authMode === "login"
      ? await login(username, password)
      : await register(username, password);
    await showApp(data.user);
  } catch (error) {
    setAuthError(error.message);
  } finally {
    els.authSubmit.disabled = false;
    renderAuthMode();
  }
});

els.authToggle.addEventListener("click", () => {
  authMode = authMode === "login" ? "register" : "login";
  setAuthError("");
  renderAuthMode();
});

els.logout.addEventListener("click", async () => {
  await logout().catch(() => {});
  showAuth();
});

els.deleteJob.addEventListener("click", async () => {
  const job = (Array.isArray(state.jobs) ? state.jobs : []).find((item) => item.id === state.activeJobId);
  if (!job) return;
  openDeleteOverlay(job);
});

els.deleteOverlayCancel.addEventListener("click", closeDeleteOverlay);
els.deleteOverlay.addEventListener("click", (event) => {
  if (event.target === els.deleteOverlay) {
    closeDeleteOverlay();
  }
});

els.deleteOverlayConfirm.addEventListener("click", async () => {
  if (!pendingDeleteJobId) return;
  els.deleteOverlayConfirm.disabled = true;
  try {
    await jobs.deleteActiveJob(pendingDeleteJobId);
    closeDeleteOverlay();
  } catch (error) {
    if (handleAuthError(error)) {
      closeDeleteOverlay();
      return;
    }
    setDeleteOverlayError(error.message);
  } finally {
    els.deleteOverlayConfirm.disabled = false;
  }
});

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && !els.deleteOverlay.classList.contains("hidden")) {
    closeDeleteOverlay();
  }
});

els.file.addEventListener("change", () => {
  if (!state.jobNameEdited || !els.jobName.value.trim()) {
    els.jobName.value = getDefaultJobName();
    state.jobNameEdited = false;
  }
});

els.jobName.addEventListener("input", () => {
  state.jobNameEdited = true;
});

els.form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const submit = els.form.querySelector("button");
  submit.disabled = true;
  submit.textContent = "Starting";

  try {
    if (!els.jobName.value.trim()) {
      els.jobName.value = getDefaultJobName();
    }
    await jobs.submitJob();
  } catch (error) {
    if (handleAuthError(error)) return;
    alert(error.message);
  } finally {
    submit.disabled = false;
    submit.textContent = "Run Solve";
  }
});

/**
 * Initializes server status and restores an existing authenticated session when possible.
 *
 * @returns {Promise<void>}
 */
async function init() {
  await updateServerStatus();
  try {
    const data = await getMe();
    await showApp(data.user);
  } catch {
    showAuth();
  }
}

/**
 * Updates the topbar health badge from `/api/health`.
 *
 * @returns {Promise<void>}
 */
async function updateServerStatus() {
  try {
    await checkHealth();
    els.status.textContent = "Server ready";
  } catch {
    els.status.textContent = "Server offline";
  }
}

/**
 * Returns the current default job name based on the selected upload file.
 *
 * @returns {string}
 */
function getDefaultJobName() {
  return defaultJobName(els.file.files[0]?.name);
}

/**
 * Switches from the auth form to the app shell and loads the user's job list.
 *
 * @param {import("./api.js").User} user
 * @returns {Promise<void>}
 */
async function showApp(user) {
  state.currentUser = user;
  els.authPanel.classList.add("hidden");
  els.appLayout.classList.remove("hidden");
  els.authUser.textContent = user.username;
  els.authUser.classList.remove("hidden");
  els.logout.classList.remove("hidden");

  try {
    await jobs.refreshJobs();
  } catch (error) {
    if (!handleAuthError(error)) throw error;
  }
}

/**
 * Resets client-only user, job, and frame state before showing the login/register form.
 *
 * @returns {void}
 */
function showAuth() {
  closeDeleteOverlay();
  state.currentUser = null;
  state.jobs = [];
  state.activeJobId = null;
  state.activeFrameUrl = null;
  state.activeFrames = [];
  state.jobCache.clear();

  els.appLayout.classList.add("hidden");
  els.authPanel.classList.remove("hidden");
  els.authUser.classList.add("hidden");
  els.logout.classList.add("hidden");
  els.deleteJob.classList.add("hidden");
  els.authPassword.value = "";
  renderAuthMode();
}

/**
 * Updates labels and autocomplete attributes for login vs register mode.
 *
 * @returns {void}
 */
function renderAuthMode() {
  const isLogin = authMode === "login";
  els.authTitle.textContent = isLogin ? "Login" : "Register";
  els.authSubmit.textContent = isLogin ? "Login" : "Register";
  els.authToggle.textContent = isLogin ? "Create an account" : "Use existing account";
  els.authPassword.autocomplete = isLogin ? "current-password" : "new-password";
}

/**
 * Shows or hides the auth error message.
 *
 * @param {string} message
 * @returns {void}
 */
function setAuthError(message) {
  els.authError.textContent = message;
  els.authError.classList.toggle("hidden", !message);
}

/**
 * Handles expired or invalid sessions consistently across API calls.
 *
 * @param {Error & { status?: number }} error
 * @returns {boolean} True when the error was handled as an auth failure.
 */
function handleAuthError(error) {
  if (error.status !== 401) return false;
  showAuth();
  setAuthError("Please log in again.");
  return true;
}

function openDeleteOverlay(job) {
  pendingDeleteJobId = job.id;
  els.deleteOverlayTitle.textContent = `Delete "${jobTitle(job)}"?`;
  els.deleteOverlayBody.textContent = "This removes the job and its saved outputs.";
  setDeleteOverlayError("");
  els.deleteOverlayConfirm.disabled = false;
  els.deleteOverlay.classList.remove("hidden");
  els.deleteOverlayCancel.focus();
}

function closeDeleteOverlay() {
  pendingDeleteJobId = null;
  setDeleteOverlayError("");
  els.deleteOverlay.classList.add("hidden");
}

function setDeleteOverlayError(message) {
  els.deleteOverlayError.textContent = message;
  els.deleteOverlayError.classList.toggle("hidden", !message);
}

init();
