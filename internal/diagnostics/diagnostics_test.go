package diagnostics

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

func TestJSONOutputSchemaAndBlockingPolicy(t *testing.T) {
	report := New([]Diagnostic{{
		Rule: "silent-failure-fallback", Severity: rules.SeverityError,
		Confidence: 0.93, Threshold: 0.90, File: "client.go",
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
	if decoded["version"].(float64) != 2 || decoded["passed"].(bool) {
		t.Fatalf("unexpected envelope: %s", output.String())
	}
	summary := decoded["summary"].(map[string]any)
	if summary["errors"].(float64) != 1 || summary["semantic_checks"].(float64) != 1 {
		t.Fatalf("unexpected summary: %s", output.String())
	}
}

func TestIncompletePoliciesNeverClaimPass(t *testing.T) {
	for policy, code := range map[string]int{"error": 2, "warning": 0, "ignore": 0} {
		report := New(nil, Summary{UnitsSkipped: 1})
		report.SetIncomplete([]SkippedUnit{{File: "large.go", StartLine: 1, Reason: "provider token limit exceeded"}}, policy)
		if report.Complete || report.Passed || report.Status != "incomplete" || report.ExitCode() != code {
			t.Fatalf("policy %s: %+v", policy, report)
		}
		var output bytes.Buffer
		if err := WriteText(&output, report); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(output.String(), "No blocking semantic violations found.") || !strings.Contains(output.String(), "Analysis incomplete") {
			t.Fatal(output.String())
		}
		output.Reset()
		if err := WriteJSON(&output, report); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output.String(), "provider token limit exceeded") {
			t.Fatal(output.String())
		}
	}
}

func TestWarningsDoNotBlock(t *testing.T) {
	report := New([]Diagnostic{{Severity: rules.SeverityWarning}}, Summary{})
	if !report.Passed {
		t.Fatal("warning unexpectedly blocked")
	}
}

func TestSignalOrderingAndConfidenceSchema(t *testing.T) {
	report := New([]Diagnostic{{Confidence: .96, Signals: []SignalScore{{ID: "returns-success-or-default", Score: .96}, {ID: "intercepts-failure", Score: .99}}}}, Summary{})
	if report.Diagnostics[0].Signals[0].ID != "intercepts-failure" {
		t.Fatal("signals not sorted")
	}
	var output bytes.Buffer
	if err := WriteJSON(&output, report); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "probability") || !strings.Contains(output.String(), `"confidence": 0.96`) {
		t.Fatal(output.String())
	}
	output.Reset()
	if err := WriteText(&output, report); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "intercepts-failure=0.99") || !strings.Contains(output.String(), "returns-success-or-default=0.96") {
		t.Fatal(output.String())
	}
}
