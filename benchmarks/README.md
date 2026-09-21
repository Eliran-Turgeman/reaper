# Patch benchmark

Keep generated reports, API probes and exploratory run output in the Git-ignored
`.local/` directory. This tree retains reusable fixtures, experiment specifications,
and the frozen baselines required by tests and release gates. Experimental results
do not authorize changes to production questions or thresholds.

New request records retain the supplied fixture state separately from scores,
the requested model and provider-reported resolved model, request/provider ID,
per-call token usage and reported cost when available, and elapsed milliseconds
(including retries). Missing usage or cost remains absent, not zero. Historical
artifacts are unchanged. State contains source evidence; a signal's `evidence`
field still contains its question, not a model-generated explanation. The minimum
of signal values is a rule score, not a calibrated violation probability.

Git experiment files may add `state_format: "json-text"` or `"json-object"`.
Both contain identical named evidence fields; the former sends JSON as a string,
the latter as a native object. Omitting the field preserves historical text.
An optional `criteria` map uses complete question IDs as keys and objects with
nonempty `true` and `false` descriptions. Questions remain Noul. These are
experiment-only controls, not production configuration settings. Exact criteria
and structured state are recorded and included in request fingerprints. Native
state, text state, and different criteria cannot share production cache entries.

The same `--benchmark-experiment` flag also works with `--eval-dir evals` (without
`--benchmark-dir`) for all 14 seed-rule corpora. Example experiments require
`context: "current"`; repository snapshots and helper retrieval require Git
fixtures. Existing example inputs and provenance remain identical when no
experiment is selected. This enables matched wording/state/criteria comparisons
without inventing task descriptions or converting snippet labels to Git labels.

For separately frozen decomposition experiments, `signals` may replace a rule's
signal list with `ID`, `Instructions`, optional `Criteria`, and optional `Negate`.
At least one factual supporting signal is required. A negated signal contributes
`1 - P(yes)` to the minimum; raw provider values remain in request records and
observations mark `negated: true`. Observations name the composition and effective
rule version. This is an experimental decision score, not a joint probability or
a validated permission cutoff. Overrides never alter the production registry,
severity, applicability, or thresholds, and cannot be used by the release gate.

Permission experiments may instead mark one signal per rule with
`"Role": "permission"` and supply `permission.allowed_at_least` plus
`permission.disallowed_at_most`. The factual score then uses factual signals
only. The raw permission score is reported separately as `allowed`, `disallowed`,
or `uncertain`; it never suppresses or boosts the factual score. These boundaries
are frozen experiment controls, not production calibration.
`permission-decision-v2/policy-targeted.json` covers authorization and
validation with targeted Go evidence. `policy-contracts.json` covers fallback,
cancellation, and assertion task contracts with current diff context.
When `permission_action` is `suppress-allowed`, only a classified `allowed`
outcome suppresses the experimental finding. `uncertain` stays visible and the
factual score remains unchanged.
`policy-assertion-dev.json` is an explicitly development-only assertion boundary
experiment. It uses a lower `allowed` boundary after partial-permission fixtures
separated from exact permission; it is not independent calibration or a default.

`permission-decision-v3` replaces the overloaded permission question with two
independently scored requirements: the task must authorize the exact changed
scope, and the implementation must stay within that authorization without
conflicting preservation requirements. Permission uses the minimum of those
signals, so every condition must reach the `allowed` boundary before suppression.
Its targeted policy also distinguishes changed unresolved calls from identical
unresolved calls on both snapshots. The latter remain disclosed limitations but
do not alone force an evidence abstention.

`permission-decision-v4` keeps those permission requirements but evaluates them
in a separate model request so additional factual questions cannot influence the
permission scores. Its fallback and assertion rules use maximum composition over
complete violation alternatives: discarded failures, fallback/success conversion,
and specific assertion-property losses. This remains question-driven semantic
evaluation; it adds no AST-derived rule facts.

Experiments may select `maximum-of-signals-v1` for a rule through the
`compositions` map. This is used for cancellation alternatives where either
detaching caller cancellation or converting cancellation into success is enough;
production rules retain their existing composition.

Targeted experiments may set `evidence_policy: "require-complete-targeted"`.
Reports then classify missing, unparseable, unsupported, or budget-truncated
before/after function evidence as `insufficient-evidence`. Such cases retain
their raw request records but do not produce findings. Positive abstentions count
as misses; negative abstentions do not count as confirmed true negatives.

`complete-go-v1` contains syntactically complete paired development fixtures for
explicitly allowed, explicitly disallowed, and uncertain authorization/validation
tasks. Each case records its historical predecessor. It remains synthetic
development evidence and cannot satisfy the independent release gate.

`predicate-dev-v2` contains one matched positive/negative pair for discarded
errors, cancellation ownership, and assertion specificity. The three
`factual-predicates-v2` specifications must be run independently against this
fixed corpus before any combined candidate is proposed.

`permission-edge-v2` contains vague, partial, wrong-operation, and conflicting
task instructions. These cases retain a factual violation and label permission
as `disallowed` or `uncertain`; neither outcome may suppress the finding.

