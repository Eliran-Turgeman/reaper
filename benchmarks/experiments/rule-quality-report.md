# Rule-quality implementation and experiment report

Measured 20 September 2026 with `openrouter / typesafe/jev-1.13`. Production prompts, thresholds and historical result baselines are unchanged. PR #1 remains a draft.

## What changed

- Existing-file insertion-only changes now reach regression rules. Real Git headers, including spaced paths, determine file eligibility.
- Git benchmarks use the same collector, extraction and runner as `check`; an HTTP parity test verifies identical requests. Coverage misses stay in recall.
- Reports retain every signal, actual grouped questions, scores and request/context fingerprints. Corpus, rule, configuration and experiment fingerprints prevent invalid new-baseline comparisons.
- Specific task requirements may permit the exact contract change; vague requests do not. This policy was confirmed by the user and is exercised by paired fixtures.
- The release workflow now requires the approved absolute target: recall >=80%, false-positive rate <=1%, and >=20 positive/100 negative independently reviewed cases per blocking rule. Missing review fails before inference. Current evidence does not meet that gate.

## Measured results at unchanged thresholds

The first real Git run on the historical 32-case validation corpus detected 1/12 positives and flagged 0/20 negatives in both isolated and configured grouping. All 32 labeled rules were evaluated. This remains inadequate detection quality. [Isolated records](production-path-v1/validation-isolated.json) and [configured records](production-path-v1/validation-configured.json) preserve the inputs and signals.

The authorization/validation experiment holds model, cases, grouping and minimum-signal composition fixed. “Context” appends before/after fixture snapshots; “both” changes questions and context. Each row below pools the same development cases, including the four multi-file follow-ups. It is not a holdout result.

| Condition | Authorization detected | Authorization false positives | Validation detected | Validation false positives |
|---|---:|---:|---:|---:|
| current | 0/11 | 0/15 | 0/11 | 0/12 |
| questions | 0/11 | 0/15 | 1/11 | 0/12 |
| context | 0/11 | 0/15 | 0/11 | 0/12 |
| both | 0/11 | 0/15 | 1/11 | 0/12 |

Scores are semantic confidence outputs; minimum composition is not a calibrated joint probability. A higher score on positives is only useful if negatives remain separated and repeated results support a threshold.

## Main findings and decisions

1. **Validation wording helps, but does not solve the default-threshold problem.** The archive traversal case moves from 0.36 to 0.91 with revised questions, and invalid port handling reaches 0.96. Specific permitted contract changes also score higher (for example the Go contract pair reaches 0.61), so do not reduce a cutoff based only on improved positives.
2. **Authorization wording is not a uniform improvement.** The insertion-only bypass scores 0.30 with revised questions in the first run and overlaps permitted changes. Keep this candidate experimental.
3. **Relevant helper context has a real measured effect.** With ordinary hunk context the good and no-op helpers score nearly alike. With both changes, the no-op authorization helper scores 0.72 versus 0.06 for its enforcing counterpart; validation scores 0.61 versus 0.06. The next production-context change should retrieve the specific referenced enforcement helper with provenance and bounded completeness. These four Go cases do not justify sending whole repositories or claiming general retrieval accuracy.
4. **Longer context alone is not universally better.** Most original fixtures already fit in a hunk; appending their full files mainly changes repetition/formatting. On the insertion-only validation bypass, revised questions/current context score 0.78, while both changes score 0.51. Preserve separate experiments.
5. **No candidate is promoted.** Default recall remains low, some question changes introduce overlap, and there is no independently reviewed holdout. Freeze a rule-specific candidate only after the remaining failure families are addressed; then review fresh cases before inference.

## Code changes and per-case scores

Each corpus link contains exact before/after code, task and label rationale. Raw result files include each constituent predicate and its score; the table reports their minimum per evaluated unit, then the maximum across units. `NE` means no labeled-rule evaluation (intentional for the new-file exclusion).

### validation

[Exact code changes](../validation/cases.json). Results: [current](auth-validation-v2/results/validation-current.json), [questions](auth-validation-v2/results/validation-questions.json), [context](auth-validation-v2/results/validation-context.json), [both](auth-validation-v2/results/validation-both.json).

