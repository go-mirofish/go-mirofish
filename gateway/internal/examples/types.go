package examples

import "time"

type Definition struct {
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	ConfigPath  string   `json:"config_path"`
	Profiles    []string `json:"profiles"`
}

type RunOptions struct {
	RepoRoot   string
	OutputRoot string
	Profile    string
	SmokeOnly  bool
}

type RunResult struct {
	ExampleKey       string            `json:"example_key"`
	Title            string            `json:"title"`
	Profile          string            `json:"profile"`
	AgentCount       int               `json:"agent_count"`
	InteractionCount int               `json:"interaction_count"`
	TaskCount        int               `json:"task_count"`
	Artifacts        map[string]string `json:"artifacts"`
	Summary          map[string]any    `json:"summary"`
	OutputDir        string            `json:"output_dir"`
	LocalOnly        bool              `json:"local_only"`
	CompletedAt      string            `json:"completed_at"`
}

type BenchmarkEnvironment struct {
	Timestamp string `json:"timestamp"`
	GitCommit string `json:"git_commit"`
	Hostname  string `json:"hostname"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	CPUCount  int    `json:"cpu_count"`
}

type BenchmarkThresholds struct {
	StartupWarnMS float64 `json:"startup_warn_ms"`
	StartupFailMS float64 `json:"startup_fail_ms"`
	RuntimeWarnMS float64 `json:"runtime_warn_ms"`
	RuntimeFailMS float64 `json:"runtime_fail_ms"`
}

type BenchmarkEvaluation struct {
	Status   string   `json:"status"`
	Warnings []string `json:"warnings"`
	Failures []string `json:"failures"`
}

type BenchmarkResult struct {
	ExampleKey          string               `json:"example_key"`
	Title               string               `json:"title"`
	Profile             string               `json:"profile"`
	ConfigName          string               `json:"config_name"`
	AgentCount          int                  `json:"agent_count"`
	InteractionCount    int                  `json:"interaction_count"`
	TaskCount           int                  `json:"task_count"`
	StartupLatencyMS    float64              `json:"startup_latency_ms"`
	TotalRuntimeMS      float64              `json:"total_runtime_ms"`
	ArtifactCount       int                  `json:"artifact_count"`
	ArtifactSuccess     bool                 `json:"artifact_success"`
	DeterministicReplay bool                 `json:"deterministic_replay"`
	MemoryAllocBytes    uint64               `json:"memory_alloc_bytes"`
	LocalOnly           bool                 `json:"local_only"`
	Environment         BenchmarkEnvironment `json:"environment"`
	Thresholds          BenchmarkThresholds  `json:"thresholds"`
	Evaluation          BenchmarkEvaluation  `json:"evaluation"`
	Artifacts           map[string]string    `json:"artifacts"`
	ArtifactHashes      map[string]string    `json:"artifact_hashes"`
	CompletedAt         string               `json:"completed_at"`
}

type BenchmarkSuite struct {
	GeneratedAt string            `json:"generated_at"`
	Results     []BenchmarkResult `json:"results"`
}

type SmokeResult struct {
	ExampleKey       string            `json:"example_key"`
	Title            string            `json:"title"`
	Profile          string            `json:"profile"`
	Success          bool              `json:"success"`
	ArtifactSuccess  bool              `json:"artifact_success"`
	AgentCount       int               `json:"agent_count"`
	InteractionCount int               `json:"interaction_count"`
	TaskCount        int               `json:"task_count"`
	Artifacts        map[string]string `json:"artifacts"`
	OutputDir        string            `json:"output_dir"`
	FailureReason    string            `json:"failure_reason,omitempty"`
	CompletedAt      string            `json:"completed_at"`
}

type SmokeSuite struct {
	GeneratedAt string        `json:"generated_at"`
	Results     []SmokeResult `json:"results"`
}

type CompareResult struct {
	ExampleKey         string  `json:"example_key"`
	Profile            string  `json:"profile"`
	RuntimeDeltaMS     float64 `json:"runtime_delta_ms"`
	StartupDeltaMS     float64 `json:"startup_delta_ms"`
	MemoryDeltaBytes   int64   `json:"memory_delta_bytes"`
	ArtifactCountDelta int     `json:"artifact_count_delta"`
	StatusChanged      bool    `json:"status_changed"`
}

type CompareReport struct {
	Base    string          `json:"base"`
	Current string          `json:"current"`
	Results []CompareResult `json:"results"`
}

type ScenarioProfile struct {
	AgentCount       int                 `json:"agent_count"`
	InteractionCount int                 `json:"interaction_count"`
	TaskCount        int                 `json:"task_count"`
	Params           map[string]any      `json:"params,omitempty"`
	Thresholds       BenchmarkThresholds `json:"thresholds"`
}

type runnerFunc func(RunOptions) (RunResult, error)

type registeredExample struct {
	def Definition
	run runnerFunc
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
