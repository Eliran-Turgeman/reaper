# Coding-agent workflow

After making meaningful code changes:

1. Run the repository's normal formatter, compiler, linters, and tests.
2. Run `reaper check --task "$REAPER_TASK"`.
3. Treat Reaper errors as issues that must be investigated.
4. Inspect warnings and fix them when applicable.
5. After changing code in response to diagnostics, run Reaper again.
6. Do not disable, suppress, or weaken a rule merely to make the check pass
   unless that rule is genuinely inappropriate for the repository.

For machine-readable integration, use `reaper check --format json` and consume
the versioned diagnostic envelope. Exit code 0 means no blocking errors, 1 means
blocking semantic violations, and 2 means Reaper could not complete.
