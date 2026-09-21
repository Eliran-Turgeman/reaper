# Factual questions and separate permission: measured results

Measured 21 September 2026. Configurations were frozen in `8b2dd1a` before measurement. [Protocol](protocol.json). All 54 runs completed: six corpora, three conditions, three repeats. Production questions and thresholds are unchanged.

## Outcome

Factual decomposition improves several authorization and validation scores. Adding a separate permission question correctly suppresses explicitly allowed contract changes, but the tested minimum-of-complements composition also suppresses true positives. No condition is ready for production promotion.

`baseline` uses production questions with native objects; `atomic` replaces five blocking rules with factual questions plus true/false criteria and deliberately omits their task exceptions; `policy` adds a separate permission question and computes `min(factual scores, 1 − permission score)`. Atomic is a sensitivity probe, not a deployable policy. Git conditions share targeted context; seed conditions share current context. Baseline here differs from the text baseline in the simpler-question experiment.

## All corpora at unchanged thresholds

Cells show true positives / false positives across three runs. Repeats do not increase independent sample size.

| Corpus | Positive / negative cases | Baseline TP / FP | Atomic TP / FP | Policy TP / FP |
|---|---:|---:|---:|---:|
| seed | 140 / 140 | 13–16 / 3 | 14–16 / 9–12 | 5–6 / 0 |
| validation | 12 / 20 | 1 / 0 | 5 / 1–2 | 0 / 0 |
| quality-dev | 12 / 12 | 0 / 0 | 4–6 / 12 | 0 / 0 |
| git-path | 3 / 3 | 0 / 0 | 0 / 0 | 0 / 0 |
| helpers | 6 / 6 | 0 / 0 | 0 / 0 | 0 / 0 |
| contracts | 6 / 6 | 1 / 2 | 3 / 3–4 | 1 / 0 |

The 12 false positives on `quality-dev` in the atomic condition are explicit contract changes whose permission exception was deliberately removed. They are evidence that factual loss alone is insufficient, not an unexpected production regression.

## Every rule

TP / FP ranges are reported separately for every rule and corpus. A zero in all conditions is a remaining detection failure when positive cases exist.

| Corpus | Rule | Positive / negative | Baseline | Atomic | Policy |
|---|---|---:|---:|---:|---:|
| seed | boolean-mode-parameter | 10 / 10 | 0 / 0 | 0 / 0 | 0 / 0 |
| seed | caller-managed-mechanics | 10 / 10 | 0 / 0 | 0 / 0 | 0 / 0 |
| seed | ceremonial-abstraction | 10 / 10 | 0 / 0 | 0 / 0 | 0 / 0 |
| seed | implementation-detail-exposure | 10 / 10 | 0 / 0 | 0 / 0 | 0 / 0 |
| seed | named-special-case | 10 / 10 | 0 / 0 | 0 / 0 | 0 / 0 |
| seed | narrating-comment | 10 / 10 | 4–5 / 0 | 4–5 / 0 | 4–5 / 0 |
| seed | removed-authorization-check | 10 / 10 | 0 / 0 | 0 / 3–4 | 0 / 0 |
| seed | removed-validation | 10 / 10 | 0 / 0 | 0 / 3–4 | 0 / 0 |
| seed | scope-creep | 10 / 10 | 0–1 / 0 | 0–1 / 0 | 0 / 0 |
| seed | silent-failure-fallback | 10 / 10 | 0 / 0 | 2–3 / 3–4 | 0 / 0 |
| seed | swallowed-cancellation | 10 / 10 | 0 / 0 | 0 / 0 | 0 / 0 |
| seed | unchanged-argument-forwarder | 10 / 10 | 1 / 0 | 1 / 0 | 1 / 0 |
| seed | unused-extensibility-point | 10 / 10 | 0 / 0 | 0 / 0 | 0 / 0 |
| seed | weakened-test-assertion | 10 / 10 | 8–9 / 3 | 7 / 0–1 | 0 / 0 |
| validation | removed-authorization-check | 3 / 5 | 0 / 0 | 2 / 1 | 0 / 0 |
| validation | removed-validation | 3 / 5 | 0 / 0 | 2 / 0–1 | 0 / 0 |
| validation | silent-failure-fallback | 3 / 5 | 1 / 0 | 1 / 0 | 0 / 0 |
| validation | swallowed-cancellation | 3 / 5 | 0 / 0 | 0 / 0 | 0 / 0 |
| quality-dev | removed-authorization-check | 6 / 6 | 0 / 0 | 0–1 / 6 | 0 / 0 |
| quality-dev | removed-validation | 6 / 6 | 0 / 0 | 4–5 / 6 | 0 / 0 |
| git-path | removed-authorization-check | 1 / 3 | 0 / 0 | 0 / 0 | 0 / 0 |
| git-path | removed-validation | 1 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| git-path | swallowed-cancellation | 1 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| helpers | removed-authorization-check | 3 / 3 | 0 / 0 | 0 / 0 | 0 / 0 |
| helpers | removed-validation | 3 / 3 | 0 / 0 | 0 / 0 | 0 / 0 |
| contracts | silent-failure-fallback | 2 / 2 | 0 / 0 | 2 / 2 | 1 / 0 |
| contracts | swallowed-cancellation | 2 / 2 | 0 / 0 | 0 / 0 | 0 / 0 |
| contracts | weakened-test-assertion | 2 / 2 | 1 / 2 | 1 / 1–2 | 0 / 0 |

