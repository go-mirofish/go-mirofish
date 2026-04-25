package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/internal/artifactcontract"
	"github.com/go-mirofish/go-mirofish/gateway/internal/telemetry"
)

type ipcCommand = artifactcontract.IPCCommand
type ipcResponse = artifactcontract.IPCResponse

type IPCClient struct {
	simulationDir string
}

func NewIPCClient(simulationDir string) *IPCClient {
	return &IPCClient{simulationDir: simulationDir}
}

func (c *IPCClient) Send(ctx context.Context, commandType string, args map[string]any, timeout time.Duration) (IPCResult, error) {
	commandID := fmt.Sprintf("%d", time.Now().UnixNano())
	command := artifactcontract.IPCCommand{
		WorkerProtocolVersion: ProtocolVersion,
		WorkerProtocolName:    ProtocolName,
		TransportRole:         ProtocolRoleCommand,
		CommandID:             commandID,
		CommandType:           commandType,
		Args:                  args,
		Timestamp:             time.Now().Format(time.RFC3339),
	}

	commandsDir := filepath.Join(c.simulationDir, "ipc_commands")
	responsesDir := filepath.Join(c.simulationDir, "ipc_responses")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		return IPCResult{}, workerError("mkdir commands", ErrWorkerUnavailable, "", err)
	}
	if err := os.MkdirAll(responsesDir, 0o755); err != nil {
		return IPCResult{}, workerError("mkdir responses", ErrWorkerUnavailable, "", err)
	}

	commandPath := filepath.Join(commandsDir, commandID+".json")
	responsePath := filepath.Join(responsesDir, commandID+".json")
	raw, err := artifactcontract.WriteIPCCommandJSON(command)
	if err != nil {
		return IPCResult{}, workerError("marshal command", ErrWorkerBadRequest, "", err)
	}
	if err := os.WriteFile(commandPath, raw, 0o644); err != nil {
		return IPCResult{}, workerError("write command", ErrWorkerUnavailable, "", err)
	}
	defer os.Remove(commandPath)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return IPCResult{}, workerError("send", ErrWorkerTimeout, "", ctx.Err())
		default:
		}

		if _, err := os.Stat(responsePath); err == nil {
			rawResp, err := os.ReadFile(responsePath)
			if err != nil {
				return IPCResult{}, workerError("read response", ErrWorkerUnavailable, "", err)
			}
			_ = os.Remove(responsePath)
			resp, err := artifactcontract.ReadIPCResponseJSON(rawResp)
			if err != nil {
				return IPCResult{}, workerError("decode response", ErrWorkerUnavailable, "", err)
			}
			if err := validateProtocolEnvelope(resp.WorkerProtocolVersion, resp.WorkerProtocolName, resp.TransportRole, ProtocolRoleResponse); err != nil {
				return IPCResult{}, err
			}
			return IPCResult{
				WorkerProtocolVersion: resp.WorkerProtocolVersion,
				Success:               resp.Status == "completed",
				Timestamp:             resp.Timestamp,
				Error:                 derefString(resp.Error),
				Result:                resp.Result,
			}, nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return IPCResult{}, workerError("poll response", ErrWorkerTimeout, "timed out waiting for worker response", nil)
}

func validateProtocolEnvelope(version, name, role, wantRole string) error {
	if version != ProtocolVersion {
		return workerError("validate protocol", ErrWorkerIncompatible, fmt.Sprintf("worker protocol version mismatch: got %q want %q", version, ProtocolVersion), nil)
	}
	if name != "" && name != ProtocolName {
		return workerError("validate protocol", ErrWorkerIncompatible, fmt.Sprintf("worker protocol name mismatch: got %q want %q", name, ProtocolName), nil)
	}
	if role != wantRole {
		return workerError("validate protocol", ErrWorkerIncompatible, fmt.Sprintf("worker transport role mismatch: got %q want %q", role, wantRole), nil)
	}
	return nil
}

func workerError(op string, kind error, detail string, err error) error {
	reason := detail
	if reason == "" {
		if err != nil {
			reason = err.Error()
		} else if kind != nil {
			reason = kind.Error()
		}
	}
	telemetry.RecordWorkerFailure(op, reason)
	return &Error{Op: op, Kind: kind, Detail: detail, Err: err}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
