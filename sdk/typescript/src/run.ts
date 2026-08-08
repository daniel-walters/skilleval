import { spawn } from "node:child_process";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";

import { stringify as stringifyYaml } from "yaml";

import type { EvalDocument, RunOptions, RunOverrides } from "./eval.js";
import type { Result, Summary } from "./result.js";

export interface RunResult {
  result: Result;
  workspace: string;
  summary?: Summary;
  /**
   * Exit code from the skilleval process. Non-zero when the CLI failed YAML
   * expects or a pass-rate gate after writing Result; Result is still returned
   * so `expect()` can assert (e.g. `run.status` on non-finished attempts).
   */
  exitCode: number;
}

type CliFlags = {
  model: string;
  runner?: string;
  out?: string;
  history?: string;
  noHistory?: boolean;
  baseline?: string;
  noBaseline?: boolean;
  timeout?: string;
};

/**
 * Run an eval via the skilleval Go CLI.
 *
 * - `run({ name, prompt, skill, model, ... })` writes a temp eval YAML (no expects).
 * - `run(await loadEval(path), { model })` uses the loaded YAML file as-is.
 * - Objects with `sourcePath` (from `loadEval`) always use that file — never a temp rewrite.
 */
export async function run(opts: RunOptions): Promise<RunResult>;
export async function run(ev: EvalDocument, opts: RunOverrides): Promise<RunResult>;
export async function run(
  evalOrOpts: RunOptions | EvalDocument,
  overrides?: RunOverrides,
): Promise<RunResult> {
  const { evalPath, flags, cleanup } = await resolveEvalAndFlags(evalOrOpts, overrides);
  try {
    return await invokeCli(evalPath, flags);
  } finally {
    if (cleanup) {
      await fs.rm(cleanup, { recursive: true, force: true }).catch(() => undefined);
    }
  }
}

async function resolveEvalAndFlags(
  evalOrOpts: RunOptions | EvalDocument,
  overrides?: RunOverrides,
): Promise<{ evalPath: string; flags: CliFlags; cleanup?: string }> {
  if (shouldWriteTempEval(evalOrOpts, overrides)) {
    const opts = evalOrOpts as RunOptions;
    if (!opts.model) {
      throw new Error("run: model is required");
    }
    const tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), "skilleval-ts-"));
    const evalPath = path.join(tmpDir, "eval.yaml");
    const doc: Record<string, unknown> = {
      schemaVersion: 1,
      name: opts.name,
      prompt: opts.prompt,
      skill: path.resolve(opts.skill),
    };
    if (opts.input) {
      doc.input = path.resolve(opts.input);
    }
    if (opts.mcp) {
      doc.mcp = path.resolve(opts.mcp);
    }
    if (opts.attempts !== undefined && opts.attempts > 0) {
      doc.attempts = opts.attempts;
    }
    if (opts.passRate !== undefined) {
      doc.passRate = opts.passRate;
    }
    await fs.writeFile(evalPath, stringifyYaml(doc), "utf8");
    return {
      evalPath,
      flags: {
        model: opts.model,
        runner: opts.runner,
        out: opts.out,
        history: opts.history,
        noHistory: opts.noHistory,
        baseline: opts.baseline,
        noBaseline: opts.noBaseline,
        timeout: opts.timeout,
      },
      cleanup: tmpDir,
    };
  }

  const ev = evalOrOpts as EvalDocument;
  if (!ev.sourcePath) {
    throw new Error("run: loaded eval is missing sourcePath; use loadEval(path)");
  }
  const model = overrides?.model ?? readStringField(evalOrOpts, "model");
  if (!model) {
    throw new Error("run: model is required");
  }
  return {
    evalPath: ev.sourcePath,
    flags: {
      model,
      runner: overrides?.runner ?? readStringField(evalOrOpts, "runner"),
      out: overrides?.out ?? readStringField(evalOrOpts, "out"),
      history: overrides?.history ?? readStringField(evalOrOpts, "history"),
      noHistory: overrides?.noHistory ?? readBoolField(evalOrOpts, "noHistory"),
      baseline: overrides?.baseline ?? readStringField(evalOrOpts, "baseline"),
      noBaseline: overrides?.noBaseline ?? readBoolField(evalOrOpts, "noBaseline"),
      timeout: overrides?.timeout ?? readStringField(evalOrOpts, "timeout"),
    },
  };
}

/**
 * True when run should materialize a temp eval YAML.
 * Loaded evals (`sourcePath` set) always use the on-disk file, even if a
 * `model` field is present on the same object (e.g. `{ ...ev, model }`).
 */
export function shouldWriteTempEval(
  evalOrOpts: RunOptions | EvalDocument,
  overrides?: RunOverrides,
): boolean {
  if (hasSourcePath(evalOrOpts)) {
    return false;
  }
  return overrides === undefined && "model" in evalOrOpts && typeof evalOrOpts.model === "string";
}

