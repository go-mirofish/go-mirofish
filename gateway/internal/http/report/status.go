package reporthttp

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	var (
		reportID     string
		simulationID string
	)
	if r.Method == http.MethodGet {
		reportID = strings.TrimSpace(r.URL.Query().Get("report_id"))
		simulationID = strings.TrimSpace(r.URL.Query().Get("simulation_id"))
	} else if r.Method == http.MethodPost {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			if v, ok := payload["report_id"].(string); ok {
				reportID = strings.TrimSpace(v)
			}
			if v, ok := payload["simulation_id"].(string); ok {
				simulationID = strings.TrimSpace(v)
			}
		}
	} else {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if reportID != "" {
		status, err := h.service.StatusByReportID(reportID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": status})
		return
	}
	if simulationID != "" {
		status, err := h.service.StatusBySimulationID(simulationID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": status})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "report_id or simulation_id is required"})
}
