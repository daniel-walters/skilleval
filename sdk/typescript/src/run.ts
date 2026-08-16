import { spawn } from "node:child_process";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";

import { stringify as stringifyYaml } from "yaml";

import { missingBinaryHint, resolveSkillevalBinary } from "./binary.js";
import { loadDotEnv } from "./envfile.js";
import { resolveEvalRel } from "./evalpath.js";
import type { EvalDocument, PassRateExpect, RunOptions, RunOverrides } from "./evalTypes.js";
import {
  expectAttempts,
  type AttemptExpectFn,
  type BatchExpectReport,
} from "./expect.js";
import type { AttemptOutcome, Result, Summary } from "./result.js";

export type { AttemptOutcome } from "./result.js";

export interface RunResult {
  result: Result;
  workspace: string;
  /** One slot per scheduled attempt (runner errors have `error`, no Result). */
  attempts: AttemptOutcome[];
  /** TS `batch.expect` gate; not the CLI YAML passRate for programmatic runs. */
  passRate?: PassRateExpect;
  summary?: Summary;
  /**
   * Exit code from the skilleval process. Non-zero when the CLI failed YAML
   * expects or a pass-rate gate after writing Result; Result is still returned
   * so `expect()` can assert (e.g. `run.status` on non-finished attempts).
   */
  exitCode: number;
  /**
   * Score TS expects on every attempt, then gate on `passRate.min` (default 1).
   * Isolated per attempt so allowed flakes do not fail the process.
   */
  expect(fn: AttemptExpectFn): BatchExpectReport;
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
 *
 * Loads `.env` from process cwd before spawning (missing file is fine;
 * process env wins), matching the Go CLI. Relative skill/input/mcp paths
 * resolve from the eval file directory, not from cwd.
 */
export async function run(opts: RunOptions): Promise<RunResult>;
export async function run(ev: EvalDocument, opts: RunOverrides): Promise<RunResult>;
export async function run(
  evalOrOpts: RunOptions | EvalDocument,
  overrides?: RunOverrides,
): Promise<RunResult> {
  loadDotEnv();
  const { evalPath, flags, cleanup, passRate } = await resolveEvalAndFlags(
    evalOrOpts,
    overrides,
  );
  try {
    return await invokeCli(evalPath, flags, passRate);
  } finally {
    if (cleanup) {
      await fs.rm(cleanup, { recursive: true, force: true }).catch(() => undefined);
    }
  }
}

/** Resolve eval path + CLI flags (exported for unit tests). */
export async function resolveEvalAndFlags(
  evalOrOpts: RunOptions | EvalDocument,
  overrides?: RunOverrides,
): Promise<{
  evalPath: string;
  flags: CliFlags;
  cleanup?: string;
  passRate?: PassRateExpect;
}> {
  if (shouldWriteTempEval(evalOrOpts, overrides)) {
    const opts = evalOrOpts as RunOptions;
    if (typeof opts.model !== "string" || !opts.model) {
      throw new Error("run: model is required");
    }
    const tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), "skilleval-ts-"));
    const evalPath = path.join(tmpDir, "eval.yaml");
    const doc: Record<string, unknown> = {
      schemaVersion: 1,
      name: opts.name,
      prompt: opts.prompt,
      skill: resolveEvalRel(opts.skill),
    };
    if (opts.replies && opts.replies.length > 0) {
      doc.replies = opts.replies;
    }
    if (opts.input) {
      doc.input = resolveEvalRel(opts.input);
    }
    if (opts.mcp) {
      doc.mcp = resolveEvalRel(opts.mcp);
    }
    if (opts.attempts !== undefined && opts.attempts > 0) {
      doc.attempts = opts.attempts;
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
      passRate: opts.passRate,
    };
  }

  const ev = evalOrOpts as EvalDocument;
  if (!ev.sourcePath) {
    throw new Error(
      "run: expected a loaded eval from loadEval(path), or pass model on run({ … })",
    );
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
    passRate: ev.passRate,
  };
}

/**
 * True when run should materialize a temp eval YAML.
 * Loaded evals (`sourcePath` set) always use the on-disk file, even if a
 * `model` field is present on the same object (e.g. `{ ...ev, model }`).
 * Missing or non-string `model` still takes this path so callers get
 * "model is required" instead of a misleading sourcePath error.
 */
export function shouldWriteTempEval(
  evalOrOpts: RunOptions | EvalDocument,
  overrides?: RunOverrides,
): boolean {
  if (hasSourcePath(evalOrOpts)) {
    return false;
  }
  return overrides === undefined;
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

async function invokeCli(
  evalPath: string,
  flags: CliFlags,
  passRate?: PassRateExpect,
): Promise<RunResult> {
  const bin = resolveSkillevalBinary();
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

    const slots = parseAttemptSlots(stdout);
    const writes = slots.filter((s) => s.outPath);
    if (writes.length === 0) {
      const detail = stderr.trim() || stdout.trim() || `exit ${exitCode}`;
      throw new Error(`run: skilleval produced no Result (${detail})`);
    }

    const attempts: AttemptOutcome[] = [];
    for (const slot of slots) {
      if (slot.outPath && slot.workspace) {
        const result = await readJson<Result>(slot.outPath);
        attempts.push({ result, workspace: slot.workspace });
      } else {
        attempts.push({ error: slot.error ?? "no Result" });
      }
    }

    const lastWrite = lastSuccessfulAttempt(attempts);
    if (!lastWrite) {
      const detail = stderr.trim() || stdout.trim() || `exit ${exitCode}`;
      throw new Error(`run: skilleval produced no Result (${detail})`);
    }

    let summary: Summary | undefined;
    try {
      summary = await readJson<Summary>(summaryOutPath(outPath));
    } catch {
      // optional
    }

    // Non-zero exit is expected when YAML expects / pass-rate fail after a
    // Result was written; still return so TS expect() can assert.
    return createRunResult({
      result: lastWrite.result,
      workspace: lastWrite.workspace,
      attempts,
      passRate,
      summary,
      exitCode,
    });
  } finally {
    if (outCleanup) {
      await fs.rm(outCleanup, { recursive: true, force: true }).catch(() => undefined);
    }
  }
}

