package preparehttp

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	intgraph "github.com/go-mirofish/go-mirofish/gateway/internal/graph"
	intontology "github.com/go-mirofish/go-mirofish/gateway/internal/ontology"
	intprovider "github.com/go-mirofish/go-mirofish/gateway/internal/provider"
	localfs "github.com/go-mirofish/go-mirofish/gateway/internal/store/localfs"
)

type JSONFunc func(http.ResponseWriter, int, any)

type GraphService interface {
	GetGraphData(ctx context.Context, graphID string) (map[string]any, error)
}

type Handler struct {
	store     *localfs.Store
	graph     GraphService
	writeJSON JSONFunc
}

var newProviderExecutor = func(cfg intprovider.Config) intprovider.Executor {
	return intprovider.NewExecutor(cfg, nil)
}

func NewHandler(store *localfs.Store, graph GraphService, writeJSON JSONFunc) *Handler {
	return &Handler{store: store, graph: graph, writeJSON: writeJSON}
}

func (h *Handler) HandleGenerateProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	graphID := strings.TrimSpace(fmt.Sprint(payload["graph_id"]))
	if graphID == "" || graphID == "<nil>" {
		h.writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "Graph ID is required"})
		return
	}
	platform := strings.TrimSpace(fmt.Sprint(payload["platform"]))
	if platform == "" || platform == "<nil>" {
		platform = "reddit"
	}
	entityTypes := normalizeStringList(payload["entity_types"])
	graphData, err := h.graph.GetGraphData(r.Context(), graphID)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	filtered := intgraph.FilterEntitiesFromGraphData(graphData, entityTypes, true)
	entities := mapSlice(filtered["entities"])
	if len(entities) == 0 {
		h.writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "No matching entities"})
		return
	}
	useLLM, _ := payload["use_llm"].(bool)
	profiles, err := h.generateProfilesViaGo(r.Context(), entities, platform, useLLM)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"platform":     platform,
			"entity_types": filtered["entity_types"],
			"count":        len(profiles),
			"profiles":     profiles,
		},
	})
}

func (h *Handler) HandlePrepare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	simulationID := strings.TrimSpace(fmt.Sprint(payload["simulation_id"]))
	if simulationID == "" || simulationID == "<nil>" {
		h.writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "Simulation ID is required"})
		return
	}
	forceRegenerate, _ := payload["force_regenerate"].(bool)
	entityTypes := normalizeStringList(payload["entity_types"])
	state, err := h.store.ReadSimulation(simulationID)
	if err != nil {
		if os.IsNotExist(err) {
			h.writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "Simulation not found: " + simulationID})
			return
		}
		h.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	projectID, _ := state["project_id"].(string)
	project, err := h.store.ReadProject(projectID)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	simulationRequirement, _ := project["simulation_requirement"].(string)
	if simulationRequirement == "" {
		h.writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "Project simulation requirement is missing"})
		return
	}
	if !forceRegenerate && h.isPrepared(simulationID) {
		h.writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
			"simulation_id":    simulationID,
			"status":           "ready",
			"message":          "Simulation is already prepared",
			"already_prepared": true,
		}})
		return
	}
	taskID := h.createTask("simulation_prepare", map[string]any{"simulation_id": simulationID, "project_id": projectID})
	state["status"] = "preparing"
	state["updated_at"] = time.Now().Format(time.RFC3339)
	_ = h.store.WriteSimulation(simulationID, state)
	useLLM, _ := payload["use_llm_for_profiles"].(bool)
	go h.runTask(taskID, simulationID, simulationRequirement, entityTypes, useLLM)
	h.writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"simulation_id": simulationID,
		"task_id":       taskID,
		"status":        "preparing",
		"message":       "Simulation prepare started",
	}})
}

