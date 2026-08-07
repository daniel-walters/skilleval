# skilleval (TypeScript)

In-repo TypeScript client for [skilleval](../../README.md). Runs evals through the Go `skilleval` CLI and asserts with typed `expect` matchers that mirror the shared YAML catalog.

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
  prompt: "…",
  skill: "./skills/refactor-helper",
  input: "./fixtures/refactor-helper",
  model: process.env.MODEL!,
});

expect(result, workspace).turns.toBeLessThanOrEqual(15);
expect(result, workspace).file("src/foo.go").toHaveBeenModified().toContain(/func Foo/);
```

Or load an existing eval YAML:

```ts
const ev = await loadEval("./eval.yaml");
const { result, workspace } = await run(ev, { model: process.env.MODEL! });
```

## Develop

```bash
npm test    # tsc + node --test
npm run build
```

Authoring narrative and root README examples live in a follow-up docs ticket; this package README covers install and the public API surface only.