| Case | Label | What changed / label basis | Current | Questions | Context | Both |
|---|---|---|---:|---:|---:|---:|
| tenant-query | positive | The tenant restriction disappears while lookup still returns the document. | 0.86 | 0.79 | 0.82 | 0.81 |
| export-role | positive | The only visible permission check disappears. | 0.92 | 0.73 | 0.88 | 0.79 |
| late-permission | positive | Checking permission after the side effect no longer prevents unauthorized archival. | 0.74 | 0.88 | 0.73 | 0.93 |
| extracted-permission | negative | The visible helper preserves the same permission check before publication. | 0.04 | 0.04 | 0.04 | 0.03 |
| owner-early-return | negative | The condition is inverted but unauthorized actors are still rejected. | 0.04 | 0.05 | 0.04 | 0.05 |
| public-catalog | negative | The explicit task authorizes making this operation public, an exclusion in the rule. | 0.17 | 0.25 | 0.16 | 0.41 |
| retired-admin-operation | negative | The protected operation is removed together with its guard. | 0.20 | 0.06 | 0.16 | 0.07 |
| authorization-error-wrap | negative | The same authorization failure still prevents key rotation. | 0.02 | 0.03 | 0.02 | 0.03 |
| positive-transfer | positive | Non-positive amounts now reach the debit operation without a visible contract. | 0.39 | 0.79 | 0.48 | 0.75 |
| archive-traversal | positive | Removing the containment check permits writes outside the extraction root. | 0.36 | 0.91 | 0.50 | 0.93 |
| invalid-port-condition | positive | The new condition cannot be true, so the range check is bypassed. | 0.67 | 0.96 | 0.70 | 0.96 |
| validator-helper | negative | The visible helper preserves input validation. | 0.04 | 0.05 | 0.03 | 0.05 |
| explicit-zero-capacity | negative | Accepting zero is an explicit requested contract change. | 0.20 | 0.51 | 0.18 | 0.57 |
| bounds-equivalence | negative | Combining the disjunction preserves both bounds. | 0.02 | 0.03 | 0.03 | 0.03 |
| removed-shell-operation | negative | No guarded operation remains, so removing the input check does not weaken execution. | 0.09 | 0.11 | 0.07 | 0.10 |
| validated-constructor | negative | The visible constructor enforces the original range before use. | 0.04 | 0.06 | 0.05 | 0.06 |
### quality-dev

[Exact code changes](../quality-dev/cases.json). Results: [current](auth-validation-v2/results/quality-dev-current.json), [questions](auth-validation-v2/results/quality-dev-questions.json), [context](auth-validation-v2/results/quality-dev-context.json), [both](auth-validation-v2/results/quality-dev-both.json).

| Case | Label | What changed / label basis | Current | Questions | Context | Both |
|---|---|---|---:|---:|---:|---:|
| auth-vague-task-go | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.43 | 0.51 | 0.56 | 0.69 |
| auth-explicit-contract-go | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.21 | 0.25 | 0.17 | 0.25 |
| val-vague-task-go | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.33 | 0.73 | 0.37 | 0.71 |
| val-explicit-contract-go | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.11 | 0.61 | 0.13 | 0.61 |
| auth-vague-task-py | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.62 | 0.69 | 0.64 | 0.75 |
| auth-explicit-contract-py | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.20 | 0.35 | 0.20 | 0.37 |
| val-vague-task-py | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.46 | 0.80 | 0.51 | 0.83 |
| val-explicit-contract-py | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.14 | 0.57 | 0.21 | 0.55 |
| auth-vague-task-ts | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.55 | 0.68 | 0.59 | 0.79 |
| auth-explicit-contract-ts | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.19 | 0.26 | 0.17 | 0.38 |
| val-vague-task-ts | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.25 | 0.89 | 0.47 | 0.86 |
| val-explicit-contract-ts | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.13 | 0.49 | 0.16 | 0.55 |
| auth-vague-task-js | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.61 | 0.73 | 0.60 | 0.74 |
| auth-explicit-contract-js | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.19 | 0.32 | 0.18 | 0.32 |
| val-vague-task-js | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.45 | 0.85 | 0.54 | 0.85 |
| val-explicit-contract-js | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.13 | 0.42 | 0.16 | 0.51 |
| auth-vague-task-java | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.57 | 0.61 | 0.55 | 0.76 |
| auth-explicit-contract-java | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.21 | 0.29 | 0.19 | 0.39 |
| val-vague-task-java | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.35 | 0.83 | 0.58 | 0.83 |
| val-explicit-contract-java | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.12 | 0.55 | 0.15 | 0.60 |
| auth-vague-task-cs | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.70 | 0.68 | 0.61 | 0.77 |
| auth-explicit-contract-cs | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.25 | 0.30 | 0.19 | 0.35 |
| val-vague-task-cs | positive | The operation remains and the only visible guard is removed; the vague simplification request does not authorize a contract change. | 0.41 | 0.83 | 0.57 | 0.84 |
| val-explicit-contract-cs | negative | The task explicitly requests this exact contract expansion; the user-approved intent policy permits it. | 0.13 | 0.56 | 0.15 | 0.54 |
### git-path

[Exact code changes](../git-path/cases.json). Results: [current](auth-validation-v2/results/git-path-current.json), [questions](auth-validation-v2/results/git-path-questions.json), [context](auth-validation-v2/results/git-path-context.json), [both](auth-validation-v2/results/git-path-both.json).

