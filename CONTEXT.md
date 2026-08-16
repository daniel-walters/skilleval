# skilleval

Deterministic evals for agent skills: run a skill, record observables, check them against expects — no LLM judge.

## Language

**Expect**:
A declarative check against a run’s observables. YAML and TypeScript express the same contract.
_Avoid_: assertion (prefer for TypeScript matcher API only), judge

**Contains** (file / text):
A single string match that must appear in the text (literal substring or `/regex/`).
_Avoid_: includes (that word is for set membership)

**Excludes** (file content):
A list of string matches that must all be absent from a touched file’s body (literal or `/regex/`).
_Avoid_: notContains (YAML key); use TypeScript `.not.toContain` only as the API spelling

**Excludes** (set):
Forbidden members of a set observable (`toolsUsed`, `skills.activated`). Same polarity word as file-content excludes; different subject (names vs text).

**String match**:
Literal text, or a slash-delimited regex, used by contains / equals / file-content excludes.
