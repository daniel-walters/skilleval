# skilleval (TypeScript)

In-repo TypeScript client for [skilleval](https://github.com/daniel-walters/skilleval/blob/main/README.md#typescript). Runs evals through the Go `skilleval` CLI and asserts with typed `expect` matchers that mirror the shared YAML catalog.

YAML remains valid CLI authoring; this package is the typed alternative. Root authoring docs (install → `run` → `expect`) live in the [README TypeScript section](https://github.com/daniel-walters/skilleval/blob/main/README.md#typescript). First eval, models, and tool names: [`docs/authoring.md`](https://github.com/daniel-walters/skilleval/blob/main/docs/authoring.md). Full worked example: [`examples/refactor-helper/eval.ts`](https://github.com/daniel-walters/skilleval/blob/main/examples/refactor-helper/eval.ts).

## Prerequisites

- Node.js 18+
- Runner credentials (`CURSOR_API_KEY` / `ANTHROPIC_API_KEY`) in the process environment or a cwd `.env` — `run()` loads `.env` the same as the Go CLI (process env wins). Put `model` on the `run({ … })` call (or pass `--model` to the YAML CLI).

## Install

From a new folder:

```bash
npm init -y
npm install @danielwaltersdev/skilleval
```

That pulls in a platform optionalDependency (`@danielwaltersdev/skilleval-<os>-<arch>`) with the Go binary and links `skilleval` on `node_modules/.bin`. `tsx` ships with the package. `run()` and `package.json` scripts use the bin automatically. Mint `CURSOR_API_KEY` at [Dashboard → API Keys](https://cursor.com/dashboard/api); Claude uses `ANTHROPIC_API_KEY`. Model ids are runner-specific — see [docs/authoring.md](https://github.com/daniel-walters/skilleval/blob/main/docs/authoring.md#credentials-and-models).

YAML:

```json
{
  "scripts": {
    "eval": "skilleval run ./eval.yaml --model composer-2.5"
  }
}
```

TypeScript-only (no `tsx` on the eval file) — discovers `eval.{ts,mts,js,mjs}` and `*.eval.{ts,mts,js,mjs}` under cwd (skips `node_modules`, hidden dirs, and `dist`-like build outputs), or pass `./eval.ts`:

```json
{
  "scripts": {
    "eval": "skilleval run"
  }
}
```

Put `model` on `run({ … })`. Script evals do not take Go CLI flags (`--model`, etc.). Relative `skill` / `input` paths resolve from the eval file’s directory (the process cwd when the script runs), so keep the script next to those folders or path relative to it. Resolution order for the Go binary: `SKILLEVAL_BIN` → packaged platform binary → `skilleval` on `PATH`.

From a clone of this repo (development):

```bash
cd sdk/typescript
npm install
npm run build
```

For local `run()` / bin without a published platform package, build the Go CLI and put it on `PATH`, or set `SKILLEVAL_BIN`.

## Usage

```ts
import { run, expect, loadEval } from "@danielwaltersdev/skilleval";

const { result, workspace } = await run({
  name: "refactor-helper",
  prompt: `Use the refactor-helper skill on this package:

1. Refactor src/foo.go for clarity (simplify Foo; keep package demo and the Foo name).
2. Extract a small helper into a new file src/new.go (e.g. a greetPrefix helper used by Foo).
3. Delete src/gone.go — it is obsolete legacy code.

Do all three: modify foo.go, create new.go, delete gone.go.`,
  skill: "./skills/refactor-helper",
  input: "./fixtures/refactor-helper",
  model: "composer-2.5",
});

expect(result).turns.toBeLessThanOrEqual(15);
expect(result).costUSD.toBeLessThanOrEqual(1);
expect(result).toolsUsed.toInclude("read", "edit").not.toInclude("web");
expect(result).toolCalls.named("edit").toBeGreaterThanOrEqual(1);
expect(result).toolCalls.toIncludeInOrder([
  { name: ["write", "edit"], args: { path: "src/foo.go" } },
]);
expect(result).skills.activated.toInclude("refactor-helper");
expect(result, workspace).file("src/foo.go").toHaveBeenModified().toContain(/func Foo/).not.toContain("TODO");
expect(result, workspace).file("src/new.go").toHaveBeenCreated().toContain("package demo");
expect(result, workspace).file("src/gone.go").toHaveBeenDeleted();
expect(result).finalMessage.toMatch(/Refactor/);
```

Example run (from the example directory, with this package installed / linked):

```bash
cd examples/refactor-helper
skilleval run ./eval.ts
```

`run` returns `{ result, workspace, attempts, passRate?, summary?, exitCode, expect }`. `result` / `workspace` are the last successful write (single-attempt evals keep working). A non-zero `exitCode` means the Go CLI failed YAML expects or a YAML pass-rate gate after writing Result; Result is still returned so `expect()` can assert. That CLI `exitCode` is not a shell tool’s process status — those live on `result.metrics.toolCalls[].exitCode` when observed.

Programmatic multi-run: pass `attempts` on `run({ … })` so the CLI loops. Pass `passRate: { min }` (0–1) for the **TypeScript** gate — it is not copied into the temp YAML (no expects there). Then score every attempt with `batch.expect`:

```ts
const batch = await run({
  name: "refactor-helper",
  prompt: "...",
  skill: "./skills/refactor-helper",
  model: "composer-2.5",
  attempts: 10,
  passRate: { min: 0.8 },
});

batch.expect(({ result, workspace }) => {
  expect(result, workspace).turns.toBeLessThanOrEqual(15);
});
```

`batch.expect` isolates matcher failures per attempt and fails the process only when the TS pass rate is below `min` (default **1** when `passRate` is omitted). Runner-error slots count as failed and skip the callback. `loadEval` preserves `attempts` / `passRate` on the typed document; the on-disk YAML still drives the CLI checker when using `sourcePath` (YAML expects stay CLI-owned). Optional `replies: string[]` scripts mid-run user messages after the initial prompt (interactive skills; see root README).

By default the Go CLI retains history under `.skilleval/history` and compares to the prior run when one exists. `run()` tees the Go CLI stdout and stderr to the Node process (PASS/FAIL, summary, `vs baseline:` diffs, and `agent running…` progress) while still capturing them for Result parsing and error messages. Colors follow TTY / `FORCE_COLOR` / `NO_COLOR` (same as the Go CLI). Pass `noHistory` / `noBaseline` for ephemeral runs, or `history` / `baseline` path overrides (same semantics as the CLI flags). Pass `timeout` (Go duration string, e.g. `"30m"`) to forward `--timeout` to the CLI.

```ts
await run({
  name: "refactor-helper",
  prompt: "...",
  skill: "./skills/refactor-helper",
  model: "composer-2.5",
  noHistory: true,
  noBaseline: true,
});
```

Or load an existing eval YAML (keeps prompt/skill/input in YAML; assert in TypeScript):

```ts
const ev = await loadEval("./eval.yaml");
const { result, workspace } = await run(ev, { model: "composer-2.5" });
```

## Matcher namespaces

| Namespace | Matchers |
| --- | --- |
| `turns` / `durationMs` / `costUSD` | `toBeLessThan`, `toBeLessThanOrEqual`, `toBeGreaterThan`, `toBeGreaterThanOrEqual`, `toBeEqual` |
| `toolCalls` | same numeric matchers (total count); `named(tool)` → numeric count for that name; `toIncludeInOrder([{ name, args?, exitCode? }, …])` ordered subsequence (gaps allowed; `name` is one string or a nonempty list; string arg = equals, `RegExp` = match; `exitCode` is one integer or a nonempty list, only for `shell` / `Bash`); `.not.toIncludeInOrder` forbids each step independently (YAML `orderExcludes`) |
| `usage.inputTokens` (and `outputTokens`, `cacheReadTokens`, `cacheWriteTokens`, `totalTokens`) | same numeric matchers |
| `toolsUsed` | `toInclude(...strings)`, `.not.toInclude(...strings)` |
| `skills.activated` | `toInclude(...strings)`, `.not.toInclude(...strings)` |
| `file(path)` | `toHaveStatus`, `toHaveBeenCreated` / `Modified` / `Deleted`, `toContain`, `toEqual`, `.not.toContain(...patterns)` (pass `workspace` for content) |
| `finalMessage` | `toContain(string)`, `toMatch(RegExp)`, `toEqual(string \| RegExp)` |

Nil `costUSD` (unknown/unpriced model, or harness omit without a catalog hit) fails any cost bound — same as YAML expects.

Expect failures are **collected**, not short-circuited: every failing matcher in an eval script is recorded, then the full list is printed on process exit (non-zero). Call `expect.report()` to throw an `ExpectError` with all pending failures immediately. Non-`finished` results still hard-fail as `run.status` before other checks. `ExpectError` exposes checker-style `path` + `reason` (first failure) and `failures` (the full list).

## Develop

```bash
npm test    # tsc + node --test
npm run build
```