## Why findings disappear

For authorization and validation at threshold 0.95, the complement composition requires the permission answer to be at most 0.05 even when both factual answers exceed 0.95. A small estimated permission probability can therefore block a strong factual finding. This is an application decision-policy issue; Noul supplies an estimated P(yes), not a calibrated severity or violation confidence.

Using the factual answers from each **same policy request**, the following otherwise-above-threshold positive cases are missed solely because of the permission term. This calculation holds model outputs fixed; it does not mix scores from separate calls.

| Corpus | Permission-only misses, runs 1 / 2 / 3 |
|---|---:|
| seed | 9 / 10 / 10 |
| validation | 5 / 5 / 5 |
| quality-dev | 5 / 4 / 4 |
| git-path | 0 / 0 / 0 |
| helpers | 0 / 0 / 0 |
| contracts | 2 / 2 / 2 |

Other misses remain factual: the helper positives stay below 0.95 even before permission is applied; cancellation ownership and lost error propagation also receive low scores. Replacing composition alone cannot repair all failures. The next experiment should freeze a candidate-specific permission decision (with an explicit uncertain outcome) and evaluate it independently of factual detection. This report does not select new cutoffs from the observed scores.

## Coverage and limitations

All eligible cases were evaluated. The new-file authorization exclusion in `git-path` is intentionally ineligible (one negative case per run); it is not a model-confirmed safe change. Reported TP/FP counts preserve the original corpus denominator. Provider completion is not proof of sufficient source evidence. Counts below describe one policy run per corpus. Limitations can coexist and are counted once per case.

| Corpus | Languages | Cases with evidence limitations | Parse failure | Unsupported language |
|---|---|---:|---:|---:|
| seed | csharp: 56, go: 56, java: 56, python: 56, typescript: 56 | 0 | 0 | 0 |
| validation | csharp: 4, go: 9, java: 3, javascript: 4, python: 8, typescript: 4 | 32 | 9 | 23 |
| quality-dev | csharp: 4, go: 4, java: 4, javascript: 4, python: 4, typescript: 4 | 24 | 0 | 20 |
| git-path | go: 4, python: 1 | 5 | 0 | 1 |
| helpers | go: 12 | 12 | 0 | 0 |
| contracts | go: 4, java: 2, python: 4, typescript: 2 | 12 | 4 | 8 |

- Targeted resolution is bounded, lexical, Go-only, and one-hop. Parse failures, unresolved direct calls, receiver/import limitations and omitted source remain in exact request state. No general-purpose abstention classifier is implemented.
- Some historical fixtures are source fragments without complete package declarations; matched Go parsing can fail while the original diff and changed code still reach the model. Fixing these fixtures is a future paired corpus revision, not an undisclosed edit to this run.
- These are previously inspected synthetic development corpora, with overlapping templates across suites. Results are not independent held-out release evidence and must not be pooled as such.
- Rewriting factual questions and adding their criteria occur together in the atomic contrast; this phase cannot attribute that difference to wording alone. Phase A separately measured the effect of criteria.
- The nine advisory rules retain their baseline questions in this phase. Their small repeat variation is not evidence of a treatment effect.

