# skilleval (TypeScript)

In-repo TypeScript client for [skilleval](../../README.md#typescript). Runs evals through the Go `skilleval` CLI and asserts with typed `expect` matchers that mirror the shared YAML catalog.

YAML remains valid CLI authoring; this package is the typed alternative. Root authoring docs (install → `run` → `expect`) live in the [README TypeScript section](../../README.md#typescript). Full worked example: [`examples/refactor-helper/eval.ts`](../../examples/refactor-helper/eval.ts).

## Prerequisites

- Node.js 18+
- `skilleval` binary on `PATH`, or set `SKILLEVAL_BIN` to the binary path
- Runner credentials unchanged from the Go CLI (`CURSOR_API_KEY` / `ANTHROPIC_API_KEY`)

## Install (local)

```bash
cd sdk/typescript
npm install
npm run build
```

From another package in the monorepo:

```bash
npm install ../path/to/skilleval/sdk/typescript
```

## Usage

```ts
import { run, expect, loadEval } from "skilleval";

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

`run` returns `{ result, workspace, summary?, exitCode }`. A non-zero `exitCode` means the Go CLI failed YAML expects or a pass-rate gate after writing Result; Result is still returned so `expect()` can assert.

By default the Go CLI retains history under `.skilleval/history` and compares to the prior run when one exists. Pass `noHistory` / `noBaseline` for ephemeral runs, or `history` / `baseline` path overrides (same semantics as the CLI flags):

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
| `turns` / `costUSD` | `toBeLessThan`, `toBeLessThanOrEqual`, `toBeGreaterThan`, `toBeGreaterThanOrEqual`, `toBeEqual` |
| `toolsUsed` | `toInclude(...strings)`, `.not.toInclude(...strings)` |
| `skills.activated` | `toInclude(...strings)` |
| `file(path)` | `toHaveStatus`, `toHaveBeenCreated` / `Modified` / `Deleted`, `toContain`, `toEqual` (pass `workspace` for content) |
| `finalMessage` | `toContain(string)`, `toMatch(RegExp)`, `toEqual(string \| RegExp)` |

Non-`finished` results fail as `run.status` before other checks. Matchers throw `ExpectError` with checker-style `path` + `reason`.

## Develop

```bash
npm test    # tsc + node --test
npm run build
```
