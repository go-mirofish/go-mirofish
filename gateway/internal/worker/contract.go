package worker

import "context"

type Platform string

const (
	PlatformTwitter  Platform = "twitter"
	PlatformReddit   Platform = "reddit"
	PlatformParallel Platform = "parallel"
)

type StartRequest struct {
	SimulationID            string   `json:"simulation_id"`
	Platform                Platform `json:"platform"`
	MaxRounds               int      `json:"max_rounds,omitempty"`
	EnableGraphMemoryUpdate bool     `json:"enable_graph_memory_update,omitempty"`
	GraphID                 string   `json:"graph_id,omitempty"`
}

type StartResponse struct {
	SimulationID             string `json:"simulation_id"`
	RunnerStatus             string `json:"runner_status"`
	ProcessPID               int    `json:"process_pid,omitempty"`
	StartedAt                string `json:"started_at,omitempty"`
	MaxRoundsApplied         int    `json:"max_rounds_applied,omitempty"`
	GraphMemoryUpdateEnabled bool   `json:"graph_memory_update_enabled,omitempty"`
	GraphID                  string `json:"graph_id,omitempty"`
}

type StopRequest struct {
	SimulationID string `json:"simulation_id"`
}

type EnvStatusRequest struct {
	SimulationID string `json:"simulation_id"`
}

type CloseEnvRequest struct {
	SimulationID string `json:"simulation_id"`
	Timeout      int    `json:"timeout,omitempty"`
}

type InterviewRequest struct {
	SimulationID string   `json:"simulation_id"`
	AgentID      int      `json:"agent_id"`
	Prompt       string   `json:"prompt"`
	Platform     Platform `json:"platform,omitempty"`
	Timeout      int      `json:"timeout,omitempty"`
}

type BatchInterviewItem struct {
	AgentID  int      `json:"agent_id"`
	Prompt   string   `json:"prompt"`
	Platform Platform `json:"platform,omitempty"`
}

type BatchInterviewRequest struct {
	SimulationID string               `json:"simulation_id"`
	Interviews   []BatchInterviewItem `json:"interviews"`
	Platform     Platform             `json:"platform,omitempty"`
	Timeout      int                  `json:"timeout,omitempty"`
}

type AllInterviewRequest struct {
	SimulationID string   `json:"simulation_id"`
	Prompt       string   `json:"prompt"`
	Platform     Platform `json:"platform,omitempty"`
	Timeout      int      `json:"timeout,omitempty"`
}

type IPCResult struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp,omitempty"`
	Error     string `json:"error,omitempty"`
	Result    any    `json:"result,omitempty"`
}

type EnvStatus struct {
	SimulationID     string `json:"simulation_id"`
	EnvAlive         bool   `json:"env_alive"`
	TwitterAvailable bool   `json:"twitter_available"`
	RedditAvailable  bool   `json:"reddit_available"`
	Message          string `json:"message"`
}

type Bridge interface {
	StartSimulation(context.Context, StartRequest) (StartResponse, error)
	StopSimulation(context.Context, StopRequest) (map[string]any, error)
	Interview(context.Context, InterviewRequest) (IPCResult, error)
	BatchInterview(context.Context, BatchInterviewRequest) (IPCResult, error)
	InterviewAll(context.Context, AllInterviewRequest) (IPCResult, error)
	EnvStatus(context.Context, EnvStatusRequest) (EnvStatus, error)
	CloseEnv(context.Context, CloseEnvRequest) (IPCResult, error)
}
