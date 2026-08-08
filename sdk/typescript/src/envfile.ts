import fs from "node:fs";

/**
 * Load KEY=VALUE pairs from path into process.env.
 * Missing file is a no-op. Already-set process environment variables win
 * (same semantics as the Go CLI).
 */
export function loadDotEnv(filePath: string = ".env"): void {
  let raw: string;
  try {
    raw = fs.readFileSync(filePath, "utf8");
  } catch (err) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code === "ENOENT") {
      return;
    }
    throw new Error(`envfile: ${filePath}: ${(err as Error).message}`);
  }

  for (const line of raw.split(/\r?\n/)) {
    const parsed = parseEnvLine(line);
    if (!parsed) {
      continue;
    }
    if (process.env[parsed.key] !== undefined) {
      continue;
    }
    process.env[parsed.key] = parsed.value;
  }
}

/** Parse a single .env line. Returns null for blank lines and comments. */
export function parseEnvLine(line: string): { key: string; value: string } | null {
  const trimmed = line.trim();
  if (!trimmed || trimmed.startsWith("#")) {
    return null;
  }

  let body = trimmed;
  if (body.startsWith("export ")) {
    body = body.slice("export ".length).trim();
  }

  const eq = body.indexOf("=");
  if (eq <= 0) {
    throw new Error(`envfile: invalid line: ${line}`);
  }

  const key = body.slice(0, eq).trim();
  if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) {
    throw new Error(`envfile: invalid key: ${key}`);
  }

  let value = body.slice(eq + 1).trim();
  if (
    (value.startsWith('"') && value.endsWith('"') && value.length >= 2) ||
    (value.startsWith("'") && value.endsWith("'") && value.length >= 2)
  ) {
    value = value.slice(1, -1);
  } else if (
    (value.startsWith('"') && !value.endsWith('"')) ||
    (value.startsWith("'") && !value.endsWith("'"))
  ) {
    throw new Error(`envfile: unterminated quote in ${key}`);
  }

  return { key, value };
}
