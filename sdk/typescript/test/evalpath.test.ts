import assert from "node:assert/strict";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";

import {
  SCRIPT_EVAL_DIR_ENV,
  resolveEvalRel,
  scriptEvalDir,
} from "../src/evalpath.js";

describe("resolveEvalRel", () => {
  it("returns empty and absolute paths unchanged", () => {
    assert.equal(resolveEvalRel(""), "");
    assert.equal(resolveEvalRel("/abs/skill"), "/abs/skill");
  });

  it("joins relative paths to the given eval dir", () => {
    const dir = path.join(os.tmpdir(), "suite");
    assert.equal(resolveEvalRel("./skill", dir), path.join(dir, "skill"));
    assert.equal(
      resolveEvalRel("../.claude/skills/x", dir),
      path.resolve(dir, "../.claude/skills/x"),
    );
  });
});

describe("scriptEvalDir", () => {
  it("uses SKILLEVAL_EVAL_DIR when set, else cwd", () => {
    const prev = process.env[SCRIPT_EVAL_DIR_ENV];
    const nested = path.join(os.tmpdir(), "eval-pkg");
    try {
      delete process.env[SCRIPT_EVAL_DIR_ENV];
      assert.equal(scriptEvalDir(), process.cwd());
      process.env[SCRIPT_EVAL_DIR_ENV] = nested;
      assert.equal(scriptEvalDir(), path.resolve(nested));
    } finally {
      if (prev === undefined) {
        delete process.env[SCRIPT_EVAL_DIR_ENV];
      } else {
        process.env[SCRIPT_EVAL_DIR_ENV] = prev;
      }
    }
  });
});
