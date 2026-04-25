package simulation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/internal/runinstructions"
	simulationstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/simulation"
	intworker "github.com/go-mirofish/go-mirofish/gateway/internal/worker"
)

var (
	ErrProjectIDRequired    = errors.New("project_id is required")
	ErrSimulationIDRequired = errors.New("simulation_id is required")
	ErrProjectNotFound      = errors.New("project not found")
	ErrGraphNotBuilt        = errors.New("graph not built")
)

type Service struct {
	store  *simulationstore.Store
	bridge intworker.Bridge
	now    func() time.Time
}

func NewService(store *simulationstore.Store, bridge intworker.Bridge) *Service {
	return &Service{
		store:  store,
		bridge: bridge,
		now:    time.Now,
	}
}

func (s *Service) Create(req CreateRequest) (map[string]any, error) {
	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		return nil, ErrProjectIDRequired
	}

	project, err := s.store.ReadProject(projectID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	graphID := strings.TrimSpace(req.GraphID)
	if graphID == "" {
		graphID = stringValue(project["graph_id"])
	}
	if graphID == "" {
		return nil, ErrGraphNotBuilt
	}

	enableTwitter := true
	if req.EnableTwitter != nil {
		enableTwitter = *req.EnableTwitter
	}
	enableReddit := true
	if req.EnableReddit != nil {
		enableReddit = *req.EnableReddit
	}

	simulationID, err := newSimulationID()
	if err != nil {
		return nil, err
	}
	now := s.now().Format(time.RFC3339)
	state := map[string]any{
		"simulation_id":    simulationID,
		"project_id":       projectID,
		"graph_id":         graphID,
		"enable_twitter":   enableTwitter,
		"enable_reddit":    enableReddit,
		"status":           "created",
		"entities_count":   0,
		"profiles_count":   0,
		"entity_types":     []any{},
		"config_generated": false,
		"config_reasoning": "",
		"current_round":    0,
		"twitter_status":   "not_started",
		"reddit_status":    "not_started",
		"created_at":       now,
		"updated_at":       now,
		"error":            nil,
	}
	if err := s.store.WriteState(simulationID, state); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *Service) Delete(ctx context.Context, simulationID string) (map[string]any, error) {
	simulationID = strings.TrimSpace(simulationID)
	if simulationID == "" {
		return nil, ErrSimulationIDRequired
	}

	warnings := []string{}
	runState, err := s.store.ReadRunState(simulationID)
	if err == nil && shouldStopRunState(runState) && s.bridge != nil {
		if _, stopErr := s.bridge.StopSimulation(ctx, intworker.StopRequest{SimulationID: simulationID}); stopErr != nil {
			warnings = append(warnings, stopErr.Error())
		}
	}

	if err := s.store.DeleteSimulation(simulationID); err != nil {
		return nil, err
	}

	return map[string]any{
		"simulation_id": simulationID,
		"warnings":      warnings,
	}, nil
}

func (s *Service) List(projectID string) ([]map[string]any, error) {
	return s.store.ListSimulations(strings.TrimSpace(projectID))
}

