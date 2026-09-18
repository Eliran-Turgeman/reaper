package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBaseURL           = "https://api.typesafe.ai"
	OpenRouterDefaultBaseURL = "https://openrouter.ai"
)

type HTTPClient struct {
	baseURL    string
	endpoint   string
	provider   string
	apiKey     string
	httpClient *http.Client
	maxRetries int
	mu         sync.Mutex
	stats      Stats
	sleep      func(context.Context, time.Duration) error
}

type HTTPOptions struct {
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
	MaxRetries int
	HTTPClient *http.Client
}

type APIError struct {
	Provider   string
	StatusCode int
	RequestID  string
	Body       string
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("%s API returned HTTP %d", e.Provider, e.StatusCode)
	if e.RequestID != "" {
		message += " (request " + e.RequestID + ")"
	}
	if e.Body != "" {
		message += ": " + e.Body
	}
	return message
}

func NewHTTPClient(options HTTPOptions) (*HTTPClient, error) {
	return newHTTPClient(options, "TypeSafe", DefaultBaseURL, "/v1/systemone", "TYPESAFE_API_KEY")
}

func NewOpenRouterClient(options HTTPOptions) (*HTTPClient, error) {
	return newHTTPClient(options, "OpenRouter", OpenRouterDefaultBaseURL, "/api/alpha/decisions", "OPENROUTER_API_KEY")
}

func NewProviderClient(provider string, options HTTPOptions) (*HTTPClient, error) {
	switch provider {
	case "typesafe":
		return NewHTTPClient(options)
	case "openrouter":
		return NewOpenRouterClient(options)
	default:
		return nil, fmt.Errorf("unsupported evaluation provider %q", provider)
	}
}

func newHTTPClient(options HTTPOptions, provider, defaultBaseURL, endpoint, keyEnv string) (*HTTPClient, error) {
	if strings.TrimSpace(options.APIKey) == "" {
		return nil, fmt.Errorf("%s is required when semantic evaluation through %s is needed", keyEnv, provider)
	}
	if options.BaseURL == "" {
		options.BaseURL = defaultBaseURL
	}
	if options.Timeout <= 0 {
		options.Timeout = 30 * time.Second
	}
	if options.MaxRetries < 0 {
		return nil, errors.New("max retries cannot be negative")
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: options.Timeout}
	}
	return &HTTPClient{
		baseURL: strings.TrimRight(options.BaseURL, "/"), endpoint: endpoint,
		provider: provider, apiKey: options.APIKey,
		httpClient: client, maxRetries: options.MaxRetries, sleep: sleepContext,
	}, nil
}

func (c *HTTPClient) Evaluate(ctx context.Context, request EvaluationRequest) (EvaluationResponse, error) {
	if request.Model == "" || request.State == "" || len(request.Questions) == 0 {
		return EvaluationResponse{}, errors.New("model, state, and at least one question are required")
	}
	wire := wireRequest{State: request.State, Model: request.Model, Questions: map[string]wireQuestion{}}
	for _, question := range request.Questions {
		if question.ID == "" || strings.TrimSpace(question.Instructions) == "" {
			return EvaluationResponse{}, errors.New("question ID and instructions are required")
		}
		if _, exists := wire.Questions[question.ID]; exists {
			return EvaluationResponse{}, fmt.Errorf("duplicate question ID %q", question.ID)
		}
		wire.Questions[question.ID] = wireQuestion{Type: "noul", Instructions: question.Instructions}
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return EvaluationResponse{}, fmt.Errorf("encode Jev request: %w", err)
	}

	for attempt := 0; ; attempt++ {
		response, retryDelay, err := c.do(ctx, body, request.Questions)
		if err == nil {
			return response, nil
		}
		var apiErr *APIError
		apiFailure := errors.As(err, &apiErr)
		transient := apiFailure && retryable(apiErr.StatusCode)
		if !apiFailure {
			var netErr net.Error
			transient = errors.As(err, &netErr) && ctx.Err() == nil
		}
		if !transient || attempt >= c.maxRetries {
			return EvaluationResponse{}, err
		}
		c.mu.Lock()
		c.stats.Retries++
		c.mu.Unlock()
		if retryDelay <= 0 || retryDelay > time.Minute {
			retryDelay = backoff(attempt)
		}
		if err := c.sleep(ctx, retryDelay); err != nil {
			return EvaluationResponse{}, err
		}
	}
}

