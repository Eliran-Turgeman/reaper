package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"testing"
)

type singleEvaluator struct {
	score    float64
	calls    int
	requests []Request
}

func (e *singleEvaluator) Evaluate(_ context.Context, r Request) (Response, error) {
	e.calls++
	e.requests = append(e.requests, r)
	if len(r.Questions) != 1 {
		return Response{}, fmt.Errorf("batch unsupported")
	}
	return Response{Scores: map[string]float64{r.Questions[0].ID: e.score}, Calls: []CallMetadata{{RequestID: r.Questions[0].ID}}}, nil
}

func TestSplitRequestsPreserveStructuredStateAndCriteria(t *testing.T) {
	e := &singleEvaluator{score: .8}
	state := json.RawMessage(`{"before":"guard","after":"operation"}`)
	criteria := &NoulCriteria{True: "yes", False: "no"}
	_, err := Evaluate(context.Background(), e, Request{Model: "m", StructuredState: state, Questions: []Question{{ID: "a", Criteria: criteria}, {ID: "b", Criteria: criteria}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range e.requests {
		if string(request.StructuredState) != string(state) || request.Questions[0].Criteria != criteria {
			t.Fatal("split lost question input", request)
		}
	}
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
