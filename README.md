# skilleval

A testing framework for agent skills (`SKILL.md`).

Run a skill against a prompt and fixtures, capture what happened, and check deterministic expectations — tools used, turns, cost, file changes, and similar observables. Built so skill behavior can be evaluated without a second judging model.

Core is Go; language SDKs may come later for writing tests in the ecosystem you prefer.

## CI

GitHub Actions is the source of truth for merge readiness. On pull requests and pushes to `main`, CI runs `gofmt`, `go vet`, `go test`, and `golangci-lint`. Local pre-commit is recommended for a faster feedback loop, but a green CI check is what counts for merge.
