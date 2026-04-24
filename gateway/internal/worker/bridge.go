package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type SimulationStateReader interface {
	ReadSimulation(simulationID string) (map[string]any, error)
	ReadRunState(simulationID string) (map[string]any, error)
}

type LocalPythonBridge struct {
	SimulationsDir string
	ScriptsDir     string
	PythonPath     string
}

func NewLocalPythonBridge(simulationsDir, scriptsDir, pythonPath string) *LocalPythonBridge {
	return &LocalPythonBridge{
		SimulationsDir: simulationsDir,
		ScriptsDir:     scriptsDir,
		PythonPath:     pythonPath,
	}
}

func (b *LocalPythonBridge) StartSimulation(ctx context.Context, req StartRequest) (StartResponse, error) {
	if req.SimulationID == "" {
		return StartResponse{}, &Error{Op: "StartSimulation", Kind: ErrWorkerBadRequest, Detail: "simulation_id is required"}
	}
	configPath := filepath.Join(b.SimulationsDir, req.SimulationID, "simulation_config.json")
	if _, err := os.Stat(configPath); err != nil {
		return StartResponse{}, &Error{Op: "StartSimulation", Kind: ErrWorkerNotFound, Err: err}
	}
	scriptName := "run_parallel_simulation.py"
	switch req.Platform {
	case PlatformTwitter:
		scriptName = "run_twitter_simulation.py"
	case PlatformReddit:
		scriptName = "run_reddit_simulation.py"
	}
	scriptPath := filepath.Join(b.ScriptsDir, scriptName)
	args := []string{scriptPath, "--config", configPath}
	if req.MaxRounds > 0 {
		args = append(args, "--max-rounds", fmt.Sprintf("%d", req.MaxRounds))
	}
	cmd := exec.CommandContext(context.Background(), b.PythonPath, args...)
	cmd.Dir = filepath.Dir(configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = procAttrForChild()
	if err := cmd.Start(); err != nil {
		return StartResponse{}, &Error{Op: "StartSimulation", Kind: ErrWorkerUnavailable, Err: err}
	}

	runState, _ := b.waitForRunState(req.SimulationID, 10*time.Second)
	resp := StartResponse{
		SimulationID: req.SimulationID,
		RunnerStatus: "running",
		ProcessPID:   cmd.Process.Pid,
		StartedAt:    time.Now().Format(time.RFC3339),
	}
	if req.MaxRounds > 0 {
		resp.MaxRoundsApplied = req.MaxRounds
	}
	resp.GraphMemoryUpdateEnabled = req.EnableGraphMemoryUpdate
	resp.GraphID = req.GraphID
	if runState != nil {
		if status, _ := runState["runner_status"].(string); status != "" {
			resp.RunnerStatus = status
		}
		if started, _ := runState["started_at"].(string); started != "" {
			resp.StartedAt = started
		}
		if pid := intValue(runState["process_pid"]); pid > 0 {
			resp.ProcessPID = pid
		}
	}
	return resp, nil
}

func (b *LocalPythonBridge) StopSimulation(ctx context.Context, req StopRequest) (map[string]any, error) {
	runState, err := b.readRunState(req.SimulationID)
	if err != nil {
		return nil, &Error{Op: "StopSimulation", Kind: ErrWorkerNotFound, Err: err}
	}
	pid := intValue(runState["process_pid"])
	if pid <= 0 {
		return runState, nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, &Error{Op: "StopSimulation", Kind: ErrWorkerUnavailable, Err: err}
	}
	_ = proc.Signal(syscall.SIGTERM)
	_ = ctx
	time.Sleep(2 * time.Second)
	return b.readRunState(req.SimulationID)
}

func (b *LocalPythonBridge) Interview(ctx context.Context, req InterviewRequest) (IPCResult, error) {
	if !b.envAlive(req.SimulationID) {
		return IPCResult{}, &Error{Op: "Interview", Kind: ErrWorkerNotReady, Detail: "simulation environment is not running"}
	}
	return NewIPCClient(filepath.Join(b.SimulationsDir, req.SimulationID)).Send(ctx, "interview", map[string]any{
		"agent_id": req.AgentID,
		"prompt":   req.Prompt,
		"platform": req.Platform,
	}, timeoutOrDefault(req.Timeout, 60))
}

func (b *LocalPythonBridge) BatchInterview(ctx context.Context, req BatchInterviewRequest) (IPCResult, error) {
	if !b.envAlive(req.SimulationID) {
		return IPCResult{}, &Error{Op: "BatchInterview", Kind: ErrWorkerNotReady, Detail: "simulation environment is not running"}
	}
	items := make([]map[string]any, 0, len(req.Interviews))
	for _, item := range req.Interviews {
		items = append(items, map[string]any{"agent_id": item.AgentID, "prompt": item.Prompt, "platform": item.Platform})
	}
	return NewIPCClient(filepath.Join(b.SimulationsDir, req.SimulationID)).Send(ctx, "batch_interview", map[string]any{
		"interviews": items,
		"platform":   req.Platform,
	}, timeoutOrDefault(req.Timeout, 120))
}

func (b *LocalPythonBridge) InterviewAll(ctx context.Context, req AllInterviewRequest) (IPCResult, error) {
	state, err := b.readSimulationState(req.SimulationID)
	if err != nil {
		return IPCResult{}, &Error{Op: "InterviewAll", Kind: ErrWorkerNotFound, Err: err}
	}
	configPath := filepath.Join(b.SimulationsDir, req.SimulationID, "simulation_config.json")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return IPCResult{}, &Error{Op: "InterviewAll", Kind: ErrWorkerUnavailable, Err: err}
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return IPCResult{}, &Error{Op: "InterviewAll", Kind: ErrWorkerUnavailable, Err: err}
	}
	agentConfigs, _ := cfg["agent_configs"].([]any)
	items := make([]BatchInterviewItem, 0, len(agentConfigs))
	for _, rawItem := range agentConfigs {
		if item, ok := rawItem.(map[string]any); ok {
			items = append(items, BatchInterviewItem{
				AgentID:  intValue(item["agent_id"]),
				Prompt:   req.Prompt,
				Platform: req.Platform,
			})
		}
	}
	_ = state
	return b.BatchInterview(ctx, BatchInterviewRequest{
		SimulationID: req.SimulationID,
		Interviews:   items,
		Platform:     req.Platform,
		Timeout:      req.Timeout,
	})
}

