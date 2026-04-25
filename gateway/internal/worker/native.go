package worker

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/internal/artifactcontract"
)

type NativeBridge struct {
	SimulationsDir string

	mu       sync.Mutex
	sessions map[string]*nativeSession
}

type nativeSession struct {
	cancel context.CancelFunc
	done   chan struct{}

	platforms       []Platform
	agentIDs        []int
	minutesPerRound int
}

type nativeInterviewEntry struct {
	AgentID   int    `json:"agent_id"`
	Prompt    string `json:"prompt"`
	Response  string `json:"response"`
	Timestamp string `json:"timestamp"`
	Platform  string `json:"platform"`
}

func NewNativeBridge(simulationsDir string) *NativeBridge {
	return &NativeBridge{
		SimulationsDir: simulationsDir,
		sessions:       map[string]*nativeSession{},
	}
}

func (b *NativeBridge) StartSimulation(ctx context.Context, req StartRequest) (StartResponse, error) {
	if req.SimulationID == "" {
		return StartResponse{}, workerError("NativeStartSimulation", ErrWorkerBadRequest, "simulation_id is required", nil)
	}
	if req.WorkerProtocolVersion == "" {
		req.WorkerProtocolVersion = ProtocolVersion
	}
	if req.WorkerProtocolVersion != ProtocolVersion {
		return StartResponse{}, workerError("NativeStartSimulation", ErrWorkerIncompatible, fmt.Sprintf("unsupported worker protocol version %q", req.WorkerProtocolVersion), nil)
	}

	config, err := b.readConfig(req.SimulationID)
	if err != nil {
		return StartResponse{}, workerError("NativeStartSimulation", ErrWorkerNotFound, "", err)
	}
	agentIDs := config.agentIDs()
	totalRounds := config.totalRounds(req.MaxRounds)
	platforms := requestedPlatforms(req.Platform, config)
	minutesPerRound := config.TimeConfig.MinutesPerRound
	if minutesPerRound <= 0 {
		minutesPerRound = 60
	}
	now := time.Now().Format(time.RFC3339)
	twitterAvailable, redditAvailable := availablePlatforms(platforms)

	runState := artifactcontract.RunState{
		WorkerProtocolVersion: ProtocolVersion,
		SimulationID:          req.SimulationID,
		RunnerStatus:          "running",
		CurrentRound:          0,
		TotalRounds:           totalRounds,
		SimulatedHours:        0,
		TotalSimulationHours:  maxInt(config.TimeConfig.TotalSimulationHours, 1),
		ProgressPercent:       0,
		TwitterCurrentRound:   0,
		RedditCurrentRound:    0,
		TwitterSimulatedHours: 0,
		RedditSimulatedHours:  0,
		TwitterRunning:        twitterAvailable,
		RedditRunning:         redditAvailable,
		TwitterCompleted:      false,
		RedditCompleted:       false,
		StartedAt:             now,
		ProcessPID:            0,
		TwitterActionsCount:   0,
		RedditActionsCount:    0,
		TotalActionsCount:     0,
		UpdatedAt:             now,
	}
	if err := b.writeRunState(req.SimulationID, runState); err != nil {
		return StartResponse{}, workerError("NativeStartSimulation", ErrWorkerUnavailable, "", err)
	}
	if err := b.writeRuntimeState(req.SimulationID, runtimeStatePayload(req.SimulationID, runState)); err != nil {
		return StartResponse{}, workerError("NativeStartSimulation", ErrWorkerUnavailable, "", err)
	}
	if err := b.writeEnvStatus(req.SimulationID, nativeEnvStatus("alive", now, twitterAvailable, redditAvailable)); err != nil {
		return StartResponse{}, workerError("NativeStartSimulation", ErrWorkerUnavailable, "", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	session := &nativeSession{
		cancel:          cancel,
		done:            make(chan struct{}),
		platforms:       platforms,
		agentIDs:        agentIDs,
		minutesPerRound: minutesPerRound,
	}

	b.mu.Lock()
	if existing := b.sessions[req.SimulationID]; existing != nil {
		b.mu.Unlock()
		return StartResponse{}, workerError("NativeStartSimulation", ErrWorkerUnavailable, "simulation already running", nil)
	}
	b.sessions[req.SimulationID] = session
	b.mu.Unlock()

	go b.runSimulation(runCtx, req.SimulationID, session, runState)
	_ = ctx
	return StartResponse{
		SimulationID:             req.SimulationID,
		RunnerStatus:             "running",
		ProcessPID:               0,
		StartedAt:                runState.StartedAt,
		MaxRoundsApplied:         req.MaxRounds,
		GraphMemoryUpdateEnabled: req.EnableGraphMemoryUpdate,
		GraphID:                  req.GraphID,
	}, nil
}

func (b *NativeBridge) StopSimulation(ctx context.Context, req StopRequest) (map[string]any, error) {
	b.mu.Lock()
	session := b.sessions[req.SimulationID]
	b.mu.Unlock()
	if session == nil {
		runState, err := b.readRunState(req.SimulationID)
		if err != nil {
			return nil, workerError("NativeStopSimulation", ErrWorkerNotFound, "", err)
		}
		return runStateToMap(runState), nil
	}
	session.cancel()
	select {
	case <-session.done:
	case <-ctx.Done():
		return nil, workerError("NativeStopSimulation", ErrWorkerTimeout, "", ctx.Err())
	case <-time.After(5 * time.Second):
		return nil, workerError("NativeStopSimulation", ErrWorkerTimeout, "timed out waiting for native simulation stop", nil)
	}
	runState, err := b.readRunState(req.SimulationID)
	if err != nil {
		return nil, workerError("NativeStopSimulation", ErrWorkerUnavailable, "", err)
	}
	return runStateToMap(runState), nil
}

func (b *NativeBridge) Interview(ctx context.Context, req InterviewRequest) (IPCResult, error) {
	env, err := b.EnvStatus(ctx, EnvStatusRequest{SimulationID: req.SimulationID})
	if err != nil {
		return IPCResult{}, err
	}
	if !env.EnvAlive {
		return IPCResult{}, workerError("NativeInterview", ErrWorkerNotReady, "simulation environment is not running", nil)
	}
	result, err := b.interviewResult(req.SimulationID, req.AgentID, req.Prompt, req.Platform)
	if err != nil {
		return IPCResult{}, err
	}
	return IPCResult{
		WorkerProtocolVersion: ProtocolVersion,
		Success:               true,
		Timestamp:             time.Now().Format(time.RFC3339),
		Result:                result,
	}, nil
}

func (b *NativeBridge) BatchInterview(ctx context.Context, req BatchInterviewRequest) (IPCResult, error) {
	env, err := b.EnvStatus(ctx, EnvStatusRequest{SimulationID: req.SimulationID})
	if err != nil {
		return IPCResult{}, err
	}
	if !env.EnvAlive {
		return IPCResult{}, workerError("NativeBatchInterview", ErrWorkerNotReady, "simulation environment is not running", nil)
	}
	results := map[string]any{}
	for _, item := range req.Interviews {
		platforms := []Platform{item.Platform}
		if item.Platform == "" {
			platforms = []Platform{PlatformTwitter, PlatformReddit}
		}
		for _, platform := range platforms {
			result, err := b.interviewResult(req.SimulationID, item.AgentID, item.Prompt, platform)
			if err != nil {
				continue
			}
			results[string(platform)+"_"+fmt.Sprintf("%d", item.AgentID)] = result
		}
	}
	if len(results) == 0 {
		return IPCResult{}, workerError("NativeBatchInterview", ErrWorkerUnavailable, "no interview results produced", nil)
	}
	return IPCResult{
		WorkerProtocolVersion: ProtocolVersion,
		Success:               true,
		Timestamp:             time.Now().Format(time.RFC3339),
		Result: map[string]any{
			"interviews_count": len(results),
			"results":          results,
		},
	}, nil
}

func (b *NativeBridge) InterviewAll(ctx context.Context, req AllInterviewRequest) (IPCResult, error) {
	config, err := b.readConfig(req.SimulationID)
	if err != nil {
		return IPCResult{}, workerError("NativeInterviewAll", ErrWorkerNotFound, "", err)
	}
	items := make([]BatchInterviewItem, 0, len(config.AgentConfigs))
	for _, item := range config.AgentConfigs {
		items = append(items, BatchInterviewItem{AgentID: item.AgentID, Prompt: req.Prompt, Platform: req.Platform})
	}
	return b.BatchInterview(ctx, BatchInterviewRequest{
		SimulationID: req.SimulationID,
		Interviews:   items,
		Platform:     req.Platform,
		Timeout:      req.Timeout,
	})
}

func (b *NativeBridge) EnvStatus(ctx context.Context, req EnvStatusRequest) (EnvStatus, error) {
	_ = ctx
	raw, err := os.ReadFile(filepath.Join(b.SimulationsDir, req.SimulationID, "env_status.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return EnvStatus{
				SimulationID:     req.SimulationID,
				EnvAlive:         false,
				TwitterAvailable: false,
				RedditAvailable:  false,
				Message:          "No environment status (simulation env not started or not created yet)",
			}, nil
		}
		return EnvStatus{}, workerError("NativeEnvStatus", ErrWorkerNotFound, "", err)
	}
	status, err := artifactcontract.ReadEnvStatusJSON(raw)
	if err != nil {
		return EnvStatus{}, workerError("NativeEnvStatus", ErrWorkerUnavailable, "", err)
	}
	twitterAvailable := status.TwitterAvailable
	redditAvailable := status.RedditAvailable
	if status.Status == "alive" && !twitterAvailable && !redditAvailable {
		twitterAvailable = true
		redditAvailable = true
	}
	return EnvStatus{
		WorkerProtocolVersion: status.WorkerProtocolVersion,
		SimulationID:          req.SimulationID,
		EnvAlive:              status.Status == "alive",
		TwitterAvailable:      twitterAvailable,
		RedditAvailable:       redditAvailable,
		Message:               ternary(status.Status == "alive", "Environment is running", "Environment is not running"),
	}, nil
}

func (b *NativeBridge) CloseEnv(ctx context.Context, req CloseEnvRequest) (IPCResult, error) {
	_, err := b.StopSimulation(ctx, StopRequest{SimulationID: req.SimulationID})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return IPCResult{}, err
	}
	now := time.Now().Format(time.RFC3339)
	if runState, runErr := b.readRunState(req.SimulationID); runErr == nil {
		runState.RunnerStatus = "stopped"
		runState.TwitterRunning = false
		runState.RedditRunning = false
		runState.UpdatedAt = now
		if runState.CompletedAt == "" {
			runState.CompletedAt = now
		}
		_ = b.writeRunState(req.SimulationID, runState)
		_ = b.writeRuntimeState(req.SimulationID, runtimeStatePayload(req.SimulationID, runState))
	}
	if err := b.writeEnvStatus(req.SimulationID, nativeEnvStatus("stopped", now, false, false)); err != nil {
		return IPCResult{}, workerError("NativeCloseEnv", ErrWorkerUnavailable, "", err)
	}
	return IPCResult{
		WorkerProtocolVersion: ProtocolVersion,
		Success:               true,
		Timestamp:             time.Now().Format(time.RFC3339),
		Result:                map[string]any{"closed": true},
	}, nil
}

