import { spawn } from "node:child_process";
import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

import { missingBinaryHint, resolveSkillevalBinary } from "./binary.js";
import {
  discoverEvalScripts,
  isScriptEvalPath,
} from "./discover.js";

const EXIT_OK = 0;
const EXIT_FAIL = 1;
const EXIT_USAGE = 2;

export type SpawnFn = typeof spawn;

export type CliDeps = {
  spawn: SpawnFn;
  resolveBinary: () => string;
  resolveTsxImport: () => string;
  cwd: () => string;
  stdout: (line: string) => void;
  stderr: (line: string) => void;
};

const defaultDeps: CliDeps = {
  spawn,
  resolveBinary: resolveSkillevalBinary,
  resolveTsxImport,
  cwd: () => process.cwd(),
  stdout: (line) => console.log(line),
  stderr: (line) => console.error(line),
};

/**
 * Resolve the tsx ESM loader path shipped with this package (not PATH).
 */
export function resolveTsxImport(): string {
  const require = createRequire(import.meta.url);
  try {
    return require.resolve("tsx/esm");
  } catch {
    try {
      return require.resolve("tsx");
    } catch (err) {
      const detail = err instanceof Error ? err.message : String(err);
      throw new Error(
        `tsx is required to run TypeScript evals but could not be resolved (${detail}); ` +
          `reinstall @danielwaltersdev/skilleval`,
      );
    }
  }
}

type RunParse =
  | { kind: "forward"; argv: string[] }
  | { kind: "scripts"; files: string[]; flags: string[] }
  | { kind: "usage"; message: string };

/**
 * Split `run` argv into flags vs a single optional positional path.
 * Flags may appear before or after the path (same as the Go CLI).
 */
export function parseRunArgv(args: string[]): {
  flags: string[];
  positional: string | undefined;
  error?: string;
} {
  const flags: string[] = [];
  const positionals: string[] = [];
  for (let i = 0; i < args.length; i++) {
    const a = args[i]!;
    if (a === "--") {
      positionals.push(...args.slice(i + 1));
      break;
    }
    if (a.startsWith("-")) {
      flags.push(a);
      // Value-taking flags: --model ID, --runner, --out, --timeout, --history, --baseline
      if (
        (a === "--model" ||
          a === "--runner" ||
          a === "--out" ||
          a === "--timeout" ||
          a === "--history" ||
          a === "--baseline") &&
        i + 1 < args.length &&
        !args[i + 1]!.startsWith("-")
      ) {
        flags.push(args[++i]!);
      }
      continue;
    }
    positionals.push(a);
  }
  if (positionals.length > 1) {
    return {
      flags,
      positional: undefined,
      error: `run: expected at most one path, got ${positionals.length}`,
    };
  }
  return { flags, positional: positionals[0] };
}

function planRun(args: string[], cwd: string): RunParse {
  const { flags, positional, error } = parseRunArgv(args);
  if (error) {
    return { kind: "usage", message: error };
  }
  if (!positional) {
    return { kind: "scripts", files: [], flags };
  }
  if (isScriptEvalPath(positional)) {
    return {
      kind: "scripts",
      files: [path.resolve(cwd, positional)],
      flags,
    };
  }
  // YAML / other: forward entire original argv to Go (including "run").
  return { kind: "forward", argv: ["run", ...args] };
}

function spawnInherit(
  deps: CliDeps,
  command: string,
  args: string[],
  opts: { cwd?: string; env?: NodeJS.ProcessEnv } = {},
): Promise<number> {
  return new Promise((resolve, reject) => {
    const child = deps.spawn(command, args, {
      stdio: "inherit",
      env: opts.env ?? process.env,
      cwd: opts.cwd,
      windowsHide: true,
    });
    child.on("error", reject);
    child.on("close", (code, signal) => {
      if (signal) {
        resolve(EXIT_FAIL);
        return;
      }
      resolve(code ?? EXIT_FAIL);
    });
  });
}

async function forwardToGo(deps: CliDeps, argv: string[]): Promise<number> {
  let bin: string;
  try {
    bin = deps.resolveBinary();
  } catch (err) {
    const detail = err instanceof Error ? err.message : String(err);
    deps.stderr(`skilleval: ${detail}`);
    return EXIT_FAIL;
  }
  try {
    return await spawnInherit(deps, bin, argv);
  } catch (err) {
    const detail = err instanceof Error ? err.message : String(err);
    deps.stderr(`skilleval: failed to spawn ${bin}: ${detail} (${missingBinaryHint(bin)})`);
    return EXIT_FAIL;
  }
}

