package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type GraphSpec struct {
	GraphID     string `json:"graph_id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type EntityAttributeSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type SourceTargetSpec struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type EntityTypeSpec struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Attributes  []EntityAttributeSpec `json:"attributes"`
}

type EdgeTypeSpec struct {
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	Attributes    []EntityAttributeSpec `json:"attributes"`
	SourceTargets []SourceTargetSpec    `json:"source_targets"`
}

type OntologySpec struct {
	EntityTypes []EntityTypeSpec `json:"entity_types"`
	EdgeTypes   []EdgeTypeSpec   `json:"edge_types"`
}

type Episode struct {
	Data      string `json:"data"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at,omitempty"`
	GraphID   string `json:"graph_id,omitempty"`
}

func (z *ZepClient) CreateGraph(ctx context.Context, spec GraphSpec) error {
	return z.do(ctx, "POST", "graph/create", spec, nil)
}

func (z *ZepClient) SubmitOntology(ctx context.Context, graphID string, ontology OntologySpec) error {
	payload := map[string]any{
		"graph_ids":    []string{graphID},
		"entity_types": ontology.EntityTypes,
		"edge_types":   ontology.EdgeTypes,
	}
	return z.do(ctx, "PUT", "entity-types", payload, nil)
}

func (z *ZepClient) IngestEpisode(ctx context.Context, episode Episode) error {
	payload := map[string]any{
		"data":       episode.Data,
		"type":       episode.Type,
		"graph_id":   episode.GraphID,
		"created_at": episode.CreatedAt,
	}
	return z.do(ctx, "POST", "graph", payload, nil)
}

func (z *ZepClient) IngestBatch(ctx context.Context, graphID string, episodes []Episode) error {
	payloadEpisodes := make([]map[string]any, 0, len(episodes))
	for _, episode := range episodes {
		payloadEpisodes = append(payloadEpisodes, map[string]any{
			"data":       episode.Data,
			"type":       episode.Type,
			"created_at": episode.CreatedAt,
		})
	}
	return z.do(ctx, "POST", "graph-batch", map[string]any{
		"graph_id": graphID,
		"episodes": payloadEpisodes,
	}, nil)
}

func ChunkText(text string, chunkSize int, overlap int) []string {
	if chunkSize <= 0 {
		chunkSize = 500
	}
	if overlap < 0 {
		overlap = 0
	}
	if len(text) <= chunkSize {
		if text == "" {
			return nil
		}
		return []string{text}
	}
	var chunks []string
	start := 0
	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunk := text[start:end]
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end == len(text) {
			break
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}

func EncodeEpisodes(chunks []string, graphID string) []Episode {
	now := time.Now().Format(time.RFC3339)
	out := make([]Episode, 0, len(chunks))
	for _, chunk := range chunks {
		out = append(out, Episode{
			Data:      chunk,
			Type:      "text",
			GraphID:   graphID,
			CreatedAt: now,
		})
	}
	return out
}

func ToOntologySpec(payload map[string]any) (OntologySpec, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return OntologySpec{}, err
	}
	var spec OntologySpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return OntologySpec{}, err
	}
	return spec, nil
}

func (z *ZepClient) CreateGraphID() string {
	return fmt.Sprintf("go_mirofish_%d", time.Now().UnixNano())
}
