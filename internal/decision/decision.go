// Package decision defines provider-independent semantic scoring.
package decision

import (
	"context"
	"errors"
	"fmt"
	"math"
)

type Question struct {
	ID           string
	Instructions string
}
type Request struct {
	Model     string
	State     string
	Questions []Question
}
type Usage struct {
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	CostUSD      *float64 `json:"cost_usd,omitempty"`
}

// CallMetadata preserves provider-reported identity and usage. Missing values
// remain missing; requested model aliases are not resolved model identities.
type CallMetadata struct {
	ResolvedModel string `json:"resolved_model,omitempty"`
	Provider      string `json:"provider,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	Usage         *Usage `json:"usage,omitempty"`
}

type Response struct {
	Scores map[string]float64
	Calls  []CallMetadata
}
type Stats struct {
	UsageResponses int
	Requests       int
	Retries        int
	InputTokens    int
	OutputTokens   int
}
type Capabilities struct {
	MaxContextTokens *int   `json:"max_context_tokens"`
	Batching         bool   `json:"batching"`
	StructuredScores bool   `json:"structured_scores"`
	PrivacyMode      string `json:"privacy_mode"`
}
type Evaluator interface {
	Evaluate(context.Context, Request) (Response, error)
	Stats() Stats
	Capabilities() Capabilities
}

type ContextLimitError struct{ Cause error }

// Evaluate honors backend batching capabilities and validates normalized scores.
func Evaluate(ctx context.Context, evaluator Evaluator, request Request) (Response, error) {
	requests := []Request{request}
	if !evaluator.Capabilities().Batching {
		requests = nil
		for _, question := range request.Questions {
			requests = append(requests, Request{Model: request.Model, State: request.State, Questions: []Question{question}})
		}
	}
	out := Response{Scores: map[string]float64{}}
	for _, batch := range requests {
		response, err := evaluator.Evaluate(ctx, batch)
		if err != nil {
			return Response{}, err
		}
		out.Calls = append(out.Calls, response.Calls...)
		for _, question := range batch.Questions {
			score, ok := response.Scores[question.ID]
			if !ok || math.IsNaN(score) || math.IsInf(score, 0) || score < 0 || score > 1 {
				return Response{}, fmt.Errorf("invalid or missing normalized score for %s", question.ID)
			}
			out.Scores[question.ID] = score
		}
	}
	return out, nil
}

func (e *ContextLimitError) Error() string { return "provider context limit exceeded" }
func (e *ContextLimitError) Unwrap() error { return e.Cause }
func IsContextLimit(err error) bool        { var limit *ContextLimitError; return errors.As(err, &limit) }
