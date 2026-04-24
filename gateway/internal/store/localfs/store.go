package localfs

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	reportstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/report"
)

type Store struct {
	ProjectsDir    string
	ReportsDir     string
	TasksDir       string
	SimulationsDir string
	ScriptsDir     string
}

func New(projectsDir, reportsDir, tasksDir, simulationsDir, scriptsDir string) *Store {
	return &Store{
		ProjectsDir:    projectsDir,
		ReportsDir:     reportsDir,
		TasksDir:       tasksDir,
		SimulationsDir: simulationsDir,
		ScriptsDir:     scriptsDir,
	}
}

func (s *Store) ProjectDir(projectID string) string { return filepath.Join(s.ProjectsDir, projectID) }
func (s *Store) ProjectMetaPath(projectID string) string {
	return filepath.Join(s.ProjectDir(projectID), "project.json")
}
func (s *Store) TaskPath(taskID string) string { return filepath.Join(s.TasksDir, taskID+".json") }
func (s *Store) SimulationDir(simulationID string) string {
	return filepath.Join(s.SimulationsDir, simulationID)
}
func (s *Store) SimulationStatePath(simulationID string) string {
	return filepath.Join(s.SimulationDir(simulationID), "state.json")
}
func (s *Store) SimulationRunStatePath(simulationID string) string {
	return filepath.Join(s.SimulationDir(simulationID), "run_state.json")
}
func (s *Store) SimulationConfigPath(simulationID string) string {
	return filepath.Join(s.SimulationDir(simulationID), "simulation_config.json")
}

func (s *Store) ReadProject(projectID string) (map[string]any, error) {
	return readJSON(s.ProjectMetaPath(projectID))
}

func (s *Store) WriteProject(projectID string, payload map[string]any) error {
	path := s.ProjectMetaPath(projectID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func (s *Store) ReadTask(taskID string) (map[string]any, error) {
	return readJSON(s.TaskPath(taskID))
}

func (s *Store) WriteTask(taskID string, payload map[string]any) error {
	if err := os.MkdirAll(s.TasksDir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.TaskPath(taskID), raw, 0o644)
}

func (s *Store) ReadSimulation(simulationID string) (map[string]any, error) {
	return readJSON(s.SimulationStatePath(simulationID))
}

func (s *Store) WriteSimulation(simulationID string, payload map[string]any) error {
	if err := os.MkdirAll(s.SimulationDir(simulationID), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.SimulationStatePath(simulationID), raw, 0o644)
}

func (s *Store) ReadSimulationProfiles(simulationID, platform string) ([]any, bool, any, error) {
	simDir := s.SimulationDir(simulationID)
	if _, err := os.Stat(simDir); err != nil {
		return nil, false, nil, err
	}
	var profilePath string
	if platform == "twitter" {
		profilePath = filepath.Join(simDir, "twitter_profiles.csv")
	} else {
		profilePath = filepath.Join(simDir, "reddit_profiles.json")
	}
	info, err := os.Stat(profilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []any{}, false, nil, nil
		}
		return nil, false, nil, err
	}
	modifiedAt := info.ModTime().Format(time.RFC3339)
	if platform == "twitter" {
		file, err := os.Open(profilePath)
		if err != nil {
			return nil, true, modifiedAt, err
		}
		defer file.Close()
		reader := csv.NewReader(file)
		rows, err := reader.ReadAll()
		if err != nil || len(rows) == 0 {
			return []any{}, true, modifiedAt, nil
		}
		headers := rows[0]
		profiles := make([]any, 0, len(rows)-1)
		for _, row := range rows[1:] {
			record := map[string]any{}
			for i, header := range headers {
				if i < len(row) {
					record[header] = row[i]
				}
			}
			profiles = append(profiles, record)
		}
		return profiles, true, modifiedAt, nil
	}
	raw, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, true, modifiedAt, err
	}
	var payload []any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return []any{}, true, modifiedAt, nil
	}
	return payload, true, modifiedAt, nil
}

func (s *Store) ReadSimulationConfigWithMeta(simulationID string) (map[string]any, bool, any, error) {
	simDir := s.SimulationDir(simulationID)
	if _, err := os.Stat(simDir); err != nil {
		return nil, false, nil, err
	}
	configPath := s.SimulationConfigPath(simulationID)
	info, err := os.Stat(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil, nil
		}
		return nil, false, nil, err
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, true, info.ModTime().Format(time.RFC3339), err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, true, info.ModTime().Format(time.RFC3339), nil
	}
	return payload, true, info.ModTime().Format(time.RFC3339), nil
}

func (s *Store) ReadSimulationRunState(simulationID string) (map[string]any, error) {
	return readJSON(s.SimulationRunStatePath(simulationID))
}

