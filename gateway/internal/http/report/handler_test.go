package reporthttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/internal/memory"
	intprovider "github.com/go-mirofish/go-mirofish/gateway/internal/provider"
	intreport "github.com/go-mirofish/go-mirofish/gateway/internal/report"
	reportstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/report"
)

type lookupStub struct {
	simulation map[string]any
	project    map[string]any
	err        error
}

func (l lookupStub) ReadSimulation(simulationID string) (map[string]any, error) {
	if l.err != nil {
		return nil, l.err
	}
	return l.simulation, nil
}

func (l lookupStub) ReadProject(projectID string) (map[string]any, error) {
	if l.err != nil {
		return nil, l.err
	}
	return l.project, nil
}

type memoryStub struct {
	resp     memory.SearchResponse
	err      error
	graph    memory.GraphData
	graphErr error
}

func (m memoryStub) AddFact(ctx context.Context, fact memory.Fact) error { return nil }
func (m memoryStub) GetFacts(ctx context.Context, graphID string, limit int) ([]memory.Fact, error) {
	return nil, nil
}
func (m memoryStub) SearchGraph(ctx context.Context, req memory.SearchRequest) (memory.SearchResponse, error) {
	if m.err != nil {
		return memory.SearchResponse{}, m.err
	}
	return m.resp, nil
}
func (m memoryStub) DeleteNode(ctx context.Context, nodeID string) error { return nil }
func (m memoryStub) GetGraphData(ctx context.Context, graphID string) (memory.GraphData, error) {
	if m.graphErr != nil {
		return memory.GraphData{}, m.graphErr
	}
	return m.graph, nil
}
func (m memoryStub) DeleteGraph(ctx context.Context, graphID string) error { return nil }

type providerStub struct {
	content string
	err     error
}

func (p providerStub) Execute(ctx context.Context, req intprovider.ProviderRequest) (intprovider.ProviderResponse, error) {
	if p.err != nil {
		return intprovider.ProviderResponse{}, p.err
	}
	return intprovider.ProviderResponse{Content: p.content, StatusCode: 200, Provider: "stub", Model: req.Model}, nil
}

type storeStub struct {
	base            reportstore.Store
	createErr       error
	saveProgressErr error
	saveSectionErr  error
	saveSectionOnce bool
	sectionCalls    int
}

func (s *storeStub) CreateReport(reportID string, meta reportstore.ReportMeta) error {
	if s.createErr != nil {
		return s.createErr
	}
	return s.base.CreateReport(reportID, meta)
}

func (s *storeStub) SaveMeta(reportID string, meta reportstore.ReportMeta) error {
	return s.base.SaveMeta(reportID, meta)
}

func (s *storeStub) LoadMeta(reportID string) (reportstore.ReportMeta, error) {
	return s.base.LoadMeta(reportID)
}

func (s *storeStub) SaveProgress(reportID string, progress reportstore.Progress) error {
	if s.saveProgressErr != nil {
		return s.saveProgressErr
	}
	return s.base.SaveProgress(reportID, progress)
}

func (s *storeStub) LoadProgress(reportID string) (reportstore.Progress, error) {
	return s.base.LoadProgress(reportID)
}

func (s *storeStub) SaveSection(reportID string, index int, title string, content string) error {
	s.sectionCalls++
	if s.saveSectionErr != nil && (!s.saveSectionOnce || s.sectionCalls == 1) {
		return s.saveSectionErr
	}
	return s.base.SaveSection(reportID, index, title, content)
}

func (s *storeStub) LoadSections(reportID string) ([]map[string]any, error) {
	return s.base.LoadSections(reportID)
}

func (s *storeStub) SaveMarkdown(reportID string, markdown string) error {
	return s.base.SaveMarkdown(reportID, markdown)
}

func (s *storeStub) LoadMarkdown(reportID string) (string, error) {
	return s.base.LoadMarkdown(reportID)
}

func (s *storeStub) DeleteReport(reportID string) error {
	return s.base.DeleteReport(reportID)
}

func (s *storeStub) ListReports(simulationID string, limit int) ([]reportstore.ReportMeta, error) {
	return s.base.ListReports(simulationID, limit)
}