| Case | Label | What changed / label basis | Current | Questions | Context | Both |
|---|---|---|---:|---:|---:|---:|
| auth-inserted-bypass | positive | The inserted fast path reaches the same protected read before authorization. | 0.34 | 0.30 | 0.55 | 0.51 |
| validation-inserted-bypass | positive | The fast path persists values previously rejected by the nonnegative guard. | 0.44 | 0.78 | 0.33 | 0.51 |
| auth-new-file-exclusion | negative | No prior authorization behavior exists; this before/after regression rule intentionally excludes new files. | NE | NE | NE | NE |
| auth-operation-deleted | negative | The guarded operation and guard are deleted together; no protected read remains. | 0.21 | 0.06 | 0.21 | 0.06 |
| auth-equivalent-helper | negative | The visible helper still rejects the same actor before the read. | 0.03 | 0.03 | 0.03 | 0.03 |
### context-dev

[Exact code changes](../context-dev/cases.json). Results: [current](auth-validation-v2/results/context-dev-current.json), [questions](auth-validation-v2/results/context-dev-questions.json), [context](auth-validation-v2/results/context-dev-context.json), [both](auth-validation-v2/results/context-dev-both.json).

| Case | Label | What changed / label basis | Current | Questions | Context | Both |
|---|---|---|---:|---:|---:|---:|
| auth-helper-does-not-enforce | positive | The unchanged helper returns success for every input; replacing the inline guard loses enforcement. | 0.18 | 0.32 | 0.50 | 0.72 |
| auth-helper-enforces | negative | The unchanged helper enforces the same guard before the retained operation. | 0.18 | 0.33 | 0.06 | 0.06 |
| validation-helper-does-not-enforce | positive | The unchanged helper returns success for every input; replacing the inline guard loses enforcement. | 0.25 | 0.18 | 0.55 | 0.61 |
| validation-helper-enforces | negative | The unchanged helper enforces the same guard before the retained operation. | 0.21 | 0.19 | 0.06 | 0.06 |

## Cancellation, fallback and assertion contracts

These experiments use the original validation cases plus [12 explicit-contract fixtures](../contract-dev/cases.json). “Questions” changes all three candidate rule questions; “ownership” changes only the cancellation ownership predicate and uses original fallback/assertion questions. Thresholds are unchanged.

| Condition | Rule | Detected | False positives |
|---|---|---:|---:|
| current | swallowed-cancellation | 0/5 | 0/7 |
| current | silent-failure-fallback | 1/5 | 0/7 |
| current | weakened-test-assertion | 2/2 | 2/2 |
| questions | swallowed-cancellation | 0/5 | 0/7 |
| questions | silent-failure-fallback | 1/5 | 0/7 |
| questions | weakened-test-assertion | 0/2 | 0/2 |
| ownership | swallowed-cancellation | 0/5 | 0/7 |
| ownership | silent-failure-fallback | 1/5 | 0/7 |
| ownership | weakened-test-assertion | 2/2 | 2/2 |

The assertion revision separates the required order/message checks (0.72/0.77) from specifically permitted reductions (0.15/0.28). However, both positives fall below the 0.93 default. This fixes a contract ambiguity in the experiment, not production recall.

The first cancellation revision still ranks independent audit work above main-operation regressions. The simpler ownership predicate separates the tested negatives (0.04–0.13 across the two corpora) from positives (0.22–0.55), but at very low scores with a narrow margin. A low blocking cutoff is not justified by this small set.

| Contract case | Label | Current | Questions | Ownership |
|---|---|---:|---:|---:|
| cancel-main-query | positive | 0.62 | 0.52 | 0.44 |
| cancel-independent-audit | negative | 0.81 | 0.84 | 0.07 |
| cancel-success-result | positive | 0.62 | 0.46 | 0.37 |
| cancel-cleanup-rethrow | negative | 0.04 | 0.03 | 0.04 |
| fallback-required-zone | positive | 0.83 | 0.89 | 0.86 |
| fallback-optional-zone | negative | 0.36 | 0.17 | 0.33 |
| fallback-required-draft | positive | 0.72 | 0.89 | 0.69 |
| fallback-disposable-draft | negative | 0.31 | 0.23 | 0.29 |
| assert-order-required | positive | 0.95 | 0.72 | 0.95 |
| assert-order-unspecified | negative | 0.94 | 0.15 | 0.93 |
| assert-message-required | positive | 0.97 | 0.77 | 0.97 |
| assert-message-retired | negative | 0.96 | 0.28 | 0.96 |

