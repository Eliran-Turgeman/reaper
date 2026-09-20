<p align="center">
  <img src="assets/reaper-logo.png" alt="Reaper logo" width="280">
</p>

<h1 align="center">Reaper</h1>

<p align="center"><strong>Semantic lint for coding agents.</strong></p>

Licensed under the [MIT License](LICENSE).

[![CI](https://github.com/Eliran-Turgeman/reaper/actions/workflows/ci.yml/badge.svg)](https://github.com/Eliran-Turgeman/reaper/actions/workflows/ci.yml)

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
irm https://raw.githubusercontent.com/Eliran-Turgeman/reaper/main/scripts/install.ps1 | iex
```

The installer adds Reaper to your user `PATH`. Open a new terminal, then run
`reaper version`.

### macOS and Linux

Install the latest release:

```sh
curl -fsSL https://raw.githubusercontent.com/Eliran-Turgeman/reaper/main/scripts/install.sh | sh
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
go install github.com/Eliran-Turgeman/reaper/cmd/reaper@latest
```

### Build from source

```sh
git clone https://github.com/Eliran-Turgeman/reaper
cd reaper
go build -o reaper ./cmd/reaper
```

Maintainers publish a release by pushing a semantic-version tag such as
`v0.1.0`. GitHub Actions tests the project and attaches binaries for Windows,
macOS, and Linux on x64 and ARM64, along with `checksums.txt`.

## Configure

### Security & Privacy

Reaper sends evaluated code, paths, diffs, before/after context, task text, and
rule predicates to the configured inference provider. API keys are read from
environment variables; only numeric scores are cached locally by default under
`.git/reaper-cache`. Review [the data-flow and privacy documentation](docs/privacy.md)
before using private repositories or enterprise code.

Create `.reaper.yaml`:

```sh
reaper init
```

Reaper supports TypeSafe directly or OpenRouter:

The rule engine uses a provider-independent decision API with normalized scores.
Jev remains the backend for both supported transports; this is an extension
boundary, not a claim of support for arbitrary models. Adapters expose batching,
structured-score support, privacy mode, and maximum context (null when unknown).
Provider-specific payloads and token-limit parsing stay in the adapter layer.

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

Staged checks read surrounding changed-file context from the Git index, so
unstaged edits cannot contradict the staged diff. Deleted files have no current
surrounding context. Failure to read a required staged blob stops extraction.

Audit every supported Git-tracked source file in the current codebase:

```sh
reaper check --all
```

Full-codebase audits support Go, Python, TypeScript, JavaScript, C#, Java, Ruby,
Rust, C, C++, Kotlin, Swift, PHP, Scala, Dart, Elixir, Lua, and Objective-C.
JavaScript includes CommonJS `.cjs` files used by the first-party Action.
They honor configured exclusions and optional path arguments, and skip
`weakened-test-assertion` and `scope-creep`, which require a before-and-after code change.
Reaper uses this source-language allowlist, so untracked files and non-source
files such as Markdown, text, JSON, and YAML are not included. Dependency and
build directories such as `vendor`, `node_modules`, and `dist` are excluded at
any directory depth, and minified source files such as `jquery.min.js` are
ignored. If another source file exceeds the provider's token limit, Reaper logs
the skipped file to stderr and continues checking the rest of the codebase.
Such runs are **incomplete**, never a successful analysis. Provider failures
also retain findings from completed units. JSON exposes `complete`, `status`,
`passed`, and `skipped_units` with file, lines, rules, and reason.
Set `incomplete_analysis: error` (the default, including CI), `warning`, or
`ignore` in `.reaper.yaml`. The default exits 2 for incomplete analysis;
warning/ignore allow exit 0 when there are no semantic errors, but still
report `complete: false`, `passed: false`, and the missing coverage. Explicit
exclusions, unsupported file types, and inapplicable rules are outside the
requested analysis scope and do not make a run incomplete.

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

Task precedence is explicit `--task` (including an empty value), `--task-file`,
`REAPER_TASK`, then opt-in `--task-from-pr`. The last reads `GITHUB_EVENT_PATH`
without an API request and uses only PR title and body, capped at 4,000 characters.
It removes HTML comments and blocks between `<!-- generated:start -->` and
`<!-- generated:end -->` (also accepting `reaper:generated:` markers). Other
generated prose cannot be reliably identified; use these markers or an explicit
task. PR comments are never used. Missing PR context stays empty. `--verbose`
prints the context source. The Action enables PR derivation automatically unless
its `task` input is supplied.

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

Verbose output and JSON summaries include evaluator requests, retries, reported
input/output tokens, usage completeness, rule-level cache hit rate, and duration.
Cost is null when unknown. Set `pricing.input_per_million_usd` and
`pricing.output_per_million_usd` to your contracted rates for an estimate from
reported tokens. Partial usage yields a partial estimate, not a billing total;
cached runs with no requests cost zero in this estimate. Rates are never fetched
or assumed. Provider capabilities appear in JSON.

Run `python scripts/benchmark-latency.py --binary ./reaper` to measure small (1),
medium (8), and large (32) generated file changes with three uncached samples.
Measured medians and workload limitations are in [latency.json](benchmarks/latency.json).
The recorded Windows run measured 0.57 s / 1.30 s / 3.32 s medians respectively;
network conditions, model service, and patch complexity affect these values.

Errors exit with code `1`, tool failures exit with code `2`, and warnings do not
block by default.

## Rules

### Incremental adoption

Save a complete run with `reaper check --format json > findings.json`, then run
`reaper baseline create --report findings.json --reason "Existing backlog"`.
Use `reaper check --baseline .reaper-baseline.json` to fail only on new findings.
Creation refuses incomplete reports and overwrites. Fingerprints include rule,
file, and normalized changed content, so line movement within a file survives;
file renames or semantic edits require reviewing the baseline again.

An inline `// reaper: ignore silent-failure-fallback -- compatibility contract`
(or `#`/`--` comment) suppresses that rule in the analyzed hunk containing the
directive or immediately following it. A known rule and nonempty reason are
required. Suppressed findings and reasons remain in JSON, with a summary count.
Suppressions never hide incomplete analysis.

### Local feedback

Use a diagnostic's `fingerprint` as its finding ID:
`reaper feedback <finding-id> useful --report findings.json` or
`reaper feedback <finding-id> false-positive --report findings.json`.
Feedback is appended to `.reaper-feedback.jsonl` (override with `--file`).
`reaper feedback metrics --report findings.json` reports per-rule findings and
suppressed counts from that report, plus accepted/false-positive counts across
the local feedback history. The latest decision per ID wins. Nothing is uploaded;
keep the file private and delete it to remove feedback.

### GitHub Action

The first-party composite Action builds a binary from the exact Action revision.
Pin the Action to a reviewed commit SHA for reproducibility. After this change is
published to `main`, the following workflow runs on pull requests:

```yaml
name: Reaper
on: pull_request
permissions:
  contents: read
jobs:
  reaper:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          persist-credentials: false
      - uses: Eliran-Turgeman/reaper/.github/actions/reaper@main
        with:
          provider: openrouter
          model: typesafe/jev-1.13
          api-key: ${{ secrets.OPENROUTER_API_KEY }}
```

For coding agents, use `reaper check --format agent --task "the user's request"`.
Ready-to-copy, versioned instructions for AGENTS.md, Codex, Claude Code, and
Cursor are in [integrations/](integrations/README.md).
It emits compact JSON with rule, severity, file, lines, signal evidence,
confidence, a remediation objective, and task context for task-dependent rules.
The `complete` and `skipped_units` fields prevent treating incomplete analysis
as a clean result. Follow this loop: modify code → run tests → run Reaper →
fix blocking findings and applicable warnings → rerun tests and Reaper.
Investigate exit 2 as an analysis/tool failure; do not weaken rules to pass.

Inputs also include `base`, newline-separated `paths`, `config`, and
`fail-on-warning`. The CLI equivalents are `--config` and `--fail-on-warning`.
Findings become line annotations; the JSON report is saved to
`$RUNNER_TEMP/reaper-report.json`. Fork PRs without secrets fail with a clear
message before inference. Use `pull_request`, with reviewed trusted execution
for forks; do not expose secrets using `pull_request_target` on untrusted code.
Action metadata follows [GitHub's composite Action format](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax).

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
| `removed-authorization-check` | Removed authorization guards without equivalent visible protection |
| `removed-validation` | Removed validation while its guarded operation remains |
| `swallowed-cancellation` | Newly discarded cancellation that continues work or reports success |

Before-and-after regression rules also inspect insertion-only changes to existing
files, such as a new early return that bypasses an unchanged authorization guard.
New files remain excluded from rules that require existing protection to compare.

Most rules are intentionally hunk-local: they report only behavior visible in
the supplied change rather than making repository-wide architectural claims.
Surrounding context now prefers the smallest enclosing declaration: Go uses its
native parser; Python uses conservative indentation scanning; JavaScript,
TypeScript, C#, and Java use conservative declaration/brace scanning with comments
and literals masked. These scanners are not complete parsers (multiline signatures,
JS regex literals, and unusual syntax may fall back). Malformed or unsupported
input retains the six-line hunk context. Changed before/after code and diagnostic
locations remain tied to the original hunk; only surrounding context expands.
Rules declare `local`, `patch`, or `repository-search` context. Only
`unused-extensibility-point` currently requests repository search. Reaper searches
up to four named introduced types in tracked, supported, nonexcluded files;
retrieved lines are capped at 40 hits / 12 KiB, skip symlinks and files over 1 MiB,
and are sent only to that rule. This lexical search is not exhaustive symbol
resolution. Findings include `context_evidence` and `--debug` prints retrieved
context even below the threshold; keep reports and debug logs
private because this field can contain additional source code.
Rules with multiple required signals ask one narrow question per signal and use
the weakest signal confidence as the final result. A diagnostic therefore fires
only when every required local observation clears the configured threshold.
JSON diagnostics expose `confidence` (renamed from `probability`) and a sorted
`signals` array containing each signal's ID, score, and evaluated predicate.
Text findings show the same score breakdown. Confidence is a minimum predicate
score, not a calibrated joint probability; the predicate is evidence of what
was evaluated, not a model-generated explanation or quotation from the code.
The JSON diagnostic envelope is now version 2 for this field rename; consumers
of version 1 must migrate to `confidence`. Agent output has its own version 1.
`scope-creep` remains patch-scoped because relatedness can only be judged
against the complete change and supplied task.

Run `reaper rules` to see the configured defaults.

New configurations enable only `regressions` (`silent-failure-fallback`,
`weakened-test-assertion`, `scope-creep`, `removed-authorization-check`,
`removed-validation`, `swallowed-cancellation`). The three new rules require
modified existing production code, skip audits/tests, and each includes 10
positive and 10 negative cases across five languages, including preserved guards
and explicit contract changes. Opt into other packs with
`packs: [regressions, agent-slop]`. `agent-slop` contains ceremonial abstractions,
unchanged forwarders, and unused extensibility; `style` contains narrating
comments and boolean mode parameters; `architecture` contains the other boundary
and caller-duty rules. Subjective rules default to warnings. Explicit per-rule
`enabled` settings override pack selection, preserving existing configurations.
Use `packs: []` to select no packs, or `reaper rules --pack regressions` to list
one pack. This repository explicitly enables all its existing rules.

## Evaluate the rules

The [patch benchmark](benchmarks/README.md) contains 32 realistically generated
service changes (24 clean, eight positive), all development cases. The measured
`openrouter / typesafe/jev-1.13` run in [results.json](benchmarks/patches/results.json)
had **0/8 detected positives and 0/24 false positives** at current defaults.
Recall was 0%; precision is undefined with no predicted positives (JSON uses 0).
This is a small pilot exposing a threshold limitation, not evidence of production
accuracy. Run `reaper eval --benchmark-dir benchmarks/patches --format json`.
Add `--benchmark-mode git` to exercise real Git extraction and configured rule
grouping, including eligibility misses and below-threshold signal scores. See
[Git benchmark modes and fixtures](benchmarks/README.md#evaluating-the-real-git-path).

Release publication additionally requires the absolute targets in
[`benchmarks/release-policy.json`](benchmarks/release-policy.json): at least 80%
recall, at most 1% false positives, and at least 20 positive/100 negative
independently reviewed cases for each blocking rule. The current synthetic
corpora do not meet these requirements, so the release workflow intentionally
stops at that gate. Ordinary pull-request CI remains available.
Release CI compares the pinned-model results against the checked-in metrics.
It also runs all labeled rule examples against `evals/expected-metrics.json`,
printing precision/recall deltas and blocking drops larger than 0.02.
The release workflow requires `OPENROUTER_API_KEY`; unavailable inference fails
the release. Review baseline updates and model changes as rule-quality changes.
Maintainers can manually run the **Release** workflow on a branch to validate
both quality gates with the repository secret. Manual runs never publish,
including when a tag is selected; only tag pushes publish releases. Installation
smoke tests follow successful publishing runs, not manual quality validation.
The zero-recall pilot baseline alone provides no useful recall regression floor;
the full rule corpus provides the additional gate.

Add `--optimize precision --min-recall 0.6 --format json` to `reaper eval`
(including patch benchmarks) for a 0–1 sweep in 0.01 increments and per-rule
candidate thresholds. Ties favor recall, then the highest qualifying threshold.
No configuration is modified. Hidden-test cases are rejected for calibration.
Validate candidates on a separate held-out corpus before changing defaults.

### SARIF output

`reaper check --format sarif > reaper.sarif` emits SARIF 2.1.0, including
confidence, thresholds, line ranges, and incomplete-analysis notifications.
Patch-wide findings have no invented file location. Output is tested against
the [OASIS SARIF schema](https://github.com/oasis-tcs/sarif-spec/blob/main/sarif-2.1/schema/sarif-schema-2.1.0.json).
After installing Reaper, use these steps with `security-events: write` permission:

```yaml
- name: Run Reaper
  run: reaper check --diff origin/main --format sarif > reaper.sarif
  env:
    TYPESAFE_API_KEY: ${{ secrets.TYPESAFE_API_KEY }}
- uses: github/codeql-action/upload-sarif@v4
  if: always() && hashFiles('reaper.sarif') != ''
  with:
    sarif_file: reaper.sarif
```

See [GitHub's SARIF upload requirements](https://docs.github.com/en/code-security/how-tos/find-and-fix-code-vulnerabilities/integrate-with-existing-tools/upload-sarif-file)
for repository eligibility and permissions. The run step keeps its failing exit
status while the upload step still runs.

The repository includes 280 labeled examples for measuring rule quality:

```sh
reaper eval
reaper eval --rule narrating-comment
reaper eval --format json
```

Reaper is not a formatter, code generator, or general-purpose reviewer. It sits
between deterministic static analysis and full LLM code review: narrow,
repeatable semantic checks designed for the coding loop.
