# Context-only evidence development cases

The first four-condition experiment mostly used short files already visible in
their entirety. Snapshot context there can measure repetition/formatting effects,
but does not establish a benefit from new evidence.

These four follow-up cases keep helper implementations in an unchanged second
file. Replacing the inline guard with the helper is safe only when that helper
actually enforces the same condition. Both classes use the same helper names;
the implementation, rather than the name, establishes the label. Ordinary hunk
context omits that unchanged file for these rules. Snapshot context includes it.
The source maps expose the same amount of evidence for positives and negatives.

Evaluate all four `auth-validation-v2` conditions with isolated grouping. These
cases were authored after viewing initial experiment results and are development
data, not a holdout or evidence of production-wide accuracy.
