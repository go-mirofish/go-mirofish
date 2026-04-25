package ontologyhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	apphttp "github.com/go-mirofish/go-mirofish/gateway/internal/http/app"
	intprovider "github.com/go-mirofish/go-mirofish/gateway/internal/provider"
	localfs "github.com/go-mirofish/go-mirofish/gateway/internal/store/localfs"
)

type execStub struct {
	content string
	err     error
}

func (e execStub) Execute(ctx context.Context, req intprovider.ProviderRequest) (intprovider.ProviderResponse, error) {
	if e.err != nil {
		return intprovider.ProviderResponse{}, e.err
	}
	return intprovider.ProviderResponse{Content: e.content, StatusCode: 200, Model: req.Model, Provider: "stub"}, nil
}

func TestHandleGenerateValidation(t *testing.T) {
	store := localfs.New(filepath.Join(t.TempDir(), "projects"), "", "", "", "")
	var proxied bool
	handler := NewHandler(store, func(w http.ResponseWriter, r *http.Request) {
		proxied = true
		apphttp.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
	}, apphttp.WriteJSON)

	t.Run("missing requirement", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("files", "seed.txt")
		_, _ = part.Write([]byte("hello"))
		_ = writer.Close()
		req := httptest.NewRequest(http.MethodPost, "/api/graph/ontology/generate", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		handler.HandleGenerate(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("pdf_handled_in_gateway", func(t *testing.T) {
		prevPDF := ExtractPDFForOntology
		ExtractPDFForOntology = func(_ []byte) (string, error) { return "Extracted PDF body for tests", nil }
		t.Cleanup(func() { ExtractPDFForOntology = prevPDF })
		t.Setenv("LLM_API_KEY", "k")
		t.Setenv("LLM_MODEL_NAME", "m")
		t.Setenv("LLM_BASE_URL", "http://stub")
		prevEx := newExecutor
		newExecutor = func(cfg intprovider.Config) intprovider.Executor {
			return execStub{content: `{"entity_types":[{"name":"person","description":"Person","attributes":[{"name":"name","type":"text","description":"Name"}],"examples":["Alice"]}],"edge_types":[{"name":"works_at","description":"Employment","attributes":[],"source_targets":[{"source":"Person","target":"Organization"}]}],"analysis_summary":"ok"}`}
		}
		t.Cleanup(func() { newExecutor = prevEx })

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("simulation_requirement", "test")
		part, _ := writer.CreateFormFile("files", "seed.pdf")
		_, _ = part.Write([]byte("%PDF-fake"))
		_ = writer.Close()
		req := httptest.NewRequest(http.MethodPost, "/api/graph/ontology/generate", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		handler.HandleGenerate(rec, req)
		if rec.Code != http.StatusOK || proxied {
			t.Fatalf("expected 200 in gateway, proxied=%v, code=%d body=%s", proxied, rec.Code, rec.Body.String())
		}
	})
}

func TestHandleGenerateSuccessAndProviderFailure(t *testing.T) {
	storeRoot := t.TempDir()
	store := localfs.New(filepath.Join(storeRoot, "projects"), "", "", "", "")
	handler := NewHandler(store, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("did not expect proxy")
	}, apphttp.WriteJSON)

	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("LLM_MODEL_NAME", "test-model")
	t.Setenv("LLM_BASE_URL", "http://stub")
	prevFactory := newExecutor
	newExecutor = func(cfg intprovider.Config) intprovider.Executor {
		return execStub{content: `{"entity_types":[{"name":"person","description":"Person","attributes":[{"name":"name","type":"text","description":"Name"}],"examples":["Alice"]}],"edge_types":[{"name":"works_at","description":"Employment","attributes":[],"source_targets":[{"source":"Person","target":"Organization"}]}],"analysis_summary":"ok"}`}
	}
	defer func() { newExecutor = prevFactory }()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("project_name", "Demo")
	_ = writer.WriteField("simulation_requirement", "Analyze scenario")
	part, _ := writer.CreateFormFile("files", "seed.txt")
	_, _ = part.Write([]byte("plain text content"))
	_ = writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/graph/ontology/generate", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	handler.HandleGenerate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	data := payload["data"].(map[string]any)
	projectID := data["project_id"].(string)
	if _, err := os.Stat(filepath.Join(storeRoot, "projects", projectID, "project.json")); err != nil {
		t.Fatalf("expected project.json: %v", err)
	}

	newExecutor = func(cfg intprovider.Config) intprovider.Executor {
		return execStub{err: errors.New("upstream failure")}
	}
	body2 := &bytes.Buffer{}
	writer2 := multipart.NewWriter(body2)
	_ = writer2.WriteField("simulation_requirement", "Analyze scenario")
	part2, _ := writer2.CreateFormFile("files", "seed.txt")
	_, _ = part2.Write([]byte("plain text content"))
	_ = writer2.Close()
	req2 := httptest.NewRequest(http.MethodPost, "/api/graph/ontology/generate", body2)
	req2.Header.Set("Content-Type", writer2.FormDataContentType())
	rec2 := httptest.NewRecorder()
	handler.HandleGenerate(rec2, req2)
	if rec2.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "upstream failure") {
		t.Fatalf("unexpected failure body: %s", rec2.Body.String())
	}
}
