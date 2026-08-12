import assert from "node:assert/strict";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";
import { normalizeToolArgs, relativizeToolPath } from "./toolargs.mjs";

describe("normalizeToolArgs", () => {
  it("returns undefined for missing or non-object input", () => {
    assert.equal(normalizeToolArgs(undefined), undefined);
    assert.equal(normalizeToolArgs(null), undefined);
    assert.equal(normalizeToolArgs("x"), undefined);
    assert.equal(normalizeToolArgs([]), undefined);
  });

  it("maps file_path to path", () => {
    assert.deepEqual(normalizeToolArgs({ file_path: "src/a.go" }), {
      path: "src/a.go",
    });
  });

  it("prefers path over file_path", () => {
    assert.deepEqual(
      normalizeToolArgs({ path: "a.ts", file_path: "b.ts" }),
      { path: "a.ts" },
    );
  });

  it("keeps command and other scalars", () => {
    assert.deepEqual(
      normalizeToolArgs({
        command: "go test",
        timeout: 30,
        ok: true,
        note: null,
      }),
      { command: "go test", timeout: 30, ok: true, note: null },
    );
  });

  it("strips large body fields", () => {
    assert.deepEqual(
      normalizeToolArgs({
        path: "f.ts",
        fileText: "huge",
        content: "huge",
        oldText: "a",
        newText: "b",
        old_string: "a",
        new_string: "b",
      }),
      { path: "f.ts" },
    );
  });

  it("drops nested objects and arrays", () => {
    assert.deepEqual(
      normalizeToolArgs({ path: "x", nested: { a: 1 }, list: [1] }),
      { path: "x" },
    );
  });

  it("returns undefined when everything was stripped", () => {
    assert.equal(normalizeToolArgs({ content: "only body" }), undefined);
  });

  it("relativizes absolute path under cwd", () => {
    const cwd = path.join(os.tmpdir(), "skilleval-ws");
    const abs = path.join(cwd, "src", "foo.go");
    assert.deepEqual(normalizeToolArgs({ path: abs }, cwd), {
      path: "src/foo.go",
    });
  });

  it("relativizes Claude file_path under cwd", () => {
    const cwd = path.join(os.tmpdir(), "skilleval-ws");
    const abs = path.join(cwd, "pkg", "a.go");
    assert.deepEqual(normalizeToolArgs({ file_path: abs }, cwd), {
      path: "pkg/a.go",
    });
  });

  it("leaves already-relative paths unchanged when under cwd", () => {
    const cwd = path.join(os.tmpdir(), "skilleval-ws");
    assert.deepEqual(normalizeToolArgs({ path: "src/foo.go" }, cwd), {
      path: "src/foo.go",
    });
  });

  it("leaves paths outside cwd unchanged", () => {
    const cwd = path.join(os.tmpdir(), "skilleval-ws");
    const outside = path.join(os.tmpdir(), "other", "x.go");
    assert.deepEqual(normalizeToolArgs({ path: outside }, cwd), {
      path: outside.split(path.sep).join("/"),
    });
  });
});

describe("relativizeToolPath", () => {
  it("no-ops without cwd", () => {
    assert.equal(relativizeToolPath("/tmp/a/b.go"), "/tmp/a/b.go");
  });
});