func (h *Handler) HandlePrepareStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	simulationID := strings.TrimSpace(fmt.Sprint(payload["simulation_id"]))
	taskID := strings.TrimSpace(fmt.Sprint(payload["task_id"]))
	if simulationID != "" && simulationID != "<nil>" && h.isPrepared(simulationID) {
		h.writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
			"simulation_id":    simulationID,
			"status":           "ready",
			"progress":         100,
			"message":          "Simulation is ready",
			"already_prepared": true,
		}})
		return
	}
	if taskID == "" || taskID == "<nil>" {
		h.writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "Task ID or simulation ID is required"})
		return
	}
	task, err := h.store.ReadTask(taskID)
	if err != nil {
		if os.IsNotExist(err) {
			h.writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "Task not found: " + taskID})
			return
		}
		h.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": task})
}

func (h *Handler) runTask(taskID, simulationID, simulationRequirement string, entityTypes []string, useLLM bool) {
	ctx := context.Background()
	h.updateTask(taskID, map[string]any{"status": "processing", "progress": 0, "message": "Preparing environment"})
	state, err := h.store.ReadSimulation(simulationID)
	if err != nil {
		h.failTask(taskID, err.Error())
		return
	}
	graphID, _ := state["graph_id"].(string)
	graphData, err := h.graph.GetGraphData(ctx, graphID)
	if err != nil {
		state["status"] = "failed"
		state["error"] = err.Error()
		state["updated_at"] = time.Now().Format(time.RFC3339)
		_ = h.store.WriteSimulation(simulationID, state)
		h.failTask(taskID, err.Error())
		return
	}
	filtered := intgraph.FilterEntitiesFromGraphData(graphData, entityTypes, true)
	entities := mapSlice(filtered["entities"])
	if len(entities) == 0 {
		err = fmt.Errorf("no matching entities")
		state["status"] = "failed"
		state["error"] = err.Error()
		state["updated_at"] = time.Now().Format(time.RFC3339)
		_ = h.store.WriteSimulation(simulationID, state)
		h.failTask(taskID, err.Error())
		return
	}
	h.updateTask(taskID, map[string]any{"status": "processing", "progress": 25, "message": "Generating profiles"})
	redditProfiles, err := h.generateProfilesViaGo(ctx, entities, "reddit", useLLM)
	if err != nil {
		state["status"] = "failed"
		state["error"] = err.Error()
		state["updated_at"] = time.Now().Format(time.RFC3339)
		_ = h.store.WriteSimulation(simulationID, state)
		h.failTask(taskID, err.Error())
		return
	}
	twitterProfiles, err := h.generateProfilesViaGo(ctx, entities, "twitter", useLLM)
	if err != nil {
		state["status"] = "failed"
		state["error"] = err.Error()
		state["updated_at"] = time.Now().Format(time.RFC3339)
		_ = h.store.WriteSimulation(simulationID, state)
		h.failTask(taskID, err.Error())
		return
	}
	if err := h.writeSimulationProfiles(simulationID, redditProfiles, twitterProfiles); err != nil {
		state["status"] = "failed"
		state["error"] = err.Error()
		state["updated_at"] = time.Now().Format(time.RFC3339)
		_ = h.store.WriteSimulation(simulationID, state)
		h.failTask(taskID, err.Error())
		return
	}
	h.updateTask(taskID, map[string]any{"status": "processing", "progress": 70, "message": "Generating simulation config"})
	config, err := h.generateSimulationConfigViaGo(ctx, simulationID, state, simulationRequirement, entities, useLLM)
	if err != nil {
		state["status"] = "failed"
		state["error"] = err.Error()
		state["updated_at"] = time.Now().Format(time.RFC3339)
		_ = h.store.WriteSimulation(simulationID, state)
		h.failTask(taskID, err.Error())
		return
	}
	if err := h.writeSimulationConfig(simulationID, config); err != nil {
		state["status"] = "failed"
		state["error"] = err.Error()
		state["updated_at"] = time.Now().Format(time.RFC3339)
		_ = h.store.WriteSimulation(simulationID, state)
		h.failTask(taskID, err.Error())
		return
	}
	state["status"] = "ready"
	state["entities_count"] = len(entities)
	state["profiles_count"] = len(redditProfiles)
	state["entity_types"] = filtered["entity_types"]
	state["config_generated"] = true
	state["config_reasoning"] = "Go deterministic prepare pipeline"
	state["updated_at"] = time.Now().Format(time.RFC3339)
	state["error"] = nil
	_ = h.store.WriteSimulation(simulationID, state)
	h.completeTask(taskID, map[string]any{
		"simulation_id":    simulationID,
		"status":           "ready",
		"entities_count":   len(entities),
		"profiles_count":   len(redditProfiles),
		"entity_types":     filtered["entity_types"],
		"config_generated": true,
	})
}