func (b *NativeBridge) runSimulation(ctx context.Context, simulationID string, session *nativeSession, state artifactcontract.RunState) {
	defer close(session.done)
	defer func() {
		b.mu.Lock()
		delete(b.sessions, simulationID)
		b.mu.Unlock()
	}()

	sched := NativeRoundScheduler{
		TotalRounds: state.TotalRounds,
		Platforms:   session.platforms,
		AgentIDs:    session.agentIDs,
	}
	sched.ForEachRound(func(round int) bool {
		select {
		case <-ctx.Done():
			state.RunnerStatus = "stopped"
			state.TwitterRunning = false
			state.RedditRunning = false
			state.CompletedAt = time.Now().Format(time.RFC3339)
			state.UpdatedAt = state.CompletedAt
			_ = b.writeRunState(simulationID, state)
			_ = b.writeRuntimeState(simulationID, runtimeStatePayload(simulationID, state))
			_ = b.writeEnvStatus(simulationID, nativeEnvStatus("stopped", state.CompletedAt, state.TwitterCompleted || state.TwitterActionsCount > 0, state.RedditCompleted || state.RedditActionsCount > 0))
			return false
		default:
		}

		_ = b.appendTimelineEvent(simulationID, round, "round_start", map[string]any{
			"platforms": len(session.platforms),
			"agents":    len(session.agentIDs),
		})

		for _, platform := range session.platforms {
			events := make([]artifactcontract.ActionEvent, 0, len(session.agentIDs))
			for _, agentID := range session.agentIDs {
				events = append(events, artifactcontract.ActionEvent{
					RoundNum:   round,
					Timestamp:  time.Now().Format(time.RFC3339),
					Platform:   string(platform),
					AgentID:    agentID,
					AgentName:  fmt.Sprintf("Agent %d", agentID),
					ActionType: actionTypeForPlatform(platform),
					ActionArgs: map[string]any{"content": fmt.Sprintf("%s round %d agent %d", platform, round, agentID)},
					Success:    true,
				})
			}
			_ = b.appendActions(simulationID, platform, events)
			if platform == PlatformTwitter {
				state.TwitterCurrentRound = round
				state.TwitterSimulatedHours = round * session.minutesPerRound / 60
				state.TwitterActionsCount += len(events)
			} else {
				state.RedditCurrentRound = round
				state.RedditSimulatedHours = round * session.minutesPerRound / 60
				state.RedditActionsCount += len(events)
			}
		}
		state.CurrentRound = round
		state.SimulatedHours = round * session.minutesPerRound / 60
		state.TotalActionsCount = state.TwitterActionsCount + state.RedditActionsCount
		state.ProgressPercent = float64(round) / float64(maxInt(state.TotalRounds, 1)) * 100
		state.UpdatedAt = time.Now().Format(time.RFC3339)
		_ = b.writeRunState(simulationID, state)
		_ = b.writeRuntimeState(simulationID, runtimeStatePayload(simulationID, state))
		_ = b.appendTimelineEvent(simulationID, round, "round_complete", map[string]any{
			"twitter_actions": state.TwitterActionsCount,
			"reddit_actions":  state.RedditActionsCount,
		})
		time.Sleep(50 * time.Millisecond)
		return true
	})

	if state.RunnerStatus == "stopped" {
		return
	}

	state.RunnerStatus = "completed"
	state.TwitterRunning = false
	state.RedditRunning = false
	state.TwitterCompleted = hasPlatform(session.platforms, PlatformTwitter)
	state.RedditCompleted = hasPlatform(session.platforms, PlatformReddit)
	state.CompletedAt = time.Now().Format(time.RFC3339)
	state.UpdatedAt = state.CompletedAt
	_ = b.writeRunState(simulationID, state)
	_ = b.writeRuntimeState(simulationID, runtimeStatePayload(simulationID, state))
	_ = b.writeEnvStatus(simulationID, nativeEnvStatus("stopped", state.CompletedAt, state.TwitterCompleted, state.RedditCompleted))
}

