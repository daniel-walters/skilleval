/**
 * Literal string or RegExp match, mirroring Go eval.StringMatch.
 * RegExp uses the same semantics as YAML /pattern/ (substring contains vs
 * full-string equals).
 */

export type StringOrRegexp = string | RegExp;

export function matchContains(haystack: string, expected: StringOrRegexp): boolean {
  if (typeof expected === "string") {
    return haystack.includes(expected);
  }
  return expected.test(haystack);
}

export function matchEquals(haystack: string, expected: StringOrRegexp): boolean {
  if (typeof expected === "string") {
    return haystack === expected;
  }
  const m = haystack.match(expected);
  if (!m || m.index === undefined) {
    return false;
  }
  // Full-string match (same as Go FindStringIndex spanning [0, len)).
  return m.index === 0 && m[0]!.length === haystack.length;
}

/** Display form for failure reasons (literal quoted; regex as /source/). */
export function matchDisplay(expected: StringOrRegexp): string {
  if (typeof expected === "string") {
    return JSON.stringify(expected);
  }
  return `/${expected.source}/`;
}

export function isRegex(expected: StringOrRegexp): boolean {
  return expected instanceof RegExp;
}