func (h *Handler) isPrepared(simulationID string) bool {
	for _, name := range []string{"state.json", "simulation_config.json", "reddit_profiles.json", "twitter_profiles.csv"} {
		if _, err := os.Stat(filepath.Join(h.store.SimulationDir(simulationID), name)); err != nil {
			return false
		}
	}
	state, err := h.store.ReadSimulation(simulationID)
	if err != nil {
		return false
	}
	status, _ := state["status"].(string)
	return strings.TrimSpace(status) == "ready" || boolValue(state["config_generated"])
}

func (h *Handler) createTask(taskType string, metadata map[string]any) string {
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().Format(time.RFC3339)
	payload := map[string]any{
		"task_id":         taskID,
		"task_type":       taskType,
		"status":          "pending",
		"created_at":      now,
		"updated_at":      now,
		"progress":        0,
		"message":         "",
		"progress_detail": map[string]any{},
		"result":          nil,
		"error":           nil,
		"metadata":        metadata,
	}
	_ = h.store.WriteTask(taskID, payload)
	return taskID
}

func (h *Handler) updateTask(taskID string, updates map[string]any) {
	task, err := h.store.ReadTask(taskID)
	if err != nil {
		return
	}
	for key, value := range updates {
		task[key] = value
	}
	task["updated_at"] = time.Now().Format(time.RFC3339)
	_ = h.store.WriteTask(taskID, task)
}

func (h *Handler) failTask(taskID, message string) {
	h.updateTask(taskID, map[string]any{"status": "failed", "message": "Task failed", "error": message})
}

func (h *Handler) completeTask(taskID string, result map[string]any) {
	h.updateTask(taskID, map[string]any{"status": "completed", "progress": 100, "message": "Task completed", "result": result})
}

func (h *Handler) writeSimulationProfiles(simulationID string, redditProfiles, twitterProfiles []map[string]any) error {
	simDir := h.store.SimulationDir(simulationID)
	if err := os.MkdirAll(simDir, 0o755); err != nil {
		return err
	}
	redditRaw, err := json.MarshalIndent(redditProfiles, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(simDir, "reddit_profiles.json"), redditRaw, 0o644); err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(simDir, "twitter_profiles.csv"))
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	headers := []string{"user_id", "username", "name", "bio", "persona", "friend_count", "follower_count", "statuses_count", "created_at", "age", "gender", "mbti", "country", "profession"}
	if err := writer.Write(headers); err != nil {
		return err
	}
	for _, profile := range twitterProfiles {
		row := []string{
			fmt.Sprint(profile["user_id"]),
			fmt.Sprint(profile["username"]),
			fmt.Sprint(profile["name"]),
			fmt.Sprint(profile["bio"]),
			fmt.Sprint(profile["persona"]),
			fmt.Sprint(profile["friend_count"]),
			fmt.Sprint(profile["follower_count"]),
			fmt.Sprint(profile["statuses_count"]),
			fmt.Sprint(profile["created_at"]),
			fmt.Sprint(profile["age"]),
			fmt.Sprint(profile["gender"]),
			fmt.Sprint(profile["mbti"]),
			fmt.Sprint(profile["country"]),
			fmt.Sprint(profile["profession"]),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func (h *Handler) writeSimulationConfig(simulationID string, payload map[string]any) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.store.SimulationConfigPath(simulationID), raw, 0o644)
}

type providerContentGenerator struct {
	exec  intprovider.Executor
	model string
}

