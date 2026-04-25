// Package artifactcontract holds Go-owned JSON contracts for worker/simulation disk artifacts
// and IPC envelopes, with validation and I/O helpers.
package artifactcontract

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RunnerStatus matches backend RunnerStatus in simulation_runner.py
type RunnerStatus string

const (
	RunnerIdle      RunnerStatus = "idle"
	RunnerStarting  RunnerStatus = "starting"
	RunnerRunning   RunnerStatus = "running"
	RunnerPaused    RunnerStatus = "paused"
	RunnerStopping  RunnerStatus = "stopping"
	RunnerStopped   RunnerStatus = "stopped"
	RunnerCompleted RunnerStatus = "completed"
	RunnerFailed    RunnerStatus = "failed"
)

// RunState is the on-disk run_state.json shape (subset enforced by ValidateRunState).
type RunState struct {
	WorkerProtocolVersion string  `json:"worker_protocol_version"`
	SimulationID          string  `json:"simulation_id"`
	RunnerStatus          string  `json:"runner_status"`
	CurrentRound          int     `json:"current_round"`
	TotalRounds           int     `json:"total_rounds"`
	SimulatedHours        int     `json:"simulated_hours,omitempty"`
	TotalSimulationHours  int     `json:"total_simulation_hours,omitempty"`
	ProgressPercent       float64 `json:"progress_percent"`
	TwitterCurrentRound   int     `json:"twitter_current_round,omitempty"`
	RedditCurrentRound    int     `json:"reddit_current_round,omitempty"`
	TwitterSimulatedHours int     `json:"twitter_simulated_hours,omitempty"`
	RedditSimulatedHours  int     `json:"reddit_simulated_hours,omitempty"`
	TwitterRunning        bool    `json:"twitter_running,omitempty"`
	RedditRunning         bool    `json:"reddit_running,omitempty"`
	TwitterCompleted      bool    `json:"twitter_completed,omitempty"`
	RedditCompleted       bool    `json:"reddit_completed,omitempty"`
	StartedAt             string  `json:"started_at,omitempty"`
	CompletedAt           string  `json:"completed_at,omitempty"`
	ProcessPID            int     `json:"process_pid,omitempty"`
	TwitterActionsCount   int     `json:"twitter_actions_count"`
	RedditActionsCount    int     `json:"reddit_actions_count"`
	TotalActionsCount     int     `json:"total_actions_count"`
	UpdatedAt             string  `json:"updated_at"`
	Error                 *string `json:"error,omitempty"`
}

// ValidateRunState enforces required public fields.
func ValidateRunState(s *RunState) error {
	if s == nil {
		return fmt.Errorf("run state: nil")
	}
	if strings.TrimSpace(s.SimulationID) == "" {
		return fmt.Errorf("run state: missing simulation_id")
	}
	if strings.TrimSpace(s.RunnerStatus) == "" {
		return fmt.Errorf("run state: missing runner_status")
	}
	if strings.TrimSpace(s.WorkerProtocolVersion) == "" {
		return fmt.Errorf("run state: missing worker_protocol_version")
	}
	return nil
}

// ReadRunStateJSON parses and validates run_state.json bytes.
func ReadRunStateJSON(raw []byte) (RunState, error) {
	var s RunState
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, err
	}
	if err := ValidateRunState(&s); err != nil {
		return s, err
	}
	return s, nil
}

// WriteRunStateJSON returns indented JSON.
func WriteRunStateJSON(s RunState) ([]byte, error) {
	if err := ValidateRunState(&s); err != nil {
		return nil, err
	}
	return json.MarshalIndent(s, "", "  ")
}
