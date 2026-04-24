package examples

import "path/filepath"

type defiConfig struct {
	ExampleKey        string                     `json:"example_key"`
	Title             string                     `json:"title"`
	AgentCount        int                        `json:"agent_count"`
	RunCommand        string                     `json:"run_command"`
	ExpectedArtifacts []string                   `json:"expected_artifacts"`
	Profiles          map[string]ScenarioProfile `json:"profiles"`
}

type marketSeed struct {
	InitialPrice        float64 `json:"initial_price"`
	PanicSentimentLevel float64 `json:"panic_sentiment_level"`
}

type archetype struct {
	Name               string  `json:"name"`
	Count              int     `json:"count"`
	CascadeSensitivity float64 `json:"cascade_sensitivity"`
	PressureMultiplier float64 `json:"pressure_multiplier"`
}

func runDefiStress(opts RunOptions) (RunResult, error) {
	root := mustPath(opts.RepoRoot, "examples", "defi-sentiment-stress-test")
	outputDir := resultOutputRoot(opts, "defi-sentiment-stress-test")
	var cfg defiConfig
	if err := readJSONFile(filepath.Join(root, "config.json"), &cfg); err != nil {
		return RunResult{}, err
	}
	profile := defaultProfile(opts.Profile)
	selected := cfg.Profiles[profile]
	if selected.AgentCount == 0 {
		selected.AgentCount = cfg.AgentCount
	}
	var seed marketSeed
	if err := readJSONFile(filepath.Join(root, "market_seed.json"), &seed); err != nil {
		return RunResult{}, err
	}
	var archetypes []archetype
	if err := readJSONFile(filepath.Join(root, "risk_archetypes.json"), &archetypes); err != nil {
		return RunResult{}, err
	}

	threshold := seed.InitialPrice
	totalPressure := 0.0
	for _, a := range archetypes {
		totalPressure += float64(a.Count) * a.CascadeSensitivity * a.PressureMultiplier
	}
	cascadePrice := seed.InitialPrice * (1 - clampFloat(totalPressure/1000, 0.05, 0.45))

	artifactPath := filepath.Join(outputDir, "liquidation_cascade_forecast.json")
	payload := map[string]any{
		"example":                   cfg.Title,
		"agent_count":               cfg.AgentCount,
		"initial_price":             seed.InitialPrice,
		"panic_sentiment_level":     seed.PanicSentimentLevel,
		"panic_sell_threshold":      threshold * (1 - seed.PanicSentimentLevel),
		"cascade_price_region_low":  cascadePrice * 0.97,
		"cascade_price_region_high": cascadePrice * 1.03,
		"total_pressure_score":      totalPressure,
	}
	if err := writeJSONArtifact(artifactPath, payload); err != nil {
		return RunResult{}, err
	}
	return RunResult{
		ExampleKey:       cfg.ExampleKey,
		Title:            cfg.Title,
		Profile:          profile,
		AgentCount:       selected.AgentCount,
		InteractionCount: selected.InteractionCount,
		TaskCount:        selected.TaskCount,
		Artifacts:        map[string]string{"liquidation_cascade_forecast.json": artifactPath},
		OutputDir:        outputDir,
		Summary:          map[string]any{"cascade_price_region": []float64{cascadePrice * 0.97, cascadePrice * 1.03}},
		LocalOnly:        true,
		CompletedAt:      nowRFC3339(),
	}, nil
}