func (b *LocalPythonBridge) EnvStatus(ctx context.Context, req EnvStatusRequest) (EnvStatus, error) {
	_ = ctx
	envStatusPath := filepath.Join(b.SimulationsDir, req.SimulationID, "env_status.json")
	raw, err := os.ReadFile(envStatusPath)
	if err != nil {
		return EnvStatus{}, &Error{Op: "EnvStatus", Kind: ErrWorkerNotFound, Err: err}
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return EnvStatus{}, &Error{Op: "EnvStatus", Kind: ErrWorkerUnavailable, Err: err}
	}
	alive := payload["status"] == "alive"
	return EnvStatus{
		SimulationID:     req.SimulationID,
		EnvAlive:         alive,
		TwitterAvailable: boolValue(payload["twitter_available"]),
		RedditAvailable:  boolValue(payload["reddit_available"]),
		Message:          ternary(alive, "Environment is running", "Environment is not running"),
	}, nil
}

func (b *LocalPythonBridge) CloseEnv(ctx context.Context, req CloseEnvRequest) (IPCResult, error) {
	return NewIPCClient(filepath.Join(b.SimulationsDir, req.SimulationID)).Send(ctx, "close_env", map[string]any{}, timeoutOrDefault(req.Timeout, 30))
}

func (b *LocalPythonBridge) readSimulationState(simulationID string) (map[string]any, error) {
	raw, err := os.ReadFile(filepath.Join(b.SimulationsDir, simulationID, "state.json"))
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (b *LocalPythonBridge) readRunState(simulationID string) (map[string]any, error) {
	raw, err := os.ReadFile(filepath.Join(b.SimulationsDir, simulationID, "run_state.json"))
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (b *LocalPythonBridge) waitForRunState(simulationID string, timeout time.Duration) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if state, err := b.readRunState(simulationID); err == nil {
			return state, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil, &Error{Op: "waitForRunState", Kind: ErrWorkerTimeout, Detail: "timed out waiting for run_state.json"}
}

func (b *LocalPythonBridge) envAlive(simulationID string) bool {
	status, err := b.EnvStatus(context.Background(), EnvStatusRequest{SimulationID: simulationID})
	return err == nil && status.EnvAlive
}

func timeoutOrDefault(seconds int, fallback int) time.Duration {
	if seconds <= 0 {
		seconds = fallback
	}
	return time.Duration(seconds) * time.Second
}

func ternary(cond bool, ifTrue, ifFalse string) string {
	if cond {
		return ifTrue
	}
	return ifFalse
}

func intValue(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return int(parsed)
		}
		if parsed, err := v.Float64(); err == nil {
			return int(parsed)
		}
	}
	return 0
}

func boolValue(value any) bool {
	got, _ := value.(bool)
	return got
}
