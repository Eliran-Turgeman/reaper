# Jev question and context improvements: implementation results

Measured 21 September 2026. This report follows the
[capability audit and rule-by-rule plan](jev-capabilities-and-rule-audit.md).

## Outcome

The measurement and experiment infrastructure is implemented, all 14 rules have
been tested with simpler questions, and all five blocking rules have been tested
with separate factual and permission questions. **None of the measured candidates
is ready to replace the production defaults.**

The 84 completed runs show that simpler language alone is insufficient. Factual
questions improve several scores, but permission composition and missing source
evidence remain significant problems. We fixed two source-collection bugs and
added an opt-in bounded helper-context path. Questions, thresholds, historical
baselines, and strict release requirements remain unchanged by default.

## What is implemented

| Plan item | Implementation and status |
|---|---|
| Faithful measurement | Exact state, questions and criteria, raw signal values, composition/version, request fingerprints, resolved model, provider request ID, token usage, reported cost and elapsed request time are retained. Missing usage remains unknown. |
| Structured state and criteria | Native object/array state and explicit Noul true/false criteria work through the decision API, provider adapter, cache and benchmark. Legacy text remains supported; invalid or ambiguous state fails before dispatch. |
| All-rule wording sweep | Frozen shorter instructions for all 28 signals, reducing instruction words from 940 to 620. Thirty repeated runs separate wording, representation and criteria changes. |
| Before/after source consistency | Staged changed-file extraction and repository search both read captured index blobs. Unstaged edits/deletions cannot change their evidence; an index change during diff collection requires a retry. |
| Targeted helper evidence | The benchmark and opt-in CLI share one bounded Go collector. Authorization and validation receive matched functions and one-hop same-package helper candidates, with exclusions and resolution limits recorded. Irrelevant test files no longer consume the helper budget. |
| Factual versus permission questions | Fifty-four repeated runs compare baseline, factual-only, and factual-plus-permission conditions for authorization, validation, cancellation, assertions and fallback. Experiments preserve raw permission P(yes) separately from the complemented score. |
| Advisory rules | All nine are included in the language/state/criteria comparisons. Parser-based candidate selection, richer architecture evidence and scope-group composition remain follow-up work. |
| Calibration and promotion | Not performed: quality and independent-review prerequisites are not met. No threshold was selected from these results. |

The helper option is `reaper check --experimental-context targeted-go`. It is an
explicit experiment, supports Go helper resolution only, and does not support
`--all`. Production default checks retain their current questions and context.
See [README](../README.md) and [privacy/data flow](privacy.md) for limits and the
source sent to the provider.

## Measured results

Each condition ran three times against frozen inputs. Ranges below describe
variation across repeats; they do not increase independent sample counts.

| Comparison | Detected positives | False positives | Interpretation |
|---|---:|---:|---|
| Seed original text, 140 positive / 140 negative | 11–13 | 3–4 | Original-condition baseline |
| Seed shorter wording only | 7–9 | 1–2 | Fewer false positives, also fewer detections |
| Seed shorter questions + native object + criteria | 15–17 | 5 | More detections, more false positives |
| Git validation native-object baseline, 12 positive / 20 negative | 1 | 0 | Phase B baseline |
| Git validation factual questions | 5 | 1–2 | Better sensitivity; task exceptions intentionally absent |
| Git validation factual + permission composition | 0 | 0 | Permission term suppresses valid findings |
| Helper corpus factual questions, 6 positive / 6 negative | 0 | 0 | Improved score separation still falls below 0.95 |
| Contract corpus baseline, 6 positive / 6 negative | 1 | 2 | Existing assertion policy mistakes |
| Contract corpus factual + permission composition | 1 | 0 | Removes these false positives, but recall remains low |

The two phases use different baselines: Phase A starts with original flat text;
Phase B uses native objects and targeted evidence where applicable. Do not
attribute differences between phases to one question change.

- [Phase A: simpler questions, state and criteria](../benchmarks/experiments/simple-questions-v1/report.md)
  includes all 14 rules, 30 raw reports, input-integrity checks and reproduction commands.
- [Phase B: factual and permission questions](../benchmarks/experiments/atomic-policy-v1/report.md)
  includes every rule/corpus result, 54 raw reports, source-coverage limitations,
  and nine exact diff/question/score examples.

All recorded calls resolved to `typesafe/jev-1.13-20260917`. Across both phases,
8,001 recorded calls used 3,754,272 input and 488,499 output tokens, with
provider-reported cost approximately **$0.157679**. This total excludes earlier
capability probes and repository Reaper checks. Concurrent request timing is
recorded for investigation, not presented as a controlled latency benchmark.

