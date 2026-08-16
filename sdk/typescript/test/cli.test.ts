import assert from "node:assert/strict";
import { EventEmitter } from "node:events";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { describe, it } from "node:test";
import type { ChildProcess } from "node:child_process";

import {
  main,
  parseRunArgv,
  scriptNodeArgs,
  type CliDeps,
  type SpawnFn,
} from "../src/cli.js";

function fakeChild(exitCode: number): ChildProcess {
  const ee = new EventEmitter() as ChildProcess;
  queueMicrotask(() => ee.emit("close", exitCode, null));
  return ee;
}

describe("parseRunArgv", () => {
  it("splits flags before and after the path", () => {
    assert.deepEqual(parseRunArgv(["--model", "m", "eval.yaml"]), {
      flags: ["--model", "m"],
      positional: "eval.yaml",
    });
    assert.deepEqual(parseRunArgv(["eval.ts", "--no-history"]), {
      flags: ["--no-history"],
      positional: "eval.ts",
    });
  });

  it("treats single-dash Go flags as value-taking", () => {
    assert.deepEqual(parseRunArgv(["-model", "m", "eval.yaml"]), {
      flags: ["-model", "m"],
      positional: "eval.yaml",
    });
    assert.deepEqual(parseRunArgv(["eval.yaml", "-runner", "claude"]), {
      flags: ["-runner", "claude"],
      positional: "eval.yaml",
    });
  });

  it("allows no positional", () => {
    assert.deepEqual(parseRunArgv([]), {
      flags: [],
      positional: undefined,
    });
  });

  it("marks multiple positionals as ambiguous", () => {
    const r = parseRunArgv(["a.ts", "b.ts"]);
    assert.equal(r.positional, undefined);
    assert.equal(r.ambiguous, true);
  });
});

describe("scriptNodeArgs", () => {
  it("uses tsx import for TypeScript", () => {
    const { command, args } = scriptNodeArgs("/tmp/eval.ts", () => "/pkg/tsx/esm");
    assert.equal(command, process.execPath);
    assert.deepEqual(args, ["--import", "/pkg/tsx/esm", "/tmp/eval.ts"]);
  });

  it("uses node alone for JavaScript", () => {
    const { command, args } = scriptNodeArgs("/tmp/eval.js", () => "/pkg/tsx/esm");
    assert.equal(command, process.execPath);
    assert.deepEqual(args, ["/tmp/eval.js"]);
  });
});

