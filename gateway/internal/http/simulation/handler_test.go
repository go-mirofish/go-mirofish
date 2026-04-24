package simulationhttp

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	intsimulation "github.com/go-mirofish/go-mirofish/gateway/internal/simulation"
	simulationstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/simulation"
)

func TestHandleRouteReadAndAdminSurface(t *testing.T) {
	t.Parallel()

	handler, root := newSimulationHandlerFixture(t)

	tests := []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"state", http.MethodGet, "/api/simulation/sim-1", "", http.StatusOK, `"simulation_id":"sim-1"`},
		{"run status", http.MethodGet, "/api/simulation/sim-1/status", "", http.StatusOK, `"runner_status":"running"`},
		{"run status detail", http.MethodGet, "/api/simulation/sim-1/run-status/detail", "", http.StatusOK, `"all_actions":[`},
		{"profiles", http.MethodGet, "/api/simulation/sim-1/profiles?platform=reddit", "", http.StatusOK, `"count":1`},
		{"config realtime", http.MethodGet, "/api/simulation/sim-1/config/realtime", "", http.StatusOK, `"config_generated":true`},
		{"actions", http.MethodGet, "/api/simulation/sim-1/actions?platform=reddit", "", http.StatusOK, `"count":2`},
		{"timeline", http.MethodGet, "/api/simulation/sim-1/timeline", "", http.StatusOK, `"rounds_count":1`},
		{"agent stats", http.MethodGet, "/api/simulation/sim-1/agent-stats", "", http.StatusOK, `"agents_count":2`},
		{"posts", http.MethodGet, "/api/simulation/sim-1/posts?platform=reddit", "", http.StatusOK, `"posts":[`},
		{"comments", http.MethodGet, "/api/simulation/sim-1/comments", "", http.StatusOK, `"comments":[`},
		{"config download", http.MethodGet, "/api/simulation/sim-1/config/download", "", http.StatusOK, `"simulation_id":"sim-1"`},
		{"script download", http.MethodGet, "/api/simulation/script/run_parallel_simulation.py/download", "", http.StatusOK, "print('runner')"},
		{"list", http.MethodGet, "/api/simulation/list?project_id=proj-1", "", http.StatusOK, `"count":1`},
		{"history", http.MethodGet, "/api/simulation/history?limit=20", "", http.StatusOK, `"report_id":"report-1"`},
		{"interview history", http.MethodPost, "/api/simulation/interview/history", `{"simulation_id":"sim-1","limit":10}`, http.StatusOK, `"history":[`},
		{"unknown route", http.MethodGet, "/api/simulation/sim-1/unknown", "", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			handler.HandleRoute(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body %q missing %q", rec.Body.String(), tt.wantBody)
			}
		})
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/simulation/create", strings.NewReader(`{"project_id":"proj-1"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handler.HandleRoute(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want 200", createRec.Code)
	}

	var createPayload map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create payload: %v", err)
	}
	createData, _ := createPayload["data"].(map[string]any)
	createdID, _ := createData["simulation_id"].(string)
	if !strings.HasPrefix(createdID, "sim_") {
		t.Fatalf("simulation_id = %q, want sim_*", createdID)
	}
	if _, err := os.Stat(filepath.Join(root, "simulations", createdID, "state.json")); err != nil {
		t.Fatalf("expected created state.json: %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/api/simulation/delete", strings.NewReader(`{"simulation_id":"sim-delete"}`))
	deleteReq.Header.Set("Content-Type", "application/json")
	deleteRec := httptest.NewRecorder()
	handler.HandleRoute(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", deleteRec.Code)
	}
	if _, err := os.Stat(filepath.Join(root, "simulations", "sim-delete")); !os.IsNotExist(err) {
		t.Fatalf("expected sim-delete removed, stat err = %v", err)
	}
}

