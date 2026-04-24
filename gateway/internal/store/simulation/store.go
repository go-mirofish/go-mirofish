package simulationstore

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrSimulationNotFound = errors.New("simulation not found")

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
	return readJSON(filepath.Join(s.SimulationDir(simulationID), "state.json"))
}

func (s *Store) ReadRunState(simulationID string) (map[string]any, error) {
	return readJSON(filepath.Join(s.SimulationDir(simulationID), "run_state.json"))
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
	path := filepath.Join(s.SimulationDir(simulationID), "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
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
		items, err := s.interviewHistoryFromDB(s.DBPath(simulationID, p), p, agentID, limit)
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
	pyCode := `
import json
import sqlite3
import sys

db_path = sys.argv[1]
platform = sys.argv[2]
limit = int(sys.argv[3])
agent_arg = sys.argv[4]

query = """
    SELECT user_id, info, created_at
    FROM trace
    WHERE action = 'interview'
"""
params = []
if agent_arg:
    query += " AND user_id = ?"
    params.append(int(agent_arg))
query += " ORDER BY created_at DESC LIMIT ?"
params.append(limit)

try:
    conn = sqlite3.connect(db_path)
    cur = conn.cursor()
    cur.execute(query, params)
    rows = []
    for user_id, info_json, created_at in cur.fetchall():
        try:
            info = json.loads(info_json) if info_json else {}
        except json.JSONDecodeError:
            info = {"raw": info_json}
        rows.append({
            "agent_id": user_id,
            "response": info.get("response"),
            "prompt": info.get("prompt", ""),
            "timestamp": created_at,
            "platform": platform,
        })
    conn.close()
    print(json.dumps(rows))
except Exception:
    print("[]")
`

	agentArg := ""
	if agentID != nil {
		agentArg = strconv.Itoa(*agentID)
	}

	cmd := exec.Command("python3", "-c", pyCode, dbPath, platform, strconv.Itoa(limit), agentArg)
	raw, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
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

func toString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
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
