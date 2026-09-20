package diagnostics

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

func TestAgentFindingsContainEvidenceAndObjective(t *testing.T) {
	for _, rule := range rules.All() {
		if objectives[rule.ID] == "" {
			t.Errorf("missing remediation: %s", rule.ID)
		}
	}
	report := New([]Diagnostic{{Rule: "scope-creep", File: "<patch>", Signals: []SignalScore{{ID: "unrelated-substantial-change", Score: .99}}}}, Summary{})
	var output bytes.Buffer
	if err := WriteAgent(&output, report, "Fix retries"); err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(output.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	finding := value["findings"].([]any)[0].(map[string]any)
	if finding["task"] != "Fix retries" || finding["objective"] == "" || len(finding["evidence"].([]any)) != 1 {
		t.Fatal(finding)
	}
}
