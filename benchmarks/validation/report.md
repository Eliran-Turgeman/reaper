# Measured validation results

Measured with `openrouter` / `typesafe/jev-1.13` on 2026-09-20, after committing
the labels and protocol in `83df346`. Defaults and both candidate families were
selected before these scores were observed. No threshold was optimized here.

| Rule | Threshold source | Threshold | TP | FP | TN | FN | Precision | Recall |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| `removed-authorization-check` | default | 0.95 | 0 | 0 | 5 | 3 | N/A (no findings) | 0.0% |
| `removed-authorization-check` | patch | 0.84 | 2 | 0 | 5 | 1 | 100.0% | 66.7% |
| `removed-authorization-check` | seed | 0.85 | 2 | 0 | 5 | 1 | 100.0% | 66.7% |
| `removed-validation` | default | 0.95 | 0 | 0 | 5 | 3 | N/A (no findings) | 0.0% |
| `removed-validation` | patch | 0.53 | 1 | 0 | 5 | 2 | 100.0% | 33.3% |
| `removed-validation` | seed | 0.52 | 1 | 0 | 5 | 2 | 100.0% | 33.3% |
| `silent-failure-fallback` | default | 0.92 | 1 | 0 | 5 | 2 | 100.0% | 33.3% |
| `silent-failure-fallback` | patch | 0.19 | 3 | 1 | 4 | 0 | 75.0% | 100.0% |
| `silent-failure-fallback` | seed | 0.39 | 3 | 0 | 5 | 0 | 100.0% | 100.0% |
| `swallowed-cancellation` | default | 0.95 | 0 | 0 | 5 | 3 | N/A (no findings) | 0.0% |
| `swallowed-cancellation` | patch | 0.38 | 3 | 1 | 4 | 0 | 75.0% | 100.0% |
| `swallowed-cancellation` | seed | 0.66 | 1 | 1 | 4 | 2 | 50.0% | 33.3% |

The CLI uses zero precision when no findings fire; this table uses N/A to
distinguish an undefined ratio from observed false positives. Across current
defaults, only 1 of 12 regressions was detected, with 0 of 20 clean patches
flagged. Each rule has just three positives and five negatives; these counts
cannot establish reliable production precision or recall.

## All candidate errors

| Rule | Candidate source | Case | Error | Score |
|---|---|---|---|---:|
| `removed-authorization-check` | patch | `late-permission` | missed regression | 0.56 |
| `removed-authorization-check` | seed | `late-permission` | missed regression | 0.56 |
| `removed-validation` | patch | `positive-transfer` | missed regression | 0.45 |
| `removed-validation` | patch | `archive-traversal` | missed regression | 0.29 |
| `removed-validation` | seed | `positive-transfer` | missed regression | 0.45 |
| `removed-validation` | seed | `archive-traversal` | missed regression | 0.29 |
| `silent-failure-fallback` | patch | `explicit-offline-cache` | false positive | 0.26 |
| `swallowed-cancellation` | patch | `independent-audit-job` | false positive | 0.68 |
| `swallowed-cancellation` | seed | `detached-query` | missed regression | 0.63 |
| `swallowed-cancellation` | seed | `dropped-fetch-signal` | missed regression | 0.59 |
| `swallowed-cancellation` | seed | `independent-audit-job` | false positive | 0.68 |

## Decision

- Keep all blocking defaults unchanged and PR #1 in draft.
- Authorization candidates are encouraging but miss permission checks moved after side effects. Two detections and five clean examples are insufficient release evidence.
- Validation candidates fail two of three regression cases, including archive path containment. The original development recall did not generalize.
- The patch-derived failure-fallback candidate flags an explicitly requested offline cache fallback. The seed-derived candidate has no errors here, but its earlier seed precision was only 63.6%; this small set does not override that evidence.
- Both cancellation candidates flag explicitly requested independent audit work; the higher candidate also misses a detached query and a removed abort signal.
- Do not pick a new threshold between observed scores: that would turn this validation set into another calibration set. Next quality work should target these failure modes, use fresh independently reviewed cases, and obtain approval before changing blocking defaults.

## Reproduction and limits

See [the protocol](protocol.json), [labeled cases](cases.json), and [raw scores](results.json).
Run `go test ./internal/eval -run TestValidation -v` to verify the frozen corpus,
recompute recorded metrics, and print both preselected candidate comparisons.
The model ID is pinned, but provider behavior can change. These are public
synthetic development cases authored and labeled by the same coding agent;
there is no independent human review or production base-rate evidence.
