import fs from "node:fs";
import path from "node:path";

import type { FileOutcome, FileStatus, Result } from "../result.js";
import {
  isRegex,
  matchContains,
  matchDisplay,
  matchEquals,
  type StringOrRegexp,
} from "../stringMatch.js";
import { fail } from "./error.js";

export class FileMatchers {
  private readonly prefix: string;
  private readonly outcome: FileOutcome | undefined;

  constructor(
    private readonly result: Result,
    private readonly relPath: string,
    private readonly workspace: string | undefined,
  ) {
    this.prefix = `files[${relPath}]`;
    this.outcome = result.outcomes.files?.[relPath];
  }

  toHaveStatus(status: FileStatus): this {
    if (!this.outcome) {
      fail(`${this.prefix}.status`, "path missing from outcomes.files");
      return this;
    }
    if (this.outcome.status !== status) {
      fail(
        `${this.prefix}.status`,
        `status is ${JSON.stringify(this.outcome.status)}, want ${JSON.stringify(status)}`,
      );
    }
    return this;
  }

  toHaveBeenCreated(): this {
    return this.toHaveStatus("created");
  }

  toHaveBeenModified(): this {
    return this.toHaveStatus("modified");
  }

  toHaveBeenDeleted(): this {
    return this.toHaveStatus("deleted");
  }

  toContain(expected: StringOrRegexp): this {
    this.assertContent(expected, "contains");
    return this;
  }

  toEqual(expected: StringOrRegexp): this {
    this.assertContent(expected, "equals");
    return this;
  }

  private assertContent(expected: StringOrRegexp, kind: "contains" | "equals"): void {
    if (!this.outcome) {
      fail(this.prefix, "path missing from outcomes.files");
      return;
    }
    if (this.outcome.status === "deleted") {
      fail(this.prefix, "content expects cannot be checked for a deleted file");
      return;
    }
    const body = readWorkspaceFile(this.workspace, this.relPath);
    if (body === undefined) {
      return;
    }
    if (kind === "contains") {
      if (!matchContains(body, expected)) {
        const reason = isRegex(expected)
          ? `file does not match ${matchDisplay(expected)}`
          : `file does not contain ${matchDisplay(expected)}`;
        fail(`${this.prefix}.contains`, reason);
      }
      return;
    }
    if (!matchEquals(body, expected)) {
      const reason = isRegex(expected)
        ? `file content does not match ${matchDisplay(expected)}`
        : "file content does not equal expected value";
      fail(`${this.prefix}.equals`, reason);
    }
  }
}

/** Read workspace file; soft-fails and returns undefined when unreadable. */
function readWorkspaceFile(
  workspace: string | undefined,
  rel: string,
): string | undefined {
  if (!workspace) {
    fail(
      `files[${rel}]`,
      "workspace is required for file content checks",
    );
    return undefined;
  }
  let full: string;
  try {
    full = containedPath(workspace, rel);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    fail(`files[${rel}]`, msg);
    return undefined;
  }
  try {
    return fs.readFileSync(full, "utf8");
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    fail(`files[${rel}]`, `read ${rel}: ${msg}`);
    return undefined;
  }
}

/** Join workspace + rel, rejecting absolute or escaping paths (Go parity). */
export function containedPath(workspace: string, rel: string): string {
  if (!rel) {
    throw new Error(`path ${JSON.stringify(rel)} must be relative to workspace`);
  }
  if (path.isAbsolute(rel)) {
    throw new Error(`path ${JSON.stringify(rel)} must be relative to workspace`);
  }
  const root = path.resolve(workspace);
  const full = path.resolve(root, rel);
  const relToRoot = path.relative(root, full);
  if (relToRoot === ".." || relToRoot.startsWith(`..${path.sep}`)) {
    throw new Error(`path ${JSON.stringify(rel)} must be relative to workspace`);
  }
  return full;
}