## Exact examples

Examples below use run 1. Scores are raw estimated P(yes); the final rule score is a composition heuristic. The complete request, including criteria and helper bodies, is linked for each example. All diff blocks are the exact focal diff the model received, not reconstructed patches.

### validation / export-role

Rule: `removed-authorization-check`. Expected: **positive**. Threshold: **0.95**. [Exact run](results/validation-policy-run1.json).

Task: Simplify the export handler while retaining administrator access.

```diff
 def export(actor):
-    if "administrator" not in actor.roles:
-        raise PermissionError("admin required")
     return database.export_all()
```

| Signal | Complete question | P(yes) |
|---|---|---:|
| previous-authorization | Did the before code require authorization for an operation? | 0.98 |
| unguarded-operation | Can the after code run an operation without the authorization that guarded the same operation before the change? | 0.97 |
| task-permits-change | Does the task explicitly require the specific authorization removal shown in the change? | 0.09 |

Final scores: baseline **0.90**, atomic **0.97**, policy **0.91** → **no finding**.

### validation / invalid-port-condition

Rule: `removed-validation`. Expected: **positive**. Threshold: **0.95**. [Exact run](results/validation-policy-run1.json).

Task: Refactor port validation without relaxing the valid range.

```diff
 function connect(port: number) {
-  if (port < 1 || port > 65535) throw new RangeError("port");
+  if (port < 1 && port > 65535) throw new RangeError("port");
   return socket.connect(port);
 }
```

| Signal | Complete question | P(yes) |
|---|---|---:|
| previous-validation | Did the before code reject invalid input before an operation? | 0.98 |
| unguarded-operation | Can input rejected by the before code reach the same operation in the after code? | 0.96 |
| task-permits-change | Does the task explicitly require accepting the previously rejected inputs allowed by this change? | 0.09 |

Final scores: baseline **0.66**, atomic **0.96**, policy **0.91** → **no finding**.

### helpers / auth-helper-does-not-enforce

Rule: `removed-authorization-check`. Expected: **positive**. Threshold: **0.95**. [Exact run](results/helpers-policy-run1.json).

Task: Use the shared permission helper while preserving editor-only reads.

```diff
 package service
 func Read(actor User) (Record, error) {
- if !actor.IsEditor { return Record{}, ErrDenied }
+ if err := ensurePermission(actor); err != nil { return Record{}, err }
  return store.Read()
 }
```

| Signal | Complete question | P(yes) |
|---|---|---:|
| previous-authorization | Did the before code require authorization for an operation? | 0.94 |
| unguarded-operation | Can the after code run an operation without the authorization that guarded the same operation before the change? | 0.91 |
| task-permits-change | Does the task explicitly require the specific authorization removal shown in the change? | 0.22 |

Final scores: baseline **0.17**, atomic **0.89**, policy **0.78** → **no finding**.

### helpers / auth-helper-enforces

Rule: `removed-authorization-check`. Expected: **negative**. Threshold: **0.95**. [Exact run](results/helpers-policy-run1.json).

Task: Use the shared permission helper while preserving editor-only reads.

```diff
 package service
 func Read(actor User) (Record, error) {
- if !actor.IsEditor { return Record{}, ErrDenied }
+ if err := ensurePermission(actor); err != nil { return Record{}, err }
  return store.Read()
 }
```

| Signal | Complete question | P(yes) |
|---|---|---:|
| previous-authorization | Did the before code require authorization for an operation? | 0.95 |
| unguarded-operation | Can the after code run an operation without the authorization that guarded the same operation before the change? | 0.11 |
| task-permits-change | Does the task explicitly require the specific authorization removal shown in the change? | 0.30 |

Final scores: baseline **0.06**, atomic **0.13**, policy **0.11** → **no finding**.

### quality-dev / val-explicit-contract-go

