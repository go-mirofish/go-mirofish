package simulationhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	intgraph "github.com/go-mirofish/go-mirofish/gateway/internal/graph"
	simulation "github.com/go-mirofish/go-mirofish/gateway/internal/simulation"
	simulationstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/simulation"
	intworker "github.com/go-mirofish/go-mirofish/gateway/internal/worker"
)

type Handler struct {
	service *simulation.Service
	store   *simulationstore.Store
	graph   interface {
		GetGraphData(ctx context.Context, graphID string) (map[string]any, error)
	}
}

func NewHandler(service *simulation.Service, store *simulationstore.Store, graph interface {
	GetGraphData(ctx context.Context, graphID string) (map[string]any, error)
}) *Handler {
	return &Handler{service: service, store: store, graph: graph}
}

func (h *Handler) HandleRoute(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	const prefix = "api/simulation/"
	if !strings.HasPrefix(trimmed, prefix) {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(trimmed, prefix)

	if strings.HasPrefix(rest, "entities/") {
		h.handleEntitiesRoute(w, r, strings.TrimPrefix(rest, "entities/"))
		return
	}

	switch rest {
	case "run":
		h.handleStart(w, r)
		return
	case "create":
		h.handleCreate(w, r)
		return
	case "delete":
		h.handleDelete(w, r)
		return
	case "start":
		h.handleStart(w, r)
		return
	case "stop":
		h.handleStop(w, r)
		return
	case "interview":
		h.handleInterview(w, r)
		return
	case "interview/batch":
		h.handleBatchInterview(w, r)
		return
	case "interview/all":
		h.handleInterviewAll(w, r)
		return
	case "interview/history":
		h.handleInterviewHistory(w, r)
		return
	case "env-status":
		h.handleEnvStatus(w, r)
		return
	case "close-env":
		h.handleCloseEnv(w, r)
		return
	case "list":
		h.handleList(w, r)
		return
	case "history":
		h.handleHistory(w, r)
		return
	}

	if strings.HasPrefix(rest, "script/") {
		h.handleScriptRoute(w, r, rest)
		return
	}

	simulationID, suffix := splitSimulationRoute(rest)
	if simulationID == "" {
		http.NotFound(w, r)
		return
	}

	switch suffix {
	case "":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleSimulationState(w, simulationID)
	case "status", "run-status":
		h.handleRunStatus(w, simulationID)
	case "run-status/detail":
		h.handleRunStatusDetail(w, r, simulationID)
	case "profiles", "profiles/realtime":
		h.handleProfiles(w, simulationID, strings.TrimSpace(r.URL.Query().Get("platform")))
	case "config":
		h.handleConfig(w, simulationID)
	case "config/realtime":
		h.handleConfigRealtime(w, simulationID)
	case "config/download":
		h.HandleConfigDownload(w, r, simulationID)
	case "actions":
		h.HandleActions(w, r, simulationID)
	case "timeline":
		h.HandleTimeline(w, r, simulationID)
	case "agent-stats":
		h.HandleAgentStats(w, r, simulationID)
	case "posts":
		h.HandlePosts(w, r, simulationID)
	case "comments":
		h.HandleComments(w, r, simulationID)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleEntitiesRoute(w http.ResponseWriter, r *http.Request, rest string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.graph == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": "graph provider unavailable"})
		return
	}
	if strings.Contains(rest, "/by-type/") {
		parts := strings.SplitN(rest, "/by-type/", 2)
		graphID, entityType := parts[0], parts[1]
		graphData, err := h.graph.GetGraphData(r.Context(), graphID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
			return
		}
		result := intgraph.FilterEntitiesFromGraphData(graphData, []string{entityType}, true)
		entities, _ := result["entities"].([]map[string]any)
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"entity_type": entityType, "count": len(entities), "entities": entities}})
		return
	}
	if strings.Count(rest, "/") == 1 {
		parts := strings.SplitN(rest, "/", 2)
		graphID, entityUUID := parts[0], parts[1]
		graphData, err := h.graph.GetGraphData(r.Context(), graphID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
			return
		}
		result := intgraph.FilterEntitiesFromGraphData(graphData, nil, true)
		entities, _ := result["entities"].([]map[string]any)
		for _, entity := range entities {
			if stringify(entity["uuid"]) == entityUUID {
				writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": entity})
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "Entity not found: " + entityUUID})
		return
	}

	graphID := strings.TrimSpace(rest)
	var entityTypes []string
	if raw := strings.TrimSpace(r.URL.Query().Get("entity_types")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			if cleaned := strings.TrimSpace(part); cleaned != "" {
				entityTypes = append(entityTypes, cleaned)
			}
		}
	}
	enrich := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("enrich"))) != "false"
	graphData, err := h.graph.GetGraphData(r.Context(), graphID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	result := intgraph.FilterEntitiesFromGraphData(graphData, entityTypes, enrich)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": result})
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req simulation.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.Create(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": resp})
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		SimulationID string `json:"simulation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.Delete(r.Context(), payload.SimulationID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": resp})
}

func (h *Handler) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req intworker.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.Start(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": resp})
}

func (h *Handler) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req intworker.StopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.Stop(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": resp})
}

func (h *Handler) handleInterview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req intworker.InterviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.Interview(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": resp.Success, "data": resp})
}

func (h *Handler) handleBatchInterview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req intworker.BatchInterviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.BatchInterview(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": resp.Success, "data": resp})
}

func (h *Handler) handleInterviewAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req intworker.AllInterviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.InterviewAll(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": resp.Success, "data": resp})
}

func (h *Handler) handleEnvStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req intworker.EnvStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.EnvStatus(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": resp})
}

func (h *Handler) handleCloseEnv(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req intworker.CloseEnvRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.CloseEnv(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": resp.Success, "data": resp})
}

func (h *Handler) handleInterviewHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		SimulationID string `json:"simulation_id"`
		Platform     string `json:"platform"`
		AgentID      *int   `json:"agent_id"`
		Limit        int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	history, err := h.service.InterviewHistory(payload.SimulationID, payload.Platform, payload.AgentID, payload.Limit)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"count": len(history), "history": history}})
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items, err := h.service.List(strings.TrimSpace(r.URL.Query().Get("project_id")))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items, "count": len(items)})
}

func (h *Handler) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.service.History(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items, "count": len(items)})
}

func (h *Handler) HandleActions(w http.ResponseWriter, r *http.Request, simulationID string) {
	limit, offset, platform := commonQuery(r)
	actions, err := h.service.Actions(simulationID, platform, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"count": len(actions), "actions": actions}})
}

func (h *Handler) HandleTimeline(w http.ResponseWriter, r *http.Request, simulationID string) {
	startRound, _ := strconv.Atoi(r.URL.Query().Get("start_round"))
	endRound, _ := strconv.Atoi(r.URL.Query().Get("end_round"))
	timeline, err := h.service.Timeline(simulationID, startRound, endRound)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"rounds_count": len(timeline), "timeline": timeline}})
}

func (h *Handler) HandleAgentStats(w http.ResponseWriter, r *http.Request, simulationID string) {
	stats, err := h.service.AgentStats(simulationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"agents_count": len(stats), "stats": stats}})
}

func (h *Handler) HandlePosts(w http.ResponseWriter, r *http.Request, simulationID string) {
	limit, offset, platform := commonQuery(r)
	if platform == "" {
		platform = "reddit"
	}
	posts, err := h.service.Posts(simulationID, platform, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"platform": platform, "total": len(posts), "count": len(posts), "posts": posts}})
}

func (h *Handler) HandleComments(w http.ResponseWriter, r *http.Request, simulationID string) {
	limit, offset, _ := commonQuery(r)
	comments, err := h.service.Comments(simulationID, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"count": len(comments), "comments": comments}})
}

func (h *Handler) HandleConfigDownload(w http.ResponseWriter, r *http.Request, simulationID string) {
	config, exists, _, err := h.store.ReadConfigWithMeta(simulationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	if !exists || config == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "Config not found"})
		return
	}
	http.ServeFile(w, r, simulationstore.ConfigPath(h.store.SimulationDir(simulationID)))
}

func (h *Handler) HandleScriptDownload(w http.ResponseWriter, r *http.Request, scriptName string) {
	path, err := h.store.ScriptPath(scriptName)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": err.Error()})
		return
	}
	http.ServeFile(w, r, path)
}

func (h *Handler) handleSimulationState(w http.ResponseWriter, simulationID string) {
	state, err := h.store.ReadState(simulationID)
	if err != nil {
		if errorsIsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "Simulation not found: " + simulationID})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	if status, _ := state["status"].(string); status == "ready" {
		state["run_instructions"] = simulation.BuildRunInstructions(h.store.SimulationDir(simulationID), h.store.ScriptsDir, simulationID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": state})
}

func (h *Handler) handleProfiles(w http.ResponseWriter, simulationID, platform string) {
	if platform == "" {
		platform = "reddit"
	}
	profiles, exists, modifiedAt, err := h.store.ReadProfileArtifactsWithMeta(simulationID, platform)
	if err != nil {
		if errorsIsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "Simulation not found: " + simulationID})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	state, _ := h.store.ReadState(simulationID)
	status := stringValue(state["status"])
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"simulation_id":    simulationID,
			"platform":         platform,
			"count":            len(profiles),
			"total_expected":   intValue(state["entities_count"]),
			"is_generating":    status == "preparing",
			"file_exists":      exists,
			"file_modified_at": modifiedAt,
			"profiles":         profiles,
		},
	})
}

func (h *Handler) handleConfig(w http.ResponseWriter, simulationID string) {
	config, _, _, err := h.store.ReadConfigWithMeta(simulationID)
	if err != nil {
		if errorsIsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "Config not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	if config == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "Config not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": config})
}

func (h *Handler) handleConfigRealtime(w http.ResponseWriter, simulationID string) {
	config, exists, modifiedAt, err := h.store.ReadConfigWithMeta(simulationID)
	if err != nil {
		if errorsIsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "Simulation not found: " + simulationID})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	state, _ := h.store.ReadState(simulationID)
	status := stringValue(state["status"])
	data := map[string]any{
		"simulation_id":    simulationID,
		"file_exists":      exists,
		"file_modified_at": modifiedAt,
		"is_generating":    status == "preparing",
		"config_generated": boolValue(state["config_generated"]),
		"config":           config,
	}
	if status == "preparing" {
		data["generation_stage"] = "generating_profiles"
		if intValue(state["profiles_count"]) > 0 {
			data["generation_stage"] = "generating_config"
		}
	} else if status == "ready" {
		data["generation_stage"] = "completed"
	}
	if config != nil {
		data["summary"] = simulation.ConfigSummary(config)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func (h *Handler) handleRunStatus(w http.ResponseWriter, simulationID string) {
	runState, err := h.store.ReadRunState(simulationID)
	if err != nil && !errorsIsNotExist(err) {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": simulation.NormalizeRunStatus(simulationID, runState)})
}

func (h *Handler) handleRunStatusDetail(w http.ResponseWriter, r *http.Request, simulationID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := h.service.RunStatusDetail(simulationID, strings.TrimSpace(r.URL.Query().Get("platform")))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func (h *Handler) handleScriptRoute(w http.ResponseWriter, r *http.Request, rest string) {
	if !strings.HasSuffix(rest, "/download") {
		http.NotFound(w, r)
		return
	}
	scriptName := strings.TrimSuffix(strings.TrimPrefix(rest, "script/"), "/download")
	h.HandleScriptDownload(w, r, scriptName)
}

func splitSimulationRoute(rest string) (string, string) {
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], "/")
}

func errorsIsNotExist(err error) bool {
	return os.IsNotExist(err)
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

func boolValue(value any) bool {
	got, _ := value.(bool)
	return got
}

func commonQuery(r *http.Request) (int, int, string) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	platform := r.URL.Query().Get("platform")
	return limit, offset, platform
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func stringify(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
