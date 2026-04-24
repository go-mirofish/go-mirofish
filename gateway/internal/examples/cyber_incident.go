package examples

import "path/filepath"

type incidentConfig struct {
	ExampleKey        string                     `json:"example_key"`
	Title             string                     `json:"title"`
	AgentCount        int                        `json:"agent_count"`
	RunCommand        string                     `json:"run_command"`
	ExpectedArtifacts []string                   `json:"expected_artifacts"`
	Profiles          map[string]ScenarioProfile `json:"profiles"`
}

type department struct {
	Name             string  `json:"name"`
	ConfusionFactor  float64 `json:"confusion_factor"`
	BroadcastPower   float64 `json:"broadcast_power"`
	InternalDelayMin float64 `json:"internal_delay_min"`
}

type commEdge struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Weight float64 `json:"weight"`
}

func runZeroDayIncident(opts RunOptions) (RunResult, error) {
	root := mustPath(opts.RepoRoot, "examples", "zero-day-incident-drill")
	outputDir := resultOutputRoot(opts, "zero-day-incident-drill")
	var cfg incidentConfig
	if err := readJSONFile(filepath.Join(root, "config.json"), &cfg); err != nil {
		return RunResult{}, err
	}
	profile := defaultProfile(opts.Profile)
	selected := cfg.Profiles[profile]
	if selected.AgentCount == 0 {
		selected.AgentCount = cfg.AgentCount
	}
	var departments []department
	if err := readJSONFile(filepath.Join(root, "departments.json"), &departments); err != nil {
		return RunResult{}, err
	}
	var edges []commEdge
	if err := readJSONFile(filepath.Join(root, "communication_graph.json"), &edges); err != nil {
		return RunResult{}, err
	}

	internalSpread := map[string]float64{}
	externalSpread := map[string]float64{}
	worstDept := ""
	worstScore := -1.0
	for _, dept := range departments {
		internalScore := dept.ConfusionFactor * dept.InternalDelayMin
		externalScore := dept.ConfusionFactor * dept.BroadcastPower
		for _, edge := range edges {
			if edge.From == dept.Name || edge.To == dept.Name {
				internalScore += edge.Weight
				externalScore += edge.Weight * 0.6
			}
		}
		internalSpread[dept.Name] = internalScore
		externalSpread[dept.Name] = externalScore
		if internalScore+externalScore > worstScore {
			worstScore = internalScore + externalScore
			worstDept = dept.Name
		}
	}

	artifactPath := filepath.Join(outputDir, "incident_report.json")
	payload := map[string]any{
		"example":               cfg.Title,
		"agent_count":           cfg.AgentCount,
		"internal_spread":       internalSpread,
		"external_spread":       externalSpread,
		"bottleneck_department": worstDept,
		"confusion_score":       worstScore,
		"recommended_fix":       "Route legal approval and public messaging through a single incident commander channel.",
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
		Artifacts:        map[string]string{"incident_report.json": artifactPath},
		OutputDir:        outputDir,
		Summary:          map[string]any{"bottleneck_department": worstDept},
		LocalOnly:        true,
		CompletedAt:      nowRFC3339(),
	}, nil
}
