# skilleval

A testing framework for agent skills (`SKILL.md`).

Run a skill against a prompt and fixtures, capture what happened, and check deterministic expectations — tools used, turns, cost, file changes, and similar observables. Built so skill behavior can be evaluated without a second judging model.

Core is Go. TypeScript authors can use the in-repo client at [`sdk/typescript`](sdk/typescript) (`run` + typed `expect`) over the same CLI and expect catalog.

### Supported in v0.1

| In scope | Not yet |
| --- | --- |
| Cursor and Claude runners | Other agent runtimes (Codex, Gemini, …) |
| Single-eval CLI (`run` / `compare`) | Suite discovery / directory of evals |
| Multi-run batches, history, baseline compare | Model matrix in one invocation |
| Native project MCP seeding | Interactive OAuth MCP in CI |
| YAML evals + TypeScript client (`npm install skilleval`) | Bundled Go binary inside the npm package |
| GitHub Release binaries + `go install` | Homebrew / apt |

**Versioning:** CLI releases use semver tags (`v0.1.0`, …). Eval YAML and Result JSON stay on `schemaVersion: 1` until a breaking contract change; that bumps the schema number rather than silently changing meaning. Patch releases (`v0.1.x`) are for install, docs, and correctness fixes — not new product surface.

## Install

Prerequisites:

- **Node.js** (both runners embed a small Node helper; required at runtime)
- **Go** (only if you install via `go install` or build from source; module path targets Go 1.26+)

### GitHub Release binary

