package artifactcontract

import (
	"encoding/json"
	"fmt"
	"strings"
)

// IPCCommand is a gateway->worker command file (ipc_commands/<id>.json).
type IPCCommand struct {
	WorkerProtocolVersion string         `json:"worker_protocol_version"`
	WorkerProtocolName    string         `json:"worker_protocol_name"`
	TransportRole         string         `json:"transport_role"`
	CommandID             string         `json:"command_id"`
	CommandType           string         `json:"command_type"`
	Args                  map[string]any `json:"args"`
	Timestamp             string         `json:"timestamp"`
}

// IPCResponse is ipc_responses/<id>.json.
type IPCResponse struct {
	WorkerProtocolVersion string         `json:"worker_protocol_version"`
	WorkerProtocolName    string         `json:"worker_protocol_name"`
	TransportRole         string         `json:"transport_role"`
	CommandID             string         `json:"command_id"`
	Status                string         `json:"status"`
	Result                map[string]any `json:"result,omitempty"`
	Error                 *string        `json:"error,omitempty"`
	Timestamp             string         `json:"timestamp"`
}

// ValidateIPCCommand checks envelope fields.
func ValidateIPCCommand(c *IPCCommand) error {
	if c == nil {
		return fmt.Errorf("ipc command: nil")
	}
	if strings.TrimSpace(c.WorkerProtocolVersion) == "" {
		return fmt.Errorf("ipc command: missing worker_protocol_version")
	}
	if strings.TrimSpace(c.TransportRole) == "" {
		return fmt.Errorf("ipc command: missing transport_role")
	}
	if strings.TrimSpace(c.CommandID) == "" {
		return fmt.Errorf("ipc command: missing command_id")
	}
	if strings.TrimSpace(c.CommandType) == "" {
		return fmt.Errorf("ipc command: missing command_type")
	}
	return nil
}

// ValidateIPCResponse checks envelope fields.
func ValidateIPCResponse(r *IPCResponse) error {
	if r == nil {
		return fmt.Errorf("ipc response: nil")
	}
	if strings.TrimSpace(r.WorkerProtocolVersion) == "" {
		return fmt.Errorf("ipc response: missing worker_protocol_version")
	}
	if strings.TrimSpace(r.TransportRole) == "" {
		return fmt.Errorf("ipc response: missing transport_role")
	}
	if strings.TrimSpace(r.CommandID) == "" {
		return fmt.Errorf("ipc response: missing command_id")
	}
	if strings.TrimSpace(r.Status) == "" {
		return fmt.Errorf("ipc response: missing status")
	}
	return nil
}

// ReadIPCCommandJSON parses command JSON.
func ReadIPCCommandJSON(raw []byte) (IPCCommand, error) {
	var c IPCCommand
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, err
	}
	return c, ValidateIPCCommand(&c)
}

// ReadIPCResponseJSON parses response JSON.
func ReadIPCResponseJSON(raw []byte) (IPCResponse, error) {
	var r IPCResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, err
	}
	return r, ValidateIPCResponse(&r)
}

// WriteIPCCommandJSON serializes a validated command envelope.
func WriteIPCCommandJSON(c IPCCommand) ([]byte, error) {
	if err := ValidateIPCCommand(&c); err != nil {
		return nil, err
	}
	return json.MarshalIndent(c, "", "  ")
}

// WriteIPCResponseJSON serializes a validated response envelope.
func WriteIPCResponseJSON(r IPCResponse) ([]byte, error) {
	if err := ValidateIPCResponse(&r); err != nil {
		return nil, err
	}
	return json.MarshalIndent(r, "", "  ")
}
