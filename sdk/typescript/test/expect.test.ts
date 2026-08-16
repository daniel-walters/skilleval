import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { afterEach, describe, it } from "node:test";

import { expect, ExpectError } from "../src/expect.js";
import type { Result } from "../src/result.js";

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

afterEach(() => {
  expect.clear();
});

function assertFail(fn: () => void, pathPrefix: string): ExpectError {
  expect.clear();
  try {
    fn();
  } catch (err) {
    assert.ok(err instanceof ExpectError, `got ${String(err)}`);
    assert.equal(err.path, pathPrefix);
    return err;
  }
  try {
    expect.report();
    assert.fail("expected ExpectError");
  } catch (err) {
    assert.ok(err instanceof ExpectError, `got ${String(err)}`);
    assert.equal(err.path, pathPrefix);
    return err;
  }
}

describe("expect status gate", () => {
  it("fails run.status for error results before other checks", () => {
    const r = loadResult("error-status");
    assertFail(() => expect(r).turns.toBeLessThanOrEqual(10), "run.status");
  });

  it("fails run.status for cancelled results", () => {
    const r = loadResult("cancelled-status");
    assertFail(() => expect(r).finalMessage.toContain("x"), "run.status");
  });
});

describe("expect turns / costUSD", () => {
  it("passes numeric bounds on pass fixture", () => {
    const r = loadResult("pass");
    expect(r).turns.toBeLessThanOrEqual(15).toBeGreaterThanOrEqual(1);
    expect(r).turns.toBeGreaterThan(0).toBeLessThan(20);
    expect(r).costUSD.toBeLessThanOrEqual(1).toBeGreaterThan(0);
  });

  it("passes full bound suite on pass-numeric-bounds", () => {
    const r = loadResult("pass-numeric-bounds");
    expect(r).turns.toBeGreaterThanOrEqual(1).toBeLessThanOrEqual(15);
    expect(r).turns.toBeGreaterThan(0).toBeLessThan(20).toBeEqual(4);
    expect(r).costUSD.toBeGreaterThanOrEqual(0).toBeLessThanOrEqual(1);
    expect(r).costUSD.toBeGreaterThan(0).toBeLessThan(2).toBeEqual(0.25);
  });

  it("passes durationMs, toolCalls count, and usage on pass-extended-metrics", () => {
    const r = loadResult("pass-extended-metrics");
    expect(r).durationMs.toBeGreaterThanOrEqual(1000).toBeLessThanOrEqual(20000);
    expect(r).toolCalls.toBeGreaterThanOrEqual(1).toBeLessThanOrEqual(5);
    expect(r).usage.inputTokens.toBeLessThanOrEqual(200);
    expect(r).usage.totalTokens.toBeEqual(150);
  });

  it("fails turns.max", () => {
    const r = loadResult("fail-turns");
    assertFail(() => expect(r).turns.toBeLessThanOrEqual(5), "turns.max");
  });

  it("fails turns.min", () => {
    const r = loadResult("fail-turns-min");
    assertFail(() => expect(r).turns.toBeGreaterThanOrEqual(10), "turns.min");
  });

  it("fails turns.gt", () => {
    const r = loadResult("fail-turns-gt");
    assertFail(() => expect(r).turns.toBeGreaterThan(10), "turns.gt");
  });

  it("fails turns.lt", () => {
    const r = loadResult("fail-turns-lt");
    assertFail(() => expect(r).turns.toBeLessThan(3), "turns.lt");
  });

  it("fails turns.eq", () => {
    const r = loadResult("fail-turns-eq");
    assertFail(() => expect(r).turns.toBeEqual(3), "turns.eq");
  });

  it("fails durationMs.max", () => {
    const r = loadResult("fail-duration-ms");
    assertFail(() => expect(r).durationMs.toBeLessThanOrEqual(1000), "durationMs.max");
  });

  it("fails toolCalls.min", () => {
    const r = loadResult("fail-tool-calls");
    assertFail(() => expect(r).toolCalls.toBeGreaterThanOrEqual(1), "toolCalls.min");
  });

  it("fails usage bounds", () => {
    const r = loadResult("fail-usage");
    assertFail(() => expect(r).usage.inputTokens.toBeLessThanOrEqual(50), "usage.inputTokens.max");
    assertFail(() => expect(r).usage.totalTokens.toBeLessThan(100), "usage.totalTokens.lt");
  });

  it("fails costUSD.max", () => {
    const r = loadResult("fail-cost-exceeded");
    assertFail(() => expect(r).costUSD.toBeLessThanOrEqual(0.1), "costUSD.max");
  });

  it("fails nil costUSD for any bound", () => {
    const r = loadResult("fail-null-cost");
    assertFail(() => expect(r).costUSD.toBeLessThanOrEqual(1), "costUSD.max");
    const r2 = loadResult("fail-null-cost-bounds");
    assertFail(() => expect(r2).costUSD.toBeGreaterThanOrEqual(0), "costUSD.min");
    assertFail(() => expect(r2).costUSD.toBeGreaterThan(0), "costUSD.gt");
  });
});

