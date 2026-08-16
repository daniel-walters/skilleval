import type { PassRateExpect } from "./evalTypes.js";
import type { AttemptOutcome, Result } from "./result.js";
import {
  clearFailures,
  fail,
  failHard,
  peekFailures,
  reportFailures,
  withIsolatedFailures,
  type ExpectFailure,
} from "./matchers/error.js";
import { FileMatchers } from "./matchers/file.js";
import { FinalMessageMatchers } from "./matchers/finalMessage.js";
import { NumericFloatMatchers, NumericIntMatchers } from "./matchers/numeric.js";
import { SkillsActivatedMatchers, ToolsUsedMatchers } from "./matchers/sets.js";
import { ToolCallsMatchers } from "./matchers/toolCalls.js";

export { ExpectError } from "./matchers/error.js";
export type { ExpectFailure } from "./matchers/error.js";
export type { ArgMatcher, ToolCallOrderStep } from "./matchers/toolCalls.js";

export interface Expectation {
  readonly turns: NumericIntMatchers;
  readonly durationMs: NumericIntMatchers;
  readonly toolCalls: ToolCallsMatchers;
  readonly usage: {
    readonly inputTokens: NumericIntMatchers;
    readonly outputTokens: NumericIntMatchers;
    readonly cacheReadTokens: NumericIntMatchers;
    readonly cacheWriteTokens: NumericIntMatchers;
    readonly totalTokens: NumericIntMatchers;
  };
  readonly costUSD: NumericFloatMatchers;
  readonly toolsUsed: ToolsUsedMatchers;
  readonly skills: { readonly activated: SkillsActivatedMatchers };
  readonly finalMessage: FinalMessageMatchers;
  file(relPath: string): FileMatchers;
}

export type ExpectFn = {
  (result: Result, workspace?: string): Expectation;
  /** Drain collected failures; throw ExpectError with the full list if any. */
  report(): void;
  /** Clear pending failures without throwing (tests). */
  clear(): void;
  /** Snapshot of pending failures. */
  failures(): readonly ExpectFailure[];
};

export type AttemptExpectFn = (ctx: { result: Result; workspace: string }) => void;

/** TS-side batch score from `RunResult.expect` (not the CLI YAML summary). */
export interface BatchExpectReport {
  attempts: number;
  passed: number;
  passRate: number;
}

/**
 * Typed expect matchers over a Result. Failures are collected (not
 * short-circuited) and reported together — via process beforeExit in eval
 * scripts, or `expect.report()` for an immediate throw. Non-finished results
 * hard-fail as run.status before any other check.
 *
 * @param workspace Attempt workspace path; required for file content matchers.
 */
export const expect: ExpectFn = Object.assign(
  function expect(result: Result, workspace?: string): Expectation {
    return new ExpectationImpl(result, workspace);
  },
  {
    report: reportFailures,
    clear: clearFailures,
    failures: peekFailures,
  },
);

/**
 * Apply `fn` to each attempt with an isolated failure bag, then gate on
 * `passRate.min` (default 1). Per-attempt failures are printed; they are not
 * left in the process bag when the batch meets the minimum. Below min, records
 * `passRate.min` so beforeExit / `expect.report()` fail the process.
 *
 * `write` is for tests; defaults to stdout.
 */
