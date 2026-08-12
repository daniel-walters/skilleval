/**
 * Normalize pre-call tool args for Result.metrics.toolCalls.
 * - Map Claude file_path → path
 * - Relativize path against the run cwd (forward slashes)
 * - Keep small scalars; strip large body fields
 */

import path from "node:path";

const STRIP_KEYS = new Set([
  "fileText",
  "content",
  "oldText",
  "newText",
  "old_string",
  "new_string",
  "contents",
  "patch",
  "diff",
]);

/**
 * Make a filesystem path workspace-relative when it falls under cwd.
 * Paths outside cwd are left unchanged. Output uses forward slashes.
 *
 * @param {string} filePath
 * @param {string} [cwd]
 * @returns {string}
 */
export function relativizeToolPath(filePath, cwd) {
  if (typeof filePath !== "string" || !filePath || typeof cwd !== "string" || !cwd) {
    return filePath;
  }
  const root = path.resolve(cwd);
  const abs = path.isAbsolute(filePath)
    ? path.resolve(filePath)
    : path.resolve(root, filePath);
  const rel = path.relative(root, abs);
  if (!rel || rel.startsWith("..") || path.isAbsolute(rel)) {
    return filePath.split(path.sep).join("/");
  }
  return rel.split(path.sep).join("/");
}

/**
 * @param {unknown} raw
 * @param {string} [cwd] Attempt workspace directory for path relativization
 * @returns {Record<string, string | number | boolean | null> | undefined}
 */
export function normalizeToolArgs(raw, cwd) {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return undefined;
  }
  /** @type {Record<string, string | number | boolean | null>} */
  const out = {};
  for (const [key, value] of Object.entries(raw)) {
    if (STRIP_KEYS.has(key)) continue;
    if (key === "file_path") {
      if (out.path === undefined && typeof value === "string") {
        out.path = value;
      }
      continue;
    }
    if (
      value === null ||
      typeof value === "string" ||
      typeof value === "number" ||
      typeof value === "boolean"
    ) {
      out[key] = value;
    }
  }
  if (typeof out.path === "string") {
    out.path = relativizeToolPath(out.path, cwd);
  }
  return Object.keys(out).length > 0 ? out : undefined;
}