func newSimulationHandlerFixture(t *testing.T) (*Handler, string) {
	t.Helper()

	root := t.TempDir()
	simDir := filepath.Join(root, "simulations", "sim-1")
	deleteDir := filepath.Join(root, "simulations", "sim-delete")
	projectDir := filepath.Join(root, "projects", "proj-1")
	reportDir := filepath.Join(root, "reports", "report-1")

	for _, dir := range []string{
		filepath.Join(simDir, "reddit"),
		filepath.Join(simDir, "twitter"),
		deleteDir,
		projectDir,
		reportDir,
		filepath.Join(root, "scripts"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", dir, err)
		}
	}

	writeJSONFile(t, filepath.Join(projectDir, "project.json"), map[string]any{
		"project_id":             "proj-1",
		"graph_id":               "graph-1",
		"simulation_requirement": "Simulate a test scenario",
		"files": []map[string]any{
			{"filename": "seed.pdf"},
			{"filename": "notes.md"},
		},
	})
	writeJSONFile(t, filepath.Join(simDir, "state.json"), map[string]any{
		"simulation_id":    "sim-1",
		"project_id":       "proj-1",
		"graph_id":         "graph-1",
		"status":           "ready",
		"config_generated": true,
		"entities_count":   1,
		"created_at":       "2026-01-01T00:00:00Z",
	})
	writeJSONFile(t, filepath.Join(simDir, "run_state.json"), map[string]any{
		"simulation_id": "sim-1",
		"runner_status": "running",
		"current_round": 1,
		"total_rounds":  2,
	})
	writeJSONFile(t, filepath.Join(simDir, "simulation_config.json"), map[string]any{
		"simulation_id":          "sim-1",
		"simulation_requirement": "Simulate a test scenario",
		"agent_configs":          []map[string]any{{"agent_id": 1}},
		"time_config":            map[string]any{"total_simulation_hours": 12, "minutes_per_round": 60},
		"event_config":           map[string]any{"initial_posts": []any{"p1"}, "hot_topics": []any{"t1"}},
		"twitter_config":         map[string]any{"enabled": true},
		"reddit_config":          map[string]any{"enabled": true},
		"generated_at":           "2026-01-01T00:00:00Z",
		"llm_model":              "stub-model",
	})
	writeJSONFile(t, filepath.Join(simDir, "reddit_profiles.json"), []map[string]any{{"agent_id": 1, "name": "Alice"}})
	writeCSVFile(t, filepath.Join(simDir, "twitter_profiles.csv"), [][]string{{"agent_id", "name"}, {"1", "Alice"}})
	writeActionLog(t, filepath.Join(simDir, "reddit", "actions.jsonl"), []map[string]any{
		{"round": 1, "timestamp": "2026-01-01T00:00:02Z", "platform": "reddit", "agent_id": 1, "agent_name": "Alice", "action_type": "CREATE_POST", "action_args": map[string]any{"content": "hello"}, "success": true},
		{"round": 1, "timestamp": "2026-01-01T00:00:03Z", "platform": "reddit", "agent_id": 2, "agent_name": "Bob", "action_type": "CREATE_COMMENT", "action_args": map[string]any{"content": "reply"}, "success": true},
	})
	writeActionLog(t, filepath.Join(simDir, "twitter", "actions.jsonl"), []map[string]any{
		{"round": 1, "timestamp": "2026-01-01T00:00:01Z", "platform": "twitter", "agent_id": 1, "agent_name": "Alice", "action_type": "POST_TWEET", "action_args": map[string]any{"content": "tweet"}, "success": true},
	})
	writeJSONFile(t, filepath.Join(reportDir, "meta.json"), map[string]any{
		"report_id":     "report-1",
		"simulation_id": "sim-1",
		"created_at":    "2026-01-02T00:00:00Z",
		"status":        "completed",
	})
	if err := os.WriteFile(filepath.Join(root, "scripts", "run_parallel_simulation.py"), []byte("print('runner')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile script: %v", err)
	}
	createInterviewDB(t, filepath.Join(simDir, "twitter_simulation.db"), []map[string]any{
		{"user_id": 1, "info": map[string]any{"prompt": "how are you?", "response": "fine"}, "created_at": "2026-01-01T00:00:04Z"},
	})
	createInterviewDB(t, filepath.Join(simDir, "reddit_simulation.db"), []map[string]any{
		{"user_id": 2, "info": map[string]any{"prompt": "what changed?", "response": "a lot"}, "created_at": "2026-01-01T00:00:05Z"},
	})

	store := simulationstore.New(
		filepath.Join(root, "simulations"),
		filepath.Join(root, "scripts"),
		filepath.Join(root, "projects"),
		filepath.Join(root, "reports"),
	)
	service := intsimulation.NewService(store, nil)
	return NewHandler(service, store, graphServiceStub{data: map[string]any{
		"graph_id": "graph-1",
		"nodes": []map[string]any{
			{"uuid": "node-1", "name": "Alice", "labels": []string{"Entity", "PERSON"}, "summary": "Person", "attributes": map[string]any{}},
			{"uuid": "node-2", "name": "Acme", "labels": []string{"Entity", "COMPANY"}, "summary": "Company", "attributes": map[string]any{}},
		},
		"edges": []map[string]any{
			{"name": "WORKS_AT", "fact": "Alice works at Acme", "source_node_uuid": "node-1", "target_node_uuid": "node-2"},
		},
	}}), root
}

type graphServiceStub struct {
	data map[string]any
	err  error
}

func (g graphServiceStub) GetGraphData(ctx context.Context, graphID string) (map[string]any, error) {
	return g.data, g.err
}

func writeJSONFile(t *testing.T, path string, payload any) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func writeCSVFile(t *testing.T, path string, rows [][]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create CSV: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	if err := writer.WriteAll(rows); err != nil {
		t.Fatalf("WriteAll CSV: %v", err)
	}
}

func writeActionLog(t *testing.T, path string, items []map[string]any) {
	t.Helper()
	var lines []string
	for _, item := range items {
		raw, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("Marshal action: %v", err)
		}
		lines = append(lines, string(raw))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile actions: %v", err)
	}
}

func createInterviewDB(t *testing.T, dbPath string, rows []map[string]any) {
	t.Helper()

	script := `
import json
import sqlite3
import sys

db_path = sys.argv[1]
rows = json.loads(sys.argv[2])
conn = sqlite3.connect(db_path)
cur = conn.cursor()
cur.execute("CREATE TABLE trace (user_id INTEGER, action TEXT, info TEXT, created_at TEXT)")
for row in rows:
    cur.execute(
        "INSERT INTO trace (user_id, action, info, created_at) VALUES (?, 'interview', ?, ?)",
        (row["user_id"], json.dumps(row["info"]), row["created_at"]),
    )
conn.commit()
conn.close()
`

	rawRows, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("marshal interview rows: %v", err)
	}

	cmd := exec.Command("python3", "-c", script, dbPath, string(rawRows))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create interview db: %v\n%s", err, output)
	}
}