Rule: `removed-validation`. Expected: **negative**. Threshold: **0.95**. [Exact run](results/quality-dev-policy-run1.json).

Task: Change the quantity contract to accept every integer, including zero and negative values representing inventory adjustments; remove the positive-only guard.

```diff
 package orders
 func SetQuantity(n int) error {
- if n < 1 { return ErrInvalid }
  return store.SetQuantity(n)
 }
```

| Signal | Complete question | P(yes) |
|---|---|---:|
| previous-validation | Did the before code reject invalid input before an operation? | 0.98 |
| unguarded-operation | Can input rejected by the before code reach the same operation in the after code? | 0.95 |
| task-permits-change | Does the task explicitly require accepting the previously rejected inputs allowed by this change? | 0.97 |

Final scores: baseline **0.13**, atomic **0.96**, policy **0.03** → **no finding**.

### validation / lost-read-error

Rule: `silent-failure-fallback`. Expected: **positive**. Threshold: **0.92**. [Exact run](results/validation-policy-run1.json).

Task: Simplify loading balances while preserving storage failures.

```diff
 func Balance(id string) (int, error) {
     n, err := store.Balance(id)
-    if err != nil { return 0, err }
+    if err != nil { return 0, nil }
     return n, nil
 }
```

| Signal | Complete question | P(yes) |
|---|---|---:|
| handles-failure | Does the changed code handle a failed operation or invalid condition? | 0.50 |
| failure-hidden | Does the changed failure-handling path continue or return success or a fallback without making the failure observable? | 0.96 |
| contract-permits-fallback | Does the task or supplied contract explicitly require the fallback used for the failure in this change? | 0.10 |

Final scores: baseline **0.92**, atomic **0.53**, policy **0.50** → **no finding**.

### contracts / cancel-main-query

Rule: `swallowed-cancellation`. Expected: **positive**. Threshold: **0.95**. [Exact run](results/contracts-policy-run1.json).

Task: Make the request handler more reliable.

```diff
 func Load(ctx context.Context) (Rows, error) {
- return db.Query(ctx)
+ return db.Query(context.Background())
 }
```

| Signal | Complete question | P(yes) |
|---|---|---:|
| lost-cancellation | Does the change stop passing or reporting cancellation that the before code passed or reported? | 0.94 |
| main-work-continues | Can cancellation now leave the main operation running or reported as successful? | 0.88 |
| task-permits-change | Does the task explicitly require the changed work to continue independently of caller cancellation? | 0.16 |

Final scores: baseline **0.46**, atomic **0.90**, policy **0.84** → **no finding**.

### contracts / assert-order-required

Rule: `weakened-test-assertion`. Expected: **positive**. Threshold: **0.93**. [Exact run](results/contracts-policy-run1.json).

Task: Simplify the test for ranked search keys without changing its order guarantee.

```diff
 func TestKeys(t *testing.T) {
- require.Equal(t, []string{"blue", "green", "red"}, keys())
+ require.ElementsMatch(t, []string{"blue", "green", "red"}, keys())
 }
```

| Signal | Complete question | P(yes) |
|---|---|---:|
| protection-lost | Does the after test accept an outcome its before assertions rejected, or skip behavior those assertions checked? | 0.83 |
| task-permits-change | Does the task explicitly require the specific assertion change shown between the before and after tests? | 0.23 |

Final scores: baseline **0.92**, atomic **0.82**, policy **0.77** → **no finding**.

### contracts / assert-message-retired

Rule: `weakened-test-assertion`. Expected: **negative**. Threshold: **0.93**. [Exact run](results/contracts-policy-run1.json).

Task: The legacy parser message is no longer part of the contract. Remove its assertion while retaining the required ValueError exception type.

```diff
 def test_parse():
-    with pytest.raises(ValueError, match="legacy parser failed"):
+    with pytest.raises(ValueError):
         parse(value)
```

| Signal | Complete question | P(yes) |
|---|---|---:|
| protection-lost | Does the after test accept an outcome its before assertions rejected, or skip behavior those assertions checked? | 0.96 |
| task-permits-change | Does the task explicitly require the specific assertion change shown between the before and after tests? | 0.96 |