export function expectAttempts(
  attempts: readonly AttemptOutcome[],
  fn: AttemptExpectFn,
  passRate?: PassRateExpect,
  write: (line: string) => void = (line) => console.log(line),
): BatchExpectReport {
  const n = attempts.length;
  const min = passRate?.min ?? 1;
  const failedIndexes: number[] = [];
  let passed = 0;

  for (let i = 0; i < n; i++) {
    const slot = attempts[i]!;
    const index = i + 1;
    const failures = scoreAttempt(slot, fn);
    if (failures.length === 0) {
      passed++;
      write(`attempt ${index}/${n}: PASS`);
      continue;
    }
    failedIndexes.push(index);
    write(`attempt ${index}/${n}: FAIL`);
    for (const f of failures) {
      write(`  ${f.path}: ${f.reason}`);
    }
  }

  const rate = n === 0 ? 0 : passed / n;
  write("---");
  write(`passRate: ${rate} (${passed}/${n})`);

  if (rate < min) {
    const failed =
      failedIndexes.length > 0 ? ` (failed attempts ${failedIndexes.join(", ")})` : "";
    fail("passRate.min", `pass rate ${rate} below min ${min}${failed}`);
  }

  return { attempts: n, passed, passRate: rate };
}

function scoreAttempt(slot: AttemptOutcome, fn: AttemptExpectFn): ExpectFailure[] {
  if (slot.error !== undefined || slot.result === undefined || slot.workspace === undefined) {
    return [{ path: "run.error", reason: slot.error ?? "no Result" }];
  }
  const result = slot.result;
  const workspace = slot.workspace;
  return withIsolatedFailures(() => {
    fn({ result, workspace });
  });
}

class ExpectationImpl implements Expectation {
  constructor(
    private readonly result: Result,
    private readonly workspace: string | undefined,
  ) {}

  private ensureFinished(): void {
    if (this.result.status !== "finished") {
      failHard(
        "run.status",
        `status is ${JSON.stringify(this.result.status)}, want "finished"`,
      );
    }
  }

  get turns(): NumericIntMatchers {
    this.ensureFinished();
    return new NumericIntMatchers("turns", this.result.metrics.turns);
  }

  get durationMs(): NumericIntMatchers {
    this.ensureFinished();
    return new NumericIntMatchers("durationMs", this.result.metrics.durationMs);
  }

  get toolCalls(): ToolCallsMatchers {
    this.ensureFinished();
    const calls = this.result.metrics.toolCalls ?? [];
    return new ToolCallsMatchers(calls);
  }

  get usage(): {
    readonly inputTokens: NumericIntMatchers;
    readonly outputTokens: NumericIntMatchers;
    readonly cacheReadTokens: NumericIntMatchers;
    readonly cacheWriteTokens: NumericIntMatchers;
    readonly totalTokens: NumericIntMatchers;
  } {
    this.ensureFinished();
    const u = this.result.metrics.usage ?? {
      inputTokens: 0,
      outputTokens: 0,
      cacheReadTokens: 0,
      cacheWriteTokens: 0,
      totalTokens: 0,
    };
    return {
      inputTokens: new NumericIntMatchers("usage.inputTokens", u.inputTokens),
      outputTokens: new NumericIntMatchers("usage.outputTokens", u.outputTokens),
      cacheReadTokens: new NumericIntMatchers("usage.cacheReadTokens", u.cacheReadTokens),
      cacheWriteTokens: new NumericIntMatchers("usage.cacheWriteTokens", u.cacheWriteTokens),
      totalTokens: new NumericIntMatchers("usage.totalTokens", u.totalTokens),
    };
  }

  get costUSD(): NumericFloatMatchers {
    this.ensureFinished();
    return new NumericFloatMatchers("costUSD", this.result.metrics.costUSD);
  }

  get toolsUsed(): ToolsUsedMatchers {
    this.ensureFinished();
    return new ToolsUsedMatchers(this.result.metrics.toolsUsed ?? []);
  }

  get skills(): { readonly activated: SkillsActivatedMatchers } {
    this.ensureFinished();
    const activated = new SkillsActivatedMatchers(this.result.skills.activated ?? []);
    return { activated };
  }

  get finalMessage(): FinalMessageMatchers {
    this.ensureFinished();
    return new FinalMessageMatchers(this.result.finalMessage ?? "");
  }

  file(relPath: string): FileMatchers {
    this.ensureFinished();
    return new FileMatchers(this.result, relPath, this.workspace);
  }
}
