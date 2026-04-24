package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type zepGraphNode struct {
	UUID       string         `json:"uuid"`
	UUIDAlt    string         `json:"uuid_"`
	Name       string         `json:"name"`
	Labels     []string       `json:"labels"`
	Summary    string         `json:"summary"`
	Attributes map[string]any `json:"attributes"`
	CreatedAt  any            `json:"created_at"`
}

type zepGraphEdge struct {
	UUID           string         `json:"uuid"`
	UUIDAlt        string         `json:"uuid_"`
	Name           string         `json:"name"`
	Fact           string         `json:"fact"`
	SourceNodeUUID string         `json:"source_node_uuid"`
	TargetNodeUUID string         `json:"target_node_uuid"`
	Attributes     map[string]any `json:"attributes"`
	CreatedAt      any            `json:"created_at"`
	ValidAt        any            `json:"valid_at"`
	InvalidAt      any            `json:"invalid_at"`
	ExpiredAt      any            `json:"expired_at"`
	Episodes       any            `json:"episodes"`
	EpisodeIDs     any            `json:"episode_ids"`
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type ZepClient struct {
	baseURL string
	apiKey  string
	client  HTTPDoer
}

func NewZepClient(baseURL, apiKey string, client HTTPDoer) *ZepClient {
	if client == nil {
		client = &http.Client{}
	}
	return &ZepClient{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, client: client}
}

func (z *ZepClient) AddFact(ctx context.Context, fact Fact) error {
	payload := map[string]any{
		"graph_id": fact.GraphID,
		"data":     fact.Data,
		"type":     "text",
	}
	return z.do(ctx, http.MethodPost, "graph", payload, nil)
}

func (z *ZepClient) GetFacts(ctx context.Context, graphID string, limit int) ([]Fact, error) {
	var payload struct {
		Episodes []struct {
			UUID      string `json:"uuid"`
			GraphID   string `json:"graph_id"`
			Data      string `json:"data"`
			CreatedAt string `json:"created_at"`
		} `json:"episodes"`
	}
	if err := z.do(ctx, http.MethodGet, fmt.Sprintf("graph/episodes/graph/%s?lastn=%d", graphID, limit), nil, &payload); err != nil {
		return nil, err
	}
	out := make([]Fact, 0, len(payload.Episodes))
	for _, ep := range payload.Episodes {
		createdAt, _ := time.Parse(time.RFC3339, ep.CreatedAt)
		out = append(out, Fact{ID: ep.UUID, GraphID: ep.GraphID, Data: ep.Data, CreatedAt: createdAt})
	}
	return out, nil
}

func (z *ZepClient) SearchGraph(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	var payload struct {
		Edges []Edge   `json:"edges"`
		Nodes []Node   `json:"nodes"`
		Facts []string `json:"facts"`
	}
	body := map[string]any{
		"query":    req.Query,
		"graph_id": req.GraphID,
		"limit":    req.Limit,
		"scope":    req.Scope,
	}
	if err := z.do(ctx, http.MethodPost, "graph/search", body, &payload); err != nil {
		return SearchResponse{}, err
	}
	return SearchResponse{Nodes: payload.Nodes, Edges: payload.Edges, Facts: payload.Facts}, nil
}

func (z *ZepClient) DeleteNode(ctx context.Context, nodeID string) error {
	return z.do(ctx, http.MethodDelete, "graph/node/"+nodeID, nil, nil)
}

func (z *ZepClient) GetGraphData(ctx context.Context, graphID string) (GraphData, error) {
	nodes, err := z.fetchAllNodes(ctx, graphID)
	if err != nil {
		return GraphData{}, err
	}
	edges, err := z.fetchAllEdges(ctx, graphID)
	if err != nil {
		return GraphData{}, err
	}

	nodeNameByID := map[string]string{}
	nodesData := make([]GraphNode, 0, len(nodes))
	for _, node := range nodes {
		id := firstNonEmptyString(node.UUIDAlt, node.UUID)
		nodeNameByID[id] = node.Name
		nodesData = append(nodesData, GraphNode{
			UUID:       id,
			Name:       node.Name,
			Labels:     node.Labels,
			Summary:    node.Summary,
			Attributes: node.Attributes,
			CreatedAt:  node.CreatedAt,
		})
	}

	edgesData := make([]GraphEdge, 0, len(edges))
	for _, edge := range edges {
		edgesData = append(edgesData, GraphEdge{
			UUID:           firstNonEmptyString(edge.UUIDAlt, edge.UUID),
			Name:           edge.Name,
			Fact:           edge.Fact,
			FactType:       firstNonEmptyString(edge.Name, ""),
			SourceNodeUUID: edge.SourceNodeUUID,
			TargetNodeUUID: edge.TargetNodeUUID,
			SourceNodeName: nodeNameByID[edge.SourceNodeUUID],
			TargetNodeName: nodeNameByID[edge.TargetNodeUUID],
			Attributes:     edge.Attributes,
			CreatedAt:      edge.CreatedAt,
			ValidAt:        edge.ValidAt,
			InvalidAt:      edge.InvalidAt,
			ExpiredAt:      edge.ExpiredAt,
			Episodes:       normalizeEpisodes(edge.Episodes, edge.EpisodeIDs),
		})
	}

	return GraphData{
		GraphID:   graphID,
		Nodes:     nodesData,
		Edges:     edgesData,
		NodeCount: len(nodesData),
		EdgeCount: len(edgesData),
	}, nil
}

func (z *ZepClient) DeleteGraph(ctx context.Context, graphID string) error {
	return z.do(ctx, http.MethodDelete, "graph/"+graphID, nil, nil)
}

func (z *ZepClient) fetchAllNodes(ctx context.Context, graphID string) ([]zepGraphNode, error) {
	var all []zepGraphNode
	var cursor string
	for {
		payload := map[string]any{"limit": 100}
		if cursor != "" {
			payload["uuid_cursor"] = cursor
		}
		var batch []zepGraphNode
		if err := z.do(ctx, http.MethodPost, "graph/node/graph/"+graphID, payload, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
		cursor = firstNonEmptyString(batch[len(batch)-1].UUIDAlt, batch[len(batch)-1].UUID)
		if cursor == "" {
			break
		}
	}
	return all, nil
}

func (z *ZepClient) fetchAllEdges(ctx context.Context, graphID string) ([]zepGraphEdge, error) {
	var all []zepGraphEdge
	var cursor string
	for {
		payload := map[string]any{"limit": 100}
		if cursor != "" {
			payload["uuid_cursor"] = cursor
		}
		var batch []zepGraphEdge
		if err := z.do(ctx, http.MethodPost, "graph/edge/graph/"+graphID, payload, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
		cursor = firstNonEmptyString(batch[len(batch)-1].UUIDAlt, batch[len(batch)-1].UUID)
		if cursor == "" {
			break
		}
	}
	return all, nil
}

func normalizeEpisodes(values ...any) []string {
	var result []string
	for _, value := range values {
		switch typed := value.(type) {
		case []any:
			for _, item := range typed {
				result = append(result, fmt.Sprint(item))
			}
		case []string:
			result = append(result, typed...)
		case string:
			if typed != "" {
				result = append(result, typed)
			}
		}
	}
	return result
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (z *ZepClient) do(ctx context.Context, method, relPath string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, z.baseURL+"/"+strings.TrimLeft(relPath, "/"), reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Api-Key "+z.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := z.client.Do(req)
	if err != nil {
		return &Error{Op: "Do", Kind: ErrMemoryUnavailable, Err: err}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Error{Op: "ReadAll", Kind: ErrMemoryUnavailable, Err: err}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return &Error{Op: method + " " + relPath, Kind: ErrMemoryUnauthorized, StatusCode: resp.StatusCode}
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return &Error{Op: method + " " + relPath, Kind: ErrMemoryUnavailable, StatusCode: resp.StatusCode, RetryAfter: retryAfter(resp.Header.Get("Retry-After"))}
	}
	if resp.StatusCode >= 400 {
		return &Error{Op: method + " " + relPath, Kind: ErrMemoryInvalid, StatusCode: resp.StatusCode, Err: fmt.Errorf(strings.TrimSpace(string(raw)))}
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return &Error{Op: "unmarshal", Kind: ErrMemoryInvalid, Err: err}
		}
	}
	return nil
}

func retryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	d, err := time.ParseDuration(value + "s")
	if err != nil {
		return 0
	}
	return d
}
