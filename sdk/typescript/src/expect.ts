import type { Result } from "./result.js";
import { fail } from "./matchers/error.js";
import { FileMatchers } from "./matchers/file.js";
import { FinalMessageMatchers } from "./matchers/finalMessage.js";
import { NumericFloatMatchers, NumericIntMatchers } from "./matchers/numeric.js";
import { SkillsActivatedMatchers, ToolsUsedMatchers } from "./matchers/sets.js";

export { ExpectError } from "./matchers/error.js";

export interface Expectation {
  readonly turns: NumericIntMatchers;
  readonly costUSD: NumericFloatMatchers;
  readonly toolsUsed: ToolsUsedMatchers;
  readonly skills: { readonly activated: SkillsActivatedMatchers };
  readonly finalMessage: FinalMessageMatchers;
  file(relPath: string): FileMatchers;
}

/**
 * Typed expect matchers over a Result. Eager-throws ExpectError with
 * checker-style path + reason. Non-finished results fail as run.status
 * before any other check.
 *
 * @param workspace Attempt workspace path; required for file content matchers.
 */
export function expect(result: Result, workspace?: string): Expectation {
  return new ExpectationImpl(result, workspace);
}

class ExpectationImpl implements Expectation {
  constructor(
    private readonly result: Result,
    private readonly workspace: string | undefined,
  ) {}

  private ensureFinished(): void {
    if (this.result.status !== "finished") {
      fail(
        "run.status",
        `status is ${JSON.stringify(this.result.status)}, want "finished"`,
      );
    }
  }

  get turns(): NumericIntMatchers {
    this.ensureFinished();
    return new NumericIntMatchers("turns", this.result.metrics.turns);
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
