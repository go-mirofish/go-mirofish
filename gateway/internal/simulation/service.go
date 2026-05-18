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

	intgovernor "github.com/go-mirofish/go-mirofish/gateway/internal/governor"
	"github.com/go-mirofish/go-mirofish/gateway/internal/runinstructions"
	simulationstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/simulation"
	sovereignstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/sovereign"
	intworker "github.com/go-mirofish/go-mirofish/gateway/internal/worker"
)

var (
	ErrProjectIDRequired    = errors.New("project_id is required")
	ErrSimulationIDRequired = errors.New("simulation_id is required")
	ErrProjectNotFound      = errors.New("project not found")
	ErrGraphNotBuilt        = errors.New("graph not built")
)

type Service struct {
	store    *simulationstore.Store
	bridge   intworker.Bridge
	governor *intgovernor.Service
	now      func() time.Time
}

func NewService(store *simulationstore.Store, bridge intworker.Bridge) *Service {
	return &Service{
		store:  store,
		bridge: bridge,
		now:    time.Now,
	}
}

func NewServiceWithGovernor(store *simulationstore.Store, bridge intworker.Bridge, governor *intgovernor.Service) *Service {
	service := NewService(store, bridge)
	service.governor = governor
	return service
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
	if s.governor != nil && s.governor.Enabled() {
		sovereignState, err := s.governor.InitializeSimulation(context.Background(), simulationID)
		if err != nil {
			return nil, err
		}
		state["sovereign"] = sovereignRuntimeToMap(sovereignState)
		if err := s.store.WriteState(simulationID, state); err != nil {
			return nil, err
		}
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
	if s.governor != nil && s.governor.Enabled() {
		if err := s.governor.DeleteSimulation(ctx, simulationID); err != nil {
			warnings = append(warnings, err.Error())
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
	simulations, err := s.store.ListSimulations(strings.TrimSpace(projectID))
	if err != nil {
		return nil, err
	}
	return s.enrichSimulationSummaries(simulations, 0)
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

	return s.enrichSimulationSummaries(simulations, limit)
}

func (s *Service) enrichSimulationSummaries(simulations []map[string]any, limit int) ([]map[string]any, error) {
	if limit > 0 && len(simulations) > limit {
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
		if sovereignState, err := s.ObservedSovereignRuntime(context.Background(), simulationID); err == nil && sovereignState != nil {
			simCopy["sovereign"] = sovereignState
			ApplySovereignSummaryOverlay(simCopy, sovereignState)
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
	if s.nativeRunnerActive(req.SimulationID) {
		return intworker.StartResponse{}, errors.New("simulation already running")
	}
	if s.governor != nil && s.governor.Enabled() && req.SimulationID != "" {
		state, err := s.ensureSovereignInitialized(ctx, req.SimulationID, true)
		if err != nil {
			return intworker.StartResponse{}, err
		}
		if state.Status == intgovernor.StatusCreated {
			state, err = s.governor.SetStatus(ctx, req.SimulationID, []string{intgovernor.StatusCreated}, intgovernor.StatusReady, "")
			if err != nil {
				return intworker.StartResponse{}, err
			}
			_ = s.syncSovereignControlState(req.SimulationID, state)
		}
		state, err = s.governor.SetStatus(ctx, req.SimulationID, []string{intgovernor.StatusCreated, intgovernor.StatusReady, intgovernor.StatusRunning, intgovernor.StatusStopped, intgovernor.StatusCompleted, intgovernor.StatusFailed}, intgovernor.StatusRunning, "")
		if err != nil {
			return intworker.StartResponse{}, err
		}
		_ = s.syncSovereignControlState(req.SimulationID, state)
	}
	resp, err := s.bridge.StartSimulation(ctx, req)
	if err != nil {
		if s.nativeRunnerActive(req.SimulationID) {
			if s.governor != nil && s.governor.Enabled() && req.SimulationID != "" {
				if state, markErr := s.governor.SetStatus(ctx, req.SimulationID, []string{intgovernor.StatusCreated, intgovernor.StatusReady, intgovernor.StatusRunning, intgovernor.StatusStopped, intgovernor.StatusCompleted, intgovernor.StatusFailed}, intgovernor.StatusRunning, ""); markErr == nil {
					_ = s.syncSovereignControlState(req.SimulationID, state)
				}
			}
			return intworker.StartResponse{}, err
		}
		if s.governor != nil && s.governor.Enabled() && req.SimulationID != "" {
			if state, markErr := s.governor.SetStatus(ctx, req.SimulationID, []string{intgovernor.StatusReady, intgovernor.StatusRunning}, intgovernor.StatusFailed, err.Error()); markErr == nil {
				_ = s.syncSovereignControlState(req.SimulationID, state)
			}
		}
		return intworker.StartResponse{}, err
	}
	return resp, nil
}

func (s *Service) Stop(ctx context.Context, req intworker.StopRequest) (map[string]any, error) {
	if s.bridge == nil {
		return nil, errors.New("simulation worker bridge unavailable")
	}
	req.SimulationID = strings.TrimSpace(req.SimulationID)
	if req.SimulationID == "" {
		return nil, ErrSimulationIDRequired
	}
	if s.governor != nil && s.governor.Enabled() && req.SimulationID != "" {
		if _, err := s.ensureSovereignInitialized(ctx, req.SimulationID, true); err != nil {
			return nil, err
		}
	}
	if !s.nativeRunnerActive(req.SimulationID) {
		runnerStatus := "stopped"
		if _, err := s.bridge.CloseEnv(ctx, intworker.CloseEnvRequest{SimulationID: req.SimulationID}); err != nil {
			return nil, err
		}
		if s.governor != nil && s.governor.Enabled() && req.SimulationID != "" {
			if current, statusErr := s.governor.Status(ctx, req.SimulationID); statusErr == nil {
				switch current.Status {
				case intgovernor.StatusCompleted, intgovernor.StatusFailed:
					runnerStatus = current.Status
				default:
					if state, markErr := s.governor.SetStatus(ctx, req.SimulationID, []string{intgovernor.StatusCreated, intgovernor.StatusReady, intgovernor.StatusRunning, intgovernor.StatusStopped}, intgovernor.StatusStopped, ""); markErr == nil {
						_ = s.syncSovereignControlState(req.SimulationID, state)
						runnerStatus = state.Status
					}
				}
			}
		}
		return map[string]any{"simulation_id": req.SimulationID, "runner_status": runnerStatus}, nil
	}
	resp, err := s.bridge.StopSimulation(ctx, req)
	if err != nil {
		if s.governor != nil && s.governor.Enabled() && req.SimulationID != "" {
			if state, markErr := s.governor.SetStatus(ctx, req.SimulationID, []string{intgovernor.StatusRunning, intgovernor.StatusStopping, intgovernor.StatusReady}, intgovernor.StatusFailed, err.Error()); markErr == nil {
				_ = s.syncSovereignControlState(req.SimulationID, state)
			}
		}
		return nil, err
	}
	if s.governor != nil && s.governor.Enabled() && req.SimulationID != "" {
		if state, markErr := s.governor.SetStatus(ctx, req.SimulationID, []string{intgovernor.StatusRunning, intgovernor.StatusStopping, intgovernor.StatusReady}, intgovernor.StatusStopped, ""); markErr == nil {
			_ = s.syncSovereignControlState(req.SimulationID, state)
		}
	}
	return resp, nil
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

func (s *Service) SovereignStatus(ctx context.Context, simulationID string) (map[string]any, error) {
	if s.governor == nil || !s.governor.Enabled() {
		return nil, nil
	}
	state, err := s.ensureSovereignInitialized(ctx, strings.TrimSpace(simulationID), false)
	if err != nil {
		return nil, err
	}
	data := sovereignRuntimeToMap(state)
	profile := s.governor.Profile()
	data["profile_config"] = map[string]any{
		"name":                profile.Name,
		"tick_interval_ms":    profile.TickIntervalMs,
		"max_parallel_agents": profile.MaxParallelAgents,
		"truth_mode":          profile.TruthMode,
		"compaction_mode":     profile.CompactionMode,
	}
	return data, nil
}

func (s *Service) AdvanceSovereignTick(ctx context.Context, simulationID string) (map[string]any, error) {
	if s.governor == nil || !s.governor.Enabled() {
		return nil, nil
	}
	state, err := s.ensureSovereignInitialized(ctx, strings.TrimSpace(simulationID), true)
	if err != nil {
		return nil, err
	}
	if state.Status == intgovernor.StatusCreated {
		state, err = s.governor.SetStatus(ctx, strings.TrimSpace(simulationID), []string{intgovernor.StatusCreated}, intgovernor.StatusReady, "")
		if err != nil {
			return nil, err
		}
		_ = s.syncSovereignControlState(strings.TrimSpace(simulationID), state)
	}
	if _, err := s.ReconcileSovereignRuntime(ctx, strings.TrimSpace(simulationID)); err != nil && err != sovereignstore.ErrSimulationRuntimeNotFound {
		return nil, err
	}
	if s.nativeRunnerActive(simulationID) {
		return nil, errors.New("sovereign tick is unavailable while native runner is active")
	}
	state, err = s.governor.AdvanceTick(ctx, strings.TrimSpace(simulationID))
	if err != nil {
		return nil, err
	}
	_ = s.syncSovereignControlState(simulationID, state)
	if _, err := s.governor.DecayClaims(ctx, strings.TrimSpace(simulationID), s.now()); err != nil {
		return nil, err
	}
	return sovereignRuntimeToMap(state), nil
}

func (s *Service) SovereignTruth(ctx context.Context, simulationID string) ([]map[string]any, error) {
	if s.governor == nil || !s.governor.Enabled() {
		return nil, nil
	}
	if _, err := s.ensureSovereignInitialized(ctx, strings.TrimSpace(simulationID), false); err != nil {
		if err == sovereignstore.ErrSimulationRuntimeNotFound {
			if _, stateErr := s.store.ReadState(strings.TrimSpace(simulationID)); stateErr == nil {
				return []map[string]any{}, nil
			}
		}
		return nil, err
	}
	items, err := s.governor.ObserveTruthClaims(ctx, strings.TrimSpace(simulationID), s.now())
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"simulation_id": item.SimulationID,
			"claim_id":      item.ClaimID,
			"claim_type":    item.ClaimType,
			"subject":       item.Subject,
			"source":        item.Source,
			"source_kind":   item.SourceKind,
			"claim_text":    item.ClaimText,
			"truth_status":  publicTruthStatus(item.TruthStatus),
			"confidence":    item.Confidence,
			"evidence_refs": item.EvidenceRefs,
			"version":       item.Version,
			"valid_from":    item.ValidFrom,
			"valid_to":      item.ValidTo,
			"decay_at":      item.DecayAt,
			"updated_at":    item.UpdatedAt,
			"updated_by":    item.UpdatedBy,
		})
	}
	return out, nil
}

func (s *Service) RecordSovereignTruth(ctx context.Context, simulationID string, payload map[string]any) (map[string]any, error) {
	if s.governor == nil || !s.governor.Enabled() {
		return nil, nil
	}
	if strings.TrimSpace(stringValue(payload["truth_status"])) != "" {
		return nil, errors.New("truth_status is Governor-owned and may not be set by the caller")
	}
	if payload["confidence"] != nil {
		return nil, errors.New("confidence is Governor-owned and may not be set by the caller")
	}
	if strings.TrimSpace(stringValue(payload["decay_at"])) != "" {
		return nil, errors.New("decay_at is Governor-owned and may not be set by the caller")
	}
	if strings.TrimSpace(stringValue(payload["valid_to"])) != "" {
		return nil, errors.New("valid_to is Governor-owned and may not be set by the caller")
	}
	if _, err := s.ensureSovereignInitialized(ctx, strings.TrimSpace(simulationID), true); err != nil {
		return nil, err
	}
	if _, exists := payload["truth_status"]; exists {
		return nil, errors.New("truth_status is Governor-owned and may not be set by the caller")
	}
	if _, exists := payload["confidence"]; exists {
		return nil, errors.New("confidence is Governor-owned and may not be set by the caller")
	}
	claim, err := s.governor.RecordClaim(ctx, strings.TrimSpace(simulationID), intgovernor.ClaimInput{
		ClaimID:      stringValue(payload["claim_id"]),
		ClaimType:    stringValue(payload["claim_type"]),
		Subject:      stringValue(payload["subject"]),
		Source:       stringValue(payload["source"]),
		SourceKind:   stringValue(payload["source_kind"]),
		ClaimText:    stringValue(payload["claim_text"]),
		EvidenceRefs: stringSliceValue(payload["evidence_refs"]),
		ValidFrom:    stringValue(payload["valid_from"]),
		ValidTo:      stringValue(payload["valid_to"]),
		DecayAt:      stringValue(payload["decay_at"]),
		UpdatedBy:    stringValue(payload["updated_by"]),
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"simulation_id": claim.SimulationID,
		"claim_id":      claim.ClaimID,
		"claim_type":    claim.ClaimType,
		"subject":       claim.Subject,
		"source":        claim.Source,
		"source_kind":   claim.SourceKind,
		"claim_text":    claim.ClaimText,
		"truth_status":  publicTruthStatus(claim.TruthStatus),
		"confidence":    claim.Confidence,
		"evidence_refs": claim.EvidenceRefs,
		"version":       claim.Version,
		"valid_from":    claim.ValidFrom,
		"valid_to":      claim.ValidTo,
		"decay_at":      claim.DecayAt,
		"updated_at":    claim.UpdatedAt,
		"updated_by":    claim.UpdatedBy,
	}, nil
}

func (s *Service) SovereignMemory(ctx context.Context, simulationID string) ([]map[string]any, error) {
	if s.governor == nil || !s.governor.Enabled() {
		return nil, nil
	}
	if _, err := s.ensureSovereignInitialized(ctx, strings.TrimSpace(simulationID), false); err != nil {
		if err == sovereignstore.ErrSimulationRuntimeNotFound {
			if _, stateErr := s.store.ReadState(strings.TrimSpace(simulationID)); stateErr == nil {
				return []map[string]any{}, nil
			}
		}
		return nil, err
	}
	items, err := s.governor.ListMemorySummaries(ctx, strings.TrimSpace(simulationID))
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"simulation_id": item.SimulationID,
			"summary_id":    item.SummaryID,
			"scope":         item.Scope,
			"start_tick":    item.StartTick,
			"end_tick":      item.EndTick,
			"content":       item.Content,
			"created_at":    item.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) CompactSovereignMemory(ctx context.Context, simulationID string) (map[string]any, error) {
	if s.governor == nil || !s.governor.Enabled() {
		return nil, nil
	}
	if _, err := s.ensureSovereignInitialized(ctx, strings.TrimSpace(simulationID), true); err != nil {
		return nil, err
	}
	if _, err := s.ReconcileSovereignRuntime(ctx, strings.TrimSpace(simulationID)); err != nil && err != sovereignstore.ErrSimulationRuntimeNotFound {
		return nil, err
	}
	summary, err := s.governor.Compact(ctx, strings.TrimSpace(simulationID))
	if err != nil {
		return nil, err
	}
	if _, err := s.governor.DecayClaims(ctx, strings.TrimSpace(simulationID), s.now()); err != nil {
		return nil, err
	}
	return map[string]any{
		"simulation_id": summary.SimulationID,
		"summary_id":    summary.SummaryID,
		"scope":         summary.Scope,
		"start_tick":    summary.StartTick,
		"end_tick":      summary.EndTick,
		"content":       summary.Content,
		"created_at":    summary.CreatedAt,
	}, nil
}

func (s *Service) ReconcileSovereignRuntime(ctx context.Context, simulationID string) (map[string]any, error) {
	if s.governor == nil || !s.governor.Enabled() {
		return nil, nil
	}
	if _, err := s.ensureSovereignInitialized(ctx, strings.TrimSpace(simulationID), false); err != nil {
		return nil, err
	}
	target := ""
	lastError := ""
	activeRunner := s.nativeRunnerActive(simulationID)
	if runState, err := s.store.ReadRunState(strings.TrimSpace(simulationID)); err == nil {
		normalized := NormalizeRunStatus(simulationID, runState)
		switch stringValue(normalized["runner_status"]) {
		case "running", "starting":
			if activeRunner {
				target = intgovernor.StatusRunning
			}
		case "completed":
			target = intgovernor.StatusCompleted
		case "failed":
			target = intgovernor.StatusFailed
		case "stopped":
			target = intgovernor.StatusStopped
		}
		if target != "" {
			lastError = stringValue(runState["error"])
		}
	}
	runtimeState, err := s.store.ReadRuntimeState(strings.TrimSpace(simulationID))
	if err != nil {
		if isNotExist(err) {
			return s.SovereignStatus(ctx, simulationID)
		}
		return nil, err
	}
	switch stringValue(runtimeState["status"]) {
	case "completed":
		if !activeRunner {
			target = intgovernor.StatusCompleted
		}
	case "failed":
		if !activeRunner {
			target = intgovernor.StatusFailed
		}
	case "stopped":
		if !activeRunner {
			target = intgovernor.StatusStopped
		}
	case "running":
		if target == "" && activeRunner {
			target = intgovernor.StatusRunning
		}
	}
	if target != "" {
		if lastError == "" {
			lastError = stringValue(runtimeState["error"])
		}
		current, err := s.governor.Status(ctx, simulationID)
		if err == nil {
			if shouldPreferGovernorRunning(current.UpdatedAt, current.Status, target, runStateTimestamp(runtimeState)) {
				target = current.Status
				lastError = current.LastError
			}
			current.Status = target
			current.LastError = lastError
			if runState, err := s.store.ReadRunState(strings.TrimSpace(simulationID)); err == nil {
				current.CurrentTick = intValue(NormalizeRunStatus(simulationID, runState)["current_round"])
			} else if intValue(runtimeState["current_round"]) > current.CurrentTick {
				current.CurrentTick = intValue(runtimeState["current_round"])
			}
			state, syncErr := s.governor.SyncRuntimeState(ctx, current)
			if syncErr == nil {
				_ = s.syncSovereignControlState(simulationID, state)
			}
		}
	}
	return s.SovereignStatus(ctx, simulationID)
}

func (s *Service) ObservedSovereignRuntime(ctx context.Context, simulationID string) (map[string]any, error) {
	if s.governor == nil || !s.governor.Enabled() {
		return nil, nil
	}
	base, err := s.SovereignStatus(ctx, simulationID)
	if err != nil {
		if err != sovereignstore.ErrSimulationRuntimeNotFound {
			return nil, err
		}
		state, stateErr := s.store.ReadState(simulationID)
		if stateErr != nil {
			if isNotExist(stateErr) {
				return nil, sovereignstore.ErrSimulationRuntimeNotFound
			}
			return nil, stateErr
		}
		baseStatus := stringValue(state["status"])
		if s.nativeRunnerActive(simulationID) {
			switch baseStatus {
			case intgovernor.StatusCompleted, intgovernor.StatusFailed, intgovernor.StatusStopped:
				baseStatus = intgovernor.StatusRunning
			}
		}
		base = map[string]any{
			"simulation_id": simulationID,
			"mode":          intgovernor.ModeSovereign,
			"profile":       s.governor.Profile().Name,
			"status":        baseStatus,
			"current_tick":  intValue(state["current_round"]),
			"last_tick_at":  "",
			"last_error":    "",
			"created_at":    stringValue(state["created_at"]),
			"updated_at":    stringValue(state["updated_at"]),
			"profile_config": map[string]any{
				"name":                s.governor.Profile().Name,
				"tick_interval_ms":    s.governor.Profile().TickIntervalMs,
				"max_parallel_agents": s.governor.Profile().MaxParallelAgents,
				"truth_mode":          s.governor.Profile().TruthMode,
				"compaction_mode":     s.governor.Profile().CompactionMode,
			},
		}
	}
	if base == nil {
		return nil, nil
	}
	out := cloneMap(base)
	target := ""
	lastError := ""
	activeRunner := s.nativeRunnerActive(simulationID)
	if runState, err := s.store.ReadRunState(strings.TrimSpace(simulationID)); err == nil {
		normalized := NormalizeRunStatus(simulationID, runState)
		switch stringValue(normalized["runner_status"]) {
		case "running", "starting":
			if activeRunner {
				target = intgovernor.StatusRunning
			}
		case "completed":
			target = intgovernor.StatusCompleted
		case "failed":
			target = intgovernor.StatusFailed
		case "stopped":
			target = intgovernor.StatusStopped
		}
		out["current_tick"] = intValue(normalized["current_round"])
		lastError = stringValue(runState["error"])
	}
	if runtimeState, err := s.store.ReadRuntimeState(strings.TrimSpace(simulationID)); err == nil {
		switch stringValue(runtimeState["status"]) {
		case "completed":
			if !activeRunner {
				target = intgovernor.StatusCompleted
			}
		case "failed":
			if !activeRunner {
				target = intgovernor.StatusFailed
			}
		case "stopped":
			if !activeRunner {
				target = intgovernor.StatusStopped
			}
		case "running":
			if target == "" && activeRunner {
				target = intgovernor.StatusRunning
			}
		}
		if lastError == "" {
			lastError = stringValue(runtimeState["error"])
		}
		if intValue(runtimeState["current_round"]) > intValue(out["current_tick"]) {
			out["current_tick"] = intValue(runtimeState["current_round"])
		}
		if shouldPreferGovernorRunning(stringValue(base["updated_at"]), stringValue(base["status"]), target, runStateTimestamp(runtimeState)) {
			target = stringValue(base["status"])
			lastError = stringValue(base["last_error"])
		}
	}
	if target != "" {
		out["status"] = target
		if lastError != "" {
			out["last_error"] = lastError
		}
	}
	return out, nil
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
		"simulation_id":           firstNonEmpty(stringValue(raw["simulation_id"]), simulationID),
		"runner_status":           firstNonEmpty(stringValue(raw["runner_status"]), "idle"),
		"current_round":           intValue(raw["current_round"]),
		"total_rounds":            intValue(raw["total_rounds"]),
		"progress_percent":        floatValue(raw["progress_percent"]),
		"simulated_hours":         intValue(raw["simulated_hours"]),
		"total_simulation_hours":  intValue(raw["total_simulation_hours"]),
		"twitter_current_round":   intValue(firstNonNil(raw["twitter_current_round"], raw["twitter_round"])),
		"reddit_current_round":    intValue(firstNonNil(raw["reddit_current_round"], raw["reddit_round"])),
		"twitter_simulated_hours": intValue(raw["twitter_simulated_hours"]),
		"reddit_simulated_hours":  intValue(raw["reddit_simulated_hours"]),
		"twitter_running":         boolValue(raw["twitter_running"]),
		"reddit_running":          boolValue(raw["reddit_running"]),
		"twitter_completed":       boolValue(raw["twitter_completed"]),
		"reddit_completed":        boolValue(raw["reddit_completed"]),
		"twitter_actions_count":   intValue(raw["twitter_actions_count"]),
		"reddit_actions_count":    intValue(raw["reddit_actions_count"]),
		"total_actions_count":     intValue(raw["total_actions_count"]),
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

func stringSliceValue(value any) []string {
	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s := stringValue(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{strings.TrimSpace(v)}
	default:
		return nil
	}
}

func publicTruthStatus(status string) string {
	switch status {
	case intgovernor.StatusObserved:
		return intgovernor.StatusSpeculative
	default:
		return status
	}
}

func shouldStopRunState(runState map[string]any) bool {
	switch stringValue(runState["runner_status"]) {
	case "starting", "running", "paused", "stopping":
		return true
	default:
		return false
	}
}

func (s *Service) NativeRunnerActiveStatus(simulationID string) bool {
	return s.nativeRunnerActive(simulationID)
}

func (s *Service) nativeRunnerActive(simulationID string) bool {
	type sessionAlive interface {
		SessionAlive(string) bool
	}
	if bridge, ok := s.bridge.(sessionAlive); ok && !bridge.SessionAlive(strings.TrimSpace(simulationID)) {
		return false
	}
	if runState, err := s.store.ReadRunState(strings.TrimSpace(simulationID)); err == nil {
		return shouldStopRunState(runState)
	}
	if runtimeState, err := s.store.ReadRuntimeState(strings.TrimSpace(simulationID)); err == nil {
		switch stringValue(runtimeState["status"]) {
		case "starting", "running", "paused", "stopping":
			return true
		}
	}
	return false
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

func sovereignRuntimeToMap(state sovereignstore.RuntimeState) map[string]any {
	return map[string]any{
		"simulation_id": state.SimulationID,
		"mode":          state.Mode,
		"profile":       state.Profile,
		"status":        state.Status,
		"current_tick":  state.CurrentTick,
		"last_tick_at":  state.LastTickAt,
		"last_error":    state.LastError,
		"created_at":    state.CreatedAt,
		"updated_at":    state.UpdatedAt,
	}
}

func ApplySovereignSummaryOverlay(payload map[string]any, sovereignState map[string]any) {
	if payload == nil || sovereignState == nil {
		return
	}
	shouldPromote := false
	switch stringValue(payload["status"]) {
	case "ready", "running", "stopped", "completed", "failed":
		shouldPromote = true
	case "created":
		switch stringValue(sovereignState["status"]) {
		case "running", "stopped", "completed", "failed":
			shouldPromote = true
		default:
			if intValue(sovereignState["current_tick"]) > 0 {
				shouldPromote = true
			}
		}
	}
	if !shouldPromote {
		return
	}
	if status, ok := sovereignState["status"]; ok {
		payload["status"] = status
	}
	if tick, ok := sovereignState["current_tick"]; ok {
		if intValue(payload["current_round"]) <= intValue(tick) {
			payload["current_round"] = tick
		}
	}
	if updatedAt, ok := sovereignState["updated_at"]; ok {
		payload["updated_at"] = updatedAt
	}
}

func shouldPreferGovernorRunning(currentUpdatedAt, currentStatus, targetStatus, runtimeTimestamp string) bool {
	if currentStatus != intgovernor.StatusRunning {
		return false
	}
	switch targetStatus {
	case intgovernor.StatusCompleted, intgovernor.StatusFailed, intgovernor.StatusStopped:
	default:
		return false
	}
	if runtimeTimestamp == "" {
		return false
	}
	currentTime, err1 := time.Parse(time.RFC3339, currentUpdatedAt)
	runtimeTime, err2 := time.Parse(time.RFC3339, runtimeTimestamp)
	if err1 != nil || err2 != nil {
		return false
	}
	return runtimeTime.Before(currentTime)
}

func runStateTimestamp(payload map[string]any) string {
	for _, key := range []string{"updated_at", "completed_at", "timestamp", "started_at"} {
		if value := stringValue(payload[key]); value != "" {
			return value
		}
	}
	return ""
}

func (s *Service) ensureSovereignInitialized(ctx context.Context, simulationID string, create bool) (sovereignstore.RuntimeState, error) {
	state, err := s.governor.Status(ctx, simulationID)
	if err == nil {
		return state, nil
	}
	if err != sovereignstore.ErrSimulationRuntimeNotFound {
		return sovereignstore.RuntimeState{}, err
	}
	if !create {
		return sovereignstore.RuntimeState{}, sovereignstore.ErrSimulationRuntimeNotFound
	}
	if _, err := s.store.ReadState(simulationID); err != nil {
		if isNotExist(err) {
			return sovereignstore.RuntimeState{}, sovereignstore.ErrSimulationRuntimeNotFound
		}
		return sovereignstore.RuntimeState{}, err
	}
	seed, err := s.legacySovereignSeed(simulationID)
	if err != nil {
		return sovereignstore.RuntimeState{}, err
	}
	return s.governor.AdoptSimulation(ctx, seed)
}

func (s *Service) legacySovereignSeed(simulationID string) (sovereignstore.RuntimeState, error) {
	state, err := s.store.ReadState(simulationID)
	if err != nil {
		return sovereignstore.RuntimeState{}, err
	}
	seedStatus := firstNonEmpty(stringValue(state["status"]), intgovernor.StatusCreated)
	if s.nativeRunnerActive(simulationID) {
		switch seedStatus {
		case intgovernor.StatusCompleted, intgovernor.StatusFailed, intgovernor.StatusStopped:
			seedStatus = intgovernor.StatusRunning
		}
	}
	seed := sovereignstore.RuntimeState{
		SimulationID: simulationID,
		Mode:         intgovernor.ModeSovereign,
		Profile:      s.governor.Profile().Name,
		Status:       seedStatus,
		CurrentTick:  intValue(state["current_round"]),
		CreatedAt:    stringValue(state["created_at"]),
		UpdatedAt:    stringValue(state["updated_at"]),
	}
	if runState, err := s.store.ReadRunState(simulationID); err == nil {
		normalized := NormalizeRunStatus(simulationID, runState)
		seed.CurrentTick = intValue(normalized["current_round"])
		switch stringValue(normalized["runner_status"]) {
		case "running", "starting":
			seed.Status = intgovernor.StatusRunning
		case "completed":
			seed.Status = intgovernor.StatusCompleted
		case "failed":
			seed.Status = intgovernor.StatusFailed
		case "stopped":
			seed.Status = intgovernor.StatusStopped
		}
		seed.LastError = stringValue(runState["error"])
	}
	if runtimeState, err := s.store.ReadRuntimeState(simulationID); err == nil {
		activeRunner := s.nativeRunnerActive(simulationID)
		switch stringValue(runtimeState["status"]) {
		case "running":
			if seed.Status == "" || seed.Status == intgovernor.StatusCreated || seed.Status == intgovernor.StatusReady {
				seed.Status = intgovernor.StatusRunning
			}
		case "completed":
			if !activeRunner {
				seed.Status = intgovernor.StatusCompleted
			}
		case "failed":
			if !activeRunner {
				seed.Status = intgovernor.StatusFailed
			}
		case "stopped":
			if !activeRunner {
				seed.Status = intgovernor.StatusStopped
			}
		}
		if seed.LastError == "" {
			seed.LastError = stringValue(runtimeState["error"])
		}
		if intValue(runtimeState["current_round"]) > seed.CurrentTick {
			seed.CurrentTick = intValue(runtimeState["current_round"])
		}
	}
	if seed.CreatedAt == "" {
		seed.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if seed.UpdatedAt == "" {
		seed.UpdatedAt = seed.CreatedAt
	}
	if seed.Status == "" {
		seed.Status = intgovernor.StatusCreated
	}
	return seed, nil
}

func (s *Service) syncSovereignControlState(simulationID string, state sovereignstore.RuntimeState) error {
	payload, err := s.store.ReadState(simulationID)
	if err != nil {
		if !isNotExist(err) {
			return err
		}
		payload = map[string]any{
			"simulation_id": simulationID,
		}
	}
	if payload["project_id"] == nil {
		payload["project_id"] = ""
	}
	if payload["graph_id"] == nil {
		payload["graph_id"] = ""
	}
	payload["sovereign"] = sovereignRuntimeToMap(state)
	return s.store.WriteState(simulationID, payload)
}

func newSimulationID() (string, error) {
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "sim_" + hex.EncodeToString(raw[:]), nil
}
