/** Result status: agent completion state (not expectation pass/fail). */
export type Status = "finished" | "error" | "cancelled";

/** How a path changed in the attempt workspace. */
export type FileStatus = "created" | "modified" | "deleted";

export type ToolCallStatus = "running" | "completed" | "error";

/** One attempt of one eval (schemaVersion 1). */
export interface Result {
  schemaVersion: number;
  id: string;
  startedAt: string;
  finishedAt: string;
  eval: EvalInfo;
  status: Status;
  metrics: Metrics;
  skills: Skills;
  outcomes: Outcomes;
  error: string | null;
  finalMessage: string;
}

export interface EvalInfo {
  name: string;
  prompt: string;
  skill: string;
  model: string;
  runner: string;
  attempt: number;
  totalAttempts?: number;
}

export interface Metrics {
  turns: number;
  durationMs: number;
  toolsUsed: string[];
  toolCalls: ToolCall[];
  usage: Usage;
  costUSD: number | null;
}

export interface ToolCall {
  name: string;
  status: ToolCallStatus;
}

export interface Usage {
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  totalTokens: number;
}

export interface Skills {
  activated: string[];
}

export interface Outcomes {
  files: Record<string, FileOutcome>;
}

export interface FileOutcome {
  status: FileStatus;
}

/** Batch summary written beside --out (optional on run return). */
export interface Summary {
  attempts: number;
  passed: number;
  passRate: number;
  avgTurns?: number;
  avgCostUSD?: number;
  avgDurationMs?: number;
}
