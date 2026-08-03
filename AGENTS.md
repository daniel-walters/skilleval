# skilleval

skilleval is a testing framework for agent skills (`SKILL.md`).

You give it a skill, a prompt, and optional input fixtures. It runs an agent with that skill available, records observables from the attempt (tools, turns, cost, which files changed, final message, and similar), and checks them against deterministic expectations. The goal is reliable skill evals without a second model judging the run.

The core is implemented in Go. SDKs for other languages may be added later so tests can be written in more familiar styles.
