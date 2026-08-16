import type { ToolCall } from "../result.js";
import { matchContains, matchEquals } from "../stringMatch.js";
import { fail } from "./error.js";
import { NumericIntMatchers } from "./numeric.js";

/** string → equals; RegExp → substring/regex match (YAML contains). */
export type ArgMatcher = string | RegExp;

export interface ToolCallOrderStep {
  name: string | readonly [string, ...string[]];
  args?: Record<string, ArgMatcher>;
  /** One integer or a nonempty list; only valid when every name is shell or Bash. */
  exitCode?: number | readonly [number, ...number[]];
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
    validateOrderSteps(steps);
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
        fail(`toolCalls.order[${stepIdx}]`, unmatchedOrderStepReason(step));
        // Match Go: stop after the first unmatched order step.
        return this;
      }
    }
    return this;
  }

  /**
   * Each step must not match any tool call (independent; not a subsequence dual).
   */
  get not(): { toIncludeInOrder: (steps: ToolCallOrderStep[]) => ToolCallsMatchers } {
    return {
      toIncludeInOrder: (steps: ToolCallOrderStep[]) => {
        validateOrderSteps(steps);
        for (let stepIdx = 0; stepIdx < steps.length; stepIdx++) {
          const step = steps[stepIdx]!;
          let matched = false;
          let unknown = false;
          for (const call of this.calls) {
            if (!matchesNameArgs(call, step)) continue;
            if (step.exitCode === undefined) {
              matched = true;
              continue;
            }
            if (call.exitCode === undefined) {
              unknown = true;
              continue;
            }
            if (exitCodeMatches(call.exitCode, step.exitCode)) {
              matched = true;
            }
          }
          if (matched) {
            fail(
              `toolCalls.orderExcludes[${stepIdx}]`,
              `forbidden tool call matched order step ${JSON.stringify(step.name)}`,
            );
            continue;
          }
          if (unknown) {
            fail(`toolCalls.orderExcludes[${stepIdx}]`, "exitCode unknown, cannot assert absence");
          }
        }
        return this;
      },
    };
  }

  private count(): NumericIntMatchers {
    return new NumericIntMatchers("toolCalls", this.calls.length);
  }
}

function unmatchedOrderStepReason(step: ToolCallOrderStep): string {
  const reason = `no matching tool call for order step ${JSON.stringify(step.name)}`;
  const hasArgs = step.args !== undefined && Object.keys(step.args).length > 0;
  const hasExit = step.exitCode !== undefined;
  if (hasArgs && hasExit) return `${reason} with given args and exit code`;
  if (hasArgs) return `${reason} with given args`;
  if (hasExit) return `${reason} with given exit code`;
  return reason;
}

function isShellToolName(name: string): boolean {
  return name === "shell" || name === "Bash";
}

function stepNames(name: ToolCallOrderStep["name"]): readonly string[] {
  return typeof name === "string" ? [name] : name;
}

function validateOrderSteps(steps: ToolCallOrderStep[]): void {
  for (const step of steps) {
    const name = step.name;
    if (typeof name === "string") {
      if (name.trim() === "") {
        throw new Error("toIncludeInOrder: name is required");
      }
    } else if (!Array.isArray(name) || name.length === 0) {
      throw new Error("toIncludeInOrder: name is required");
    } else {
      for (const n of name) {
        if (typeof n !== "string" || n.trim() === "") {
          throw new Error("toIncludeInOrder: name is required");
        }
      }
    }
    if (step.exitCode === undefined) continue;
    const codes = normalizeExitCodes(step.exitCode);
    if (codes.length === 0) {
      throw new Error("toIncludeInOrder: exitCode must not be empty");
    }
    for (const n of stepNames(step.name)) {
      if (!isShellToolName(n)) {
        throw new Error(`toIncludeInOrder: exitCode is only valid for shell or Bash, not ${JSON.stringify(n)}`);
      }
    }
  }
}

function normalizeExitCodes(exitCode: number | readonly number[]): number[] {
  if (typeof exitCode === "number") {
    if (!Number.isInteger(exitCode)) {
      throw new Error("toIncludeInOrder: exitCode must be an integer");
    }
    return [exitCode];
  }
  if (!Array.isArray(exitCode)) {
    throw new Error("toIncludeInOrder: exitCode must not be empty");
  }
  const out: number[] = [];
  for (const n of exitCode) {
    if (typeof n !== "number" || !Number.isInteger(n)) {
      throw new Error("toIncludeInOrder: exitCode must be an integer");
    }
    out.push(n);
  }
  return out;
}

function exitCodeMatches(actual: number, want: number | readonly number[]): boolean {
  const codes = typeof want === "number" ? [want] : want;
  return codes.includes(actual);
}

function nameMatches(callName: string, name: ToolCallOrderStep["name"]): boolean {
  if (typeof name === "string") {
    return callName === name;
  }
  return name.includes(callName);
}

function matchesNameArgs(call: ToolCall, step: ToolCallOrderStep): boolean {
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

function matchesStep(call: ToolCall, step: ToolCallOrderStep): boolean {
  if (!matchesNameArgs(call, step)) return false;
  if (step.exitCode === undefined) return true;
  if (call.exitCode === undefined) return false;
  return exitCodeMatches(call.exitCode, step.exitCode);
}

function argString(v: string | number | boolean | null | undefined): string {
  if (v === null || v === undefined) return "";
  return String(v);
}
