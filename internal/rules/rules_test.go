package rules

import "testing"

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

func TestComposeRejectsMissingSignal(t *testing.T) {
	rule := Rule{ID: "composite", Signals: []Signal{{ID: "first"}, {ID: "second"}}}
	if _, ok := rule.Compose(map[string]float64{"composite:first": 0.95}); ok {
		t.Fatal("composition succeeded with a missing required signal")
	}
}
