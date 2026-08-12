/**
 * Normalized agent log events for the result sidecar artefact.
 * Shape: { schemaVersion: 1, events: [...] }
 */

import { normalizeToolArgs } from "./toolargs.mjs";

/**
 * @param {unknown[]} events
 * @returns {{ schemaVersion: number, events: unknown[] }}
 */
export function makeLog(events) {
  return { schemaVersion: 1, events: Array.isArray(events) ? events : [] };
}

/**
 * @param {string} text
 * @returns {{ type: string, text: string }}
 */
export function userEvent(text) {
  return { type: "user", text: typeof text === "string" ? text : String(text ?? "") };
}

/**
 * @param {string} text
 * @returns {{ type: string, text: string }}
 */
export function errorEvent(text) {
  return { type: "error", text: typeof text === "string" ? text : String(text ?? "") };
}

/**
 * Extract assistant text from a Cursor stream assistant event.
 * @param {unknown} event
 * @returns {string}
 */
export function assistantTextFromStreamEvent(event) {
  const content = event?.message?.content;
  if (typeof event?.text === "string" && event.text) {
    return event.text;
  }
  if (!Array.isArray(content)) {
    return "";
  }
  return content
    .filter((b) => b?.type === "text" && typeof b.text === "string")
    .map((b) => b.text)
    .join("");
}

/**
 * Append a Cursor stream event to the log (fallback when conversation is unavailable).
 * Coalesces tool_call start/complete by call_id when present.
 *
 * @param {object[]} events
 * @param {object} event
 * @param {string} [cwd]
 */
export function appendStreamEvent(events, event, cwd) {
  if (!event || typeof event !== "object") return;

  if (event.type === "assistant") {
    const text = assistantTextFromStreamEvent(event);
    if (text) {
      events.push({ type: "assistant", text });
    }
    return;
  }

  if (event.type !== "tool_call") return;

  const name = event.name ?? "unknown";
  const status = event.status ?? "completed";
  const callId = event.call_id;
  const args = normalizeToolArgs(event.args, cwd);
  const entry = { type: "tool_call", name, status };
  if (args) entry.args = args;

  if (callId) {
    for (let i = events.length - 1; i >= 0; i--) {
      const e = events[i];
      if (e?.type === "tool_call" && e.callId === callId) {
        events[i] = {
          ...e,
          name,
          status,
          args: args ?? e.args,
        };
        return;
      }
    }
    entry.callId = callId;
  } else if (status !== "running") {
    for (let i = events.length - 1; i >= 0; i--) {
      const e = events[i];
      if (e?.type === "tool_call" && e.name === name && e.status === "running") {
        events[i] = {
          ...e,
          status,
          args: args ?? e.args,
        };
        return;
      }
    }
  }

  events.push(entry);
}

/**
 * Convert Cursor run.conversation() turns into normalized log events.
 * Returns null when conversation has no usable agent steps (caller should keep stream events).
 *
 * @param {unknown} conversation
 * @param {string} prompt
 * @param {string} [cwd]
 * @returns {object[] | null}
 */
export function eventsFromConversation(conversation, prompt, cwd) {
  if (!Array.isArray(conversation) || conversation.length === 0) {
    return null;
  }

  const events = [userEvent(prompt)];
  let sawStep = false;

  for (const turn of conversation) {
    if (turn?.type !== "agentConversationTurn") continue;
    const agentTurn = turn.turn ?? {};
    const userText = agentTurn.userMessage?.text;
    if (typeof userText === "string" && userText && events.length === 1) {
      // Prefer conversation's user message when it differs; keep prompt as first.
      if (userText !== prompt) {
        events[0] = userEvent(userText);
      }
    }
    for (const step of agentTurn.steps ?? []) {
      if (step?.type === "assistantMessage") {
        const text = step.message?.text;
        if (typeof text === "string" && text) {
          events.push({ type: "assistant", text });
          sawStep = true;
        }
      } else if (step?.type === "toolCall") {
        const msg = step.message ?? {};
        const name = msg.name ?? "unknown";
        const status = msg.status ?? "completed";
        const entry = { type: "tool_call", name, status };
        const args = normalizeToolArgs(msg.args ?? msg.input, cwd);
        if (args) entry.args = args;
        events.push(entry);
        sawStep = true;
      }
    }
  }

  return sawStep ? events : null;
}

/**
 * Append Claude Agent SDK message content to the log.
 *
 * @param {object[]} events
 * @param {object} message
 * @param {string} [cwd]
 */
export function appendClaudeMessage(events, message, cwd) {
  if (!message || message.type !== "assistant" || !Array.isArray(message.message?.content)) {
    return;
  }
  for (const block of message.message.content) {
    if (!block || typeof block !== "object") continue;
    if (block.type === "text" && typeof block.text === "string" && block.text) {
      events.push({ type: "assistant", text: block.text });
      continue;
    }
    if (block.type === "tool_use") {
      const name = block.name ?? "unknown";
      const entry = { type: "tool_call", name, status: "completed" };
      const args = normalizeToolArgs(block.input, cwd);
      if (args) entry.args = args;
      events.push(entry);
    }
  }
}

/**
 * Drop internal callId fields before emitting the sidecar log.
 * @param {object[]} events
 * @returns {object[]}
 */
export function finalizeLogEvents(events) {
  return (events ?? []).map((e) => {
    if (!e || e.type !== "tool_call") return e;
    const { callId, ...rest } = e;
    return rest;
  });
}
