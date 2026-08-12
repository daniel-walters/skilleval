/**
 * Aggregate observables across multi-turn agent legs (scripted replies).
 */

/** @returns {{ inputTokens: number, outputTokens: number, cacheReadTokens: number, cacheWriteTokens: number, totalTokens: number }} */
export function emptyUsage() {
  return {
    inputTokens: 0,
    outputTokens: 0,
    cacheReadTokens: 0,
    cacheWriteTokens: 0,
    totalTokens: 0,
  };
}

/**
 * @param {{ inputTokens?: number, outputTokens?: number, cacheReadTokens?: number, cacheWriteTokens?: number, totalTokens?: number } | null | undefined} a
 * @param {{ inputTokens?: number, outputTokens?: number, cacheReadTokens?: number, cacheWriteTokens?: number, totalTokens?: number } | null | undefined} b
 */
export function addUsage(a, b) {
  const left = a ?? emptyUsage();
  const right = b ?? emptyUsage();
  return {
    inputTokens: (left.inputTokens ?? 0) + (right.inputTokens ?? 0),
    outputTokens: (left.outputTokens ?? 0) + (right.outputTokens ?? 0),
    cacheReadTokens: (left.cacheReadTokens ?? 0) + (right.cacheReadTokens ?? 0),
    cacheWriteTokens: (left.cacheWriteTokens ?? 0) + (right.cacheWriteTokens ?? 0),
    totalTokens: (left.totalTokens ?? 0) + (right.totalTokens ?? 0),
  };
}

/**
 * @param {number | null | undefined} a
 * @param {number | null | undefined} b
 * @returns {number | null}
 */
export function addCost(a, b) {
  const hasA = typeof a === "number";
  const hasB = typeof b === "number";
  if (!hasA && !hasB) return null;
  return (hasA ? a : 0) + (hasB ? b : 0);
}

/**
 * Ordered user messages for one attempt: initial prompt plus scripted replies.
 * @param {string} prompt
 * @param {string[]} [replies]
 * @returns {string[]}
 */
export function userMessages(prompt, replies) {
  const out = [prompt];
  if (Array.isArray(replies)) {
    for (const r of replies) {
      if (typeof r === "string" && r) out.push(r);
    }
  }
  return out;
}