func (b *NativeBridge) interviewResult(simulationID string, agentID int, prompt string, platform Platform) (map[string]any, error) {
	if platform == "" {
		platform = PlatformParallel
	}
	if platform == PlatformParallel {
		platforms := map[string]any{}
		for _, item := range []Platform{PlatformTwitter, PlatformReddit} {
			res, err := b.interviewResult(simulationID, agentID, prompt, item)
			if err == nil {
				platforms[string(item)] = res
			}
		}
		return map[string]any{
			"agent_id":  agentID,
			"prompt":    prompt,
			"platforms": platforms,
		}, nil
	}

	name := b.lookupAgentName(simulationID, agentID, platform)
	response := fmt.Sprintf("%s responds to %q", name, prompt)
	entry := nativeInterviewEntry{
		AgentID:   agentID,
		Prompt:    prompt,
		Response:  response,
		Timestamp: time.Now().Format(time.RFC3339),
		Platform:  string(platform),
	}
	if err := b.appendInterview(simulationID, platform, entry); err != nil {
		return nil, workerError("NativeInterview", ErrWorkerUnavailable, "", err)
	}
	return map[string]any{
		"agent_id":  agentID,
		"response":  response,
		"timestamp": entry.Timestamp,
		"platform":  string(platform),
	}, nil
}