Raw [contract results](contracts-v2/results/contract-dev-questions.json) and [validation results](contracts-v2/results/validation-questions.json) preserve every question and score; the corresponding `current` and `ownership` files retain the other conditions.

## Repeated-request stability

The [protocol](auth-validation-v2/stability-protocol.json) and seven-case subset were committed before two additional runs. Together with the original run this gives three scores per case/condition. Every repeated case has the exact same request-fingerprint multiset as its original. The exploratory cutoffs (authorization 0.40, validation 0.67) were fixed for stability probes only; they are not selected defaults.

| Condition | Case | Three scores | Probe cutoff | Classification changed? |
|---|---|---|---:|---|
| current | late-permission | 0.74, 0.69, 0.73 | 0.40 | no |
| current | explicit-zero-capacity | 0.20, 0.21, 0.21 | 0.67 | no |
| current | val-vague-task-go | 0.33, 0.33, 0.32 | 0.67 | no |
| current | val-explicit-contract-go | 0.11, 0.10, 0.11 | 0.67 | no |
| current | auth-explicit-contract-py | 0.20, 0.20, 0.22 | 0.40 | no |
| current | auth-inserted-bypass | 0.34, 0.43, 0.30 | 0.40 | yes |
| current | validation-inserted-bypass | 0.44, 0.39, 0.45 | 0.67 | no |
| questions | late-permission | 0.88, 0.84, 0.84 | 0.40 | no |
| questions | explicit-zero-capacity | 0.51, 0.54, 0.55 | 0.67 | no |
| questions | val-vague-task-go | 0.73, 0.72, 0.72 | 0.67 | no |
| questions | val-explicit-contract-go | 0.61, 0.56, 0.58 | 0.67 | no |
| questions | auth-explicit-contract-py | 0.35, 0.37, 0.40 | 0.40 | yes |
| questions | auth-inserted-bypass | 0.30, 0.42, 0.29 | 0.40 | yes |
| questions | validation-inserted-bypass | 0.78, 0.76, 0.78 | 0.67 | no |
| context | late-permission | 0.73, 0.74, 0.80 | 0.40 | no |
| context | explicit-zero-capacity | 0.18, 0.21, 0.18 | 0.67 | no |
| context | val-vague-task-go | 0.37, 0.36, 0.34 | 0.67 | no |
| context | val-explicit-contract-go | 0.13, 0.12, 0.11 | 0.67 | no |
| context | auth-explicit-contract-py | 0.20, 0.20, 0.21 | 0.40 | no |
| context | auth-inserted-bypass | 0.55, 0.57, 0.60 | 0.40 | no |
| context | validation-inserted-bypass | 0.33, 0.35, 0.36 | 0.67 | no |
| both | late-permission | 0.93, 0.92, 0.92 | 0.40 | no |
| both | explicit-zero-capacity | 0.57, 0.52, 0.61 | 0.67 | no |
| both | val-vague-task-go | 0.71, 0.71, 0.72 | 0.67 | no |
| both | val-explicit-contract-go | 0.61, 0.52, 0.51 | 0.67 | no |
| both | auth-explicit-contract-py | 0.37, 0.37, 0.35 | 0.40 | no |
| both | auth-inserted-bypass | 0.51, 0.47, 0.60 | 0.40 | no |
| both | validation-inserted-bypass | 0.51, 0.50, 0.48 | 0.67 | no |

3/28 case-condition combinations cross their exploratory cutoff in three runs. This is not an estimated deployment flip rate: the subset deliberately targets known errors and boundaries. None crosses the unchanged 0.95 defaults; these cases remain undetected or correctly unflagged there.

## Remaining work and release status

- Build bounded production retrieval for referenced enforcement helpers and test it on longer/multi-file changes; preserve unknown or incomplete context rather than treating it as proof of safety.
- Rework authorization insertion-path discrimination. Retain validation question-only and the task-aware assertion/ownership questions as candidate directions, not a universal combined prompt.
- Assemble fresh source-grounded cases disjoint by change family/repository, with at least the approved support per blocking rule. Obtain independent label/contract review; the current author cannot self-attest independence.
- Freeze each candidate prompt/context/threshold before that fresh evaluation; repeat borderline requests. Do not reset historical baselines to make a gate green.
- Keep PR #1 draft. The strict release gate intentionally fails because independent review and adequate evidence are missing. No merge, publication or blocking-threshold change has occurred.

Implementation verification: `go test ./...` passed between code steps; formatting, `go vet ./...`, build, Node Action tests and actionlint passed. Reaper completed each bounded implementation check without findings. A combined check of this quality follow-up against `03a9515` completed 77 units and 387 semantic checks with zero errors or warnings. This does not erase the earlier whole-roadmap scope-context limitation. Hosted CI and manual release validation are linked from draft PR #1; release validation must stop at the unreviewed-corpus gate.
