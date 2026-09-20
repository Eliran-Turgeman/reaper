package eval

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/config"
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

func TestExperimentRejectsUnknownQuestionsAndOversizedContext(t *testing.T) {
	for _, data := range []string{`{"version":1,"name":"x","context":"current","questions":{"unknown":"text"}}`, `{"version":1,"name":"x","context":"current","threshold":0}`, `{"version":1,"name":"x","context":"current"} {}`} {
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
