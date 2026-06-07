import type { Job } from "../types";

export function formatDate(value: string | number | Date): string {
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(new Date(value));
}

export function formatSeconds(value: number): string {
  return `${Number(value).toFixed(2)}s`;
}

export function defaultJobName(fileName: string | undefined): string {
  const meshName = meshNameFromFile(fileName || "mesh");
  return `${formatJobNameTime(new Date())}_${meshName}`;
}

export function meshNameFromFile(fileName: string): string {
  const lastDot = fileName.lastIndexOf(".");
  return lastDot > 0 ? fileName.slice(0, lastDot) : fileName;
}

export function formatJobNameTime(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, "0");
  return [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate()),
  ].join("-") + "_" + [
    pad(date.getHours()),
    pad(date.getMinutes()),
    pad(date.getSeconds()),
  ].join("-");
}

export function jobTitle(job: Job): string {
  return job.name || job.inputName || job.id;
}

export function sanitizeDownloadStem(value: unknown): string {
  return String(value)
    .trim()
    .replace(/\.[^.]+$/, "")
    .replace(/[^A-Za-z0-9_-]+/g, "_")
    .replace(/_+/g, "_")
    .replace(/^_+|_+$/g, "") || "mesh";
}
