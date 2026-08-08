# skilleval (TypeScript)

In-repo TypeScript client for [skilleval](https://github.com/daniel-walters/skilleval/blob/main/README.md#typescript). Runs evals through the Go `skilleval` CLI and asserts with typed `expect` matchers that mirror the shared YAML catalog.

YAML remains valid CLI authoring; this package is the typed alternative. Root authoring docs (install → `run` → `expect`) live in the [README TypeScript section](https://github.com/daniel-walters/skilleval/blob/main/README.md#typescript). Full worked example: [`examples/refactor-helper/eval.ts`](https://github.com/daniel-walters/skilleval/blob/main/examples/refactor-helper/eval.ts).

## Prerequisites

- Node.js 18+
- Node.js 18+
- Runner credentials (`CURSOR_API_KEY` / `ANTHROPIC_API_KEY`) and `MODEL` in the **process** environment — TypeScript `run()` does not load `.env` (the Go CLI does when invoked directly)

## Install

```bash
npm install @danielwaltersdev/skilleval
```

That pulls in a platform optionalDependency (`@danielwaltersdev/skilleval-<os>-<arch>`) with the Go binary and links `skilleval` on `node_modules/.bin`. `run()` and `package.json` scripts use it automatically:

```json
{
  "scripts": {
    "eval": "skilleval run ./eval.yaml --model \"$MODEL\""
  }
}
```

Resolution order: `SKILLEVAL_BIN` → packaged platform binary → `skilleval` on `PATH`.

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
  model: process.env.MODEL!,
});

expect(result).turns.toBeLessThanOrEqual(15);
expect(result).costUSD.toBeLessThanOrEqual(1);
expect(result).toolsUsed.toInclude("read", "edit").not.toInclude("web");
expect(result).skills.activated.toInclude("refactor-helper");
expect(result, workspace).file("src/foo.go").toHaveBeenModified().toContain(/func Foo/);
expect(result, workspace).file("src/new.go").toHaveBeenCreated().toContain("package demo");
expect(result, workspace).file("src/gone.go").toHaveBeenDeleted();
expect(result).finalMessage.toMatch(/Refactor/);
```

Example run (from the example directory):

```bash
cd examples/refactor-helper
MODEL=composer-2.5 npx tsx eval.ts
```

`run` returns `{ result, workspace, summary?, exitCode }`. A non-zero `exitCode` means the Go CLI failed YAML expects or a pass-rate gate after writing Result; Result is still returned so `expect()` can assert.

Programmatic multi-run uses the same YAML fields as the CLI. Pass `attempts` and optional `passRate: { min }` (0–1) on `run({ … })`; `loadEval` preserves them on the typed document (the on-disk YAML still drives the CLI when using `sourcePath`).

By default the Go CLI retains history under `.skilleval/history` and compares to the prior run when one exists. Baseline compare still runs on the Go side (history on disk / returned `summary`), but compare/`PASS` text is not forwarded to Node process stdout — use the CLI for visible diffs or inspect history/`summary`. Pass `noHistory` / `noBaseline` for ephemeral runs, or `history` / `baseline` path overrides (same semantics as the CLI flags). Pass `timeout` (Go duration string, e.g. `"30m"`) to forward `--timeout` to the CLI.

```ts
await run({
  name: "refactor-helper",
  prompt: "...",
  skill: "./skills/refactor-helper",
  model: process.env.MODEL!,
  noHistory: true,
  noBaseline: true,
});
```

Or load an existing eval YAML (keeps prompt/skill/input in YAML; assert in TypeScript):

```ts
const ev = await loadEval("./eval.yaml");
const { result, workspace } = await run(ev, { model: process.env.MODEL! });
```

## Matcher namespaces

| Namespace | Matchers |
| --- | --- |
| `turns` / `durationMs` / `toolCalls` / `costUSD` | `toBeLessThan`, `toBeLessThanOrEqual`, `toBeGreaterThan`, `toBeGreaterThanOrEqual`, `toBeEqual` |
| `usage.inputTokens` (and `outputTokens`, `cacheReadTokens`, `cacheWriteTokens`, `totalTokens`) | same numeric matchers |
| `toolsUsed` | `toInclude(...strings)`, `.not.toInclude(...strings)` |
| `skills.activated` | `toInclude(...strings)`, `.not.toInclude(...strings)` |
| `file(path)` | `toHaveStatus`, `toHaveBeenCreated` / `Modified` / `Deleted`, `toContain`, `toEqual` (pass `workspace` for content) |
| `finalMessage` | `toContain(string)`, `toMatch(RegExp)`, `toEqual(string \| RegExp)` |

Nil `costUSD` (unknown/unpriced model, or harness omit without a catalog hit) fails any cost bound — same as YAML expects.

Non-`finished` results fail as `run.status` before other checks. Matchers throw `ExpectError` with checker-style `path` + `reason`.

## Develop

```bash
npm test    # tsc + node --test
npm run build
```
