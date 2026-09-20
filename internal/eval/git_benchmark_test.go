package eval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
)

type benchmarkEvaluator struct {
	mu       sync.Mutex
	requests []decision.Request
	score    float64
	err      error
}

func TestQualityDevelopmentCorpusRetainsBothLabelsPerLanguage(t *testing.T) {
	cases, err := LoadGitCases("../../benchmarks/quality-dev")
	if err != nil {
		t.Fatal(err)
	}
	coverage := map[string]int{}
	for _, c := range cases {
		if c.Split != "dev" {
			t.Fatal("task-policy fixtures must not become holdout data")
		}
		for name := range c.AfterFiles {
			coverage[c.Rule+":"+semantic.Language(name)+":"+c.Expected]++
		}
	}
	if len(cases) != 24 || len(coverage) != 24 {
		t.Fatalf("missing rule/language/label coverage: %v", coverage)
	}
	if _, err := LoadGitCases("../../benchmarks/git-path"); err != nil {
		t.Fatal(err)
	}
}

func TestContractDevelopmentCorpusExercisesLabeledRules(t *testing.T) {
	cases, err := LoadGitCases("../../benchmarks/contract-dev")
	if err != nil {
		t.Fatal(err)
	}
	coverage := map[string]int{}
	for _, c := range cases {
		result, err := runGitCase(context.Background(), &benchmarkEvaluator{score: .5}, c, config.Defaults(), "isolated")
		if err != nil || result.Evaluated == nil || !*result.Evaluated {
			t.Fatalf("%s did not exercise its rule: %v", c.ID, err)
		}
		coverage[c.Rule+":"+c.Expected]++
	}
	if len(cases) != 12 || len(coverage) != 6 {
		t.Fatalf("missing contract coverage: %v", coverage)
	}
	for _, variant := range []string{"current", "questions", "ownership"} {
		if _, err := LoadExperiment("../../benchmarks/experiments/contracts-v2/" + variant + ".json"); err != nil {
			t.Fatal(err)
		}
	}
}

func (e *benchmarkEvaluator) Evaluate(_ context.Context, r decision.Request) (decision.Response, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.requests = append(e.requests, r)
	scores := map[string]float64{}
	for _, q := range r.Questions {
		scores[q.ID] = e.score
	}
	return decision.Response{Scores: scores}, e.err
}
func (*benchmarkEvaluator) Stats() decision.Stats { return decision.Stats{} }
func (*benchmarkEvaluator) Capabilities() decision.Capabilities {
	return decision.Capabilities{Batching: true, StructuredScores: true}
}

func TestGitBenchmarkCapturesInsertedBypassBelowThreshold(t *testing.T) {
	before := "package service\nfunc Read(fast bool) error {\n if !allowed() { return denied }\n return read()\n}\n"
	after := strings.Replace(before, " if !allowed()", " if fast { return read() }\n if !allowed()", 1)
	c := GitCase{PatchCase: PatchCase{ID: "insert", Rule: "removed-authorization-check", Task: "Preserve authorization", Expected: "positive", Split: "dev"}, BeforeFiles: map[string]string{"service with space.go": before}, AfterFiles: map[string]string{"service with space.go": after}}
	e := &benchmarkEvaluator{score: .8}
	result, err := runGitCase(context.Background(), e, c, config.Defaults(), "configured")
	if err != nil {
		t.Fatal(err)
	}
	if result.Evaluated == nil || !*result.Evaluated || result.Score != .8 || len(result.Observations) != 1 || len(result.Observations[0].Signals) != 2 {
		t.Fatalf("missing below-threshold evidence: %+v", result)
	}
	if result.Observations[0].File != "service with space.go" {
		t.Fatal(result.Observations[0])
	}
	batched := false
	for _, r := range e.requests {
		if strings.Contains(r.State, "+ if fast { return read() }") && strings.Contains(r.State, "PREVIOUS CODE") && len(r.Questions) > 2 {
			batched = true
		}
	}
	if !batched {
		t.Fatal("benchmark did not use real extracted diff and configured rule grouping")
	}
	m := Metrics(c.Rule, []ScoredCase{result}, .95)
	if m.FalseNegative != 1 {
		t.Fatalf("miss not counted: %+v", m)
	}
}