`auth-validation-edge-v3` applies the same permission-edge families to complete
Go authorization and validation fixtures so targeted before/after evidence can
be assessed without historical fragment parse failures.

`reaper eval --benchmark-dir benchmarks/patches --format json` evaluates labeled
patches through the same runner as `check`. Every case stores its task, before
and after code, diff, source path, expected label, rationale, rule, provenance,
and split. This pilot has 32 realistically generated service patches, 75% clean,
covering four regression rules. It is not independent production evidence.
Templates share context and all belong to **dev**; do not report these as a
held-out test set. Real agent patches should retain their issue/commit/license
provenance and undergo independent labeling before inclusion.

Keep train/dev and hidden-test tasks disjoint by repository and change family.
Hidden-test labels belong in a separate private corpus, not this public tree.
The loader accepts `train`, `dev`, and `hidden-test`; select a separate corpus
directory when evaluating a held-out set. Do not calibrate against hidden tests.

`patches/results.json` records a measured run, including model/provider,
per-case scores, and per-rule precision, recall, false-positive rate, and default
threshold. Small sample sizes and generated patches limit generalization.
Release CI uses the pinned model ID `typesafe/jev-1.13` and compares these metrics
with a 0.02 maximum precision/recall drop. Missing credentials, provider failures,
or missing baseline rules fail the gate; model changes require a reviewed new
baseline. Provider-side behavior can change even for an unchanged model ID.

The [validation corpus](validation/README.md) contains 32 separately authored
patches, including hard negatives. Its frozen protocol and recorded baseline
support reproducible comparison; they do not establish release readiness.

## Evaluating the real Git path

The default `--benchmark-mode snippet` preserves historical measurements: it
constructs a full-snippet unit and enables only the labeled rule. To measure
extraction, eligibility, repository context and normal rule grouping, use:

```sh
reaper eval --benchmark-dir benchmarks/validation --benchmark-mode git --format json
reaper eval --benchmark-dir benchmarks/git-path --benchmark-mode git --format json
```

Git mode creates a temporary repository, stages the before snapshot, writes the
after snapshot, and collects an actual diff using the same collector and runner
as `check`. It uses the loaded configuration, disables the score cache and retains
below-threshold observations with every constituent signal. It does not run
fixture code, create commits, use Git hooks or contact a Git remote.

For a matched grouping experiment, add `--benchmark-grouping isolated` to enable
only the labeled rule. The default `configured` grouping preserves configured
rule enablement, exclusions and batching. These modes measure different inputs;
the baseline gate rejects comparisons across Git/snippet modes or grouping modes.
Historical snippet baselines remain unchanged.

Each case's score is the maximum labeled-rule score across its extracted units;
each unit's score retains the rule's normal composition. `evaluated: false` and
`coverage_reason` record extraction, exclusion or eligibility misses. Unevaluated
positives remain false negatives, even at a zero threshold. Provider failures or
incomplete analysis fail the run instead of being reported as low scores. Labels
and rationales are retained in the corpus, never sent as evaluator evidence.

Existing single-file cases work unchanged; the stored `diff` is ignored in Git
mode and regenerated from `before` and `after`. For multi-file cases replace
`before`, `after` and `file` with complete `before_files` and `after_files` objects
mapping repository-relative paths to file contents. Supply both objects (an empty
object represents an empty tree). An absent after path is a deletion; an empty
string is an existing empty file. Unchanged helpers must occur in both snapshots
to be available to repository retrieval. Paths must be local, canonical and
forward-slash separated; `.git`, traversal and conflicting case aliases are
rejected. Metadata (`id`, `task`, `rule`, `expected`, `rationale`, `provenance`,
`split`) follows the existing format. `git-path/cases.json` is a small development
suite for extraction and control-flow families, not independent holdout evidence.

## Reproducing a measurement

All evaluation modes now include constituent signals and per-case request
records, including scores below the reporting threshold. Each record contains
the requested model, actual grouped questions and scores, a context SHA-256 and
a request SHA-256. Records follow provider capability splitting; concurrent calls
are sorted by fingerprint for stable output. Provider-internal retries are not
separate logical evaluations. Provider-reported resolved model identity is
retained when available; missing identity remains unknown. The requested model
ID alone is not proof of an unchanged backend revision.

Report provenance binds the loaded corpus, rule versions/questions, relevant
configuration and context protocol. Fingerprints hash JSON-encoded input values;
request hashes cover the normalized request before provider wire serialization.
Gold labels/rationales affect the corpus fingerprint, never the request. Git and
CLI checks share request construction, with an integration test comparing their
actual HTTP payloads on the same change. The seed and historical snippet modes
retain their original state format so old measurements remain reproducible; use
Git mode for production-path claims.

New baselines with provenance reject mismatched inputs even if metrics match.
Historical baselines remain usable with an explicit warning that input equivalence
cannot be verified. They are not silently upgraded or remeasured. A new context
protocol must receive a new protocol version and a separately reviewed baseline.

`git-v2` identifies the corrected root-test-directory classification and hunk
extraction that retains leading increment/decrement code. Previous `git-v1`
results remain historical; do not silently replace their provenance.