func (b *NativeBridge) appendTimelineEvent(simulationID string, round int, event string, fields map[string]any) error {
	rec := map[string]any{
		"ts":    time.Now().Format(time.RFC3339Nano),
		"round": round,
		"event": event,
	}
	for k, v := range fields {
		rec[k] = v
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	path := filepath.Join(b.SimulationsDir, simulationID, "event_timeline.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(raw, '\n'))
	return err
}

func (b *NativeBridge) appendActions(simulationID string, platform Platform, events []artifactcontract.ActionEvent) error {
	raw, err := artifactcontract.FormatActionsJSONL(events)
	if err != nil {
		return err
	}
	dir := filepath.Join(b.SimulationsDir, simulationID, string(platform))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(dir, "actions.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(raw)
	return err
}

func (b *NativeBridge) appendInterview(simulationID string, platform Platform, entry nativeInterviewEntry) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	path := filepath.Join(b.SimulationsDir, simulationID, string(platform)+"_interviews.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(raw, '\n'))
	return err
}

func (b *NativeBridge) writeRunState(simulationID string, state artifactcontract.RunState) error {
	raw, err := artifactcontract.WriteRunStateJSON(state)
	if err != nil {
		return err
	}
	path := filepath.Join(b.SimulationsDir, simulationID, "run_state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func (b *NativeBridge) writeEnvStatus(simulationID string, status artifactcontract.EnvStatus) error {
	raw, err := artifactcontract.WriteEnvStatusJSON(status)
	if err != nil {
		return err
	}
	path := filepath.Join(b.SimulationsDir, simulationID, "env_status.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func (b *NativeBridge) writeRuntimeState(simulationID string, updates map[string]any) error {
	path := filepath.Join(b.SimulationsDir, simulationID, "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &payload)
	}
	for key, value := range updates {
		if value == nil {
			delete(payload, key)
			continue
		}
		payload[key] = value
	}
	payload["simulation_id"] = simulationID
	payload[ProtocolVersionField] = ProtocolVersion
	payload[ProtocolNameField] = ProtocolName
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func (b *NativeBridge) readRunState(simulationID string) (artifactcontract.RunState, error) {
	raw, err := os.ReadFile(filepath.Join(b.SimulationsDir, simulationID, "run_state.json"))
	if err != nil {
		return artifactcontract.RunState{}, err
	}
	return artifactcontract.ReadRunStateJSON(raw)
}

func runStateToMap(state artifactcontract.RunState) map[string]any {
	raw, _ := artifactcontract.WriteRunStateJSON(state)
	var payload map[string]any
	_ = json.Unmarshal(raw, &payload)
	return payload
}

func runtimeStatePayload(simulationID string, state artifactcontract.RunState) map[string]any {
	payload := map[string]any{
		"simulation_id":           simulationID,
		"status":                  runtimeStatusFromRunnerState(state.RunnerStatus),
		"runner_status":           state.RunnerStatus,
		"current_round":           state.CurrentRound,
		"total_rounds":            state.TotalRounds,
		"simulated_hours":         state.SimulatedHours,
		"total_simulation_hours":  state.TotalSimulationHours,
		"progress_percent":        state.ProgressPercent,
		"twitter_current_round":   state.TwitterCurrentRound,
		"reddit_current_round":    state.RedditCurrentRound,
		"twitter_simulated_hours": state.TwitterSimulatedHours,
		"reddit_simulated_hours":  state.RedditSimulatedHours,
		"twitter_running":         state.TwitterRunning,
		"reddit_running":          state.RedditRunning,
		"twitter_completed":       state.TwitterCompleted,
		"reddit_completed":        state.RedditCompleted,
		"twitter_actions_count":   state.TwitterActionsCount,
		"reddit_actions_count":    state.RedditActionsCount,
		"total_actions_count":     state.TotalActionsCount,
		"started_at":              state.StartedAt,
		"updated_at":              state.UpdatedAt,
		"completed_at":            state.CompletedAt,
	}
	if state.Error != nil {
		payload["error"] = *state.Error
	} else {
		payload["error"] = nil
	}
	return payload
}

type nativeConfig struct {
	SimulationID string         `json:"simulation_id"`
	AgentConfigs []nativeAgent  `json:"agent_configs"`
	TimeConfig   nativeTime     `json:"time_config"`
	Twitter      map[string]any `json:"twitter_config"`
	Reddit       map[string]any `json:"reddit_config"`
}

type nativeAgent struct {
	AgentID int `json:"agent_id"`
}

type nativeTime struct {
	TotalSimulationHours int `json:"total_simulation_hours"`
	MinutesPerRound      int `json:"minutes_per_round"`
}

func (c nativeConfig) totalRounds(maxRounds int) int {
	minutes := c.TimeConfig.MinutesPerRound
	if minutes <= 0 {
		minutes = 60
	}
	total := c.TimeConfig.TotalSimulationHours
	if total <= 0 {
		total = 1
	}
	rounds := total * 60 / minutes
	if rounds <= 0 {
		rounds = 1
	}
	if maxRounds > 0 && maxRounds < rounds {
		return maxRounds
	}
	return rounds
}

func (c nativeConfig) agentIDs() []int {
	out := make([]int, 0, len(c.AgentConfigs))
	for _, item := range c.AgentConfigs {
		if item.AgentID > 0 {
			out = append(out, item.AgentID)
		}
	}
	if len(out) == 0 {
		return []int{1}
	}
	return out
}

func requestedPlatforms(platform Platform, config nativeConfig) []Platform {
	switch platform {
	case PlatformTwitter:
		return []Platform{PlatformTwitter}
	case PlatformReddit:
		return []Platform{PlatformReddit}
	default:
		return []Platform{PlatformTwitter, PlatformReddit}
	}
}

func (b *NativeBridge) readConfig(simulationID string) (nativeConfig, error) {
	raw, err := os.ReadFile(filepath.Join(b.SimulationsDir, simulationID, "simulation_config.json"))
	if err != nil {
		return nativeConfig{}, err
	}
	var cfg nativeConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nativeConfig{}, err
	}
	return cfg, nil
}

func (b *NativeBridge) lookupAgentName(simulationID string, agentID int, platform Platform) string {
	if platform == PlatformTwitter {
		path := filepath.Join(b.SimulationsDir, simulationID, "twitter_profiles.csv")
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()
			rows, err := csv.NewReader(file).ReadAll()
			if err == nil && len(rows) > 1 {
				headers := rows[0]
				userIdx := indexOf(headers, "user_id")
				nameIdx := indexOf(headers, "name")
				for _, row := range rows[1:] {
					if userIdx >= 0 && userIdx < len(row) && row[userIdx] == fmt.Sprintf("%d", agentID) && nameIdx >= 0 && nameIdx < len(row) {
						return row[nameIdx]
					}
				}
			}
		}
	}
	if platform == PlatformReddit {
		path := filepath.Join(b.SimulationsDir, simulationID, "reddit_profiles.json")
		raw, err := os.ReadFile(path)
		if err == nil {
			var payload []map[string]any
			if json.Unmarshal(raw, &payload) == nil {
				for _, item := range payload {
					if intValue(item["user_id"]) == agentID || intValue(item["agent_id"]) == agentID {
						if name, _ := item["name"].(string); strings.TrimSpace(name) != "" {
							return name
						}
					}
				}
			}
		}
	}
	return fmt.Sprintf("Agent %d", agentID)
}

func actionTypeForPlatform(platform Platform) string {
	if platform == PlatformTwitter {
		return "POST_TWEET"
	}
	return "CREATE_POST"
}

func indexOf(items []string, target string) int {
	for idx, item := range items {
		if item == target {
			return idx
		}
	}
	return -1
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
