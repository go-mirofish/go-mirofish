package reportstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreReadAndDeleteParity(t *testing.T) {
	reportsDir := filepath.Join(t.TempDir(), "reports")
	store := NewFileStore(reportsDir)

	meta := ReportMeta{
		ReportID:              "report-1",
		SimulationID:          "sim-1",
		GraphID:               "graph-1",
		SimulationRequirement: "test requirement",
		Status:                "completed",
		CreatedAt:             "2026-04-24T00:00:00Z",
	}
	if err := store.CreateReport("report-1", meta); err != nil {
		t.Fatalf("create report: %v", err)
	}
	if err := store.SaveMarkdown("report-1", "# Report\n\nBody"); err != nil {
		t.Fatalf("save markdown: %v", err)
	}
	if err := store.SaveProgress("report-1", Progress{Status: "completed", Progress: 100, Message: "done"}); err != nil {
		t.Fatalf("save progress: %v", err)
	}
	if err := store.SaveSection("report-1", 1, "Section One", "Content"); err != nil {
		t.Fatalf("save section: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "report-1", "agent_log.jsonl"), []byte("{\"action\":\"tool_call\"}\n"), 0o644); err != nil {
		t.Fatalf("write agent log: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "report-1", "console_log.txt"), []byte("line one\nline two\n"), 0o644); err != nil {
		t.Fatalf("write console log: %v", err)
	}

	loadedMeta, err := store.LoadMeta("report-1")
	if err != nil {
		t.Fatalf("load meta: %v", err)
	}
	if loadedMeta.MarkdownContent != "# Report\n\nBody" {
		t.Fatalf("expected markdown hydration, got %q", loadedMeta.MarkdownContent)
	}

	sections, err := store.LoadSections("report-1")
	if err != nil {
		t.Fatalf("load sections: %v", err)
	}
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0]["section_index"].(int) != 1 {
		t.Fatalf("expected section index 1, got %#v", sections[0]["section_index"])
	}

	reports, err := store.ListReports("", 10)
	if err != nil {
		t.Fatalf("list reports: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}

	bySim, found, err := store.FindBySimulation("sim-1")
	if err != nil {
		t.Fatalf("find by simulation: %v", err)
	}
	if !found || bySim.ReportID != "report-1" {
		t.Fatalf("expected report-1, got found=%v report=%+v", found, bySim)
	}

	agentLog, err := store.GetAgentLog("report-1", 0)
	if err != nil {
		t.Fatalf("load agent log: %v", err)
	}
	if len(agentLog["logs"].([]map[string]any)) != 1 {
		t.Fatalf("expected 1 agent log entry, got %#v", agentLog["logs"])
	}

	consoleLog, err := store.GetConsoleLog("report-1", 1)
	if err != nil {
		t.Fatalf("load console log: %v", err)
	}
	logs, _ := consoleLog["logs"].([]string)
	if len(logs) != 1 || logs[0] != "line two" {
		t.Fatalf("unexpected console logs: %#v", consoleLog["logs"])
	}

	if err := store.DeleteReport("report-1"); err != nil {
		t.Fatalf("delete report: %v", err)
	}
	if _, err := os.Stat(filepath.Join(reportsDir, "report-1")); !os.IsNotExist(err) {
		t.Fatalf("expected report dir removed, stat err=%v", err)
	}
}

func TestFileStoreLegacyCompatibilityAndHelpers(t *testing.T) {
	reportsDir := filepath.Join(t.TempDir(), "reports")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		t.Fatalf("mkdir reports dir: %v", err)
	}
	store := NewFileStore(reportsDir)

	if err := os.WriteFile(filepath.Join(reportsDir, "legacy.json"), []byte(`{"report_id":"legacy","simulation_id":"sim-legacy","graph_id":"graph-legacy","simulation_requirement":"legacy","status":"completed","created_at":"2026-04-24T00:00:00Z"}`), 0o644); err != nil {
		t.Fatalf("write legacy meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "legacy.md"), []byte("# Legacy\n\nBody"), 0o644); err != nil {
		t.Fatalf("write legacy markdown: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(reportsDir, "legacy"), 0o755); err != nil {
		t.Fatalf("mkdir legacy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "legacy", "meta.json"), []byte(`{"report_id":"legacy","simulation_id":"sim-legacy","graph_id":"graph-legacy","simulation_requirement":"legacy","status":"completed","created_at":"2026-04-24T00:00:00Z"}`), 0o644); err != nil {
		t.Fatalf("write folder meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportsDir, "legacy", "full_report.md"), []byte("# Legacy\n\nBody"), 0o644); err != nil {
		t.Fatalf("write folder markdown: %v", err)
	}

	meta, err := store.LoadMeta("legacy")
	if err != nil {
		t.Fatalf("load legacy meta: %v", err)
	}
	if meta.MarkdownContent != "# Legacy\n\nBody" {
		t.Fatalf("expected legacy markdown hydration, got %q", meta.MarkdownContent)
	}

	sections, err := store.LoadSections("missing")
	if err != nil {
		t.Fatalf("expected missing dir to return empty sections, got %v", err)
	}
	if len(sections) != 0 {
		t.Fatalf("expected empty sections, got %d", len(sections))
	}

	if err := store.DeleteReport("legacy"); err != nil {
		t.Fatalf("delete legacy report: %v", err)
	}

	if got := twoDigits(1); got != "01" {
		t.Fatalf("expected 01, got %s", got)
	}
	if got := parseSectionIndex("section_03.md"); got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
}

func TestFileStoreValidationAndCorruption(t *testing.T) {
	reportsDir := filepath.Join(t.TempDir(), "reports")
	store := NewFileStore(reportsDir)

	if err := store.SaveMeta("report-bad", ReportMeta{}); err == nil {
		t.Fatalf("expected SaveMeta validation error")
	}
	if err := store.SaveProgress("report-bad", Progress{Status: "completed", Progress: 101}); err == nil {
		t.Fatalf("expected SaveProgress validation error")
	}
	if err := store.SaveMarkdown("report-bad", "   "); err == nil {
		t.Fatalf("expected SaveMarkdown validation error")
	}
	if err := store.SaveSection("report-bad", 1, "", "body"); err == nil {
		t.Fatalf("expected SaveSection validation error")
	}

	reportDir := filepath.Join(reportsDir, "report-corrupt")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("mkdir corrupt report dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "meta.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("write corrupt meta: %v", err)
	}
	if _, err := store.LoadMeta("report-corrupt"); err == nil {
		t.Fatalf("expected LoadMeta error for corrupt json")
	}
}