func (s *storeStub) GetAgentLog(reportID string, fromLine int) (map[string]any, error) {
	return s.base.GetAgentLog(reportID, fromLine)
}

func (s *storeStub) GetAgentLogStream(reportID string) ([]map[string]any, error) {
	return s.base.GetAgentLogStream(reportID)
}

func (s *storeStub) GetConsoleLog(reportID string, fromLine int) (map[string]any, error) {
	return s.base.GetConsoleLog(reportID, fromLine)
}

func (s *storeStub) GetConsoleLogStream(reportID string) ([]string, error) {
	return s.base.GetConsoleLogStream(reportID)
}

func (s *storeStub) FindBySimulation(simulationID string) (reportstore.ReportMeta, bool, error) {
	return s.base.FindBySimulation(simulationID)
}

func newHandlerForTest(t *testing.T, store reportstore.Store, mem memory.Client, providerExec intprovider.Executor) *Handler {
	t.Helper()
	planner := intreport.NewPlanner(providerExec, "model")
	service := intreport.NewService(store, mem, planner)
	return NewHandler(service, lookupStub{
		simulation: map[string]any{"simulation_id": "sim-1", "project_id": "proj-1", "graph_id": "graph-1"},
		project:    map[string]any{"project_id": "proj-1", "simulation_requirement": "test requirement"},
	}, "model", providerExec)
}

func decodePayload(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload
}

func startGeneration(t *testing.T, handler *Handler, body string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/report/generate", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.HandleGenerate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", rec.Code, rec.Body.String())
	}
	payload := decodePayload(t, rec)
	return payload["data"].(map[string]any)["report_id"].(string)
}

func waitForStatus(t *testing.T, handler *Handler, reportID string, expected string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/generate/status?report_id="+reportID, nil)
		rec := httptest.NewRecorder()
		handler.HandleStatus(rec, req)
		if rec.Code == http.StatusOK {
			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode status response: %v", err)
			}
			data := payload["data"].(map[string]any)
			if data["status"] == expected {
				return data
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for status=%s", expected)
	return nil
}

func TestHandleGenerateAndStatusSuccess(t *testing.T) {
	store := reportstore.NewFileStore(filepath.Join(t.TempDir(), "reports"))
	handler := newHandlerForTest(t, store, memoryStub{resp: memory.SearchResponse{Facts: []string{"fact-a", "fact-b"}}}, nil)

	reportID := startGeneration(t, handler, `{"simulation_id":"sim-1"}`)
	status := waitForStatus(t, handler, reportID, "completed")
	if status["progress"].(float64) != 100 {
		t.Fatalf("expected progress 100, got %#v", status["progress"])
	}
}

func TestHandleGenerateBadRequest(t *testing.T) {
	store := reportstore.NewFileStore(filepath.Join(t.TempDir(), "reports"))
	handler := newHandlerForTest(t, store, memoryStub{}, nil)

	tests := []struct {
		name         string
		body         string
		wantFragment string
	}{
		{name: "invalid json", body: `{"simulation_id":`, wantFragment: "invalid JSON body"},
		{name: "missing simulation id", body: `{}`, wantFragment: "invalid report request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/report/generate", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			handler.HandleGenerate(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
			payload := decodePayload(t, rec)
			if payload["success"] != false {
				t.Fatalf("expected success=false, got %#v", payload["success"])
			}
			if !strings.Contains(payload["error"].(string), tt.wantFragment) {
				t.Fatalf("expected error containing %q, got %q", tt.wantFragment, payload["error"])
			}
		})
	}
}

func TestHandleGenerateRunFailuresSurfaceFailedStatus(t *testing.T) {
	tests := []struct {
		name         string
		store        reportstore.Store
		memoryClient memory.Client
		providerExec intprovider.Executor
		wantMessage  string
	}{
		{
			name:         "memory failure",
			store:        reportstore.NewFileStore(filepath.Join(t.TempDir(), "reports-memory")),
			memoryClient: memoryStub{err: errors.New("memory down")},
			wantMessage:  "memory down",
		},
		{
			name:         "provider failure",
			store:        reportstore.NewFileStore(filepath.Join(t.TempDir(), "reports-provider")),
			memoryClient: memoryStub{resp: memory.SearchResponse{Facts: []string{"fact"}}},
			providerExec: providerStub{err: errors.New("provider failed")},
			wantMessage:  "provider failed",
		},
		{
			name:         "memory cancellation",
			store:        reportstore.NewFileStore(filepath.Join(t.TempDir(), "reports-memory-canceled")),
			memoryClient: memoryStub{err: context.Canceled},
			wantMessage:  context.Canceled.Error(),
		},
		{
			name:         "provider timeout",
			store:        reportstore.NewFileStore(filepath.Join(t.TempDir(), "reports-provider-timeout")),
			memoryClient: memoryStub{resp: memory.SearchResponse{Facts: []string{"fact"}}},
			providerExec: providerStub{err: context.DeadlineExceeded},
			wantMessage:  context.DeadlineExceeded.Error(),
		},
		{
			name: "persistence failure",
			store: &storeStub{
				base:            reportstore.NewFileStore(filepath.Join(t.TempDir(), "reports-persistence")),
				saveSectionErr:  errors.New("save section failed"),
				saveSectionOnce: true,
			},
			memoryClient: memoryStub{resp: memory.SearchResponse{Facts: []string{"fact"}}},
			wantMessage:  "save section failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newHandlerForTest(t, tt.store, tt.memoryClient, tt.providerExec)
			reportID := startGeneration(t, handler, `{"simulation_id":"sim-1"}`)
			status := waitForStatus(t, handler, reportID, "failed")
			if !strings.Contains(status["message"].(string), tt.wantMessage) {
				t.Fatalf("expected message containing %q, got %#v", tt.wantMessage, status["message"])
			}
		})
	}
}

