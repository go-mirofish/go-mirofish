package examples

import (
	"path/filepath"
	"strings"
)

type literaryConfig struct {
	ExampleKey        string                     `json:"example_key"`
	Title             string                     `json:"title"`
	AgentCount        int                        `json:"agent_count"`
	RunCommand        string                     `json:"run_command"`
	ExpectedArtifacts []string                   `json:"expected_artifacts"`
	Profiles          map[string]ScenarioProfile `json:"profiles"`
}

type characterState struct {
	Name       string   `json:"name"`
	Motivation string   `json:"motivation"`
	Memory     []string `json:"memory"`
}

func runLiteraryEnding(opts RunOptions) (RunResult, error) {
	root := mustPath(opts.RepoRoot, "examples", "lost-ending-literary-simulator")
	outputDir := resultOutputRoot(opts, "lost-ending-literary-simulator")
	var cfg literaryConfig
	if err := readJSONFile(filepath.Join(root, "config.json"), &cfg); err != nil {
		return RunResult{}, err
	}
	profile := defaultProfile(opts.Profile)
	selected := cfg.Profiles[profile]
	if selected.AgentCount == 0 {
		selected.AgentCount = cfg.AgentCount
	}
	var characters []characterState
	if err := readJSONFile(filepath.Join(root, "characters.json"), &characters); err != nil {
		return RunResult{}, err
	}
	chapterState, err := readTextFile(filepath.Join(root, "chapter_state.txt"))
	if err != nil {
		return RunResult{}, err
	}

	lines := []string{"Draft ending:"}
	score := 0.0
	for _, character := range characters {
		line := character.Name + " chooses a course shaped by " + character.Motivation + "."
		if len(character.Memory) > 0 {
			line += " They remember " + character.Memory[0] + "."
		}
		lines = append(lines, line)
		if strings.Contains(strings.ToLower(chapterState), strings.ToLower(character.Motivation)) {
			score += 1
		}
	}
	score = ratio(score, float64(len(characters)))

	textPath := filepath.Join(outputDir, "draft_ending.txt")
	jsonPath := filepath.Join(outputDir, "draft_ending.json")
	body := strings.Join(lines, "\n\n")
	if err := writeTextArtifact(textPath, body); err != nil {
		return RunResult{}, err
	}
	if err := writeJSONArtifact(jsonPath, map[string]any{
		"example":           cfg.Title,
		"agent_count":       cfg.AgentCount,
		"consistency_score": score,
		"draft_ending":      body,
	}); err != nil {
		return RunResult{}, err
	}
	return RunResult{
		ExampleKey:       cfg.ExampleKey,
		Title:            cfg.Title,
		Profile:          profile,
		AgentCount:       selected.AgentCount,
		InteractionCount: selected.InteractionCount,
		TaskCount:        selected.TaskCount,
		Artifacts:        map[string]string{"draft_ending.txt": textPath, "draft_ending.json": jsonPath},
		OutputDir:        outputDir,
		Summary:          map[string]any{"consistency_score": score},
		LocalOnly:        true,
		CompletedAt:      nowRFC3339(),
	}, nil
}
