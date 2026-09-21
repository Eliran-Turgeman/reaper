# Jev capabilities and complete rule audit

Research date: 21 September 2026. Audited Reaper revision: `85cf044d91a3619d52e01e93fd9abb2cd15f4482`.

This is a design review of all **14 rules and 28 production signals**, supported by vendor documentation, local source inspection, existing benchmark evidence, and three live API smoke requests. The proposed questions below are **unmeasured design candidates**, not production replacements. No rule, threshold, release target, or production context collector changed in this audit.

## Main finding

Simpler questions are the right direction, but we should simplify the *decision* as well as the sentence. Reaper currently asks Jev to identify the relevant code, compare versions, infer missing contracts, apply task exceptions, and sometimes reason about repository-wide absence within a single question. Some of that belongs in evidence collection or ordinary code.

The proposed division of work is:

1. Reaper selects a concrete candidate and supplies source-located evidence from consistent before/after snapshots.
2. Jev answers a short, self-contained semantic question about that evidence.
3. Reaper combines factual judgments, separately evaluates task permission where applicable, and reports insufficient evidence explicitly.
4. Thresholds are calibrated after the questions, evidence, and composition are stable.

The user's observation about `reduces-visible-protection` is correct: Jev does not go looking through the repository. Our request contains no tools. The instruction about not inferring coverage elsewhere limits speculation; it does not stop retrieval. We can replace that broad warning with a question about the exact behavior checked by the changed test, and supply a moved assertion helper when relevant.

## 1. What Jev can do, and what we actually use