describe("expect tools and skills", () => {
  it("passes include / not include on pass", () => {
    const r = loadResult("pass");
    expect(r).toolsUsed.toInclude("read", "edit");
    expect(r).toolsUsed.not.toInclude("web");
    expect(r).skills.activated.toInclude("helper");
    expect(r).skills.activated.not.toInclude("other-skill");
  });

  it("fails toolsUsed.includes and excludes", () => {
    const r = loadResult("fail-tools");
    assertFail(() => expect(r).toolsUsed.toInclude("edit"), "toolsUsed.includes");
    assertFail(() => expect(r).toolsUsed.not.toInclude("web"), "toolsUsed.excludes");
  });

  it("fails skills.activated.includes", () => {
    const r = loadResult("fail-skills");
    assertFail(
      () => expect(r).skills.activated.toInclude("helper"),
      "skills.activated.includes",
    );
  });

  it("fails skills.activated.excludes", () => {
    const r = loadResult("fail-skills-excludes");
    assertFail(
      () => expect(r).skills.activated.not.toInclude("helper"),
      "skills.activated.excludes",
    );
  });
});

describe("expect toolCalls order / named", () => {
  it("passes subsequence, named counts, and arg predicates", () => {
    const r = loadResult("pass-tool-calls-order");
    expect(r).toolCalls.toBeGreaterThanOrEqual(2);
    expect(r).toolCalls.named("edit").toBeGreaterThanOrEqual(1).toBeLessThanOrEqual(2);
    expect(r).toolCalls.named("shell").toBeEqual(1);
    expect(r).toolCalls.toIncludeInOrder([
      { name: "edit", args: { path: "src/foo.go" } },
      { name: "shell", args: { command: /go test/ } },
    ]);
  });

  it("fails when order is reversed", () => {
    const r = loadResult("fail-tool-calls-order");
    assertFail(
      () =>
        expect(r).toolCalls.toIncludeInOrder([
          { name: "edit", args: { path: "src/foo.go" } },
          { name: "shell", args: { command: /go test/ } },
        ]),
      "toolCalls.order[1]",
    );
  });

  it("fails named count bounds", () => {
    const r = loadResult("fail-tool-calls-named");
    assertFail(() => expect(r).toolCalls.named("edit").toBeGreaterThanOrEqual(2), "toolCalls.named.edit.min");
  });

  it("fails when arg predicate does not match", () => {
    const r = loadResult("fail-tool-calls-args");
    assertFail(
      () =>
        expect(r).toolCalls.toIncludeInOrder([
          { name: "edit", args: { path: "src/wrong.go" } },
        ]),
      "toolCalls.order[0]",
    );
  });

  it("matches the first name in a list", () => {
    const r = loadResult("pass-tool-calls-order-first-name");
    expect(r).toolCalls.toIncludeInOrder([
      { name: ["write", "edit"], args: { path: "src/foo.go" } },
    ]);
  });

  it("matches a later name in a list", () => {
    const r = loadResult("pass-tool-calls-order-later-name");
    expect(r).toolCalls.toIncludeInOrder([
      { name: ["write", "edit"], args: { path: "src/foo.go" } },
    ]);
  });

  it("fails when no name in the list matches", () => {
    const r = loadResult("fail-tool-calls-order-names");
    assertFail(
      () =>
        expect(r).toolCalls.toIncludeInOrder([
          { name: ["write", "edit"], args: { path: "src/foo.go" } },
        ]),
      "toolCalls.order[0]",
    );
  });

  it("throws for a blank name instead of failing the check", () => {
    const r = loadResult("pass-tool-calls-order");
    assert.throws(
      () => expect(r).toolCalls.toIncludeInOrder([{ name: "" }]),
      { message: "toIncludeInOrder: name is required" },
    );
  });

  it("throws for an empty name list instead of failing the check", () => {
    const r = loadResult("pass-tool-calls-order");
    const empty = [] as unknown as [string, ...string[]];
    assert.throws(
      () => expect(r).toolCalls.toIncludeInOrder([{ name: empty }]),
      { message: "toIncludeInOrder: name is required" },
    );
  });
});

