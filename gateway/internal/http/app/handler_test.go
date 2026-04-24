package apphttp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestHandleHealth(t *testing.T) {
	handler := New("http://backend.test", nil, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.HandleHealth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"service":"go-mirofish-gateway"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestHandleAPIProxy(t *testing.T) {
	target, _ := url.Parse("http://backend.test")
	proxy := httputil.NewSingleHostReverseProxy(target)
	var gotPath string
	proxy.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"success":true}`)),
		}, nil
	})
	handler := New(target.String(), proxy, nil, "")
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.HandleAPIProxy(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotPath != "/api/test" {
		t.Fatalf("expected proxied path /api/test, got %s", gotPath)
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
	handler := New("http://backend.test", nil, nil, dist)

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
