package graphhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	intgraph "github.com/go-mirofish/go-mirofish/gateway/internal/graph"
	graphstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/graph"
)

type Service interface {
	StartBuild(ctx context.Context, req intgraph.BuildRequest) (intgraph.BuildResponse, error)
	GetGraphData(ctx context.Context, graphID string) (map[string]any, error)
	DeleteGraph(ctx context.Context, graphID string) error
	GetTask(taskID string) (graphstore.TaskState, error)
	ListTasks(taskType string) ([]graphstore.TaskState, error)
	GetProject(projectID string) (map[string]any, error)
	ListProjects(limit int) ([]map[string]any, error)
	DeleteProject(projectID string) error
	ResetProject(projectID string) (map[string]any, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) HandleBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req intgraph.BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.StartBuild(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": resp})
}

func (h *Handler) HandleGraphData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	graphID := routeParam(r.URL.Path, "/api/graph/data/")
	if graphID == "" {
		http.NotFound(w, r)
		return
	}
	data, err := h.service.GetGraphData(r.Context(), graphID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func (h *Handler) HandleDeleteGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	graphID := routeParam(r.URL.Path, "/api/graph/delete/")
	if graphID == "" {
		http.NotFound(w, r)
		return
	}
	if err := h.service.DeleteGraph(r.Context(), graphID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Graph deleted: " + graphID,
	})
}

func (h *Handler) HandleTaskGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	taskID := routeParam(r.URL.Path, "/api/graph/task/")
	if taskID == "" {
		http.NotFound(w, r)
		return
	}
	task, err := h.service.GetTask(taskID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": task})
}

func (h *Handler) HandleTaskList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tasks, err := h.service.ListTasks(strings.TrimSpace(r.URL.Query().Get("task_type")))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    tasks,
		"count":   len(tasks),
	})
}

func (h *Handler) HandleProjectList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	projects, err := h.service.ListProjects(limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    projects,
		"count":   len(projects),
	})
}

func (h *Handler) HandleProjectRoute(w http.ResponseWriter, r *http.Request) {
	rest := routeParam(r.URL.Path, "/api/graph/project/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(rest, "/reset") {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		projectID := strings.TrimSuffix(rest, "/reset")
		if projectID == "" {
			http.NotFound(w, r)
			return
		}
		project, err := h.service.ResetProject(projectID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Project reset: " + projectID,
			"data":    project,
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		project, err := h.service.GetProject(rest)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": project})
	case http.MethodDelete:
		if err := h.service.DeleteProject(rest); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "Project deleted: " + rest,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func routeParam(routePath, prefix string) string {
	if !strings.HasPrefix(routePath, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path.Clean(routePath), "/")
	prefix = strings.TrimPrefix(prefix, "/")
	if !strings.HasPrefix(rest, prefix) {
		return ""
	}
	value := strings.TrimPrefix(rest, prefix)
	value = strings.TrimPrefix(value, "/")
	value = strings.TrimSpace(value)
	if value == "." {
		return ""
	}
	return value
}

func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, os.ErrNotExist):
		status = http.StatusNotFound
	case strings.Contains(strings.ToLower(err.Error()), "is required"):
		status = http.StatusBadRequest
	case errors.Is(err, context.Canceled):
		status = http.StatusRequestTimeout
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout
	}
	writeJSON(w, status, map[string]any{"success": false, "error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
