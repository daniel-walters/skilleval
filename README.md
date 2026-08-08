# skilleval

A testing framework for agent skills (`SKILL.md`).

Run a skill against a prompt and fixtures, capture what happened, and check deterministic expectations — tools used, turns, cost, file changes, and similar observables. Built so skill behavior can be evaluated without a second judging model.

Core is Go. TypeScript authors can use the in-repo client at [`sdk/typescript`](sdk/typescript) (`run` + typed `expect`) over the same CLI and expect catalog.

## Install

Prerequisites:

- **Go** (module path targets Go 1.26+)
- **Node.js** (both runners embed a small Node helper)

Install the CLI:

```bash
go install github.com/daniel-walters/skilleval/cmd/skilleval@latest
```

Or build from a clone:

```bash
go build -o skilleval ./cmd/skilleval
```

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
| `name` | Eval id (also used under `--history`) |
| `prompt` | What the agent should do |
| `skill` | Directory containing `SKILL.md` |
| `input` | Optional fixture directory copied into the workspace |
| `mcp` | Optional path to a native MCP JSON file seeded into the workspace |
| `expects` | Deterministic checks on the Result / workspace |

String matches (`contains` / `equals`) are either a literal or a slash-delimited regex (`/pattern/`).

File expects use workspace-relative paths. Optional `status` is one of `created`, `modified`, or `deleted`; you can also assert content with `contains` / `equals`.

`turns` and `costUSD` accept optional numeric bounds (all independently checked when set):

| Field | Meaning |
| --- | --- |
| `min` | actual ≥ bound |
| `max` | actual ≤ bound |
| `gt` | actual > bound |
| `lt` | actual < bound |
| `eq` | actual == bound |

Nil `costUSD` on the Result fails any set cost bound. Existing max-only evals keep working.

See [`examples/refactor-helper/eval.yaml`](examples/refactor-helper/eval.yaml) (or the TypeScript twin [`eval.ts`](examples/refactor-helper/eval.ts)) for a complete document, or [`examples/mcp-ping/eval.yaml`](examples/mcp-ping/eval.yaml) for an MCP-dependent skill.

### TypeScript

YAML remains the CLI authoring path. The TypeScript package is the typed alternative: same Go CLI under the hood, same expect catalog, IntelliSense on matchers.

Prerequisites beyond Install above:

- Node.js 18+
- `skilleval` on `PATH`, or `SKILLEVAL_BIN` pointing at the binary

Install the package locally (not published to npm yet):

```bash
cd sdk/typescript
npm install
npm run build
```

From another package:

```bash
npm install /path/to/skilleval/sdk/typescript
```

Credentials are unchanged — see [Credentials](#credentials) (`CURSOR_API_KEY` / `ANTHROPIC_API_KEY`).

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

Matcher namespaces (`turns` / `costUSD`, `toolsUsed`, `skills.activated`, `file`, `finalMessage`) are documented in [`sdk/typescript/README.md`](sdk/typescript/README.md).

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

Retain summaries across runs with `--history`:

```bash
skilleval run eval.yaml --model <ID> --history .skilleval/history
```

That archives each summary under `.skilleval/history/<eval-name>/<timestamp>.json` and updates `latest.json` in the same folder.

Compare the current run to a prior summary with `--baseline` (informational deltas only — does not change exit status):

```bash
skilleval run eval.yaml --model <ID> \
  --history .skilleval/history \
  --baseline .skilleval/history/refactor-helper/latest.json
```

Or compare two summary files without re-running:

```bash
skilleval compare result-summary.json .skilleval/history/refactor-helper/latest.json
```

In CI, upload the current `*-summary.json` as an artifact; on the next job, download the prior summary and pass it as `--baseline`.

## CI

GitHub Actions is the source of truth for merge readiness. On pull requests and pushes to `main`, CI runs `gofmt`, `go vet`, `go test`, and `golangci-lint`. Local pre-commit is recommended for a faster feedback loop, but a green CI check is what counts for merge.
