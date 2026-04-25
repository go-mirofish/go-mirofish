package worker

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeBridgeLifecycleAndInterview(t *testing.T) {
	root := t.TempDir()
	simDir := filepath.Join(root, "sim-1")
	if err := os.MkdirAll(simDir, 0o755); err != nil {
		t.Fatalf("mkdir sim dir: %v", err)
	}
	config := map[string]any{
		"simulation_id": "sim-1",
		"agent_configs": []map[string]any{{"agent_id": 1}, {"agent_id": 2}},
		"time_config": map[string]any{
			"total_simulation_hours": 1,
			"minutes_per_round":      30,
		},
		"twitter_config": map[string]any{"enabled": true},
		"reddit_config":  map[string]any{"enabled": true},
	}
	raw, _ := json.Marshal(config)
	if err := os.WriteFile(filepath.Join(simDir, "simulation_config.json"), raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simDir, "reddit_profiles.json"), []byte(`[{"agent_id":1,"name":"Alice"},{"agent_id":2,"name":"Bob"}]`), 0o644); err != nil {
		t.Fatalf("write reddit profiles: %v", err)
	}
	twitterFile, err := os.Create(filepath.Join(simDir, "twitter_profiles.csv"))
	if err != nil {
		t.Fatalf("create twitter csv: %v", err)
	}
	writer := csv.NewWriter(twitterFile)
	_ = writer.Write([]string{"user_id", "name"})
	_ = writer.Write([]string{"1", "Alice"})
	_ = writer.Write([]string{"2", "Bob"})
	writer.Flush()
	_ = twitterFile.Close()

	bridge := NewNativeBridge(root)
	startResp, err := bridge.StartSimulation(context.Background(), StartRequest{
		SimulationID:          "sim-1",
		Platform:              PlatformParallel,
		WorkerProtocolVersion: ProtocolVersion,
	})
	if err != nil {
		t.Fatalf("StartSimulation: %v", err)
	}
	if startResp.RunnerStatus != "running" {
		t.Fatalf("unexpected runner status: %q", startResp.RunnerStatus)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, err := bridge.EnvStatus(context.Background(), EnvStatusRequest{SimulationID: "sim-1"})
		if err == nil && status.EnvAlive {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	interview, err := bridge.Interview(context.Background(), InterviewRequest{
		SimulationID: "sim-1",
		AgentID:      1,
		Prompt:       "status",
		Platform:     PlatformTwitter,
	})
	if err != nil {
		t.Fatalf("Interview: %v", err)
	}
	if !interview.Success {
		t.Fatalf("expected successful interview")
	}

	batch, err := bridge.BatchInterview(context.Background(), BatchInterviewRequest{
		SimulationID: "sim-1",
		Interviews:   []BatchInterviewItem{{AgentID: 1, Prompt: "hello"}},
	})
	if err != nil {
		t.Fatalf("BatchInterview: %v", err)
	}
	if !batch.Success {
		t.Fatalf("expected successful batch interview")
	}

	stopResp, err := bridge.StopSimulation(context.Background(), StopRequest{SimulationID: "sim-1"})
	if err != nil {
		t.Fatalf("StopSimulation: %v", err)
	}
	if stopResp["runner_status"] != "stopped" && stopResp["runner_status"] != "completed" {
		t.Fatalf("unexpected stop status: %#v", stopResp["runner_status"])
	}

	if _, err := os.Stat(filepath.Join(simDir, "twitter", "actions.jsonl")); err != nil {
		t.Fatalf("expected twitter actions.jsonl: %v", err)
	}
	if _, err := os.Stat(filepath.Join(simDir, "twitter_interviews.jsonl")); err != nil {
		t.Fatalf("expected twitter_interviews.jsonl: %v", err)
	}
}
