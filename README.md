# skilleval

**Deterministic evals for agent skills.**

Run a skill against a real agent, then assert on deterministic observables — tools, files, turns, cost, final message. No LLM-as-judge.

```ts
import { run, expect } from "@danielwaltersdev/skilleval";

const { result, workspace } = await run({
  name: "refactor-helper",
  prompt: "…",
  skill: "./skills/refactor-helper",
  model: process.env.MODEL!,
});

expect(result).skills.activated.toInclude("refactor-helper");
expect(result, workspace).file("src/foo.go").toHaveBeenModified();
expect(result).costUSD.toBeLessThanOrEqual(1);
```

Ship skills with the same confidence you ship code.

---

## Install

```bash
npm install @danielwaltersdev/skilleval
```

That gives you the typed client and a platform `skilleval` binary. Node.js 18+.

Credentials for live runs (cwd `.env` is loaded automatically; process env wins):

| Runner | Env |
| --- | --- |
| Cursor (default) | `CURSOR_API_KEY` |
| Claude (`runner: "claude"`) | `ANTHROPIC_API_KEY` (or local `claude login`) |

```bash
echo 'CURSOR_API_KEY=...' > .env
export MODEL=composer-2.5
```

Prefer a standalone binary? Grab one from [GitHub Releases](https://github.com/daniel-walters/skilleval/releases), or `go install github.com/daniel-walters/skilleval/cmd/skilleval@v0.1.0`.

---

## Write an eval

Drop a skill package next to your eval (must contain `SKILL.md` with `name` / `description` frontmatter):

```text
my-eval/
  eval.ts
  skills/my-skill/SKILL.md
  fixtures/my-skill/   # optional; copied into the attempt workspace
  mcp.json             # optional; native MCP config
```

### TypeScript

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

Run it:

```bash
skilleval run ./eval.ts
# or discover eval.ts / *.eval.ts under cwd:
skilleval run
```

```json
{
  "scripts": {
    "eval": "skilleval run"
  }
}
```

Full example: [`examples/refactor-helper/eval.ts`](examples/refactor-helper/eval.ts). Matcher reference: [`sdk/typescript/README.md`](sdk/typescript/README.md).

Load an existing YAML eval and assert in TypeScript if you want both:

```ts
import { loadEval, run, expect } from "@danielwaltersdev/skilleval";

const ev = await loadEval("./eval.yaml");
const { result, workspace } = await run(ev, { model: process.env.MODEL! });
```

### YAML

```yaml
schemaVersion: 1
name: refactor-helper
prompt: |
  Use the refactor-helper skill…
skill: skills/refactor-helper
input: fixtures/refactor-helper
expects:
  turns:
    max: 15
  costUSD:
    max: 1
  toolsUsed:
    includes: [read, edit]
    excludes: [web]
  skills:
    activated:
      includes: [refactor-helper]
  files:
    src/foo.go:
      status: modified
      contains: /func Foo/
    src/new.go:
      status: created
      contains: "package demo"
    src/gone.go:
      status: deleted
  finalMessage:
    contains: /Refactor/
```

```bash
skilleval run ./eval.yaml --model composer-2.5
skilleval run ./eval.yaml --model composer-2.5 --runner claude
```

Paths in the YAML are relative to the YAML file. Complete examples: [`examples/refactor-helper/eval.yaml`](examples/refactor-helper/eval.yaml), [`examples/mcp-ping/eval.yaml`](examples/mcp-ping/eval.yaml).

---

## Assertions

Same catalog whether you write TypeScript or YAML. String matches are a literal or a slash-delimited regex (`/pattern/`). File paths are workspace-relative; optional `status` is `created` | `modified` | `deleted`.

Numeric bounds (`turns`, `durationMs`, `toolCalls`, `costUSD`, and `usage.*Tokens`):

| Bound | Meaning |
| --- | --- |
| `min` / `toBeGreaterThanOrEqual` | actual ≥ bound |
| `max` / `toBeLessThanOrEqual` | actual ≤ bound |
| `gt` / `toBeGreaterThan` | actual > bound |
| `lt` / `toBeLessThan` | actual < bound |
| `eq` / `toBeEqual` | actual == bound |

`toolsUsed` and `skills.activated` support include / exclude membership. Nil `costUSD` fails any cost bound.

When the harness omits `costUSD`, skilleval estimates it from `cost/rates.json`. Unknown or unpriced models leave `costUSD` nil.

Default artifacts from a CLI run: `result.json` (per attempt) and `result-summary.json` (with `passRate` and averages). Optional `--timeout` bounds each attempt (Go duration, e.g. `30m`).

Eval and Result JSON use `schemaVersion: 1` — bumps only on a breaking contract change.

---

## MCP

Pass a native MCP JSON file via `mcp` (TypeScript `run({ mcp: "…" })` or YAML `mcp:`). It’s seeded into each attempt workspace:

| Runner | Seeded path |
| --- | --- |
| Cursor | `.cursor/mcp.json` |
| Claude | `.mcp.json` |

Both runners load project MCP only, so host/global MCP does not leak in. Put stdio server scripts under `input` so paths resolve inside the workspace. Example: [`examples/mcp-ping/`](examples/mcp-ping/).

| Bucket | Local | CI |
| --- | --- | --- |
| No auth | ✓ | ✓ |
| Env / token (`env` / `headers`) | ✓ if env set | ✓ via CI secrets |
| Interactive OAuth | Not automatable | Not supported |

Interpolation: Cursor `${env:NAME}`, Claude `${VAR}`. Do not commit secrets in fixtures.

**`toolsUsed` naming** for MCP differs by runner — Cursor uses `mcp`; Claude uses `mcp__<server>__<tool>`. Prefer runner-specific expects when asserting MCP tool names.

---

## Multi-run batches

Set `attempts` (and optional `passRate.min`) on `run({ … })` or in YAML:

```ts
await run({
  name: "refactor-helper",
  prompt: "…",
  skill: "./skills/refactor-helper",
  model: process.env.MODEL!,
  attempts: 10,
  passRate: { min: 0.8 },
});
```

```yaml
attempts: 10
passRate:
  min: 0.8
```

Each attempt gets its own Result. With `attempts > 1`, the CLI also writes `result-1.json`, … and `result-summary.json`. Exit is non-zero only when `passRate.min` is set and the batch rate is below it — not fail-on-any-attempt. Single-attempt runs keep expect-based exit behavior.

---

## History and comparison

By default, runs retain summaries under `.skilleval/history/<eval-name>/` and compare against the prior `latest.json` when one exists. `.skilleval/` is gitignored.

```bash
skilleval run ./eval.yaml --model composer-2.5
skilleval run ./eval.yaml --model composer-2.5 --no-history --no-baseline
skilleval compare result-summary.json .skilleval/history/refactor-helper/latest.json
```

In TypeScript, pass `noHistory` / `noBaseline`, or `history` / `baseline` path overrides. Comparison deltas do not change exit status.

In CI, upload `*-summary.json` as an artifact; on the next job, download the prior summary and pass `--baseline`.

---

## What’s in v0.1

| In scope | Not yet |
| --- | --- |
| Cursor and Claude runners | Other agent runtimes (Codex, Gemini, …) |
| TypeScript `run` + `expect`, script discovery | YAML suite discovery |
| Multi-run batches, history, baseline compare | Model matrix in one invocation |
| Native project MCP seeding | Interactive OAuth MCP in CI |
| npm platform binaries; GitHub Release / `go install` | Homebrew / apt |

---

## Docs

| Doc | Audience |
| --- | --- |
| [sdk/typescript/README.md](sdk/typescript/README.md) | Matcher catalog, bin resolution, local SDK develop |
| [docs/releasing.md](docs/releasing.md) | Cutting CLI + npm releases |
| [docs/development.md](docs/development.md) | Building from source, CI, SDK pins, rate catalog sync |
| [docs/adrs/](docs/adrs/) | Architecture decisions |

## License

[MIT](LICENSE)
