# 005. Tool call args and ordered subsequence expects

## Status

Accepted

## Context

Authors need to assert *how* a skill drove the agent — call order, per-tool counts, and pre-call args (edit path, shell command) — not only final outputs and set membership of tool names. Results already stored ordered `metrics.toolCalls` as `{name,status}` without args. Expects only bounded total length.

## Decision

1. **Capture lean pre-call `args`** on each `ToolCall` (schemaVersion 1, additive). Normalize Claude `file_path` → `path`; relativize `path` against the attempt workspace cwd (forward slashes); keep `command` and other small scalars; strip large bodies (`fileText`, `content`, `oldText`/`newText`, `old_string`/`new_string`, …).
2. **Order expects are ordered subsequences**, not exact full sequences: steps must appear in order with gaps allowed before/between/after (`toolCalls.order` / `toIncludeInOrder`).
3. **Per-name counts** live under `toolCalls.named` alongside existing total bounds.
4. **TS arg values** are `string` (equals) or `RegExp` (match); YAML keeps `contains` / `equals` like file/text expects.

## Consequences

- Authors can catch wrong-file edits and wrong shell commands without re-running for transcript inspection alone.
- Result artefacts stay small because bodies are stripped.
- Exact “these N calls and nothing else” still needs an explicit total `eq` / `toBeEqual` (or a future exact-sequence matcher).

## Rejected alternatives

- **Exact full-sequence matching as the default** — too brittle for agent runs with exploratory reads.
- **Store raw unfiltered args** — inflates `result.json` with file bodies and patches.
- **Jest-spy style (`toolCall("edit").toHaveBeenCalledWith`)** — awkward for multi-step order with args on every step.