func (c *HTTPClient) do(ctx context.Context, body []byte, questions []Question) (EvaluationResponse, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+c.endpoint, bytes.NewReader(body))
	if err != nil {
		return EvaluationResponse{}, 0, fmt.Errorf("create Jev request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.provider == "OpenRouter" {
		req.Header.Set("X-OpenRouter-Title", "Reaper")
	}
	c.mu.Lock()
	c.stats.Requests++
	c.mu.Unlock()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return EvaluationResponse{}, 0, fmt.Errorf("send Jev request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return EvaluationResponse{}, 0, fmt.Errorf("read Jev response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return EvaluationResponse{}, retryAfter(resp.Header), &APIError{
			Provider:   c.provider,
			StatusCode: resp.StatusCode,
			RequestID:  requestID(resp.Header),
			Body:       compactError(data),
		}
	}
	var wireResp wireResponse
	if err := json.Unmarshal(data, &wireResp); err != nil {
		return EvaluationResponse{}, 0, fmt.Errorf("decode Jev response: %w", err)
	}
	result := EvaluationResponse{Probabilities: make(map[string]float64, len(questions))}
	for _, question := range questions {
		answer, ok := wireResp.Answers[question.ID]
		if !ok {
			return EvaluationResponse{}, 0, fmt.Errorf("Jev response omitted answer %q", question.ID)
		}
		if answer.Type != "noul" || answer.Noul == nil {
			return EvaluationResponse{}, 0, fmt.Errorf("Jev answer %q is not a noul probability", question.ID)
		}
		if *answer.Noul < 0 || *answer.Noul > 1 {
			return EvaluationResponse{}, 0, fmt.Errorf("Jev answer %q is outside [0,1]", question.ID)
		}
		result.Probabilities[question.ID] = *answer.Noul
	}
	c.mu.Lock()
	c.stats.InputTokens += wireResp.Usage.InputTokens
	c.stats.OutputTokens += wireResp.Usage.OutputTokens
	c.mu.Unlock()
	return result, 0, nil
}

func requestID(header http.Header) string {
	for _, name := range []string{"x-typesafe-request-id", "x-openrouter-request-id", "x-request-id"} {
		if value := header.Get(name); value != "" {
			return value
		}
	}
	return ""
}

func (c *HTTPClient) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stats
}

type wireRequest struct {
	State     string                  `json:"state"`
	Model     string                  `json:"model"`
	Questions map[string]wireQuestion `json:"questions"`
}

type wireQuestion struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria,omitempty"`
}

type wireResponse struct {
	Model   string                `json:"model"`
	Answers map[string]wireAnswer `json:"answers"`
	Usage   wireUsage             `json:"usage"`
}

type wireAnswer struct {
	Type string   `json:"type"`
	Noul *float64 `json:"noul,omitempty"`
}

type wireUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func retryable(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500
}

func retryAfter(header http.Header) time.Duration {
	if value := header.Get("retry-after-ms"); value != "" {
		ms, err := strconv.ParseInt(value, 10, 64)
		if err == nil && ms >= 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	value := header.Get("Retry-After")
	if value == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		return time.Until(date)
	}
	return 0
}

func backoff(attempt int) time.Duration {
	delay := 500 * time.Millisecond * time.Duration(1<<min(attempt, 3))
	jitter := time.Duration(rand.Int64N(max(int64(delay/4), 1)))
	return delay - jitter
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func compactError(data []byte) string {
	value := strings.TrimSpace(string(data))
	if len(value) > 512 {
		return value[:512] + "..."
	}
	return value
}
