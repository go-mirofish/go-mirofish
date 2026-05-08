package headless

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestConfig(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	dist := filepath.Join(root, "frontend", "dist")
	if err := tSetFile(dist, "index.html", "<html>sdk</html>"); err != nil {
		t.Fatalf("write index: %v", err)
	}
	return Config{
		BindHost:        "127.0.0.1",
		Port:            "3000",
		FrontendDistDir: dist,
		ProjectsDir:     filepath.Join(root, "data", "projects"),
		ReportsDir:      filepath.Join(root, "data", "reports"),
		TasksDir:        filepath.Join(root, "data", "tasks"),
		SimulationsDir:  filepath.Join(root, "data", "simulations"),
		ScriptsDir:      filepath.Join(root, "scripts"),
	}
}

func TestNewAndHealthRoutes(t *testing.T) {
	cfg := newTestConfig(t)
	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t.Run("health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("ready", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("providers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["success"] != true {
			t.Fatalf("expected success=true, got %#v", payload["success"])
		}
	})
}

func TestRunStartsAndStopsFromContext(t *testing.T) {
	cfg := newTestConfig(t)
	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- app.ListenAndServe(ctx)
	}()
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("ListenAndServe: %v", err)
	}
}

func tSetFile(dir, name, content string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}
