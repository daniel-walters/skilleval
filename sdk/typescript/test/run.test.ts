import assert from "node:assert/strict";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";

import type { EvalDocument, RunOptions } from "../src/evalTypes.js";
import {
  appendReportFlags,
  parseAttemptSlots,
  parseWroteLines,
  resolveEvalAndFlags,
  shouldWriteTempEval,
  summaryOutPath,
} from "../src/run.js";
import { SCRIPT_EVAL_DIR_ENV } from "../src/evalpath.js";
import {
  matchContains,
  matchEquals,
} from "../src/stringMatch.js";

describe("shouldWriteTempEval", () => {
  const programmatic: RunOptions = {
    name: "n",
    prompt: "p",
    skill: "./skill",
    model: "composer-2",
  };

  const loaded: EvalDocument = {
    schemaVersion: 1,
    name: "n",
    prompt: "p",
    skill: "./skill",
    sourcePath: "/abs/eval.yaml",
  };

  it("writes temp YAML for programmatic RunOptions", () => {
    assert.equal(shouldWriteTempEval(programmatic), true);
  });

  it("still treats programmatic opts as temp when model is missing", () => {
    const { model: _omit, ...noModel } = programmatic;
    assert.equal(shouldWriteTempEval(noModel as RunOptions), true);
  });

  it("still treats programmatic opts as temp when model is not a string", () => {
    assert.equal(
      shouldWriteTempEval({ ...programmatic, model: undefined as unknown as string }),
      true,
    );
  });

  it("uses on-disk file when sourcePath is set", () => {
    assert.equal(shouldWriteTempEval(loaded, { model: "composer-2" }), false);
  });

  it("does not rewrite when sourcePath is present even with a model field", () => {
    const merged = { ...loaded, model: "composer-2" };
    assert.equal(shouldWriteTempEval(merged), false);
  });
});

describe("resolveEvalAndFlags", () => {
  it("rejects missing model on programmatic run with a clear error", async () => {
    await assert.rejects(
      () =>
        resolveEvalAndFlags({
          name: "n",
          prompt: "p",
          skill: "./skill",
        } as RunOptions),
      /run: model is required/,
    );
  });

  it("rejects non-string model on programmatic run", async () => {
    await assert.rejects(
      () =>
        resolveEvalAndFlags({
          name: "n",
          prompt: "p",
          skill: "./skill",
          model: undefined as unknown as string,
        }),
      /run: model is required/,
    );
  });

  it("writes replies into temp eval YAML", async () => {
    const { evalPath, cleanup } = await resolveEvalAndFlags({
      name: "n",
      prompt: "p",
      skill: "./skill",
      model: "composer-2",
      replies: ["yes", "proceed"],
    });
    try {
      const raw = await fs.readFile(evalPath, "utf8");
      assert.match(raw, /replies:/);
      assert.match(raw, /- yes/);
      assert.match(raw, /- proceed/);
    } finally {
      if (cleanup) {
        await fs.rm(cleanup, { recursive: true, force: true }).catch(() => undefined);
      }
    }
  });

  it("writes attempts into temp YAML and omits passRate", async () => {
    const { evalPath, cleanup, passRate } = await resolveEvalAndFlags({
      name: "n",
      prompt: "p",
      skill: "./skill",
      model: "composer-2",
      attempts: 10,
      passRate: { min: 0.8 },
    });
    try {
      const raw = await fs.readFile(evalPath, "utf8");
      assert.match(raw, /attempts: 10/);
      assert.doesNotMatch(raw, /passRate/);
      assert.deepEqual(passRate, { min: 0.8 });
    } finally {
      if (cleanup) {
        await fs.rm(cleanup, { recursive: true, force: true }).catch(() => undefined);
      }
    }
  });

  it("returns passRate from a loaded eval without rewriting YAML", async () => {
    const ev: EvalDocument = {
      schemaVersion: 1,
      name: "n",
      prompt: "p",
      skill: "./skill",
      sourcePath: "/abs/eval.yaml",
      passRate: { min: 0.5 },
    };
    const { evalPath, passRate } = await resolveEvalAndFlags(ev, { model: "composer-2" });
    assert.equal(evalPath, "/abs/eval.yaml");
    assert.deepEqual(passRate, { min: 0.5 });
  });

  it("resolves relative skill/input from SKILLEVAL_EVAL_DIR, not cwd", async () => {
    const prev = process.env[SCRIPT_EVAL_DIR_ENV];
    const evalDir = await fs.mkdtemp(path.join(os.tmpdir(), "skilleval-evaldir-"));
    process.env[SCRIPT_EVAL_DIR_ENV] = evalDir;
    try {
      const { evalPath, cleanup } = await resolveEvalAndFlags({
        name: "n",
        prompt: "p",
        skill: "./skills/demo",
        input: "./fixtures/demo",
        model: "composer-2",
      });
      try {
        const raw = await fs.readFile(evalPath, "utf8");
        assert.ok(raw.includes(path.join(evalDir, "skills", "demo")));
        assert.ok(raw.includes(path.join(evalDir, "fixtures", "demo")));
      } finally {
        if (cleanup) {
          await fs.rm(cleanup, { recursive: true, force: true }).catch(() => undefined);
        }
      }
    } finally {
      if (prev === undefined) {
        delete process.env[SCRIPT_EVAL_DIR_ENV];
      } else {
        process.env[SCRIPT_EVAL_DIR_ENV] = prev;
      }
      await fs.rm(evalDir, { recursive: true, force: true }).catch(() => undefined);
    }
  });
});

