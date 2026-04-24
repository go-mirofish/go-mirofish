package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestIPCClientSend(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		setup       func(t *testing.T, simulationDir string)
		wantErrKind error
		assert      func(t *testing.T, result IPCResult)
	}{
		{
			name:    "completed response",
			timeout: 2 * time.Second,
			setup: func(t *testing.T, simulationDir string) {
				startIPCResponder(t, simulationDir, func(t *testing.T, cmd ipcCommand) []byte {
					if cmd.CommandType != "interview" {
						t.Fatalf("expected interview command, got %q", cmd.CommandType)
					}
					raw, err := json.Marshal(ipcResponse{
						CommandID: cmd.CommandID,
						Status:    "completed",
						Result:    map[string]any{"message": "ok"},
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
					t.Fatalf("expected success result")
				}
				payload, ok := result.Result.(map[string]any)
				if !ok {
					t.Fatalf("expected map result, got %T", result.Result)
				}
				if payload["message"] != "ok" {
					t.Fatalf("expected message ok, got %#v", payload["message"])
				}
			},
		},
		{
			name:    "worker reports failure without transport error",
			timeout: 2 * time.Second,
			setup: func(t *testing.T, simulationDir string) {
				startIPCResponder(t, simulationDir, func(t *testing.T, cmd ipcCommand) []byte {
					raw, err := json.Marshal(ipcResponse{
						CommandID: cmd.CommandID,
						Status:    "failed",
						Error:     "agent unavailable",
						Timestamp: "2026-04-24T00:00:00Z",
					})
					if err != nil {
						t.Fatalf("marshal response: %v", err)
					}
					return raw
				})
			},
			assert: func(t *testing.T, result IPCResult) {
				if result.Success {
					t.Fatalf("expected unsuccessful result")
				}
				if result.Error != "agent unavailable" {
					t.Fatalf("expected worker error to be preserved, got %q", result.Error)
				}
			},
		},
		{
			name:        "timeout waiting for response",
			timeout:     50 * time.Millisecond,
			wantErrKind: ErrWorkerTimeout,
		},
		{
			name:    "invalid worker response",
			timeout: 2 * time.Second,
			setup: func(t *testing.T, simulationDir string) {
				startIPCResponder(t, simulationDir, func(t *testing.T, cmd ipcCommand) []byte {
					return []byte("{")
				})
			},
			wantErrKind: ErrWorkerUnavailable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			simulationDir := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, simulationDir)
			}

			result, err := NewIPCClient(simulationDir).Send(
				context.Background(),
				"interview",
				map[string]any{"agent_id": 5, "prompt": "hello"},
				tc.timeout,
			)
			if tc.wantErrKind != nil {
				assertWorkerErrorKind(t, err, tc.wantErrKind)
				return
			}
			if err != nil {
				t.Fatalf("Send returned error: %v", err)
			}
			tc.assert(t, result)
		})
	}
}
