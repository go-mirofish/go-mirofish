package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ipcCommand struct {
	CommandID   string         `json:"command_id"`
	CommandType string         `json:"command_type"`
	Args        map[string]any `json:"args"`
	Timestamp   string         `json:"timestamp"`
}

type ipcResponse struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
	Result    any    `json:"result"`
	Error     string `json:"error"`
	Timestamp string `json:"timestamp"`
}

type IPCClient struct {
	simulationDir string
}

func NewIPCClient(simulationDir string) *IPCClient {
	return &IPCClient{simulationDir: simulationDir}
}

func (c *IPCClient) Send(ctx context.Context, commandType string, args map[string]any, timeout time.Duration) (IPCResult, error) {
	commandID := fmt.Sprintf("%d", time.Now().UnixNano())
	command := ipcCommand{
		CommandID:   commandID,
		CommandType: commandType,
		Args:        args,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	commandsDir := filepath.Join(c.simulationDir, "ipc_commands")
	responsesDir := filepath.Join(c.simulationDir, "ipc_responses")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		return IPCResult{}, &Error{Op: "mkdir commands", Kind: ErrWorkerUnavailable, Err: err}
	}
	if err := os.MkdirAll(responsesDir, 0o755); err != nil {
		return IPCResult{}, &Error{Op: "mkdir responses", Kind: ErrWorkerUnavailable, Err: err}
	}

	commandPath := filepath.Join(commandsDir, commandID+".json")
	responsePath := filepath.Join(responsesDir, commandID+".json")
	raw, err := json.MarshalIndent(command, "", "  ")
	if err != nil {
		return IPCResult{}, &Error{Op: "marshal command", Kind: ErrWorkerBadRequest, Err: err}
	}
	if err := os.WriteFile(commandPath, raw, 0o644); err != nil {
		return IPCResult{}, &Error{Op: "write command", Kind: ErrWorkerUnavailable, Err: err}
	}
	defer os.Remove(commandPath)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return IPCResult{}, &Error{Op: "send", Kind: ErrWorkerTimeout, Err: ctx.Err()}
		default:
		}

		if _, err := os.Stat(responsePath); err == nil {
			rawResp, err := os.ReadFile(responsePath)
			if err != nil {
				return IPCResult{}, &Error{Op: "read response", Kind: ErrWorkerUnavailable, Err: err}
			}
			_ = os.Remove(responsePath)
			var resp ipcResponse
			if err := json.Unmarshal(rawResp, &resp); err != nil {
				return IPCResult{}, &Error{Op: "decode response", Kind: ErrWorkerUnavailable, Err: err}
			}
			return IPCResult{
				Success:   resp.Status == "completed",
				Timestamp: resp.Timestamp,
				Error:     resp.Error,
				Result:    resp.Result,
			}, nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return IPCResult{}, &Error{Op: "poll response", Kind: ErrWorkerTimeout, Detail: "timed out waiting for worker response"}
}
