package eval

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
)

func TestGateRejectsChangedInputsEvenWhenMetricsMatch(t *testing.T) {
	cfg := config.Defaults()
	baseline := Report{Version: 1, Model: "pinned", Provider: "typesafe", Provenance: provenance("git-v1", map[string]string{"a.go": "before"}, cfg), Rules: []RuleReport{{Rule: "r", Examples: 1}}}
	data, _ := json.Marshal(baseline)
	file := filepath.Join(t.TempDir(), "baseline.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := CheckBaseline(io.Discard, baseline, file, .02); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"corpus", "rules", "config", "protocol", "missing"} {
		t.Run(field, func(t *testing.T) {
			report := baseline
			p := *baseline.Provenance
			report.Provenance = &p
			switch field {
			case "corpus":
				p.CorpusSHA256 = fingerprint("changed body, same case ID")
			case "rules":
				p.RulesSHA256 = fingerprint("changed question")
			case "config":
				p.ConfigSHA256 = fingerprint("changed exclusion")
			case "protocol":
				p.Protocol = "git-v2"
			case "missing":
				report.Provenance = nil
			}
			if err := CheckBaseline(io.Discard, report, file, .02); err == nil {
				t.Fatal("changed input passed")
			}
		})
	}
}

type unbatchedEvaluator struct{ *benchmarkEvaluator }

func (*unbatchedEvaluator) Capabilities() decision.Capabilities { return decision.Capabilities{} }

func TestRecordingPreservesActualGroupingAndContextIdentity(t *testing.T) {
	r := &recordingEvaluator{Evaluator: &unbatchedEvaluator{&benchmarkEvaluator{score: .4}}}
	request := decision.Request{Model: "pinned", State: "code", Questions: []decision.Question{{ID: "a", Instructions: "first"}, {ID: "b", Instructions: "second"}}}
	if _, err := decision.Evaluate(context.Background(), r, request); err != nil {
		t.Fatal(err)
	}
	records := r.Records()
	if len(records) != 2 || len(records[0].Questions) != 1 || records[0].StateSHA256 != records[1].StateSHA256 || records[0].SHA256 == records[1].SHA256 {
		t.Fatal(records)
	}
	changed := request
	changed.State += "changed context"
	if fingerprint(changed) == fingerprint(request) {
		t.Fatal("context change not fingerprinted")
	}
}
