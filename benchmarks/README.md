# Patch benchmark

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

See [the calibration review](calibration-review.md) for candidates computed from
the measured scores without additional inference or changes to defaults.
The [prospective validation report](validation/report.md) compares those frozen
candidates on 32 separately authored patches, including hard negatives. It
records generalization failures and keeps all defaults unchanged.

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
separate logical evaluations. Returned model identity is unavailable through the
current decision endpoint adapter, so the requested ID is not proof of an
unchanged backend revision.

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

## Controlled question/context experiments

`--benchmark-experiment path.json` is restricted to Git benchmarks. It accepts a
version 1 JSON object with `name`, `context` (`current` or `snapshots`) and optional
`questions` mapping existing signal IDs to revised instructions. Unknown fields
and question IDs fail validation. Experiments do not modify `check`, configured
thresholds, composition, eligibility or rule grouping. Signal evidence and request
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

After an independent reviewer approves the full corpus and labels, record a
review object with `reviewer`, `evidence` (a review record reference),
`independent: true`, and `corpus_sha256` from that corpus's Git evaluation report.
The reviewer must be independent of fixture authorship. Software verifies the
hash binding and attestation fields; it cannot authenticate the person or quality
of their review. Any corpus edit invalidates that binding. Review metadata is
never sent to the evaluator. Select the reviewed corpus directory in the workflow
when it is ready, without changing these targets to make a failing run pass.

The gate accepts only configured Git evaluations of production rules, rejects
experiment overrides or missing blocking rules, and recomputes metrics from case
results. Extraction misses remain in recall. Provider failures already fail the
evaluation before the gate. Passing an old zero-recall baseline is insufficient.
