# Correctness and targeted-context follow-up

Measured 20 September 2026 using OpenRouter / `typesafe/jev-1.13`. Production questions, blocking thresholds, and release targets are unchanged. All new quality evidence is synthetic development data; PR #1 remains draft.

## What changed

1. Staged checks read changed-file surrounding context from index blobs. An HTTP-level regression verifies that unstaged restoration or deletion cannot alter staged requests. Deleted files have no current source; required source-read failures stop extraction.
2. Root `test/`, `tests/`, and `__tests__/` directories now receive test-rule eligibility. Similar production-directory names remain excluded.
3. Leading increment/decrement source text inside hunks is preserved, including its before/after content and existing-file modification eligibility.
4. Benchmark-only `matched` and `targeted` modes add source-located before/after Go functions and optionally one-hop direct helper candidates. Production retrieval is not promoted by this experiment.
5. A frozen follow-up question candidate asks about lost enforcement through replacement helpers, rather than treating a remaining error-shaped conditional as sufficient evidence of a check.

Implementation commits: `67a6b14`, `cf655d6`, `bd2184d`, `49fcc0c`; follow-up candidate frozen in `330c7b5`.

## Did the code fixes repair the original benchmark?

No measured recall change on this corpus. The historical build (`34c2117`) and corrected build each detect **1/12 positives with 0/20 false positives**. All **32/32 case request-fingerprint sets are identical**. Both evaluate all 32 cases. Score fluctuations under identical requests cannot be attributed to extraction fixes. The regression tests exercise the formerly broken edge cases that this corpus does not.

[Before result](results/validation-before.json), [after result](results/validation-after.json). The new `git-v2` provenance identifies the extraction/eligibility corrections; old `git-v1` baselines are preserved.

## Controlled context comparison

[Frozen protocol](protocol.json), [exact code and labels](cases/cases.json). Twelve Go development cases (6 positive / 6 negative) cover unchanged enforcement helpers, long caller functions, and helpers modified in the same patch. These are correlated derivatives, not 12 independent observations. Conditions repeat three times with unchanged thresholds and minimum-of-signals composition.

`current` = existing context; `matched` = current plus before/after functions; `targeted` = matched plus direct helper candidates; `questions` = existing context plus frozen v2 questions; `targeted-questions` = targeted plus v2 questions. `enforcement-questions` changes only the two first predicates from that last condition. [Follow-up protocol](enforcement-protocol.json).

All six conditions detect **0/6 positives**, with **0/6 false positives**, in every run at the production 0.95 thresholds. The improvement below is score separation, not an achieved production recall target.

| Case / expected | Current | Matched | Targeted | Questions only | Targeted + v2 questions | Targeted + enforcement questions |
|---|---:|---:|---:|---:|---:|---:|
| auth-helper-does-not-enforce / positive | 0.17–0.19 | 0.21–0.25 | 0.41–0.52 | 0.33–0.36 | 0.71–0.72 | 0.87–0.88 |
| auth-helper-enforces / negative | 0.18–0.21 | 0.18–0.24 | 0.10 | 0.33–0.36 | 0.08 | 0.07–0.08 |
| validation-helper-does-not-enforce / positive | 0.19–0.22 | 0.21–0.25 | 0.58–0.61 | 0.17–0.19 | 0.72–0.83 | 0.86–0.88 |
| validation-helper-enforces / negative | 0.19–0.20 | 0.22–0.26 | 0.08 | 0.18–0.20 | 0.07–0.08 | 0.06–0.07 |
| auth-helper-does-not-enforce-long / positive | 0.20–0.23 | 0.20–0.22 | 0.49–0.57 | 0.29–0.30 | 0.52–0.58 | 0.86–0.87 |
| auth-helper-does-not-enforce-changed-helper / positive | 0.78–0.81 | 0.80–0.82 | 0.81–0.82 | 0.82–0.84 | 0.71–0.75 | 0.88–0.89 |
| auth-helper-enforces-long / negative | 0.17–0.20 | 0.19–0.20 | 0.07–0.08 | 0.28–0.29 | 0.06–0.07 | 0.06–0.07 |
| auth-helper-enforces-changed-helper / negative | 0.17–0.18 | 0.18–0.22 | 0.10–0.12 | 0.31–0.37 | 0.08 | 0.08–0.09 |
| validation-helper-does-not-enforce-long / positive | 0.17–0.19 | 0.19–0.21 | 0.39–0.49 | 0.18 | 0.36–0.42 | 0.86–0.87 |
| validation-helper-does-not-enforce-changed-helper / positive | 0.45–0.47 | 0.46–0.48 | 0.60–0.61 | 0.77–0.78 | 0.73–0.80 | 0.87 |
| validation-helper-enforces-long / negative | 0.19–0.20 | 0.17–0.20 | 0.07 | 0.18 | 0.07–0.08 | 0.06 |
| validation-helper-enforces-changed-helper / negative | 0.19–0.24 | 0.21–0.24 | 0.07–0.08 | 0.18–0.21 | 0.06–0.07 | 0.06–0.07 |