func (s *Service) History(limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 20
	}

	simulations, err := s.store.ListSimulations("")
	if err != nil {
		return nil, err
	}
	if len(simulations) > limit {
		simulations = simulations[:limit]
	}

	enriched := make([]map[string]any, 0, len(simulations))
	for _, sim := range simulations {
		simCopy := cloneMap(sim)
		simulationID := stringValue(simCopy["simulation_id"])
		projectID := stringValue(simCopy["project_id"])

		config, _, _, _ := s.store.ReadConfigWithMeta(simulationID)
		if config != nil {
			simCopy["simulation_requirement"] = stringValue(config["simulation_requirement"])
			timeConfig := mapValue(config["time_config"])
			totalHours := intValue(timeConfig["total_simulation_hours"])
			minutesPerRound := intValue(timeConfig["minutes_per_round"])
			simCopy["total_simulation_hours"] = totalHours
			if minutesPerRound > 0 {
				simCopy["total_rounds"] = totalHours * 60 / minutesPerRound
			}
		} else {
			simCopy["simulation_requirement"] = ""
			simCopy["total_simulation_hours"] = 0
		}

		if runState, err := s.store.ReadRunState(simulationID); err == nil {
			normalized := NormalizeRunStatus(simulationID, runState)
			simCopy["current_round"] = normalized["current_round"]
			simCopy["runner_status"] = normalized["runner_status"]
			simCopy["total_actions_count"] = normalized["total_actions_count"]
			simCopy["twitter_actions_count"] = normalized["twitter_actions_count"]
			simCopy["reddit_actions_count"] = normalized["reddit_actions_count"]
			if simCopy["total_rounds"] == nil || intValue(simCopy["total_rounds"]) == 0 {
				simCopy["total_rounds"] = normalized["total_rounds"]
			}
			// Update top-level status to reflect runtime reality.
			rs := stringValue(normalized["runner_status"])
			switch rs {
			case "running", "starting":
				simCopy["status"] = "running"
			case "completed":
				simCopy["status"] = "completed"
			case "failed":
				simCopy["status"] = "failed"
			case "stopped":
				simCopy["status"] = "stopped"
			}
		} else {
			simCopy["current_round"] = 0
			simCopy["runner_status"] = "idle"
			simCopy["total_actions_count"] = 0
			simCopy["twitter_actions_count"] = 0
			simCopy["reddit_actions_count"] = 0
		}

		if projectID != "" {
			if project, err := s.store.ReadProject(projectID); err == nil {
				if files, ok := project["files"].([]any); ok {
					limited := make([]map[string]any, 0, minInt(3, len(files)))
					for _, file := range files[:minInt(3, len(files))] {
						if fileMap, ok := file.(map[string]any); ok {
							limited = append(limited, map[string]any{"filename": fileMap["filename"]})
						}
					}
					simCopy["files"] = limited
				}
			}
		}

		if report, err := s.store.FindLatestReportBySimulation(simulationID); err == nil && report != nil {
			simCopy["report_id"] = report["report_id"]
		} else {
			simCopy["report_id"] = nil
		}

		createdAt := stringValue(simCopy["created_at"])
		if len(createdAt) >= 10 {
			simCopy["created_date"] = createdAt[:10]
		} else {
			simCopy["created_date"] = ""
		}
		simCopy["version"] = "v1.0.2"
		enriched = append(enriched, simCopy)
	}

	return enriched, nil
}

func (s *Service) RunStatusDetail(simulationID, platform string) (map[string]any, error) {
	runState, err := s.store.ReadRunState(simulationID)
	if err != nil {
		if !isNotExist(err) {
			return nil, err
		}
		runState = nil
	}

	detail := NormalizeRunStatus(simulationID, runState)
	if runState == nil {
		detail["all_actions"] = []Action{}
		detail["twitter_actions"] = []Action{}
		detail["reddit_actions"] = []Action{}
		detail["recent_actions"] = []Action{}
		detail["rounds_count"] = 0
		return detail, nil
	}

	allActions, err := s.Actions(simulationID, platform, 0, 0)
	if err != nil {
		return nil, err
	}
	twitterActions := []Action{}
	if platform == "" || platform == "twitter" {
		twitterActions, err = s.Actions(simulationID, "twitter", 0, 0)
		if err != nil {
			return nil, err
		}
	}
	redditActions := []Action{}
	if platform == "" || platform == "reddit" {
		redditActions, err = s.Actions(simulationID, "reddit", 0, 0)
		if err != nil {
			return nil, err
		}
	}

	currentRound := intValue(runState["current_round"])
	recentActions := make([]Action, 0)
	if currentRound > 0 {
		for _, action := range allActions {
			if action.RoundNum == currentRound {
				recentActions = append(recentActions, action)
			}
		}
	}

	detail["all_actions"] = allActions
	detail["twitter_actions"] = twitterActions
	detail["reddit_actions"] = redditActions
	detail["recent_actions"] = recentActions
	detail["rounds_count"] = roundCount(runState, allActions, twitterActions, redditActions)
	return detail, nil
}

func (s *Service) InterviewHistory(simulationID, platform string, agentID *int, limit int) ([]map[string]any, error) {
	simulationID = strings.TrimSpace(simulationID)
	if simulationID == "" {
		return nil, ErrSimulationIDRequired
	}
	return s.store.InterviewHistory(simulationID, strings.TrimSpace(platform), agentID, limit)
}

