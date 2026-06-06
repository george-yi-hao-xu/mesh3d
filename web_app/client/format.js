export function formatDate(value) {
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(new Date(value));
}

export function formatSeconds(value) {
  return `${Number(value).toFixed(2)}s`;
}

export function defaultJobName(fileName) {
  const meshName = meshNameFromFile(fileName || "mesh");
  return `${formatJobNameTime(new Date())}_${meshName}`;
}

export function meshNameFromFile(fileName) {
  const lastDot = fileName.lastIndexOf(".");
  return lastDot > 0 ? fileName.slice(0, lastDot) : fileName;
}

export function formatJobNameTime(date) {
  const pad = (value) => String(value).padStart(2, "0");
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

export function jobTitle(job) {
  return job.name || job.inputName || job.id;
}

export function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
