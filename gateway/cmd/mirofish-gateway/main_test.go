package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)


func newTestGateway(t *testing.T, _ func(*http.Request) (*http.Response, error)) *gateway {
	t.Helper()

	return newGateway(config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		frontendDistDir: "frontend/dist",
	})
}


func newWorkerGateway(t *testing.T) *gateway {
	t.Helper()

	tmpDir := t.TempDir()
	return newGateway(config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		frontendDistDir: "frontend/dist",
		simulationsDir:  filepath.Join(tmpDir, "simulations"),
		scriptsDir:      filepath.Join(tmpDir, "scripts"),
	})
}

func serveSimulation(t *testing.T, gw *gateway, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()
	buildSimulationHandler(gw.cfg).HandleRoute(rec, req)
}

func writeGatewayJSON(t *testing.T, path string, payload any) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeGatewayFile(t *testing.T, path string, content string, mode os.FileMode) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func waitForGatewayCommand(t *testing.T, simulationDir string) map[string]any {
	t.Helper()

	commandsDir := filepath.Join(simulationDir, "ipc_commands")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(commandsDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
					continue
				}
				raw, readErr := os.ReadFile(filepath.Join(commandsDir, entry.Name()))
				if readErr != nil {
					t.Fatalf("read command file: %v", readErr)
				}
				var payload map[string]any
				if err := json.Unmarshal(raw, &payload); err != nil {
					t.Fatalf("decode command file: %v", err)
				}
				return payload
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for ipc command in %s", commandsDir)
	return nil
}

func startGatewayIPCResponder(t *testing.T, simulationDir string, responder func(t *testing.T, command map[string]any) []byte) {
	t.Helper()

	go func() {
		command := waitForGatewayCommand(t, simulationDir)
		commandID, _ := command["command_id"].(string)
		responsePath := filepath.Join(simulationDir, "ipc_responses", commandID+".json")
		if err := os.WriteFile(responsePath, responder(t, command), 0o644); err != nil {
			t.Errorf("write response file: %v", err)
		}
	}()
}

