package provider

import (
	"context"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/jev"
	"testing"
)

type fakeJev struct{ fail bool }

func (f fakeJev) Evaluate(_ context.Context, r jev.EvaluationRequest) (jev.EvaluationResponse, error) {
	if f.fail {
		return jev.EvaluationResponse{}, &jev.APIError{Body: "context_length_exceeded"}
	}
	return jev.EvaluationResponse{Probabilities: map[string]float64{r.Questions[0].ID: .8}}, nil
}
func (fakeJev) Stats() jev.Stats { return jev.Stats{Requests: 1} }
func TestJevAdapterNormalizesScoresAndLimits(t *testing.T) {
	adapter := Jev{Client: fakeJev{}}
	r, err := adapter.Evaluate(context.Background(), decision.Request{Questions: []decision.Question{{ID: "x"}}})
	if err != nil || r.Scores["x"] != .8 || !adapter.Capabilities().StructuredScores {
		t.Fatal(r, err)
	}
	adapter.Client = fakeJev{fail: true}
	_, err = adapter.Evaluate(context.Background(), decision.Request{})
	if !decision.IsContextLimit(err) {
		t.Fatal(err)
	}
}
