# Cancellation, fallback and test-contract development cases

These 12 synthetic cases have six positive and six negative labels across three
rules. They make the disputed requirements explicit in the task, under the policy
confirmed by the user: a specific contract change may be permitted; a vague request
does not permit it. Rationale and expected labels remain private to scoring.

Cancellation pairs cover request-owned main queries versus explicitly independent
audit writes, and successful returns after cancellation versus cleanup/rethrow.
Fallback pairs use identical code under required billing/authoritative storage
contracts versus optional display/disposable preview policies. Assertion pairs use
identical code under required order/message contracts versus specifically retired
requirements. These are tests of contract discrimination, not independently labeled
production incidents. Historical ambiguous seeds remain unchanged.

Compare `experiments/contracts-v2/current.json` and `questions.json` with Git mode
and isolated grouping. The candidate keeps the existing minimum composition and
all default thresholds. It changes only benchmark questions; no production rule
is promoted by this experiment. Evaluate original validation cancellation/fallback
cases too, because improvement on newly paired fixtures alone is insufficient.

The first revised cancellation wording continued to rank independent audit work
above main-operation regressions. `ownership.json` is a subsequent development
experiment: retain the original propagation predicate and ask the second predicate
only whether the work shares the caller's cancellation lifetime. It is not a frozen
holdout candidate or a production rule change.
