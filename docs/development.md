# Development

Maintainer notes for working on skilleval itself. Author-facing usage lives in the [root README](../README.md). Releases: [releasing.md](releasing.md). Architecture: [adrs/003_npm-platform-binaries.md](adrs/003_npm-platform-binaries.md).

## Build from source

```bash
go build -o skilleval ./cmd/skilleval
```

`skilleval version` prints `dev` for local builds (release builds print the tag version via GoReleaser ldflags).

Module path targets Go 1.26+.

## CI

GitHub Actions is the source of truth for merge readiness. On pull requests and pushes to `main`, CI runs `gofmt`, `go vet`, `go test`, `golangci-lint`, and the TypeScript SDK tests (`sdk/typescript`). Local pre-commit is recommended for a faster feedback loop; a green CI check is what counts for merge.

## Harness agent SDK pins

Each runner embeds a small Node helper with an exact npm pin and lockfile (`npm ci` at prepare time). Those files are the `go:embed` inputs:

| Runner | Package | Pin + lockfile |
| --- | --- | --- |
| Cursor | `@cursor/sdk` | [`runner/cursoragent/`](../runner/cursoragent/) |
| Claude | `@anthropic-ai/claude-agent-sdk` | [`runner/claudeagent/`](../runner/claudeagent/) |

[Dependabot](../.github/dependabot.yml) checks those directories weekly and opens PRs that update `package.json` + `package-lock.json` for the SDK packages only. Merge when CI is green; no separate embed step.

Manual bump: edit the pin in that helper’s `package.json`, run `npm install` there to refresh the lockfile, then commit both files.

## Cursor rate catalog sync

[`cost/rates.json`](../cost/rates.json) is the checked-in Cursor model rate table used when the harness omits `costUSD`. A weekly GitHub Action ([`.github/workflows/sync-cursor-rates.yml`](../.github/workflows/sync-cursor-rates.yml)) fetches [Cursor teams pricing](https://cursor.com/docs/account/teams/pricing), maps display names via [`cost/cursor_aliases.json`](../cost/cursor_aliases.json), and opens a review PR when the Cursor catalog (or `asOf`) would change. PRs are never auto-merged.

- Parse failures and unmapped display names fail the job; they do not invent model ids or rewrite the catalog
- Known models missing from the docs page are kept (listed as stale candidates); removals need a human edit
- The Anthropic catalog stays untouched

Local dry-run / write:

```bash
go run ./cmd/ratesync
go run ./cmd/ratesync -write
```

When Cursor adds a model name that is not in `cursor_aliases.json`, add the display name → `--model` id mapping there, then re-run the sync (or wait for the next scheduled job).
