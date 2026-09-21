---
name: reaper-check
description: Run Reaper semantic checks after modifying code and passing repository tests.
---

Integration version 1. Run the repository's tests and static checks first.
Run `reaper check --format agent --task "a short summary of the user's request"`.
Fix blocking findings and applicable warnings, then rerun tests and Reaper.
Investigate exit 2 or `complete: false`; do not claim a pass for incomplete work.
Do not weaken rules or expand the task to justify unrelated changes. Keep API
credentials in environment variables.
