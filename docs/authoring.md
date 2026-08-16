# Authoring evals

Write a small eval, run it once, inspect the result, then add expects. The [README](../README.md) has the full TypeScript and YAML examples; this page is the first-eval path and the catalogs those examples assume.

## First eval

From an empty folder (Node.js 18+):

```bash
npm init -y
npm install @danielwaltersdev/skilleval
```

`tsx` is bundled with the package — you do not add it yourself. Put `model` on `run({ … })` for TypeScript; YAML takes `--model` on the CLI. Script evals do not accept CLI flags.

Layout (the fixture directory’s *contents* become the attempt workspace root):

```text
my-eval/
  eval.ts                 # or eval.yaml
  skills/my-skill/SKILL.md
  fixtures/my-skill/      # optional
```

`SKILL.md` needs YAML frontmatter with a `name` (required; `[A-Za-z0-9][A-Za-z0-9._-]*`). `description` is recommended.

Smallest TypeScript eval:

```ts
import { run, expect } from "@danielwaltersdev/skilleval";

const { result } = await run({
  name: "my-skill",
  prompt: "Use the my-skill skill.",
  skill: "./skills/my-skill",
  model: "composer-2.5",
});

expect(result).skills.activated.toInclude("my-skill");
```

```json
{
  "scripts": {
    "eval": "skilleval run"
  }
}
```

```bash
skilleval run ./eval.ts
# discovers eval.ts / *.eval.ts under cwd (not YAML):
skilleval run
```

Smallest YAML eval:

```yaml
schemaVersion: 1
name: my-skill
prompt: Use the my-skill skill.
skill: skills/my-skill
expects:
  skills:
    activated:
      includes: [my-skill]
```

```bash
skilleval run ./eval.yaml --model composer-2.5
```

`model` and `runner` are CLI flags (or `run({ model, runner })` in TypeScript), not YAML fields. `skilleval run` with no path only discovers TypeScript. YAML always needs an explicit path, and `--model` is required.

Worked examples: [`examples/refactor-helper/`](../examples/refactor-helper/), [`examples/mcp-ping/`](../examples/mcp-ping/), [`examples/interactive-confirm/`](../examples/interactive-confirm/). Matcher catalog: [`sdk/typescript/README.md`](../sdk/typescript/README.md).

## Credentials and models

cwd `.env` is loaded automatically; process env wins. Both runners also need Node.js for their embedded helpers.


