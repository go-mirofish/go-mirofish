package simulationstore

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/internal/artifactcontract"

	_ "modernc.org/sqlite" // public interview history reads (SQLite) without a Python subprocess
)

type Store struct {
	SimulationsDir string
	ScriptsDir     string
	ProjectsDir    string
	ReportsDir     string
}

func New(simulationsDir, scriptsDir, projectsDir, reportsDir string) *Store {
	return &Store{
		SimulationsDir: simulationsDir,
		ScriptsDir:     scriptsDir,
		ProjectsDir:    projectsDir,
		ReportsDir:     reportsDir,
	}
}

func (s *Store) SimulationDir(simulationID string) string {
	return filepath.Join(s.SimulationsDir, simulationID)
}

func (s *Store) ReadState(simulationID string) (map[string]any, error) {
	controlPath := filepath.Join(s.SimulationDir(simulationID), "control_state.json")
	payload, err := readJSON(controlPath)
	if err == nil {
		return payload, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return readJSON(filepath.Join(s.SimulationDir(simulationID), "state.json"))
}

// ReadRuntimeState loads state.json written by the simulation worker (Go-native or legacy Python).
// It complements run_state.json, which not all worker entrypoints emit on every platform.
func (s *Store) ReadRuntimeState(simulationID string) (map[string]any, error) {
	path := filepath.Join(s.SimulationDir(simulationID), "state.json")
	payload, err := readJSON(path)
	if err == nil {
		return payload, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	runState, err := s.ReadRunState(simulationID)
	if err != nil {
		return nil, err
	}
	return runtimeStateFromRunState(runState), nil
}

func (s *Store) ReadRunState(simulationID string) (map[string]any, error) {
	raw, err := os.ReadFile(filepath.Join(s.SimulationDir(simulationID), "run_state.json"))
	if err != nil {
		return nil, err
	}
	state, err := artifactcontract.ReadRunStateJSON(raw)
	if err != nil {
		return nil, err
	}
	normalizedRaw, err := artifactcontract.WriteRunStateJSON(state)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(normalizedRaw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Store) ReadConfig(simulationID string) (map[string]any, error) {
	return readJSON(filepath.Join(s.SimulationDir(simulationID), "simulation_config.json"))
}

func (s *Store) ReadConfigWithMeta(simulationID string) (map[string]any, bool, string, error) {
	configPath := filepath.Join(s.SimulationDir(simulationID), "simulation_config.json")
	info, err := os.Stat(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, "", nil
		}
		return nil, false, "", err
	}
	payload, err := readJSON(configPath)
	if err != nil {
		return nil, true, info.ModTime().Format(time.RFC3339), err
	}
	return payload, true, info.ModTime().Format(time.RFC3339), nil
}

func (s *Store) ReadProfileArtifacts(simulationID, platform string) ([]map[string]any, error) {
	switch platform {
	case "twitter":
		return readCSV(filepath.Join(s.SimulationDir(simulationID), "twitter_profiles.csv"))
	default:
		return readJSONArray(filepath.Join(s.SimulationDir(simulationID), "reddit_profiles.json"))
	}
}

func (s *Store) ReadProfileArtifactsWithMeta(simulationID, platform string) ([]map[string]any, bool, string, error) {
	var profilePath string
	switch platform {
	case "twitter":
		profilePath = filepath.Join(s.SimulationDir(simulationID), "twitter_profiles.csv")
	default:
		profilePath = filepath.Join(s.SimulationDir(simulationID), "reddit_profiles.json")
	}
	info, err := os.Stat(profilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []map[string]any{}, false, "", nil
		}
		return nil, false, "", err
	}
	payload, err := s.ReadProfileArtifacts(simulationID, platform)
	if err != nil {
		return nil, true, info.ModTime().Format(time.RFC3339), err
	}
	return payload, true, info.ModTime().Format(time.RFC3339), nil
}

func (s *Store) ReadActionLogs(simulationID string, platform string) ([]map[string]any, error) {
	var paths []string
	switch platform {
	case "twitter":
		paths = []string{filepath.Join(s.SimulationDir(simulationID), "twitter", "actions.jsonl")}
	case "reddit":
		paths = []string{filepath.Join(s.SimulationDir(simulationID), "reddit", "actions.jsonl")}
	default:
		paths = []string{
			filepath.Join(s.SimulationDir(simulationID), "twitter", "actions.jsonl"),
			filepath.Join(s.SimulationDir(simulationID), "reddit", "actions.jsonl"),
		}
	}

	var out []map[string]any
	for _, path := range paths {
		items, err := readJSONL(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		out = append(out, items...)
	}
	sort.Slice(out, func(i, j int) bool {
		return toString(out[i]["timestamp"]) > toString(out[j]["timestamp"])
	})
	return out, nil
}

func (s *Store) ScriptPath(scriptName string) (string, error) {
	allowed := map[string]bool{
		"run_twitter_simulation.py":  true,
		"run_reddit_simulation.py":   true,
		"run_parallel_simulation.py": true,
		"action_logger.py":           true,
	}
	if !allowed[scriptName] {
		return "", os.ErrNotExist
	}
	path := filepath.Join(s.ScriptsDir, scriptName)
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) ReadProject(projectID string) (map[string]any, error) {
	return readJSON(filepath.Join(s.ProjectsDir, projectID, "project.json"))
}

func (s *Store) WriteState(simulationID string, payload map[string]any) error {
	path := filepath.Join(s.SimulationDir(simulationID), "control_state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (s *Store) DeleteSimulation(simulationID string) error {
	return os.RemoveAll(s.SimulationDir(simulationID))
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
		sim, err := s.ReadState(entry.Name())
		if err != nil {
			continue
		}
		if projectID != "" && toString(sim["project_id"]) != projectID {
			continue
		}
		out = append(out, sim)
	}
	sort.Slice(out, func(i, j int) bool {
		return toString(out[i]["created_at"]) > toString(out[j]["created_at"])
	})
	return out, nil
}

func (s *Store) InterviewHistory(simulationID, platform string, agentID *int, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 100
	}
	platforms := []string{"twitter", "reddit"}
	if platform == "twitter" || platform == "reddit" {
		platforms = []string{platform}
	}
	var results []map[string]any
	for _, p := range platforms {
		items, err := s.interviewHistoryFromJSONL(simulationID, p, agentID, limit)
		if err == nil && len(items) > 0 {
			results = append(results, items...)
			continue
		}
		items, err = s.interviewHistoryFromDB(s.DBPath(simulationID, p), p, agentID, limit)
		if err != nil {
			return nil, err
		}
		results = append(results, items...)
	}
	sort.Slice(results, func(i, j int) bool {
		return toString(results[i]["timestamp"]) > toString(results[j]["timestamp"])
	})
	if len(platforms) > 1 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (s *Store) interviewHistoryFromJSONL(simulationID, platform string, agentID *int, limit int) ([]map[string]any, error) {
	path := filepath.Join(s.SimulationDir(simulationID), platform+"_interviews.jsonl")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var out []map[string]any
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			return nil, err
		}
		if agentID != nil && intValueAny(payload["agent_id"]) != *agentID {
			continue
		}
		out = append(out, payload)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return toString(out[i]["timestamp"]) > toString(out[j]["timestamp"]) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Store) FindLatestReportBySimulation(simulationID string) (map[string]any, error) {
	if strings.TrimSpace(s.ReportsDir) == "" {
		return nil, nil
	}
	if err := os.MkdirAll(s.ReportsDir, 0o755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.ReportsDir)
	if err != nil {
		return nil, err
	}

	var latest map[string]any
	for _, entry := range entries {
		report, err := s.readReportMeta(entry)
		if err != nil {
			continue
		}
		if toString(report["simulation_id"]) != simulationID {
			continue
		}
		if latest == nil || toString(report["created_at"]) > toString(latest["created_at"]) {
			latest = report
		}
	}

	return latest, nil
}

func (s *Store) interviewHistoryFromDB(dbPath, platform string, agentID *int, limit int) ([]map[string]any, error) {
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(abs) + "?mode=ro"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return []map[string]any{}, nil
	}
	defer db.Close()

	var rows *sql.Rows
	if agentID != nil {
		rows, err = db.Query(`
			SELECT user_id, info, created_at
			FROM trace
			WHERE action = 'interview' AND user_id = ?
			ORDER BY created_at DESC LIMIT ?`, *agentID, limit)
	} else {
		rows, err = db.Query(`
			SELECT user_id, info, created_at
			FROM trace
			WHERE action = 'interview'
			ORDER BY created_at DESC LIMIT ?`, limit)
	}
	if err != nil {
		return []map[string]any{}, nil
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var userID int
		var infoJSON sql.NullString
		var createdAt string
		if err := rows.Scan(&userID, &infoJSON, &createdAt); err != nil {
			continue
		}
		var info map[string]any
		if infoJSON.Valid && strings.TrimSpace(infoJSON.String) != "" {
			if err := json.Unmarshal([]byte(infoJSON.String), &info); err != nil {
				info = map[string]any{"raw": infoJSON.String}
			}
		} else {
			info = map[string]any{}
		}
		out = append(out, map[string]any{
			"agent_id":   userID,
			"response":   info["response"],
			"prompt":     toString(info["prompt"]),
			"timestamp":  createdAt,
			"platform":   platform,
		})
	}
	if err := rows.Err(); err != nil {
		return []map[string]any{}, nil
	}
	return out, nil
}

