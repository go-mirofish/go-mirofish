package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Config struct {
	BaseURL             string
	APIKey              string
	DefaultModel        string
	Timeout             time.Duration
	MaxRetries          int
	RetryBackoff        time.Duration
	CircuitFailures     int
	CircuitResetTimeout time.Duration
	ProviderName        string
}

type OpenAIExecutor struct {
	cfg    Config
	client HTTPDoer
	// breakerMu protects breaker state across concurrent Execute calls.
	breakerMu sync.Mutex
	failures  int
	openedAt  time.Time
}

type completionRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    float64         `json:"temperature"`
	MaxTokens      int             `json:"max_tokens"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type completionResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content any `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

func NewExecutor(cfg Config, client HTTPDoer) *OpenAIExecutor {
	if cfg.Timeout == 0 {
		cfg.Timeout = 120 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff == 0 {
		cfg.RetryBackoff = 2 * time.Second
	}
	if cfg.CircuitFailures == 0 {
		cfg.CircuitFailures = 3
	}
	if cfg.CircuitResetTimeout == 0 {
		cfg.CircuitResetTimeout = 30 * time.Second
	}
	if cfg.ProviderName == "" {
		cfg.ProviderName = "openai-compatible"
	}
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &OpenAIExecutor{cfg: cfg, client: client}
}

func (e *OpenAIExecutor) Execute(ctx context.Context, req ProviderRequest) (ProviderResponse, error) {
	if req.Model == "" {
		req.Model = e.cfg.DefaultModel
	}
	if err := e.beforeRequest(); err != nil {
		return ProviderResponse{}, err
	}

	payload := completionRequest{
		Model:          req.Model,
		Messages:       req.Messages,
		Temperature:    req.Temperature,
		MaxTokens:      req.MaxTokens,
		ResponseFormat: req.ResponseFormat,
	}

	var lastErr error
	for attempt := 0; attempt <= e.cfg.MaxRetries; attempt++ {
		started := time.Now()
		resp, err := e.doCompletion(ctx, payload)
		if err == nil {
			e.recordSuccess()
			return ProviderResponse{
				Content:      resp.Content,
				FinishReason: resp.FinishReason,
				Provider:     e.cfg.ProviderName,
				Model:        payload.Model,
				StatusCode:   resp.StatusCode,
				Duration:     time.Since(started),
				RetryCount:   attempt,
			}, nil
		}
		lastErr = err
		if !shouldRetry(err) || attempt == e.cfg.MaxRetries {
			e.recordFailure()
			break
		}
		if waitErr := sleepWithContext(ctx, e.cfg.RetryBackoff*time.Duration(attempt+1)); waitErr != nil {
			e.recordFailure()
			return ProviderResponse{}, &Error{Op: "Execute", Kind: ErrTimeout, Provider: e.cfg.ProviderName, Err: waitErr}
		}
	}

	if lastErr == nil {
		lastErr = &Error{Op: "Execute", Kind: ErrUnavailable, Provider: e.cfg.ProviderName, Message: "provider execution failed"}
	}
	return ProviderResponse{}, lastErr
}

func (e *OpenAIExecutor) beforeRequest() error {
	e.breakerMu.Lock()
	defer e.breakerMu.Unlock()
	if e.failures < e.cfg.CircuitFailures {
		return nil
	}
	if time.Since(e.openedAt) >= e.cfg.CircuitResetTimeout {
		e.failures = 0
		e.openedAt = time.Time{}
		return nil
	}
	return &Error{
		Op:       "Execute",
		Kind:     ErrCircuitOpen,
		Provider: e.cfg.ProviderName,
		Message:  "provider circuit is open",
	}
}

func (e *OpenAIExecutor) recordSuccess() {
	e.breakerMu.Lock()
	defer e.breakerMu.Unlock()
	e.failures = 0
	e.openedAt = time.Time{}
}

func (e *OpenAIExecutor) recordFailure() {
	e.breakerMu.Lock()
	defer e.breakerMu.Unlock()
	e.failures++
	if e.failures >= e.cfg.CircuitFailures && e.openedAt.IsZero() {
		e.openedAt = time.Now()
	}
}

func (e *OpenAIExecutor) doCompletion(ctx context.Context, payload completionRequest) (struct {
	Content      string
	FinishReason string
	StatusCode   int
}, error) {
	var empty struct {
		Content      string
		FinishReason string
		StatusCode   int
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return empty, &Error{Op: "marshal", Kind: ErrClient, Provider: e.cfg.ProviderName, Err: err}
	}

	base := strings.TrimRight(e.cfg.BaseURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return empty, &Error{Op: "request", Kind: ErrClient, Provider: e.cfg.ProviderName, Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+e.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := e.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return empty, &Error{Op: "Do", Kind: ErrTimeout, Provider: e.cfg.ProviderName, Err: ctx.Err()}
		}
		return empty, &Error{Op: "Do", Kind: ErrUnavailable, Provider: e.cfg.ProviderName, Err: err}
	}
	defer httpResp.Body.Close()

	raw, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return empty, &Error{Op: "ReadAll", Kind: ErrUnavailable, Provider: e.cfg.ProviderName, Err: err}
	}

	if httpResp.StatusCode >= 400 {
		var parsed completionResponse
		_ = json.Unmarshal(raw, &parsed)
		errKind := ErrClient
		if httpResp.StatusCode >= 500 {
			errKind = ErrServer
		}
		return empty, &Error{
			Op:         "chat.completions",
			Kind:       errKind,
			Provider:   e.cfg.ProviderName,
			StatusCode: httpResp.StatusCode,
			Message:    chooseErrorMessage(parsed.Error, raw),
			RetryAfter: parseRetryAfter(httpResp.Header.Get("Retry-After")),
		}
	}

	var parsed completionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return empty, &Error{Op: "unmarshal", Kind: ErrInvalidResponse, Provider: e.cfg.ProviderName, Err: err}
	}
	if len(parsed.Choices) == 0 {
		return empty, &Error{Op: "choices", Kind: ErrInvalidResponse, Provider: e.cfg.ProviderName, Message: "provider returned no choices"}
	}
	content := normalizeContent(parsed.Choices[0].Message.Content)
	if content == "" {
		return empty, &Error{Op: "content", Kind: ErrInvalidResponse, Provider: e.cfg.ProviderName, Message: "provider returned empty content"}
	}
	return struct {
		Content      string
		FinishReason string
		StatusCode   int
	}{
		Content:      content,
		FinishReason: parsed.Choices[0].FinishReason,
		StatusCode:   httpResp.StatusCode,
	}, nil
}

func normalizeContent(content any) string {
	switch value := content.(type) {
	case string:
		return strings.TrimSpace(value)
	case []any:
		var parts []string
		for _, item := range value {
			if part, ok := item.(string); ok && strings.TrimSpace(part) != "" {
				parts = append(parts, strings.TrimSpace(part))
				continue
			}
			if part, ok := item.(map[string]any); ok {
				if text, _ := part["text"].(string); strings.TrimSpace(text) != "" {
					parts = append(parts, strings.TrimSpace(text))
				}
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return strings.TrimSpace(fmt.Sprint(content))
	}
}

func chooseErrorMessage(errPayload *struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}, raw []byte) string {
	if errPayload != nil && errPayload.Message != "" {
		return errPayload.Message
	}
	return strings.TrimSpace(string(raw))
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := time.ParseDuration(value + "s"); err == nil {
		return seconds
	}
	return 0
}

func shouldRetry(err error) bool {
	var providerErr *Error
	if !errors.As(err, &providerErr) {
		return false
	}
	if providerErr.Kind == ErrTimeout || providerErr.Kind == ErrUnavailable {
		return true
	}
	if providerErr.StatusCode == http.StatusTooManyRequests || providerErr.Kind == ErrServer {
		return true
	}
	return false
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
