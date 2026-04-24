package preparehttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	apphttp "github.com/go-mirofish/go-mirofish/gateway/internal/http/app"
	intprovider "github.com/go-mirofish/go-mirofish/gateway/internal/provider"
	localfs "github.com/go-mirofish/go-mirofish/gateway/internal/store/localfs"
)

type graphStub struct {
	data map[string]any
	err  error
}

func (g graphStub) GetGraphData(ctx context.Context, graphID string) (map[string]any, error) {
	return g.data, g.err
}

type providerExecStub struct {
	content string
	err     error
}

func (p providerExecStub) Execute(ctx context.Context, req intprovider.ProviderRequest) (intprovider.ProviderResponse, error) {
	if p.err != nil {
		return intprovider.ProviderResponse{}, p.err
	}
	return intprovider.ProviderResponse{Content: p.content, StatusCode: 200, Model: req.Model, Provider: "stub"}, nil
}

func TestHandleGenerateProfilesDeterministicAndLLM(t *testing.T) {
	graph := graphStub{data: map[string]any{
		"graph_id": "graph-1",
		"nodes": []map[string]any{
			{"uuid": "node-1", "name": "Alice", "labels": []string{"Entity", "Person"}, "summary": "City planner", "attributes": map[string]any{}},
		},
		"edges": []map[string]any{},
	}}
	store := localfs.New(filepath.Join(t.TempDir(), "projects"), "", filepath.Join(t.TempDir(), "tasks"), filepath.Join(t.TempDir(), "sims"), filepath.Join(t.TempDir(), "scripts"))
	handler := NewHandler(store, graph, apphttp.WriteJSON)

	t.Run("deterministic success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/simulation/generate-profiles", bytes.NewBufferString(`{"graph_id":"graph-1","platform":"reddit","use_llm":false}`))
		rec := httptest.NewRecorder()
		handler.HandleGenerateProfiles(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"count":1`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("llm-backed success", func(t *testing.T) {
		prevFactory := newProviderExecutor
		newProviderExecutor = func(cfg intprovider.Config) intprovider.Executor {
			return providerExecStub{content: `{"bio":"bio","persona":"persona","interested_topics":["Policy"]}`}
		}
		defer func() { newProviderExecutor = prevFactory }()
		t.Setenv("LLM_BASE_URL", "http://stub")
		t.Setenv("LLM_API_KEY", "key")
		t.Setenv("LLM_MODEL_NAME", "model")
		req := httptest.NewRequest(http.MethodPost, "/api/simulation/generate-profiles", bytes.NewBufferString(`{"graph_id":"graph-1","platform":"reddit","use_llm":true}`))
		rec := httptest.NewRecorder()
		handler.HandleGenerateProfiles(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"Policy"`) {
			t.Fatalf("unexpected llm body: %s", rec.Body.String())
		}
	})
}

func TestHandlePrepareValidationAndAccepted(t *testing.T) {
	graph := graphStub{data: map[string]any{
		"graph_id": "graph-1",
		"nodes": []map[string]any{
			{"uuid": "node-1", "name": "Alice", "labels": []string{"Entity", "Person"}, "summary": "City planner", "attributes": map[string]any{}},
		},
		"edges": []map[string]any{},
	}}

	t.Run("validation failure", func(t *testing.T) {
		root := t.TempDir()
		store := localfs.New(filepath.Join(root, "projects"), filepath.Join(root, "reports"), filepath.Join(root, "tasks"), filepath.Join(root, "sims"), filepath.Join(root, "scripts"))
		handler := NewHandler(store, graph, apphttp.WriteJSON)
		req := httptest.NewRequest(http.MethodPost, "/api/simulation/prepare", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		handler.HandlePrepare(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("accepted", func(t *testing.T) {
		root := t.TempDir()
		store := localfs.New(filepath.Join(root, "projects"), filepath.Join(root, "reports"), filepath.Join(root, "tasks"), filepath.Join(root, "sims"), filepath.Join(root, "scripts"))
		handler := NewHandler(store, graph, apphttp.WriteJSON)
		project := map[string]any{
			"project_id":             "proj-1",
			"graph_id":               "graph-1",
			"simulation_requirement": "Test simulation",
		}
		if err := store.WriteProject("proj-1", project); err != nil {
			t.Fatalf("WriteProject: %v", err)
		}
		if err := store.WriteSimulation("sim-1", map[string]any{"simulation_id": "sim-1", "project_id": "proj-1", "graph_id": "graph-1", "status": "created"}); err != nil {
			t.Fatalf("WriteSimulation: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/simulation/prepare", bytes.NewBufferString(`{"simulation_id":"sim-1","use_llm_for_profiles":false}`))
		rec := httptest.NewRecorder()
		handler.HandlePrepare(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		taskID := payload["data"].(map[string]any)["task_id"].(string)
		if taskID == "" {
			t.Fatalf("expected task_id")
		}
		if _, err := store.ReadTask(taskID); err != nil {
			t.Fatalf("expected task persisted: %v", err)
		}
		// runTask is async; wait so Windows can remove t.TempDir() (no open handles).
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			task, err := store.ReadTask(taskID)
			if err != nil {
				break
			}
			st, _ := task["status"].(string)
			if st == "completed" || st == "failed" {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
	})
}