func TestSimulationRunAliasForwardsToStart(t *testing.T) {
	tmpDir := t.TempDir()
	simID := "sim-1"
	simDir := filepath.Join(tmpDir, simID)
	if err := os.MkdirAll(simDir, 0o755); err != nil {
		t.Fatalf("mkdir simulation dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simDir, "simulation_config.json"), []byte(`{"time_config":{"total_simulation_hours":72,"minutes_per_round":60}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	gw := newGateway(config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		frontendDistDir: "frontend/dist",
		simulationsDir:  tmpDir,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/simulation/run", strings.NewReader(`{"simulation_id":"sim-1","platform":"parallel"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	serveSimulation(t, gw, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := payload["data"].(map[string]any)
	if data["runner_status"] != "running" {
		t.Fatalf("expected running runner_status, got %#v", data["runner_status"])
	}

	// Stop the simulation so the NativeBridge goroutine exits before TempDir cleanup (required on Windows).
	stopReq := httptest.NewRequest(http.MethodPost, "/api/simulation/stop", strings.NewReader(`{"simulation_id":"sim-1"}`))
	stopReq.Header.Set("Content-Type", "application/json")
	serveSimulation(t, gw, stopReq, httptest.NewRecorder())
	waitForSimTerminal(t, filepath.Join(tmpDir, simID))
}

func TestSimulationStatusAliasUsesIdleFallbackWithoutRunState(t *testing.T) {
	tmpDir := t.TempDir()
	gw := newTestGateway(t, func(r *http.Request) (*http.Response, error) {
		t.Fatalf("did not expect proxy call, got %s %s", r.Method, r.URL.Path)
		return nil, nil
	})
	gw.cfg.simulationsDir = tmpDir

	req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-9/status?verbose=1", nil)
	rec := httptest.NewRecorder()

	serveSimulation(t, gw, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := payload["data"].(map[string]any)
	if data["runner_status"] != "idle" {
		t.Fatalf("expected idle runner_status, got %#v", data["runner_status"])
	}
}

func TestSimulationRunStatusControlPlaneUsesDurableRunState(t *testing.T) {
	tmpDir := t.TempDir()
	simDir := filepath.Join(tmpDir, "sim-456")
	if err := os.MkdirAll(simDir, 0o755); err != nil {
		t.Fatalf("mkdir simulation dir: %v", err)
	}

	runState := map[string]any{
		"worker_protocol_version": "1.0",
		"simulation_id":           "sim-456",
		"runner_status":           "running",
		"current_round":           4,
		"total_rounds":            20,
		"progress_percent":        20.0,
		"simulated_hours":         2,
		"total_simulation_hours":  10,
		"twitter_running":         true,
		"reddit_running":          false,
		"twitter_actions_count":   12,
		"reddit_actions_count":    3,
		"total_actions_count":     15,
		"recent_actions":          []any{map[string]any{"platform": "twitter"}},
		"started_at":              "2026-04-24T00:00:00Z",
		"updated_at":              "2026-04-24T00:05:00Z",
	}
	raw, err := json.MarshalIndent(runState, "", "  ")
	if err != nil {
		t.Fatalf("marshal run state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simDir, "run_state.json"), raw, 0o644); err != nil {
		t.Fatalf("write run state: %v", err)
	}

	gw := newTestGateway(t, func(r *http.Request) (*http.Response, error) {
		t.Fatalf("did not expect proxy call, got %s %s", r.Method, r.URL.Path)
		return nil, nil
	})
	gw.cfg.simulationsDir = tmpDir

	t.Run("direct run-status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-456/run-status", nil)
		rec := httptest.NewRecorder()

		serveSimulation(t, gw, req, rec)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		data := payload["data"].(map[string]any)
		if data["runner_status"] != "running" {
			t.Fatalf("expected running, got %#v", data["runner_status"])
		}
		if data["current_round"].(float64) != 4 {
			t.Fatalf("expected current_round 4, got %#v", data["current_round"])
		}
		if _, ok := data["recent_actions"]; ok {
			t.Fatalf("expected run-status response to omit detail-only fields")
		}
	})

	t.Run("status alias", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-456/status", nil)
		rec := httptest.NewRecorder()

		serveSimulation(t, gw, req, rec)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		data := payload["data"].(map[string]any)
		if data["simulation_id"] != "sim-456" {
			t.Fatalf("expected sim-456, got %#v", data["simulation_id"])
		}
		if data["total_actions_count"].(float64) != 15 {
			t.Fatalf("expected total_actions_count 15, got %#v", data["total_actions_count"])
		}
	})
}

func waitForSimTerminal(t *testing.T, simDir string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(filepath.Join(simDir, "run_state.json"))
		if err != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		var state map[string]any
		if err := json.Unmarshal(raw, &state); err != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		st, _ := state["runner_status"].(string)
		if st == "stopped" || st == "completed" || st == "failed" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWorkerControlHandlers(t *testing.T) {
	type testCase struct {
		name       string
		body       string
		invoke     func(*gateway, http.ResponseWriter, *http.Request)
		setup      func(t *testing.T, gw *gateway)
		cleanup    func(t *testing.T, gw *gateway)
		wantStatus int
		assert     func(t *testing.T, payload map[string]any)
	}

	tests := []testCase{
		{
			name: "start success defaults to parallel platform",
			body: `{"simulation_id":"sim-start","max_rounds":3,"enable_graph_memory_update":true,"graph_id":"graph-9"}`,
			invoke: func(gw *gateway, w http.ResponseWriter, r *http.Request) {
				r.URL.Path = "/api/simulation/start"
				serveSimulation(t, gw, r, w.(*httptest.ResponseRecorder))
			},
			setup: func(t *testing.T, gw *gateway) {
				simDir := filepath.Join(gw.cfg.simulationsDir, "sim-start")
				writeGatewayJSON(t, filepath.Join(simDir, "simulation_config.json"), map[string]any{
					"time_config": map[string]any{"total_simulation_hours": 24},
				})
			},
			cleanup: func(t *testing.T, gw *gateway) {
				// Stop the simulation so the background goroutine exits and releases
				// file handles before t.TempDir cleanup (required on Windows).
				stopReq := httptest.NewRequest(http.MethodPost, "/api/simulation/stop", strings.NewReader(`{"simulation_id":"sim-start"}`))
				stopReq.Header.Set("Content-Type", "application/json")
				stopRec := httptest.NewRecorder()
				serveSimulation(t, gw, stopReq, stopRec)
				waitForSimTerminal(t, filepath.Join(gw.cfg.simulationsDir, "sim-start"))
			},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, payload map[string]any) {
				if payload["success"] != true {
					t.Fatalf("expected success=true, got %#v", payload["success"])
				}
				data := payload["data"].(map[string]any)
				if data["runner_status"] != "running" {
					t.Fatalf("expected running status, got %#v", data["runner_status"])
				}
				if data["max_rounds_applied"].(float64) != 3 {
					t.Fatalf("expected max_rounds_applied 3, got %#v", data["max_rounds_applied"])
				}
				if data["graph_memory_update_enabled"] != true {
					t.Fatalf("expected graph memory update enabled")
				}
				if data["graph_id"] != "graph-9" {
					t.Fatalf("expected graph_id graph-9, got %#v", data["graph_id"])
				}
			},
		},
		{
			name: "stop success returns run state",
			body: `{"simulation_id":"sim-stop"}`,
			invoke: func(gw *gateway, w http.ResponseWriter, r *http.Request) {
				r.URL.Path = "/api/simulation/stop"
				serveSimulation(t, gw, r, w.(*httptest.ResponseRecorder))
			},
			setup: func(t *testing.T, gw *gateway) {
				writeGatewayJSON(t, filepath.Join(gw.cfg.simulationsDir, "sim-stop", "run_state.json"), map[string]any{
					"worker_protocol_version": "1.0",
					"simulation_id":           "sim-stop",
					"runner_status":           "stopped",
				})
			},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, payload map[string]any) {
				if payload["success"] != true {
					t.Fatalf("expected success=true, got %#v", payload["success"])
				}
				data := payload["data"].(map[string]any)
				if data["runner_status"] != "stopped" {
					t.Fatalf("expected stopped status, got %#v", data["runner_status"])
				}
			},
		},
		{
			name: "interview success",
			body: `{"simulation_id":"sim-interview","agent_id":7,"prompt":"hello","platform":"twitter"}`,
			invoke: func(gw *gateway, w http.ResponseWriter, r *http.Request) {
				r.URL.Path = "/api/simulation/interview"
				serveSimulation(t, gw, r, w.(*httptest.ResponseRecorder))
			},
			setup: func(t *testing.T, gw *gateway) {
				simDir := filepath.Join(gw.cfg.simulationsDir, "sim-interview")
				writeGatewayJSON(t, filepath.Join(simDir, "env_status.json"), map[string]any{
					"worker_protocol_version": "1.0",
					"worker_protocol_name":    "go-mirofish-worker",
					"transport_role":          "worker_runtime_state",
					"status":                  "alive",
				})
			},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, payload map[string]any) {
				if payload["success"] != true {
					t.Fatalf("expected success=true, got %#v", payload["success"])
				}
				data := payload["data"].(map[string]any)
				if data["success"] != true {
					t.Fatalf("expected ipc success=true, got %#v", data["success"])
				}
				result := data["result"].(map[string]any)
				if result["response"] == nil && result["response"] == "" {
					t.Fatalf("expected response field in native interview result, got %#v", result)
				}
			},
		},
		{
			name: "interview env not alive returns bad request",
			body: `{"simulation_id":"sim-invalid","agent_id":8,"prompt":"hello"}`,
			invoke: func(gw *gateway, w http.ResponseWriter, r *http.Request) {
				r.URL.Path = "/api/simulation/interview"
				serveSimulation(t, gw, r, w.(*httptest.ResponseRecorder))
			},
			setup: func(t *testing.T, gw *gateway) {
				// No env_status.json → NativeBridge returns env not alive → 400
			},
			wantStatus: http.StatusBadRequest,
			assert: func(t *testing.T, payload map[string]any) {
				if payload["success"] != false {
					t.Fatalf("expected success=false, got %#v", payload["success"])
				}
				if payload["error"] == "" {
					t.Fatalf("expected error message")
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gw := newWorkerGateway(t)
			if tc.setup != nil {
				tc.setup(t, gw)
			}

			req := httptest.NewRequest(http.MethodPost, "/worker-control", strings.NewReader(tc.body))
			req = req.WithContext(context.Background())
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			tc.invoke(gw, rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}

			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			tc.assert(t, payload)

			if tc.cleanup != nil {
				tc.cleanup(t, gw)
			}
		})
	}
}

