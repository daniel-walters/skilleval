# 002. TypeScript SDK as CLI-backed client

## Status

Accepted

## Context

skilleval’s core is Go. Many skill authors live in TypeScript and want a familiar `run` → `expect` authoring path. Spike DAN-20 aligned on building a TypeScript SDK rather than waiting, and on integrating via the Go CLI rather than a native TS harness.

Without a recorded decision, later work risks re-litigating CLI-vs-native packaging and shipping either a shallow spawn wrapper or a divergent second runner.

## Decision

Ship an in-repo TypeScript package at `sdk/typescript` (npm name `@danielwaltersdev/skilleval`; unscoped `skilleval` is blocked by npm as too similar to `skill-eval`) as a **deep client over the Go CLI**:

- `run` / eval execution invoke the `skilleval` binary (`SKILLEVAL_BIN`, then `PATH`).
- Authors assert with typed **Jest-like matcher names** in TypeScript. The package does **not** depend on Vitest or Jest and is not a test runner.
- There is **one expect catalog** shared with YAML; v1 adds no TypeScript-only asserts.
- Harness, agents, and checker **policy** stay Go-owned for v1. The SDK surfaces failures with checker-style `path` + `reason`; catalog semantics live in Go.

## Consequences

- Authors get IntelliSense on what each Result part supports without reimplementing the harness.
- One binary owns attempts, cost, history, and agent adapters; the TypeScript package stays thin.
- Matcher parity with YAML requires the Go catalog to express the same bounds (e.g. numeric `min` / `max` / `lt` / `gt` / `eq`).
- Publishing / npm release automation is separate from this packaging choice.

## Rejected alternatives

- **Shallow CLI wrap** — thin `spawn` plus raw Result with no typed matchers. Pushes catalog and assertion complexity onto every caller; fails the deep-modules bar.
- **Full native TypeScript runner** — reimplements harness, agents, and checker in TS. Drifts from Go, doubles maintenance, and is out of scope for v1.
