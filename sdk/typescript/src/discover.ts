import fs from "node:fs/promises";
import path from "node:path";

/** Extensions accepted for script evals (explicit path or discovery). */
export const SCRIPT_EXTENSIONS = new Set([".ts", ".mts", ".js", ".mjs"]);

/**
 * True when basename matches discovery naming:
 * `eval.{ts,mts,js,mjs}` or `*.eval.{ts,mts,js,mjs}`.
 */
export function isDiscoverableEvalScript(basename: string): boolean {
  const lower = basename.toLowerCase();
  const ext = path.extname(lower);
  if (!SCRIPT_EXTENSIONS.has(ext)) {
    return false;
  }
  const stem = lower.slice(0, -ext.length);
  return stem === "eval" || stem.endsWith(".eval");
}

/** True when path looks like a Node/TS script eval (by extension only). */
export function isScriptEvalPath(filePath: string): boolean {
  return SCRIPT_EXTENSIONS.has(path.extname(filePath).toLowerCase());
}

/** Build / coverage dirs that must never contribute discovered evals. */
const SKIP_DIRS = new Set([
  "node_modules",
  "dist",
  "dist-test",
  "build",
  "coverage",
  "out",
]);

function shouldSkipDir(name: string): boolean {
  return SKIP_DIRS.has(name) || name.startsWith(".");
}

/**
 * Walk `root` for discoverable eval scripts. Skips `node_modules` and
 * hidden directories. Returns absolute paths sorted for stable order.
 */
export async function discoverEvalScripts(root: string): Promise<string[]> {
  const absRoot = path.resolve(root);
  const found: string[] = [];

  async function walk(dir: string): Promise<void> {
    let entries;
    try {
      entries = await fs.readdir(dir, { withFileTypes: true });
    } catch (err) {
      const code = err && typeof err === "object" && "code" in err ? (err as NodeJS.ErrnoException).code : undefined;
      if (code === "ENOENT" || code === "ENOTDIR") {
        return;
      }
      throw err;
    }
    for (const ent of entries) {
      const full = path.join(dir, ent.name);
      if (ent.isDirectory()) {
        if (!shouldSkipDir(ent.name)) {
          await walk(full);
        }
        continue;
      }
      if (ent.isFile() && isDiscoverableEvalScript(ent.name)) {
        found.push(full);
      }
    }
  }

  await walk(absRoot);
  found.sort();
  return found;
}
