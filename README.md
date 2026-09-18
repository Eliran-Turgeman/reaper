<p align="center">
  <img src="assets/reaper-logo.png" alt="Reaper logo" width="280">
</p>

<h1 align="center">Reaper</h1>

<p align="center"><strong>Semantic lint for coding agents.</strong></p>

Reaper catches engineering problems that normal linters cannot understand, such
as comments that merely narrate code, shallow or pass-through abstractions,
leaked implementation details, silent error fallbacks, weakened tests, and
changes unrelated to the task.

```text
client.go:84-91: warning [redundant-comment] confidence=0.96
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
reaper check --format json     # machine-readable output
reaper check --verbose         # show evaluation details
reaper check --no-cache        # force fresh evaluations
```

Errors exit with code `1`, tool failures exit with code `2`, and warnings do not
block by default.

## Rules

| Rule | Detects |
|---|---|
| `redundant-comment` | Comments that merely repeat adjacent code |
| `speculative-generality` | Abstraction or flexibility without a real need |
| `pass-through-layer` | Layers that add indirection without a new abstraction |
| `complexity-pushed-upward` | Complexity imposed on callers that a module could hide |
| `information-leakage` | Internal design knowledge exposed across module boundaries |
| `special-general-mixture` | Special-purpose policy embedded in general mechanisms |
| `shallow-module` | Abstractions with costly interfaces and little hidden complexity |
| `defensive-fallback` | Fallbacks that hide errors or invalid states |
| `weakened-test` | Test changes that reduce protection |
| `scope-creep` | Substantial changes unrelated to the supplied task |

Run `reaper rules` to see the configured defaults.

## Evaluate the rules

The repository includes 200 labeled examples for measuring rule quality:

```sh
reaper eval
reaper eval --rule redundant-comment
reaper eval --format json
```

Reaper is not a formatter, code generator, or general-purpose reviewer. It sits
between deterministic static analysis and full LLM code review: narrow,
repeatable semantic checks designed for the coding loop.
