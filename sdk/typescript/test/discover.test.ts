import assert from "node:assert/strict";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";

import {
  discoverEvalScripts,
  isDiscoverableEvalScript,
  isScriptEvalPath,
} from "../src/discover.js";

describe("isDiscoverableEvalScript", () => {
  it("matches eval.* and *.eval.* basenames", () => {
    assert.equal(isDiscoverableEvalScript("eval.ts"), true);
    assert.equal(isDiscoverableEvalScript("eval.mts"), true);
    assert.equal(isDiscoverableEvalScript("eval.js"), true);
    assert.equal(isDiscoverableEvalScript("eval.mjs"), true);
    assert.equal(isDiscoverableEvalScript("foo.eval.ts"), true);
    assert.equal(isDiscoverableEvalScript("bar.eval.js"), true);
  });

  it("rejects non-eval names and wrong extensions", () => {
    assert.equal(isDiscoverableEvalScript("evaluate.ts"), false);
    assert.equal(isDiscoverableEvalScript("eval.yaml"), false);
    assert.equal(isDiscoverableEvalScript("eval.test.ts"), false);
    assert.equal(isDiscoverableEvalScript("readme.md"), false);
  });
});

describe("isScriptEvalPath", () => {
  it("accepts script extensions regardless of basename", () => {
    assert.equal(isScriptEvalPath("my-suite.ts"), true);
    assert.equal(isScriptEvalPath("/abs/path/foo.mjs"), true);
    assert.equal(isScriptEvalPath("eval.yaml"), false);
  });
});

describe("discoverEvalScripts", () => {
  it("finds eval.ts and *.eval.ts, skips node_modules, dist, and hidden dirs", async () => {
    const root = await fs.mkdtemp(path.join(os.tmpdir(), "skilleval-discover-"));
    try {
      await fs.mkdir(path.join(root, "nested", "deep"), { recursive: true });
      await fs.mkdir(path.join(root, "node_modules", "pkg"), { recursive: true });
      await fs.mkdir(path.join(root, "dist"), { recursive: true });
      await fs.mkdir(path.join(root, ".hidden"), { recursive: true });

      await fs.writeFile(path.join(root, "eval.ts"), "");
      await fs.writeFile(path.join(root, "nested", "deep", "suite.eval.ts"), "");
      await fs.writeFile(path.join(root, "nested", "other.ts"), "");
      await fs.writeFile(path.join(root, "node_modules", "pkg", "eval.ts"), "");
      await fs.writeFile(path.join(root, "dist", "eval.js"), "");
      await fs.writeFile(path.join(root, ".hidden", "eval.ts"), "");

      const found = await discoverEvalScripts(root);
      assert.deepEqual(
        found.map((f) => path.relative(root, f)).sort(),
        ["eval.ts", path.join("nested", "deep", "suite.eval.ts")].sort(),
      );
    } finally {
      await fs.rm(root, { recursive: true, force: true });
    }
  });
});
