# Reaper coding loop (integration version 1)

After meaningful code changes, run the repository's formatter, compiler,
linters, and tests. Set REAPER_TASK to a concise, accurate description of the
user's requested change, then run `reaper check --format agent --task "$REAPER_TASK"`.
Fix blocking findings and applicable warnings, rerun tests, and rerun Reaper.
Exit 2 or `complete: false` means analysis did not finish; investigate before
claiming success. Do not disable rules or rewrite the task to justify unrelated
changes. Keep credentials in environment variables, never source files.