function hasSourcePath(evalOrOpts: RunOptions | EvalDocument): boolean {
  return (
    typeof (evalOrOpts as EvalDocument).sourcePath === "string" &&
    !!(evalOrOpts as EvalDocument).sourcePath
  );
}

function readStringField(obj: object, key: string): string | undefined {
  if (!(key in obj)) {
    return undefined;
  }
  const v = (obj as Record<string, unknown>)[key];
  return typeof v === "string" && v ? v : undefined;
}

function readBoolField(obj: object, key: string): boolean | undefined {
  if (!(key in obj)) {
    return undefined;
  }
  const v = (obj as Record<string, unknown>)[key];
  return typeof v === "boolean" ? v : undefined;
}

/**
 * Append history/baseline CLI flags. Omits flags when unset so the Go CLI
 * defaults (retain under .skilleval/history, auto-compare latest) apply.
 */
export function appendReportFlags(
  args: string[],
  flags: Pick<CliFlags, "history" | "noHistory" | "baseline" | "noBaseline">,
): void {
  if (flags.noHistory) {
    args.push("--no-history");
  } else if (flags.history) {
    args.push("--history", path.resolve(flags.history));
  }
  if (flags.noBaseline) {
    args.push("--no-baseline");
  } else if (flags.baseline) {
    args.push("--baseline", path.resolve(flags.baseline));
  }
}

async function invokeCli(evalPath: string, flags: CliFlags): Promise<RunResult> {
  const bin = resolveBinary();
  let outPath: string;
  let outCleanup: string | undefined;
  if (flags.out) {
    outPath = path.resolve(flags.out);
  } else {
    outCleanup = await fs.mkdtemp(path.join(os.tmpdir(), "skilleval-out-"));
    outPath = path.join(outCleanup, "result.json");
  }

  const args = ["run", evalPath, "--model", flags.model, "--out", outPath];
  if (flags.runner) {
    args.push("--runner", flags.runner);
  }
  if (flags.timeout) {
    args.push("--timeout", flags.timeout);
  }
  appendReportFlags(args, flags);

  try {
    const { stdout, stderr, code } = await spawnCapture(bin, args);
    const exitCode = code ?? 1;

    const writes = parseWroteLines(stdout);
    if (writes.length === 0) {
      const detail = stderr.trim() || stdout.trim() || `exit ${exitCode}`;
      throw new Error(`run: skilleval produced no Result (${detail})`);
    }

    const last = writes[writes.length - 1]!;
    const result = await readJson<Result>(last.outPath);
    let summary: Summary | undefined;
    try {
      summary = await readJson<Summary>(summaryOutPath(outPath));
    } catch {
      // optional
    }

    // Non-zero exit is expected when YAML expects / pass-rate fail after a
    // Result was written; still return so TS expect() can assert.
    return { result, workspace: last.workspace, summary, exitCode };
  } finally {
    if (outCleanup) {
      await fs.rm(outCleanup, { recursive: true, force: true }).catch(() => undefined);
    }
  }
}

function resolveBinary(): string {
  const fromEnv = process.env.SKILLEVAL_BIN?.trim();
  if (fromEnv) {
    return fromEnv;
  }
  return "skilleval";
}

interface WroteLine {
  outPath: string;
  workspace: string;
}

/** Parse CLI stdout lines like: wrote <path> (status=finished workspace=/tmp/...) */
export function parseWroteLines(stdout: string): WroteLine[] {
  const re =
    /(?:attempt \d+\/\d+: )?wrote (.+?) \(status=\w+ workspace=(.+?)\)/g;
  const out: WroteLine[] = [];
  let m: RegExpExecArray | null;
  while ((m = re.exec(stdout)) !== null) {
    out.push({ outPath: m[1]!, workspace: m[2]! });
  }
  return out;
}

export function summaryOutPath(outPath: string): string {
  const ext = path.extname(outPath);
  const stem = ext ? outPath.slice(0, -ext.length) : outPath;
  return `${stem}-summary${ext}`;
}

function spawnCapture(
  bin: string,
  args: string[],
): Promise<{ stdout: string; stderr: string; code: number | null }> {
  return new Promise((resolve, reject) => {
    const child = spawn(bin, args, {
      stdio: ["ignore", "pipe", "pipe"],
      env: process.env,
    });
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk: Buffer) => {
      stdout += chunk.toString("utf8");
    });
    child.stderr.on("data", (chunk: Buffer) => {
      stderr += chunk.toString("utf8");
    });
    child.on("error", (err) => {
      reject(
        new Error(
          `run: failed to spawn ${bin}: ${err.message} (set SKILLEVAL_BIN or install skilleval on PATH)`,
        ),
      );
    });
    child.on("close", (code) => {
      resolve({ stdout, stderr, code });
    });
  });
}

async function readJson<T>(filePath: string): Promise<T> {
  const raw = await fs.readFile(filePath, "utf8");
  return JSON.parse(raw) as T;
}
