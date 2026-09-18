package diagnostics

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Eliran-Turgeman/repear/internal/rules"
)

func TestJSONOutputSchemaAndBlockingPolicy(t *testing.T) {
	report := New([]Diagnostic{{
		Rule: "silent-failure-fallback", Severity: rules.SeverityError,
		Probability: 0.93, Threshold: 0.90, File: "client.go",
		StartLine: 4, EndLine: 6, Message: "New fallback may hide an unexpected failure.",
	}}, Summary{UnitsEvaluated: 1, SemanticChecks: 1, JevRequests: 1})
	var output bytes.Buffer
	if err := WriteJSON(&output, report); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["version"].(float64) != 1 || decoded["passed"].(bool) {
		t.Fatalf("unexpected envelope: %s", output.String())
	}
	summary := decoded["summary"].(map[string]any)
	if summary["errors"].(float64) != 1 || summary["semantic_checks"].(float64) != 1 {
		t.Fatalf("unexpected summary: %s", output.String())
	}
}

func TestWarningsDoNotBlock(t *testing.T) {
	report := New([]Diagnostic{{Severity: rules.SeverityWarning}}, Summary{})
	if !report.Passed {
		t.Fatal("warning unexpectedly blocked")
	}
}
