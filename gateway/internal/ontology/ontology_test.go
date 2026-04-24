package ontology

import (
	"context"
	"testing"
)

type stubGenerator struct {
	response string
	err      error
}

func (s stubGenerator) Execute(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	return s.response, s.err
}

func TestOntologyBuilderBuild(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
	}{
		{
			name:    "normalizes ontology",
			payload: `{"entity_types":[{"name":"city official","description":"A city official involved in the event","attributes":[{"name":"created at","description":"created at"}],"examples":["a"]},{"name":"Person","description":"person","attributes":[{"name":"full_name","description":"name"}]},{"name":"Organization","description":"org","attributes":[{"name":"org_name","description":"name"}]}],"edge_types":[{"name":"works for","description":"relation","source_targets":[{"source":"city official","target":"Organization"},{"source":"city official","target":"Organization"}]}],"analysis_summary":"summary"}`,
		},
		{
			name:    "rejects invalid json",
			payload: `{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(stubGenerator{response: tt.payload})
			_, err := builder.Build(context.Background(), BuildInput{
				SimulationRequirement: "req",
				SourceText:            "text",
			})
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigResolverResolve(t *testing.T) {
	resolver := NewConfigResolver(nil)
	cfg, err := resolver.Resolve(context.Background(), "sim", "proj", "graph", "req", "model", "base", []Entity{{UUID: "1", Name: "Alice", Labels: []string{"Person"}, Summary: "summary"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SimulationID != "sim" || len(cfg.AgentConfigs) != 1 {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestProfileGeneratorGenerate(t *testing.T) {
	gen := NewProfileGenerator(nil, "")
	profiles, err := gen.Generate(context.Background(), []Entity{{UUID: "1", Name: "Alice Doe", Labels: []string{"Person"}, Summary: "summary"}}, "reddit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 1 || profiles[0].Username == "" {
		t.Fatalf("unexpected profiles: %#v", profiles)
	}
}
