package memory

import (
	"context"
	"errors"
	"time"
)

type Node struct {
	UUID       string         `json:"uuid"`
	Name       string         `json:"name"`
	Labels     []string       `json:"labels"`
	Summary    string         `json:"summary"`
	Attributes map[string]any `json:"attributes"`
}

type Edge struct {
	UUID           string         `json:"uuid"`
	Name           string         `json:"name"`
	Fact           string         `json:"fact"`
	SourceNodeUUID string         `json:"source_node_uuid"`
	TargetNodeUUID string         `json:"target_node_uuid"`
	Attributes     map[string]any `json:"attributes"`
}

type GraphNode struct {
	UUID       string         `json:"uuid"`
	Name       string         `json:"name"`
	Labels     []string       `json:"labels"`
	Summary    string         `json:"summary"`
	Attributes map[string]any `json:"attributes"`
	CreatedAt  any            `json:"created_at"`
}

type GraphEdge struct {
	UUID           string         `json:"uuid"`
	Name           string         `json:"name"`
	Fact           string         `json:"fact"`
	FactType       string         `json:"fact_type"`
	SourceNodeUUID string         `json:"source_node_uuid"`
	TargetNodeUUID string         `json:"target_node_uuid"`
	SourceNodeName string         `json:"source_node_name"`
	TargetNodeName string         `json:"target_node_name"`
	Attributes     map[string]any `json:"attributes"`
	CreatedAt      any            `json:"created_at"`
	ValidAt        any            `json:"valid_at"`
	InvalidAt      any            `json:"invalid_at"`
	ExpiredAt      any            `json:"expired_at"`
	Episodes       []string       `json:"episodes"`
}

type GraphData struct {
	GraphID   string      `json:"graph_id"`
	Nodes     []GraphNode `json:"nodes"`
	Edges     []GraphEdge `json:"edges"`
	NodeCount int         `json:"node_count"`
	EdgeCount int         `json:"edge_count"`
}

type Fact struct {
	ID        string    `json:"id"`
	GraphID   string    `json:"graph_id"`
	Data      string    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

type SearchRequest struct {
	Query   string `json:"query"`
	GraphID string `json:"graph_id"`
	Limit   int    `json:"limit"`
	Scope   string `json:"scope,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
}

type SearchResponse struct {
	Nodes      []Node   `json:"nodes"`
	Edges      []Edge   `json:"edges"`
	Facts      []string `json:"facts"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

type Client interface {
	AddFact(context.Context, Fact) error
	GetFacts(context.Context, string, int) ([]Fact, error)
	SearchGraph(context.Context, SearchRequest) (SearchResponse, error)
	DeleteNode(context.Context, string) error
	GetGraphData(context.Context, string) (GraphData, error)
	DeleteGraph(context.Context, string) error
}

var (
	ErrMemoryUnauthorized = errors.New("memory unauthorized")
	ErrMemoryUnavailable  = errors.New("memory unavailable")
	ErrMemoryInvalid      = errors.New("memory invalid response")
)

type Error struct {
	Op         string
	Kind       error
	StatusCode int
	RetryAfter time.Duration
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Kind != nil {
		return e.Kind.Error()
	}
	return "memory error"
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
