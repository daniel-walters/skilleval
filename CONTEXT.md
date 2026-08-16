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

**Excludes** (order):
An order step that must not match any tool call. Each excluded step is checked on its own — not as an ordered subsequence. Same polarity word as set and file-content excludes; the subject is a tool-call pattern.

**String match**:
Literal text, or a slash-delimited regex, used by contains / equals / file-content excludes.

**Order step**:
One check in a tool-call order expect. A call matches the step when name, args, and exit code all satisfy it. Filters that are not set are ignored.
_Avoid_: alternation, sequence step

**Names** (on an order step):
The exact tool names that can satisfy an order step. The field is `name`. It holds one name or a list of names. A tool call matches if its tool name equals one of these names.
_Avoid_: alternation, alias, pattern, or-match

**Exit code**:
The integer process status of a `shell` or `Bash` tool invocation. Omitted when the runner cannot observe it. Distinct from tool-call status (whether the invocation completed or errored).
_Avoid_: status, success, failure, CLI exitCode

**Invalid expect**:
An expect that is not well-formed. Examples: an order step with no names, a name with no text, an empty exit-code list, a non-integer exit code, or an exit-code filter on a name other than `shell` or `Bash`.
_Avoid_: failed check

**Failed check**:
A valid expect that the result does not satisfy.
_Avoid_: invalid expect, error

**Attempt**:
One scheduled try of an eval. A batch may run many attempts; each produces its own Result or a runner error.
_Avoid_: retry

**Pass rate**:
The fraction of attempts whose expects passed (`passed / attempts`). A batch may require a minimum pass rate.
_Avoid_: success rate, accuracy
