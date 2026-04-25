package apphttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	handler := New(nil, "")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.HandleHealth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"service":"go-mirofish-gateway"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"control_plane":"go"`) {
		t.Fatalf("expected ownership in health: %s", rec.Body.String())
	}
}

func TestHandleReady(t *testing.T) {
	handler := New(nil, "")

	t.Run("ready", func(t *testing.T) {
		handler.Ready = func(ctx context.Context) (map[string]any, error) {
			return map[string]any{"data_dirs": "ok"}, nil
		}
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()
		handler.HandleReady(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"status":"ready"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("not ready", func(t *testing.T) {
		handler.Ready = func(ctx context.Context) (map[string]any, error) {
			return map[string]any{"data_dirs": "down"}, context.DeadlineExceeded
		}
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()
		handler.HandleReady(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"status":"not_ready"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})
}

func TestHandleMetrics(t *testing.T) {
	requestMetrics = &metricsSnapshot{RouteData: map[string]routeMetric{
		"GET /health": {Count: 2, Errors: 1},
	}}
	handler := New(nil, "")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.HandleMetrics(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"GET /health"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestRequestLoggerSetsRequestIDAndCapturesPanic(t *testing.T) {
	requestMetrics = &metricsSnapshot{RouteData: map[string]routeMetric{}}
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/panic" {
			panic("boom")
		}
		if RequestIDFromContext(r.Context()) == "" {
			t.Fatalf("expected request id in context")
		}
		WriteJSON(w, http.StatusCreated, map[string]any{"ok": true})
	}))

	t.Run("normal request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}
		if rec.Header().Get("X-Request-ID") == "" {
			t.Fatalf("expected X-Request-ID header")
		}
	})

	t.Run("panic request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandleAPIProxy(t *testing.T) {
	handler := New(nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.HandleAPIProxy(rec, req)
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone (unmatched /api/ routes removed), got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "route not found") {
		t.Fatalf("expected route not found message in body, got: %s", rec.Body.String())
	}
}

func TestHandleFrontendStaticAndFallback(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<html>index</html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dist, "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	handler := New(nil, dist)

	t.Run("serves asset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		rec := httptest.NewRecorder()
		handler.HandleFrontend(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "console.log('ok')") {
			t.Fatalf("unexpected asset body: %s", rec.Body.String())
		}
	})

	t.Run("falls back to index", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/does/not/exist", nil)
		rec := httptest.NewRecorder()
		handler.HandleFrontend(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "<html>index</html>") {
			t.Fatalf("unexpected fallback body: %s", rec.Body.String())
		}
	})
}
