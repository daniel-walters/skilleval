import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";

import { loadDotEnv, parseEnvLine } from "../src/envfile.js";

describe("parseEnvLine", () => {
  it("parses KEY=VALUE", () => {
    assert.deepEqual(parseEnvLine("FOO=bar"), { key: "FOO", value: "bar" });
  });

  it("allows export prefix and quoted values", () => {
    assert.deepEqual(parseEnvLine('export FOO="bar baz"'), {
      key: "FOO",
      value: "bar baz",
    });
    assert.deepEqual(parseEnvLine("FOO='bar'"), { key: "FOO", value: "bar" });
  });

  it("skips blanks and comments", () => {
    assert.equal(parseEnvLine(""), null);
    assert.equal(parseEnvLine("  # comment"), null);
  });

  it("rejects unterminated quotes", () => {
    assert.throws(() => parseEnvLine('FOO="bar'), /unterminated quote/);
  });
});

describe("loadDotEnv", () => {
  it("no-ops when the file is missing", () => {
    loadDotEnv(path.join(os.tmpdir(), `skilleval-missing-${Date.now()}.env`));
  });

  it("fills unset keys and does not override process env", () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "skilleval-env-"));
    const file = path.join(dir, ".env");
    const fillKey = "SKILLEVAL_TEST_ENVFILE_FILL";
    const keepKey = "SKILLEVAL_TEST_ENVFILE_KEEP";
    fs.writeFileSync(file, `${fillKey}=fromfile\n${keepKey}=fromfile\n`, "utf8");

    const prevFill = process.env[fillKey];
    const prevKeep = process.env[keepKey];
    delete process.env[fillKey];
    process.env[keepKey] = "fromenv";

    try {
      loadDotEnv(file);
      assert.equal(process.env[fillKey], "fromfile");
      assert.equal(process.env[keepKey], "fromenv");
    } finally {
      if (prevFill === undefined) {
        delete process.env[fillKey];
      } else {
        process.env[fillKey] = prevFill;
      }
      if (prevKeep === undefined) {
        delete process.env[keepKey];
      } else {
        process.env[keepKey] = prevKeep;
      }
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });
});
