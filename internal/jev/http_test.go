package jev

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPClientSerializesBatchAndParsesNoulAnswers(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v1/systemone" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("unexpected request: %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body struct {
			State     string                  `json:"state"`
			Model     string                  `json:"model"`
			Questions map[string]wireQuestion `json:"questions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Questions) != 2 || body.Questions["a"].Type != "noul" || body.Model != "jev-latest" {
			t.Errorf("unexpected request body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"jev-1.13.0","answers":{"a":{"type":"noul","noul":0.93},"b":{"type":"noul","noul":0.12}},"usage":{"input_tokens":10,"output_tokens":2}}`))
	}))
	defer server.Close()
	client, err := NewHTTPClient(HTTPOptions{BaseURL: server.URL, APIKey: "secret", MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Evaluate(context.Background(), EvaluationRequest{
		Model: "jev-latest", State: "code",
		Questions: []Question{{ID: "a", Instructions: "A?"}, {ID: "b", Instructions: "B?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Probabilities["a"] != 0.93 || response.Probabilities["b"] != 0.12 {
		t.Fatalf("unexpected response: %#v", response)
	}
	if stats := client.Stats(); stats.Requests != 1 || stats.InputTokens != 10 || stats.OutputTokens != 2 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestHTTPClientRetriesTransientButNotPermanentFailures(t *testing.T) {
	t.Run("transient", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) == 1 {
				w.Header().Set("retry-after-ms", "0")
				http.Error(w, "rate limited", http.StatusTooManyRequests)
				return
			}
			_, _ = w.Write([]byte(`{"model":"jev-1.13.0","answers":{"a":{"type":"noul","noul":0.5}},"usage":{}}`))
		}))
		defer server.Close()
		client, _ := NewHTTPClient(HTTPOptions{BaseURL: server.URL, APIKey: "secret", MaxRetries: 2})
		client.sleep = func(context.Context, time.Duration) error { return nil }
		_, err := client.Evaluate(context.Background(), EvaluationRequest{
			Model: "jev-latest", State: "code", Questions: []Question{{ID: "a", Instructions: "A?"}},
		})
		if err != nil || calls.Load() != 2 || client.Stats().Retries != 1 {
			t.Fatalf("retry failed: calls=%d stats=%#v err=%v", calls.Load(), client.Stats(), err)
		}
	})
	t.Run("permanent", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.Header().Set("x-typesafe-request-id", "req-1")
			http.Error(w, "invalid key", http.StatusUnauthorized)
		}))
		defer server.Close()
		client, _ := NewHTTPClient(HTTPOptions{BaseURL: server.URL, APIKey: "secret", MaxRetries: 2})
		_, err := client.Evaluate(context.Background(), EvaluationRequest{
			Model: "jev-latest", State: "code", Questions: []Question{{ID: "a", Instructions: "A?"}},
		})
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.RequestID != "req-1" || calls.Load() != 1 {
			t.Fatalf("expected one permanent API failure, calls=%d err=%v", calls.Load(), err)
		}
	})
}

func TestHTTPClientCancellationAndMalformedResponse(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()
		client, _ := NewHTTPClient(HTTPOptions{BaseURL: server.URL, APIKey: "secret"})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := client.Evaluate(ctx, EvaluationRequest{
			Model: "jev-latest", State: "code", Questions: []Question{{ID: "a", Instructions: "A?"}},
		})
		if err == nil {
			t.Fatal("expected cancellation error")
		}
	})
	t.Run("malformed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"answers":{"a":{"type":"choice"}}}`))
		}))
		defer server.Close()
		client, _ := NewHTTPClient(HTTPOptions{BaseURL: server.URL, APIKey: "secret"})
		_, err := client.Evaluate(context.Background(), EvaluationRequest{
			Model: "jev-latest", State: "code", Questions: []Question{{ID: "a", Instructions: "A?"}},
		})
		if err == nil {
			t.Fatal("expected malformed answer error")
		}
	})
}

func TestHTTPClientRequiresAPIKey(t *testing.T) {
	if _, err := NewHTTPClient(HTTPOptions{}); err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestOpenRouterClientUsesDecisionsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/alpha/decisions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer openrouter-secret" {
			t.Errorf("unexpected authorization header")
		}
		if r.Header.Get("X-OpenRouter-Title") != "Reaper" {
			t.Errorf("missing OpenRouter app title")
		}
		var body wireRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "typesafe/jev-1.13" || body.Questions["rule"].Type != "noul" {
			t.Errorf("unexpected OpenRouter body: %#v", body)
		}
		_, _ = w.Write([]byte(`{"id":"gen-1","model":"typesafe/jev-1.13","provider":"TypeSafe","answers":{"rule":{"type":"noul","noul":0.87}},"usage":{"input_tokens":8,"output_tokens":0,"cost":0.0001}}`))
	}))
	defer server.Close()
	client, err := NewProviderClient("openrouter", HTTPOptions{
		BaseURL: server.URL, APIKey: "openrouter-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Evaluate(context.Background(), EvaluationRequest{
		Model: "typesafe/jev-1.13", State: "changed code",
		Questions: []Question{{ID: "rule", Instructions: "Is this undesirable?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Probabilities["rule"] != 0.87 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestOpenRouterClientRequiresItsOwnCredential(t *testing.T) {
	_, err := NewOpenRouterClient(HTTPOptions{})
	if err == nil || !strings.Contains(err.Error(), "OPENROUTER_API_KEY") {
		t.Fatalf("unexpected missing credential error: %v", err)
	}
}
