package eval

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/provider"
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
	for _, field := range []string{"corpus", "rules", "config", "protocol", "experiment", "missing"} {
		t.Run(field, func(t *testing.T) {
			report := baseline
			p := *baseline.Provenance
			report.Provenance = &p
			switch field {
			case "experiment":
				p.ExperimentSHA256 = fingerprint("changed experiment")
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

func TestRecordingRetainsProviderMetadataAndUnknownUsage(t *testing.T) {
	for _, withUsage := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("x-request-id", "header-request")
			body := map[string]any{"model": "resolved-version", "provider": "TypeSafe", "answers": map[string]any{"a": map[string]any{"type": "noul", "noul": .8}}}
			if withUsage {
				body["id"] = "body-request"
				body["usage"] = map[string]any{"input_tokens": 12, "output_tokens": 2, "cost": 0}
			}
			_ = json.NewEncoder(w).Encode(body)
		}))
		client, err := provider.New("openrouter", provider.Options{BaseURL: server.URL, APIKey: "test"})
		if err != nil {
			t.Fatal(err)
		}
		recorder := &recordingEvaluator{Evaluator: client}
		request := decision.Request{Model: "requested-alias", State: "fixture evidence", Questions: []decision.Question{{ID: "a", Instructions: "A?"}}}
		_, err = decision.Evaluate(context.Background(), recorder, request)
		server.Close()
		if err != nil {
			t.Fatal(err)
		}
		record := recorder.Records()[0]
		if record.Model != "requested-alias" || record.State != request.State || record.SHA256 != fingerprint(request) || record.ElapsedMS < 0 || len(record.Calls) != 1 {
			t.Fatal(record)
		}
		call := record.Calls[0]
		if call.ResolvedModel != "resolved-version" || call.Provider != "TypeSafe" {
			t.Fatal(call)
		}
		if withUsage {
			if call.RequestID != "body-request" || call.Usage == nil || call.Usage.InputTokens != 12 || call.Usage.OutputTokens != 2 || call.Usage.CostUSD == nil || *call.Usage.CostUSD != 0 {
				t.Fatal(call)
			}
		} else if call.RequestID != "header-request" || call.Usage != nil {
			t.Fatal("invented missing usage", call)
		}
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
