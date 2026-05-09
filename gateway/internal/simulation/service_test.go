package simulation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	intgovernor "github.com/go-mirofish/go-mirofish/gateway/internal/governor"
	simulationstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/simulation"
	sovereignstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/sovereign"
	intworker "github.com/go-mirofish/go-mirofish/gateway/internal/worker"
)

func TestNativeActionRoundParity(t *testing.T) {
	got := toAction(map[string]any{
		"round_num":   3,
		"timestamp":   "2026-01-01T00:00:00Z",
		"platform":    "twitter",
		"agent_id":    7,
		"agent_name":  "Agent 7",
		"action_type": "POST_TWEET",
		"action_args": map[string]any{"content": "hello"},
		"success":     true,
	}, "")
	if got.RoundNum != 3 {
		t.Fatalf("expected round_num parity for native artifacts, got %d", got.RoundNum)
	}
}

func TestNormalizeRunStatusIncludesPlatformProgress(t *testing.T) {
	got := NormalizeRunStatus("sim-1", map[string]any{
		"simulation_id":          "sim-1",
		"runner_status":          "running",
		"twitter_current_round":  2,
		"reddit_current_round":   1,
		"twitter_running":        true,
		"reddit_running":         false,
		"twitter_completed":      false,
		"reddit_completed":       true,
		"twitter_actions_count":  4,
		"reddit_actions_count":   3,
		"total_actions_count":    7,
		"progress_percent":       50,
		"total_simulation_hours": 4,
	})
	if got["twitter_current_round"] != 2 {
		t.Fatalf("expected twitter_current_round to survive normalization, got %#v", got["twitter_current_round"])
	}
	if got["reddit_completed"] != true {
		t.Fatalf("expected reddit_completed to survive normalization, got %#v", got["reddit_completed"])
	}
}

