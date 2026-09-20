# Agent integrations, version 1

Install Reaper and configure a provider credential before using any integration.
These are instruction files, not automatic shell hooks; the agent runs the
repository's tests before invoking Reaper. Review and merge instructions with
existing project files rather than overwriting them.

## Generic agents

Merge `AGENTS.md` into your repository's root `AGENTS.md`. Set `REAPER_TASK` from
the user's request. In PowerShell use `$env:REAPER_TASK` instead of `$REAPER_TASK`.

## Codex

Merge `codex/AGENTS.md` into the project's root `AGENTS.md`. Codex discovers
project instructions there; see the [official AGENTS.md guide](https://developers.openai.com/codex/guides/agents-md).

## Claude Code

Copy `claude/SKILL.md` to `.claude/skills/reaper-check/SKILL.md`. Invoke
`/reaper-check` after edits, or let the agent select it by description.
See [Claude Code skills](https://code.claude.com/docs/en/skills).

## Cursor

Copy `cursor/reaper.mdc` to `.cursor/rules/reaper.mdc`. It uses the always-apply
project-rule format; see [Cursor rules](https://cursor.com/docs/rules).
