package diagnostics

import (
	"encoding/json"
	"io"
)

var objectives = map[string]string{
	"removed-authorization-check":    "Restore equivalent authorization before the protected operation.",
	"removed-validation":             "Restore equivalent input validation before the guarded operation.",
	"swallowed-cancellation":         "Preserve cancellation propagation and stop the main operation when cancellation is requested.",
	"silent-failure-fallback":        "Preserve failure semantics; propagate or explicitly handle the error instead of returning successful defaults.",
	"weakened-test-assertion":        "Restore the specific behavioral protection removed or weakened by the test change.",
	"scope-creep":                    "Remove changes unrelated to the supplied task while retaining necessary implementation and tests.",
	"narrating-comment":              "Remove the redundant narration or explain the reason, constraint, or invariant.",
	"unchanged-argument-forwarder":   "Remove redundant forwarding or make its required boundary behavior explicit.",
	"ceremonial-abstraction":         "Simplify the abstraction while preserving required policy and invariants.",
	"boolean-mode-parameter":         "Express distinct operations clearly without an ambiguous caller-selected boolean mode.",
	"caller-managed-mechanics":       "Encapsulate mechanical duties at the owning boundary while preserving caller policy.",
	"implementation-detail-exposure": "Keep implementation details behind the boundary unless required by its public contract.",
	"named-special-case":             "Move product-specific policy out of the reusable mechanism.",
	"unused-extensibility-point":     "Remove speculative extensibility unless a concrete task requirement justifies it.",
}

func WriteAgent(w io.Writer, report Report, task string) error {
	findings := make([]any, 0, len(report.Diagnostics))
	for _, item := range report.Diagnostics {
		evidence := make([]any, 0, len(item.Signals))
		for _, signal := range item.Signals {
			evidence = append(evidence, map[string]any{"signal": signal.ID, "score": signal.Score})
		}
		finding := map[string]any{"rule": item.Rule, "severity": item.Severity, "file": item.File, "lines": []int{item.StartLine, item.EndLine}, "confidence": item.Confidence, "evidence": evidence, "objective": objectives[item.Rule]}
		if item.ContextEvidence != "" {
			finding["context_evidence"] = item.ContextEvidence
		}
		finding["fingerprint"] = item.Fingerprint
		if task != "" && (item.Rule == "scope-creep" || item.Rule == "unused-extensibility-point") {
			finding["task"] = task
		}
		findings = append(findings, finding)
	}
	return json.NewEncoder(w).Encode(map[string]any{"version": 1, "status": report.Status, "complete": report.Complete, "findings": findings, "skipped_units": report.Skipped})
}
