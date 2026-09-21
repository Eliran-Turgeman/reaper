# Security and privacy

Reaper performs inference outside the machine. Before checking a private
repository, obtain approval for the selected provider and the code being sent.

For each evaluated unit the request contains its repository-relative file path,
language, diff (except in audit mode), current code, previous code when available,
surrounding code when different from current code, and task context when supplied.
Rule predicates, their identifiers, and the configured model are also sent.
The patch-scoped scope-creep rule sends the aggregated patch and task.
Repository-search rules additionally send bounded matching lines from tracked,
supported, nonexcluded repository files, including their paths and line numbers.
These retrieved lines are also included in finding `context_evidence`.
For `--staged`, both changed-file context and repository-search evidence use
captured index blobs; unstaged source text is excluded from those reads.
With `--experimental-context targeted-go`, authorization and validation checks
also send bounded before/after function and helper bodies from eligible Go files.
The before source comes from the selected comparison base, so this opt-in mode
can send relevant historical code. Other rules do not receive these helper bodies.
`--all` can send entire tracked source files. `reaper eval` sends each example's language,
task, before code, after code, and rule predicates. Git history, PR comments,
environment variables, and unrelated files are not intentionally included;
the historical-source exception for targeted context is described above.
Secrets present in source code, diffs, or task text are part of that content;
Reaper does not provide a secret-redaction guarantee.

`provider: typesafe` sends requests directly to `https://api.typesafe.ai`.
`provider: openrouter` sends requests through `https://openrouter.ai` to the
selected model provider. `base_url` overrides the endpoint; use only an approved
endpoint. Reaper does not control provider retention, training, routing, or
regional processing. Review the provider's current contract and your enterprise
requirements before use; Reaper makes no zero-retention claim.

API keys are read from `TYPESAFE_API_KEY` or `OPENROUTER_API_KEY` and used in the
Authorization header. Reaper does not save keys in configuration or its cache.
In GitHub Actions, pass credentials as secrets. Do not expose them to untrusted
fork workflows. A fork without credentials fails before inference.

Caching is local and enabled by default. The default location is
`<repository>/.git/reaper-cache`; `cache.dir` can override it (relative paths are
relative to the Git root). Entries contain numeric signal scores. SHA-256 keys
incorporate Reaper version, semantic schema, provider/model, complete evaluation
state, rule version, and predicate instructions. Explicit Noul criteria and the
state representation also separate cache entries. Benchmark request records retain fixture source
state, exact questions/criteria, resolved model metadata, usage, and elapsed time;
treat these artifacts as source-bearing reports. Credentials are not recorded.
Source text and credentials are not written as cache values. A score cache is still sensitive local metadata;
control access and do not upload it as a public CI artifact. Use `--no-cache` or
`cache.enabled: false` to disable it. Removing the cache simply causes inference
to be repeated.

`--verbose` and `--debug` report paths, rule IDs, scores, thresholds, task source,
and request/cache counts. `--debug` also prints retrieved repository context for
rules that requested it, which can contain source secrets. It does not dump
complete provider requests. Provider
errors can contain provider-supplied response text, so keep logs and diagnostics
private. There is no raw request debug switch. JSON, agent output, and SARIF
contain finding metadata; agent output can include task text. Treat exported
reports with the same care as the repository.

For enterprise use, set an approved provider/model and endpoint, narrow paths
and exclusions before running, review tasks for sensitive data, and isolate
credentials and local caches. There is no background telemetry upload.
