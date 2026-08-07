---
name: refactor-helper
description: Refactors small Go packages for clarity — simplify messy functions, extract helpers, and remove dead files.
---

# Refactor Helper

When asked to refactor Go code in this workspace:

1. **Simplify** unclear functions in place (clearer control flow, fewer temps).
2. **Extract** shared helpers into a new file when that improves clarity.
3. **Delete** obsolete/dead files the user identifies (or that are clearly unused after the refactor).

Prefer small, idiomatic Go. Do not leave unused legacy files behind when the prompt asks to remove them.