function displayPath(file: string, cwd: string): string {
  const rel = path.relative(cwd, file);
  if (rel && !rel.startsWith("..") && !path.isAbsolute(rel)) {
    return rel;
  }
  return file;
}

/**
 * Build node argv to execute a script eval.
 * TS/MTS use the package-local tsx loader; JS/MJS use node directly.
 */
export function scriptNodeArgs(
  file: string,
  resolveTsx: () => string,
): { command: string; args: string[] } {
  const ext = path.extname(file).toLowerCase();
  if (ext === ".ts" || ext === ".mts") {
    return {
      command: process.execPath,
      args: ["--import", resolveTsx(), file],
    };
  }
  return { command: process.execPath, args: [file] };
}

async function runOneScript(
  deps: CliDeps,
  file: string,
  cwd: string,
): Promise<number> {
  if (!fs.existsSync(file)) {
    deps.stderr(`skilleval: eval file not found: ${file}`);
    return EXIT_FAIL;
  }
  const label = displayPath(file, cwd);
  let command: string;
  let args: string[];
  try {
    ({ command, args } = scriptNodeArgs(file, deps.resolveTsxImport));
  } catch (err) {
    const detail = err instanceof Error ? err.message : String(err);
    deps.stderr(`skilleval: ${detail}`);
    return EXIT_FAIL;
  }
  const fileDir = path.dirname(file);
  let code: number;
  try {
    code = await spawnInherit(deps, command, args, { cwd: fileDir });
  } catch (err) {
    const detail = err instanceof Error ? err.message : String(err);
    deps.stderr(`skilleval: failed to run ${label}: ${detail}`);
    deps.stdout(`FAIL ${label}`);
    return EXIT_FAIL;
  }
  if (code === EXIT_OK) {
    deps.stdout(`PASS ${label}`);
    return EXIT_OK;
  }
  deps.stdout(`FAIL ${label}`);
  return code === EXIT_USAGE ? EXIT_USAGE : EXIT_FAIL;
}

async function runScripts(
  deps: CliDeps,
  files: string[],
  flags: string[],
  cwd: string,
): Promise<number> {
  if (flags.length > 0) {
    deps.stderr(
      "skilleval: TypeScript/JavaScript evals do not accept CLI flags " +
        `(got ${flags.join(" ")}); set model in the script or via MODEL in the environment`,
    );
    return EXIT_USAGE;
  }

  let targets = files;
  if (targets.length === 0) {
    targets = await discoverEvalScripts(cwd);
    if (targets.length === 0) {
      deps.stderr(
        "skilleval: no TypeScript evals found under " +
          `${cwd} (looking for eval.{ts,mts,js,mjs} or *.eval.{ts,mts,js,mjs}). ` +
          "Pass a path: skilleval run ./eval.ts or skilleval run ./eval.yaml --model ID",
      );
      return EXIT_USAGE;
    }
  }

  let overall = EXIT_OK;
  for (const file of targets) {
    const code = await runOneScript(deps, file, cwd);
    if (code !== EXIT_OK) {
      overall = EXIT_FAIL;
    }
  }
  return overall;
}

/**
 * npm bin entry: route `run` for script evals / discovery; otherwise forward
 * to the Go skilleval binary.
 *
 * @returns process exit code
 */
export async function main(
  argv: string[],
  deps: CliDeps = defaultDeps,
): Promise<number> {
  if (argv.length === 0) {
    return forwardToGo(deps, argv);
  }

  const cmd = argv[0]!;
  if (cmd !== "run") {
    return forwardToGo(deps, argv);
  }

  const cwd = deps.cwd();
  const plan = planRun(argv.slice(1), cwd);
  switch (plan.kind) {
    case "usage":
      deps.stderr(`skilleval: ${plan.message}`);
      return EXIT_USAGE;
    case "forward":
      return forwardToGo(deps, plan.argv);
    case "scripts":
      return runScripts(deps, plan.files, plan.flags, cwd);
  }
}
