package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Eliran-Turgeman/repear/internal/cache"
	"github.com/Eliran-Turgeman/repear/internal/config"
	"github.com/Eliran-Turgeman/repear/internal/jev"
	"github.com/Eliran-Turgeman/repear/internal/semantic"
)

type mockClient struct {
	mu       sync.Mutex
	requests []jev.EvaluationRequest
	delay    map[string]time.Duration
	failures map[string]error
}

func (m *mockClient) Evaluate(_ context.Context, request jev.EvaluationRequest) (jev.EvaluationResponse, error) {
	if delay := m.delay[request.State]; delay > 0 {
		time.Sleep(delay)
	}
	m.mu.Lock()
	m.requests = append(m.requests, request)
	m.mu.Unlock()
	if err := m.failures[request.State]; err != nil {
		return jev.EvaluationResponse{}, err
	}
	scores := map[string]float64{}
	for _, question := range request.Questions {
		scores[question.ID] = 0.95
	}
	return jev.EvaluationResponse{Probabilities: scores}, nil
}

func (m *mockClient) Stats() jev.Stats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return jev.Stats{Requests: len(m.requests)}
}

func TestRunnerBatchesRulesAppliesPolicyAndCaches(t *testing.T) {
	cfg := config.Defaults()
	client := &mockClient{}
	store := cache.NewMemory()
	engine := Runner{Config: cfg, Client: client, Cache: store}
	unit := semantic.Unit{
		FilePath: "client_test.go", Language: "go", Diff: "+// Loop over values.\n+for range values {}",
		NewContent: "// Loop over values.\nfor range values {}", OldContent: "assert.Equal(t, 2, got)",
		StartLine: 10, EndLine: 12, IsTest: true, ExistingModified: true,
		ContainsComments: true,
	}
	report, err := engine.Run(context.Background(), []semantic.Unit{unit}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("rules were not batched: %d requests", len(client.requests))
	}
	if got := len(client.requests[0].Questions); got != 9 {
		t.Fatalf("got %d batched questions, want 9", got)
	}
	if len(report.Diagnostics) != 9 || report.Summary.Errors != 2 || report.Summary.Warnings != 7 {
		t.Fatalf("unexpected policy result: %#v", report)
	}
	report, err = engine.Run(context.Background(), []semantic.Unit{unit}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 1 || report.Summary.CacheHits != 9 {
		t.Fatalf("cache was not used: requests=%d summary=%#v", len(client.requests), report.Summary)
	}
}

func TestRunnerSortsDiagnosticsDespiteConcurrentCompletion(t *testing.T) {
	cfg := config.Defaults()
	disabled := false
	for id, rc := range cfg.Rules {
		if id != "defensive-fallback" {
			rc.Enabled = &disabled
			cfg.Rules[id] = rc
		}
	}
	first := semantic.Unit{FilePath: "z.go", Language: "go", Diff: "+x", NewContent: "x", StartLine: 8, EndLine: 8}
	second := semantic.Unit{FilePath: "a.go", Language: "go", Diff: "+y", NewContent: "y", StartLine: 3, EndLine: 3}
	client := &mockClient{delay: map[string]time.Duration{buildState(first, "", false): 20 * time.Millisecond}}
	engine := Runner{Config: cfg, Client: client, Cache: cache.Disabled{}}
	report, err := engine.Run(context.Background(), []semantic.Unit{first, second}, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Diagnostics[0].File != "a.go" || report.Diagnostics[1].File != "z.go" {
		t.Fatalf("diagnostics not sorted: %#v", report.Diagnostics)
	}
}

func TestRunnerSkipsTaskRuleWithoutTaskAndHonorsGlobExcludes(t *testing.T) {
	cfg := config.Defaults()
	for id, rc := range cfg.Rules {
		enabled := id == "scope-creep" || id == "defensive-fallback"
		rc.Enabled = &enabled
		cfg.Rules[id] = rc
	}
	unit := semantic.Unit{FilePath: "vendor/deep/client.go"}
	engine := Runner{Config: cfg, Cache: cache.Disabled{}}
	if got := engine.WorkCount([]semantic.Unit{unit}, ""); got != 0 {
		t.Fatalf("got %d jobs, expected excluded file and skipped task rule", got)
	}
}

func TestRunnerExcludesNestedBuildAndDependencyDirectories(t *testing.T) {
	cfg := config.Defaults()
	engine := Runner{Config: cfg, Cache: cache.Disabled{}}
	for _, file := range []string{
		"EmailCollector.Api/wwwroot/lib/bootstrap/dist/js/bootstrap.js",
		"client/vendor/library/source.go",
		"web/node_modules/package/index.js",
	} {
		if got := engine.WorkCount([]semantic.Unit{{FilePath: file}}, ""); got != 0 {
			t.Fatalf("got %d jobs for excluded file %s", got, file)
		}
	}
}

func TestRunnerLogsAndContinuesAfterProviderTokenLimit(t *testing.T) {
	cfg := config.Defaults()
	disabled := false
	for id, rc := range cfg.Rules {
		if id != "defensive-fallback" {
			rc.Enabled = &disabled
			cfg.Rules[id] = rc
		}
	}
	large := semantic.Unit{
		FilePath: "large.go", Language: "go", NewContent: "large",
		StartLine: 1, EndLine: 100,
	}
	small := semantic.Unit{
		FilePath: "small.go", Language: "go", NewContent: "small",
		StartLine: 1, EndLine: 1,
	}
	client := &mockClient{failures: map[string]error{
		buildState(large, "", false): &jev.APIError{
			Provider: "OpenRouter", StatusCode: 400,
			Body: `{"detail":{"error_type":"max_tokens_exceeded"}}`,
		},
	}}
	var notices []string
	engine := Runner{
		Config: cfg, Client: client, Cache: cache.Disabled{},
		Notice: func(format string, args ...any) {
			notices = append(notices, fmt.Sprintf(format, args...))
		},
	}
	report, err := engine.Run(context.Background(), []semantic.Unit{large, small}, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.UnitsSkipped != 1 || report.Summary.UnitsEvaluated != 1 ||
		report.Summary.SemanticChecks != 1 || len(client.requests) != 2 {
		t.Fatalf("unexpected report after skip: %#v requests=%d", report, len(client.requests))
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "skip large.go:1") {
		t.Fatalf("unexpected skip notices: %#v", notices)
	}
}

func TestRunnerCacheIsIsolatedByProvider(t *testing.T) {
	cfg := config.Defaults()
	disabled := false
	for id, rc := range cfg.Rules {
		if id != "defensive-fallback" {
			rc.Enabled = &disabled
			cfg.Rules[id] = rc
		}
	}
	unit := semantic.Unit{FilePath: "client.go", Language: "go", Diff: "+x", NewContent: "x"}
	client := &mockClient{}
	store := cache.NewMemory()
	engine := Runner{Config: cfg, Client: client, Cache: store}
	if _, err := engine.Run(context.Background(), []semantic.Unit{unit}, ""); err != nil {
		t.Fatal(err)
	}
	engine.Config.Provider = config.ProviderOpenRouter
	engine.Config.Model = config.DefaultModel(config.ProviderOpenRouter)
	if _, err := engine.Run(context.Background(), []semantic.Unit{unit}, ""); err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 2 {
		t.Fatalf("provider change reused a cache entry: requests=%d", len(client.requests))
	}
}

func TestRunnerCacheIsIsolatedByReaperVersion(t *testing.T) {
	cfg := config.Defaults()
	disabled := false
	for id, rc := range cfg.Rules {
		if id != "defensive-fallback" {
			rc.Enabled = &disabled
			cfg.Rules[id] = rc
		}
	}
	unit := semantic.Unit{FilePath: "client.go", Language: "go", Diff: "+x", NewContent: "x"}
	client := &mockClient{}
	store := cache.NewMemory()
	engine := Runner{Config: cfg, Client: client, Cache: store, Version: "0.1.0"}
	if _, err := engine.Run(context.Background(), []semantic.Unit{unit}, ""); err != nil {
		t.Fatal(err)
	}
	engine.Version = "0.2.0"
	if _, err := engine.Run(context.Background(), []semantic.Unit{unit}, ""); err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 2 {
		t.Fatalf("Reaper version change reused a cache entry: requests=%d", len(client.requests))
	}
}

func TestWorkCountDoesNotDuplicateVerboseSkipLogs(t *testing.T) {
	cfg := config.Defaults()
	for id, rc := range cfg.Rules {
		enabled := id == "redundant-comment"
		rc.Enabled = &enabled
		cfg.Rules[id] = rc
	}
	logs := 0
	engine := Runner{
		Config: cfg,
		Cache:  cache.Disabled{},
		Verbose: func(string, ...any) {
			logs++
		},
	}
	unit := semantic.Unit{FilePath: "client.go", Language: "go"}
	if got := engine.WorkCount([]semantic.Unit{unit}, ""); got != 0 {
		t.Fatalf("got %d jobs, want 0", got)
	}
	if logs != 0 {
		t.Fatalf("planning emitted %d verbose messages", logs)
	}
	if _, err := engine.Run(context.Background(), []semantic.Unit{unit}, ""); err != nil {
		t.Fatal(err)
	}
	if logs != 1 {
		t.Fatalf("run emitted %d verbose messages, want 1", logs)
	}
}

func TestRunnerDebugLogsEveryEvaluationWithRawConfidence(t *testing.T) {
	cfg := config.Defaults()
	disabled := false
	for id, rc := range cfg.Rules {
		if id != "defensive-fallback" {
			rc.Enabled = &disabled
		} else {
			rc.Threshold = 0.99
		}
		cfg.Rules[id] = rc
	}
	var logs []string
	engine := Runner{
		Config: cfg,
		Client: &mockClient{},
		Cache:  cache.NewMemory(),
		Debug:  true,
		Verbose: func(format string, args ...any) {
			logs = append(logs, fmt.Sprintf(format, args...))
		},
	}
	unit := semantic.Unit{
		FilePath: "client.go", Language: "go", Diff: "+return nil",
		NewContent: "return nil", StartLine: 7, EndLine: 7,
	}
	report, err := engine.Run(context.Background(), []semantic.Unit{unit}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 0 {
		t.Fatalf("below-threshold evaluation became a diagnostic: %#v", report.Diagnostics)
	}
	const expected = "evaluation defensive-fallback for client.go:7 confidence=0.95 threshold=0.99 result=pass source=provider"
	if !strings.Contains(strings.Join(logs, "\n"), expected) {
		t.Fatalf("debug output does not contain %q: %#v", expected, logs)
	}

	logs = nil
	if _, err := engine.Run(context.Background(), []semantic.Unit{unit}, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(logs, "\n"), "result=pass source=cache") {
		t.Fatalf("cached debug evaluation was not identified: %#v", logs)
	}
}

func TestRunnerAuditSkipsRulesThatRequireChangeContext(t *testing.T) {
	cfg := config.Defaults()
	var logs []string
	client := &mockClient{}
	engine := Runner{
		Config: cfg,
		Client: client,
		Cache:  cache.Disabled{},
		Audit:  true,
		Verbose: func(format string, args ...any) {
			logs = append(logs, fmt.Sprintf(format, args...))
		},
	}
	unit := semantic.Unit{
		FilePath: "client_test.go", Language: "go",
		Diff: "+// Explains why this is required.", NewContent: "// Explains why this is required.",
		StartLine: 1, EndLine: 1, IsTest: true, ContainsComments: true, ExistingModified: true,
	}
	report, err := engine.Run(context.Background(), []semantic.Unit{unit}, "some task")
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.SemanticChecks != 8 {
		t.Fatalf("got %d audit checks, want 8", report.Summary.SemanticChecks)
	}
	if len(client.requests) != 1 || len(client.requests[0].Questions) != 8 {
		t.Fatalf("unexpected audit request: %#v", client.requests)
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "skip weakened-test") || !strings.Contains(joined, "skip scope-creep") {
		t.Fatalf("change-context rules were not logged as skipped: %s", joined)
	}
	if !strings.Contains(client.requests[0].State, "MODE\nEXISTING CODE AUDIT") ||
		strings.Contains(client.requests[0].State, "\nDIFF\n") {
		t.Fatalf("unexpected audit state: %s", client.requests[0].State)
	}
	for _, question := range client.requests[0].Questions {
		if !strings.HasPrefix(question.Instructions, "Evaluate the current code as an existing-code audit") {
			t.Fatalf("audit instructions missing for %s: %s", question.ID, question.Instructions)
		}
	}
}

func TestRunnerEvaluatesSemanticRulesForDeletionOnlyHunk(t *testing.T) {
	cfg := config.Defaults()
	enabledRules := map[string]bool{
		"speculative-generality":   true,
		"pass-through-layer":       true,
		"complexity-pushed-upward": true,
		"information-leakage":      true,
		"special-general-mixture":  true,
		"shallow-module":           true,
		"defensive-fallback":       true,
	}
	for id, rc := range cfg.Rules {
		enabled := enabledRules[id]
		rc.Enabled = &enabled
		cfg.Rules[id] = rc
	}
	client := &mockClient{}
	engine := Runner{Config: cfg, Client: client, Cache: cache.Disabled{}}
	unit := semantic.Unit{
		FilePath: "client.go", Language: "go",
		Diff: "-if err != nil {\n-\treturn err\n-}", OldContent: "if err != nil {\n\treturn err\n}",
		StartLine: 10, EndLine: 10, ExistingModified: true,
	}
	if _, err := engine.Run(context.Background(), []semantic.Unit{unit}, ""); err != nil {
		t.Fatal(err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("got %d requests, want 1", len(client.requests))
	}
	if got := len(client.requests[0].Questions); got != len(enabledRules) {
		t.Fatalf("got %d questions, want %d", got, len(enabledRules))
	}
}
