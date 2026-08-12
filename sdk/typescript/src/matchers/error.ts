/** Checker-style expectation failure (path + reason). */
export type ExpectFailure = {
  path: string;
  reason: string;
};

/** One or more collected expectation failures. */
export class ExpectError extends Error {
  readonly failures: readonly ExpectFailure[];
  /** First failure path (back-compat). */
  readonly path: string;
  /** First failure reason (back-compat). */
  readonly reason: string;

  constructor(path: string, reason: string);
  constructor(failures: ExpectFailure[]);
  constructor(pathOrFailures: string | ExpectFailure[], reason?: string) {
    const failures: ExpectFailure[] =
      typeof pathOrFailures === "string"
        ? [{ path: pathOrFailures, reason: reason ?? "" }]
        : pathOrFailures;
    super(formatFailures(failures));
    this.name = "ExpectError";
    this.failures = failures;
    this.path = failures[0]?.path ?? "";
    this.reason = failures[0]?.reason ?? "";
  }
}

function formatFailures(failures: ExpectFailure[]): string {
  return failures.map((f) => `${f.path}: ${f.reason}`).join("\n");
}

const bag: ExpectFailure[] = [];
let flushRegistered = false;

function registerFlush(): void {
  if (flushRegistered) return;
  flushRegistered = true;
  process.on("beforeExit", () => {
    if (bag.length === 0) return;
    for (const f of bag) {
      console.error(`${f.path}: ${f.reason}`);
    }
    bag.length = 0;
    if (process.exitCode === undefined || process.exitCode === 0) {
      process.exitCode = 1;
    }
  });
}

/** Soft-collect a failure; does not throw. Matchers continue so all asserts run. */
export function fail(path: string, reason: string): void {
  bag.push({ path, reason });
  registerFlush();
}

/** Hard-fail (e.g. run.status gate); throws immediately and skips further expects. */
export function failHard(path: string, reason: string): never {
  throw new ExpectError(path, reason);
}

/** Drain collected failures without throwing. */
export function clearFailures(): void {
  bag.length = 0;
}

/** Snapshot of pending failures (does not clear). */
export function peekFailures(): readonly ExpectFailure[] {
  return bag.slice();
}

/**
 * Drain the bag; if any failures were collected, throw ExpectError with all of them.
 * Prevents the beforeExit handler from printing the same list.
 */
export function reportFailures(): void {
  if (bag.length === 0) return;
  const failures = bag.splice(0, bag.length);
  throw new ExpectError(failures);
}
