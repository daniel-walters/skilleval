# 009. TypeScript evals keep invocation cwd

## Status

Accepted

## Context

[ADR 004](004_npm-bin-typescript-run.md) ran script evals with `cwd` set to the eval file’s directory so relative `skill` / `input` / `mcp` paths resolved. That also moved `.env` loading and history away from the directory the user invoked `skilleval` in. YAML never chdirs: it joins those paths to the eval file and reads `.env` from process cwd. `skilleval run ./nested/eval.ts` from a repo root then failed with `CURSOR_API_KEY is required` even when root `.env` was present.

## Decision

Do not chdir when executing TypeScript/JavaScript evals. Resolve relative `skill` / `input` / `mcp` from the eval file’s directory. Load `.env` and write history from the invocation cwd, same as YAML.

## Consequences

- A repo-root `.env` works for nested `eval.ts` files.
- History for `skilleval run ./nested/eval.ts` lands next to the invocation, not the eval file (YAML parity). Authors who `cd` into the eval package keep history there.
- `run({ skill: "./…" })` no longer depends on process cwd matching the eval file.

## Rejected alternatives

- **Keep chdir; load `.env` from invocation cwd only** — fixes credentials but leaves history and other cwd-relative behavior split from YAML.
- **Look in both directories for `.env`** — unclear precedence when both exist; process-env-wins already covers an explicit override.
