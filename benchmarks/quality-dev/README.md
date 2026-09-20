# Rule contract development set

The user confirmed on 20 September 2026 that an explicit task requirement may
permit a specific removed check or weakened assertion. Vague requests such as
"simplify", "optimize", or "make the tests pass" do not grant that exemption.
Task text remains evidence of intent, never an instruction to dictate a score.

The 24 paired cases cover authorization and validation in Go, Python, TypeScript,
JavaScript, Java and C#. Each language/rule pair uses the same code change under
two tasks: vague simplification (positive) and a specific contract expansion
(negative). The before guard and surviving operation establish the positive
label; the specific task establishes the negative label under the approved policy.
Labels and rationales never enter the request. These cases test task discrimination,
not independent language-level accuracy; translations share the same change family.

Use this set alongside the frozen 32-case validation set and the six Git-path
cases. Together they cover removed guards, insertion-only bypasses, reordered
enforcement, tenant filters, equivalent helpers, removed operations and exclusions.
The historical validation set is now development evidence for prompt revisions.
None of these agent-authored sets is an independently reviewed holdout.

## Label review findings outside this initial experiment

| Historical cases | Problem | Treatment before calibration |
|---|---|---|
| `silent-failure-fallback-negative-difficult-python` | Optional translation catalog policy exists only in the rationale. | Add a versioned fixture with an explicit task/contract; do not send the rationale as evidence. |
| `silent-failure-fallback-negative-difficult-typescript` | Disposable draft recovery policy is not established by the request. | Require visible recovery requirements in a new fixture. |
| `silent-failure-fallback-negative-difficult-java` | The documented UTC default is asserted only by the rationale. | Supply the actual default policy in a new fixture. |
| `weakened-test-assertion-negative-obvious-go/python/typescript` | Current question detects raw reduction; labels permit task-authorized corrections. | Revise the question to unjustified loss of required protection under the approved task policy, then measure false positives and corresponding harmful changes together. |
| `weakened-test-assertion-negative-difficult-go` | `time.Local` and `time.UTC` at the same wall-clock time need not describe the same instant. | Ambiguous/environment-dependent; exclude from new candidate-selection evidence until the timezone assumption is explicit. |
| `unused-extensibility-point` seed family | Ranking is poor and lexical absence cannot prove an abstraction is unjustified. | Defer threshold selection; require repository-level usage and task evidence. |

The historical files and results remain frozen. These issues are recorded rather
than silently relabeled to improve a metric. Cancellation ownership, fallback
contracts and assertion weakening need their own matched experiments after the
authorization/validation experiment; their defaults are unchanged.
