package reporthttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

func (h *Handler) HandleRoute(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	const prefix = "api/report/"
	if !strings.HasPrefix(trimmed, prefix) {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(trimmed, prefix)
	if rest == "" || rest == "." {
		http.NotFound(w, r)
		return
	}

	switch {
	case rest == "generate":
		h.HandleGenerate(w, r)
		return
	case rest == "generate/status":
		h.HandleStatus(w, r)
		return
	case rest == "list":
		h.handleList(w, r)
		return
	case rest == "chat":
		h.handleChat(w, r)
		return
	case rest == "tools/search":
		h.handleSearchTool(w, r)
		return
	case rest == "tools/statistics":
		h.handleStatisticsTool(w, r)
		return
	case strings.HasPrefix(rest, "by-simulation/"):
		h.handleBySimulation(w, r, strings.TrimPrefix(rest, "by-simulation/"))
		return
	case strings.HasPrefix(rest, "check/"):
		h.handleCheck(w, r, strings.TrimPrefix(rest, "check/"))
		return
	}

	reportID, suffix := splitReportPath(rest)
	if reportID == "" {
		http.NotFound(w, r)
		return
	}
	switch {
	case suffix == "" && r.Method == http.MethodGet:
		h.handleGet(w, r, reportID)
	case suffix == "" && r.Method == http.MethodDelete:
		h.handleDelete(w, r, reportID)
	case suffix == "/download":
		h.handleDownload(w, r, reportID)
	case suffix == "/progress":
		h.handleProgress(w, r, reportID)
	case suffix == "/sections":
		h.handleSections(w, r, reportID)
	case strings.HasPrefix(suffix, "/section/"):
		indexRaw := strings.TrimPrefix(suffix, "/section/")
		h.handleSection(w, r, reportID, indexRaw)
	case suffix == "/agent-log":
		h.handleAgentLog(w, r, reportID)
	case suffix == "/agent-log/stream":
		h.handleAgentLogStream(w, r, reportID)
	case suffix == "/console-log":
		h.handleConsoleLog(w, r, reportID)
	case suffix == "/console-log/stream":
		h.handleConsoleLogStream(w, r, reportID)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request, reportID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	report, err := h.service.Get(reportID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": report})
}

func (h *Handler) handleBySimulation(w http.ResponseWriter, r *http.Request, simulationID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	report, found, err := h.service.GetBySimulation(simulationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "No report found for simulation: " + simulationID, "has_report": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": report, "has_report": true})
}

func (h *Handler) handleCheck(w http.ResponseWriter, r *http.Request, simulationID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	report, found, err := h.service.GetBySimulation(simulationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	data := map[string]any{
		"simulation_id":      simulationID,
		"has_report":         found,
		"report_status":      nil,
		"report_id":          nil,
		"interview_unlocked": false,
	}
	if found {
		data["report_status"] = report["status"]
		data["report_id"] = report["report_id"]
		data["interview_unlocked"] = report["status"] == "completed"
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
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
	reports, err := h.service.List(strings.TrimSpace(r.URL.Query().Get("simulation_id")), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": reports, "count": len(reports)})
}

func (h *Handler) handleProgress(w http.ResponseWriter, r *http.Request, reportID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	progress, err := h.service.Progress(reportID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": progress})
}

func (h *Handler) handleSections(w http.ResponseWriter, r *http.Request, reportID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sections, err := h.service.Sections(reportID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": sections})
}

func (h *Handler) handleSection(w http.ResponseWriter, r *http.Request, reportID, sectionIndexRaw string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	index, err := strconv.Atoi(sectionIndexRaw)
	if err != nil || index < 0 {
		http.Error(w, "invalid section index", http.StatusBadRequest)
		return
	}
	section, err := h.service.Section(reportID, index)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": section})
}

func (h *Handler) handleDownload(w http.ResponseWriter, r *http.Request, reportID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := h.service.Download(reportID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+reportID+".md\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request, reportID string) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := h.service.Delete(reportID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "Report deleted: " + reportID})
}

func (h *Handler) handleAgentLog(w http.ResponseWriter, r *http.Request, reportID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fromLine, _ := strconv.Atoi(r.URL.Query().Get("from_line"))
	data, err := h.service.AgentLog(reportID, fromLine)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func (h *Handler) handleAgentLogStream(w http.ResponseWriter, r *http.Request, reportID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	logs, err := h.service.AgentLogStream(reportID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"logs": logs, "count": len(logs)}})
}

func (h *Handler) handleConsoleLog(w http.ResponseWriter, r *http.Request, reportID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fromLine, _ := strconv.Atoi(r.URL.Query().Get("from_line"))
	data, err := h.service.ConsoleLog(reportID, fromLine)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func (h *Handler) handleConsoleLogStream(w http.ResponseWriter, r *http.Request, reportID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	logs, err := h.service.ConsoleLogStream(reportID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"logs": logs, "count": len(logs)}})
}

func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		SimulationID string           `json:"simulation_id"`
		Message      string           `json:"message"`
		ChatHistory  []map[string]any `json:"chat_history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	result, err := h.service.Chat(r.Context(), payload.SimulationID, payload.Message, payload.ChatHistory, h.lookup, h.model, h.provider)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": result})
}

func (h *Handler) handleSearchTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		GraphID string `json:"graph_id"`
		Query   string `json:"query"`
		Limit   int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	result, err := h.service.SearchGraph(r.Context(), payload.GraphID, payload.Query, payload.Limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": result})
}

func (h *Handler) handleStatisticsTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		GraphID string `json:"graph_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	result, err := h.service.GraphStatistics(r.Context(), payload.GraphID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": result})
}

func splitReportPath(rest string) (reportID, suffix string) {
	parts := strings.SplitN(rest, "/", 2)
	reportID = parts[0]
	if len(parts) == 2 {
		suffix = "/" + parts[1]
	}
	return reportID, suffix
}
