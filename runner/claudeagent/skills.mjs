/**
 * Name of a skill activated via the Claude Skill tool_use input.
 * Prefer skill_name (Agent SDK); accept skill for older/alternate payloads.
 */
export function activatedSkillFromInput(input) {
  if (!input || typeof input !== "object") return null;
  const name = input.skill_name ?? input.skill;
  return typeof name === "string" && name ? name : null;
}