Final scores: baseline **0.97**, atomic **0.95**, policy **0.04** → **no finding**.

## Reproducibility

All 1098 case/condition combinations retain identical request fingerprints across repeats. All 1098 atomic/policy case comparisons retain identical evidence and factual questions. Baseline uses the same state. Question IDs alone were never treated as input descriptions.

Resolved model: `typesafe/jev-1.13-20260917`. Recorded calls: 3,321; input tokens: 1,730,511; output tokens: 207,699; provider-reported cost: **$0.072681**. Elapsed request time is retained, but concurrent calls do not establish a controlled latency comparison.

Reproduce seeds with `reaper eval --eval-dir evals --benchmark-experiment <condition>-current.json --format json`. For each Git corpus use `--benchmark-dir <corpus-path> --benchmark-mode git --benchmark-grouping isolated --benchmark-experiment <condition>-targeted.json`. See the frozen protocol for paths; run each three times.

[Machine-readable summary](summary.json). All raw reports:

- seed / baseline: [run 1](results/seed-baseline-run1.json), [run 2](results/seed-baseline-run2.json), [run 3](results/seed-baseline-run3.json)
- seed / atomic: [run 1](results/seed-atomic-run1.json), [run 2](results/seed-atomic-run2.json), [run 3](results/seed-atomic-run3.json)
- seed / policy: [run 1](results/seed-policy-run1.json), [run 2](results/seed-policy-run2.json), [run 3](results/seed-policy-run3.json)
- validation / baseline: [run 1](results/validation-baseline-run1.json), [run 2](results/validation-baseline-run2.json), [run 3](results/validation-baseline-run3.json)
- validation / atomic: [run 1](results/validation-atomic-run1.json), [run 2](results/validation-atomic-run2.json), [run 3](results/validation-atomic-run3.json)
- validation / policy: [run 1](results/validation-policy-run1.json), [run 2](results/validation-policy-run2.json), [run 3](results/validation-policy-run3.json)
- quality-dev / baseline: [run 1](results/quality-dev-baseline-run1.json), [run 2](results/quality-dev-baseline-run2.json), [run 3](results/quality-dev-baseline-run3.json)
- quality-dev / atomic: [run 1](results/quality-dev-atomic-run1.json), [run 2](results/quality-dev-atomic-run2.json), [run 3](results/quality-dev-atomic-run3.json)
- quality-dev / policy: [run 1](results/quality-dev-policy-run1.json), [run 2](results/quality-dev-policy-run2.json), [run 3](results/quality-dev-policy-run3.json)
- git-path / baseline: [run 1](results/git-path-baseline-run1.json), [run 2](results/git-path-baseline-run2.json), [run 3](results/git-path-baseline-run3.json)
- git-path / atomic: [run 1](results/git-path-atomic-run1.json), [run 2](results/git-path-atomic-run2.json), [run 3](results/git-path-atomic-run3.json)
- git-path / policy: [run 1](results/git-path-policy-run1.json), [run 2](results/git-path-policy-run2.json), [run 3](results/git-path-policy-run3.json)
- helpers / baseline: [run 1](results/helpers-baseline-run1.json), [run 2](results/helpers-baseline-run2.json), [run 3](results/helpers-baseline-run3.json)
- helpers / atomic: [run 1](results/helpers-atomic-run1.json), [run 2](results/helpers-atomic-run2.json), [run 3](results/helpers-atomic-run3.json)
- helpers / policy: [run 1](results/helpers-policy-run1.json), [run 2](results/helpers-policy-run2.json), [run 3](results/helpers-policy-run3.json)
- contracts / baseline: [run 1](results/contracts-baseline-run1.json), [run 2](results/contracts-baseline-run2.json), [run 3](results/contracts-baseline-run3.json)
- contracts / atomic: [run 1](results/contracts-atomic-run1.json), [run 2](results/contracts-atomic-run2.json), [run 3](results/contracts-atomic-run3.json)
- contracts / policy: [run 1](results/contracts-policy-run1.json), [run 2](results/contracts-policy-run2.json), [run 3](results/contracts-policy-run3.json)