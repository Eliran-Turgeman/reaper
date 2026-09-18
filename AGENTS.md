# Coding-agent workflow

After making meaningful code changes:

1. Run the repository's normal formatter, compiler, linters, and tests.
2. Set `REAPER_TASK` to a short, accurate summary of the user's requested
   change. Do not include unrelated cleanup or changes you decided to make.
3. Run `reaper check --task "$REAPER_TASK"`.
4. Treat Reaper errors as issues that must be investigated.
5. Inspect warnings and fix them when applicable.
6. After changing code in response to diagnostics, run Reaper again.
7. Do not disable, suppress, or weaken a rule merely to make the check pass
   unless that rule is genuinely inappropriate for the repository.

`REAPER_TASK` is task context, not a rule name or custom check. Reaper always
runs every configured rule that applies. The `scope-creep` rule additionally
uses this task description to detect substantial changes unrelated to what the
user requested. If `REAPER_TASK` is empty, `scope-creep` is skipped.

For machine-readable integration, use `reaper check --format json` and consume
the versioned diagnostic envelope. Exit code 0 means no blocking errors, 1 means
blocking semantic violations, and 2 means Reaper could not complete.
