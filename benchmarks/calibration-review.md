# Calibration review

Computed from the existing measured scores using Reaper’s threshold sweep with minimum recall 0.6. No new inference requests were made and no default thresholds were changed.

These are development-set candidates, not recommendations ready for release. The patch pilot has eight examples per rule and shares templates; the seed corpus has 20 examples per rule. Independent validation is required before changing blocking defaults.

| Corpus | Rule | Candidate threshold | Precision | Recall | Examples |
|---|---|---:|---:|---:|---:|
| Patch pilot | `removed-authorization-check` | 0.84 | 100.0% | 100.0% | 8 |
| Patch pilot | `removed-validation` | 0.53 | 100.0% | 100.0% | 8 |
| Patch pilot | `silent-failure-fallback` | 0.19 | 100.0% | 100.0% | 8 |
| Patch pilot | `swallowed-cancellation` | 0.38 | 100.0% | 100.0% | 8 |
| Seed corpus | `boolean-mode-parameter` | 0.38 | 100.0% | 60.0% | 20 |
| Seed corpus | `caller-managed-mechanics` | 0.25 | 88.9% | 80.0% | 20 |
| Seed corpus | `ceremonial-abstraction` | 0.30 | 80.0% | 80.0% | 20 |
| Seed corpus | `implementation-detail-exposure` | 0.40 | 100.0% | 80.0% | 20 |
| Seed corpus | `named-special-case` | 0.60 | 100.0% | 100.0% | 20 |
| Seed corpus | `narrating-comment` | 0.83 | 100.0% | 100.0% | 20 |
| Seed corpus | `removed-authorization-check` | 0.85 | 100.0% | 100.0% | 20 |
| Seed corpus | `removed-validation` | 0.52 | 100.0% | 100.0% | 20 |
| Seed corpus | `scope-creep` | 0.16 | 85.7% | 60.0% | 20 |
| Seed corpus | `silent-failure-fallback` | 0.39 | 63.6% | 70.0% | 20 |
| Seed corpus | `swallowed-cancellation` | 0.66 | 90.0% | 90.0% | 20 |
| Seed corpus | `unchanged-argument-forwarder` | 0.55 | 100.0% | 90.0% | 20 |
| Seed corpus | `unused-extensibility-point` | 0.09 | 58.8% | 100.0% | 20 |
| Seed corpus | `weakened-test-assertion` | 0.97 | 85.7% | 60.0% | 20 |

## Release decisions

- Do not lower all thresholds together. In the seed corpus, the best qualifying silent-failure candidate has only 63.6% precision and unused-extensibility has 58.8%; these are unsuitable as evidence for stricter blocking defaults.
- Removed authorization and validation are promising candidates for independent validation. Their two development corpora suggest similar thresholds, but neither measures real repository base rates.
- Keep current defaults until a separately labeled validation corpus supports a reviewed change. Do not recalibrate on a hidden test set.
- The existing full-patch scope check remains incomplete when the provider rejects its context size; group-level checks do not establish whole-patch coverage.
