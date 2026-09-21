package jev

import (
	"context"
	"encoding/json"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
)

type Question struct {
	ID           string                 `json:"id"`
	Instructions string                 `json:"instructions"`
	Criteria     *decision.NoulCriteria `json:"criteria,omitempty"`
}

type EvaluationRequest struct {
	Model           string          `json:"model"`
	State           string          `json:"state"`
	Questions       []Question      `json:"questions"`
	StructuredState json.RawMessage `json:"structured_state,omitempty"`
}

type EvaluationResponse struct {
	Probabilities map[string]float64
	Model         string
	Provider      string
	RequestID     string
	Usage         *Usage
}

type Usage struct {
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	CostUSD      *float64 `json:"cost,omitempty"`
}

type Client interface {
	Evaluate(context.Context, EvaluationRequest) (EvaluationResponse, error)
	Stats() Stats
}

type Stats struct {
	UsageResponses int `json:"usage_responses"`
	Requests       int `json:"jev_requests"`
	Retries        int `json:"retries,omitempty"`
	InputTokens    int `json:"input_tokens,omitempty"`
	OutputTokens   int `json:"output_tokens,omitempty"`
}
