package localfs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTripAndMissingBehavior(t *testing.T) {
	root := t.TempDir()
	store := New(
		filepath.Join(root, "projects"),
		filepath.Join(root, "reports"),
		filepath.Join(root, "tasks"),
		filepath.Join(root, "sims"),
		filepath.Join(root, "scripts"),
	)

	project := map[string]any{"project_id": "proj-1", "name": "Demo"}
	if err := store.WriteProject("proj-1", project); err != nil {
		t.Fatalf("WriteProject: %v", err)
	}
	gotProject, err := store.ReadProject("proj-1")
	if err != nil || gotProject["name"] != "Demo" {
		t.Fatalf("ReadProject: %#v %v", gotProject, err)
	}

	task := map[string]any{"task_id": "task-1", "status": "pending"}
	if err := store.WriteTask("task-1", task); err != nil {
		t.Fatalf("WriteTask: %v", err)
	}
	gotTask, err := store.ReadTask("task-1")
	if err != nil || gotTask["status"] != "pending" {
		t.Fatalf("ReadTask: %#v %v", gotTask, err)
	}

	sim := map[string]any{"simulation_id": "sim-1", "project_id": "proj-1", "created_at": "2026-01-01T00:00:00Z"}
	if err := store.WriteSimulation("sim-1", sim); err != nil {
		t.Fatalf("WriteSimulation: %v", err)
	}
	gotSim, err := store.ReadSimulation("sim-1")
	if err != nil || gotSim["project_id"] != "proj-1" {
		t.Fatalf("ReadSimulation: %#v %v", gotSim, err)
	}

	if _, err := store.ReadProject("missing"); err == nil {
		t.Fatalf("expected missing project error")
	}

	if err := os.MkdirAll(filepath.Dir(store.ProjectMetaPath("broken")), 0o755); err != nil {
		t.Fatalf("mkdir broken project dir: %v", err)
	}
	if err := os.WriteFile(store.ProjectMetaPath("broken"), []byte("{"), 0o644); err != nil {
		t.Fatalf("write broken project: %v", err)
	}
	if _, err := store.ReadProject("broken"); err == nil {
		t.Fatalf("expected corrupt project error")
	}
}

func TestStoreSimulationArtifactsAndReports(t *testing.T) {
	root := t.TempDir()
	store := New(
		filepath.Join(root, "projects"),
		filepath.Join(root, "reports"),
		filepath.Join(root, "tasks"),
		filepath.Join(root, "sims"),
		filepath.Join(root, "scripts"),
	)

	if err := os.MkdirAll(store.SimulationDir("sim-1"), 0o755); err != nil {
		t.Fatalf("MkdirAll sim: %v", err)
	}
	if err := os.WriteFile(store.SimulationConfigPath("sim-1"), []byte(`{"simulation_id":"sim-1"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.SimulationDir("sim-1"), "reddit_profiles.json"), []byte(`[{"user_id":1}]`), 0o644); err != nil {
		t.Fatalf("write profiles: %v", err)
	}
	config, exists, _, err := store.ReadSimulationConfigWithMeta("sim-1")
	if err != nil || !exists || config["simulation_id"] != "sim-1" {
		t.Fatalf("ReadSimulationConfigWithMeta: %#v %v exists=%v", config, err, exists)
	}
	profiles, exists, _, err := store.ReadSimulationProfiles("sim-1", "reddit")
	if err != nil || !exists || len(profiles) != 1 {
		t.Fatalf("ReadSimulationProfiles: %#v %v exists=%v", profiles, err, exists)
	}

	reportDir := filepath.Join(store.ReportsDir, "report-1")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("MkdirAll report: %v", err)
	}
	meta := map[string]any{"report_id": "report-1", "simulation_id": "sim-1", "created_at": "2026-01-02T00:00:00Z"}
	rawMeta, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(reportDir, "meta.json"), rawMeta, 0o644); err != nil {
		t.Fatalf("write report meta: %v", err)
	}
	report, err := store.FindReportBySimulation("sim-1")
	if err != nil || report["report_id"] != "report-1" {
		t.Fatalf("FindReportBySimulation: %#v %v", report, err)
	}

	instructions := store.BuildRunInstructions("sim-1")
	if instructions["config_file"] == "" {
		t.Fatalf("expected config_file in run instructions")
	}
}
