# Priority roadmap implementation status

Implemented in the supplied P0 → P1 → P2 → P3 order, with `go test ./...`
passing before proceeding between rows. This record describes the implementation
in [PR #1](https://github.com/Eliran-Turgeman/reaper/pull/1); it does not claim
that these changes have been merged or released.

| Priority | Row | Delivered |
|---|---|---|
| P0 | Canonical naming | Module/imports, installers, README, release metadata, local Go installation, installation smoke workflow |
| P0 | License | MIT license, README link, license included in release archives |
| P0 | Normal CI | Push/PR tests, vet, build and formatting on Linux/Windows/macOS; Linux race tests |
| P0 | Incomplete analysis | Explicit status/completeness, skipped-unit reasons, configurable policy, exit 2 by default |
| P1 | GitHub Action | Matching-source binary, safe inputs, secret handling, diff base, annotations and fork failure |
| P1 | SARIF/annotations | SARIF 2.1.0 with offline official-schema validation; escaped workflow annotations |
| P1 | Evidence/scores | Sorted signal scores/predicates, composed confidence, JSON envelope version 2 |
| P1 | Agent output | Compact findings, task context, evidence, fingerprint and remediation objective |
| P1 | CI task context | Explicit task/file/environment precedence; sanitized PR title/body; provenance logging |
| P1 | Rule packs | Regression defaults, opt-in subjective packs, per-rule overrides and pack filtering |
| P1 | Regression rules | Removed authorization, removed validation and swallowed cancellation; 20 labeled cases each |
| P1 | Privacy | Request contents, endpoints, keys, cache, context retrieval, logs and enterprise guidance |
| P2 | Patch benchmark | 32 realistically generated patches, 75% clean, provenance/splits, measured metrics and release comparison |
| P2 | Calibration | Threshold sweeps, precision optimization and minimum recall; no automatic config mutation |
| P2 | Baselines/suppressions | Fingerprints, baseline creation/checking, reasoned inline suppression, retained suppressed findings |
| P2 | Agent integrations | Versioned AGENTS.md, Codex, Claude Code skill and Cursor rule with separate instructions |
| P2 | Local feedback | Useful/false-positive feedback and local per-rule metrics; no upload |
| P2 | Release gates | Pinned-model patch and 280-example corpus gates, measured baselines, tolerances and deltas |
| P3 | Logical boundaries | Native Go parsing; conservative common-language scanning; hunk fallback |
| P3 | Selective context | Rule-declared context, bounded tracked-file reference searches, isolated evaluation and debug evidence |
| P3 | Provider boundary | Normalized decision API, Jev adapter, capability metadata, non-batching support and score validation |
| P3 | Cost/latency | Per-run requests/tokens/cache/duration, optional explicit pricing, measured generated-workload latency |

## Verification and remaining limits

- Formatting, `go test ./...`, `go vet ./...`, Windows build, Linux/macOS cross
  builds, Node Action tests, workflow linting, SARIF schema validation, and local
  canonical-path Go installation passed.
- The public `go install github.com/Eliran-Turgeman/reaper/cmd/reaper@latest`
  still resolves to v0.4.0 with the old module declaration. A maintainer must
  publish the corrected module before that required public smoke check passes.
  The stable Action reference and release smoke execution likewise require
  merging and publishing these changes.
- [Hosted CI](https://github.com/Eliran-Turgeman/reaper/actions/runs/35505148098)
  passed on Linux, Windows, and macOS, including Linux race testing and local
  installation on all three platforms. The first Windows run exposed CRLF
  checkout conversion; `.gitattributes` now preserves LF for Go source files.
- [Manual release validation](https://github.com/Eliran-Turgeman/reaper/actions/runs/35505066556)
  passed both pinned-model quality gates using the dedicated repository secret.
  Precision and recall deltas were zero for every rule. Publishing was skipped
  as intended. This validates gate operation, not satisfactory detection quality.
- Full-patch Reaper analysis encountered the provider's context limit for the
  patch-scoped `scope-creep` rule. The report correctly remained incomplete
  (exit 2); no rule or policy was weakened. Six bounded code groups passed,
  followed by passing checks on final fixes. Group checks do not replace the
  missing whole-patch evaluation.
- `go test -race ./...` passed on the current source in an isolated Ubuntu WSL
  workspace using checksum-verified Go 1.25.1 and GCC. Native Windows CGO was
  unavailable. The hosted Linux CI job also passed race testing.
- The patch corpus is a small generated **development** pilot, not independently
  mined agent history or a hidden test set. At default thresholds it detected
  0/8 positives with 0/24 false positives. The measured full corpus also exposes
  poor recall for many rules. Baselines prevent further regressions; they do not
  establish satisfactory detection quality. Calibration produces candidates
  for separate review without lowering thresholds automatically.
  [The calibration review](../benchmarks/calibration-review.md) records the
  current candidates and their precision/recall tradeoffs. Defaults are unchanged.
- A [separately frozen 32-case validation set](../benchmarks/validation/report.md)
  tested the preselected thresholds across six languages. Current defaults
  detected 1/12 positives with 0/20 false positives. Candidate thresholds exposed
  missed authorization/validation regressions and false positives for permitted
  fallback/cancellation behavior. All labels and thresholds were committed before
  inference; these remain synthetic cases without independent human review.
  The PR remains in draft, and no defaults or release baselines were changed.
- Logical extraction keeps changed hunks/locations and expands surrounding
  context. Non-Go scanners are deliberately conservative and are not complete
  parsers; unusual syntax falls back. Repository retrieval is lexical, bounded,
  and not proof that no other implementation exists.
- Token costs use user-supplied rates and reported usage only. Unknown cost is
  null; partial usage is identified. The latency pilot used three Windows
  samples per generated workload, with medians 0.57/1.30/3.32 seconds for
  1/8/32 changed files, respectively.

## Rule-quality follow-up (20 September 2026)

The [measured experiment report](../benchmarks/experiments/rule-quality-report.md)
contains exact case references, constituent signal records, four-condition
comparisons and repeated-request stability. This follow-up keeps production
questions, thresholds and historical result baselines unchanged.

| Approved plan step | Status and evidence |
|---|---|
| Fix eligibility and use real Git fixtures | Complete. Insertion-only existing-file changes reach regression rules; new files retain their exclusion. Actual Git tests cover renamed, empty and deleted files; spaced paths are parsed correctly. |
| Shared request path and signal/provenance records | Complete for production-path evaluation. Git benchmark/check HTTP payload parity is tested. Seed/snippet legacy state formats remain explicit; question assembly is shared and every mode records signals and requests. |
| Review label contracts and expand fixtures | User approved specific-task exemptions. Added 24 cross-language task-policy cases, 12 cancellation/fallback/assertion cases, six extraction cases and four multi-file context cases. Ambiguous historical labels are documented without rewriting history. Independent review remains outstanding. |
| Authorization/validation question-context experiment | Complete as a development experiment: current, questions, context, both; matched inputs and fixed thresholds. Added a follow-up that supplies otherwise invisible helper implementations. No universal candidate is ready to promote. |
| Cancellation/fallback/assertion semantics | Candidate questions measured. Assertion and fallback separation improve on paired cases; cancellation requires the simpler ownership question to separate independent work. Default-threshold recall is still inadequate. Production promotion is deferred. |
| Freeze candidates and evaluate fresh cases | Two additional runs on a committed seven-case stability subset are complete; exact request fingerprints match. Three of 28 case-condition combinations cross exploratory probe cutoffs. Final candidate selection and independently reviewed fresh holdout evaluation remain outstanding. |
| Absolute release gates and default review | Implemented with the user's approved >=80% recall, <=1% false positives, and >=20 positive/100 negative independently reviewed cases per blocking rule. Missing review fails before inference. No default change is approved or applied. |

`go test ./...` passed between implementation steps. Formatting, vet, build,
Node Action tests and actionlint passed. Each bounded Reaper check completed
without findings. The new release gate intentionally fails on the current
unreviewed synthetic corpus; this supersedes the earlier passing relative gates
as a release-readiness decision. PR CI remains separate from release readiness.

A combined Reaper check of the quality follow-up against `03a9515` completed
77 units / 387 semantic checks without warnings or errors. This covers the
follow-up changes; the earlier whole-roadmap scope-context limit remains recorded
above. The release-gate CLI prints its missing-review reason before exiting 1.

The next implementation direction is bounded retrieval of referenced enforcement
helpers, with explicit evidence provenance and coverage, followed by fresh
independent review and candidate validation. Four synthetic Go helper cases are
insufficient to justify broad repository transmission or production thresholds.