func TestStartFailureAndStopSyncSovereignState(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "projects", "proj-1")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll project: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "simulations"), 0o755); err != nil {
		t.Fatalf("MkdirAll simulations: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), []byte(`{"project_id":"proj-1","graph_id":"graph-1"}`), 0o644); err != nil {
		t.Fatalf("WriteFile project.json: %v", err)
	}

	store := simulationstore.New(
		filepath.Join(root, "simulations"),
		filepath.Join(root, "scripts"),
		filepath.Join(root, "projects"),
		filepath.Join(root, "reports"),
	)
	governor := intgovernor.NewService(sovereignstore.New(filepath.Join(root, "simulations", "sovereign.db")), intgovernor.DefaultProfile)

	failing := NewServiceWithGovernor(store, stubBridge{startErr: context.DeadlineExceeded}, governor)
	created, err := failing.Create(CreateRequest{ProjectID: "proj-1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	simulationID := created["simulation_id"].(string)

	if _, err := failing.Start(context.Background(), intworker.StartRequest{
		SimulationID: simulationID,
		Platform:     intworker.PlatformParallel,
	}); err == nil {
		t.Fatal("expected start error")
	}
	control, err := store.ReadState(simulationID)
	if err != nil {
		t.Fatalf("ReadState after start failure: %v", err)
	}
	sovereign, _ := control["sovereign"].(map[string]any)
	if sovereign["status"] != intgovernor.StatusFailed {
		t.Fatalf("expected failed sovereign status, got %#v", sovereign["status"])
	}

	successful := NewServiceWithGovernor(store, stubBridge{stopResp: map[string]any{"simulation_id": "sim-stop"}}, governor)
	createdStop, err := successful.Create(CreateRequest{ProjectID: "proj-1"})
	if err != nil {
		t.Fatalf("Create for stop case: %v", err)
	}
	stopSimulationID := createdStop["simulation_id"].(string)
	if _, err := governor.SetStatus(context.Background(), stopSimulationID, []string{intgovernor.StatusReady}, intgovernor.StatusRunning, ""); err != nil {
		t.Fatalf("SetStatus running: %v", err)
	}
	if _, err := successful.Stop(context.Background(), intworker.StopRequest{SimulationID: stopSimulationID}); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	control, err = store.ReadState(stopSimulationID)
	if err != nil {
		t.Fatalf("ReadState after stop: %v", err)
	}
	sovereign, _ = control["sovereign"].(map[string]any)
	if sovereign["status"] != intgovernor.StatusStopped {
		t.Fatalf("expected stopped sovereign status, got %#v", sovereign["status"])
	}

	restartable := NewServiceWithGovernor(store, stubBridge{startResp: intworker.StartResponse{SimulationID: stopSimulationID}}, governor)
	if _, err := restartable.Start(context.Background(), intworker.StartRequest{
		SimulationID: stopSimulationID,
		Platform:     intworker.PlatformParallel,
	}); err != nil {
		t.Fatalf("Restart Start: %v", err)
	}
	control, err = store.ReadState(stopSimulationID)
	if err != nil {
		t.Fatalf("ReadState after restart: %v", err)
	}
	sovereign, _ = control["sovereign"].(map[string]any)
	if sovereign["status"] != intgovernor.StatusRunning {
		t.Fatalf("expected running sovereign status after restart, got %#v", sovereign["status"])
	}
}

func TestReconcileSovereignRuntimeFromNativeCompletion(t *testing.T) {
	root := t.TempDir()
	simDir := filepath.Join(root, "simulations", "sim-3")
	projectDir := filepath.Join(root, "projects", "proj-1")
	if err := os.MkdirAll(simDir, 0o755); err != nil {
		t.Fatalf("MkdirAll sim: %v", err)
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), []byte(`{"project_id":"proj-1","graph_id":"graph-1"}`), 0o644); err != nil {
		t.Fatalf("WriteFile project.json: %v", err)
	}

	store := simulationstore.New(
		filepath.Join(root, "simulations"),
		filepath.Join(root, "scripts"),
		filepath.Join(root, "projects"),
		filepath.Join(root, "reports"),
	)
	governor := intgovernor.NewService(sovereignstore.New(filepath.Join(root, "simulations", "sovereign.db")), intgovernor.DefaultProfile)
	service := NewServiceWithGovernor(store, nil, governor)
	created, err := service.Create(CreateRequest{ProjectID: "proj-1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	simulationID := created["simulation_id"].(string)
	if err := os.WriteFile(filepath.Join(store.SimulationDir(simulationID), "state.json"), []byte(`{"simulation_id":"`+simulationID+`","status":"completed"}`), 0o644); err != nil {
		t.Fatalf("WriteFile state.json: %v", err)
	}
	reconciled, err := service.ReconcileSovereignRuntime(context.Background(), simulationID)
	if err != nil {
		t.Fatalf("ReconcileSovereignRuntime: %v", err)
	}
	if reconciled["status"] != intgovernor.StatusCompleted {
		t.Fatalf("expected sovereign completed status, got %#v", reconciled["status"])
	}
	control, err := store.ReadState(simulationID)
	if err != nil {
		t.Fatalf("ReadState after reconcile: %v", err)
	}
	sovereignState, _ := control["sovereign"].(map[string]any)
	if sovereignState["status"] != intgovernor.StatusCompleted {
		t.Fatalf("expected sovereign completed after reconcile, got %#v", sovereignState["status"])
	}
}

func TestAdvanceSovereignTickRejectsRuntimeStateOnlyRunner(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "projects", "proj-1")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), []byte(`{"project_id":"proj-1","graph_id":"graph-1"}`), 0o644); err != nil {
		t.Fatalf("WriteFile project.json: %v", err)
	}

	store := simulationstore.New(
		filepath.Join(root, "simulations"),
		filepath.Join(root, "scripts"),
		filepath.Join(root, "projects"),
		filepath.Join(root, "reports"),
	)
	governor := intgovernor.NewService(sovereignstore.New(filepath.Join(root, "simulations", "sovereign.db")), intgovernor.DefaultProfile)
	service := NewServiceWithGovernor(store, nil, governor)
	created, err := service.Create(CreateRequest{ProjectID: "proj-1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	simulationID := created["simulation_id"].(string)
	if err := os.WriteFile(filepath.Join(store.SimulationDir(simulationID), "state.json"), []byte(`{"simulation_id":"`+simulationID+`","status":"running"}`), 0o644); err != nil {
		t.Fatalf("WriteFile state.json: %v", err)
	}
	if _, err := service.AdvanceSovereignTick(context.Background(), simulationID); err == nil {
		t.Fatal("expected sovereign tick rejection when runtime state shows active runner")
	}
}

func TestStartLazilyInitializesMissingSovereignRuntime(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "projects", "proj-1")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), []byte(`{"project_id":"proj-1","graph_id":"graph-1"}`), 0o644); err != nil {
		t.Fatalf("WriteFile project.json: %v", err)
	}

	store := simulationstore.New(
		filepath.Join(root, "simulations"),
		filepath.Join(root, "scripts"),
		filepath.Join(root, "projects"),
		filepath.Join(root, "reports"),
	)
	legacyState := map[string]any{
		"simulation_id": "legacy-sim",
		"project_id":    "proj-1",
		"graph_id":      "graph-1",
		"status":        "ready",
	}
	if err := store.WriteState("legacy-sim", legacyState); err != nil {
		t.Fatalf("WriteState legacy-sim: %v", err)
	}

	governor := intgovernor.NewService(sovereignstore.New(filepath.Join(root, "simulations", "sovereign.db")), intgovernor.DefaultProfile)
	service := NewServiceWithGovernor(store, stubBridge{startResp: intworker.StartResponse{SimulationID: "legacy-sim"}}, governor)
	if _, err := service.Start(context.Background(), intworker.StartRequest{
		SimulationID: "legacy-sim",
		Platform:     intworker.PlatformParallel,
	}); err != nil {
		t.Fatalf("Start legacy-sim: %v", err)
	}
	control, err := store.ReadState("legacy-sim")
	if err != nil {
		t.Fatalf("ReadState legacy-sim: %v", err)
	}
	sovereign, _ := control["sovereign"].(map[string]any)
	if sovereign["status"] != intgovernor.StatusRunning {
		t.Fatalf("expected lazily initialized sovereign status running, got %#v", sovereign["status"])
	}
}

