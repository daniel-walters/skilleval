/** Eval document fields shared with YAML (without expects — TS asserts via expect). */
export interface EvalDocument {
  schemaVersion: number;
  name: string;
  prompt: string;
  /** Path to a skill directory containing SKILL.md. */
  skill: string;
  input?: string;
  mcp?: string;
  attempts?: number;
  /**
   * Absolute path to the source YAML when loaded via loadEval.
   * When set, run uses this file instead of writing a temp eval.
   */
  sourcePath?: string;
}

/** Options for programmatic run (writes a temp eval YAML without expects). */
export interface RunOptions {
  name: string;
  prompt: string;
  skill: string;
  input?: string;
  mcp?: string;
  attempts?: number;
  /** Required model id for the agent runner. */
  model: string;
  /** Agent runtime; default cursor. */
  runner?: "cursor" | "claude" | string;
  out?: string;
  history?: string;
  baseline?: string;
}

/** Overrides when running a loaded eval. */
export interface RunOverrides {
  model: string;
  runner?: "cursor" | "claude" | string;
  out?: string;
  history?: string;
  baseline?: string;
}