| Runner | Env | Where to mint |
| --- | --- | --- |
| Cursor (default) | `CURSOR_API_KEY` | [Cursor Dashboard → API Keys](https://cursor.com/dashboard/api) |
| Claude (`--runner claude` / `runner: "claude"`) | `ANTHROPIC_API_KEY` (or local `claude login`) | [Anthropic Console → API keys](https://console.anthropic.com/settings/keys) |


`--model` / `run({ model })` is a **runner-specific** id. Cursor ids and Anthropic ids are not interchangeable.


| Runner | Example ids | Catalog |
| --- | --- | --- |
| Cursor | `composer-2.5`, `auto`, `claude-4.5-sonnet` | Cursor model id. Cost estimates use [`cost/rates.json`](../cost/rates.json). |
| Claude | `claude-sonnet-5`, `claude-opus-5` | [Anthropic model ids](https://docs.anthropic.com/en/docs/about-claude/models) |


```bash
skilleval run ./eval.yaml --model composer-2.5
skilleval run ./eval.yaml --model claude-sonnet-5 --runner claude
```

Unknown or unpriced models leave `costUSD` nil; any cost expect then fails.

## Inspect, then add expects

Do not invent `toolsUsed` names or file paths. Run a thin eval first, then tighten expects from the artifacts:

1. Run with `skills.activated` (and maybe a cost/turns cap).
2. Open `result.json` — `metrics.toolsUsed`, `metrics.toolCalls`, `outcomes.files`, `finalMessage`, `skills.activated`.
3. Open `result-agent-log.json` beside it for the turn/tool transcript (debug only; not an expect surface).
4. Add expects that match what the skill should keep doing.

Default artifacts: `result.json` (per attempt), `result-agent-log.json`, `result-summary.json`. With `attempts > 1`, also `result-1.json` / `result-1-agent-log.json`, ….

## Tool names

skilleval records the runner’s native tool names. They are **not** the same across Cursor and Claude. Use a name list on an order step when the eval should pass on both (`name: [shell, Bash]`).


| Intent | Cursor | Claude |
| --- | --- | --- |
| Read a file | `read` | `Read` |
| Create a file | `write` | `Write` |
| Edit in place | `edit` | `Edit` |
| Shell | `shell` | `Bash` |
| MCP | `mcp` | `mcp__<server>__<tool>` |


`toolsUsed` and `toolCalls.named` are exact strings — `named("edit")` does not match Claude `Edit`. Prefer runner-specific expects for MCP. Portable file-edit order:

```yaml
toolCalls:
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
```

```ts
expect(result).toolCalls.toIncludeInOrder([
  { name: ["write", "edit"], args: { path: "src/foo.go" } },
  { name: ["shell", "Bash"], args: { command: /git commit/ }, exitCode: 0 },
]);
```

## Skill activation

`skills.activated` is an observed signal, not “the prompt mentioned the skill”:

- **Cursor** — the agent completed a `read` of that skill’s `SKILL.md`.
- **Claude** — the agent invoked the `Skill` tool with that skill name.

If the agent never loads the skill body, the include expect fails even when the final message looks right.

## YAML fields

Required: `schemaVersion` (`1`), `name`, `prompt`, `skill`.

Optional:


| Field | Meaning |
| --- | --- |
| `replies` | Ordered mid-run user messages (see README [Interactive skills](../README.md#interactive-skills-replies)) |
| `input` | Fixture directory; contents become the workspace |
| `mcp` | Native MCP JSON file (see README [MCP](../README.md#mcp)) |
| `attempts` | How many times to run (default 1) |
| `passRate.min` | Batch gate, 0–1 (YAML CLI only fails the process when this is set) |
| `expects` | Predicates over the Result (omit to record a run with no checks) |


`expects` keys (all optional; each set bound is checked):

| Key | YAML | TypeScript |
| --- | --- | --- |
| `turns` / `durationMs` / `costUSD` / `usage.*Tokens` | `min` `max` `gt` `lt` `eq` | `toBeGreaterThanOrEqual` / `toBeLessThanOrEqual` / `toBeGreaterThan` / `toBeLessThan` / `toBeEqual` |
| `toolsUsed` / `skills.activated` | `includes` / `excludes` | `.toInclude` / `.not.toInclude` |
| `toolCalls` | total bounds; `named.<tool>` bounds; `order`; `orderExcludes` | numeric + `named()` + `toIncludeInOrder` / `.not.toIncludeInOrder` |
| `files.<path>` | `status` (`created` \| `modified` \| `deleted`); `contains` / `equals`; `excludes` | `.file(path)` + status / `toContain` / `toEqual` / `.not.toContain` |
| `finalMessage` | `contains` / `equals` | `.toContain` / `.toMatch` / `.toEqual` |

File `contains` / `equals` / `excludes` require a recorded, **non-deleted** file outcome. An untouched fixture path fails those checks.

String matches: YAML is a literal or a slash-delimited **Go** regexp (`/pattern/`, e.g. `/(?i)deleted/`). TypeScript file/message matchers take a string (literal) or a **JavaScript** `RegExp` (`/deleted/i`). Do not copy `(?i)` into TypeScript or `/pattern/i` into YAML.

TypeScript `run({ … })` writes `schemaVersion` for you. Relative `skill` / `input` / `mcp` paths resolve from the eval file’s directory.
