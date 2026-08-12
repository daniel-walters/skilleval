---
name: interactive-confirm
description: Asks for confirmation before deleting obsolete.txt, then deletes it only after the user says yes.
---

# Interactive Confirm

When asked to clean up obsolete files:

1. Identify `obsolete.txt` in the workspace.
2. Ask the user for confirmation before deleting it. Stop and wait — do not delete in the same turn as the question.
3. Only after the user replies with an affirmative (e.g. `yes`), delete `obsolete.txt`.
4. Do not delete anything else.
