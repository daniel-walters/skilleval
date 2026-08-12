/**
 * Normalize pre-call tool args for Result.metrics.toolCalls.
 * - Map Claude file_path → path
 * - Keep small scalars; strip large body fields
 */

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
 * @param {unknown} raw
 * @returns {Record<string, string | number | boolean | null> | undefined}
 */
export function normalizeToolArgs(raw) {
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
  return Object.keys(out).length > 0 ? out : undefined;
}