`git-v3` additionally binds parser-backed Go candidate selection for boolean
inputs and single-call forwarders. On parseable Go changes, those advisory rules
are sent to Jev only when syntax establishes the relevant candidate shape.
Unsupported languages and unparseable Go retain the previous semantic path.

`git-v4` binds targeted Go evidence protocol `go-functions-v2`, which resolves
one-hop helpers as before but only treats unresolved direct calls as incomplete
when the call expression changed between snapshots. Identical unresolved calls
remain visible to the evaluator and report without automatically abstaining.

## Controlled question/context experiments

`--benchmark-experiment path.json` accepts a version 1 JSON object with `name`,
`context` (`current`, `snapshots`, `matched` or `targeted`) and optional `questions`
mapping signal IDs to revised instructions. Seed evaluations require `current`;
patch experiments require Git mode. Optional state, criteria and signal controls
are described above. Unknown fields and question IDs fail validation. Experiments
do not modify `check`, configured thresholds, eligibility or rule grouping.
Signal evidence and request
fingerprints describe the transformed request actually sent to the evaluator.
The experiment specification has its own provenance hash.

The four specifications in `experiments/auth-validation-v2` hold questions and
context independently constant or vary them together. For example:

```sh
reaper eval --benchmark-dir benchmarks/quality-dev --benchmark-mode git --benchmark-grouping isolated --benchmark-experiment benchmarks/experiments/auth-validation-v2/questions.json --format json
```

Snapshot context appends the complete before/after file maps from the fixture as
code evidence. It includes every fixture file, so use curated fixtures suitable
for submission to the provider. It does not append expected labels, rationale or
provenance. Evidence larger than 64 KiB fails instead of being truncated. This is
a context experiment on small fixtures, not the production retrieval policy.

The `matched` and `targeted` experimental context modes preserve each focal hunk
and add source-located Go function bodies from both fixture snapshots. `targeted`
also includes one-hop direct function candidates from the same package directory;
it omits unrelated functions, test files, imports and receiver-method resolution.
Lexical candidates are not type-checked proof of enforcement. Notes identify
unresolved calls, parse failures and limits: 128 files / 1 MiB scanned per side,
256 KiB per helper file, and 16 KiB of added evidence per unit. Whole functions
are omitted when the evidence budget is exhausted. These modes use explicit
fixture snapshots, not live repository discovery, and do not change production
questions, thresholds, rule scope, or retrieval behavior.
Compare the four conditions on identical cases, model, grouping and thresholds;
repeat borderline cases before proposing defaults. A score change from duplicated
already-visible context does not establish a gain from additional information.

## Absolute release requirements

The user approved `release-policy.json` on 20 September 2026. Every default
blocking rule must achieve recall >= 0.80 and false-positive rate <= 0.01 on at
least 20 positive and 100 negative independently reviewed examples. These are
point-estimate targets, not statistical confidence bounds or proof of deployment
precision. The release workflow checks them before relative development baselines.

Run `reaper eval --benchmark-dir benchmarks/validation --benchmark-mode git
--quality-policy benchmarks/release-policy.json`. Without an independent review,
this exits 1 before making provider requests. The checked-in `review: null` is
intentional: no agent-authored corpus is asserted to have independent human review.
Normal PR CI and exploratory evaluations continue to work.

After an independent reviewer approves the full hidden-test corpus and labels, record a
review object with `reviewer`, `evidence` (a review record reference),
`independent: true`, and `corpus_sha256` from that corpus's Git evaluation report.
The reviewer must be independent of fixture authorship. Software verifies the
hash binding and attestation fields; it cannot authenticate the person or quality
of their review. Any corpus edit invalidates that binding. Review metadata is
never sent to the evaluator. Select the reviewed corpus directory in the workflow
when it is ready, without changing these targets to make a failing run pass.
The quality policy requires every release case to use the `hidden-test` split;
train or development cases cannot authorize release even if reviewed.

The gate accepts only configured Git evaluations of production rules, rejects
experiment overrides or missing blocking rules, and recomputes metrics from case
results. Extraction misses remain in recall. Provider failures already fail the
evaluation before the gate. Passing an old zero-recall baseline is insufficient.

## Frozen development candidate

`candidates/blocking-v3.json` is the current development candidate. It combines
the v3 targeted authorization/validation experiment with the v4 question-only
fallback/assertion decomposition. `blocking-v1.json` and `blocking-v2.json`
remain historical predecessors. Loading a bundle verifies every artifact
SHA-256. All are explicitly `release_eligible: false`; changing any bound
artifact requires a new candidate version and measurements.

## Blinded independent review

Export a label-free review package without provider access:

```sh
reaper review-pack --benchmark-dir path/to/corpus --output review-pack.json
```

The package contains task and before/after source, but excludes expected labels,
rationales, provenance, model scores, and requests. It binds the full source
corpus, selected source cases, and blinded payload with separate SHA-256 values.
Use `--limit` and `--seed` for a deterministic mixed pilot. Reviewers fill
`factual_outcome`, `permission`, `evidence`, and notes using the included rubric.
Release review still requires an independent attestation bound to the complete
hidden-test corpus; exporting a package does not create that attestation.
