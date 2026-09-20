# Prospective candidate validation

This separate set evaluates thresholds already selected in the
[calibration review](../calibration-review.md). `protocol.json` freezes both
candidate families and the SHA-256 of `cases.json` (UTF-8, LF line endings)
before inference. Do not optimize thresholds on the validation scores.

The 32 patches include three positives and five hard negatives per rule across
Go, Python, TypeScript, JavaScript, Java, and C#. Hard negatives include visible
equivalent guards, explicit contract changes, removed protected operations,
error propagation, and cancellation cleanup. The Go test checks the corpus
fingerprint and rejects exact overlap with the original patch pilot.

These are synthetic snippets authored and labeled by the same coding agent,
not independent human labels, compiled production programs, or a hidden test
set. Each snippet supplies the evidence needed for its label, but omitted
dependencies are illustrative. The set remains public `dev` data. Its 37.5%
positive rate is deliberately diagnostic and does not estimate production
precision or prevalence. Language and rule slices are too small for reliable
generalization. No blocking defaults or release baselines change here.

Reproduce scoring with:

```sh
go run ./cmd/reaper eval --benchmark-dir benchmarks/validation --provider openrouter --model typesafe/jev-1.13 --format json > benchmarks/validation/results.json
```

Compare all results at current defaults and both preselected candidate
thresholds, including failures. Treat any promising result as grounds for
larger independently labeled validation, not automatic approval to lower a
blocking threshold.

The [measured comparison](report.md) records all results and candidate errors.
Default thresholds caught 1 of 12 positives; the candidate comparisons exposed
missed regressions and false positives. These results do not justify a general
threshold reduction.