func TestGitBenchmarkRetainsEligibilityMisses(t *testing.T) {
	for _, reason := range []string{"new file", "disabled rule", "excluded file"} {
		t.Run(reason, func(t *testing.T) {
			c := GitCase{PatchCase: PatchCase{ID: reason, Rule: "removed-authorization-check", Task: "Check coverage", Expected: "positive", Split: "dev"}, BeforeFiles: map[string]string{}, AfterFiles: map[string]string{"service.go": "package service\nfunc Read() { read() }\n"}}
			cfg := config.Defaults()
			if reason != "new file" {
				c.BeforeFiles["service.go"] = "package service\nfunc Read() { authorize(); read() }\n"
			}
			if reason == "disabled rule" {
				rc := cfg.Rules[c.Rule]
				enabled := false
				rc.Enabled = &enabled
				cfg.Rules[c.Rule] = rc
			}
			if reason == "excluded file" {
				cfg.Exclude = append(cfg.Exclude, "service.go")
			}
			result, err := runGitCase(context.Background(), &benchmarkEvaluator{score: 1}, c, cfg, "configured")
			if err != nil {
				t.Fatal(err)
			}
			if result.Evaluated == nil || *result.Evaluated || result.CoverageReason == "" {
				t.Fatalf("lost coverage miss: %+v", result)
			}
			// A zero cutoff must never convert an unevaluated case into a detection.
			if m := Metrics(c.Rule, []ScoredCase{result}, 0); m.FalseNegative != 1 || m.TruePositive != 0 {
				t.Fatal(m)
			}
		})
	}
}

func TestGitBenchmarkUsesRepositoryContextAndIsolatedGrouping(t *testing.T) {
	before := "package service\nfunc Run() {}\n"
	after := before + "type Store interface { Put() }\n"
	c := GitCase{PatchCase: PatchCase{ID: "references", Rule: "unused-extensibility-point", Task: "Add storage", Expected: "negative", Split: "dev"}, BeforeFiles: map[string]string{"service.go": before, "other.go": "package service\nfunc Use(s Store) { s.Put() }\n"}, AfterFiles: map[string]string{"service.go": after, "other.go": "package service\nfunc Use(s Store) { s.Put() }\n"}}
	e := &benchmarkEvaluator{score: .1}
	result, err := runGitCase(context.Background(), e, c, config.Defaults(), "isolated")
	if err != nil {
		t.Fatal(err)
	}
	if !*result.Evaluated || len(e.requests) != 1 || len(e.requests[0].Questions) != 3 || !strings.Contains(e.requests[0].State, "REPOSITORY SEARCH EVIDENCE") || !strings.Contains(e.requests[0].State, "other.go:2:") {
		t.Fatalf("missing retrieval/grouping: %+v %+v", result, e.requests)
	}
}

func TestGitBenchmarkRejectsUnsafeSnapshots(t *testing.T) {
	for _, name := range []string{"../outside.go", "/outside.go", "C:/outside.go", ".git/config", "folder/.GIT/hooks/x", "a/../b.go", "a\\b.go", "folder./x.go"} {
		if safeFixturePath(name) {
			t.Errorf("accepted unsafe path %q", name)
		}
	}
	dir := t.TempDir()
	cases := []GitCase{{PatchCase: PatchCase{ID: "unsafe", Rule: "removed-validation", Task: "Validate", Expected: "positive", Rationale: "r", Provenance: "test", Split: "dev"}, BeforeFiles: map[string]string{}, AfterFiles: map[string]string{"../outside.go": "data"}}}
	data, _ := json.Marshal(cases)
	if err := os.WriteFile(filepath.Join(dir, "cases.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadGitCases(dir); err == nil {
		t.Fatal("unsafe fixture loaded")
	}
}

func TestGitBenchmarkProviderFailureCannotBecomeMiss(t *testing.T) {
	c := GitCase{PatchCase: PatchCase{ID: "failure", Rule: "removed-validation", Task: "Keep validation", Expected: "positive"}, BeforeFiles: map[string]string{"a.go": "package a\nfunc Run() { validate(); work() }\n"}, AfterFiles: map[string]string{"a.go": "package a\nfunc Run() { work() }\n"}}
	_, err := runGitCase(context.Background(), &benchmarkEvaluator{err: &decision.ContextLimitError{}}, c, config.Defaults(), "isolated")
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("provider failure treated as scored result: %v", err)
	}
}
