<p align="center">
  <img src="assets/reaper-logo.png" alt="Reaper logo" width="280">
</p>

<h1 align="center">Reaper</h1>

<p align="center"><strong>Semantic lint for coding agents.</strong></p>

Reaper catches concrete semantic problems that normal linters cannot understand,
such as comments that merely narrate code, unchanged-argument forwarders,
ceremonial abstractions, caller-managed mechanics, exposed implementation
details, silent error fallbacks, weakened tests, and changes unrelated to the
task.

```text
client.go:84-91: warning [narrating-comment] confidence=0.96
  Comment appears to merely narrate the adjacent implementation.
```

Reaper checks Git changes and returns deterministic diagnostics that developers,
CI jobs, and coding agents can act on immediately.

## Install

### Windows

Install the latest release from PowerShell:

```powershell
irm https://raw.githubusercontent.com/Eliran-Turgeman/repear/main/scripts/install.ps1 | iex
```

The installer adds Reaper to your user `PATH`. Open a new terminal, then run
`reaper version`.

### macOS and Linux

Install the latest release:

```sh
curl -fsSL https://raw.githubusercontent.com/Eliran-Turgeman/repear/main/scripts/install.sh | sh
```

The installer uses `/usr/local/bin` when it is writable, otherwise
`~/.local/bin`. It tells you if that directory needs to be added to `PATH`.

Both installers detect the operating system and CPU architecture, download the
matching GitHub release, and verify its SHA-256 checksum before installing it.
Set `REAPER_VERSION` to install a specific version or `REAPER_INSTALL_DIR` to
choose the destination.

### Install with Go

If you have Go 1.25 or newer:

```sh
go install github.com/Eliran-Turgeman/repear/cmd/reaper@latest
```

### Build from source

```sh
git clone https://github.com/Eliran-Turgeman/repear
cd repear
go build -o reaper ./cmd/reaper
```

Maintainers publish a release by pushing a semantic-version tag such as
`v0.1.0`. GitHub Actions tests the project and attaches binaries for Windows,
macOS, and Linux on x64 and ARM64, along with `checksums.txt`.

## Configure

Create `.reaper.yaml`:

```sh
reaper init
```

Reaper supports TypeSafe directly or OpenRouter:

```yaml
version: 1
provider: openrouter
model: typesafe/jev-1.13
```

Set the matching API key:

```sh
export OPENROUTER_API_KEY="..."
# or: export TYPESAFE_API_KEY="..."
```

PowerShell:

```powershell
$env:OPENROUTER_API_KEY = "..."
```

## Use

Check the current working-tree changes:

```sh
reaper check
```

Check staged changes:

```sh
reaper check --staged
```

Audit every supported Git-tracked source file in the current codebase:

```sh
reaper check --all
```

Full-codebase audits support Go, Python, TypeScript, JavaScript, C#, Java, Ruby,
Rust, C, C++, Kotlin, Swift, PHP, Scala, Dart, Elixir, Lua, and Objective-C.
They honor configured exclusions and optional path arguments, and skip
`weakened-test-assertion` and `scope-creep`, which require a before-and-after code change.
Reaper uses this source-language allowlist, so untracked files and non-source
files such as Markdown, text, JSON, and YAML are not included. Dependency and
build directories such as `vendor`, `node_modules`, and `dist` are excluded at
any directory depth, and minified source files such as `jquery.min.js` are
ignored. If another source file exceeds the provider's token limit, Reaper logs
the skipped file to stderr and continues checking the rest of the codebase.

Give Reaper a short, accurate summary of the requested change so it can detect
unrelated work:

```sh
export REAPER_TASK="Do not retry authentication failures"
reaper check --task "$REAPER_TASK"
```

PowerShell:

```powershell
$env:REAPER_TASK = "Do not retry authentication failures"
reaper check --task $env:REAPER_TASK
```

`REAPER_TASK` represents the user's request. It is not a rule name and does not
replace the configured rules: Reaper still runs every applicable rule. The
`scope-creep` rule uses the task description to identify substantial changes
that do not support the request. Keep it focused on what the user asked for,
rather than expanding it to justify additional changes. If no task is supplied,
only `scope-creep` is skipped.

Useful options:

```sh
reaper check --diff main       # compare with a Git reference
reaper check src/              # limit the check to a path
reaper check --all             # audit all tracked files
reaper check --format json     # machine-readable output
reaper check --verbose         # show evaluation details
reaper check --debug           # show raw confidence for every evaluation
reaper check --no-cache        # force fresh evaluations
```

Debug output is written to stderr and includes below-threshold checks, their
configured thresholds, and whether each confidence came from the provider or
cache. Normal text and JSON output remain unchanged.

Errors exit with code `1`, tool failures exit with code `2`, and warnings do not
block by default.

## Rules

| Rule | Detects |
|---|---|
| `narrating-comment` | Changed comments that merely repeat adjacent code |
| `unchanged-argument-forwarder` | New callables that only relay the same operation and arguments |
| `ceremonial-abstraction` | New abstractions with visible ceremony and little local behavior |
| `boolean-mode-parameter` | New boolean inputs that select different execution modes |
| `caller-managed-mechanics` | Low-level configuration, sequencing, cleanup, or retry duties imposed on callers |
| `implementation-detail-exposure` | Concrete implementation details added to a visible boundary |
| `named-special-case` | Named product or workflow exceptions inside reusable mechanisms |
| `unused-extensibility-point` | Extensibility machinery with no concrete use visible in the patch |
| `silent-failure-fallback` | Failures visibly replaced with success or default values |
| `weakened-test-assertion` | Existing test checks that are removed, skipped, or broadened |
| `scope-creep` | Substantial changes unrelated to the supplied task |

Most rules are intentionally hunk-local: they report only behavior visible in
the supplied change rather than making repository-wide architectural claims.
Rules with multiple required signals ask one narrow question per signal and use
the weakest signal confidence as the final result. A diagnostic therefore fires
only when every required local observation clears the configured threshold.
`scope-creep` remains patch-scoped because relatedness can only be judged
against the complete change and supplied task.

Run `reaper rules` to see the configured defaults.

## Evaluate the rules

The repository includes 220 labeled examples for measuring rule quality:

```sh
reaper eval
reaper eval --rule narrating-comment
reaper eval --format json
```

Reaper is not a formatter, code generator, or general-purpose reviewer. It sits
between deterministic static analysis and full LLM code review: narrow,
repeatable semantic checks designed for the coding loop.
