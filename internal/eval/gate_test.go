package eval

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestQualityGateRejectsRegressionAndModelChanges(t *testing.T) {
	baseline := Report{Version: 1, Provider: "openrouter", Model: "pinned", Rules: []RuleReport{{Rule: "r", Examples: 20, Precision: .9, Recall: .8}}}
	data, _ := json.Marshal(baseline)
	path := filepath.Join(t.TempDir(), "baseline.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := CheckBaseline(io.Discard, baseline, path, .02); err != nil {
		t.Fatal(err)
	}
	baseline.Rules[0].Recall = .7
	if err := CheckBaseline(io.Discard, baseline, path, .02); err == nil {
		t.Fatal("recall regression passed")
	}
	baseline.Rules[0].Recall = .8
	baseline.Model = "other"
	if err := CheckBaseline(io.Discard, baseline, path, .02); err == nil {
		t.Fatal("model drift passed")
	}
	baseline.Model = "pinned"
	baseline.Rules[0].Threshold = .5
	if err := CheckBaseline(io.Discard, baseline, path, .02); err == nil {
		t.Fatal("threshold drift passed")
	}
	baseline.Rules[0].Threshold = 0
	baseline.Rules = nil
	if err := CheckBaseline(io.Discard, baseline, path, .02); err == nil {
		t.Fatal("missing rule passed")
	}
}
