package decision

import (
	"context"
	"fmt"
	"math"
	"testing"
)

type singleEvaluator struct {
	score float64
	calls int
}

func (e *singleEvaluator) Evaluate(_ context.Context, r Request) (Response, error) {
	e.calls++
	if len(r.Questions) != 1 {
		return Response{}, fmt.Errorf("batch unsupported")
	}
	return Response{Scores: map[string]float64{r.Questions[0].ID: e.score}, Calls: []CallMetadata{{RequestID: r.Questions[0].ID}}}, nil
}
func (e *singleEvaluator) Stats() Stats               { return Stats{Requests: e.calls} }
func (e *singleEvaluator) Capabilities() Capabilities { return Capabilities{StructuredScores: true} }
func TestUnbatchedBackendAndInvalidScores(t *testing.T) {
	e := &singleEvaluator{score: .8}
	request := Request{Questions: []Question{{ID: "a"}, {ID: "b"}}}
	response, err := Evaluate(context.Background(), e, request)
	if err != nil || e.calls != 2 || response.Scores["a"] != .8 || response.Scores["b"] != .8 {
		t.Fatal(response, err)
	}
	if len(response.Calls) != 2 || response.Calls[0].RequestID != "a" || response.Calls[1].RequestID != "b" {
		t.Fatal("split calls lost provenance", response.Calls)
	}
	for _, score := range []float64{-1, 2, math.NaN(), math.Inf(1)} {
		e.score = score
		if _, err := Evaluate(context.Background(), e, request); err == nil {
			t.Fatal("accepted invalid score", score)
		}
	}
}