Download a prebuilt archive for your OS/arch from the [GitHub Releases](https://github.com/daniel-walters/skilleval/releases) page, extract `skilleval`, and put it on your `PATH`.

Archives are named `skilleval_<version>_<os>_<arch>` (`.tar.gz`, or `.zip` on Windows) and ship with a `checksums.txt`.

### go install

With Go installed:

```bash
go install github.com/daniel-walters/skilleval/cmd/skilleval@v0.1.0
```

Replace `v0.1.0` with a [release tag](https://github.com/daniel-walters/skilleval/releases). `@latest` also works once tags exist.

### Build from a clone

```bash
go build -o skilleval ./cmd/skilleval
```

Confirm the binary with `skilleval version` (prints `dev` for local builds; release builds print the tag version).

### Cutting a release

On `main`, tag and push:

```bash
git tag v0.1.0
git push origin v0.1.0
```

That runs the [Release](.github/workflows/release.yml) workflow (GoReleaser), which publishes multi-platform archives and checksums to GitHub Releases, and the [Publish npm](.github/workflows/publish-npm.yml) workflow, which publishes `sdk/typescript` as [`skilleval` on npm](https://www.npmjs.com/package/skilleval) via [npm trusted publishing](https://docs.npmjs.com/trusted-publishers) (OIDC — no long-lived `NPM_TOKEN`).

## Authoring an eval

An eval is a YAML file plus a skill directory (and optional input fixtures / MCP config). Paths in the YAML are relative to the YAML file.

Typical layout:

```text
my-eval/
  eval.yaml
  skills/my-skill/SKILL.md
  fixtures/my-skill/   # optional; copied into the attempt workspace
  mcp.json             # optional; native MCP config (see MCP below)
```

### Skill

Put a skill package at the path named by `skill`. It must contain `SKILL.md` with YAML frontmatter (`name`, `description`) and markdown instructions.

### Eval YAML

Minimum fields:

| Field | Meaning |
| --- | --- |
| `schemaVersion` | `1` |
| `name` | Eval id (also used for default history paths) |
| `prompt` | What the agent should do |
| `skill` | Directory containing `SKILL.md` |
| `input` | Optional fixture directory copied into the workspace |
| `mcp` | Optional path to a native MCP JSON file seeded into the workspace |
| `expects` | Deterministic checks on the Result / workspace |

String matches (`contains` / `equals`) are either a literal or a slash-delimited regex (`/pattern/`).

File expects use workspace-relative paths. Optional `status` is one of `created`, `modified`, or `deleted`; you can also assert content with `contains` / `equals`.

`turns`, `durationMs`, `toolCalls` (count of `metrics.toolCalls`), and `costUSD` accept optional numeric bounds (all independently checked when set):

| Field | Meaning |
| --- | --- |
| `min` | actual ≥ bound |
| `max` | actual ≤ bound |
| `gt` | actual > bound |
| `lt` | actual < bound |
| `eq` | actual == bound |

`usage` nests the same int bounds under `inputTokens`, `outputTokens`, `cacheReadTokens`, `cacheWriteTokens`, and `totalTokens` (e.g. `usage.totalTokens.max`).

`toolsUsed` and `skills.activated` support `includes` / `excludes` membership lists.

Nil `costUSD` on the Result fails any set cost bound. Existing max-only evals keep working.

See [`examples/refactor-helper/eval.yaml`](examples/refactor-helper/eval.yaml) (or the TypeScript twin [`eval.ts`](examples/refactor-helper/eval.ts)) for a complete document, or [`examples/mcp-ping/eval.yaml`](examples/mcp-ping/eval.yaml) for an MCP-dependent skill.

### TypeScript

YAML remains the CLI authoring path. The TypeScript package is the typed alternative: same Go CLI under the hood, same expect catalog, IntelliSense on matchers.

Prerequisites beyond Install above:

- Node.js 18+
- `skilleval` on `PATH`, or `SKILLEVAL_BIN` pointing at the binary

```bash
npm install skilleval
```

Requires the Go `skilleval` CLI on `PATH` (or `SKILLEVAL_BIN`). Credentials are unchanged — see [Credentials](#credentials) (`CURSOR_API_KEY` / `ANTHROPIC_API_KEY`).

Minimal `run` + `expect` equivalent to the refactor-helper YAML:

```ts
import { run, expect } from "skilleval";

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

Full worked example: [`examples/refactor-helper/eval.ts`](examples/refactor-helper/eval.ts).

You can also keep YAML for the prompt/skill/input and assert in TypeScript:

```ts
import { loadEval, run, expect } from "skilleval";

const ev = await loadEval("./eval.yaml");
const { result, workspace } = await run(ev, { model: process.env.MODEL! });
// expect(result, workspace)…
```

Matcher namespaces (`turns` / `durationMs` / `toolCalls` / `usage` / `costUSD`, `toolsUsed`, `skills.activated`, `file`, `finalMessage`) are documented in [`sdk/typescript/README.md`](sdk/typescript/README.md).

### MCP

Skills that need MCP tools can supply a **native** MCP JSON file via `mcp`. The harness copies it into each attempt workspace at the runner’s project path (no parallel skilleval schema):

| Runner | Seeded path |
| --- | --- |
| Cursor (`--runner cursor`) | `.cursor/mcp.json` |
| Claude (`--runner claude`) | `.mcp.json` (workspace root) |

Both runners load project MCP only (`settingSources: ["project"]`), so host/global MCP does not leak into the attempt.

Put stdio server scripts (and other files the MCP command needs) under `input` so paths in `mcp.json` resolve inside the seeded workspace. Example: [`examples/mcp-ping/`](examples/mcp-ping/).

**Auth matrix**

| Bucket | Mechanism | Local | CI |
| --- | --- | --- | --- |
| No auth | stdio/HTTP MCP with no secrets | Supported | Supported |
| Env / token | `env` / `headers` with interpolation | Supported if env set | Supported via CI secrets |
| Interactive OAuth | Browser / app login only | Not automatable in the SDK | Not supported |

Do not commit secrets in fixtures. Use runner interpolation instead:

- **Cursor:** `${env:NAME}` in `env` / `headers`
- **Claude:** `${VAR}` in `env` / `headers`

Example snippet (token never stored in the file):

```json
{
  "mcpServers": {
    "my-api": {
      "command": "npx",
      "args": ["-y", "my-mcp-server"],
      "env": {
        "API_TOKEN": "${env:API_TOKEN}"
      }
    }
  }
}
```

For Claude, prefer `"API_TOKEN": "${API_TOKEN}"` (Claude’s `${VAR}` form).

**`toolsUsed` naming** differs by runner when asserting MCP calls:

- Cursor: generic tool name `mcp`
- Claude: server-qualified `mcp__<server>__<tool>` (e.g. `mcp__echo-mcp__ping`)

The same eval YAML cannot share a single `toolsUsed.includes` entry across both runners for MCP; write runner-specific expects or run Cursor and Claude as separate evals.

### Credentials

Live runs need credentials in the environment (or a `.env` in the current working directory):

- **Cursor** (`--runner cursor`, default): `CURSOR_API_KEY`
- **Claude** (`--runner claude`): `ANTHROPIC_API_KEY` for CI / headless runs, or an existing `claude login` / Max subscription session locally (`claude auth login`)

```bash
echo 'CURSOR_API_KEY=...' > .env
# or
echo 'ANTHROPIC_API_KEY=...' > .env
```

Already-set process environment variables win over `.env` (use that in CI). `.env` is gitignored. Local Claude subscription auth does not replace `ANTHROPIC_API_KEY` in CI.

### Run to a Verdict

```bash
skilleval run examples/refactor-helper/eval.yaml --model <ID>
skilleval run examples/mcp-ping/eval.yaml --model <ID>
```

Use `--runner claude` for the Claude agent. The eval YAML stays runner-agnostic for skill/input/mcp seeding; MCP `toolsUsed` expects may still need runner-specific names (see above). The Result records which runner produced the attempt.

Optional `--timeout` bounds each attempt (Go duration, e.g. `30m` or `5m`). When omitted, attempts run without a deadline. On timeout the harness cancels the agent helper and exits with an error (no Result written for that attempt).

When the harness omits `costUSD`, skilleval estimates it from a **per-provider** rate table (`cost/rates.json`): Cursor uses the `cursor` catalog; Claude prefers the SDK’s `total_cost_usd` and does not fall back to Cursor rates (the `anthropic` catalog is reserved for later and may be empty).

After the run, expects are checked against the Result (and attempt workspace when needed). The CLI prints `PASS` or `FAIL` and exits non-zero when the check fails.

Artifacts (defaults):

- `result.json` — per-attempt Result
- `result-summary.json` — summary with `passRate` and average metrics (written for every run, including single-attempt)

## Running evals

```bash
skilleval run <eval.yaml> --model <ID>
skilleval run <eval.yaml> --model <ID> --runner claude
```

Every run writes a summary JSON beside `--out` (default `result-summary.json`) with `passRate` and average metrics, including single-attempt runs.

### Multi-run batches

Set `attempts` in the eval YAML to run the same eval more than once:

```yaml
schemaVersion: 1
name: refactor-helper
attempts: 10
passRate:
  min: 0.8   # optional batch gate
prompt: ...
skill: ...
expects: ...
```

Each attempt still gets its own Result and `PASS`/`FAIL`. When `attempts` is greater than 1, the CLI also writes per-attempt files derived from `--out` (default `result.json`) — e.g. `result-1.json`, `result-2.json` — plus `result-summary.json`, and prints aggregate `passRate`, `avgTurns`, and `avgCostUSD` (when available).

Batch exit status is **not** fail-on-any-attempt. The process exits non-zero only when `passRate.min` is set and the batch pass rate is below that threshold. Omit `passRate` to treat multi-run as informational (exit 0 after the summary, barring harness errors). Single-attempt runs (`attempts` omitted or `1`) keep the original expect-based pass/fail exit behavior.

### History and comparison

By default, `skilleval run` retains each summary under `.skilleval/history/<eval-name>/` (timestamped JSON plus `latest.json`) and prints an informational comparison against that eval's prior `latest.json` when one exists. First runs (no prior history) skip compare cleanly. `.skilleval/` is gitignored.

```bash
skilleval run eval.yaml --model <ID>
```

Opt out for ephemeral one-shots:

```bash
skilleval run eval.yaml --model <ID> --no-history --no-baseline
# retain but skip compare:
skilleval run eval.yaml --model <ID> --no-baseline
```

Override the history directory or pass an explicit baseline (e.g. a CI artifact). Comparison deltas do not change exit status:

```bash
skilleval run eval.yaml --model <ID> \
  --history /tmp/eval-history \
  --baseline /tmp/prior-summary.json
```

Or compare two summary files without re-running:

```bash
skilleval compare result-summary.json .skilleval/history/refactor-helper/latest.json
```

In CI, upload the current `*-summary.json` as an artifact; on the next job, download the prior summary and pass it as `--baseline`.

## CI

GitHub Actions is the source of truth for merge readiness. On pull requests and pushes to `main`, CI runs `gofmt`, `go vet`, `go test`, `golangci-lint`, and the TypeScript SDK tests (`sdk/typescript`). Local pre-commit is recommended for a faster feedback loop, but a green CI check is what counts for merge.

### Harness agent SDK pins

Each runner embeds a small Node helper with an exact npm pin and lockfile (`npm ci` at prepare time). Those files are the `go:embed` inputs:

| Runner | Package | Pin + lockfile |
| --- | --- | --- |
| Cursor | `@cursor/sdk` | [`runner/cursoragent/`](runner/cursoragent/) |
| Claude | `@anthropic-ai/claude-agent-sdk` | [`runner/claudeagent/`](runner/claudeagent/) |

[Dependabot](.github/dependabot.yml) checks those directories weekly and opens PRs that update `package.json` + `package-lock.json` for the SDK packages only. Merge when CI is green; no separate embed step. Manual bump: edit the pin in that helper’s `package.json`, run `npm install` there to refresh the lockfile, then commit both files.

### Cursor rate catalog sync

[`cost/rates.json`](cost/rates.json) is the checked-in Cursor model rate table used when the harness omits `costUSD`. A weekly GitHub Action ([`.github/workflows/sync-cursor-rates.yml`](.github/workflows/sync-cursor-rates.yml)) fetches [Cursor teams pricing](https://cursor.com/docs/account/teams/pricing), maps display names via [`cost/cursor_aliases.json`](cost/cursor_aliases.json), and opens a review PR when the Cursor catalog (or `asOf`) would change. PRs are never auto-merged.

- Parse failures and unmapped display names fail the job; they do not invent model ids or rewrite the catalog
- Known models missing from the docs page are kept (listed as stale candidates); removals need a human edit
- The Anthropic catalog stays untouched

Local dry-run / write:

```bash
go run ./cmd/ratesync
go run ./cmd/ratesync -write
```

When Cursor adds a model name that is not in `cursor_aliases.json`, add the display name → `--model` id mapping there, then re-run the sync (or wait for the next scheduled job).

## License

[MIT](LICENSE)
