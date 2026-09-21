# Simpler questions, state representation, and criteria

Measured 21 September 2026 with the configurations frozen in `a975506`. These are existing synthetic development cases; all production questions and thresholds remain unchanged. [Frozen protocol](protocol.json).

## Outcome

The shorter instructions use **620 words instead of 940 (34% fewer)** across 28 signals. Shorter wording alone did not reliably improve detection. Explicit criteria increased detections on a few rules but also increased false positives. These results do not support blanket production promotion.

Each condition ran three times. The seed corpus has 140 positive/140 negative examples across all 14 rules; the Git validation corpus has 12 positive/20 negative cases. Repeats measure score variability, not independent sample size.

| Corpus | Condition | True positives | False positives |
|---|---|---:|---:|
| seed | current | 11–13 | 3–4 |
| seed | wording | 7–9 | 1–2 |
| seed | json-text | 6 | 2 |
| seed | json-object | 7–9 | 1–2 |
| seed | criteria | 15–17 | 5 |
| validation | current | 1 | 0 |
| validation | wording | 0 | 0 |
| validation | json-text | 0 | 0 |
| validation | json-object | 0–1 | 0 |
| validation | criteria | 1 | 0 |

`current` uses original instructions and text. `wording` changes instructions only. `json-text` uses the same shorter questions with named JSON fields serialized as text. `json-object` sends those identical fields as a native object. `criteria` adds explicit true/false descriptions to the object condition, leaving instructions and evidence unchanged.

## Every rule: seed results

Cells show true positives / false positives across the three runs. Each rule has 10 positive and 10 negative seed examples. Zero findings do not establish correctness.

| Rule | Current | Wording | JSON text | JSON object | Criteria |
|---|---:|---:|---:|---:|---:|
| boolean-mode-parameter | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| caller-managed-mechanics | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| ceremonial-abstraction | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| implementation-detail-exposure | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| named-special-case | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| narrating-comment | 2–4 / 0 | 0 / 0 | 0 / 0 | 0–1 / 0 | 4 / 0 |
| removed-authorization-check | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| removed-validation | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| scope-creep | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| silent-failure-fallback | 0 / 0 | 0–1 / 0 | 0 / 0 | 0 / 0 | 0–1 / 1 |
| swallowed-cancellation | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| unchanged-argument-forwarder | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 2–4 / 0 |
| unused-extensibility-point | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| weakened-test-assertion | 8–9 / 3–4 | 7–8 / 1–2 | 6 / 2 | 7–8 / 1–2 | 8–9 / 4 |

## Interpretation

- The current seed detections mostly come from weakened assertions and narrating comments. Many rules remain below their thresholds even on positive examples.
- Shortening the assertion question reduces some false positives but loses some true positives. Criteria increase assertion false positives and introduce a silent-fallback false positive.
- Native objects are supported and useful for naming evidence, but changing serialization alone is not an accuracy fix.
- The validation Git corpus stays at zero or one detected positive out of twelve. The subsequent [factual and permission experiment](../atomic-policy-v1/report.md) measures those changes separately from this phase.
- No threshold was selected or lowered. Seed and Git measurements are different protocols and their scores should not be pooled as independent evidence.

## Input integrity and operational measurements

All 1560 case/condition combinations have identical request fingerprints across repeats. All 936 matched wording comparisons preserve state fingerprints. All 936 object/text comparisons preserve decoded facts; the same number of criteria comparisons preserve state and question instructions.

Resolved model: `typesafe/jev-1.13-20260917`. Total recorded requests: 4,680; input tokens: 2,023,761; output tokens: 280,800; provider-reported cost: **$0.084998**. Raw request records include elapsed time, exact state, questions/criteria, scores, model identity and usage. These concurrent runs are not latency benchmarks.

## Exact artifacts

[Current](current.json), [wording](wording.json), [JSON text](json-text.json), [JSON object](json-object.json), [criteria](criteria.json), [machine-readable summary](summary.json).

- seed / current: [run 1](results/seed-current-run1.json), [run 2](results/seed-current-run2.json), [run 3](results/seed-current-run3.json)
- seed / wording: [run 1](results/seed-wording-run1.json), [run 2](results/seed-wording-run2.json), [run 3](results/seed-wording-run3.json)
- seed / json-text: [run 1](results/seed-json-text-run1.json), [run 2](results/seed-json-text-run2.json), [run 3](results/seed-json-text-run3.json)
- seed / json-object: [run 1](results/seed-json-object-run1.json), [run 2](results/seed-json-object-run2.json), [run 3](results/seed-json-object-run3.json)
- seed / criteria: [run 1](results/seed-criteria-run1.json), [run 2](results/seed-criteria-run2.json), [run 3](results/seed-criteria-run3.json)
- validation / current: [run 1](results/validation-current-run1.json), [run 2](results/validation-current-run2.json), [run 3](results/validation-current-run3.json)
- validation / wording: [run 1](results/validation-wording-run1.json), [run 2](results/validation-wording-run2.json), [run 3](results/validation-wording-run3.json)
- validation / json-text: [run 1](results/validation-json-text-run1.json), [run 2](results/validation-json-text-run2.json), [run 3](results/validation-json-text-run3.json)
- validation / json-object: [run 1](results/validation-json-object-run1.json), [run 2](results/validation-json-object-run2.json), [run 3](results/validation-json-object-run3.json)
- validation / criteria: [run 1](results/validation-criteria-run1.json), [run 2](results/validation-criteria-run2.json), [run 3](results/validation-criteria-run3.json)

Reproduce seed runs with `reaper eval --eval-dir evals --benchmark-experiment <configuration.json> --format json`. For Git validation use `--benchmark-dir benchmarks/validation --benchmark-mode git --benchmark-grouping isolated` instead of `--eval-dir`. Keep the frozen configurations and run each three times.