func (s *Store) ListSimulations(projectID string) ([]map[string]any, error) {
	if err := os.MkdirAll(s.SimulationsDir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.SimulationsDir)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		simulation, err := s.ReadSimulation(entry.Name())
		if err != nil {
			continue
		}
		if projectID != "" {
			if got, _ := simulation["project_id"].(string); got != projectID {
				continue
			}
		}
		out = append(out, simulation)
	}
	sort.Slice(out, func(i, j int) bool {
		ic, _ := out[i]["created_at"].(string)
		jc, _ := out[j]["created_at"].(string)
		return ic > jc
	})
	return out, nil
}

func (s *Store) BuildRunInstructions(simulationID string) map[string]any {
	simulationDir := s.SimulationDir(simulationID)
	configFile := s.SimulationConfigPath(simulationID)
	scriptsDir := filepath.Clean(s.ScriptsDir)
	if abs, err := filepath.Abs(simulationDir); err == nil {
		simulationDir = abs
	}
	if abs, err := filepath.Abs(configFile); err == nil {
		configFile = abs
	}
	if abs, err := filepath.Abs(scriptsDir); err == nil {
		scriptsDir = abs
	}
	twitterCmd := "python " + filepath.ToSlash(filepath.Join(scriptsDir, "run_twitter_simulation.py")) + " --config " + configFile
	redditCmd := "python " + filepath.ToSlash(filepath.Join(scriptsDir, "run_reddit_simulation.py")) + " --config " + configFile
	parallelCmd := "python " + filepath.ToSlash(filepath.Join(scriptsDir, "run_parallel_simulation.py")) + " --config " + configFile
	return map[string]any{
		"simulation_dir": simulationDir,
		"scripts_dir":    scriptsDir,
		"config_file":    configFile,
		"commands": map[string]any{
			"twitter":  twitterCmd,
			"reddit":   redditCmd,
			"parallel": parallelCmd,
		},
		"instructions": "1. Activate the conda environment: conda activate go-mirofish\n" +
			"2. Run the simulation (scripts live in " + scriptsDir + "):\n" +
			"   - Run Twitter only: " + twitterCmd + "\n" +
			"   - Run Reddit only: " + redditCmd + "\n" +
			"   - Run both platforms in parallel: " + parallelCmd,
	}
}

func (s *Store) ReadReport(reportID string) (map[string]any, error) {
	meta, err := reportstore.NewFileStore(s.ReportsDir).LoadMeta(reportID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"report_id":              meta.ReportID,
		"simulation_id":          meta.SimulationID,
		"graph_id":               meta.GraphID,
		"simulation_requirement": meta.SimulationRequirement,
		"status":                 meta.Status,
		"outline":                meta.Outline,
		"markdown_content":       meta.MarkdownContent,
		"created_at":             meta.CreatedAt,
		"completed_at":           meta.CompletedAt,
		"error":                  meta.Error,
	}, nil
}

func (s *Store) FindReportBySimulation(simulationID string) (map[string]any, error) {
	meta, found, err := reportstore.NewFileStore(s.ReportsDir).FindBySimulation(simulationID)
	if err != nil || !found {
		return nil, err
	}
	return map[string]any{
		"report_id":              meta.ReportID,
		"simulation_id":          meta.SimulationID,
		"graph_id":               meta.GraphID,
		"simulation_requirement": meta.SimulationRequirement,
		"status":                 meta.Status,
		"outline":                meta.Outline,
		"markdown_content":       meta.MarkdownContent,
		"created_at":             meta.CreatedAt,
		"completed_at":           meta.CompletedAt,
		"error":                  meta.Error,
	}, nil
}

func (s *Store) ListReports(simulationID string, limit int) ([]map[string]any, error) {
	items, err := reportstore.NewFileStore(s.ReportsDir).ListReports(simulationID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(items))
	for _, meta := range items {
		out = append(out, map[string]any{
			"report_id":              meta.ReportID,
			"simulation_id":          meta.SimulationID,
			"graph_id":               meta.GraphID,
			"simulation_requirement": meta.SimulationRequirement,
			"status":                 meta.Status,
			"outline":                meta.Outline,
			"markdown_content":       meta.MarkdownContent,
			"created_at":             meta.CreatedAt,
			"completed_at":           meta.CompletedAt,
			"error":                  meta.Error,
		})
	}
	return out, nil
}

func (s *Store) ReadReportProgress(reportID string) (map[string]any, error) {
	progress, err := reportstore.NewFileStore(s.ReportsDir).LoadProgress(reportID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"report_id":     progress.ReportID,
		"simulation_id": progress.SimulationID,
		"status":        progress.Status,
		"progress":      progress.Progress,
		"message":       progress.Message,
		"updated_at":    progress.UpdatedAt,
	}, nil
}

func (s *Store) ReadReportSections(reportID string) ([]map[string]any, error) {
	return reportstore.NewFileStore(s.ReportsDir).LoadSections(reportID)
}

func readJSON(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func TwoDigits(v int) string {
	if v < 10 {
		return "0" + strconv.Itoa(v)
	}
	return strconv.Itoa(v)
}
