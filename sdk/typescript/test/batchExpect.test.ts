import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { afterEach, describe, it } from "node:test";

import { expect, expectAttempts } from "../src/expect.js";
import type { AttemptOutcome, Result } from "../src/result.js";
import { createRunResult } from "../src/run.js";

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

function loadResult(name: string): Result {
  const raw = fs.readFileSync(path.join(fixtures, name, "result.json"), "utf8");
  return JSON.parse(raw) as Result;
}

function workspace(name: string): string {
  return path.join(fixtures, name, "workspace");
}

function withTurns(base: Result, turns: number): Result {
  return {
    ...base,
    metrics: { ...base.metrics, turns },
  };
}

function slot(result: Result, ws = "/tmp/ws"): AttemptOutcome {
  return { result, workspace: ws };
}

afterEach(() => {
  expect.clear();
});

describe("expectAttempts", () => {
  const pass = loadResult("pass");
  const ws = workspace("pass");
  const failTurns = withTurns(pass, 20);

  function turnsMax15({ result, workspace }: { result: Result; workspace: string }): void {
    expect(result, workspace).turns.toBeLessThanOrEqual(15);
  }

  it("meets min 0.8 when 8/10 pass and does not collect process failures", () => {
    const attempts: AttemptOutcome[] = [
      ...Array.from({ length: 8 }, () => slot(pass, ws)),
      slot(failTurns, ws),
      slot(failTurns, ws),
    ];
    const lines: string[] = [];
    const report = expectAttempts(attempts, turnsMax15, { min: 0.8 }, (l) => lines.push(l));
    assert.equal(report.passed, 8);
    assert.equal(report.attempts, 10);
    assert.equal(report.passRate, 0.8);
    assert.equal(expect.failures().length, 0);
    assert.ok(lines.some((l) => l.startsWith("attempt 9/10: FAIL")));
    assert.ok(lines.some((l) => l.includes("passRate: 0.8 (8/10)")));
  });

  it("fails passRate.min when 7/10 is below 0.8", () => {
    const attempts: AttemptOutcome[] = [
      ...Array.from({ length: 7 }, () => slot(pass, ws)),
      slot(failTurns, ws),
      slot(failTurns, ws),
      slot(failTurns, ws),
    ];
    const report = expectAttempts(attempts, turnsMax15, { min: 0.8 }, () => undefined);
    assert.equal(report.passed, 7);
    assert.equal(report.passRate, 0.7);
    const failures = expect.failures();
    assert.equal(failures.length, 1);
    assert.equal(failures[0]!.path, "passRate.min");
    assert.match(failures[0]!.reason, /pass rate 0\.7 below min 0\.8/);
    assert.match(failures[0]!.reason, /failed attempts 8, 9, 10/);
  });

  it("defaults min to 1 so any failed attempt fails the batch", () => {
    const attempts: AttemptOutcome[] = [slot(pass, ws), slot(failTurns, ws)];
    const report = expectAttempts(attempts, turnsMax15, undefined, () => undefined);
    assert.equal(report.passed, 1);
    assert.equal(report.passRate, 0.5);
    assert.equal(expect.failures()[0]!.path, "passRate.min");
  });

  it("does not skip later attempts after failHard on a non-finished result", () => {
    const err = loadResult("error-status");
    const seen: string[] = [];
    const attempts: AttemptOutcome[] = [slot(pass, ws), slot(err, ws), slot(pass, ws)];
    const report = expectAttempts(
      attempts,
      ({ result, workspace }) => {
        seen.push(result.id);
        expect(result, workspace).turns.toBeLessThanOrEqual(15);
      },
      { min: 0.5 },
      () => undefined,
    );
    assert.deepEqual(seen, ["run_pass", err.id, "run_pass"]);
    assert.equal(report.passed, 2);
    assert.equal(expect.failures().length, 0);
  });

  it("counts runner-error slots as failed and does not invoke fn", () => {
    let calls = 0;
    const attempts: AttemptOutcome[] = [
      slot(pass, ws),
      { error: "agent crashed" },
      slot(pass, ws),
    ];
    const lines: string[] = [];
    const report = expectAttempts(
      attempts,
      () => {
        calls++;
      },
      { min: 0.5 },
      (l) => lines.push(l),
    );
    assert.equal(calls, 2);
    assert.equal(report.passed, 2);
    assert.ok(lines.some((l) => l === "attempt 2/3: FAIL"));
    assert.ok(lines.some((l) => l.includes("run.error: agent crashed")));
    assert.equal(expect.failures().length, 0);
  });
});

describe("createRunResult.expect", () => {
  it("uses passRate from the batch", () => {
    const pass = loadResult("pass");
    const ws = workspace("pass");
    const batch = createRunResult({
      result: pass,
      workspace: ws,
      attempts: [slot(pass, ws), slot(withTurns(pass, 20), ws)],
      passRate: { min: 0.5 },
      exitCode: 0,
    });
    const orig = console.log;
    console.log = () => undefined;
    try {
      const report = batch.expect(({ result, workspace }) => {
        expect(result, workspace).turns.toBeLessThanOrEqual(15);
      });
      assert.equal(report.passed, 1);
      assert.equal(expect.failures().length, 0);
    } finally {
      console.log = orig;
    }
  });
});