func TestHandleGeneratePersistenceFailureReturnsBadRequest(t *testing.T) {
	store := &storeStub{
		base:            reportstore.NewFileStore(filepath.Join(t.TempDir(), "reports")),
		saveProgressErr: errors.New("progress write failed"),
	}
	handler := newHandlerForTest(t, store, memoryStub{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/report/generate", bytes.NewBufferString(`{"simulation_id":"sim-1"}`))
	rec := httptest.NewRecorder()
	handler.HandleGenerate(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	payload := decodePayload(t, rec)
	if !strings.Contains(payload["error"].(string), "progress write failed") {
		t.Fatalf("expected persistence error, got %q", payload["error"])
	}
}

func errorStringLooksLikeFileNotOpen(s string) bool {
	// Go surfaces OS-specific phrasing: Unix "no such file or directory" vs
	// Windows "The system cannot find the path specified."
	return strings.Contains(s, "no such file or directory") ||
		strings.Contains(s, "cannot find the path") ||
		strings.Contains(s, "cannot find the file")
}

func TestHandleStatusBadRequestAndMissing(t *testing.T) {
	store := reportstore.NewFileStore(filepath.Join(t.TempDir(), "reports"))
	handler := newHandlerForTest(t, store, memoryStub{}, nil)

	tests := []struct {
		name         string
		method       string
		target       string
		body         string
		wantStatus   int
		wantFragment string
		wantFSError  bool
	}{
		{
			name:         "missing identifiers via get",
			method:       http.MethodGet,
			target:       "/api/report/generate/status",
			wantStatus:   http.StatusBadRequest,
			wantFragment: "report_id or simulation_id is required",
		},
		{
			name:         "missing report id",
			method:       http.MethodGet,
			target:       "/api/report/generate/status?report_id=missing",
			wantStatus:   http.StatusNotFound,
			wantFSError:  true,
		},
		{
			name:         "missing simulation id",
			method:       http.MethodPost,
			target:       "/api/report/generate/status",
			body:         `{"simulation_id":"missing"}`,
			wantStatus:   http.StatusNotFound,
			wantFragment: reportstore.ErrProgressNotFound.Error(),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			handler.HandleStatus(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rec.Code)
			}
			payload := decodePayload(t, rec)
			errStr := payload["error"].(string)
			if tt.wantFSError {
				if !errorStringLooksLikeFileNotOpen(errStr) {
					t.Fatalf("expected a file-not-found style error, got %q", errStr)
				}
				return
			}
			if !strings.Contains(errStr, tt.wantFragment) {
				t.Fatalf("expected error containing %q, got %q", tt.wantFragment, errStr)
			}
		})
	}
}

func TestHandleRouteReportParitySurfaces(t *testing.T) {
	reportsDir := filepath.Join(t.TempDir(), "reports")
	store := reportstore.NewFileStore(reportsDir)
	mem := memoryStub{
		resp: memory.SearchResponse{
			Facts: []string{"fact-a"},
			Edges: []memory.Edge{{UUID: "edge-1", Name: "related_to", Fact: "fact-a", SourceNodeUUID: "node-1", TargetNodeUUID: "node-2"}},
			Nodes: []memory.Node{{UUID: "node-1", Name: "Node One", Labels: []string{"Entity", "Person"}, Summary: "summary"}},
		},
		graph: memory.GraphData{
			GraphID: "graph-1",
			Nodes: []memory.GraphNode{
				{UUID: "node-1", Name: "Node One", Labels: []string{"Entity", "Person"}},
				{UUID: "node-2", Name: "Node Two", Labels: []string{"Company"}},
			},
			Edges: []memory.GraphEdge{
				{Name: "related_to"},
				{Name: "related_to"},
				{Name: "mentions"},
			},
		},
	}
	handler := newHandlerForTest(t, store, mem, providerStub{content: "answer from provider"})

	reportID := startGeneration(t, handler, `{"simulation_id":"sim-1"}`)
	waitForStatus(t, handler, reportID, "completed")

	reportDir := filepath.Join(reportsDir, reportID)
	if err := os.WriteFile(filepath.Join(reportDir, "agent_log.jsonl"), []byte("{\"action\":\"tool_call\",\"details\":{\"tool_name\":\"search_graph\"}}\n"), 0o644); err != nil {
		t.Fatalf("write agent log: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "console_log.txt"), []byte("line one\nline two\n"), 0o644); err != nil {
		t.Fatalf("write console log: %v", err)
	}

	t.Run("get report", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/"+reportID, nil)
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		payload := decodePayload(t, rec)
		data := payload["data"].(map[string]any)
		if data["report_id"] != reportID {
			t.Fatalf("expected report_id %s, got %#v", reportID, data["report_id"])
		}
	})

	t.Run("list reports", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/list?limit=10", nil)
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("by simulation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/by-simulation/sim-1", nil)
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("check", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/check/sim-1", nil)
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		payload := decodePayload(t, rec)
		data := payload["data"].(map[string]any)
		if data["interview_unlocked"] != true {
			t.Fatalf("expected interview_unlocked=true, got %#v", data["interview_unlocked"])
		}
	})

	t.Run("progress", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/progress", nil)
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("sections and single section", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/sections", nil)
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/section/1", nil)
		rec = httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("download", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/download", nil)
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, reportID+".md") {
			t.Fatalf("expected attachment filename, got %q", got)
		}
	})

	t.Run("chat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/report/chat", bytes.NewBufferString(`{"simulation_id":"sim-1","message":"what happened?"}`))
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		payload := decodePayload(t, rec)
		data := payload["data"].(map[string]any)
		if data["response"] != "answer from provider" {
			t.Fatalf("expected provider answer, got %#v", data["response"])
		}
	})

	t.Run("agent logs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/agent-log?from_line=0", nil)
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/agent-log/stream", nil)
		rec = httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("console logs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/console-log?from_line=1", nil)
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/report/"+reportID+"/console-log/stream", nil)
		rec = httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("tools", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/report/tools/search", bytes.NewBufferString(`{"graph_id":"graph-1","query":"trend","limit":5}`))
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/report/tools/statistics", bytes.NewBufferString(`{"graph_id":"graph-1"}`))
		rec = httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		payload := decodePayload(t, rec)
		data := payload["data"].(map[string]any)
		if data["total_edges"].(float64) != 3 {
			t.Fatalf("expected total_edges=3, got %#v", data["total_edges"])
		}
	})

	t.Run("delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/report/"+reportID, nil)
		rec := httptest.NewRecorder()
		handler.HandleRoute(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}
