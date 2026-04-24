package worker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func posixShellForTest(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"sh", "bash"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	if runtime.GOOS != "windows" {
		return "/bin/sh"
	}
	t.Skip("need sh or bash in PATH (e.g. Git for Windows) for LocalPythonBridge tests")
	panic("unreachable")
}

func assertWorkerErrorKind(t *testing.T, err error, want error) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error kind %v, got nil", want)
	}

	var workerErr *Error
	if !errors.As(err, &workerErr) {
		t.Fatalf("expected worker error, got %T: %v", err, err)
	}
	if workerErr.Kind != want {
		t.Fatalf("expected worker error kind %v, got %v", want, workerErr.Kind)
	}
}

func writeTestJSON(t *testing.T, path string, payload any) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal json for %s: %v", path, err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeTestFile(t *testing.T, path string, content string, mode os.FileMode) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func waitForCommand(t *testing.T, simulationDir string) ipcCommand {
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
				var cmd ipcCommand
				if err := json.Unmarshal(raw, &cmd); err != nil {
					t.Fatalf("decode command file: %v", err)
				}
				return cmd
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for ipc command in %s", commandsDir)
	return ipcCommand{}
}

func startIPCResponder(t *testing.T, simulationDir string, responder func(t *testing.T, cmd ipcCommand) []byte) {
	t.Helper()

	go func() {
		cmd := waitForCommand(t, simulationDir)
		responsePath := filepath.Join(simulationDir, "ipc_responses", cmd.CommandID+".json")
		if err := os.WriteFile(responsePath, responder(t, cmd), 0o644); err != nil {
			t.Errorf("write response file: %v", err)
		}
	}()
}

func TestLocalPythonBridgeStartSimulation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		simID := "sim-start"
		bridge := NewLocalPythonBridge(tmpDir, filepath.Join(tmpDir, "scripts"), posixShellForTest(t))

		writeTestJSON(t, filepath.Join(tmpDir, simID, "simulation_config.json"), map[string]any{
			"time_config": map[string]any{"total_simulation_hours": 24},
		})
		writeTestFile(t, filepath.Join(bridge.ScriptsDir, "run_parallel_simulation.py"), `#!/bin/sh
CONFIG=""
while [ $# -gt 0 ]; do
  case "$1" in
    --config)
      CONFIG="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
SIMDIR=$(dirname "$CONFIG")
cat > "$SIMDIR/run_state.json" <<'JSON'
{"simulation_id":"sim-start","runner_status":"running","process_pid":4242,"started_at":"2026-04-24T00:00:00Z"}
JSON
sleep 1
`, 0o755)

		resp, err := bridge.StartSimulation(context.Background(), StartRequest{
			SimulationID:            simID,
			Platform:                PlatformParallel,
			MaxRounds:               5,
			EnableGraphMemoryUpdate: true,
			GraphID:                 "graph-1",
		})
		if err != nil {
			t.Fatalf("StartSimulation returned error: %v", err)
		}
		if resp.SimulationID != simID {
			t.Fatalf("expected simulation id %q, got %q", simID, resp.SimulationID)
		}
		if resp.RunnerStatus != "running" {
			t.Fatalf("expected running status, got %q", resp.RunnerStatus)
		}
		if resp.ProcessPID != 4242 {
			t.Fatalf("expected pid 4242, got %d", resp.ProcessPID)
		}
		if resp.StartedAt != "2026-04-24T00:00:00Z" {
			t.Fatalf("expected started_at from run_state, got %q", resp.StartedAt)
		}
		if resp.MaxRoundsApplied != 5 {
			t.Fatalf("expected max_rounds_applied 5, got %d", resp.MaxRoundsApplied)
		}
		if !resp.GraphMemoryUpdateEnabled {
			t.Fatalf("expected graph memory update to stay enabled")
		}
		if resp.GraphID != "graph-1" {
			t.Fatalf("expected graph_id graph-1, got %q", resp.GraphID)
		}
	})

	tests := []struct {
		name        string
		req         StartRequest
		setup       func(t *testing.T, bridge *LocalPythonBridge)
		wantErrKind error
	}{
		{
			name:        "missing simulation id",
			req:         StartRequest{},
			wantErrKind: ErrWorkerBadRequest,
		},
		{
			name: "missing config",
			req: StartRequest{
				SimulationID: "missing-config",
				Platform:     PlatformParallel,
			},
			wantErrKind: ErrWorkerNotFound,
		},
		{
			name: "worker unavailable",
			req: StartRequest{
				SimulationID: "sim-unavailable",
				Platform:     PlatformParallel,
			},
			setup: func(t *testing.T, bridge *LocalPythonBridge) {
				writeTestJSON(t, filepath.Join(bridge.SimulationsDir, "sim-unavailable", "simulation_config.json"), map[string]any{
					"time_config": map[string]any{"total_simulation_hours": 24},
				})
				bridge.PythonPath = filepath.Join(bridge.ScriptsDir, "missing-python")
			},
			wantErrKind: ErrWorkerUnavailable,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			bridge := NewLocalPythonBridge(tmpDir, filepath.Join(tmpDir, "scripts"), posixShellForTest(t))
			if tc.setup != nil {
				tc.setup(t, bridge)
			}

			_, err := bridge.StartSimulation(context.Background(), tc.req)
			assertWorkerErrorKind(t, err, tc.wantErrKind)
		})
	}
}

