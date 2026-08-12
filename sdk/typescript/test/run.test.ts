import assert from "node:assert/strict";
import fs from "node:fs/promises";
import { describe, it } from "node:test";

import type { EvalDocument, RunOptions } from "../src/evalTypes.js";
import {
  appendReportFlags,
  parseWroteLines,
  resolveEvalAndFlags,
  shouldWriteTempEval,
  summaryOutPath,
} from "../src/run.js";
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
