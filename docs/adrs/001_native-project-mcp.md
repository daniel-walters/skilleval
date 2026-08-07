# 001. Native project MCP as fixture contract

## Status

Accepted

## Context

Many skills only work when MCP servers are available. The harness already places skills into runner-private trees and runs agents with `settingSources: ["project"]`. A spike (DAN-24) confirmed that seeding each runner’s native project MCP file enables MCP tools in attempt workspaces, and that host/global MCP does not leak in.

Authors need a runner-agnostic way to supply MCP config without inventing a second config language or baking runner-specific paths into `input` fixtures.

## Decision

Evals may set optional `mcp` to a path to a **native** MCP JSON file (the same shape Cursor/Claude already understand). The harness copies that file into the attempt workspace as:

| Runner | Path |
| --- | --- |
| Cursor | `.cursor/mcp.json` |
| Claude | `.mcp.json` |

Secrets stay out of committed fixtures; runners expand `${env:NAME}` (Cursor) / `${VAR}` (Claude) from the process environment. Interactive OAuth is unsupported for unattended evals.

## Consequences

- MCP-dependent skills become evaluable with the same seed-then-run flow as other evals.
- One fixture works for both runners; destination paths stay an implementation detail of each agent adapter.
- `toolsUsed` MCP names still differ by runner (`mcp` vs `mcp__<server>__<tool>`); expects authors must account for that.
- Claude’s seeded `.mcp.json` is excluded from file outcomes (alongside `.claude/`).

## Rejected alternatives

- **Parallel skilleval MCP schema** — duplicates native config and drifts from Cursor/Claude.
- **Runner-specific paths only under `input`** — breaks runner-agnostic evals (Cursor vs Claude paths differ).
- **Mock / recorded MCP by default** — prefer real no-auth or env-token servers first.
