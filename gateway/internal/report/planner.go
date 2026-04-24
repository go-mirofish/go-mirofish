package report

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-mirofish/go-mirofish/gateway/internal/provider"
)

type SectionPlan struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Outline struct {
	Title    string        `json:"title"`
	Summary  string        `json:"summary"`
	Sections []SectionPlan `json:"sections"`
}

type Planner struct {
	executor provider.Executor
	model    string
}

func NewPlanner(executor provider.Executor, model string) *Planner {
	return &Planner{executor: executor, model: model}
}

func (p *Planner) Plan(ctx context.Context, simulationRequirement string, facts []string) (Outline, error) {
	if p.executor == nil {
		return deterministicOutline(simulationRequirement), nil
	}
	req := provider.ProviderRequest{
		Model: p.model,
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: "Return compact JSON with title, summary, and 3 sections."},
			{Role: provider.RoleUser, Content: "Scenario:\n" + simulationRequirement + "\n\nFacts:\n" + strings.Join(facts, "\n")},
		},
		Temperature:    0,
		MaxTokens:      4096,
		ResponseFormat: &provider.ResponseFormat{Type: provider.ResponseFormatJSONObject},
	}
	resp, err := p.executor.Execute(ctx, req)
	if err != nil {
		return deterministicOutline(simulationRequirement), nil
	}
	var outline Outline
	if err := json.Unmarshal([]byte(resp.Content), &outline); err != nil || len(outline.Sections) == 0 {
		return deterministicOutline(simulationRequirement), nil
	}
	return outline, nil
}

func deterministicOutline(requirement string) Outline {
	return Outline{
		Title:   "Future prediction report",
		Summary: "A deterministic report assembled from graph and simulation evidence.",
		Sections: []SectionPlan{
			{Title: "Projected future state", Description: requirement},
			{Title: "Agent and group reactions", Description: "How different groups reacted in the simulation"},
			{Title: "Risks and opportunities", Description: "Key risks and opportunities surfaced by the run"},
		},
	}
}
