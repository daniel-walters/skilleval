/** Eval document fields shared with YAML (without expects — TS asserts via expect). */
export interface EvalDocument {
  schemaVersion: number;
  name: string;
  prompt: string;
  /**
   * Optional ordered follow-up user messages after the initial prompt
   * finishes each agent leg (interactive skill testing).
   */
  replies?: string[];
  /** Path to a skill directory containing SKILL.md. */
  skill: string;
  input?: string;
  mcp?: string;
  attempts?: number;
  /** Batch pass-rate gate (same shape as YAML passRate.min). */
  passRate?: PassRateExpect;
  /**
   * Absolute path to the source YAML when loaded via loadEval.
   * When set, run uses this file instead of writing a temp eval.
   */
  sourcePath?: string;
}

/** Batch-level minimum pass rate across attempts (0–1). */
export interface PassRateExpect {
  min: number;
}

/** Options for programmatic run (writes a temp eval YAML without expects). */
export interface RunOptions {
  name: string;
  prompt: string;
  /** Optional ordered follow-up user messages (see EvalDocument.replies). */
  replies?: string[];
  skill: string;
  input?: string;
  mcp?: string;
  attempts?: number;
  passRate?: PassRateExpect;
  /** Required model id for the agent runner. */
  model: string;
  /** Agent runtime; default cursor. */
  runner?: "cursor" | "claude" | string;
  out?: string;
  /** Override history directory (CLI default: .skilleval/history). */
  history?: string;
  /** Skip retaining summary history. */
  noHistory?: boolean;
  /** Override baseline summary path (CLI default: prior latest.json when present). */
  baseline?: string;
  /** Skip baseline comparison. */
  noBaseline?: boolean;
  /** Per-attempt agent timeout (Go duration, e.g. "30m"); forwarded as --timeout. */
  timeout?: string;
}

/** Overrides when running a loaded eval. */
export interface RunOverrides {
  model: string;
  runner?: "cursor" | "claude" | string;
  out?: string;
  /** Override history directory (CLI default: .skilleval/history). */
  history?: string;
  /** Skip retaining summary history. */
  noHistory?: boolean;
  /** Override baseline summary path (CLI default: prior latest.json when present). */
  baseline?: string;
  /** Skip baseline comparison. */
  noBaseline?: boolean;
  /** Per-attempt agent timeout (Go duration, e.g. "30m"); forwarded as --timeout. */
  timeout?: string;
}
