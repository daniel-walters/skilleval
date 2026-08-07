/** Checker-style expectation failure (path + reason). */
export class ExpectError extends Error {
  readonly path: string;
  readonly reason: string;

  constructor(path: string, reason: string) {
    super(`${path}: ${reason}`);
    this.name = "ExpectError";
    this.path = path;
    this.reason = reason;
  }
}

export function fail(path: string, reason: string): never {
  throw new ExpectError(path, reason);
}
