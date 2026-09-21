package runner

import (
	"context"
	"encoding/json"
	"github.com/Eliran-Turgeman/reaper/internal/cache"
	"github.com/Eliran-Turgeman/reaper/internal/config"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
	"testing"
)

func TestStateRepresentationAndCriteriaSeparateCacheEntries(t *testing.T) {
	client := &mockClient{}
	r := Runner{Config: config.Defaults(), Client: client, Cache: cache.NewMemory()}
	unit := semantic.Unit{FilePath: "x.go", Language: "go", NewContent: "TASK\nsource text", OldContent: "old", Diff: "+new", SurroundingCode: "body"}
	rule, _ := rules.Get("narrating-comment")
	rule.Signals = append([]rules.Signal(nil), rule.Signals...)
	for _, format := range []string{"", "json-text", "json-object"} {
		r.StateFormat = format
		for repeat := 0; repeat < 2; repeat++ {
			if result := r.evaluate(context.Background(), unit, []rules.Rule{rule}, "real task"); result.err != nil {
				t.Fatal(result.err)
			}
		}
	}
	if len(client.requests) != 3 {
		t.Fatalf("representations shared a cache entry: %d", len(client.requests))
	}
	var fields map[string]string
	if err := json.Unmarshal(client.requests[2].StructuredState, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["current_code"] != unit.NewContent || fields["task"] != "real task" || string(client.requests[2].StructuredState) != client.requests[1].State || client.requests[2].State != "" {
		t.Fatal("state facts changed", fields)
	}
	for _, boundary := range []string{"first", "second"} {
		rule.Signals[0].Criteria = &decision.NoulCriteria{True: boundary, False: "no"}
		if result := r.evaluate(context.Background(), unit, []rules.Rule{rule}, "real task"); result.err != nil {
			t.Fatal(result.err)
		}
	}
	if len(client.requests) != 5 || client.requests[4].Questions[0].Criteria.True != "second" {
		t.Fatal("criteria change reused a cached score")
	}
}