func TestStopBeforeStartDoesNotFailReadySimulation(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "projects", "proj-1")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), []byte(`{"project_id":"proj-1","graph_id":"graph-1"}`), 0o644); err != nil {
		t.Fatalf("WriteFile project.json: %v", err)
	}

	store := simulationstore.New(
		filepath.Join(root, "simulations"),
		filepath.Join(root, "scripts"),
		filepath.Join(root, "projects"),
		filepath.Join(root, "reports"),
	)
	governor := intgovernor.NewService(sovereignstore.New(filepath.Join(root, "simulations", "sovereign.db")), intgovernor.DefaultProfile)
	service := NewServiceWithGovernor(store, stubBridge{}, governor)
	created, err := service.Create(CreateRequest{ProjectID: "proj-1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	simulationID := created["simulation_id"].(string)
	if _, err := service.Stop(context.Background(), intworker.StopRequest{SimulationID: simulationID}); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	control, err := store.ReadState(simulationID)
	if err != nil {
		t.Fatalf("ReadState: %v", err)
	}
	sovereign, _ := control["sovereign"].(map[string]any)
	if sovereign["status"] != intgovernor.StatusStopped {
		t.Fatalf("expected stopped sovereign status, got %#v", sovereign["status"])
	}
}

func TestObservedSovereignRuntimeForLegacySimulationWithoutRow(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "projects", "proj-1")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), []byte(`{"project_id":"proj-1","graph_id":"graph-1"}`), 0o644); err != nil {
		t.Fatalf("WriteFile project.json: %v", err)
	}

	store := simulationstore.New(
		filepath.Join(root, "simulations"),
		filepath.Join(root, "scripts"),
		filepath.Join(root, "projects"),
		filepath.Join(root, "reports"),
	)
	legacyState := map[string]any{
		"simulation_id": "legacy-observed",
		"project_id":    "proj-1",
		"graph_id":      "graph-1",
		"status":        "ready",
		"current_round": 0,
		"created_at":    "2026-05-08T00:00:00Z",
		"updated_at":    "2026-05-08T00:00:00Z",
	}
	if err := store.WriteState("legacy-observed", legacyState); err != nil {
		t.Fatalf("WriteState legacy-observed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.SimulationDir("legacy-observed"), "run_state.json"), []byte(`{"worker_protocol_version":"1.0","simulation_id":"legacy-observed","runner_status":"running","current_round":3,"total_rounds":5}`), 0o644); err != nil {
		t.Fatalf("WriteFile run_state.json: %v", err)
	}

	governor := intgovernor.NewService(sovereignstore.New(filepath.Join(root, "simulations", "sovereign.db")), intgovernor.DefaultProfile)
	service := NewServiceWithGovernor(store, nil, governor)
	observed, err := service.ObservedSovereignRuntime(context.Background(), "legacy-observed")
	if err != nil {
		t.Fatalf("ObservedSovereignRuntime: %v", err)
	}
	if observed["status"] != intgovernor.StatusRunning {
		t.Fatalf("expected observed running status, got %#v", observed["status"])
	}
	if observed["current_tick"] != 3 {
		t.Fatalf("expected observed tick 3, got %#v", observed["current_tick"])
	}
}

type stubBridge struct {
	startErr  error
	startResp intworker.StartResponse
	stopResp  map[string]any
}

func (s stubBridge) StartSimulation(context.Context, intworker.StartRequest) (intworker.StartResponse, error) {
	return s.startResp, s.startErr
}

func (s stubBridge) StopSimulation(context.Context, intworker.StopRequest) (map[string]any, error) {
	return s.stopResp, nil
}

func (s stubBridge) Interview(context.Context, intworker.InterviewRequest) (intworker.IPCResult, error) {
	return intworker.IPCResult{}, nil
}

func (s stubBridge) BatchInterview(context.Context, intworker.BatchInterviewRequest) (intworker.IPCResult, error) {
	return intworker.IPCResult{}, nil
}

func (s stubBridge) InterviewAll(context.Context, intworker.AllInterviewRequest) (intworker.IPCResult, error) {
	return intworker.IPCResult{}, nil
}

func (s stubBridge) EnvStatus(context.Context, intworker.EnvStatusRequest) (intworker.EnvStatus, error) {
	return intworker.EnvStatus{}, nil
}

func (s stubBridge) CloseEnv(context.Context, intworker.CloseEnvRequest) (intworker.IPCResult, error) {
	return intworker.IPCResult{}, nil
}