func TestSimulationRunStatusDetailStillProxiesToBackend(t *testing.T) {
	gw := newTestGateway(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-9/run-status/detail", nil)
	rec := httptest.NewRecorder()

	serveSimulation(t, gw, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReportStatusAliasUsesProgressForReportID(t *testing.T) {
	gw := newGateway(config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		frontendDistDir: "frontend/dist",
		reportsDir:      t.TempDir(),
	})
	reportHandler := buildReportHandler(gw.cfg)

	reportDir := filepath.Join(gw.cfg.reportsDir, "report-42")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("mkdir report dir: %v", err)
	}
	writeGatewayJSON(t, filepath.Join(reportDir, "meta.json"), map[string]any{
		"report_id":        "report-42",
		"simulation_id":    "sim-42",
		"status":           "completed",
		"created_at":       "2026-04-24T00:00:00Z",
		"markdown_content": "# Report",
	})
	writeGatewayJSON(t, filepath.Join(reportDir, "progress.json"), map[string]any{
		"report_id":     "report-42",
		"simulation_id": "sim-42",
		"status":        "completed",
		"progress":      100,
		"message":       "done",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/report/generate/status?report_id=report-42", nil)
	rec := httptest.NewRecorder()

	reportHandler.HandleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReportStatusAliasBridgesQueryToPOSTBody(t *testing.T) {
	gw := newGateway(config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		frontendDistDir: "frontend/dist",
		reportsDir:      t.TempDir(),
	})
	reportHandler := buildReportHandler(gw.cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/report/generate/status?task_id=task-7&simulation_id=sim-7", nil)
	rec := httptest.NewRecorder()

	reportHandler.HandleStatus(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestProjectControlPlaneGetListResetDelete(t *testing.T) {
	tmpDir := t.TempDir()
	projectID := "proj_test123"
	projectDir := filepath.Join(tmpDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	project := map[string]any{
		"project_id":             projectID,
		"name":                   "Example Project",
		"status":                 "graph_completed",
		"created_at":             "2026-04-24T00:00:00Z",
		"updated_at":             "2026-04-24T00:00:00Z",
		"files":                  []any{},
		"total_text_length":      42,
		"ontology":               map[string]any{"entity_types": []any{}, "edge_types": []any{}},
		"analysis_summary":       "ok",
		"graph_id":               "go_mirofish_1",
		"graph_build_task_id":    "task-1",
		"simulation_requirement": "demo",
		"chunk_size":             500,
		"chunk_overlap":          50,
		"error":                  nil,
	}
	raw, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		t.Fatalf("marshal project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), raw, 0o644); err != nil {
		t.Fatalf("write project file: %v", err)
	}

	gw := newGateway(config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		frontendDistDir: "frontend/dist",
		projectsDir:     tmpDir,
	})
	graphHandler := buildGraphHandler(gw.cfg)

	t.Run("get", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/graph/project/"+projectID, nil)
		rec := httptest.NewRecorder()
		graphHandler.HandleProjectRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/graph/project/list?limit=10", nil)
		rec := httptest.NewRecorder()
		graphHandler.HandleProjectList(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("reset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/graph/project/"+projectID+"/reset", nil)
		rec := httptest.NewRecorder()
		graphHandler.HandleProjectRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		updated, err := os.ReadFile(filepath.Join(projectDir, "project.json"))
		if err != nil {
			t.Fatalf("read updated project: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(updated, &payload); err != nil {
			t.Fatalf("decode updated project: %v", err)
		}
		if payload["status"] != "ontology_generated" {
			t.Fatalf("expected ontology_generated, got %#v", payload["status"])
		}
		if payload["graph_id"] != nil {
			t.Fatalf("expected graph_id nil, got %#v", payload["graph_id"])
		}
	})

	t.Run("delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/graph/project/"+projectID, nil)
		rec := httptest.NewRecorder()
		graphHandler.HandleProjectRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
			t.Fatalf("expected project dir removed, stat err=%v", err)
		}
	})
}

func TestTaskControlPlaneGetAndList(t *testing.T) {
	tmpDir := t.TempDir()
	task := map[string]any{
		"task_id":         "task-123",
		"task_type":       "graph_build",
		"status":          "processing",
		"created_at":      "2026-04-24T00:00:00Z",
		"updated_at":      "2026-04-24T00:00:01Z",
		"progress":        55,
		"message":         "Working",
		"progress_detail": map[string]any{},
		"result":          nil,
		"error":           nil,
		"metadata":        map[string]any{"project_id": "proj-1"},
	}
	raw, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "task-123.json"), raw, 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	gw := newGateway(config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		frontendDistDir: "frontend/dist",
		tasksDir:        tmpDir,
	})
	graphHandler := buildGraphHandler(gw.cfg)

	t.Run("get", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/graph/task/task-123", nil)
		rec := httptest.NewRecorder()
		graphHandler.HandleTaskGet(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/graph/tasks?task_type=graph_build", nil)
		rec := httptest.NewRecorder()
		graphHandler.HandleTaskList(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode list response: %v", err)
		}
		if payload["count"].(float64) != 1 {
			t.Fatalf("expected count 1, got %#v", payload["count"])
		}
	})
}

func TestSimulationControlPlaneGetListAndRunStatus(t *testing.T) {
	tmpDir := t.TempDir()
	simulation := map[string]any{
		"simulation_id":    "sim-123",
		"project_id":       "proj-1",
		"graph_id":         "graph-1",
		"status":           "ready",
		"entities_count":   23,
		"profiles_count":   23,
		"entity_types":     []any{"Person", "Organization"},
		"config_generated": true,
		"config_reasoning": "ok",
		"current_round":    0,
		"twitter_status":   "not_started",
		"reddit_status":    "not_started",
		"created_at":       "2026-04-24T00:00:00Z",
		"updated_at":       "2026-04-24T00:00:01Z",
		"error":            nil,
	}
	simDir := filepath.Join(tmpDir, "sim-123")
	if err := os.MkdirAll(simDir, 0o755); err != nil {
		t.Fatalf("mkdir simulation dir: %v", err)
	}
	raw, err := json.MarshalIndent(simulation, "", "  ")
	if err != nil {
		t.Fatalf("marshal simulation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simDir, "state.json"), raw, 0o644); err != nil {
		t.Fatalf("write state file: %v", err)
	}
	runState := map[string]any{
		"worker_protocol_version": "1.0",
		"simulation_id":           "sim-123",
		"runner_status":           "completed",
		"current_round":           3,
		"total_rounds":            3,
		"progress_percent":        100.0,
		"twitter_actions_count":   11,
		"reddit_actions_count":    11,
		"total_actions_count":     22,
	}
	runRaw, err := json.MarshalIndent(runState, "", "  ")
	if err != nil {
		t.Fatalf("marshal run state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simDir, "run_state.json"), runRaw, 0o644); err != nil {
		t.Fatalf("write run_state file: %v", err)
	}
	simConfig := map[string]any{
		"simulation_requirement": "Simulate a test scenario",
		"agent_configs":          []any{map[string]any{"agent_id": 1}},
		"time_config": map[string]any{
			"total_simulation_hours": 72,
			"minutes_per_round":      60,
		},
		"event_config": map[string]any{
			"initial_posts": []any{map[string]any{"content": "seed"}},
			"hot_topics":    []any{"topic-1", "topic-2"},
		},
		"twitter_config": map[string]any{},
		"reddit_config":  map[string]any{},
		"generated_at":   "2026-04-24T00:02:00Z",
		"llm_model":      "gemini-2.5-flash",
	}
	configRaw, err := json.MarshalIndent(simConfig, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simDir, "simulation_config.json"), configRaw, 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	redditProfiles := []map[string]any{{"username": "alice", "bio": "bio"}}
	redditRaw, err := json.MarshalIndent(redditProfiles, "", "  ")
	if err != nil {
		t.Fatalf("marshal reddit profiles: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simDir, "reddit_profiles.json"), redditRaw, 0o644); err != nil {
		t.Fatalf("write reddit profiles: %v", err)
	}
	twitterFile, err := os.Create(filepath.Join(simDir, "twitter_profiles.csv"))
	if err != nil {
		t.Fatalf("create twitter profiles: %v", err)
	}
	writer := csv.NewWriter(twitterFile)
	if err := writer.Write([]string{"username", "bio"}); err != nil {
		t.Fatalf("write twitter header: %v", err)
	}
	if err := writer.Write([]string{"bob", "radio"}); err != nil {
		t.Fatalf("write twitter row: %v", err)
	}
	writer.Flush()
	if err := twitterFile.Close(); err != nil {
		t.Fatalf("close twitter csv: %v", err)
	}

	gw := newGateway(config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		frontendDistDir: "frontend/dist",
		simulationsDir:  tmpDir,
	})

	t.Run("get simulation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-123", nil)
		rec := httptest.NewRecorder()
		serveSimulation(t, gw, req, rec)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode get response: %v", err)
		}
		data := payload["data"].(map[string]any)
		if _, ok := data["run_instructions"].(map[string]any); !ok {
			t.Fatalf("expected run_instructions for ready simulation, got %#v", data["run_instructions"])
		}
	})

	t.Run("list simulations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/simulation/list?project_id=proj-1", nil)
		rec := httptest.NewRecorder()
		serveSimulation(t, gw, req, rec)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode list response: %v", err)
		}
		if payload["count"].(float64) != 1 {
			t.Fatalf("expected count 1, got %#v", payload["count"])
		}
	})

	t.Run("history", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/simulation/history?limit=20", nil)
		rec := httptest.NewRecorder()
		serveSimulation(t, gw, req, rec)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("run status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-123/run-status", nil)
		rec := httptest.NewRecorder()
		serveSimulation(t, gw, req, rec)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode run-status response: %v", err)
		}
		data := payload["data"].(map[string]any)
		if data["runner_status"] != "completed" {
			t.Fatalf("expected completed runner_status, got %#v", data["runner_status"])
		}
	})

	t.Run("config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-123/config", nil)
		rec := httptest.NewRecorder()
		serveSimulation(t, gw, req, rec)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("config realtime", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-123/config/realtime", nil)
		rec := httptest.NewRecorder()
		serveSimulation(t, gw, req, rec)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("profiles realtime reddit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-123/profiles/realtime?platform=reddit", nil)
		rec := httptest.NewRecorder()
		serveSimulation(t, gw, req, rec)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("profiles twitter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-123/profiles?platform=twitter", nil)
		rec := httptest.NewRecorder()
		serveSimulation(t, gw, req, rec)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestValidateStartupAndReadiness(t *testing.T) {
	root := t.TempDir()
	cfg := config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		frontendDistDir: filepath.Join(root, "frontend", "dist"),
		projectsDir:     filepath.Join(root, "projects"),
		reportsDir:      filepath.Join(root, "reports"),
		tasksDir:        filepath.Join(root, "tasks"),
		simulationsDir:  filepath.Join(root, "simulations"),
		scriptsDir:      filepath.Join(root, "scripts"),
	}

	if err := os.MkdirAll(cfg.frontendDistDir, 0o755); err != nil {
		t.Fatalf("mkdir frontend dist: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfg.frontendDistDir, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	if err := validateStartup(cfg); err != nil {
		t.Fatalf("validateStartup returned error: %v", err)
	}

	ready := readinessChecker(cfg)
	payload, err := ready(context.Background())
	if err != nil {
		t.Fatalf("expected readiness success (no backend dependency), got err=%v payload=%#v", err, payload)
	}
	if payload["worker_runtime"] != "native" {
		t.Fatalf("expected worker_runtime=native, got %#v", payload["worker_runtime"])
	}
	if payload["frontend"] != "dist" {
		t.Fatalf("expected frontend=dist, got %#v", payload["frontend"])
	}
}

func TestReportControlPlaneReadEndpointsAndProxyFallback(t *testing.T) {
	tmpDir := t.TempDir()
	reportID := "report-123"
	reportDir := filepath.Join(tmpDir, reportID)
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("mkdir report dir: %v", err)
	}

	report := map[string]any{
		"report_id":              reportID,
		"simulation_id":          "sim-123",
		"graph_id":               "graph-123",
		"simulation_requirement": "Stress test",
		"status":                 "completed",
		"outline": map[string]any{
			"title":   "Future Report",
			"summary": "Summary",
			"sections": []any{
				map[string]any{"title": "Section One", "content": ""},
			},
		},
		"markdown_content": "",
		"created_at":       "2026-04-24T00:00:00Z",
		"completed_at":     "2026-04-24T00:10:00Z",
		"error":            nil,
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "meta.json"), raw, 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "full_report.md"), []byte("# Future Report\n\nBody"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "progress.json"), []byte(`{"status":"generating","progress":45,"updated_at":"2026-04-24T00:05:00Z"}`), 0o644); err != nil {
		t.Fatalf("write progress: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "section_01.md"), []byte("## Section One\n\nContent"), 0o644); err != nil {
		t.Fatalf("write section: %v", err)
	}

	gw := newGateway(config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		frontendDistDir: "frontend/dist",
		reportsDir:      tmpDir,
	})
	reportHandler := buildReportHandler(gw.cfg)

	t.Run("get report", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/"+reportID, nil)
		rec := httptest.NewRecorder()
		reportHandler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		data := payload["data"].(map[string]any)
		if data["markdown_content"] != "# Future Report\n\nBody" {
			t.Fatalf("expected hydrated markdown, got %#v", data["markdown_content"])
		}
	})

	t.Run("list reports", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/list?limit=10", nil)
		rec := httptest.NewRecorder()
		reportHandler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("by simulation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/by-simulation/sim-123", nil)
		rec := httptest.NewRecorder()
		reportHandler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("check status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/check/sim-123", nil)
		rec := httptest.NewRecorder()
		reportHandler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		data := payload["data"].(map[string]any)
		if data["interview_unlocked"] != true {
			t.Fatalf("expected interview_unlocked true, got %#v", data["interview_unlocked"])
		}
	})

	t.Run("progress", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/progress", nil)
		rec := httptest.NewRecorder()
		reportHandler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("sections", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/sections", nil)
		rec := httptest.NewRecorder()
		reportHandler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("single section", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/section/1", nil)
		rec := httptest.NewRecorder()
		reportHandler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("delete handled in go", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/report/"+reportID, nil)
		rec := httptest.NewRecorder()
		reportHandler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}
