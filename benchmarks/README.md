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
