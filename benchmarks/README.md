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
