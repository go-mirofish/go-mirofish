package reporthttp

import (
	"encoding/json"
	"net/http"

	"github.com/go-mirofish/go-mirofish/gateway/internal/provider"
	"github.com/go-mirofish/go-mirofish/gateway/internal/report"
)

type SimulationLookup interface {
	ReadSimulation(simulationID string) (map[string]any, error)
	ReadProject(projectID string) (map[string]any, error)
}

type Handler struct {
	service  *report.Service
	lookup   SimulationLookup
	model    string
	provider provider.Executor
}

func NewHandler(service *report.Service, lookup SimulationLookup, model string, provider provider.Executor) *Handler {
	return &Handler{service: service, lookup: lookup, model: model, provider: provider}
}

func (h *Handler) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req report.ReportGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	resp, err := h.service.Generate(r.Context(), req, h.lookup, h.model, h.provider)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": resp})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
