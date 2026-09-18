<p align="center">
  <img src="assets/reaper-logo.png" alt="Reaper logo" width="280">
</p>

<h1 align="center">Reaper</h1>

<p align="center"><strong>Semantic lint for coding agents.</strong></p>

Reaper catches engineering problems that normal linters cannot understand, such
as comments that merely narrate code, silent error fallbacks, weakened tests,
unnecessary abstractions, and changes unrelated to the task.

```text
client.go:84-91: warning [redundant-comment] confidence=0.96
  Comment appears to merely narrate the adjacent implementation.
```

Reaper checks Git changes and returns deterministic diagnostics that developers,
CI jobs, and coding agents can act on immediately.

## Install

Reaper requires Go 1.25 or newer:

```sh
go install github.com/Eliran-Turgeman/repear/cmd/reaper@latest
```

Or build it from source:

```sh
git clone https://github.com/Eliran-Turgeman/repear
cd repear
go build -o reaper ./cmd/reaper
```

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

Give Reaper the task so it can also detect unrelated changes:

```sh
reaper check --task "Do not retry authentication failures"
```

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
| `defensive-fallback` | Fallbacks that hide errors or invalid states |
| `weakened-test` | Test changes that reduce protection |
| `scope-creep` | Substantial changes unrelated to the supplied task |

Run `reaper rules` to see the configured defaults.

## Evaluate the rules

The repository includes 100 labeled examples for measuring rule quality:

```sh
reaper eval
reaper eval --rule redundant-comment
reaper eval --format json
```

Reaper is not a formatter, code generator, or general-purpose reviewer. It sits
between deterministic static analysis and full LLM code review: narrow,
repeatable semantic checks designed for the coding loop.
