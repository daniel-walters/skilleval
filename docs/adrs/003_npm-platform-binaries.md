# 003. npm ships platform binaries

## Status

Accepted

## Context

The TypeScript package is a CLI-backed client (ADR 002). Authors still had to install the Go `skilleval` binary separately (`PATH` / `SKILLEVAL_BIN`), which blocked the common desire to `npm install` once and run `skilleval` from `package.json` scripts inside a consumer repo.

## Decision

Ship the Go binary through **platform-specific npm packages** as `optionalDependencies` of `@danielwaltersdev/skilleval` (esbuild-style):

- One package per `os`/`cpu` (e.g. `@danielwaltersdev/skilleval-darwin-arm64`) containing only the native binary.
- The main package exposes `"bin": { "skilleval": "bin/skilleval.js" }` so npm links `node_modules/.bin/skilleval`.
- Resolution order: `SKILLEVAL_BIN` → packaged platform binary → `skilleval` on `PATH`.
- All packages share the same semver as the `v*` Git tag / CLI release.

## Consequences

- `npm install @danielwaltersdev/skilleval` is enough for scripts and `run()` on supported platforms.
- Publish must build or obtain six binaries and publish seven packages (six platform + main) per release.
- Each new scoped package name needs a one-time first publish before npm Trusted Publishing can attach.
- Unsupported platforms get a clear error (or PATH/`SKILLEVAL_BIN` fallback when present).

## Rejected alternatives

- **Fat multi-arch tarball in the main package** — huge downloads for every install.
- **Postinstall download from GitHub Releases** — fails offline/airgapped CI, weaker lockfile reproducibility, depends on GitHub at install time.
