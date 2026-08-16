/**
 * Process exit codes for shell / Bash tool calls.
 * Only a finite integer from known fields is kept — never parsed from stdout.
 */

export const SHELL_TOOL_NAME = "shell";
export const CLAUDE_BASH_TOOL_NAME = "Bash";

export function isShellToolName(name) {
  return name === SHELL_TOOL_NAME || name === CLAUDE_BASH_TOOL_NAME;
}

/**
 * @param {string} name Tool name
 * @param {object} [event] SDK tool_call / tool_result-like object
 * @returns {number | undefined}
 */
export function exitCodeFromToolEvent(name, event) {
  if (!isShellToolName(name) || !event || typeof event !== "object") {
    return undefined;
  }
  const nested =
    event.result && typeof event.result === "object" && !Array.isArray(event.result)
      ? event.result
      : undefined;
  const content =
    event.content && typeof event.content === "object" && !Array.isArray(event.content)
      ? event.content
      : undefined;
  const candidates = [
    event.exitCode,
    event.exit_code,
    nested?.exitCode,
    nested?.exit_code,
    nested?.exit,
    content?.exitCode,
    content?.exit_code,
    content?.exit,
  ];
  for (const v of candidates) {
    const n = asFiniteInt(v);
    if (n !== undefined) return n;
  }
  return undefined;
}

function asFiniteInt(v) {
  if (typeof v === "number" && Number.isInteger(v) && Number.isFinite(v)) {
    return v;
  }
  return undefined;
}