Each cell is the range across three runs. Raw results contain every constituent signal, confidence score, focal file/line, actual question, and request fingerprint:

- current: [run 1](results/context-current-run1.json), [run 2](results/context-current-run2.json), [run 3](results/context-current-run3.json)
- matched: [run 1](results/context-matched-run1.json), [run 2](results/context-matched-run2.json), [run 3](results/context-matched-run3.json)
- targeted: [run 1](results/context-targeted-run1.json), [run 2](results/context-targeted-run2.json), [run 3](results/context-targeted-run3.json)
- questions: [run 1](results/context-questions-run1.json), [run 2](results/context-questions-run2.json), [run 3](results/context-questions-run3.json)
- targeted-questions: [run 1](results/context-targeted-questions-run1.json), [run 2](results/context-targeted-questions-run2.json), [run 3](results/context-targeted-questions-run3.json)
- enforcement-questions: [run 1](results/context-enforcement-questions-run1.json), [run 2](results/context-enforcement-questions-run2.json), [run 3](results/context-enforcement-questions-run3.json)

## What the signals show

Matched functions restore the old guarded operation when it falls outside a hunk, but they do not reveal unchanged helpers. Targeted helper bodies supply that missing evidence and lower scores for safe equivalents. Adding functions without the relevant helper is insufficient.

In the first targeted + v2 run, the long authorization example scores 0.87 for lost protection but only 0.55 for removal; long validation scores 0.86 for the remaining unsafe operation but 0.36 for removal. The minimum composition makes the weaker first predicate decisive. The enforcement-focused wording raises the final long-case scores to 0.86–0.87 while retaining low scores on their safe pairs.

This supports a concrete diagnosis: some questions confuse preserving the syntax of an error check with preserving the enforcement behind it. It does not establish a calibrated probability or justify changing the cutoff from these cases alone.

All 72/72 context case/condition combinations use identical request fingerprints across the three repeats. The old and new question conditions also have identical state-fingerprint sets. Variation therefore remains attributable to evaluator behavior under the same normalized inputs, rather than context drift.

## Broader regression check for the question candidate

Both conditions use targeted context. Only the two first-predicate questions differ. Results below keep each rule’s existing threshold; ranges cover three repeated runs. Non-Go or unparseable Go fixture fragments retain existing local evidence with an explicit limitation note.

| Corpus | Rule | Positives / negatives | v2 TP / FP | Enforcement TP / FP |
|---|---|---:|---:|---:|
| validation | removed-authorization-check | 3 / 5 | 0 / 0 | 0 / 0 |
| validation | removed-validation | 3 / 5 | 1 / 0 | 1 / 0 |
| quality-dev | removed-authorization-check | 6 / 6 | 0 / 0 | 0 / 0 |
| quality-dev | removed-validation | 6 / 6 | 0 / 0 | 0 / 0 |
| git-path | removed-authorization-check | 1 / 3 | 0 / 0 | 0 / 0 |
| git-path | removed-validation | 1 / 0 | 0 / 0 | 0 / 0 |

Raw broader results:

