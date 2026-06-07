import { makeAutoObservable, runInAction } from "mobx";
import { checkHealth, getMe, login, logout, register } from "../lib/api";
import type { AppError, User } from "../types";
import type { RootStore } from "./root-store";

export class AuthStore {
  readonly root: RootStore;
  currentUser: User | null = null;
  healthReady = false;
  authMode: "login" | "register" = "login";
  error = "";
  submitting = false;
  initialized = false;

  constructor(root: RootStore) {
    this.root = root;
    makeAutoObservable(this, { root: false });
  }

  get serverStatus(): string {
    return this.healthReady ? "Server ready" : "Server unavailable";
  }

  get isAuthenticated(): boolean {
    return Boolean(this.currentUser);
  }

  async init(): Promise<void> {
    await this.updateServerStatus();
    try {
      const data = await getMe();
      runInAction(() => {
        this.currentUser = data.user;
      });
      await this.root.warehouse.refreshUploads();
      await this.root.jobs.refreshJobs();
    } catch {
      runInAction(() => {
        this.currentUser = null;
      });
    } finally {
      runInAction(() => {
        this.initialized = true;
      });
    }
  }

  async updateServerStatus(): Promise<void> {
    try {
      await checkHealth();
      runInAction(() => {
        this.healthReady = true;
      });
    } catch {
      runInAction(() => {
        this.healthReady = false;
      });
    }
  }

  setAuthMode(mode: "login" | "register"): void {
    this.authMode = mode;
    this.error = "";
  }

  toggleAuthMode(): void {
    this.setAuthMode(this.authMode === "login" ? "register" : "login");
  }

  setError(message: string): void {
    this.error = message;
  }

  async submit(username: string, password: string): Promise<void> {
    this.error = "";
    this.submitting = true;
    try {
      const data = this.authMode === "login"
        ? await login(username.trim(), password)
        : await register(username.trim(), password);
      runInAction(() => {
        this.currentUser = data.user;
      });
      await this.root.warehouse.refreshUploads();
      await this.root.jobs.refreshJobs();
    } catch (error) {
      runInAction(() => {
        this.error = getErrorMessage(error);
      });
    } finally {
      runInAction(() => {
        this.submitting = false;
      });
    }
  }

  async logout(): Promise<void> {
    await logout().catch(() => {});
    runInAction(() => {
      this.currentUser = null;
      this.error = "";
    });
    this.root.jobs.reset();
    this.root.preview.reset();
    this.root.warehouse.reset();
    this.root.viewer.clear("No mesh loaded.");
  }

  handleAuthError(error: AppError): boolean {
    if (error.status !== 401) return false;
    void this.logout();
    return true;
  }
}

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Request failed";
}
