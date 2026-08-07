import {
  isRegex,
  matchContains,
  matchDisplay,
  matchEquals,
  type StringOrRegexp,
} from "../stringMatch.js";
import { fail } from "./error.js";

export class FinalMessageMatchers {
  constructor(private readonly message: string) {}

  toContain(expected: string): this {
    if (!matchContains(this.message, expected)) {
      fail(
        "finalMessage.contains",
        `finalMessage does not contain ${JSON.stringify(expected)}`,
      );
    }
    return this;
  }

  /** RegExp substring match (same as YAML contains: /pattern/). */
  toMatch(expected: RegExp): this {
    if (!matchContains(this.message, expected)) {
      fail(
        "finalMessage.contains",
        `finalMessage does not match ${matchDisplay(expected)}`,
      );
    }
    return this;
  }

  toEqual(expected: StringOrRegexp): this {
    if (!matchEquals(this.message, expected)) {
      const reason = isRegex(expected)
        ? `finalMessage does not match ${matchDisplay(expected)}`
        : "finalMessage does not equal expected value";
      fail("finalMessage.equals", reason);
    }
    return this;
  }
}
