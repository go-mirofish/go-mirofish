package examples

import (
	"path/filepath"
	"sort"
	"strings"
)

type prWarRoomConfig struct {
	ExampleKey          string                     `json:"example_key"`
	Title               string                     `json:"title"`
	AgentCount          int                        `json:"agent_count"`
	DurationHours       int                        `json:"duration_hours"`
	RunCommand          string                     `json:"run_command"`
	ExpectedArtifacts   []string                   `json:"expected_artifacts"`
	Profiles            map[string]ScenarioProfile `json:"profiles"`
	PersonaWeights      map[string]float64         `json:"persona_weights"`
	SensitivityKeywords map[string]float64         `json:"sensitivity_keywords"`
	RewriteHints        []string                   `json:"rewrite_hints"`
}

type personaSeed struct {
	Group     string  `json:"group"`
	Count     int     `json:"count"`
	BaseScore float64 `json:"base_score"`
}

func runProductLaunchWarRoom(opts RunOptions) (RunResult, error) {
	root := mustPath(opts.RepoRoot, "examples", "product-launch-war-room")
	outputDir := resultOutputRoot(opts, "product-launch-war-room")

	var cfg prWarRoomConfig
	if err := readJSONFile(filepath.Join(root, "config.json"), &cfg); err != nil {
		return RunResult{}, err
	}
	profile := defaultProfile(opts.Profile)
	selected := cfg.Profiles[profile]
	if selected.AgentCount == 0 {
		selected.AgentCount = cfg.AgentCount
	}
	var personas []personaSeed
	if err := readJSONFile(filepath.Join(root, "personas.json"), &personas); err != nil {
		return RunResult{}, err
	}
	pressRelease, err := readTextFile(filepath.Join(root, "press_release.txt"))
	if err != nil {
		return RunResult{}, err
	}

	sentences := sentenceSplit(pressRelease)
	type sentenceScore struct {
		Sentence string             `json:"sentence"`
		Score    float64            `json:"score"`
		ByGroup  map[string]float64 `json:"by_group"`
	}
	scored := make([]sentenceScore, 0, len(sentences))
	var top sentenceScore
	for _, sentence := range sentences {
		entry := sentenceScore{Sentence: sentence, ByGroup: map[string]float64{}}
		for _, persona := range personas {
			groupWeight := cfg.PersonaWeights[persona.Group]
			if groupWeight == 0 {
				groupWeight = 1
			}
			groupScore := persona.BaseScore * groupWeight
			lowered := strings.ToLower(sentence)
			for keyword, weight := range cfg.SensitivityKeywords {
				if strings.Contains(lowered, strings.ToLower(keyword)) {
					groupScore += weight
				}
			}
			groupScore *= float64(persona.Count)
			entry.ByGroup[persona.Group] = groupScore
			entry.Score += groupScore
		}
		if entry.Score > top.Score {
			top = entry
		}
		scored = append(scored, entry)
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })

	safeRewrite := top.Sentence
	for _, hint := range cfg.RewriteHints {
		safeRewrite = strings.ReplaceAll(safeRewrite, hint, "")
	}
	safeRewrite = strings.TrimSpace(strings.Join(strings.Fields(safeRewrite), " "))
	if safeRewrite == "" {
		safeRewrite = "We will roll out the feature gradually with stronger privacy controls and clearer user choice."
	}

	artifactPath := filepath.Join(outputDir, "risk_report.json")
	payload := map[string]any{
		"example":                   cfg.Title,
		"duration_hours":            cfg.DurationHours,
		"agent_count":               cfg.AgentCount,
		"most_triggering_sentence":  top.Sentence,
		"most_triggering_score":     top.Score,
		"group_breakdown":           top.ByGroup,
		"ranked_sentence_scores":    scored,
		"recommended_safer_rewrite": safeRewrite,
		"rewrite_notes":             []string{"Reduce absolutist claims", "Lead with user control", "Avoid surveillance-adjacent phrasing"},
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
		Artifacts:        map[string]string{"risk_report.json": artifactPath},
		OutputDir:        outputDir,
		Summary:          map[string]any{"top_sentence": top.Sentence, "top_score": top.Score},
		LocalOnly:        true,
		CompletedAt:      nowRFC3339(),
	}, nil
}
