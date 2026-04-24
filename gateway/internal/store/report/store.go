package reportstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ReportMeta struct {
	ReportID              string         `json:"report_id"`
	SimulationID          string         `json:"simulation_id"`
	GraphID               string         `json:"graph_id"`
	SimulationRequirement string         `json:"simulation_requirement"`
	Status                string         `json:"status"`
	Outline               map[string]any `json:"outline,omitempty"`
	MarkdownContent       string         `json:"markdown_content"`
	CreatedAt             string         `json:"created_at"`
	CompletedAt           string         `json:"completed_at,omitempty"`
	Error                 string         `json:"error,omitempty"`
}

type Progress struct {
	ReportID     string `json:"report_id,omitempty"`
	SimulationID string `json:"simulation_id,omitempty"`
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
	Message      string `json:"message"`
	UpdatedAt    string `json:"updated_at"`
}

type Store interface {
	CreateReport(reportID string, meta ReportMeta) error
	SaveMeta(reportID string, meta ReportMeta) error
	LoadMeta(reportID string) (ReportMeta, error)
	SaveProgress(reportID string, progress Progress) error
	LoadProgress(reportID string) (Progress, error)
	SaveSection(reportID string, index int, title string, content string) error
	LoadSections(reportID string) ([]map[string]any, error)
	SaveMarkdown(reportID string, markdown string) error
	LoadMarkdown(reportID string) (string, error)
	DeleteReport(reportID string) error
	ListReports(simulationID string, limit int) ([]ReportMeta, error)
	GetAgentLog(reportID string, fromLine int) (map[string]any, error)
	GetAgentLogStream(reportID string) ([]map[string]any, error)
	GetConsoleLog(reportID string, fromLine int) (map[string]any, error)
	GetConsoleLogStream(reportID string) ([]string, error)
	FindBySimulation(simulationID string) (ReportMeta, bool, error)
}

type FileStore struct {
	ReportsDir string
}

func NewFileStore(reportsDir string) *FileStore {
	return &FileStore{ReportsDir: reportsDir}
}

func (s *FileStore) reportDir(reportID string) string { return filepath.Join(s.ReportsDir, reportID) }
func (s *FileStore) metaPath(reportID string) string {
	return filepath.Join(s.reportDir(reportID), "meta.json")
}
func (s *FileStore) progressPath(reportID string) string {
	return filepath.Join(s.reportDir(reportID), "progress.json")
}
func (s *FileStore) markdownPath(reportID string) string {
	return filepath.Join(s.reportDir(reportID), "full_report.md")
}
func (s *FileStore) agentLogPath(reportID string) string {
	return filepath.Join(s.reportDir(reportID), "agent_log.jsonl")
}
func (s *FileStore) consoleLogPath(reportID string) string {
	return filepath.Join(s.reportDir(reportID), "console_log.txt")
}
func (s *FileStore) sectionPath(reportID string, index int) string {
	return filepath.Join(s.reportDir(reportID), "section_"+twoDigits(index)+".md")
}

func (s *FileStore) CreateReport(reportID string, meta ReportMeta) error {
	if err := os.MkdirAll(s.reportDir(reportID), 0o755); err != nil {
		return err
	}
	return s.SaveMeta(reportID, meta)
}

func (s *FileStore) SaveMeta(reportID string, meta ReportMeta) error {
	if err := os.MkdirAll(s.reportDir(reportID), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(reportID), raw, 0o644)
}

func (s *FileStore) LoadMeta(reportID string) (ReportMeta, error) {
	var meta ReportMeta
	raw, err := os.ReadFile(s.metaPath(reportID))
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return meta, err
	}
	if md, err := os.ReadFile(s.markdownPath(reportID)); err == nil {
		meta.MarkdownContent = string(md)
	}
	return meta, nil
}

func (s *FileStore) SaveProgress(reportID string, progress Progress) error {
	progress.ReportID = reportID
	progress.UpdatedAt = time.Now().Format(time.RFC3339)
	raw, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.progressPath(reportID), raw, 0o644)
}

func (s *FileStore) LoadProgress(reportID string) (Progress, error) {
	var progress Progress
	raw, err := os.ReadFile(s.progressPath(reportID))
	if err != nil {
		return progress, err
	}
	if err := json.Unmarshal(raw, &progress); err != nil {
		return progress, err
	}
	return progress, nil
}

func (s *FileStore) SaveSection(reportID string, index int, title string, content string) error {
	if err := os.MkdirAll(s.reportDir(reportID), 0o755); err != nil {
		return err
	}
	body := "## " + title + "\n\n" + strings.TrimSpace(content) + "\n"
	return os.WriteFile(s.sectionPath(reportID, index), []byte(body), 0o644)
}