func (s *Service) Start(ctx context.Context, req intworker.StartRequest) (intworker.StartResponse, error) {
	if s.bridge == nil {
		return intworker.StartResponse{}, errors.New("simulation worker bridge unavailable")
	}
	if req.Platform == "" {
		req.Platform = intworker.PlatformParallel
	}
	return s.bridge.StartSimulation(ctx, req)
}

func (s *Service) Stop(ctx context.Context, req intworker.StopRequest) (map[string]any, error) {
	if s.bridge == nil {
		return nil, errors.New("simulation worker bridge unavailable")
	}
	return s.bridge.StopSimulation(ctx, req)
}

func (s *Service) Interview(ctx context.Context, req intworker.InterviewRequest) (intworker.IPCResult, error) {
	if s.bridge == nil {
		return intworker.IPCResult{}, errors.New("simulation worker bridge unavailable")
	}
	return s.bridge.Interview(ctx, req)
}

func (s *Service) BatchInterview(ctx context.Context, req intworker.BatchInterviewRequest) (intworker.IPCResult, error) {
	if s.bridge == nil {
		return intworker.IPCResult{}, errors.New("simulation worker bridge unavailable")
	}
	return s.bridge.BatchInterview(ctx, req)
}

func (s *Service) InterviewAll(ctx context.Context, req intworker.AllInterviewRequest) (intworker.IPCResult, error) {
	if s.bridge == nil {
		return intworker.IPCResult{}, errors.New("simulation worker bridge unavailable")
	}
	return s.bridge.InterviewAll(ctx, req)
}

func (s *Service) EnvStatus(ctx context.Context, req intworker.EnvStatusRequest) (intworker.EnvStatus, error) {
	if s.bridge == nil {
		return intworker.EnvStatus{}, errors.New("simulation worker bridge unavailable")
	}
	return s.bridge.EnvStatus(ctx, req)
}

func (s *Service) CloseEnv(ctx context.Context, req intworker.CloseEnvRequest) (intworker.IPCResult, error) {
	if s.bridge == nil {
		return intworker.IPCResult{}, errors.New("simulation worker bridge unavailable")
	}
	return s.bridge.CloseEnv(ctx, req)
}

func (s *Service) Actions(simulationID, platform string, limit, offset int) ([]Action, error) {
	items, err := s.store.ReadActionLogs(simulationID, platform)
	if err != nil {
		return nil, err
	}
	var out []Action
	for _, item := range items {
		if _, ok := item["event_type"]; ok {
			continue
		}
		out = append(out, toAction(item, platform))
	}
	if offset > len(out) {
		return []Action{}, nil
	}
	end := len(out)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return out[offset:end], nil
}

func (s *Service) Timeline(simulationID string, startRound int, endRound int) ([]TimelineEntry, error) {
	items, err := s.store.ReadActionLogs(simulationID, "")
	if err != nil {
		return nil, err
	}
	rounds := map[int]*TimelineEntry{}
	activeAgents := map[int]map[int]bool{}
	for _, item := range items {
		round := actionRound(item)
		if startRound > 0 && round < startRound {
			continue
		}
		if endRound > 0 && round > endRound {
			continue
		}
		entry := rounds[round]
		if entry == nil {
			entry = &TimelineEntry{RoundNum: round, ActionTypes: map[string]int{}}
			rounds[round] = entry
			activeAgents[round] = map[int]bool{}
		}
		if _, ok := item["event_type"]; ok {
			continue
		}
		platform := valueOr(item["platform"], platformFromPath(item))
		if platform == "twitter" {
			entry.TwitterActions++
		} else if platform == "reddit" {
			entry.RedditActions++
		}
		entry.TotalActions++
		actionType := valueOr(item["action_type"], "")
		entry.ActionTypes[actionType]++
		agentID := intValue(item["agent_id"])
		activeAgents[round][agentID] = true
		ts := valueOr(item["timestamp"], "")
		if entry.FirstActionTime == "" || ts < entry.FirstActionTime {
			entry.FirstActionTime = ts
		}
		if ts > entry.LastActionTime {
			entry.LastActionTime = ts
		}
	}
	var out []TimelineEntry
	for round, entry := range rounds {
		for agentID := range activeAgents[round] {
			entry.ActiveAgents = append(entry.ActiveAgents, agentID)
		}
		sort.Ints(entry.ActiveAgents)
		entry.ActiveAgentsCount = len(entry.ActiveAgents)
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RoundNum < out[j].RoundNum })
	return out, nil
}