func ConfigPath(simDir string) string {
	return filepath.Join(simDir, "simulation_config.json")
}

func (s *Store) DBPath(simulationID, platform string) string {
	return filepath.Join(s.SimulationDir(simulationID), platform+"_simulation.db")
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

func readJSONArray(path string) ([]map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload []map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func readCSV(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []map[string]any{}, nil
	}
	headers := rows[0]
	var out []map[string]any
	for _, row := range rows[1:] {
		record := map[string]any{}
		for i, header := range headers {
			if i < len(row) {
				record[header] = row[i]
			}
		}
		out = append(out, record)
	}
	return out, nil
}

func readJSONL(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	events, err := artifactcontract.ParseActionsJSONL(file)
	if err != nil {
		_ = file.Close()
		file, openErr := os.Open(path)
		if openErr != nil {
			return nil, openErr
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		var out []map[string]any
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var payload map[string]any
			if err := json.Unmarshal([]byte(line), &payload); err != nil {
				continue
			}
			out = append(out, payload)
		}
		return out, scanner.Err()
	}
	formatted, err := artifactcontract.FormatActionsJSONL(events)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(formatted)))
	var out []map[string]any
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			return nil, err
		}
		out = append(out, payload)
	}
	return out, scanner.Err()
}

func toString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func runtimeStateFromRunState(runState map[string]any) map[string]any {
	payload := map[string]any{
		"simulation_id":           runState["simulation_id"],
		"worker_protocol_version": runState["worker_protocol_version"],
		"runner_status":           runState["runner_status"],
		"status":                  runtimeStatusFromRunnerStatus(toString(runState["runner_status"])),
		"current_round":           runState["current_round"],
		"total_rounds":            runState["total_rounds"],
		"simulated_hours":         runState["simulated_hours"],
		"total_simulation_hours":  runState["total_simulation_hours"],
		"progress_percent":        runState["progress_percent"],
		"twitter_actions_count":   runState["twitter_actions_count"],
		"reddit_actions_count":    runState["reddit_actions_count"],
		"total_actions_count":     runState["total_actions_count"],
		"started_at":              runState["started_at"],
		"updated_at":              runState["updated_at"],
		"completed_at":            runState["completed_at"],
		"error":                   runState["error"],
	}
	for _, key := range []string{
		"twitter_current_round",
		"reddit_current_round",
		"twitter_simulated_hours",
		"reddit_simulated_hours",
		"twitter_running",
		"reddit_running",
		"twitter_completed",
		"reddit_completed",
	} {
		if value, ok := runState[key]; ok {
			payload[key] = value
		}
	}
	return payload
}

func runtimeStatusFromRunnerStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "completed", "failed", "stopped":
		return strings.TrimSpace(status)
	case "running", "starting", "paused", "stopping":
		return "running"
	default:
		return "idle"
	}
}

func intValueAny(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return int(parsed)
		}
	}
	return 0
}

func (s *Store) readReportMeta(entry os.DirEntry) (map[string]any, error) {
	var path string
	switch {
	case entry.IsDir():
		path = filepath.Join(s.ReportsDir, entry.Name(), "meta.json")
	case strings.HasSuffix(entry.Name(), ".json"):
		path = filepath.Join(s.ReportsDir, entry.Name())
	default:
		return nil, os.ErrNotExist
	}
	return readJSON(path)
}