func (s *FileStore) LoadSections(reportID string) ([]map[string]any, error) {
	if _, err := os.Stat(s.reportDir(reportID)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	dirEntries, err := os.ReadDir(s.reportDir(reportID))
	if err != nil {
		return nil, err
	}
	var sections []map[string]any
	for _, entry := range dirEntries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "section_") || !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.reportDir(reportID), name))
		if err != nil {
			return nil, err
		}
		sections = append(sections, map[string]any{
			"filename":      name,
			"section_index": parseSectionIndex(name),
			"content":       string(raw),
		})
	}
	sort.Slice(sections, func(i, j int) bool {
		return sections[i]["section_index"].(int) < sections[j]["section_index"].(int)
	})
	return sections, nil
}

func (s *FileStore) SaveMarkdown(reportID string, markdown string) error {
	if err := os.MkdirAll(s.reportDir(reportID), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.markdownPath(reportID), []byte(markdown), 0o644)
}

func (s *FileStore) LoadMarkdown(reportID string) (string, error) {
	raw, err := os.ReadFile(s.markdownPath(reportID))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (s *FileStore) DeleteReport(reportID string) error {
	return os.RemoveAll(s.reportDir(reportID))
}

func (s *FileStore) ListReports(simulationID string, limit int) ([]ReportMeta, error) {
	if err := os.MkdirAll(s.ReportsDir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.ReportsDir)
	if err != nil {
		return nil, err
	}
	reports := make([]ReportMeta, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := s.LoadMeta(entry.Name())
		if err != nil {
			continue
		}
		if simulationID != "" && meta.SimulationID != simulationID {
			continue
		}
		reports = append(reports, meta)
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].CreatedAt > reports[j].CreatedAt
	})
	if limit > 0 && len(reports) > limit {
		reports = reports[:limit]
	}
	return reports, nil
}

func (s *FileStore) FindBySimulation(simulationID string) (ReportMeta, bool, error) {
	reports, err := s.ListReports(simulationID, 1)
	if err != nil {
		return ReportMeta{}, false, err
	}
	if len(reports) == 0 {
		return ReportMeta{}, false, nil
	}
	return reports[0], true, nil
}

func (s *FileStore) GetConsoleLog(reportID string, fromLine int) (map[string]any, error) {
	logPath := s.consoleLogPath(reportID)
	if _, err := os.Stat(logPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{"logs": []string{}, "total_lines": 0, "from_line": fromLine, "has_more": false}, nil
		}
		return nil, err
	}
	raw, err := os.ReadFile(logPath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if fromLine < 0 {
		fromLine = 0
	}
	if fromLine > len(lines) {
		fromLine = len(lines)
	}
	return map[string]any{"logs": lines[fromLine:], "total_lines": len(lines), "from_line": fromLine, "has_more": false}, nil
}

func (s *FileStore) GetConsoleLogStream(reportID string) ([]string, error) {
	data, err := s.GetConsoleLog(reportID, 0)
	if err != nil {
		return nil, err
	}
	if logs, ok := data["logs"].([]string); ok {
		return logs, nil
	}
	raw, _ := data["logs"].([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		out = append(out, fmt.Sprint(item))
	}
	return out, nil
}

func (s *FileStore) GetAgentLog(reportID string, fromLine int) (map[string]any, error) {
	logPath := s.agentLogPath(reportID)
	if _, err := os.Stat(logPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{"logs": []map[string]any{}, "total_lines": 0, "from_line": fromLine, "has_more": false}, nil
		}
		return nil, err
	}
	raw, err := os.ReadFile(logPath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if fromLine < 0 {
		fromLine = 0
	}
	if fromLine > len(lines) {
		fromLine = len(lines)
	}
	logs := make([]map[string]any, 0, len(lines)-fromLine)
	for _, line := range lines[fromLine:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		entry := map[string]any{}
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			logs = append(logs, entry)
		}
	}
	return map[string]any{"logs": logs, "total_lines": len(lines), "from_line": fromLine, "has_more": false}, nil
}

func (s *FileStore) GetAgentLogStream(reportID string) ([]map[string]any, error) {
	data, err := s.GetAgentLog(reportID, 0)
	if err != nil {
		return nil, err
	}
	logs, _ := data["logs"].([]map[string]any)
	return logs, nil
}

func twoDigits(v int) string {
	if v < 10 {
		return "0" + strconv.Itoa(v)
	}
	return strconv.Itoa(v)
}

func parseSectionIndex(name string) int {
	base := strings.TrimSuffix(strings.TrimPrefix(name, "section_"), ".md")
	if base == "" {
		return 0
	}
	if len(base) > 1 && base[0] == '0' {
		base = base[1:]
	}
	value := 0
	for _, r := range base {
		if r < '0' || r > '9' {
			return value
		}
		value = value*10 + int(r-'0')
	}
	return value
}

var ErrProgressNotFound = errors.New("report progress not found")
