package jev

import (
	"context"
	"encoding/json"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStructuredStateAndCriteriaReachWireWithoutStringification(t *testing.T) {
	for _, state := range []string{`{"task":"keep checks","n":9007199254740993}`, `["before","after"]`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var wire wireRequest
			if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
				t.Error(err)
				return
			}
			if string(wire.State) != state || wire.Questions["q"].Criteria == nil || wire.Questions["q"].Criteria.False != "No guard is removed." {
				t.Errorf("wrong wire payload: %+v", wire)
			}
			_, _ = w.Write([]byte(`{"answers":{"q":{"type":"noul","noul":0.9}}}`))
		}))
		client, _ := NewOpenRouterClient(HTTPOptions{BaseURL: server.URL, APIKey: "test"})
		_, err := client.Evaluate(context.Background(), EvaluationRequest{Model: "model", StructuredState: json.RawMessage(state), Questions: []Question{{ID: "q", Instructions: "Is a guard removed?", Criteria: &decision.NoulCriteria{True: "A guard is removed.", False: "No guard is removed."}}}})
		server.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestInvalidStateOrCriteriaNeverDispatch(t *testing.T) {
	client, _ := NewHTTPClient(HTTPOptions{APIKey: "test", BaseURL: "http://127.0.0.1:1"})
	for _, state := range []string{`null`, `false`, `1`, `"text"`, `{`, `{} {}`} {
		_, err := client.Evaluate(context.Background(), EvaluationRequest{Model: "m", StructuredState: json.RawMessage(state), Questions: []Question{{ID: "q", Instructions: "Q?"}}})
		if err == nil {
			t.Fatalf("accepted invalid state %s", state)
		}
	}
	for _, request := range []EvaluationRequest{
		{Model: "m", State: "text", StructuredState: json.RawMessage(`{}`), Questions: []Question{{ID: "q", Instructions: "Q?"}}},
		{Model: "m", State: "text", Questions: []Question{{ID: "q", Instructions: "Q?", Criteria: &decision.NoulCriteria{True: "yes"}}}},
		{Model: "m", State: " ", Questions: []Question{{ID: "q", Instructions: "Q?"}}},
	} {
		if _, err := client.Evaluate(context.Background(), request); err == nil {
			t.Fatal("accepted ambiguous/empty request")
		}
	}
	if client.Stats().Requests != 0 {
		t.Fatal("invalid input reached network")
	}
}