func (s *Service) AgentStats(simulationID string) ([]AgentStat, error) {
	items, err := s.store.ReadActionLogs(simulationID, "")
	if err != nil {
		return nil, err
	}
	stats := map[int]*AgentStat{}
	for _, item := range items {
		if _, ok := item["event_type"]; ok {
			continue
		}
		agentID := intValue(item["agent_id"])
		stat := stats[agentID]
		if stat == nil {
			stat = &AgentStat{
				AgentID:     agentID,
				AgentName:   valueOr(item["agent_name"], ""),
				ActionTypes: map[string]int{},
			}
			stats[agentID] = stat
		}
		platform := valueOr(item["platform"], platformFromPath(item))
		if platform == "twitter" {
			stat.TwitterActions++
		} else if platform == "reddit" {
			stat.RedditActions++
		}
		stat.TotalActions++
		actionType := valueOr(item["action_type"], "")
		stat.ActionTypes[actionType]++
		ts := valueOr(item["timestamp"], "")
		if stat.FirstActionTime == "" || ts < stat.FirstActionTime {
			stat.FirstActionTime = ts
		}
		if ts > stat.LastActionTime {
			stat.LastActionTime = ts
		}
	}
	var out []AgentStat
	for _, stat := range stats {
		out = append(out, *stat)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TotalActions > out[j].TotalActions })
	return out, nil
}

func (s *Service) Posts(simulationID, platform string, limit, offset int) ([]map[string]any, error) {
	actions, err := s.Actions(simulationID, platform, 0, 0)
	if err != nil {
		return nil, err
	}
	var posts []map[string]any
	for _, action := range actions {
		if action.ActionType != "CREATE_POST" {
			continue
		}
		posts = append(posts, map[string]any{
			"agent_id":    action.AgentID,
			"agent_name":  action.AgentName,
			"created_at":  action.Timestamp,
			"content":     action.ActionArgs["content"],
			"platform":    action.Platform,
			"action_type": action.ActionType,
		})
	}
	return paginateMaps(posts, limit, offset), nil
}

func (s *Service) Comments(simulationID string, limit, offset int) ([]map[string]any, error) {
	actions, err := s.Actions(simulationID, "reddit", 0, 0)
	if err != nil {
		return nil, err
	}
	var comments []map[string]any
	for _, action := range actions {
		if action.ActionType != "CREATE_COMMENT" {
			continue
		}
		comments = append(comments, map[string]any{
			"agent_id":    action.AgentID,
			"agent_name":  action.AgentName,
			"created_at":  action.Timestamp,
			"content":     action.ActionArgs["content"],
			"platform":    action.Platform,
			"action_type": action.ActionType,
		})
	}
	return paginateMaps(comments, limit, offset), nil
}

func NormalizeRunStatus(simulationID string, raw map[string]any) map[string]any {
	if raw == nil {
		return map[string]any{
			"simulation_id":         simulationID,
			"runner_status":         "idle",
			"current_round":         0,
			"total_rounds":          0,
			"progress_percent":      0,
			"twitter_actions_count": 0,
			"reddit_actions_count":  0,
			"total_actions_count":   0,
		}
	}

	data := map[string]any{
		"simulation_id":            firstNonEmpty(stringValue(raw["simulation_id"]), simulationID),
		"runner_status":            firstNonEmpty(stringValue(raw["runner_status"]), "idle"),
		"current_round":            intValue(raw["current_round"]),
		"total_rounds":             intValue(raw["total_rounds"]),
		"progress_percent":         floatValue(raw["progress_percent"]),
		"simulated_hours":          intValue(raw["simulated_hours"]),
		"total_simulation_hours":   intValue(raw["total_simulation_hours"]),
		"twitter_current_round":    intValue(firstNonNil(raw["twitter_current_round"], raw["twitter_round"])),
		"reddit_current_round":     intValue(firstNonNil(raw["reddit_current_round"], raw["reddit_round"])),
		"twitter_simulated_hours":  intValue(raw["twitter_simulated_hours"]),
		"reddit_simulated_hours":   intValue(raw["reddit_simulated_hours"]),
		"twitter_running":          boolValue(raw["twitter_running"]),
		"reddit_running":           boolValue(raw["reddit_running"]),
		"twitter_completed":        boolValue(raw["twitter_completed"]),
		"reddit_completed":         boolValue(raw["reddit_completed"]),
		"twitter_actions_count":    intValue(raw["twitter_actions_count"]),
		"reddit_actions_count":     intValue(raw["reddit_actions_count"]),
		"total_actions_count":      intValue(raw["total_actions_count"]),
	}
	for _, key := range []string{"started_at", "updated_at", "completed_at", "error"} {
		if raw[key] != nil {
			data[key] = raw[key]
		}
	}
	return data
}

