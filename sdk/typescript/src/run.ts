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
}

type CliFlags = {
  model: string;
  runner?: string;
  out?: string;
  history?: string;
  baseline?: string;
};

/**
 * Run an eval via the skilleval Go CLI.
 *
 * - `run({ name, prompt, skill, model, ... })` writes a temp eval YAML (no expects).
 * - `run(await loadEval(path), { model })` uses the loaded YAML file as-is.
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
  if (isRunOptions(evalOrOpts, overrides)) {
    const opts = evalOrOpts;
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
    await fs.writeFile(evalPath, stringifyYaml(doc), "utf8");
    return {
      evalPath,
      flags: {
        model: opts.model,
        runner: opts.runner,
        out: opts.out,
        history: opts.history,
        baseline: opts.baseline,
      },
      // Keep temp dir until after CLI runs (eval path must exist during spawn).
      // invokeCli reads results into memory; then we delete.
      cleanup: tmpDir,
    };
  }

  const ev = evalOrOpts;
  if (!overrides?.model) {
    throw new Error("run: model is required");
  }
  if (!ev.sourcePath) {
    throw new Error("run: loaded eval is missing sourcePath; use loadEval(path)");
  }
  return {
    evalPath: ev.sourcePath,
    flags: {
      model: overrides.model,
      runner: overrides.runner,
      out: overrides.out,
      history: overrides.history,
      baseline: overrides.baseline,
    },
  };
}

function isRunOptions(
  evalOrOpts: RunOptions | EvalDocument,
  overrides?: RunOverrides,
): evalOrOpts is RunOptions {
  return overrides === undefined && "model" in evalOrOpts && typeof evalOrOpts.model === "string";
}

async function invokeCli(evalPath: string, flags: CliFlags): Promise<RunResult> {
  const bin = resolveBinary();
  let outPath: string;
  if (flags.out) {
    outPath = path.resolve(flags.out);
  } else {
    const outDir = await fs.mkdtemp(path.join(os.tmpdir(), "skilleval-out-"));
    outPath = path.join(outDir, "result.json");
  }

  const args = ["run", evalPath, "--model", flags.model, "--out", outPath];
  if (flags.runner) {
    args.push("--runner", flags.runner);
  }
  if (flags.history) {
    args.push("--history", path.resolve(flags.history));
  }
  if (flags.baseline) {
    args.push("--baseline", path.resolve(flags.baseline));
  }

  const { stdout, stderr, code } = await spawnCapture(bin, args);

  const writes = parseWroteLines(stdout);
  if (writes.length === 0) {
    const detail = stderr.trim() || stdout.trim() || `exit ${code}`;
    throw new Error(`run: skilleval produced no Result (${detail})`);
  }

  const last = writes[writes.length - 1]!;
  const result = await readJson<Result>(last.outPath);
  // Summary is written beside --out as <stem>-summary<ext>.
  let summary: Summary | undefined;
  try {
    summary = await readJson<Summary>(summaryOutPath(outPath));
  } catch {
    // optional
  }

  return { result, workspace: last.workspace, summary };
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
