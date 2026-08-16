# 008. Shell tool exit codes

## Status

Accepted

## Context

Authors need to assert that a shell command succeeded or failed (tests passed; a dangerous command must not succeed). Tool calls already store `name`, `status`, and lean pre-call `args`. `status` is invocation completion (`completed` / `error`), not the process exit. There was no integer to expect on.

## Decision

1. Optional `exitCode` (integer) on each `ToolCall`. Omitted when the runner cannot observe it. `0` is present in JSON. Not stored in `args`. `status` does not change.
2. Record only for exact names `shell` and `Bash`. A portable order step uses `name: [shell, Bash]`.
3. Positive expects are an extra **match filter** on an order step, like `args`: find a later call whose name, args, and exit code all match. `exitCode` is one integer or a list (match if equal to any). Retries: a later success satisfies `{ command: /go test/, exitCode: 0 }`.
4. Forbidden expects are `toolCalls.orderExcludes` / TypeScript `.not.toIncludeInOrder`. Any step shape. Each exclude step is checked on its own against the full call list — not as an ordered subsequence. Empty `orderExcludes` is a no-op.
5. Unknown `exitCode`: a positive filter does not match (keep scanning). An exclude step fails closed if any name+args match has an omitted `exitCode` (`exitCode unknown, cannot assert absence`). If a full match (known forbidden code) also exists, report that in preference to unknown.
6. **Invalid expect**, not a failed check: empty `exitCode` list; non-integer; `exitCode` set when any listed name is not `shell` or `Bash`. Duplicate codes are allowed. No 0–255 clamp.
7. Additive on `schemaVersion: 1`.

## Consequences

- Authors can require a successful `go test` without treating a failed retry as the last word, and can forbid a command succeeding.
- `.not.toIncludeInOrder` does not mean “this sequence didn’t happen.”
- Claude may omit `exitCode` until `tool_result` is observed: positive exit-code steps fail; excludes fail closed.
- TypeScript `run().exitCode` remains the CLI process status, not a tool observable.
- The agent log may carry the same `exitCode` when observed; it is still not an expect surface.

## Rejected alternatives

- **Map non-zero onto `status: error`** — loses `0` vs `1` vs `127`; `status` already means invocation completion and is unused by expects.
- **Stuff the integer into `args`** — mixes pre-call and post-call; every arg matcher would see it.
- **Assert on the first name+args match** — a failed then successful `go test` would fail “tests must pass.”
- **Subsequence dual of `.not.toIncludeInOrder`** — two forbidden steps would pass unless they appeared in that order; weaker and surprising.
- **Numeric bounds (`gt` / `min` / `max`)** — exit codes are a set of integers, not a run metric.
- **Bump `schemaVersion` to 2** — optional fields are additive, same as ADR 005 and 007.
- **Exit-code-only excludes** — the operation is “this step must not match,” not a special polarity on one field.
