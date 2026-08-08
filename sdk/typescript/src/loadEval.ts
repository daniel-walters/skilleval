import fs from "node:fs/promises";
import path from "node:path";

import { parse as parseYaml } from "yaml";

import type { EvalDocument } from "./evalTypes.js";

/**
 * Load an eval YAML from disk. Paths in the document stay relative to the
 * YAML file (same as the Go CLI).
 */
export async function loadEval(evalPath: string): Promise<EvalDocument> {
  const abs = path.resolve(evalPath);
  let raw: string;
  try {
    raw = await fs.readFile(abs, "utf8");
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    throw new Error(`loadEval: read ${abs}: ${msg}`);
  }

  let doc: unknown;
  try {
    doc = parseYaml(raw);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    throw new Error(`loadEval: decode ${abs}: ${msg}`);
  }

  if (doc === null || typeof doc !== "object" || Array.isArray(doc)) {
    throw new Error(`loadEval: ${abs}: expected a YAML mapping`);
  }

  const o = doc as Record<string, unknown>;
  if (o.schemaVersion !== 1) {
    throw new Error(`loadEval: ${abs}: unsupported schemaVersion ${String(o.schemaVersion)}`);
  }
  if (typeof o.name !== "string" || !o.name) {
    throw new Error(`loadEval: ${abs}: name is required`);
  }
  if (typeof o.prompt !== "string" || !o.prompt) {
    throw new Error(`loadEval: ${abs}: prompt is required`);
  }
  if (typeof o.skill !== "string" || !o.skill) {
    throw new Error(`loadEval: ${abs}: skill is required`);
  }

  const ev: EvalDocument = {
    schemaVersion: 1,
    name: o.name,
    prompt: o.prompt,
    skill: o.skill,
    sourcePath: abs,
  };
  if (typeof o.input === "string" && o.input) {
    ev.input = o.input;
  }
  if (typeof o.mcp === "string" && o.mcp) {
    ev.mcp = o.mcp;
  }
  if (typeof o.attempts === "number" && o.attempts > 0) {
    ev.attempts = o.attempts;
  }
  if (o.passRate !== undefined && o.passRate !== null) {
    if (typeof o.passRate !== "object" || Array.isArray(o.passRate)) {
      throw new Error(`loadEval: ${abs}: passRate must be a mapping with min`);
    }
    const pr = o.passRate as Record<string, unknown>;
    if (typeof pr.min !== "number" || Number.isNaN(pr.min)) {
      throw new Error(`loadEval: ${abs}: passRate.min is required`);
    }
    if (pr.min < 0 || pr.min > 1) {
      throw new Error(`loadEval: ${abs}: passRate.min must be between 0 and 1`);
    }
    ev.passRate = { min: pr.min };
  }
  return ev;
}
