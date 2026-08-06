# skilleval

A testing framework for agent skills (`SKILL.md`).

Run a skill against a prompt and fixtures, capture what happened, and check deterministic expectations — tools used, turns, cost, file changes, and similar observables. Built so skill behavior can be evaluated without a second judging model.

Core is Go; language SDKs may come later for writing tests in the ecosystem you prefer.

## Running evals

```bash
skilleval run <eval.yaml> --model <ID>
```

After the run, expects from the eval YAML are checked against the Result (and attempt workspace when needed). The CLI prints `PASS` or `FAIL` and exits non-zero when the check fails.

### Multi-run batches

Set `attempts` in the eval YAML to run the same eval more than once:

```yaml
schemaVersion: 1
name: refactor-helper
attempts: 10
passRate:
  min: 0.8   # optional batch gate
prompt: ...
skill: ...
expects: ...
```

Each attempt still gets its own Result and `PASS`/`FAIL`. When `attempts` is greater than 1, the CLI also writes per-attempt files derived from `--out` (default `result.json`) — e.g. `result-1.json`, `result-2.json` — plus `result-summary.json`, and prints aggregate `passRate`, `avgTurns`, and `avgCostUSD` (when available).

Batch exit status is **not** fail-on-any-attempt. The process exits non-zero only when `passRate.min` is set and the batch pass rate is below that threshold. Omit `passRate` to treat multi-run as informational (exit 0 after the summary, barring harness errors). Single-attempt runs (`attempts` omitted or `1`) keep the original expect-based pass/fail exit behavior.

Live runs need a Cursor API key. Set `CURSOR_API_KEY` in the process environment, or put it in a `.env` file in the current working directory:

```bash
echo 'CURSOR_API_KEY=...' > .env
```

Already-set process environment variables win over `.env` (use that in CI). `.env` is gitignored.

## CI

GitHub Actions is the source of truth for merge readiness. On pull requests and pushes to `main`, CI runs `gofmt`, `go vet`, `go test`, and `golangci-lint`. Local pre-commit is recommended for a faster feedback loop, but a green CI check is what counts for merge.
