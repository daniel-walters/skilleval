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

/**
 * Whether Cursor run.conversation() can stand in for the whole attempt.
 * Multi-leg replies: each Run is one prompt, so last-run conversation would
 * truncate turns/log — keep stream aggregates instead.
 * @param {number} messageCount
 * @returns {boolean}
 */
export function preferConversationTranscript(messageCount) {
  return messageCount <= 1;
}

/**
 * Build a follow-up send() prompt that includes prior user/assistant text.
 * Workaround for @cursor/sdk local agents not retaining conversation across
 * send() calls (cloud agents retain natively).
 *
 * @param {string} reply - the next user message (logged as-is; this return is only for send)
 * @param {object[]} priorEvents - normalized log events from earlier legs
 * @returns {string}
 */
export function composeLocalFollowUpPrompt(reply, priorEvents) {
  const lines = [];
  for (const e of priorEvents ?? []) {
    if (!e || typeof e !== "object") continue;
    if (e.type === "user" && typeof e.text === "string" && e.text) {
      lines.push(`User: ${e.text}`);
    } else if (e.type === "assistant" && typeof e.text === "string" && e.text) {
      lines.push(`Assistant: ${e.text}`);
    } else if (e.type === "tool_call" && typeof e.name === "string" && e.name) {
      const argBits = [];
      if (e.args && typeof e.args === "object") {
        for (const [k, v] of Object.entries(e.args)) {
          if (typeof v === "string" || typeof v === "number" || typeof v === "boolean") {
            argBits.push(`${k}=${v}`);
          }
        }
      }
      lines.push(
        argBits.length > 0
          ? `Tool: ${e.name} ${argBits.join(" ")}`
          : `Tool: ${e.name}`,
      );
    }
  }
  const transcript =
    lines.length > 0
      ? lines.join("\n")
      : "(no prior transcript)";
  return [
    "Continue the same conversation. Prior turns:",
    transcript,
    "",
    "Next user message:",
    reply,
  ].join("\n");
}
