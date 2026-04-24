package apphttp

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Handler struct {
	BackendURL       string
	BackendProxy     *httputil.ReverseProxy
	FrontendDevProxy *httputil.ReverseProxy
	FrontendDistDir  string
}

func New(backendURL string, backendProxy *httputil.ReverseProxy, frontendDevProxy *httputil.ReverseProxy, frontendDistDir string) *Handler {
	return &Handler{
		BackendURL:       backendURL,
		BackendProxy:     backendProxy,
		FrontendDevProxy: frontendDevProxy,
		FrontendDistDir:  frontendDistDir,
	}
}

func (h *Handler) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "go-mirofish-gateway",
		"backend": h.BackendURL,
	})
}

func (h *Handler) HandleAPIProxy(w http.ResponseWriter, r *http.Request) {
	h.BackendProxy.ServeHTTP(w, r)
}

func (h *Handler) HandleFrontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.tryServeStatic(w, r) {
		return
	}
	if h.FrontendDevProxy != nil {
		h.FrontendDevProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "frontend assets not found; build frontend/dist or set FRONTEND_DEV_URL", http.StatusServiceUnavailable)
}

func (h *Handler) tryServeStatic(w http.ResponseWriter, r *http.Request) bool {
	distRoot := filepath.Clean(h.FrontendDistDir)
	indexPath := filepath.Join(distRoot, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return false
	}
	requestPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if requestPath == "" || requestPath == "." {
		http.ServeFile(w, r, indexPath)
		return true
	}
	candidate := filepath.Join(distRoot, filepath.FromSlash(requestPath))
	if isFileInside(candidate, distRoot) {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			http.ServeFile(w, r, candidate)
			return true
		}
	}
	http.ServeFile(w, r, indexPath)
	return true
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func isFileInside(candidate, root string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
