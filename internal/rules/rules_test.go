package rules

import (
	"math"
	"testing"
)

func TestComposeUsesWeakestRequiredSignal(t *testing.T) {
	rule := Rule{
		ID: "composite",
		Signals: []Signal{
			{ID: "first"},
			{ID: "second"},
			{ID: "third"},
		},
	}
	score, ok := rule.Compose(map[string]float64{
		"composite:first":  0.97,
		"composite:second": 0.84,
		"composite:third":  0.93,
	})
	if !ok || score != 0.84 {
		t.Fatalf("got score=%v ok=%v, want score=0.84 ok=true", score, ok)
	}
}

func TestComposeSeparatesPermissionFromFactualSignals(t *testing.T) {
	rule := Rule{ID: "r", Signals: []Signal{{ID: "factual"}, {ID: "permission", Negate: true}}}
	for _, tc := range []struct{ fact, permission, want float64 }{{.9, .02, .9}, {.9, .96, .04}, {.3, .1, .3}} {
		score, ok := rule.Compose(map[string]float64{"r:factual": tc.fact, "r:permission": tc.permission})
		if !ok || math.Abs(score-tc.want) > 1e-9 {
			t.Fatal(tc, score, ok)
		}
	}
	for _, invalid := range []float64{-1, 2, math.NaN(), math.Inf(1)} {
		if _, ok := rule.Compose(map[string]float64{"r:factual": .9, "r:permission": invalid}); ok {
			t.Fatal("invalid permission accepted", invalid)
		}
	}
}

func TestComposeRejectsMissingSignal(t *testing.T) {
	rule := Rule{ID: "composite", Signals: []Signal{{ID: "first"}, {ID: "second"}}}
	if _, ok := rule.Compose(map[string]float64{"composite:first": 0.95}); ok {
		t.Fatal("composition succeeded with a missing required signal")
	}
}