describe("parseWroteLines", () => {
  it("parses a single-attempt wrote line", () => {
    const stdout = `wrote /tmp/out/result.json (status=finished workspace=/tmp/skilleval-abc)\nPASS\n`;
    const lines = parseWroteLines(stdout);
    assert.equal(lines.length, 1);
    assert.equal(lines[0]!.outPath, "/tmp/out/result.json");
    assert.equal(lines[0]!.workspace, "/tmp/skilleval-abc");
  });

  it("parses wrote lines that include agentLog", () => {
    const stdout = `wrote /tmp/out/result.json (status=finished workspace=/tmp/w agentLog=/tmp/out/result-agent-log.json)\n`;
    const lines = parseWroteLines(stdout);
    assert.equal(lines.length, 1);
    assert.equal(lines[0]!.outPath, "/tmp/out/result.json");
    assert.equal(lines[0]!.workspace, "/tmp/w");
  });

  it("parses multi-attempt wrote lines and keeps order", () => {
    const stdout = `
attempt 1/2: wrote /tmp/result-1.json (status=finished workspace=/tmp/w1)
PASS
attempt 2/2: wrote /tmp/result-2.json (status=error workspace=/tmp/w2)
FAIL
  run.status: status is "error", want "finished"
`;
    const lines = parseWroteLines(stdout);
    assert.equal(lines.length, 2);
    assert.equal(lines[1]!.outPath, "/tmp/result-2.json");
    assert.equal(lines[1]!.workspace, "/tmp/w2");
  });

  it("ignores summary-only wrote lines without workspace", () => {
    const stdout = `wrote /tmp/result-summary.json\n`;
    assert.equal(parseWroteLines(stdout).length, 0);
  });
});

describe("parseAttemptSlots", () => {
  it("parses mixed wrote and error lines into N slots", () => {
    const stdout = `
attempt 1/3: wrote /tmp/result-1.json (status=finished workspace=/tmp/w1)
PASS
attempt 2/3: error: agent crashed
attempt 3/3: wrote /tmp/result-3.json (status=finished workspace=/tmp/w3)
PASS
`;
    const slots = parseAttemptSlots(stdout);
    assert.equal(slots.length, 3);
    assert.equal(slots[0]!.outPath, "/tmp/result-1.json");
    assert.equal(slots[0]!.workspace, "/tmp/w1");
    assert.equal(slots[1]!.error, "agent crashed");
    assert.equal(slots[1]!.outPath, undefined);
    assert.equal(slots[2]!.outPath, "/tmp/result-3.json");
  });

  it("fills missing attempt indexes as no Result", () => {
    const stdout = `attempt 1/3: wrote /tmp/result-1.json (status=finished workspace=/tmp/w1)
attempt 3/3: wrote /tmp/result-3.json (status=finished workspace=/tmp/w3)
`;
    const slots = parseAttemptSlots(stdout);
    assert.equal(slots.length, 3);
    assert.equal(slots[1]!.error, "no Result");
  });

  it("treats unprefixed wrote lines as a single attempt", () => {
    const stdout = `wrote /tmp/out/result.json (status=finished workspace=/tmp/w)\n`;
    const slots = parseAttemptSlots(stdout);
    assert.equal(slots.length, 1);
    assert.equal(slots[0]!.attempt, 1);
    assert.equal(slots[0]!.outPath, "/tmp/out/result.json");
  });
});

describe("summaryOutPath", () => {
  it("inserts -summary before extension", () => {
    assert.equal(summaryOutPath("/tmp/result.json"), "/tmp/result-summary.json");
    assert.equal(summaryOutPath("out"), "out-summary");
  });
});

describe("appendReportFlags", () => {
  it("omits flags so CLI defaults apply", () => {
    const args: string[] = [];
    appendReportFlags(args, {});
    assert.deepEqual(args, []);
  });

  it("forwards opt-outs", () => {
    const args: string[] = [];
    appendReportFlags(args, { noHistory: true, noBaseline: true });
    assert.deepEqual(args, ["--no-history", "--no-baseline"]);
  });

  it("forwards path overrides when not opting out", () => {
    const args: string[] = [];
    appendReportFlags(args, { history: "/tmp/hist", baseline: "/tmp/base.json" });
    assert.equal(args[0], "--history");
    assert.equal(args[2], "--baseline");
    assert.match(args[1]!, /hist$/);
    assert.match(args[3]!, /base\.json$/);
  });

  it("prefers opt-outs over path overrides", () => {
    const args: string[] = [];
    appendReportFlags(args, {
      noHistory: true,
      history: "/tmp/hist",
      noBaseline: true,
      baseline: "/tmp/base.json",
    });
    assert.deepEqual(args, ["--no-history", "--no-baseline"]);
  });
});

describe("stringMatch", () => {
  it("literal contains and equals", () => {
    assert.equal(matchContains("hello world", "world"), true);
    assert.equal(matchContains("hello", "x"), false);
    assert.equal(matchEquals("hello", "hello"), true);
    assert.equal(matchEquals("hello", "hell"), false);
  });

  it("regex contains is substring; equals is full-string", () => {
    assert.equal(matchContains("abc123", /\d+/), true);
    assert.equal(matchEquals("abc123", /\d+/), false);
    assert.equal(matchEquals("123", /^\d+$/), true);
  });
});
