package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/go-playground/validator/v10"
)

type OntologyBuilder struct {
	validate *validator.Validate
	gen      ContentGenerator
}

func NewBuilder(gen ContentGenerator) *OntologyBuilder {
	return &OntologyBuilder{
		validate: validator.New(),
		gen:      gen,
	}
}

func (b *OntologyBuilder) Build(ctx context.Context, input BuildInput) (Ontology, error) {
	return b.buildWithDetails(ctx, input)
}

func (b *OntologyBuilder) prompt(input BuildInput) string {
	var sb strings.Builder
	sb.WriteString("Simulation requirement:\n")
	sb.WriteString(input.SimulationRequirement)
	sb.WriteString("\n\nSource text:\n")
	sb.WriteString(input.SourceText)
	if input.AdditionalContext != "" {
		sb.WriteString("\n\nAdditional context:\n")
		sb.WriteString(input.AdditionalContext)
	}
	return sb.String()
}

var ontologySystemPrompt = `Return ONLY a compact JSON object. No prose, no code fences.

Required top-level fields:
  entity_types  – array of exactly 10 objects
  edge_types    – array of 6-10 objects
  analysis_summary – string

Each entity_type object must have:
  name        – PascalCase string
  description – string
  attributes  – array of {name, type, description} objects (type is "string", "int", etc.)

Rules for entity_types:
  - exactly 10 items total
  - first 8: concrete actor types specific to the scenario
  - last 2 must be exactly "Person" and "Organization" (in that order)

Each edge_type object must have:
  name           – UPPER_SNAKE_CASE string
  description    – string
  source_targets – array of {"source":"EntityName","target":"EntityName"} objects (NOT arrays, NOT strings)
  attributes     – optional array of {name, type, description} objects

Rules for edge_types:
  - 6-10 items total
  - source_targets elements MUST be objects with "source" and "target" string fields
  - at most 10 source_targets per edge_type

Output ONLY valid JSON. No explanation. No markdown. No code blocks.`

func sanitizeJSON(text string) string {
	text = strings.TrimSpace(text)
	// Strip thinking-model <think>...</think> blocks (Gemini 2.5, DeepSeek, etc.)
	text = regexp.MustCompile(`(?si)<think>.*?</think>`).ReplaceAllString(text, "")
	// Strip markdown code fences.
	text = regexp.MustCompile(`(?i)^` + "```" + `(?:json)?\s*`).ReplaceAllString(text, "")
	text = regexp.MustCompile("(?m)\\s*```\\s*$").ReplaceAllString(text, "")
	text = strings.TrimSpace(text)
	// Extract outermost JSON object if there is surrounding prose.
	if first := strings.Index(text, "{"); first >= 0 {
		if last := strings.LastIndex(text, "}"); last > first {
			text = text[first : last+1]
		}
	}
	return strings.TrimSpace(text)
}

func (b *OntologyBuilder) buildWithDetails(ctx context.Context, input BuildInput) (Ontology, error) {
	if err := b.validate.Struct(input); err != nil {
		return Ontology{}, fmt.Errorf("builder.Build: %w", ErrValidation)
	}
	raw, err := b.gen.Execute(ctx, ontologySystemPrompt, b.prompt(input))
	if err != nil {
		return Ontology{}, err
	}
	sanitized := sanitizeJSON(raw)
	if sanitized == "" || sanitized == "<nil>" {
		return Ontology{}, fmt.Errorf("ontology LLM returned empty or unusable content (raw len=%d); if using OpenAI-compatible APIs, ensure the model supports response_format json_object and returns a message (not null content)", len(raw))
	}
	var out Ontology
	if err := json.Unmarshal([]byte(sanitized), &out); err != nil {
		preview := sanitized
		if len(preview) > 300 {
			preview = preview[:300] + "…"
		}
		trim := strings.TrimSpace(sanitized)
		if len(trim) > 0 && trim[0] == '<' {
			return Ontology{}, fmt.Errorf("ontology model returned non-JSON (HTML/XML?); check LLM_BASE_URL, key, and model — %w; preview: %s", err, preview)
		}
		return Ontology{}, fmt.Errorf("ontology JSON parse failed: %w — raw preview: %s", err, preview)
	}
	out = normalizeOntology(out)
	if err := b.validate.Struct(out); err != nil {
		return Ontology{}, fmt.Errorf("builder.Build: %w", ErrValidation)
	}
	return out, nil
}

func normalizeOntology(in Ontology) Ontology {
	for i := range in.EntityTypes {
		in.EntityTypes[i].Name = toPascal(in.EntityTypes[i].Name)
		in.EntityTypes[i].Description = shorten(in.EntityTypes[i].Description, 100)
		in.EntityTypes[i].Attributes = dedupeAttrs(in.EntityTypes[i].Attributes)
	}
	seenEdges := map[string]bool{}
	var edgeOut []EdgeType
	for _, edge := range in.EdgeTypes {
		edge.Name = toUpperSnake(edge.Name)
		if seenEdges[edge.Name] {
			continue
		}
		seenEdges[edge.Name] = true
		edge.Description = shorten(edge.Description, 100)
		edge.Attributes = dedupeAttrs(edge.Attributes)
		edge.SourceTargets = dedupeSourceTargets(edge.SourceTargets)
		edgeOut = append(edgeOut, edge)
	}
	in.EdgeTypes = edgeOut
	sort.SliceStable(in.EntityTypes, func(i, j int) bool {
		a, b := in.EntityTypes[i].Name, in.EntityTypes[j].Name
		if a == "Person" || a == "Organization" || b == "Person" || b == "Organization" {
			order := map[string]int{"Person": 98, "Organization": 99}
			return order[a] < order[b]
		}
		return a < b
	})
	return in
}

func dedupeAttrs(attrs []EntityAttribute) []EntityAttribute {
	seen := map[string]bool{}
	out := make([]EntityAttribute, 0, len(attrs))
	for _, attr := range attrs {
		attr.Name = toSnake(attr.Name)
		if seen[attr.Name] {
			continue
		}
		seen[attr.Name] = true
		attr.Description = shorten(attr.Description, 100)
		if attr.Type == "" {
			attr.Type = "text"
		}
		out = append(out, attr)
	}
	return out
}

func dedupeSourceTargets(items []SourceTarget) []SourceTarget {
	seen := map[string]bool{}
	out := make([]SourceTarget, 0, len(items))
	for _, item := range items {
		key := item.Source + "->" + item.Target
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
		if len(out) == 10 {
			break
		}
	}
	return out
}

func toPascal(value string) string {
	parts := regexp.MustCompile(`[^a-zA-Z0-9]+`).Split(value, -1)
	var out strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		out.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			out.WriteString(strings.ToLower(part[1:]))
		}
	}
	if out.Len() == 0 {
		return "Unknown"
	}
	return out.String()
}

func toSnake(value string) string {
	value = regexp.MustCompile(`([a-z0-9])([A-Z])`).ReplaceAllString(value, "${1}_${2}")
	value = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(value, "_")
	return strings.Trim(strings.ToLower(value), "_")
}

func toUpperSnake(value string) string {
	return strings.ToUpper(toSnake(value))
}

func shorten(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return strings.TrimSpace(value[:max-3]) + "..."
}
