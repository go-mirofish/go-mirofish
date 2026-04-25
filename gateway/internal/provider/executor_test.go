package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func newHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func TestExecutorExecute(t *testing.T) {
	tests := []struct {
		name        string
		client      *http.Client
		request     ProviderRequest
		expectErr   error
		expectText  string
		retryBudget int
	}{
		{
			name: "success",
			client: newHTTPClient(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader(`{"model":"gpt","choices":[{"finish_reason":"stop","message":{"content":"hello"}}]}`)),
				}, nil
			}),
			request: ProviderRequest{
				Model:       "gpt",
				Messages:    []Message{{Role: RoleUser, Content: "hi"}},
				MaxTokens:   32,
				Temperature: 0,
			},
			expectText: "hello",
		},
		{
			name: "retries on 429",
			client: newHTTPClient(func() roundTripFunc {
				attempts := 0
				return func(r *http.Request) (*http.Response, error) {
					attempts++
					if attempts == 1 {
						return &http.Response{
							StatusCode: http.StatusTooManyRequests,
							Header:     http.Header{"Retry-After": []string{"1"}},
							Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"slow down","code":"rate_limit"}}`)),
						}, nil
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"model":"gpt","choices":[{"finish_reason":"stop","message":{"content":"ok"}}]}`)),
					}, nil
				}
			}()),
			request: ProviderRequest{
				Model:       "gpt",
				Messages:    []Message{{Role: RoleUser, Content: "hi"}},
				MaxTokens:   32,
				Temperature: 0,
			},
			expectText: "ok",
		},
		{
			name: "circuit open after failures",
			client: newHTTPClient(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"boom"}}`)),
				}, nil
			}),
			request: ProviderRequest{
				Model:       "gpt",
				Messages:    []Message{{Role: RoleUser, Content: "hi"}},
				MaxTokens:   32,
				Temperature: 0,
			},
			expectErr: ErrServer,
		},
		{
			name: "empty content rejected",
			client: newHTTPClient(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"model":"gpt","choices":[{"finish_reason":"stop","message":{"content":""}}]}`)),
				}, nil
			}),
			request: ProviderRequest{
				Model:       "gpt",
				Messages:    []Message{{Role: RoleUser, Content: "hi"}},
				MaxTokens:   32,
				Temperature: 0,
			},
			expectErr: ErrInvalidResponse,
		},
		{
			name: "json null content rejected (not fmt.Sprint nil)",
			client: newHTTPClient(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"model":"gpt","choices":[{"finish_reason":"stop","message":{"content":null}}]}`)),
				}, nil
			}),
			request: ProviderRequest{
				Model:       "gpt",
				Messages:    []Message{{Role: RoleUser, Content: "hi"}},
				MaxTokens:   32,
				Temperature: 0,
			},
			expectErr: ErrInvalidResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewExecutor(Config{
				BaseURL:             "https://example.test",
				APIKey:              "token",
				DefaultModel:        "gpt",
				MaxRetries:          1,
				RetryBackoff:        time.Millisecond,
				CircuitFailures:     1,
				CircuitResetTimeout: time.Second,
			}, tt.client)

			resp, err := exec.Execute(context.Background(), tt.request)
			if tt.expectErr != nil {
				if err == nil {
					t.Fatalf("expected error")
				}
				var providerErr *Error
				if !errors.As(err, &providerErr) {
					t.Fatalf("expected provider.Error, got %T", err)
				}
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("expected error kind %v, got %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Content != tt.expectText {
				t.Fatalf("expected %q, got %q", tt.expectText, resp.Content)
			}
		})
	}
}

func TestCircuitOpen(t *testing.T) {
	client := newHTTPClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"boom"}}`)),
		}, nil
	})
	exec := NewExecutor(Config{
		BaseURL:             "https://example.test",
		APIKey:              "token",
		DefaultModel:        "gpt",
		MaxRetries:          0,
		RetryBackoff:        time.Millisecond,
		CircuitFailures:     1,
		CircuitResetTimeout: time.Hour,
	}, client)

	req := ProviderRequest{Model: "gpt", Messages: []Message{{Role: RoleUser, Content: "hi"}}, MaxTokens: 16}
	if _, err := exec.Execute(context.Background(), req); err == nil {
		t.Fatalf("expected first failure")
	}
	if _, err := exec.Execute(context.Background(), req); err == nil || !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected circuit open, got %v", err)
	}
}
