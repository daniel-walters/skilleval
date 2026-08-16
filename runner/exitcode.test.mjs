import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  exitCodeFromToolEvent,
  isShellToolName,
} from "./exitcode.mjs";

describe("isShellToolName", () => {
  it("accepts shell and Bash only", () => {
    assert.equal(isShellToolName("shell"), true);
    assert.equal(isShellToolName("Bash"), true);
    assert.equal(isShellToolName("bash"), false);
    assert.equal(isShellToolName("edit"), false);
  });
});

describe("exitCodeFromToolEvent", () => {
  it("returns undefined for non-shell tools", () => {
    assert.equal(exitCodeFromToolEvent("edit", { exitCode: 0 }), undefined);
  });

  it("reads exitCode including zero", () => {
    assert.equal(exitCodeFromToolEvent("shell", { exitCode: 0 }), 0);
    assert.equal(exitCodeFromToolEvent("Bash", { exit_code: 1 }), 1);
  });

  it("reads nested result fields", () => {
    assert.equal(
      exitCodeFromToolEvent("shell", { result: { exitCode: 2 } }),
      2,
    );
    assert.equal(
      exitCodeFromToolEvent("shell", { result: { exit_code: 127 } }),
      127,
    );
  });

  it("ignores floats, strings, and stdout-like content", () => {
    assert.equal(exitCodeFromToolEvent("shell", { exitCode: 1.5 }), undefined);
    assert.equal(exitCodeFromToolEvent("shell", { exitCode: "0" }), undefined);
    assert.equal(
      exitCodeFromToolEvent("shell", { content: "exit 0\n" }),
      undefined,
    );
  });
});
