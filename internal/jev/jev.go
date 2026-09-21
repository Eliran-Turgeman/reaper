package jev

import "context"

type Question struct {
	ID           string `json:"id"`
	Instructions string `json:"instructions"`
}

type EvaluationRequest struct {
	Model     string     `json:"model"`
	State     string     `json:"state"`
	Questions []Question `json:"questions"`
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
