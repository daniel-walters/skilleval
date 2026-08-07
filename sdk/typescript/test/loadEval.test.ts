import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { describe, it } from "node:test";

import { loadEval } from "../src/loadEval.js";

const here = path.dirname(fileURLToPath(import.meta.url));

function findFixtures(start: string): string {
  let dir = start;
  for (;;) {
    const candidate = path.join(dir, "checker", "testdata");
    if (fs.existsSync(candidate)) {
      return candidate;
    }
    const parent = path.dirname(dir);
    if (parent === dir) {
      throw new Error("checker/testdata not found");
    }
    dir = parent;
  }
}

const fixtures = findFixtures(here);

describe("loadEval", () => {
  it("loads a fixture eval and records sourcePath", async () => {
    const evalPath = path.join(fixtures, "pass", "eval.yaml");
    const ev = await loadEval(evalPath);
    assert.equal(ev.schemaVersion, 1);
    assert.equal(ev.name, "pass");
    assert.equal(ev.sourcePath, path.resolve(evalPath));
    assert.ok(ev.skill);
  });

  it("rejects missing name", async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "skilleval-load-"));
    const p = path.join(dir, "bad.yaml");
    fs.writeFileSync(p, "schemaVersion: 1\nprompt: hi\nskill: ./x\n");
    await assert.rejects(() => loadEval(p), /name is required/);
  });
});
