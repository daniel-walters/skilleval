/**
 * Derive skills.activated from Cursor tool_call events.
 *
 * Cursor loads a skill by reading its SKILL.md (progressive disclosure).
 * A `read` of …/{.cursor|.agents}/skills/<name>/SKILL.md means that skill
 * was activated for the run.
 *
 * The SDK typically puts `args.path` on the `running` event and may omit
 * `args` on `completed` (replacing it with `result`). Record from either.
 */

const SKILL_MD_RE =
  /(?:^|[/\\])(?:\.cursor|\.agents)[/\\]skills[/\\]([^/\\]+)[/\\]SKILL\.md$/i;

/** @returns {string|null} skill folder name, or null if path is not a skill read */
export function activatedSkillFromReadPath(path) {
  if (typeof path !== "string" || !path) return null;
  const m = path.match(SKILL_MD_RE);
  return m ? m[1] : null;
}

/**
 * If event is a running or completed read of a skill SKILL.md, add its name.
 * @param {Set<string>} activated
 * @param {{ name?: string, status?: string, args?: { path?: string }, result?: { path?: string } }} event
 */
export function noteActivatedSkill(activated, event) {
  if (!event || event.name !== "read") return;
  if (event.status !== "running" && event.status !== "completed") return;
  const path = event.args?.path ?? event.result?.path;
  const skill = activatedSkillFromReadPath(path);
  if (skill) activated.add(skill);
}
