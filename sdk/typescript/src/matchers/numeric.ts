import { fail } from "./error.js";

type IntOp = "min" | "max" | "gt" | "lt" | "eq";

export class NumericIntMatchers {
  constructor(
    private readonly prefix: string,
    private readonly actual: number,
  ) {}

  toBeGreaterThanOrEqual(n: number): this {
    this.check("min", this.actual >= n, () => `${this.prefix} ${this.actual} below min ${n}`);
    return this;
  }

  toBeLessThanOrEqual(n: number): this {
    this.check("max", this.actual <= n, () => `${this.prefix} ${this.actual} exceeds max ${n}`);
    return this;
  }

  toBeGreaterThan(n: number): this {
    this.check("gt", this.actual > n, () => `${this.prefix} ${this.actual} not greater than ${n}`);
    return this;
  }

  toBeLessThan(n: number): this {
    this.check("lt", this.actual < n, () => `${this.prefix} ${this.actual} not less than ${n}`);
    return this;
  }

  toBeEqual(n: number): this {
    this.check("eq", this.actual === n, () => `${this.prefix} ${this.actual} not equal to ${n}`);
    return this;
  }

  private check(op: IntOp, ok: boolean, reason: () => string): void {
    if (!ok) {
      fail(`${this.prefix}.${op}`, reason());
    }
  }
}

type FloatOp = "min" | "max" | "gt" | "lt" | "eq";

export class NumericFloatMatchers {
  constructor(
    private readonly prefix: string,
    private readonly actual: number | null,
  ) {}

  toBeGreaterThanOrEqual(n: number): this {
    this.check("min", (v) => v >= n, (v) => `${this.prefix} ${v} below min ${n}`);
    return this;
  }

  toBeLessThanOrEqual(n: number): this {
    this.check("max", (v) => v <= n, (v) => `${this.prefix} ${v} exceeds max ${n}`);
    return this;
  }

  toBeGreaterThan(n: number): this {
    this.check("gt", (v) => v > n, (v) => `${this.prefix} ${v} not greater than ${n}`);
    return this;
  }

  toBeLessThan(n: number): this {
    this.check("lt", (v) => v < n, (v) => `${this.prefix} ${v} not less than ${n}`);
    return this;
  }

  toBeEqual(n: number): this {
    this.check("eq", (v) => v === n, (v) => `${this.prefix} ${v} not equal to ${n}`);
    return this;
  }

  private check(
    op: FloatOp,
    pred: (v: number) => boolean,
    reason: (v: number) => string,
  ): void {
    if (this.actual === null) {
      fail(
        `${this.prefix}.${op}`,
        `${this.prefix} is unknown (nil), cannot satisfy ${op} bound`,
      );
    }
    if (!pred(this.actual)) {
      fail(`${this.prefix}.${op}`, reason(this.actual));
    }
  }
}
