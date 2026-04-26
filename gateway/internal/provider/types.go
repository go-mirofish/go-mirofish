package provider

import (
	"context"
	"errors"
	"time"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type ResponseFormatType string

const (
	ResponseFormatJSONObject ResponseFormatType = "json_object"
)

type ResponseFormat struct {
	Type ResponseFormatType `json:"type"`
}

// ProviderKind identifies the type of inference server.
type ProviderKind string

const (
	KindOllama       ProviderKind = "ollama"
	KindLMStudio     ProviderKind = "lm-studio"
	KindLlamaCpp     ProviderKind = "llama-cpp"
	KindOpenAICompat ProviderKind = "openai-compatible"
	KindCloud        ProviderKind = "cloud"
)

// ModelTier classifies a model by approximate size / capability.
// Higher value = more capable.
type ModelTier int

const (
	TierUnknown ModelTier = iota
	TierTiny              // <3 B params — very fast, limited reasoning
	TierSmall             // 3–9 B — solid for most tasks
	TierMedium            // 10–20 B — good structured output
	TierLarge             // >20 B — best local quality
	TierCloud             // hosted API — highest quality, external cost
)

// TaskHint gives the router a signal about what kind of task is being performed
// so it can prefer more capable providers when needed.
type TaskHint string

const (
	TaskOntology TaskHint = "ontology" // complex structured JSON → needs ≥ small
	TaskProfiles TaskHint = "profiles" // structured JSON → needs ≥ small
	TaskReport   TaskHint = "report"   // long-form text → any tier
	TaskGeneral  TaskHint = "general"  // anything else → any tier
)

type Message struct {
	Role    Role   `json:"role" validate:"required,oneof=system user assistant"`
	Content string `json:"content" validate:"required"`
}

type ProviderRequest struct {
	Model          string          `json:"model" validate:"required"`
	Messages       []Message       `json:"messages" validate:"required,min=1,dive"`
	Temperature    float64         `json:"temperature"`
	MaxTokens      int             `json:"max_tokens" validate:"required,min=1"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty" validate:"omitempty"`
	// TaskHint is optional metadata for the registry router; ignored by OpenAIExecutor.
	TaskHint TaskHint `json:"-"`
}

type ProviderResponse struct {
	Content       string        `json:"content"`
	FinishReason  string        `json:"finish_reason,omitempty"`
	Provider      string        `json:"provider"`
	Model         string        `json:"model"`
	StatusCode    int           `json:"status_code"`
	Duration      time.Duration `json:"duration"`
	RetryCount    int           `json:"retry_count"`
	CorrelationID string        `json:"correlation_id,omitempty"`
}

type Executor interface {
	Execute(ctx context.Context, req ProviderRequest) (ProviderResponse, error)
}

var (
	ErrCircuitOpen     = errors.New("provider circuit open")
	ErrTimeout         = errors.New("provider timeout")
	ErrUnavailable     = errors.New("provider unavailable")
	ErrInvalidResponse = errors.New("provider invalid response")
	ErrClient          = errors.New("provider client error")
	ErrServer          = errors.New("provider server error")
)

type Error struct {
	Op          string
	Kind        error
	StatusCode  int
	RetryAfter  time.Duration
	Provider    string
	Message     string
	Correlation string
	Err         error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Kind != nil {
		return e.Kind.Error()
	}
	return "provider error"
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Err != nil {
		return e.Err
	}
	return e.Kind
}