func ConfigSummary(config map[string]any) map[string]any {
	agentConfigs, _ := config["agent_configs"].([]any)
	timeConfig, _ := config["time_config"].(map[string]any)
	eventConfig, _ := config["event_config"].(map[string]any)
	initialPosts, _ := eventConfig["initial_posts"].([]any)
	hotTopics, _ := eventConfig["hot_topics"].([]any)

	return map[string]any{
		"total_agents":        len(agentConfigs),
		"simulation_hours":    intValue(timeConfig["total_simulation_hours"]),
		"initial_posts_count": len(initialPosts),
		"hot_topics_count":    len(hotTopics),
		"has_twitter_config":  config["twitter_config"] != nil,
		"has_reddit_config":   config["reddit_config"] != nil,
		"generated_at":        config["generated_at"],
		"llm_model":           config["llm_model"],
	}
}

func BuildRunInstructions(simulationDir, scriptsDir, simulationID string) map[string]any {
	configFile := filepath.Join(simulationDir, "simulation_config.json")
	cleanScripts := filepath.Clean(scriptsDir)
	if abs, err := filepath.Abs(simulationDir); err == nil {
		simulationDir = abs
	}
	if abs, err := filepath.Abs(configFile); err == nil {
		configFile = abs
	}
	if abs, err := filepath.Abs(cleanScripts); err == nil {
		cleanScripts = abs
	}

	return runinstructions.Build(simulationID, simulationDir, configFile, cleanScripts)
}

func paginateMaps(items []map[string]any, limit, offset int) []map[string]any {
	if offset > len(items) {
		return []map[string]any{}
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}

func toAction(item map[string]any, defaultPlatform string) Action {
	return Action{
		RoundNum:   actionRound(item),
		Timestamp:  valueOr(item["timestamp"], ""),
		Platform:   firstNonEmpty(valueOr(item["platform"], ""), defaultPlatform, platformFromPath(item)),
		AgentID:    intValue(item["agent_id"]),
		AgentName:  valueOr(item["agent_name"], ""),
		ActionType: valueOr(item["action_type"], ""),
		ActionArgs: mapValue(item["action_args"]),
		Result:     item["result"],
		Success:    boolValue(item["success"]),
	}
}

func valueOr(value any, fallback string) string {
	if s, ok := value.(string); ok && s != "" {
		return s
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	got, _ := value.(string)
	return strings.TrimSpace(got)
}

func intValue(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return int(parsed)
		}
		if parsed, err := v.Float64(); err == nil {
			return int(parsed)
		}
	}
	return 0
}

func floatValue(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case json.Number:
		if parsed, err := v.Float64(); err == nil {
			return parsed
		}
	}
	return 0
}

func boolValue(value any) bool {
	got, _ := value.(bool)
	return got
}

func mapValue(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func platformFromPath(item map[string]any) string {
	platform := valueOr(item["platform"], "")
	if platform != "" {
		return platform
	}
	return "unknown"
}

func actionRound(item map[string]any) int {
	return intValue(firstNonNil(item["round_num"], item["round"]))
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func cloneMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isNotExist(err error) bool {
	return os.IsNotExist(err)
}

func shouldStopRunState(runState map[string]any) bool {
	switch stringValue(runState["runner_status"]) {
	case "starting", "running", "paused", "stopping":
		return true
	default:
		return false
	}
}

func roundCount(runState map[string]any, actionSets ...[]Action) int {
	if rounds, ok := runState["rounds"].([]any); ok {
		return len(rounds)
	}
	unique := map[int]struct{}{}
	for _, actions := range actionSets {
		for _, action := range actions {
			unique[action.RoundNum] = struct{}{}
		}
	}
	return len(unique)
}

func newSimulationID() (string, error) {
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "sim_" + hex.EncodeToString(raw[:]), nil
}
