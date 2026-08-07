import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { parseWroteLines, summaryOutPath } from "../src/run.js";
import {
  matchContains,
  matchEquals,
} from "../src/stringMatch.js";

describe("parseWroteLines", () => {
  it("parses a single-attempt wrote line", () => {
    const stdout = `wrote /tmp/out/result.json (status=finished workspace=/tmp/skilleval-abc)\nPASS\n`;
    const lines = parseWroteLines(stdout);
    assert.equal(lines.length, 1);
    assert.equal(lines[0]!.outPath, "/tmp/out/result.json");
    assert.equal(lines[0]!.workspace, "/tmp/skilleval-abc");
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
