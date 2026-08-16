# 007. More than one name on an order step

## Status

Accepted

## Context

Some skills can use `write` or `edit` for the same intent. An order step had one exact name. Authors could not say that one of several names is enough. They had to over-constrain the eval or skip the order check.

## Decision

1. The field stays `name`. It is one string or a list of strings (YAML and TypeScript).
2. A tool call matches the names if its tool name equals one of them (exact string equality). `args` on the step apply to that same call.
3. Order remains an ordered subsequence (ADR 005). The next matching call satisfies the step.
4. Additive on `schemaVersion: 1`. Single-name steps stay valid.
5. An empty list or a name with no text is an **invalid expect**, not a **failed check**. YAML: the eval does not load. TypeScript: `name` is `string | readonly [string, ...string[]]` so an empty list does not compile. The SDK throws if invalid names still reach `toIncludeInOrder`.
6. Duplicate names are permitted (same meaning as one name).

## Consequences

- Authors can write a write-or-edit order step without two evals.
- A TypeScript variable of type `string[]` is not a valid `name` (the list might be empty). Put the list in the object, or use `as const`.
- `toolCalls.named` and `toolsUsed` do not change.

## Rejected alternatives

- **Second field `names`** — two ways to write the same check.
- **Separator string (`write|edit`)** — easy to confuse with a real tool name.
- **Regular expression on `name`** — the ticket is a list of names, not a pattern.
- **Report invalid names as `toolCalls.order[N]` failed checks** — looks like the skill failed.
- **Reject duplicate names** — extra rule, no extra meaning.
