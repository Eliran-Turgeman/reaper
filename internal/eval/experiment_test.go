package eval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
)

func TestExperimentsChangeOnlyChosenInputAndRecordActualRequest(t *testing.T) {
	c := GitCase{PatchCase: PatchCase{ID: "test", Rule: "removed-validation", Task: "Keep validation", Expected: "positive", Rationale: "GOLD LABEL MUST NEVER LEAK"}, BeforeFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"}, AfterFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { store(n) }\n"}}
	var baseState, baseQuestions string
	for _, variant := range []string{"current", "questions", "context", "both"} {
		experiment, err := LoadExperiment("../../benchmarks/experiments/auth-validation-v2/" + variant + ".json")
		if err != nil {
			t.Fatal(err)
		}
		client := &benchmarkEvaluator{score: .7}
		result, err := runGitCaseExperiment(context.Background(), client, c, config.Defaults(), "isolated", experiment)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Requests) != 1 || len(client.requests) != 1 {
			t.Fatalf("missing request: %+v", result)
		}
		request := client.requests[0]
		if result.Requests[0].SHA256 != fingerprint(request) {
			t.Fatal("recorded pre-transform request")
		}
		if strings.Contains(request.State, c.Rationale) {
			t.Fatal("label evidence leaked")
		}
		if variant == "current" {
			baseState, baseQuestions = fingerprint(request.State), fingerprint(request.Questions)
		}
		wantContextChange := variant == "context" || variant == "both"
		wantQuestionChange := variant == "questions" || variant == "both"
		if (fingerprint(request.State) != baseState) != wantContextChange || (fingerprint(request.Questions) != baseQuestions) != wantQuestionChange {
			t.Fatalf("%s changed the wrong input", variant)
		}
		for _, signal := range result.Observations[0].Signals {
			for _, q := range request.Questions {
				if q.ID == c.Rule+":"+signal.ID && signal.Evidence != q.Instructions {
					t.Fatal("signal describes the wrong question")
				}
			}
		}
	}
}

func TestStructuredExperimentPreservesFactsAndRecordsCriteria(t *testing.T) {
	c := GitCase{PatchCase: PatchCase{ID: "structured", Rule: "removed-validation", Task: "Keep validation", Expected: "positive", Rationale: "PRIVATE GOLD LABEL"}, BeforeFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { if n < 0 { panic(n) }; store(n) }\n"}, AfterFiles: map[string]string{"x.go": "package x\nfunc Save(n int) { store(n) }\n"}}
	var textState string
	for _, format := range []string{"json-text", "json-object"} {
		client := &benchmarkEvaluator{score: .7}
		e := &Experiment{Version: 1, Name: "format", Context: "snapshots", StateFormat: format, Criteria: map[string]decision.NoulCriteria{"removed-validation:removes-validation": {True: "A previous check is bypassed.", False: "Previous checks still apply."}}}
		result, err := runGitCaseExperiment(context.Background(), client, c, config.Defaults(), "isolated", e)
		if err != nil {
			t.Fatal(err)
		}
		request := client.requests[0]
		if format == "json-text" {
			textState = request.State
		} else if string(request.StructuredState) != textState || result.Requests[0].StateSHA256 != fingerprint(request.StructuredState) {
			t.Fatal("representation changed the supplied facts")
		}
		if strings.Contains(textState, c.Rationale) {
			t.Fatal("label leaked into evidence")
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(textState), &fields); err != nil || fields["snapshot_evidence"] == nil {
			t.Fatal("missing structured snapshot evidence", err)
		}
		if result.Requests[0].Questions[0].Criteria == nil || result.Requests[0].SHA256 != fingerprint(request) {
			t.Fatal("criteria missing from actual request provenance")
		}
	}
}

func TestExperimentRejectsUnknownQuestionsAndOversizedContext(t *testing.T) {
	for _, data := range []string{`{"version":1,"name":"x","context":"current","questions":{"unknown":"text"}}`, `{"version":1,"name":"x","context":"current","threshold":0}`, `{"version":1,"name":"x","context":"current"} {}`, `{"version":1,"name":"x","context":"current","state_format":"yaml"}`, `{"version":1,"name":"x","context":"current","criteria":{"unknown":{"true":"yes","false":"no"}}}`, `{"version":1,"name":"x","context":"current","criteria":{"removed-validation:removes-validation":{"true":"yes"}}}`} {
		file := filepath.Join(t.TempDir(), "experiment.json")
		if err := os.WriteFile(file, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadExperiment(file); err == nil {
			t.Fatal("invalid experiment accepted")
		}
	}
	if _, err := experimentEvidence(GitCase{BeforeFiles: map[string]string{"x.go": strings.Repeat("x", 65537)}}, &Experiment{Context: "snapshots"}); err == nil {
		t.Fatal("oversized context silently accepted")
	}
}

func TestContextFixturesSupplyPreviouslyInvisibleHelperBodies(t *testing.T) {
	cases, err := LoadGitCases("../../benchmarks/context-dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 4 {
		t.Fatal("missing paired context fixtures")
	}
	for _, c := range cases {
		for _, mode := range []string{"current", "snapshots"} {
			client := &benchmarkEvaluator{score: .5}
			_, err := runGitCaseExperiment(context.Background(), client, c, config.Defaults(), "isolated", &Experiment{Version: 1, Name: "context test", Context: mode})
			if err != nil {
				t.Fatal(err)
			}
			if len(client.requests) != 1 {
				t.Fatal("fixture must produce one changed-unit request")
			}
			hasHelper := strings.Contains(client.requests[0].State, "func ensurePermission(") || strings.Contains(client.requests[0].State, "func validateAmount(")
			if hasHelper != (mode == "snapshots") {
				t.Fatalf("%s/%s supplied incorrect helper evidence", c.ID, mode)
			}
		}
	}
}
