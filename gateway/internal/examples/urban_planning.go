package examples

import "path/filepath"

type planningConfig struct {
	ExampleKey        string                     `json:"example_key"`
	Title             string                     `json:"title"`
	AgentCount        int                        `json:"agent_count"`
	RunCommand        string                     `json:"run_command"`
	ExpectedArtifacts []string                   `json:"expected_artifacts"`
	Profiles          map[string]ScenarioProfile `json:"profiles"`
}

type demographicGroup struct {
	Name               string             `json:"name"`
	PopulationWeight   float64            `json:"population_weight"`
	ScenarioAffinities map[string]float64 `json:"scenario_affinities"`
}

type planningScenario struct {
	Key         string             `json:"key"`
	Title       string             `json:"title"`
	Adjustments map[string]float64 `json:"adjustments"`
}

func runUrbanPlanning(opts RunOptions) (RunResult, error) {
	root := mustPath(opts.RepoRoot, "examples", "hyperlocal-urban-planning")
	outputDir := resultOutputRoot(opts, "hyperlocal-urban-planning")
	var cfg planningConfig
	if err := readJSONFile(filepath.Join(root, "config.json"), &cfg); err != nil {
		return RunResult{}, err
	}
	profile := defaultProfile(opts.Profile)
	selected := cfg.Profiles[profile]
	if selected.AgentCount == 0 {
		selected.AgentCount = cfg.AgentCount
	}
	var groups []demographicGroup
	if err := readJSONFile(filepath.Join(root, "demographics.json"), &groups); err != nil {
		return RunResult{}, err
	}
	var highway planningScenario
	if err := readJSONFile(filepath.Join(root, "scenario_highway.json"), &highway); err != nil {
		return RunResult{}, err
	}
	var park planningScenario
	if err := readJSONFile(filepath.Join(root, "scenario_park.json"), &park); err != nil {
		return RunResult{}, err
	}

	artifacts := map[string]string{}
	for _, scenario := range []planningScenario{highway, park} {
		var support []string
		var block []string
		scoreboard := map[string]float64{}
		for _, group := range groups {
			score := group.ScenarioAffinities[scenario.Key] + scenario.Adjustments[group.Name]
			score *= group.PopulationWeight
			scoreboard[group.Name] = score
			if score >= 0 {
				support = append(support, group.Name)
			} else {
				block = append(block, group.Name)
			}
		}
		artifactPath := filepath.Join(outputDir, "coalition_"+scenario.Key+".json")
		payload := map[string]any{
			"scenario":          scenario.Title,
			"support_coalition": support,
			"block_coalition":   block,
			"scores":            scoreboard,
			"unexpected_alignment": map[string]any{
				"groups": firstUnexpectedPair(support, block),
			},
		}
		if err := writeJSONArtifact(artifactPath, payload); err != nil {
			return RunResult{}, err
		}
		artifacts[filepath.Base(artifactPath)] = artifactPath
	}
	return RunResult{
		ExampleKey:       cfg.ExampleKey,
		Title:            cfg.Title,
		Profile:          profile,
		AgentCount:       selected.AgentCount,
		InteractionCount: selected.InteractionCount,
		TaskCount:        selected.TaskCount,
		Artifacts:        artifacts,
		OutputDir:        outputDir,
		Summary:          map[string]any{"variants": []string{"highway", "park"}},
		LocalOnly:        true,
		CompletedAt:      nowRFC3339(),
	}, nil
}

func firstUnexpectedPair(support, block []string) []string {
	if len(support) > 0 && len(block) > 0 {
		return []string{support[0], block[0]}
	}
	return []string{}
}
