import type { ToolCall } from "../result.js";
import { matchContains, matchEquals } from "../stringMatch.js";
import { fail } from "./error.js";
import { NumericIntMatchers } from "./numeric.js";

/** string → equals; RegExp → substring/regex match (YAML contains). */
export type ArgMatcher = string | RegExp;

export interface ToolCallOrderStep {
  name: string | readonly [string, ...string[]];
  args?: Record<string, ArgMatcher>;
}

/**
 * Matchers over metrics.toolCalls: total count, per-name counts, and
 * ordered subsequence (gaps allowed).
 */
export class ToolCallsMatchers {
  constructor(private readonly calls: ToolCall[]) {}

  toBeGreaterThanOrEqual(n: number): this {
    this.count().toBeGreaterThanOrEqual(n);
    return this;
  }

  toBeLessThanOrEqual(n: number): this {
    this.count().toBeLessThanOrEqual(n);
    return this;
  }

  toBeGreaterThan(n: number): this {
    this.count().toBeGreaterThan(n);
    return this;
  }

  toBeLessThan(n: number): this {
    this.count().toBeLessThan(n);
    return this;
  }

  toBeEqual(n: number): this {
    this.count().toBeEqual(n);
    return this;
  }

  /** Count bounds for calls with this tool name. */
  named(name: string): NumericIntMatchers {
    let n = 0;
    for (const c of this.calls) {
      if (c.name === name) n++;
    }
    return new NumericIntMatchers(`toolCalls.named.${name}`, n);
  }

  /**
   * Ordered subsequence: each step must appear in order; extra calls may
   * sit before, between, or after. Args: string = equals, RegExp = match.
   */
  toIncludeInOrder(steps: ToolCallOrderStep[]): this {
    validateOrderStepNames(steps);
    let i = 0;
    for (let stepIdx = 0; stepIdx < steps.length; stepIdx++) {
      const step = steps[stepIdx]!;
      let found = false;
      for (; i < this.calls.length; i++) {
        if (matchesStep(this.calls[i]!, step)) {
          i++;
          found = true;
          break;
        }
      }
      if (!found) {
        const reason =
          step.args && Object.keys(step.args).length > 0
            ? `no matching tool call for order step ${JSON.stringify(step.name)} with given args`
            : `no matching tool call for order step ${JSON.stringify(step.name)}`;
        fail(`toolCalls.order[${stepIdx}]`, reason);
        // Match Go: stop after the first unmatched order step.
        return this;
      }
    }
    return this;
  }

  private count(): NumericIntMatchers {
    return new NumericIntMatchers("toolCalls", this.calls.length);
  }
}

function validateOrderStepNames(steps: ToolCallOrderStep[]): void {
  for (const step of steps) {
    const name = step.name;
    if (typeof name === "string") {
      if (name.trim() === "") {
        throw new Error("toIncludeInOrder: name is required");
      }
      continue;
    }
    if (!Array.isArray(name) || name.length === 0) {
      throw new Error("toIncludeInOrder: name is required");
    }
    for (const n of name) {
      if (typeof n !== "string" || n.trim() === "") {
        throw new Error("toIncludeInOrder: name is required");
      }
    }
  }
}

function nameMatches(callName: string, name: ToolCallOrderStep["name"]): boolean {
  if (typeof name === "string") {
    return callName === name;
  }
  return name.includes(callName);
}

function matchesStep(call: ToolCall, step: ToolCallOrderStep): boolean {
  if (!nameMatches(call.name, step.name)) return false;
  if (!step.args) return true;
  const args = call.args ?? {};
  for (const [key, matcher] of Object.entries(step.args)) {
    if (!(key in args)) return false;
    const haystack = argString(args[key]);
    if (typeof matcher === "string") {
      if (!matchEquals(haystack, matcher)) return false;
    } else if (!matchContains(haystack, matcher)) {
      return false;
    }
  }
  return true;
}

function argString(v: string | number | boolean | null | undefined): string {
  if (v === null || v === undefined) return "";
  return String(v);
}