Jev is a structured decision model. Its input is supplied state plus questions; its outputs are decisions and probabilities, rather than generated code or explanations. Repository search and workflow orchestration remain our responsibility. [TypeSafe: System One](https://docs.typesafe.ai/concepts/system-one).

| Capability | Documented behavior | Reaper today | Recommended use |
|---|---|---|---|
| State | Text, JSON object, or array | A single formatted string | Named evidence fields; test whether native objects help accuracy |
| Noul | Probability that a yes/no proposition is true | Every signal uses Noul | Atomic factual judgments |
| Yes/no criteria | Separate descriptions of true and false outcomes | Wire struct has a criteria field, but requests never populate it | Short questions with explicit boundary cases |
| Choice | Select among named alternatives, with probabilities | Unsupported by our public adapter | Optional experiment for mutually exclusive policy or evidence states |
| Score | Distribution over ordered descriptions | Unsupported | No immediate need for blocking rules |
| Batched questions | Several questions share state | Used for local rules | Keep batching where evidence is genuinely shared |
| Result metadata | Resolved model, usage; OpenRouter also returns provider, ID, cost | Resolved model is parsed then discarded; token totals retained, cost discarded | Reproducible benchmark provenance |

The state and question variants above are explicitly supported by the **OpenRouter Decisions endpoint we use**, not just TypeSafe's direct API. [OpenRouter API schema](https://openrouter.ai/docs/api/api-reference/alphadecisions/submit-a-decisions-questions-and-answers-request).

Two details materially affect question design:

- **Question IDs are not sent to the model.** An ID such as `violates-cancellation-contract` cannot supply meaning absent from its instructions.
- **Questions are evaluated independently.** A question cannot depend on another question identifying an input, operation, abstraction, or exception. Use a named state field or an explicit noun phrase instead of “that input.” Independent evaluation also does not make the underlying events statistically independent. [TypeSafe: question primitives](https://docs.typesafe.ai/primitives).

State should hold facts; questions should state the judgment. Structured fields help distinguish the task, source, version, candidate, and related evidence. They do not create missing facts or guarantee improved accuracy. [TypeSafe: state](https://docs.typesafe.ai/concepts/state).

### Live compatibility check

All three requests to OpenRouter returned HTTP 200 and resolved to `typesafe/jev-1.13-20260917`. The synthetic example removes a negative-value guard while the task says to retain it. [Exact requests and responses](../benchmarks/experiments/jev-capabilities-v1/probe.json).

| Request | Observed result | Input / output tokens | Elapsed seconds |
|---|---|---:|---:|
| Flat state, one Noul | Negative value reaches `store`: 0.97 | 364 / 22 | 1.062 |
| Object state, Noul with true/false criteria | Same proposition: 0.97 | 414 / 22 | 0.829 |
| Object state, mixed Noul/Choice/Score | Noul 0.97; task choice `reject`, probability 0.98; guard score 0 of 2 | 552 / 74 | 0.296 |

These establish API compatibility only. Each condition ran once; the second changes both state format and criteria, and the third adds questions. This is not a quality comparison, calibration study, batching-invariance test, or latency benchmark. Reported costs sum to $0.00005586. No credential or authorization header is stored in the artifact.

### Limitations that matter for code review

The vendor documents weaknesses in complex multi-step reasoning, negation, exact counting, arithmetic, and large irrelevant contexts. It also warns that input text can influence decisions as instructions, and logically related questions need not return mathematically consistent probabilities. These limitations support shorter propositions, deterministic extraction/counting, focused evidence, and adversarial source-comment tests. They do not establish the cause of any individual benchmark score. [TypeSafe: Jev 1.13 limitations](https://docs.typesafe.ai/model-jaggedness/jev-1.13).

The direct model documentation lists 64k total state plus all questions and 32k for state plus the longest question. OpenRouter's model page lists a 32k context window. We have not tested the boundary behavior of our route. Use conservative request budgets and explicit overflow handling; the maximum accepted context is not a target context size. [TypeSafe models](https://docs.typesafe.ai/models), [OpenRouter model](https://openrouter.ai/typesafe/jev-1.13/).

### Correcting our use of “confidence”

A Noul value is the model's estimated **P(yes)**, not a separate confidence field or severity rating. A value near zero strongly supports “no”; 0.5 indicates uncertainty about the proposition. We have not established calibration on our code-review distribution. [TypeSafe: Noul](https://docs.typesafe.ai/primitives/noul).

This matters for the current authorization instruction “Return low confidence”: the intended direction is a low probability of lost protection, not low certainty. An equivalent enforcing helper should support a clear “no.” Replace that phrase with an explicit false criterion.

Reaper currently uses the minimum of a rule's signal values as its final score. That is an application decision score, **not a calibrated probability that the rule was violated**. Even perfectly calibrated marginal probabilities would not make their minimum the joint probability. Multiplying them would also require an unjustified independence assumption. Choice/Score confidence fields use their own distribution-based semantics and should not inherit Noul thresholds. [TypeSafe: confidence](https://docs.typesafe.ai/confidence).

Our diagnostic signal `Evidence` is the question text. It is not an explanation generated by Jev and does not tell us why Jev chose a score. Reports should distinguish the question, supplied source evidence, model score, and our interpretation.

## 2. What the benchmark actually establishes

The latest [targeted-context report](../benchmarks/experiments/targeted-context-v3/report.md) shows:

- Correctness fixes did not change requests on the original corpus: 32/32 case fingerprint sets matched; both builds detected 1/12 positives with 0/20 false positives.
- Relevant helper bodies plus enforcement-focused questions separated the 12 helper development cases well: positive rule scores approximately 0.86–0.89, safe counterparts 0.06–0.09. All still missed the 0.95 blocking threshold. This is better separation, not successful recall.
- Matched function bodies alone were insufficient when the decisive check lived in an unchanged helper.
- In the earlier question candidate, a long authorization case scored 0.87 on lost protection but 0.55 on removal; validation scored 0.86 versus 0.36. The minimum made the ambiguous removal question decisive.
- A genuine inserted validation bypass scores as low as 0.60, while explicitly authorized validation removals reach 0.62. No single cutoff correctly separates those examples across the observed repeats.

These observations support improving enforcement evidence and separating factual change from task permission. They do **not** prove that JSON state, criteria, Choice, or every shorter rewrite will improve results. Those remain experiments.

Additional unresolved examples include dropped fetch cancellation (0.57 rule score at a 0.95 threshold) and a historical assertion case that removed a task-retired error message (0.96 at a 0.93 threshold, a false positive). See [validation results](../benchmarks/experiments/targeted-context-v3/results/validation-after.json) and [contract results](../benchmarks/experiments/contracts-v2/results/contract-dev-current.json). These are different experiment conditions and should not be treated as one current-model ranking.

## 3. Context and adapter audit

| Finding in current code | Consequence | Proposed correction |
|---|---|---|
| `decision.Question` only exposes ID/instructions; state is a string | Cannot express native structured state or Noul criteria through the application | Add these capabilities while retaining the existing request path for controlled comparisons |
| `jev.HTTPClient` forces every question to Noul | Choice/Score require adapter work despite endpoint support | Start with Noul criteria; defer additional primitives until an experiment needs them |
| State repeats diff, old/new hunk text, and surrounding code | More input without necessarily more relevant evidence | Canonical before/after bodies plus changed spans and targeted related bodies |
| Old content is a hunk; expanded surrounding code is current-only | Asymmetric comparison can omit the original guard or operation | Match before and after declarations from the selected snapshots |
| Extensibility search reads working-tree files | Staged checks can mix index focal code with unstaged repository evidence | Read all related evidence from the same snapshot as the focal code |
| Repository search returns bounded matching lines | Search results cannot establish only one implementation or no external use | Label coverage; resolve relevant definitions; restrict absence claims to enumerable closed scopes |
| Targeted Go retrieval exists only in fixture experiments | Useful experimental results do not imply production retrieval is repaired | Promote narrowly with snapshot, resolution, budget, and language-fallback tests |
| Context limit is discovered through provider failure | A large patch can fail instead of yielding useful scope findings | Budget before dispatch; group scope evidence with explicit coverage tracking |
| Cache/provenance reflects current string questions and state | New criteria/types must not reuse old semantic results | Include criteria, type, schema, composition, context mode, and versions in identity |

Source references: [decision API](../internal/decision/decision.go), [HTTP adapter](../internal/jev/http.go), [rule questions](../internal/rules/questions.go), [rule composition](../internal/rules/rules.go), [runner](../internal/runner/runner.go), [repository context](../internal/runner/context.go), [diff extraction](../internal/diff/diff.go), [experimental helper evidence](../internal/eval/targeted_context.go).

### Proposed evidence contract

This illustrative object is a proposed schema, not an implemented API:

```json
{
  "task": {"text": "Keep rejecting negative values."},
  "candidate": {
    "file": "save.go",
    "symbol": "Save",
    "operation": "store(value)",
    "condition": "value < 0"
  },
  "before": {"snapshot": "base-commit", "code": "..."},
  "after": {"snapshot": "index-tree", "code": "..."},
  "changes": [{"before_lines": [12, 14], "after_lines": [12, 12]}],
  "related": [{"symbol": "store", "snapshot": "index-tree", "file": "store.go", "code": "..."}],
  "coverage": {"before_complete": true, "after_complete": true, "unresolved_calls": [], "truncated": false}
}
```

The operation and condition are source references or extracted expressions, not model-invented summaries or gold labels. If extraction cannot identify them reliably, use a supported fallback or record an unresolved candidate. Do not silently invent a contract such as “this is protected” from an identifier alone.

Each rule should define its required evidence. Missing required evidence produces an explicit insufficient-evidence outcome, not a zero violation score and not an assertion that the change is safe. A nonblocking warning and a release-quality coverage gate may use that outcome differently; this is a visible behavior change that needs its own documented implementation.

Start with focal before/after bodies; add only evidence that can resolve the actual question: a guard helper, assertion helper, boundary declaration, caller obligation, or task clause. Supply locations and snapshot identity. Keep unresolved imports, methods, dynamic dispatch, build constraints, and truncation visible. A complete function is not proof of complete program behavior.

One shared state can serve several independent questions. Do not attach unrelated repository material merely because it fits. A rule that needs different evidence can use a separate batch. Source code and comments are evidence; task text is policy input. Test source comments that try to redefine either.

## 4. Sweep of every rule and signal

The [production snapshot](../benchmarks/experiments/jev-capabilities-v1/current-rules.json) contains all 28 exact current instructions, versions, thresholds, scopes, and severities. Every current signal ID appears below.

Questions use proposed named fields that must be populated before evaluation. They are not drop-in replacements for the existing flat state. “Code gate” means move a reliably computable fact out of the model for supported syntax, while explicitly retaining unsupported-language coverage as a limitation. It does not mean pretend we have a universal parser.

Some proposals reverse polarity or split one signal into several. Those require new composition and versioned evaluation; feeding them into today's unchanged minimum would be wrong. Boundary descriptions below are candidate true/false criteria. They should preserve policy, not merely move a long prompt into another field.

### 4.1 `removed-authorization-check` — error, threshold 0.95

**Current problem:** the first signal bundles identifying old authorization with detecting removal/bypass. The second bundles missing replacement enforcement, retained operation, and explicit permission. A helper name or remaining `if err != nil` can be mistaken for enforcement.

| Current signal | Proposed treatment |
|---|---|
| `removes-authorization` | Split: “Does `before.code` require `candidate.permission` for `candidate.operation`?” and “Can `after.code` run `candidate.operation` without `candidate.permission`?” |
| `loses-protection` | Separate operation-presence evidence, replacement enforcement, and the task exception. Ask “Does `task.text` explicitly require removing `candidate.permission` from `candidate.operation`?” |

**Final self-contained enforcement wording:** “Can `after.code` run `candidate.operation` without `candidate.permission`?” True means a feasible path bypasses the prior permission requirement; false means every supplied relevant path still enforces it. Unknown helper semantics are insufficient evidence, not false. Authentication alone is not permission.

**Context:** before/after caller bodies; the particular actor/permission/operation references; changed or unchanged guard helpers; task text. Include middleware only when a supported relationship ties it to this operation. A function renamed `PublicCatalog` is not itself proof of task permission.

**Policy composition:** old permission requirement + operation retained + lost enforcement + no explicit task authorization. Evaluate task permission separately; clear task authorization vetoes the finding. An ambiguous permission judgment should be surfaced for review, not silently treated as either a violation or permission.

**Required pairs:** deletion versus enforcing helper extraction; inert helper versus enforcing helper; operation removed versus retained; explicitly public operation versus vague simplification; same names with different implementations; C# and other language cases.

### 4.2 `removed-validation` — error, threshold 0.95

| Current signal | Proposed treatment |
|---|---|
| `removes-validation` | Split: “Does `before.code` reject `candidate.invalid_input` before `candidate.operation`?” and “Can `candidate.invalid_input` reach `candidate.operation` in `after.code`?” |
| `retains-unsafe-operation` | Gate retained-operation identity from evidence; separately ask “Does `task.text` explicitly require accepting `candidate.invalid_input`?” |

**Boundary:** rejecting an input in one remaining branch is insufficient when a new fast path reaches the operation first. Conversely, a supported enforced type or equivalent validator can preserve the condition. Evaluate the concrete invalid-input class from the old check, not whether an operation vaguely looks “unsafe.” An AST can locate the changed branch; it cannot generally prove path feasibility.

**Context:** matched bodies, changed branch order, exact old predicate, relevant helper/type enforcement, operation identity, and task. Keep syntactic location facts separate from semantic reachability judgments.

**Required pairs:** inserted bypass versus reordered equivalent checks; explicit acceptance requirement versus “simplify”; equivalent validator versus pass-through validator; checked type versus unenforced annotation. The 0.60 bypass / 0.62 permitted-removal overlap is the first regression to resolve.

### 4.3 `swallowed-cancellation` — error, threshold 0.95

| Current signal | Proposed treatment |
|---|---|
| `discards-cancellation` | “Does `after.code` stop passing or reporting `candidate.cancellation` where `before.code` did?” Further separate lost propagation from caught cancellation if mixed examples remain ambiguous. |
| `violates-cancellation-contract` | Split outcomes: “Can `candidate.main_operation` continue after `candidate.cancellation`?” and, for catch/return cases, “Does `after.code` report success when `candidate.main_operation` is canceled?” |

**Boundary:** caller-owned work continuing after cancellation is the target. Bounded cleanup preserving the canceled result and separately owned background work are different cases. The task exception should ask explicitly whether the task requires the named work to continue independently, rather than being buried in the outcome question.

**Context:** request/operation ownership, original cancellation source, async/fetch call, catch/return behavior, bounded cleanup, and any explicit background-work requirement. A generic hunk may omit who owns the work.

**Required pairs:** dropped fetch signal versus equivalent propagation; successful return versus canceled return; bounded cleanup versus continuing main operation; explicitly detached job versus accidental background context. Outcome alternatives require OR logic, not a minimum over both.

### 4.4 `silent-failure-fallback` — error, threshold 0.92

| Current signal | Proposed treatment |
|---|---|
| `intercepts-failure` | Use syntax to select an error branch where supported; otherwise ask “Does `candidate.branch` handle a failed operation?” |
| `returns-success-or-default` | Split: “Does `candidate.branch` continue with a success result or fallback value?” and “Does `candidate.branch` expose the failure through `contract.failure_channel`?” |

**Boundary:** a fallback is not automatically a violation. The failure must become unobservable according to the applicable contract. Logging, metrics, an error return, and user-visible status are different channels; do not silently redefine whether logging qualifies. Where the established contract does not resolve this distinction, preserve existing policy and flag the case for labeling rather than invent a new rule.

**Context:** failed call, complete error branch, return type, nearby caller handling where relevant, documented failure channel, and task-authorized resilience. Add a separate permission question for an explicit fallback requirement.

**Required pairs:** silent empty collection versus returned error; logged fallback versus contract-required surfaced error; intentional cache resilience versus hidden data loss. The observability question has opposite polarity to the current violation score.

### 4.5 `weakened-test-assertion` — error, threshold 0.93

| Current signal | Proposed treatment |
|---|---|
| `reduces-visible-protection` | “Does `after.test` stop checking behavior checked by `before.test`?” Add a separate task question: “Does `task.text` explicitly retire `candidate.checked_behavior`?” |

**Boundary:** true means the changed test accepts an outcome that the old assertion rejected. False means the same behavior is still checked, including through a supplied equivalent assertion helper. Exact-message checks, tolerances, skip markers, and collection comparisons are distinct fixtures; syntax can identify candidates, but equivalence can require semantics.

**Context:** the corresponding before/after test, the changed assertion and expected value, directly relevant assertion helpers, and task text. No need for the whole suite to judge this local change. If the assertion moved to another test and the move is relevant to the rule policy, retrieve that specific replacement and document the broader evidence boundary.

**Why this improves the example:** remove the broad “Do not infer…” sentence; make the question's object the behavior of the changed test. Separately handle the user's agreed exception for a task that specifically retires an old contract. The error-message retirement false positive is a policy mismatch in addition to a wording issue.

**Required pairs:** exact-to-broad assertion versus equivalent helper; retired message versus vague cleanup; increased tolerance required by new precision contract versus unexplained increase; removed test versus relocated equivalent test.

### 4.6 `boolean-mode-parameter` — warning, threshold 0.90

| Current signal | Proposed treatment |
|---|---|
| `introduces-boolean-input` | Code gate for a newly caller-supplied boolean parameter/option where syntax and types establish it; fallback: “Does `candidate.api` add a boolean input for callers?” |
| `selects-behavior-mode` | “Does `candidate.boolean` choose how `candidate.operation` runs?” |

**Boundary:** true for a procedural mode switch; false for domain truth, feature state, or a caller-owned product/security choice. Keep these positive definitions in criteria rather than a long list inside the question. A boolean selecting behavior is not by itself enough to violate the policy.

**Context:** API declaration, uses of the boolean in the body, representative changed call sites, relevant domain contract. Avoid guessing meaning from names like `enabled` or `force` alone.

**Required pairs:** `save(async)` versus `user.isActive`; formatting mode versus explicit security policy; adding a parameter versus changing an existing boolean's implementation.

### 4.7 `caller-managed-mechanics` — warning, threshold 0.90

| Current signal | Proposed treatment |
|---|---|
| `introduces-caller-duty` | “Does `after.api` require callers to perform `candidate.duty` that `before.api` handled?” For new APIs, use a separately defined new-API candidate rather than inventing a before version. |
| `duty-is-mechanical` | “Is `candidate.duty` an internal operating step of `candidate.component`?” |

**Boundary:** true for implementation setup/order/translation that belongs inside the component; false for required input, caller-owned policy, explicit resource ownership, or necessary coordination. This is a design judgment and depends on a stated component responsibility.

**Context:** before/after API and caller examples, the specific duty, ownership contract, and task. Supplying only the callee hunk can hide an increased caller burden.

**Required pairs:** pushed retry plumbing versus caller-selected retry policy; forced lifecycle ordering versus explicitly owned resource; mechanical representation conversion versus meaningful caller data.

### 4.8 `ceremonial-abstraction` — warning, threshold 0.90

| Current signal | Proposed treatment |
|---|---|
| `introduces-abstraction` | Code gate for added declarations/wrappers; fallback: “Does the change introduce `candidate.abstraction`?” |
| `adds-interface-ceremony` | “Does using `candidate.abstraction` add caller steps or concepts?” |
| `hides-little-visible-behavior` | Prefer positive polarity: “Does `candidate.abstraction` enforce a constraint or perform a meaningful operation?” |

**Boundary:** simple forwarding/assignment alone is little behavior; enforced invariants, nontrivial policy, translation, dependency management, or state handling can be meaningful. “Meaningful” needs policy examples; this remains a warning, not an objective architecture proof.

**Context:** full small abstraction body, before/after usage, and relevant contract. A large body truncated to a trivial method cannot support this judgment. Positive behavior evidence would veto the warning; this requires composition changes.

**Required pairs:** wrapper-only service versus invariant-enforcing service; added construction steps versus simpler caller API; trivial local behavior with an explicit compatibility contract.

### 4.9 `implementation-detail-exposure` — warning, threshold 0.91

| Current signal | Proposed treatment |
|---|---|
| `changes-external-boundary` | Use supported export/visibility/reference analysis; fallback: “Does `candidate.change` alter an interface used outside `candidate.component`?” |
| `exposes-implementation-detail` | Split: “Does `candidate.exposed_value` reveal how `candidate.component` is implemented?” and “Is `candidate.exposed_value` required by `contract.public_api`?” |

**Boundary:** a storage/cache/dependency representation can be an implementation detail, but may also be the component's intended public product. Public JSON or a database adapter's database types are not automatically leakage. Without a known boundary or purpose, abstain from a confident architecture claim.

**Context:** enclosing module boundary, before/after public declaration, exposed type definition, actual caller use, and documented contract. Avoid whole-repository context merely to speculate about purpose.

**Required pairs:** accidental cache key exposure versus documented cache API; internal row type versus database adapter contract; private field edit versus public return-type edit. The contract question is a veto, not another positive violation signal.

### 4.10 `named-special-case` — warning, threshold 0.91

| Current signal | Proposed treatment |
|---|---|
| `changes-reusable-mechanism` | “Is `candidate.component` responsible for a shared technical mechanism?” |
| `branches-on-named-policy` | “Does `candidate.branch` apply a rule for a specific customer or product workflow?” |

**Boundary:** task-specific product policy embedded in a shared mechanism is the concern. Protocol requirements, boundary compatibility, and security separation remain allowed. Names are a candidate-finding hint, not proof of inappropriate policy.

**Context:** changed branch, enclosing component responsibility, protocol/security contract where relevant, and task. Separate an explicit exception judgment if the criteria become difficult to follow.

**Required pairs:** customer-ID special case versus protocol version handling; workflow-specific branch versus security boundary; same branch in product policy code versus generic infrastructure.

### 4.11 `narrating-comment` — warning, threshold 0.92

| Current signal | Proposed treatment |
|---|---|
| `restates-visible-code` | “Does `candidate.comment` only repeat what `candidate.code` does?” |

**Boundary:** true for redundant description of the adjacent operation. False for API documentation, mandatory documentation, reasons, constraints, domain meaning, warnings, or non-obvious behavior. Use parser/position facts to exclude clearly required documentation where supported; keep semantic exceptions in criteria.

**Context:** changed comment, the precise adjacent code, and declaration/doc-comment role. This is the strongest candidate for a small-context wording-only experiment; no helper graph or full patch is normally needed.

**Required pairs:** “increment count” versus explanation of why a count must change; redundant line comment versus required public API documentation; stale comment versus redundant comment, since inaccuracy is a different rule concern.

### 4.12 `unchanged-argument-forwarder` — warning, threshold 0.90

| Current signal | Proposed treatment |
|---|---|
| `introduces-forwarder` | Select changed forwarding candidates in syntax; fallback: “Does `candidate.wrapper` mainly call `candidate.target` and return the result?” |
| `preserves-operation-and-arguments` | “Does `candidate.wrapper` expose the same operation and pass the same inputs to `candidate.target`?” Split operation and argument equivalence if needed; compare exact argument identities in code when provable. |
| `adds-no-visible-behavior` | Prefer positive polarity: “Does `candidate.wrapper` add behavior beyond calling `candidate.target`?” |

**Boundary:** validation, authorization, translation, composition, transactions, lifecycle, observability, compatibility, and error handling are substantive additions. A wrapper's architectural justification elsewhere is a separate claim; preserve the current local behavioral scope.

**Context:** whole wrapper body and signature, target signature/contract, changed call relationship. Include target body only when it resolves operation or argument meaning. Positive added behavior vetoes a warning and needs different composition.

**Required pairs:** exact forwarder versus authorization wrapper; renamed argument versus transformed input; logging wrapper versus empty forwarding; compatibility adapter versus redundant alias.

### 4.13 `unused-extensibility-point` — warning, threshold 0.91

| Current signal | Proposed treatment |
|---|---|
| `introduces-extensibility` | “Does `candidate.api` let callers replace or add behavior?” Use syntax to identify concrete candidates first. |
| `only-one-visible-behavior` | Enumerate resolvable implementations/registrations/instantiations in code. If distinctness needs semantics, ask about a specific pair: “Do `candidate.implementation_a` and `candidate.implementation_b` provide different behavior?” |
| `task-does-not-request-extensibility` | Prefer positive polarity: “Does `task.text` explicitly require replaceable or additional behavior through `candidate.api`?” Code handles an absent task; preserve the existing no-task non-finding policy until deliberately revised. |

**Boundary:** multiple declarations can implement the same behavior, and one implementation in a repository can serve external clients. Do not replace semantic behavior counting with a naive declaration count. A testing seam or public extension contract can justify the mechanism even with one production implementation.

**Context:** introduced declaration, resolved implementations/registrations, relevant callers, task, public extension contract, and search coverage. Current text search is capped at 40 hits/12,000 characters, searches selected tracked files, and returns matching lines. It cannot prove global non-use. Open-world/plugin cases need a narrower claim or an insufficient-evidence outcome.

**Required pairs:** one internal implementation versus multiple distinct implementations; one implementation plus required test seam; external public extension API; truncated search; unstaged extra implementation during a staged check. Counting and absence are the largest capability mismatch in this rule.

### 4.14 `scope-creep` — warning, threshold 0.90

| Current signal | Proposed treatment |
|---|---|
| `unrelated-substantial-change` | For an identified change group: “Is `candidate.change` needed to complete `task.text`?” Separately determine whether an unrelated change is substantial. |

**Boundary:** required tests, necessary refactors, API consequences, and related safety fixes remain in scope. Missing linkage is not proof of unrelatedness. An optional Choice experiment could distinguish required/direct, required/supporting, unrelated, and insufficient evidence; these categories must have non-overlapping definitions before measurement.

**Context:** task, patch inventory, focal change group, and concrete links to related groups. Keep enough cross-file evidence to explain migrations and shared APIs. Cluster in code where relationships are reliable; retain ambiguity rather than inventing semantic summaries.

**Composition:** warn when an evaluated group is both substantial and unrelated, while reporting uncovered groups. This reverses the proposed relation question's polarity and changes patch aggregation, so it is a distinct experiment. Test split/group boundary stability and findings locations.

**Required pairs:** API change plus necessary caller updates versus unrelated feature; substantial independent refactor versus small necessary refactor; large patch within budget versus explicit partial coverage. Giving every rule whole-patch scope would not solve their evidence needs.

## 5. How to improve without confounding the result

Use a small sequence of separately reviewable changes. For each implementation item, add behavior tests, update user-visible documentation, run `go test ./...`, and run the repository's normal build/vet/format/Reaper workflow before proceeding. Keep the draft PR and the approved release policy.

1. **Make measurement faithful.** Record resolved model, exact questions/criteria, full request identity, context coverage, signal values, rule composition/version, token counts, cost, and elapsed time. Store source evidence separately from model output; avoid calling question text a model explanation. Preserve historical reports.
2. **Expose structured state and Noul criteria.** Keep legacy requests reproducible. Test serialization, validation, cache separation, result metadata, mixed snapshot rejection, and backend errors. Do not add Choice/Score merely because the API supports them.
3. **Run a wording-only baseline.** Freeze shorter, self-contained questions using the *existing* evidence and unchanged signal meaning/composition. Remove redundant “Return the probability” prefixes, replace vague references with explicit subjects, and retain policy exceptions. This isolates language simplification from new evidence and new rule definitions.
4. **Run separate state/criteria ablations.** Same facts serialized as flat text versus native object; then the same proposition with versus without true/false criteria. Hold all other inputs constant. Repeated measurements are stability checks, not new independent cases.
5. **Repair evidence and factual/policy separation for authorization and validation first.** Promote matched snapshots and bounded helper evidence with explicit unsupported cases. Then evaluate atomic enforcement and task-permission questions as a separate condition. Include the inserted-bypass and explicit-contract overlap cases, but use fresh independently labeled cases for evaluation.
6. **Apply the same method to cancellation and assertions, then silent fallback.** Ownership, specific checked behavior, and allowed resilience require dedicated context and labels. Do not assume the authorization result transfers to these rules.
7. **Evaluate advisory rules by evidence cost and clarity.** Start with comments, boolean candidates, and forwarders; then API/architecture judgments. Extensibility and patch scope need evidence/composition work before shortening language can make them dependable.
8. **Calibrate only stable candidates.** Use separate development and held-out groups, split by repository/template to limit leakage, and keep a release evaluation untouched. Maintain ≥80% recall, ≤1% empirical false positives, and ≥20 positive/100 negative independently reviewed cases per blocking rule. Minimum sample counts are release requirements, not statistical proof of a population ≤1% false-positive rate; report uncertainty as well.

Track classification quality *and coverage*: evaluated, ineligible, missing evidence, unsupported resolution, truncated, provider failed. Report TP/FN/FP/TN per rule and language, repeated-score ranges, worst safe and unsafe cases, and token/latency cost. Do not improve apparent recall by changing the denominator or treating abstentions as correct negatives.

Task permission follows the user's agreed policy: a clear requirement for the **specific** change can permit it; vague “simplify,” “clean up,” or “optimize” requests do not. Evaluate that proposition once for the relevant candidate and feed the result into rule policy. Do not generalize this exception into permission to weaken unrelated checks.

## 6. Completion and remaining uncertainty

Completed: official capability research, source audit, all 14 rules/28 signals reviewed, exact current-instruction snapshot, and three successful live API compatibility probes. Production behavior is unchanged.

Verification: `go test ./...` passes. A document integrity check confirms coverage of every production rule/signal ID, validates all local links in this audit and its experiment README, and verifies the three stored successful probe responses. No new automated behavior tests were added because this change adds research documentation and recorded requests, not production behavior.

Not established: accuracy gains from these proposed rewrites, calibration of any new composition, reliable general-purpose repository resolution, or release readiness. The next concrete implementation should add structured state/criteria and measurement metadata, followed by controlled comparisons. Lowering thresholds first would conceal the validation ranking problem rather than resolve it.
