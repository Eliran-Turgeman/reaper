# Jev API capability smoke check

Research collected 21 September 2026; production rule snapshot from revision
`85cf044d91a3619d52e01e93fd9abb2cd15f4482`.

See the [complete capability and rule audit](../../../docs/jev-capabilities-and-rule-audit.md)
for interpretation, the 14-rule/28-signal review, and proposed experiments.

- `current-rules.json`: exact production instructions, metadata, and thresholds.
- `probe.json`: three synthetic API requests and their responses, including
  actual resolved model, usage, cost, and elapsed time. Credentials and HTTP
  authorization headers are omitted.

All three requests returned HTTP 200 from the OpenRouter Decisions endpoint:
flat state with Noul; object state with Noul criteria; and object state with
Noul, Choice, and Score questions in one request. The result establishes support
for these request shapes on the tested route. It does not measure rule accuracy,
calibration, question independence, or latency performance. Each request ran once.

The structured condition changes both state representation and criteria, so
differences between conditions cannot be attributed to either independently.
The proposed audit questions have not been benchmarked and are not deployable
experiment configurations.

To reproduce a capability request, submit its complete `request` object from
`probe.json` as the JSON body to the recorded endpoint, using an independently
provided OpenRouter credential. Preserve the full response and elapsed time;
do not include the credential in saved artifacts. The model alias may resolve
to a different version in future runs.