describe("expect files", () => {
  it("passes status and content on pass fixture", () => {
    const r = loadResult("pass");
    const ws = workspace("pass");
    expect(r, ws).file("src/foo.go").toHaveBeenModified().toContain("package");
    expect(r, ws).file("src/new.go").toHaveBeenCreated();
    expect(r, ws).file("src/gone.go").toHaveBeenDeleted();
  });

  it("passes regex content on pass-regex", () => {
    const r = loadResult("pass-regex");
    const ws = workspace("pass-regex");
    expect(r, ws).file("src/foo.go").toContain(/func\s+Foo/);
  });

  it("fails missing file status", () => {
    const r = loadResult("fail-file-missing");
    assertFail(
      () => expect(r).file("src/missing.go").toHaveBeenCreated(),
      "files[src/missing.go].status",
    );
  });

  it("fails wrong file status", () => {
    const r = loadResult("fail-file-status");
    assertFail(
      () => expect(r).file("src/foo.go").toHaveStatus("modified"),
      "files[src/foo.go].status",
    );
  });

  it("rejects content on deleted files", () => {
    const r = loadResult("fail-deleted-contains");
    assertFail(
      () => expect(r, workspace("pass")).file("src/gone.go").toContain("x"),
      "files[src/gone.go]",
    );
  });

  it("requires workspace for content checks", () => {
    const r = loadResult("pass");
    assertFail(() => expect(r).file("src/foo.go").toContain("package"), "files[src/foo.go]");
  });

  it("fails file contains", () => {
    const r = loadResult("fail-file-contains");
    const ws = workspace("fail-file-contains");
    assertFail(
      () => expect(r, ws).file("src/foo.go").toContain("NOPE"),
      "files[src/foo.go].contains",
    );
  });

  it("passes file excludes on pass fixture", () => {
    const r = loadResult("pass");
    const ws = workspace("pass");
    expect(r, ws).file("src/foo.go").not.toContain("TODO", /FIXME\d+/);
  });

  it("no-ops empty not.toContain", () => {
    const r = loadResult("pass");
    const ws = workspace("pass");
    expect(r, ws).file("src/foo.go").not.toContain();
  });

  it("fails file excludes", () => {
    const r = loadResult("fail-file-excludes");
    const ws = workspace("fail-file-excludes");
    assertFail(
      () => expect(r, ws).file("src/foo.go").not.toContain("TODO", /FIXME\d+/),
      "files[src/foo.go].excludes",
    );
  });
});

describe("expect finalMessage", () => {
  it("passes contain / match / equal on pass", () => {
    const r = loadResult("pass");
    expect(r).finalMessage.toContain("Done");
    expect(r).finalMessage.toMatch(/refactor/i);
    expect(r).finalMessage.toEqual("Done with refactor.");
  });

  it("fails contains", () => {
    const r = loadResult("fail-final-message");
    assertFail(() => expect(r).finalMessage.toContain("SUCCESS"), "finalMessage.contains");
  });

  it("fails regex contains", () => {
    const r = loadResult("fail-final-message-regex");
    assertFail(() => expect(r).finalMessage.toMatch(/SUCCESS/), "finalMessage.contains");
  });
});

describe("expect collects all failures", () => {
  it("reports every failure across sequential expect calls", () => {
    const r = loadResult("fail-many");
    expect(r).turns.toBeLessThanOrEqual(1);
    expect(r).costUSD.toBeLessThanOrEqual(0.1);
    expect(r).finalMessage.toContain("Done");
    const err = (() => {
      try {
        expect.report();
        assert.fail("expected ExpectError");
      } catch (e) {
        assert.ok(e instanceof ExpectError);
        return e;
      }
    })();
    assert.deepEqual(
      err.failures.map((f) => f.path),
      ["turns.max", "costUSD.max", "finalMessage.contains"],
    );
    assert.match(err.message, /turns\.max:/);
    assert.match(err.message, /costUSD\.max:/);
    assert.match(err.message, /finalMessage\.contains:/);
  });

  it("reports every missing tool in one toInclude call", () => {
    const r = loadResult("fail-tools");
    expect(r).toolsUsed.toInclude("edit", "shell");
    const err = (() => {
      try {
        expect.report();
        assert.fail("expected ExpectError");
      } catch (e) {
        assert.ok(e instanceof ExpectError);
        return e;
      }
    })();
    assert.equal(err.failures.length, 2);
    assert.equal(err.failures[0]!.path, "toolsUsed.includes");
    assert.equal(err.failures[1]!.path, "toolsUsed.includes");
    assert.match(err.failures[0]!.reason, /edit/);
    assert.match(err.failures[1]!.reason, /shell/);
  });
});
