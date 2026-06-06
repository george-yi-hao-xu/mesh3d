import { checkHealth, getMe, login, logout, register } from "./api.js";
import { defaultJobName } from "./format.js";
import { createJobController } from "./jobs.js";
import { maxCachedMeshes, state } from "./state.js";

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
  tabs: document.querySelector("#checkpointTabs"),
  meshCanvas: document.querySelector("#meshCanvas"),
  meshCanvasMessage: document.querySelector("#meshCanvasMessage"),
  preview: document.querySelector("#meshPreview"),
  download: document.querySelector("#downloadLink"),
};

const jobs = createJobController(state, els, { maxCachedMeshes, onAuthError: handleAuthError });
let authMode = "login";

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

async function init() {
  await updateServerStatus();
  try {
    const data = await getMe();
    await showApp(data.user);
  } catch {
    showAuth();
  }
}

async function updateServerStatus() {
  try {
    await checkHealth();
    els.status.textContent = "Server ready";
  } catch {
    els.status.textContent = "Server offline";
  }
}

function getDefaultJobName() {
  return defaultJobName(els.file.files[0]?.name);
}

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

function showAuth() {
  closeEvents();
  state.currentUser = null;
  state.jobs = [];
  state.activeJobId = null;
  state.meshCache.clear();

  els.appLayout.classList.add("hidden");
  els.authPanel.classList.remove("hidden");
  els.authUser.classList.add("hidden");
  els.logout.classList.add("hidden");
  els.authPassword.value = "";
  renderAuthMode();
}

function renderAuthMode() {
  const isLogin = authMode === "login";
  els.authTitle.textContent = isLogin ? "Login" : "Register";
  els.authSubmit.textContent = isLogin ? "Login" : "Register";
  els.authToggle.textContent = isLogin ? "Create an account" : "Use existing account";
  els.authPassword.autocomplete = isLogin ? "current-password" : "new-password";
}

function setAuthError(message) {
  els.authError.textContent = message;
  els.authError.classList.toggle("hidden", !message);
}

function handleAuthError(error) {
  if (error.status !== 401) return false;
  showAuth();
  setAuthError("Please log in again.");
  return true;
}

function closeEvents() {
  if (state.events) {
    state.events.close();
    state.events = null;
  }
}

init();
