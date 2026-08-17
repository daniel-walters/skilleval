# skilleval

Agent skills are prompts, not code. There's no compiler to catch a broken one.

Working alone, regressions slip through until a user hits them. On a team it's worse: someone tweaks a shared skill, and review is just a read-through and hope. skilleval runs the skill against a real agent and checks things you can measure: tools used, files changed, turns taken, cost, the final message. A change is judged on evidence, not guesswork.

```ts
import { run, expect } from "@danielwaltersdev/skilleval";

const { result, workspace } = await run({
  name: "refactor-helper",
  prompt: "…",
  skill: "./skills/refactor-helper",
  model: "composer-2.5",
});

expect(result).skills.activated.toInclude("refactor-helper");
expect(result, workspace).file("src/foo.go").toHaveBeenModified();
expect(result).costUSD.toBeLessThanOrEqual(1);
```

Treat a skill change like a code change.

![Baseline compare](docs/demo-compare.gif)

---

## Install

```bash
npm install @danielwaltersdev/skilleval
```

This installs the typed client and a `skilleval` binary for your platform. You need Node.js 18+.

For live runs, put credentials in a `.env` in the directory you run `skilleval` from. skilleval loads that file automatically; process env wins if both are set. Both runners also need Node.js for their embedded helpers.

| Runner                      | Env                                           | Where to mint |
| --------------------------- | --------------------------------------------- | ------------- |
| Cursor (default)            | `CURSOR_API_KEY`                              | [Dashboard → API Keys](https://cursor.com/dashboard/api) |
| Claude (`runner: "claude"`) | `ANTHROPIC_API_KEY` (or local `claude login`) | [Anthropic Console](https://console.anthropic.com/settings/keys) |

```bash
echo 'CURSOR_API_KEY=...' > .env
```

`--model` is required, and the id has to match the runner. Cursor ids (`composer-2.5`, `auto`) are not Anthropic ids (`claude-sonnet-5`). For a first eval, tool names, and the YAML field list, see [docs/authoring.md](docs/authoring.md).

If you want a standalone binary, grab one from [GitHub Releases](https://github.com/daniel-walters/skilleval/releases), or `go install github.com/daniel-walters/skilleval/cmd/skilleval@v0.1.0`. The Go binary runs YAML only. TypeScript evals need the npm `skilleval` bin.

---

## Write an eval

Start in a new folder if you're using TypeScript or the npm CLI. If you're only writing YAML and using the Go binary, you can skip npm.

```bash
npm init -y
npm install @danielwaltersdev/skilleval
```

Put a skill package next to the eval. `SKILL.md` needs a `name` in frontmatter; `description` is recommended:

```text
my-eval/
  eval.ts              # or eval.yaml
  skills/my-skill/SKILL.md
  fixtures/my-skill/   # optional; directory contents become the workspace
  mcp.json             # optional; native MCP config
```

Start with `skills.activated`, run once, then add expects from `result.json`. See [docs/authoring.md](docs/authoring.md). If the skill asks questions mid-run, script those with `replies` under [Interactive skills](#interactive-skills-replies).

### TypeScript

```ts
import { run, expect } from "@danielwaltersdev/skilleval";

const { result, workspace } = await run({
  name: "refactor-helper",
  prompt: `Use the refactor-helper skill on this package:

1. Refactor src/foo.go for clarity (simplify Foo; keep package demo and the Foo name).
2. Extract a small helper into a new file src/new.go (e.g. a greetPrefix helper used by Foo).
3. Delete src/gone.go. It is obsolete legacy code.

Do all three: modify foo.go, create new.go, delete gone.go.`,
  skill: "./skills/refactor-helper",
  input: "./fixtures/refactor-helper",
  model: "composer-2.5",
});

expect(result).turns.toBeLessThanOrEqual(15);
expect(result).costUSD.toBeLessThanOrEqual(1);
expect(result).toolsUsed.toInclude("read", "edit").not.toInclude("web");
expect(result).toolCalls.named("edit").toBeGreaterThanOrEqual(1);
expect(result).toolCalls.toIncludeInOrder([
  { name: ["write", "edit"], args: { path: "src/foo.go" } },
  { name: ["shell", "Bash"], args: { command: /git commit/ }, exitCode: 0 },
]);
expect(result).toolCalls.not.toIncludeInOrder([
  { name: ["shell", "Bash"], args: { command: /rm -rf/ }, exitCode: 0 },
]);
expect(result).skills.activated.toInclude("refactor-helper");
expect(result, workspace)
  .file("src/foo.go")
  .toHaveBeenModified()
  .toContain(/func Foo/)
  .not.toContain("TODO");
expect(result, workspace)
  .file("src/new.go")
  .toHaveBeenCreated()
  .toContain("package demo");
expect(result, workspace).file("src/gone.go").toHaveBeenDeleted();
expect(result).finalMessage.toMatch(/Refactor/);
```

Run it. Paths like `skill` and `input` are relative to the eval file's directory, so keep `eval.ts` next to `skills/` and `fixtures/`, or write paths relative to that folder:

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

Full example: `[examples/refactor-helper/eval.ts](examples/refactor-helper/eval.ts)`. Matcher reference: `[sdk/typescript/README.md](sdk/typescript/README.md)`.

You can also load an existing YAML eval and assert in TypeScript:

```ts
import { loadEval, run, expect } from "@danielwaltersdev/skilleval";

const ev = await loadEval("./eval.yaml");
const { result, workspace } = await run(ev, { model: "composer-2.5" });
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
  toolCalls:
    named:
      edit:
        min: 1
    order:
      - name: [write, edit]
        args:
          path:
            equals: src/foo.go
      - name: [shell, Bash]
        args:
          command:
            contains: /git commit/
        exitCode: 0
    orderExcludes:
      - name: [shell, Bash]
        args:
          command:
            contains: rm -rf
        exitCode: 0
  skills:
    activated:
      includes: [refactor-helper]
  files:
    src/foo.go:
      status: modified
      contains: /func Foo/
      excludes:
        - TODO
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
skilleval run ./eval.yaml --model claude-sonnet-5 --runner claude
```

`--model` is required. Bare `skilleval run` does not discover YAML files; pass the path. Paths in the YAML are relative to the YAML file.

Complete examples: `[examples/refactor-helper/eval.yaml](examples/refactor-helper/eval.yaml)`, `[examples/mcp-ping/eval.yaml](examples/mcp-ping/eval.yaml)`, `[examples/interactive-confirm/eval.yaml](examples/interactive-confirm/eval.yaml)`. Tool names differ by runner (`read` vs `Read`, `shell` vs `Bash`); see [docs/authoring.md](docs/authoring.md#tool-names).

### Interactive skills (`replies`)

Skills that ask a question or confirmation mid-run can take scripted follow-ups. After the initial `prompt` finishes, each entry in `replies` is sent as the next user message (same attempt, same workspace).

```yaml
prompt: |
  Use the interactive-confirm skill to clean up obsolete files.
replies:
  - yes
```

```ts
await run({
  name: "interactive-confirm",
  prompt: "Use the interactive-confirm skill to clean up obsolete files.",
  replies: ["yes"],
  skill: "./skills/interactive-confirm",
  input: "./fixtures/interactive-confirm",
  model: "composer-2.5",
});
```

The skill must complete a turn between interactions (ask, then stop). Worked example: `[examples/interactive-confirm/](examples/interactive-confirm/)`.

| Supported                         | Not supported yet                                      |
| --------------------------------- | ------------------------------------------------------ |
| Ordered scripted mid-run replies  | Live human-in-the-loop UI                              |
| Cursor and Claude runners         | Auto-answering AskQuestion / blocking elicitation tools |
|                                   | Interactive OAuth (unchanged; see MCP)                 |

---

## Assertions

TypeScript and YAML share the same assertion catalog.

String matches are either a literal or a slash-delimited regex (`/pattern/`). File paths are relative to the workspace. Optional `status` is `created`, `modified`, or `deleted`.

File `contains`, `equals`, and `excludes` (forbidden substrings or `/regex/`; YAML list or TypeScript `.not.toContain`) only apply when the file was recorded and was not deleted. That gate applies to both positive and negated content checks.

If several expects fail in one eval, skilleval reports all of them, then fails the eval. Unfinished runs still check `run.status` before anything else.

Numeric bounds (`turns`, `durationMs`, `toolCalls`, `costUSD`, and `usage.*Tokens`):

| Bound                            | Meaning         |
| -------------------------------- | --------------- |
| `min` / `toBeGreaterThanOrEqual` | actual ≥ bound  |
| `max` / `toBeLessThanOrEqual`    | actual ≤ bound  |
| `gt` / `toBeGreaterThan`         | actual > bound  |
| `lt` / `toBeLessThan`            | actual < bound  |
| `eq` / `toBeEqual`               | actual == bound |

`toolsUsed` and `skills.activated` support include / exclude membership. Nil `costUSD` fails any cost bound.

`toolCalls` also supports:

- **Total count:** same numeric bounds as above (`min` / `toBeGreaterThanOrEqual`, and so on)
- **`named.<tool>`:** count bounds for one tool name (`named.edit.min` / `toolCalls.named("edit").toBeGreaterThanOrEqual(1)`)
- **`order` / `toIncludeInOrder`:** ordered **subsequence** (gaps allowed before, between, and after). Each step has `name` (one tool name or a nonempty list), optional `args`, and optional `exitCode` (one integer or a list; match if equal to any). A list of names matches if the call name equals any of those names (exact). YAML arg checks use `contains` / `equals` (literal or `/regex/`). In TypeScript, a string arg means equals and a `RegExp` means match. `exitCode` is only valid when every listed name is `shell` or `Bash` (empty list or a non-shell name is an **invalid expect**).
- **`orderExcludes` / `.not.toIncludeInOrder`:** each step must **not** match any tool call. Steps are independent (not an ordered subsequence). Same step shape as `order`.

Result `metrics.toolCalls` entries may include lean pre-call `args`: a normalized `path` and `command`, with large bodies stripped. Claude `file_path` is mapped to `path`. Paths under the attempt workspace are stored relative to that workspace. `shell` and `Bash` calls may include `exitCode` (the process status) when the runner can observe it; `0` is stored as `0`. If the exit code is unknown, the field is omitted. It is not mixed into `args`, and it is not the same as tool-call `status` (whether the invocation completed).

A positive `exitCode` filter does not match a call with omitted `exitCode` (the checker keeps scanning). An `orderExcludes` step that sets `exitCode` **fails closed** if any name+args match has omitted `exitCode` (`exitCode unknown, cannot assert absence`). If a matching call with a known forbidden code also exists, that match is reported instead. Portable steps use `name: [shell, Bash]`. `.not.toIncludeInOrder` does not mean "this sequence didn't happen." TypeScript `run().exitCode` is the CLI process status, not a tool observable.

When the harness omits `costUSD`, skilleval estimates it from `cost/rates.json`. Unknown or unpriced models leave `costUSD` nil.

A CLI run writes `result.json` (per attempt), `result-agent-log.json` (the full turn/tool transcript next to that result), and `result-summary.json` (with `passRate` and averages). The agent log is written whenever a result is written, including errors and cancellations. It's for debugging, not for expects. Use those artifacts to learn tool names and file outcomes before tightening expects ([docs/authoring.md](docs/authoring.md#inspect-then-add-expects)). Optional `--timeout` bounds each attempt (Go duration, for example `30m`).

YAML string matches use Go regex (`/pattern/`, `/(?i)foo/`). TypeScript matchers use JavaScript `RegExp` (`/foo/i`). `skills.activated` means the runner saw the skill load (Cursor: a `read` of `SKILL.md`; Claude: the `Skill` tool). Naming the skill in the prompt is not enough.

Eval and Result JSON use `schemaVersion: 1`. That number only bumps on a breaking contract change.

---

## MCP

Pass a native MCP JSON file via `mcp` (TypeScript `run({ mcp: "…" })` or YAML `mcp:`). It's seeded into each attempt workspace:

| Runner | Seeded path        |
| ------ | ------------------ |
| Cursor | `.cursor/mcp.json` |
| Claude | `.mcp.json`        |

Both runners load project MCP only, so host/global MCP does not leak in. Put stdio server scripts under `input` so paths resolve inside the workspace. Example: `[examples/mcp-ping/](examples/mcp-ping/)` (`eval.yaml` and `eval.ts`).

| Bucket                          | Local           | CI               |
| ------------------------------- | --------------- | ---------------- |
| No auth                         | ✓               | ✓                |
| Env / token (`env` / `headers`) | ✓ if env set    | ✓ via CI secrets |
| Interactive OAuth               | Not automatable | Not supported    |

Interpolation: Cursor `${env:NAME}`, Claude `${VAR}`. Do not commit secrets in fixtures.

MCP tool names in `toolsUsed` differ by runner. Cursor uses `mcp`; Claude uses `mcp__<server>__<tool>`. Prefer runner-specific expects when asserting MCP tool names.

---

## Multi-run batches

YAML multi-run is scored by the CLI checker. TypeScript multi-run still asks the CLI to run N times, but pass/fail lives in `batch.expect`. The programmatic `passRate` is not written into the temp YAML, because empty YAML expects would always pass.

```ts
const batch = await run({
  name: "refactor-helper",
  prompt: "…",
  skill: "./skills/refactor-helper",
  model: "composer-2.5",
  attempts: 10,
  passRate: { min: 0.8 },
});

batch.expect(({ result, workspace }) => {
  expect(result, workspace).turns.toBeLessThanOrEqual(15);
  expect(result).finalMessage.toMatch(/Refactor/);
});
```

`batch.expect` isolates matcher failures per attempt, then fails the process only when the TypeScript pass rate is below `passRate.min`. If you omit `passRate`, the minimum is 1 (every attempt must pass). YAML is different: omitting `passRate` does not fail the process.

```yaml
attempts: 10
passRate:
  min: 0.8
```

Each attempt gets its own Result. With `attempts > 1`, the CLI also writes `result-1.json`, `result-1-agent-log.json`, and so on, plus `result-summary.json`. YAML exits non-zero only when `passRate.min` is set and the batch rate is below it. It does not fail on a single failed attempt. Single-attempt runs still exit based on expects. `run()` still returns `result` and `workspace` for the last successful write, so one-shot `expect(result)` evals stay the same.

YAML expects on a `loadEval` document are still scored by the CLI. `batch.expect` is the TypeScript-expects path. CLI stdout may print PASS for empty YAML expects; the `batch.expect` block is the TypeScript score.

---

## History and comparison

By default, runs retain summaries under `.skilleval/history/<eval-name>/` and compare against the prior `latest.json` when one exists. `.skilleval/` is gitignored. On a TTY (or with `FORCE_COLOR`), `PASS`/`FAIL` and polarity-aware baseline deltas are colored green/red. Set `NO_COLOR` to disable.

```bash
skilleval run ./eval.yaml --model composer-2.5
skilleval run ./eval.yaml --model composer-2.5 --no-history --no-baseline
skilleval compare result-summary.json .skilleval/history/refactor-helper/latest.json
```

In TypeScript, `run()` tees the same CLI report (including baseline compare) to stdout. Pass `noHistory` / `noBaseline`, or `history` / `baseline` path overrides. Comparison deltas do not change exit status.

In CI, upload `*-summary.json` as an artifact. On the next job, download the prior summary and pass `--baseline`.

---

## What's in v0.1

| In scope                                             | Not yet                            |
| ---------------------------------------------------- | ---------------------------------- |
| Cursor and Claude runners                            | Other agent runtimes (see Roadmap) |
| TypeScript `run` + `expect`, script discovery        | YAML suite discovery               |
| Multi-run batches, history, baseline compare         | Model matrix in one invocation     |
| Native project MCP seeding                           | Interactive OAuth MCP in CI        |
| Scripted mid-run user replies (`replies`)            | Live HITL / AskQuestion auto-fill  |
| npm platform binaries; GitHub Release / `go install` | Homebrew / apt                     |

## Roadmap

- **Skills for authoring evals:** agent skills that help write `eval.ts` / expects against the shared catalog
- **More providers:** Gemini, Codex, and OpenCode runners alongside Cursor and Claude
- **SDKs in more languages:** Go, Python
- **CI integration:** first-class GitHub Actions helpers (install, run, baseline artifact wiring) beyond today's manual summary upload pattern
- **YAML suite discovery:** discover and run many YAML evals like script discovery already does
- **Model matrix:** one invocation across several models with aggregated compare

---

## Docs

| Doc                                                  | Audience                                              |
| ---------------------------------------------------- | ----------------------------------------------------- |
| [docs/authoring.md](docs/authoring.md)               | First eval, models, tool names, YAML fields           |
| [sdk/typescript/README.md](sdk/typescript/README.md) | Matcher catalog, bin resolution, local SDK develop    |
| [docs/releasing.md](docs/releasing.md)               | Cutting CLI + npm releases                            |
| [docs/development.md](docs/development.md)           | Building from source, CI, SDK pins, rate catalog sync |
| [docs/adrs/](docs/adrs/)                             | Architecture decisions                                |

## License

[MIT](LICENSE)
