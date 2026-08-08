# 004. npm bin runs TypeScript evals

## Status

Accepted

## Context

Authors write evals with the TypeScript SDK (`run` + `expect`) but still had to invoke them with `tsx path/to/eval.ts`. That broke the “install `@danielwaltersdev/skilleval` → call `skilleval` from `package.json` scripts” story that ADR 003 established for YAML. Teaching the Go CLI to load `.ts` would couple the YAML harness to a Node/tsx runtime.

## Decision

The npm `skilleval` bin (`bin/skilleval.js` → `dist/cli.js`) owns TypeScript/JavaScript eval execution:

- `skilleval run path/to/eval.ts` (or `.mts` / `.js` / `.mjs`) executes the file via Node, using the package-local `tsx` loader for TypeScript, with `cwd` set to the file’s directory.
- `skilleval run` with no path discovers `eval.{ts,mts,js,mjs}` and `*.eval.{ts,mts,js,mjs}` under cwd (skips `node_modules`, hidden dirs, and common build outputs like `dist`) and runs them sequentially.
- YAML paths and other commands still forward to the Go binary unchanged.
- Script evals do not accept Go CLI flags; model stays in-script or `MODEL`.

Harness, agents, and checker policy remain Go-owned (ADR 002). This package is still not a general test framework (Jest/Vitest).

## Consequences

- TS-only projects can use `"eval": "skilleval run"` without listing `tsx` on the eval script.
- The main npm package depends on `tsx` so installs work without a separate runner dependency.
- Go stays YAML-only; discovery is name-based for script evals only (not YAML suite discovery).

## Rejected alternatives

- **Go CLI shells out to node/tsx** — couples the Go binary to a JS toolchain and duplicates routing already natural in the npm shim.
- **Require authors to keep `tsx eval.ts` in scripts** — fails the single-bin install story.
- **Broad `*eval*` globs or Vitest integration** — out of scope; would turn the package into a general test runner.