- validation / targeted-questions: [run 1](results/validation-targeted-questions-run1.json), [run 2](results/validation-targeted-questions-run2.json), [run 3](results/validation-targeted-questions-run3.json)
- validation / enforcement-questions: [run 1](results/validation-enforcement-questions-run1.json), [run 2](results/validation-enforcement-questions-run2.json), [run 3](results/validation-enforcement-questions-run3.json)
- quality-dev / targeted-questions: [run 1](results/quality-dev-targeted-questions-run1.json), [run 2](results/quality-dev-targeted-questions-run2.json), [run 3](results/quality-dev-targeted-questions-run3.json)
- quality-dev / enforcement-questions: [run 1](results/quality-dev-enforcement-questions-run1.json), [run 2](results/quality-dev-enforcement-questions-run2.json), [run 3](results/quality-dev-enforcement-questions-run3.json)
- git-path / targeted-questions: [run 1](results/git-path-targeted-questions-run1.json), [run 2](results/git-path-targeted-questions-run2.json), [run 3](results/git-path-targeted-questions-run3.json)
- git-path / enforcement-questions: [run 1](results/git-path-enforcement-questions-run1.json), [run 2](results/git-path-enforcement-questions-run2.json), [run 3](results/git-path-enforcement-questions-run3.json)

## Calibration limits and next decisions

The helper subset now separates cleanly below 0.95, but it was used to develop the wording. It cannot validate a new threshold. The following conservative development-only check pools the context, original validation, task-contract and Git-path corpora: a positive counts as detected only if all three runs exceed the candidate cutoff; a negative counts as a false positive if any run exceeds it. Cases are counted once, not three times. Correlated templates still prevent treating this as independent release evidence.

| Rule | Positive / negative cases | Lowest positive score | Highest negative score | Best robust recall at ≤1% empirical FPR |
|---|---:|---:|---:|---:|
| removed-authorization-check | 13 / 17 | 0.61 | 0.44 | 13/13 (100.0%) |
| removed-validation | 13 / 14 | 0.60 | 0.62 | 12/13 (92.3%) |

These are post-hoc development calculations, not selected production thresholds. Keep the approved release policy: ≥80% recall, ≤1% false positives, and ≥20 positive/100 negative independently reviewed cases per blocking rule. No threshold was lowered and no review attestation was fabricated.

The exploratory cutoffs producing those robust counts are 0.61 for authorization
and 0.75 for validation; they were selected after examining these scores and are
not recommendations for production. One authorization negative is the intentional
new-file exclusion and is not evaluated; the other cases in this table are
evaluated. This is coverage behavior, not evidence of a model correctly assessing
that excluded case.

The remaining validation overlap is concrete: `validation-inserted-bypass`
falls to 0.60, while `val-explicit-contract-go` and `val-explicit-contract-py`
reach 0.62 despite their tasks explicitly permitting the change. A cutoff alone
cannot classify all three correctly across these runs. Authorization's weakest
positive is the C# vague-task example at 0.61, so fresh language-diverse cases
are particularly important before treating the apparent gap as dependable.

Next: test the enforcement-focused candidate on fresh independently labeled authorization/validation cases, particularly real helper extraction, explicit permission changes, and multi-file negative examples. Resolve cases where positive and permitted-change scores overlap before selecting a cutoff. Apply the same isolated process to cancellation ownership and weakened assertions; this phase does not establish that those rules are repaired.

## Boundaries and verification

The new evidence collector is an experiment over explicit fixture snapshots, not production repository retrieval. It uses Go syntax, same-directory/package direct functions, and one call hop; it does not resolve receiver methods, imported helpers, function values, build tags, transitive calls, or callers of a changed helper. Limits and parse failures are explicit, and evidence never includes gold labels. Long fixtures also contain unresolved tracing calls, so they are not proof of complete program semantics. Before promoting retrieval, implement snapshot-consistent source access and exclusions for all evidence sources, including staged repository search.

The experiment retains focal file locations and sends one bounded evidence package per unit; it does not convert rules to ScopePatch. Raw reports preserve actual request fingerprints, but do not record provider token usage or per-run latency, so no token-cost or speed improvement is claimed.

Verification: full `go test ./...` passed after each correctness fix and after the experiment implementation; formatting, `go vet ./...`, build, Node Action tests and actionlint passed. Reaper completed the three fixes and context implementation with zero warnings/errors. Hosted final-head verification is recorded on draft PR #1. Releases remain blocked pending independent corpus review and sufficient quality evidence.