describe("main routing", () => {
  it("forwards YAML run to the Go binary", async () => {
    const calls: { cmd: string; args: string[] }[] = [];
    const spawn: SpawnFn = ((cmd, args) => {
      calls.push({ cmd: String(cmd), args: args as string[] });
      return fakeChild(0);
    }) as SpawnFn;

    const deps: CliDeps = {
      spawn,
      resolveBinary: () => "/fake/skilleval",
      resolveTsxImport: () => "/fake/tsx",
      cwd: () => "/proj",
      stdout: () => undefined,
      stderr: () => undefined,
    };

    const code = await main(["run", "eval.yaml", "--model", "m"], deps);
    assert.equal(code, 0);
    assert.deepEqual(calls, [
      { cmd: "/fake/skilleval", args: ["run", "eval.yaml", "--model", "m"] },
    ]);
  });

  it("forwards YAML run with single-dash -model to Go", async () => {
    const calls: string[][] = [];
    const spawn: SpawnFn = ((_cmd, args) => {
      calls.push(args as string[]);
      return fakeChild(0);
    }) as SpawnFn;

    const code = await main(["run", "-model", "m", "eval.yaml"], {
      spawn,
      resolveBinary: () => "/fake/skilleval",
      resolveTsxImport: () => "/fake/tsx",
      cwd: () => "/proj",
      stdout: () => undefined,
      stderr: () => undefined,
    });
    assert.equal(code, 0);
    assert.deepEqual(calls, [["run", "-model", "m", "eval.yaml"]]);
  });

  it("forwards run --help to Go", async () => {
    const calls: string[][] = [];
    const spawn: SpawnFn = ((_cmd, args) => {
      calls.push(args as string[]);
      return fakeChild(0);
    }) as SpawnFn;

    const code = await main(["run", "--help"], {
      spawn,
      resolveBinary: () => "/fake/skilleval",
      resolveTsxImport: () => "/fake/tsx",
      cwd: () => "/proj",
      stdout: () => undefined,
      stderr: () => undefined,
    });
    assert.equal(code, 0);
    assert.deepEqual(calls, [["run", "--help"]]);
  });

  it("forwards non-run commands to Go", async () => {
    const calls: { cmd: string; args: string[] }[] = [];
    const spawn: SpawnFn = ((cmd, args) => {
      calls.push({ cmd: String(cmd), args: args as string[] });
      return fakeChild(0);
    }) as SpawnFn;

    const code = await main(["version"], {
      spawn,
      resolveBinary: () => "/fake/skilleval",
      resolveTsxImport: () => "/fake/tsx",
      cwd: () => "/proj",
      stdout: () => undefined,
      stderr: () => undefined,
    });
    assert.equal(code, 0);
    assert.deepEqual(calls, [{ cmd: "/fake/skilleval", args: ["version"] }]);
  });

  it("rejects CLI flags for script evals", async () => {
    const err: string[] = [];
    const code = await main(["run", "eval.ts", "--model", "m"], {
      spawn: (() => fakeChild(0)) as SpawnFn,
      resolveBinary: () => "/fake/skilleval",
      resolveTsxImport: () => "/fake/tsx",
      cwd: () => "/proj",
      stdout: () => undefined,
      stderr: (l) => err.push(l),
    });
    assert.equal(code, 2);
    assert.match(err.join("\n"), /do not accept CLI flags/);
  });

  it("prints FAIL when an explicit script path is missing", async () => {
    const out: string[] = [];
    const err: string[] = [];
    const code = await main(["run", "/no/such/eval.js"], {
      spawn: (() => fakeChild(0)) as SpawnFn,
      resolveBinary: () => "/fake/skilleval",
      resolveTsxImport: () => "/fake/tsx",
      cwd: () => "/proj",
      stdout: (l) => out.push(l),
      stderr: (l) => err.push(l),
    });
    assert.equal(code, 1);
    assert.match(err.join("\n"), /not found/);
    assert.equal(out[0], "FAIL /no/such/eval.js");
  });

  it("runs an explicit .js eval without chdir and prints PASS", async () => {
    const root = await fs.mkdtemp(path.join(os.tmpdir(), "skilleval-cli-"));
    const nested = path.join(root, "suite");
    await fs.mkdir(nested);
    const file = path.join(nested, "eval.js");
    await fs.writeFile(file, "process.exit(0);\n");

    const calls: { cmd: string; args: string[]; cwd?: string; envDir?: string }[] = [];
    const out: string[] = [];
    const spawn: SpawnFn = ((cmd, args, opts) => {
      const env = (opts as { env?: NodeJS.ProcessEnv } | undefined)?.env;
      calls.push({
        cmd: String(cmd),
        args: args as string[],
        cwd: (opts as { cwd?: string } | undefined)?.cwd,
        envDir: env?.SKILLEVAL_EVAL_DIR,
      });
      return fakeChild(0);
    }) as SpawnFn;

    try {
      const code = await main(["run", file], {
        spawn,
        resolveBinary: () => "/fake/skilleval",
        resolveTsxImport: () => "/fake/tsx",
        cwd: () => root,
        stdout: (l) => out.push(l),
        stderr: () => undefined,
      });
      assert.equal(code, 0);
      assert.equal(calls.length, 1);
      assert.equal(calls[0]!.cwd, undefined);
      assert.equal(calls[0]!.envDir, nested);
      assert.deepEqual(calls[0]!.args, [file]);
      assert.equal(out[0], `PASS ${path.relative(root, file)}`);
    } finally {
      await fs.rm(root, { recursive: true, force: true });
    }
  });

  it("discovers eval scripts when no path is given", async () => {
    const root = await fs.mkdtemp(path.join(os.tmpdir(), "skilleval-cli-disc-"));
    const file = path.join(root, "foo.eval.js");
    await fs.writeFile(file, "process.exit(0);\n");

    const calls: string[][] = [];
    const spawn: SpawnFn = ((_cmd, args) => {
      calls.push(args as string[]);
      return fakeChild(0);
    }) as SpawnFn;

    try {
      const code = await main(["run"], {
        spawn,
        resolveBinary: () => "/fake/skilleval",
        resolveTsxImport: () => "/fake/tsx",
        cwd: () => root,
        stdout: () => undefined,
        stderr: () => undefined,
      });
      assert.equal(code, 0);
      assert.equal(calls.length, 1);
      assert.deepEqual(calls[0], [file]);
    } finally {
      await fs.rm(root, { recursive: true, force: true });
    }
  });

  it("exits 2 when discovery finds nothing", async () => {
    const root = await fs.mkdtemp(path.join(os.tmpdir(), "skilleval-cli-empty-"));
    const err: string[] = [];
    try {
      const code = await main(["run"], {
        spawn: (() => fakeChild(0)) as SpawnFn,
        resolveBinary: () => "/fake/skilleval",
        resolveTsxImport: () => "/fake/tsx",
        cwd: () => root,
        stdout: () => undefined,
        stderr: (l) => err.push(l),
      });
      assert.equal(code, 2);
      assert.match(err.join("\n"), /no TypeScript evals found/);
    } finally {
      await fs.rm(root, { recursive: true, force: true });
    }
  });

  it("prints FAIL and exits 1 when a script fails", async () => {
    const root = await fs.mkdtemp(path.join(os.tmpdir(), "skilleval-cli-fail-"));
    const file = path.join(root, "eval.js");
    await fs.writeFile(file, "process.exit(1);\n");
    const out: string[] = [];

    try {
      const code = await main(["run", file], {
        spawn: (() => fakeChild(1)) as SpawnFn,
        resolveBinary: () => "/fake/skilleval",
        resolveTsxImport: () => "/fake/tsx",
        cwd: () => root,
        stdout: (l) => out.push(l),
        stderr: () => undefined,
      });
      assert.equal(code, 1);
      assert.equal(out[0], "FAIL eval.js");
    } finally {
      await fs.rm(root, { recursive: true, force: true });
    }
  });
});
