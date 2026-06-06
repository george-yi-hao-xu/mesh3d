/**
 * Formats a timestamp for compact job-list display.
 *
 * @param {string | number | Date} value
 * @returns {string}
 */
export function formatDate(value) {
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(new Date(value));
}

/**
 * Formats simulation seconds with two decimal places.
 *
 * @param {number} value
 * @returns {string}
 */
export function formatSeconds(value) {
  return `${Number(value).toFixed(2)}s`;
}

/**
 * Builds the initial job name from the selected upload file and current local time.
 *
 * @param {string | undefined} fileName
 * @returns {string}
 */
export function defaultJobName(fileName) {
  const meshName = meshNameFromFile(fileName || "mesh");
  return `${formatJobNameTime(new Date())}_${meshName}`;
}

/**
 * Removes the final extension from a file name for use in generated job names.
 *
 * @param {string} fileName
 * @returns {string}
 */
export function meshNameFromFile(fileName) {
  const lastDot = fileName.lastIndexOf(".");
  return lastDot > 0 ? fileName.slice(0, lastDot) : fileName;
}

/**
 * Formats a Date into the server-compatible job-name timestamp fragment.
 *
 * @param {Date} date
 * @returns {string}
 */
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

/**
 * Chooses the best display title for a job.
 *
 * @param {import("./api.js").Job} job
 * @returns {string}
 */
export function jobTitle(job) {
  return job.name || job.inputName || job.id;
}

/**
 * Escapes text inserted through innerHTML.
 *
 * @param {unknown} value
 * @returns {string}
 */
export function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