## What is holding the benchmark back

### 1. Permission composition imposes an unintended second high-confidence gate

For `export-role`, the task says to retain administrator access and the diff
removes the administrator check. The factual answers are **0.98** for previous
authorization and **0.97** for an unguarded operation. Explicit task permission
scores **0.09**. The tested composition produces:

```text
min(0.98, 0.97, 1 - 0.09) = 0.91
threshold = 0.95 -> no finding
```

At threshold 0.95, this composition requires permission P(yes) to be no greater
than 0.05. Low estimated permission is therefore insufficient. In each validation
repeat, five positive cases have factual answers above threshold within the same
policy request, yet all five are suppressed by its permission term. This
counterfactual holds the model outputs fixed; it does not mix separate calls.

Separate permission questions do distinguish many explicit requirements from
vague requests. That is useful evidence about the predicate, but it does not
validate the tested composition. Multiplying the probabilities would not solve
the calibration problem either.

### 2. Some factual questions still miss their intended predicate

In the first helper run, an authorization helper that never rejects an actor has
a factual score of **0.89**, compared with **0.13** for an enforcing helper.
That is useful separation but still misses the unchanged 0.95 threshold. In the
lost-read-error example, `handles-failure` scores **0.50**, while `failure-hidden`
scores **0.96**: discarding an error does not read naturally as handling one.
Cancellation ownership and assertion-contract examples also remain weak.

### 3. More structured state does not create missing evidence

The helper collector can resolve only supported lexical relationships. All nine
Go examples in the historical validation corpus are incomplete source fragments
that fail its matched-source parsing. Their original local code and diff still
reach Jev, but no matched helper evidence can be inferred from that failure.
The other 23 validation examples use unsupported languages for this collector.
The dedicated Go helper fixtures do parse, which is why they are reported
separately. Completion of a provider call must not be confused with complete
source coverage.

### 4. The evaluation set cannot authorize release

These are synthetic development cases already used to guide changes. Suites
share templates and are not independent held-out evidence. One `git-path`
negative is intentionally ineligible and receives no model evaluation; the
detailed report identifies it instead of calling it a confirmed safe prediction.

The approved release requirements remain at least 80% recall, at most 1%
empirical false positives, and at least 20 positive/100 negative independently
reviewed cases **per blocking rule**. Missing review and inadequate measured
quality continue to block release. Repeated calls do not satisfy sample counts.

## Next implementation sequence

1. Freeze a separate permission decision experiment with explicit allowed,
   disallowed and uncertain outcomes. Keep factual scores independent; ambiguous
   permission must remain visible. Validate both false permission and missed
   permission before selecting any decision boundaries. The complemented-minimum
   experiment is not a production candidate.
2. Create a versioned, syntactically complete Git fixture set while preserving
   these historical inputs. Pair each corrected fixture with its predecessor,
   measure source coverage, and add unsupported/ambiguous examples. Add explicit
   insufficient-evidence handling; the current implementation records limitations
   but does not implement a general abstention classifier.
3. Refine the weakest factual predicates independently: discarded errors,
   cancellation ownership, and the specific assertion behavior removed. Preserve
   matched positive/negative tasks and evaluate each change before combining it.
4. Work through advisory rules starting with comments, boolean candidates and
   forwarders. Use deterministic syntax for facts the parser can establish;
   reserve Jev for the remaining semantic question. Broader API/architecture,
   extensibility and patch-scope changes need their own evidence experiments.
5. Freeze stable candidates, then assemble disjoint independently reviewed cases
   and calibrate on a separate development set. Run the untouched release set
   only after those prerequisites are met.

No credential setup is needed from the user. Independent label review remains a
required human input before release, rather than a blocker for the next code
experiments. The draft PR remains unmerged and nothing is published.

## Verification

Each meaningful implementation step was followed by the full `go test ./...`
suite and the formatter/vet/build/Reaper workflow before the next item. Tests
cover metadata propagation, structured state and criteria validation, unchanged
legacy requests, cache separation, experimental signal composition, snapshot
immutability, HTTP payload parity, exclusions, bounded helper retrieval and test
files exhausting the source budget. The final local full suite, `go vet ./...`,
build and Reaper check pass; Reaper reports complete analysis with no diagnostics
for the final incremental code change. Earlier step checks likewise passed.

Passing code checks validates implementation behavior, not semantic rule quality.
The earlier whole-roadmap scope check exceeded the provider context limit; these
incremental checks do not replace that missing whole-patch semantic coverage.
