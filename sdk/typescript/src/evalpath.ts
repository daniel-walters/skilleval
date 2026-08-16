import path from "node:path";

/**
 * Set by the npm bin when it executes a script eval. Not a user-facing flag.
 * Lets `run()` resolve skill/input/mcp from the eval file without chdir.
 */
export const SCRIPT_EVAL_DIR_ENV = "SKILLEVAL_EVAL_DIR";

/** Directory relative skill/input/mcp paths resolve from (eval file, else cwd). */
export function scriptEvalDir(): string {
  const dir = process.env[SCRIPT_EVAL_DIR_ENV]?.trim();
  return dir ? path.resolve(dir) : process.cwd();
}

/** Join rel to the script eval directory when rel is not absolute (Go ResolvePath). */
export function resolveEvalRel(rel: string, evalDir = scriptEvalDir()): string {
  if (!rel || path.isAbsolute(rel)) {
    return rel;
  }
  return path.resolve(evalDir, rel);
}
