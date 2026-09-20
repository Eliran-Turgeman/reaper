// Package provider adapts backend protocols to the semantic decision API.
package provider

import (
	"context"
	"github.com/Eliran-Turgeman/reaper/internal/decision"
	"github.com/Eliran-Turgeman/reaper/internal/jev"
	"time"
)

type Options struct {
	BaseURL, APIKey string
	Timeout         time.Duration
	MaxRetries      int
}
type Jev struct{ Client jev.Client }

func New(name string, options Options) (decision.Evaluator, error) {
	client, err := jev.NewProviderClient(name, jev.HTTPOptions{BaseURL: options.BaseURL, APIKey: options.APIKey, Timeout: options.Timeout, MaxRetries: options.MaxRetries})
	if err != nil {
		return nil, err
	}
	return &Jev{Client: client}, nil
}

func (j *Jev) Evaluate(ctx context.Context, request decision.Request) (decision.Response, error) {
	wire := jev.EvaluationRequest{Model: request.Model, State: request.State}
	for _, q := range request.Questions {
		wire.Questions = append(wire.Questions, jev.Question{ID: q.ID, Instructions: q.Instructions})
	}
	response, err := j.Client.Evaluate(ctx, wire)
	if jev.IsTokenLimitError(err) {
		return decision.Response{}, &decision.ContextLimitError{Cause: err}
	}
	return decision.Response{Scores: response.Probabilities}, err
}
func (j *Jev) Stats() decision.Stats {
	s := j.Client.Stats()
	return decision.Stats{Requests: s.Requests, Retries: s.Retries, InputTokens: s.InputTokens, OutputTokens: s.OutputTokens, UsageResponses: s.UsageResponses}
}
func (j *Jev) Capabilities() decision.Capabilities {
	return decision.Capabilities{Batching: true, StructuredScores: true, PrivacyMode: "external-provider-managed"}
}
