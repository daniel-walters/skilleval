# skilleval

A testing framework for agent skills (`SKILL.md`).

Give it a skill, a prompt, and optional fixtures. It runs an agent with that skill available, records what happened (tools, turns, cost, file changes, final message, and similar), and checks those observables against deterministic expectations — no second model judging the run.

- **CLI** — YAML evals via the `skilleval` binary
- **TypeScript** — [`@danielwaltersdev/skilleval`](https://www.npmjs.com/package/@danielwaltersdev/skilleval) (`run` + typed `expect` over the same CLI)

### Supported in v0.1

| In scope | Not yet |
| --- | --- |
| Cursor and Claude runners | Other agent runtimes (Codex, Gemini, …) |
| Single-eval CLI (`run` / `compare`); TS discover by name (`eval.ts` / `*.eval.ts`) | YAML suite discovery / directory of evals |
| Multi-run batches, history, baseline compare | Model matrix in one invocation |
| Native project MCP seeding | Interactive OAuth MCP in CI |
| YAML + TypeScript client (npm ships platform CLI); GitHub Release / `go install` | Homebrew / apt |

Eval YAML and Result JSON use `schemaVersion: 1`. That number bumps only on a breaking contract change.

## Install

**Prerequisites:** Node.js at runtime (both runners embed a small Node helper). Go is only needed for `go install` or building from source.

### TypeScript (recommended)

```bash
npm install @danielwaltersdev/skilleval
```

That installs the typed client and a platform `skilleval` binary (via optionalDependencies). Use it from `package.json` scripts or `npx`:

```json
{
  "scripts": {
    "eval": "skilleval run ./eval.yaml --model \"$MODEL\""
  }
}
```

TypeScript-only projects can call `skilleval` without `tsx` on the eval file:

```json
{
  "scripts": {
    "eval": "skilleval run"
  }
}
```

That discovers `eval.{ts,mts,js,mjs}` and `*.eval.{ts,mts,js,mjs}` under the current directory (skips `node_modules`, hidden dirs, and common build outputs like `dist`), or pass an explicit path: `skilleval run ./eval.ts`. Put `model` on `run({ … })` or in `MODEL`. Override with `SKILLEVAL_BIN` if needed. See [TypeScript](#typescript) below.

### CLI binary only

Download a prebuilt archive for your OS/arch from [GitHub Releases](https://github.com/daniel-walters/skilleval/releases), extract `skilleval`, and put it on your `PATH`.

Archives are named `skilleval_<version>_<os>_<arch>` (`.tar.gz`, or `.zip` on Windows) and ship with a `checksums.txt`.

Or with Go installed:

```bash
go install github.com/daniel-walters/skilleval/cmd/skilleval@v0.1.0
```

Replace `v0.1.0` with a [release tag](https://github.com/daniel-walters/skilleval/releases) (`@latest` also works). Confirm with `skilleval version` (local builds print `dev`; release builds print the tag version).

## Credentials

Live runs need credentials in the environment. Both the **CLI** and TypeScript `run()` load a `.env` from the current working directory (process env wins over `.env`).

- **Cursor** (`--runner cursor`, default): `CURSOR_API_KEY`
- **Claude** (`--runner claude`): `ANTHROPIC_API_KEY` for CI / headless runs, or an existing `claude login` / Max subscription session locally

```bash
echo 'CURSOR_API_KEY=...' > .env
# or
echo 'ANTHROPIC_API_KEY=...' > .env
export MODEL=composer-2.5   # example Cursor model id (also used below)
```

Already-set process environment variables win over `.env` (use that in CI). `.env` is gitignored. Local Claude subscription auth does not replace `ANTHROPIC_API_KEY` in CI.

## Quick start

```bash
skilleval run examples/refactor-helper/eval.yaml --model composer-2.5
skilleval run examples/mcp-ping/eval.yaml --model composer-2.5
skilleval run examples/refactor-helper/eval.yaml --model composer-2.5 --runner claude
```

Optional `--timeout` bounds each attempt (Go duration, e.g. `30m`). When omitted, attempts run without a deadline.

The CLI checks expects against the Result (and workspace when needed), prints `PASS` or `FAIL`, and exits non-zero on failure.

Default artifacts:

- `result.json` — per-attempt Result
- `result-summary.json` — summary with `passRate` and average metrics (including single-attempt runs)

When the harness omits `costUSD`, skilleval estimates it from a per-provider rate table (`cost/rates.json`): Cursor uses the `cursor` catalog; Claude prefers the SDK’s `total_cost_usd`. Unknown or unpriced `--model` ids leave `costUSD` nil; any cost bound then fails.

## Authoring an eval

An eval is a YAML file plus a skill directory (and optional input fixtures / MCP config). Paths in the YAML are relative to the YAML file.

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

Complete examples: [`examples/refactor-helper/eval.yaml`](examples/refactor-helper/eval.yaml) and [`examples/mcp-ping/eval.yaml`](examples/mcp-ping/eval.yaml).

## TypeScript

YAML remains the CLI authoring path. The npm package is the typed alternative: same Go CLI under the hood (shipped for your platform), same expect catalog, IntelliSense on matchers.

Requires Node.js 18+. Credentials — see [Credentials](#credentials) (`run()` loads cwd `.env` the same as the CLI).

```ts
import { run, expect } from "@danielwaltersdev/skilleval";

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

Full worked example: [`examples/refactor-helper/eval.ts`](examples/refactor-helper/eval.ts). Run it with the npm `skilleval` bin (from a project that depends on `@danielwaltersdev/skilleval`, or this repo after `npm run build` in `sdk/typescript`):

```bash
cd examples/refactor-helper
MODEL=composer-2.5 skilleval run ./eval.ts
# or discover eval.ts / *.eval.ts under cwd:
MODEL=composer-2.5 skilleval run
```

Keep YAML for the prompt/skill/input and assert in TypeScript:

```ts
import { loadEval, run, expect } from "@danielwaltersdev/skilleval";

const ev = await loadEval("./eval.yaml");
const { result, workspace } = await run(ev, { model: process.env.MODEL! });
// expect(result, workspace)…
```

Matcher namespaces (`turns` / `durationMs` / `toolCalls` / `usage` / `costUSD`, `toolsUsed`, `skills.activated`, `file`, `finalMessage`) are documented in [`sdk/typescript/README.md`](sdk/typescript/README.md).

## MCP

Skills that need MCP tools can supply a **native** MCP JSON file via `mcp`. The harness copies it into each attempt workspace at the runner’s project path:

| Runner | Seeded path |
| --- | --- |
| Cursor (`--runner cursor`) | `.cursor/mcp.json` |
| Claude (`--runner claude`) | `.mcp.json` (workspace root) |

Both runners load project MCP only (`settingSources: ["project"]`), so host/global MCP does not leak into the attempt.

Put stdio server scripts (and other files the MCP command needs) under `input` so paths in `mcp.json` resolve inside the seeded workspace. Example: [`examples/mcp-ping/`](examples/mcp-ping/).

| Bucket | Mechanism | Local | CI |
| --- | --- | --- | --- |
| No auth | stdio/HTTP MCP with no secrets | Supported | Supported |
| Env / token | `env` / `headers` with interpolation | Supported if env set | Supported via CI secrets |
| Interactive OAuth | Browser / app login only | Not automatable in the SDK | Not supported |

Do not commit secrets in fixtures. Use runner interpolation:

- **Cursor:** `${env:NAME}` in `env` / `headers`
- **Claude:** `${VAR}` in `env` / `headers`

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

For Claude, prefer `"API_TOKEN": "${API_TOKEN}"`.

**`toolsUsed` naming** differs by runner for MCP calls:

- Cursor: generic tool name `mcp`
- Claude: server-qualified `mcp__<server>__<tool>` (e.g. `mcp__echo-mcp__ping`)

The same eval YAML cannot share a single `toolsUsed.includes` entry across both runners for MCP; write runner-specific expects or run Cursor and Claude as separate evals. The Result records which runner produced the attempt.

## Multi-run batches

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

Batch exit status is **not** fail-on-any-attempt. The process exits non-zero only when `passRate.min` is set and the batch pass rate is below that threshold. Omit `passRate` to treat multi-run as informational (exit 0 after the summary, barring harness errors). Single-attempt runs (`attempts` omitted or `1`) keep expect-based pass/fail exit behavior.

## History and comparison

By default, `skilleval run` retains each summary under `.skilleval/history/<eval-name>/` (timestamped JSON plus `latest.json`) and prints an informational comparison against that eval's prior `latest.json` when one exists. First runs skip compare cleanly. `.skilleval/` is gitignored.

```bash
skilleval run eval.yaml --model composer-2.5
```

Opt out for ephemeral one-shots:

```bash
skilleval run eval.yaml --model composer-2.5 --no-history --no-baseline
# retain but skip compare:
skilleval run eval.yaml --model composer-2.5 --no-baseline
```

Override the history directory or pass an explicit baseline (e.g. a CI artifact). Comparison deltas do not change exit status:

```bash
skilleval run eval.yaml --model composer-2.5 \
  --history /tmp/eval-history \
  --baseline /tmp/prior-summary.json
```

Or compare two summary files without re-running:

```bash
skilleval compare result-summary.json .skilleval/history/refactor-helper/latest.json
```

In CI, upload the current `*-summary.json` as an artifact; on the next job, download the prior summary and pass it as `--baseline`.

## Docs

| Doc | Audience |
| --- | --- |
| [docs/releasing.md](docs/releasing.md) | Cutting CLI + npm releases |
| [docs/development.md](docs/development.md) | Building from source, CI, SDK pins, rate catalog sync |
| [docs/adrs/](docs/adrs/) | Architecture decisions (incl. npm platform binaries) |

## License

[MIT](LICENSE)
