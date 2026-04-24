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
	if err := b.validate.Struct(input); err != nil {
		return Ontology{}, fmt.Errorf("builder.Build: %w", ErrValidation)
	}
	raw, err := b.gen.Execute(ctx, ontologySystemPrompt, b.prompt(input))
	if err != nil {
		return Ontology{}, err
	}
	raw = sanitizeJSON(raw)
	var out Ontology
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return Ontology{}, err
	}
	out = normalizeOntology(out)
	if err := b.validate.Struct(out); err != nil {
		return Ontology{}, fmt.Errorf("builder.Build: %w", ErrValidation)
	}
	return out, nil
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

var ontologySystemPrompt = `Return compact JSON with entity_types, edge_types, and analysis_summary.
Rules:
- exactly 10 entity types
- last 2 must be Person and Organization
- first 8 must be concrete actor types
- 6-10 relationship types
- relationship names must be UPPER_SNAKE_CASE
- source_targets per relationship must not exceed 10`

func sanitizeJSON(text string) string {
	text = strings.TrimSpace(text)
	text = regexp.MustCompile("(?i)^```(?:json)?\\s*").ReplaceAllString(text, "")
	text = regexp.MustCompile("(?m)\\s*```\\s*$").ReplaceAllString(text, "")
	return strings.TrimSpace(text)
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
