package simulationstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadRunStateUsesArtifactContract(t *testing.T) {
	root := t.TempDir()
	store := New(filepath.Join(root, "simulations"), filepath.Join(root, "scripts"), filepath.Join(root, "projects"), filepath.Join(root, "reports"))
	simDir := store.SimulationDir("sim-1")
	if err := os.MkdirAll(simDir, 0o755); err != nil {
		t.Fatalf("mkdir sim dir: %v", err)
	}
	raw := `{
	  "worker_protocol_version": "1.0",
	  "simulation_id": "sim-1",
	  "runner_status": "running",
	  "current_round": 1,
	  "total_rounds": 2,
	  "progress_percent": 50,
	  "twitter_actions_count": 1,
	  "reddit_actions_count": 1,
	  "total_actions_count": 2,
	  "updated_at": "2026-01-01T00:00:00Z"
	}`
	if err := os.WriteFile(filepath.Join(simDir, "run_state.json"), []byte(raw), 0o644); err != nil {
		t.Fatalf("write run_state.json: %v", err)
	}
	payload, err := store.ReadRunState("sim-1")
	if err != nil {
		t.Fatalf("ReadRunState: %v", err)
	}
	if payload["simulation_id"] != "sim-1" {
		t.Fatalf("unexpected simulation_id: %#v", payload["simulation_id"])
	}
}

func TestReadActionLogsFallsBackForLegacyInvalidJSONL(t *testing.T) {
	root := t.TempDir()
	store := New(filepath.Join(root, "simulations"), filepath.Join(root, "scripts"), filepath.Join(root, "projects"), filepath.Join(root, "reports"))
	redditDir := filepath.Join(store.SimulationDir("sim-1"), "reddit")
	if err := os.MkdirAll(redditDir, 0o755); err != nil {
		t.Fatalf("mkdir reddit dir: %v", err)
	}
	legacy := []map[string]any{
		{"timestamp": "2026-01-01T00:00:00Z", "platform": "reddit", "action_type": "CREATE_POST"},
	}
	raw, _ := json.Marshal(legacy[0])
	if err := os.WriteFile(filepath.Join(redditDir, "actions.jsonl"), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write actions.jsonl: %v", err)
	}
	items, err := store.ReadActionLogs("sim-1", "reddit")
	if err != nil {
		t.Fatalf("ReadActionLogs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestInterviewHistoryPrefersJSONLArtifacts(t *testing.T) {
	root := t.TempDir()
	store := New(filepath.Join(root, "simulations"), filepath.Join(root, "scripts"), filepath.Join(root, "projects"), filepath.Join(root, "reports"))
	simDir := store.SimulationDir("sim-1")
	if err := os.MkdirAll(simDir, 0o755); err != nil {
		t.Fatalf("mkdir sim dir: %v", err)
	}
	lines := []string{
		`{"agent_id":1,"prompt":"hello","response":"hi","timestamp":"2026-01-01T00:00:02Z","platform":"twitter"}`,
		`{"agent_id":2,"prompt":"status","response":"ok","timestamp":"2026-01-01T00:00:03Z","platform":"twitter"}`,
	}
	if err := os.WriteFile(filepath.Join(simDir, "twitter_interviews.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write interview history: %v", err)
	}
	items, err := store.InterviewHistory("sim-1", "twitter", nil, 10)
	if err != nil {
		t.Fatalf("InterviewHistory: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0]["agent_id"].(float64) != 2 {
		t.Fatalf("expected newest item first, got %#v", items[0])
	}
}

func TestReadRuntimeStateFallsBackToRunState(t *testing.T) {
	root := t.TempDir()
	store := New(filepath.Join(root, "simulations"), filepath.Join(root, "scripts"), filepath.Join(root, "projects"), filepath.Join(root, "reports"))
	simDir := store.SimulationDir("sim-1")
	if err := os.MkdirAll(simDir, 0o755); err != nil {
		t.Fatalf("mkdir sim dir: %v", err)
	}
	raw := `{
	  "worker_protocol_version": "1.0",
	  "simulation_id": "sim-1",
	  "runner_status": "completed",
	  "current_round": 2,
	  "total_rounds": 2,
	  "progress_percent": 100,
	  "twitter_current_round": 2,
	  "twitter_actions_count": 2,
	  "reddit_actions_count": 1,
	  "total_actions_count": 3,
	  "updated_at": "2026-01-01T00:00:00Z"
	}`
	if err := os.WriteFile(filepath.Join(simDir, "run_state.json"), []byte(raw), 0o644); err != nil {
		t.Fatalf("write run_state.json: %v", err)
	}
	payload, err := store.ReadRuntimeState("sim-1")
	if err != nil {
		t.Fatalf("ReadRuntimeState: %v", err)
	}
	if payload["status"] != "completed" {
		t.Fatalf("expected synthesized status completed, got %#v", payload["status"])
	}
	if payload["twitter_current_round"] != float64(2) {
		t.Fatalf("expected synthesized platform progress, got %#v", payload["twitter_current_round"])
	}
}
