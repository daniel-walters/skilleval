/**
 * Count model rounds for skilleval metrics.turns.
 *
 * Prefers assistantMessage steps from the Cursor conversation transcript
 * (one step ≈ one model/tool round). Falls back to stream-derived assistant
 * event count when conversation is unavailable or has no assistant steps.
 *
 * @param {number} streamTurns
 * @param {unknown} conversation
 * @returns {number}
 */
export function countTurns(streamTurns, conversation) {
  const steps = (conversation ?? [])
    .filter((t) => t?.type === "agentConversationTurn")
    .flatMap((t) => t.turn?.steps ?? []);
  const fromConversation = steps.filter(
    (s) => s?.type === "assistantMessage",
  ).length;
  if (fromConversation > 0) return fromConversation;
  return streamTurns;
}
