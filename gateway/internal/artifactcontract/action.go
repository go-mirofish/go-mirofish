package artifactcontract

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ActionEvent is one JSONL line in twitter/ or reddit/ actions.jsonl
// (aligns with AgentAction.to_dict in Python).
type ActionEvent struct {
	RoundNum   int            `json:"round_num"`
	Timestamp  string         `json:"timestamp"`
	Platform   string         `json:"platform"`
	AgentID    int            `json:"agent_id"`
	AgentName  string         `json:"agent_name"`
	ActionType string         `json:"action_type"`
	ActionArgs map[string]any `json:"action_args"`
	Result     *string        `json:"result,omitempty"`
	Success    bool           `json:"success"`
}

// ValidateActionEvent enforces line-level invariants.
func ValidateActionEvent(e *ActionEvent) error {
	if e == nil {
		return fmt.Errorf("action: nil")
	}
	if strings.TrimSpace(e.Platform) == "" {
		return fmt.Errorf("action: missing platform")
	}
	if strings.TrimSpace(e.ActionType) == "" {
		return fmt.Errorf("action: missing action_type")
	}
	return nil
}

// ParseActionsJSONL reads and validates all lines.
func ParseActionsJSONL(r io.Reader) ([]ActionEvent, error) {
	var out []ActionEvent
	sc := bufio.NewScanner(r)
	// long lines
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		t := strings.TrimSpace(sc.Text())
		if t == "" {
			continue
		}
		var ev ActionEvent
		if err := json.Unmarshal([]byte(t), &ev); err != nil {
			return out, fmt.Errorf("actions.jsonl line %d: %w", line, err)
		}
		if err := ValidateActionEvent(&ev); err != nil {
			return out, fmt.Errorf("actions.jsonl line %d: %w", line, err)
		}
		out = append(out, ev)
	}
	return out, sc.Err()
}

// FormatActionsJSONL writes JSONL.
func FormatActionsJSONL(events []ActionEvent) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	for i := range events {
		if err := ValidateActionEvent(&events[i]); err != nil {
			return nil, err
		}
		if err := enc.Encode(events[i]); err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}