func TestLocalPythonBridgeStopSimulation(t *testing.T) {
	t.Run("returns state without pid", func(t *testing.T) {
		tmpDir := t.TempDir()
		bridge := NewLocalPythonBridge(tmpDir, filepath.Join(tmpDir, "scripts"), posixShellForTest(t))
		writeTestJSON(t, filepath.Join(tmpDir, "sim-stop", "run_state.json"), map[string]any{
			"simulation_id": "sim-stop",
			"runner_status": "stopped",
		})

		resp, err := bridge.StopSimulation(context.Background(), StopRequest{SimulationID: "sim-stop"})
		if err != nil {
			t.Fatalf("StopSimulation returned error: %v", err)
		}
		if got := resp["runner_status"]; got != "stopped" {
			t.Fatalf("expected runner_status stopped, got %#v", got)
		}
	})

	t.Run("signals running process", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("signal-based process test is unix-only")
		}

		tmpDir := t.TempDir()
		shell := posixShellForTest(t)
		bridge := NewLocalPythonBridge(tmpDir, filepath.Join(tmpDir, "scripts"), shell)
		cmd := exec.Command(shell, "-c", "trap 'exit 0' TERM; while true; do sleep 1; done")
		if err := cmd.Start(); err != nil {
			t.Fatalf("start test process: %v", err)
		}
		t.Cleanup(func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			_ = cmd.Wait()
		})

		writeTestJSON(t, filepath.Join(tmpDir, "sim-running", "run_state.json"), map[string]any{
			"simulation_id": "sim-running",
			"runner_status": "running",
			"process_pid":   cmd.Process.Pid,
		})

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		resp, err := bridge.StopSimulation(context.Background(), StopRequest{SimulationID: "sim-running"})
		if err != nil {
			t.Fatalf("StopSimulation returned error: %v", err)
		}
		if got := intValue(resp["process_pid"]); got != cmd.Process.Pid {
			t.Fatalf("expected returned pid %d, got %d", cmd.Process.Pid, got)
		}

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("expected process %d to exit after SIGTERM", cmd.Process.Pid)
		}
	})
}

func TestLocalPythonBridgeInterview(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, simulationDir string)
		wantErrKind error
		assert      func(t *testing.T, result IPCResult)
	}{
		{
			name: "success",
			setup: func(t *testing.T, simulationDir string) {
				writeTestJSON(t, filepath.Join(simulationDir, "env_status.json"), map[string]any{
					"status":            "alive",
					"twitter_available": true,
					"reddit_available":  true,
				})
				startIPCResponder(t, simulationDir, func(t *testing.T, cmd ipcCommand) []byte {
					if cmd.CommandType != "interview" {
						t.Fatalf("expected interview command, got %q", cmd.CommandType)
					}
					if got := intValue(cmd.Args["agent_id"]); got != 7 {
						t.Fatalf("expected agent_id 7, got %d", got)
					}
					raw, err := json.Marshal(ipcResponse{
						CommandID: cmd.CommandID,
						Status:    "completed",
						Result:    map[string]any{"answer": "hello"},
						Timestamp: "2026-04-24T00:00:00Z",
					})
					if err != nil {
						t.Fatalf("marshal response: %v", err)
					}
					return raw
				})
			},
			assert: func(t *testing.T, result IPCResult) {
				if !result.Success {
					t.Fatalf("expected interview success")
				}
				payload, ok := result.Result.(map[string]any)
				if !ok {
					t.Fatalf("expected map result, got %T", result.Result)
				}
				if payload["answer"] != "hello" {
					t.Fatalf("expected answer hello, got %#v", payload["answer"])
				}
			},
		},
		{
			name: "environment not ready",
			setup: func(t *testing.T, simulationDir string) {
				writeTestJSON(t, filepath.Join(simulationDir, "env_status.json"), map[string]any{
					"status": "stopped",
				})
			},
			wantErrKind: ErrWorkerNotReady,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			simDir := filepath.Join(tmpDir, "sim-interview")
			bridge := NewLocalPythonBridge(tmpDir, filepath.Join(tmpDir, "scripts"), posixShellForTest(t))
			tc.setup(t, simDir)

			result, err := bridge.Interview(context.Background(), InterviewRequest{
				SimulationID: "sim-interview",
				AgentID:      7,
				Prompt:       "hello?",
				Platform:     PlatformParallel,
				Timeout:      1,
			})
			if tc.wantErrKind != nil {
				assertWorkerErrorKind(t, err, tc.wantErrKind)
				return
			}
			if err != nil {
				t.Fatalf("Interview returned error: %v", err)
			}
			tc.assert(t, result)
		})
	}
}
