# 006. Scripted user replies for interactive skills

## Status

Accepted

## Context

Many skills ask clarifying questions or confirmations mid-run. skilleval attempts were one-shot (`prompt` only), so authors either skipped those skills or stuffed answers into the initial prompt (which does not exercise the interactive path). DAN-48 needs an unattended, deterministic way to drive at least one mid-run user turn without a human-in-the-loop UI.

## Decision

1. **Optional `replies: string[]`** on the eval (YAML and TypeScript `run()`), additive on `schemaVersion: 1`. The first user message remains `prompt`; each reply is sent after the previous agent leg finishes.
2. **Wait-between-legs:** on `finished`, send the next reply; on `error` / `cancelled`, stop and leave remaining replies unused.
3. **Both runners:** Cursor keeps one `Agent` and loops `send` → `wait`. Because `@cursor/sdk` local agents do not retain conversation across `send()` today, follow-up legs inject a compact prior user/assistant/tool transcript into the next `send()` text (agent log still records the clean `replies` entry). Claude captures `session_id` and uses `options.resume` for follow-ups (not `continue: true`, which races under parallel attempts). Multi-leg Cursor attempts keep stream-aggregated turns and agent-log events; `run.conversation()` is only preferred for single-prompt attempts (each Run owns one prompt’s transcript).
4. **One Result per attempt:** sum turns / duration / usage / cost; union tools and activated skills; concatenate tool calls; last leg wins for status, error, and final message; initial prompt only on `result.eval.prompt`; full multi-user transcript in the agent log sidecar.
5. **Author obligation:** the skill must complete a turn between interactions (ask, then stop). Blocking elicitation that never finishes the run is unsupported (timeout only).

## Consequences

- Interactive confirm / clarify skills become evaluable in CI with scripted replies.
- Authors own skill pause behavior; stuffing answers into `prompt` is no longer the only option.
- Observables and expects stay single-attempt aggregates — no new “must have asked” expect.
- Cursor local follow-ups depend on transcript injection until the SDK retains native multi-turn context.

## Rejected alternatives

- **AskQuestion / elicitation auto-answer** — runner-specific and brittle; prose questions on Claude would still need replies.
- **Stuffing answers into `prompt`** — does not test the interactive path.
- **Separate evals per turn** — loses mid-run continuity and workspace state.
- **Live human-in-the-loop UI** — out of scope for unattended evals (same posture as Interactive OAuth).