func (p providerContentGenerator) Execute(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	if p.exec == nil {
		return "", fmt.Errorf("provider executor is not configured")
	}
	resp, err := p.exec.Execute(ctx, intprovider.ProviderRequest{
		Model: p.model,
		Messages: []intprovider.Message{
			{Role: intprovider.RoleSystem, Content: systemPrompt},
			{Role: intprovider.RoleUser, Content: userPrompt},
		},
		Temperature:    0,
		MaxTokens:      4096,
		ResponseFormat: &intprovider.ResponseFormat{Type: intprovider.ResponseFormatJSONObject},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (h *Handler) providerExecutor() intprovider.Executor {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("LLM_BASE_URL")), "/")
	apiKey := strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	model := strings.TrimSpace(os.Getenv("LLM_MODEL_NAME"))
	if baseURL == "" || apiKey == "" || model == "" {
		return nil
	}
	return newProviderExecutor(intprovider.Config{
		BaseURL:      baseURL,
		APIKey:       apiKey,
		DefaultModel: model,
		ProviderName: "openai-compatible",
	})
}

func (h *Handler) generateProfilesViaGo(ctx context.Context, entities []map[string]any, platform string, useLLM bool) ([]map[string]any, error) {
	model := strings.TrimSpace(os.Getenv("LLM_MODEL_NAME"))
	var gen intontology.ContentGenerator
	if useLLM {
		if exec := h.providerExecutor(); exec != nil {
			gen = providerContentGenerator{exec: exec, model: model}
		}
	}
	profileGen := intontology.NewProfileGenerator(gen, model)
	input := make([]intontology.Entity, 0, len(entities))
	for _, entity := range entities {
		input = append(input, intontology.Entity{
			UUID:       fmt.Sprint(entity["uuid"]),
			Name:       fmt.Sprint(entity["name"]),
			Labels:     toStringSlice(entity["labels"]),
			Summary:    strings.TrimSpace(fmt.Sprint(entity["summary"])),
			Attributes: toStringAnyMap(entity["attributes"]),
		})
	}
	profiles, err := profileGen.Generate(ctx, input, platform)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(profiles))
	for _, profile := range profiles {
		raw, _ := json.Marshal(profile)
		var payload map[string]any
		_ = json.Unmarshal(raw, &payload)
		out = append(out, payload)
	}
	return out, nil
}

func (h *Handler) generateSimulationConfigViaGo(ctx context.Context, simulationID string, state map[string]any, simulationRequirement string, entities []map[string]any, useLLM bool) (map[string]any, error) {
	model := strings.TrimSpace(os.Getenv("LLM_MODEL_NAME"))
	baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL"))
	var gen intontology.ContentGenerator
	if useLLM {
		if exec := h.providerExecutor(); exec != nil {
			gen = providerContentGenerator{exec: exec, model: model}
		}
	}
	resolver := intontology.NewConfigResolver(gen)
	input := make([]intontology.Entity, 0, len(entities))
	for _, entity := range entities {
		input = append(input, intontology.Entity{
			UUID:       fmt.Sprint(entity["uuid"]),
			Name:       fmt.Sprint(entity["name"]),
			Labels:     toStringSlice(entity["labels"]),
			Summary:    strings.TrimSpace(fmt.Sprint(entity["summary"])),
			Attributes: toStringAnyMap(entity["attributes"]),
		})
	}
	cfg, err := resolver.Resolve(ctx, simulationID, fmt.Sprint(state["project_id"]), fmt.Sprint(state["graph_id"]), simulationRequirement, model, baseURL, input)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func normalizeStringList(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if cleaned := strings.TrimSpace(fmt.Sprint(item)); cleaned != "" && cleaned != "<nil>" {
				out = append(out, cleaned)
			}
		}
		return out
	default:
		return nil
	}
}

func mapSlice(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		return typed
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if mapped, ok := item.(map[string]any); ok {
				out = append(out, mapped)
			}
		}
		return out
	default:
		return nil
	}
}

func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return nil
	}
}

func toStringAnyMap(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func boolValue(value any) bool {
	got, _ := value.(bool)
	return got
}