/** Build a RunResult with `expect` bound (exported for unit tests). */
export function createRunResult(
  base: Omit<RunResult, "expect">,
): RunResult {
  const runResult: RunResult = {
    ...base,
    expect(fn) {
      return expectAttempts(runResult.attempts, fn, runResult.passRate);
    },
  };
  return runResult;
}

function lastSuccessfulAttempt(
  attempts: AttemptOutcome[],
): { result: Result; workspace: string } | undefined {
  for (let i = attempts.length - 1; i >= 0; i--) {
    const a = attempts[i]!;
    if (a.result !== undefined && a.workspace !== undefined) {
      return { result: a.result, workspace: a.workspace };
    }
  }
  return undefined;
}

interface WroteLine {
  outPath: string;
  workspace: string;
}

/** One parsed CLI attempt: a wrote Result, a runner error, or a gap. */
export interface AttemptSlot {
  attempt: number;
  total: number;
  outPath?: string;
  workspace?: string;
  error?: string;
}

/**
 * Parse CLI stdout into N ordered attempt slots (wrote lines + `attempt i/n: error:`).
 */
export function parseAttemptSlots(stdout: string): AttemptSlot[] {
  const byIndex = new Map<number, AttemptSlot>();
  let total = 0;

  const wroteRe =
    /(?:attempt (\d+)\/(\d+): )?wrote (.+?) \(status=\w+ workspace=(\S+)(?: agentLog=\S+)?\)/g;
  const unprefixed: { outPath: string; workspace: string }[] = [];
  let m: RegExpExecArray | null;
  while ((m = wroteRe.exec(stdout)) !== null) {
    if (m[1] && m[2]) {
      const attempt = Number(m[1]);
      const n = Number(m[2]);
      total = Math.max(total, n);
      byIndex.set(attempt, {
        attempt,
        total: n,
        outPath: m[3],
        workspace: m[4],
      });
    } else {
      unprefixed.push({ outPath: m[3]!, workspace: m[4]! });
    }
  }

  const errRe = /attempt (\d+)\/(\d+): error: (.*)$/gm;
  while ((m = errRe.exec(stdout)) !== null) {
    const attempt = Number(m[1]);
    const n = Number(m[2]);
    total = Math.max(total, n);
    byIndex.set(attempt, { attempt, total: n, error: m[3]!.trimEnd() });
  }

  if (byIndex.size === 0 && unprefixed.length > 0) {
    total = unprefixed.length;
    unprefixed.forEach((w, i) => {
      byIndex.set(i + 1, {
        attempt: i + 1,
        total,
        outPath: w.outPath,
        workspace: w.workspace,
      });
    });
  }

  if (total === 0) {
    return [];
  }

  const slots: AttemptSlot[] = [];
  for (let i = 1; i <= total; i++) {
    const existing = byIndex.get(i);
    if (existing) {
      slots.push({ ...existing, total });
    } else {
      slots.push({ attempt: i, total, error: "no Result" });
    }
  }
  return slots;
}

/** Parse CLI stdout lines like: wrote <path> (status=finished workspace=/tmp/... [agentLog=...]) */
export function parseWroteLines(stdout: string): WroteLine[] {
  return parseAttemptSlots(stdout)
    .filter((s): s is AttemptSlot & { outPath: string; workspace: string } =>
      Boolean(s.outPath && s.workspace),
    )
    .map((s) => ({ outPath: s.outPath, workspace: s.workspace }));
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
    const env = { ...process.env };
    // Child stdout is a pipe (not a TTY); force color when the parent terminal
    // would show it so PASS/FAIL and baseline deltas stay colored when teed.
    if (process.stdout.isTTY && !process.env.NO_COLOR) {
      env.FORCE_COLOR = "1";
    }
    const child = spawn(bin, args, {
      // Pipe both streams so we can tee live output and still capture stderr
      // for "no Result" error messages (inherit would leave stderr empty).
      stdio: ["ignore", "pipe", "pipe"],
      env,
    });
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk: Buffer) => {
      stdout += chunk.toString("utf8");
      process.stdout.write(chunk);
    });
    child.stderr.on("data", (chunk: Buffer) => {
      stderr += chunk.toString("utf8");
      process.stderr.write(chunk);
    });
    child.on("error", (err) => {
      reject(
        new Error(`run: failed to spawn ${bin}: ${err.message} (${missingBinaryHint(bin)})`),
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
