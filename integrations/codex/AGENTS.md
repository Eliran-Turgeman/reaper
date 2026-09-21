# Reaper for Codex (integration version 1)

After code changes, run the repository's normal tests and static checks, then
`reaper check --format agent --task "a short summary of the user's request"`.
Fix errors and applicable warnings; rerun tests and Reaper until clean.
Investigate incomplete analysis (exit 2 or `complete: false`). Do not suppress
rules to make checks pass, broaden task context, or persist API credentials.
