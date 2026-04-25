package apphttp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/internal/telemetry"
)

type contextKey string

const requestIDKey contextKey = "request_id"

type ReadinessFunc func(context.Context) (map[string]any, error)

type Handler struct {
	FrontendDevProxy *httputil.ReverseProxy
	FrontendDistDir  string
	Ready            ReadinessFunc
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

type routeMetric struct {
	Count         int64         `json:"count"`
	Errors        int64         `json:"errors"`
	TotalDuration time.Duration `json:"total_duration_ns"`
	MaxDuration   time.Duration `json:"max_duration_ns"`
}

type metricsSnapshot struct {
	mu        sync.Mutex
	Requests  int64                  `json:"requests"`
	Errors    int64                  `json:"errors"`
	RouteData map[string]routeMetric `json:"routes"`
}

var requestMetrics = &metricsSnapshot{RouteData: map[string]routeMetric{}}

func New(frontendDevProxy *httputil.ReverseProxy, frontendDistDir string) *Handler {
	return &Handler{
		FrontendDevProxy: frontendDevProxy,
		FrontendDistDir:  frontendDistDir,
	}
}

func (h *Handler) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "go-mirofish-gateway",
		"stack":   "go",
		"runtime": map[string]any{
			"control_plane":       "go",
			"simulation_worker":   "go_native",
			"frontend":            "vue",
			"python_backend":      "removed",
			"dev_command_default": "make up && npm run dev",
		},
	})
}

func (h *Handler) HandleReady(w http.ResponseWriter, r *http.Request) {
	if h.Ready == nil {
		WriteJSON(w, http.StatusOK, map[string]any{
			"status":  "ready",
			"service": "go-mirofish-gateway",
		})
		return
	}

	payload, err := h.Ready(r.Context())
	if err != nil {
		if payload == nil {
			payload = map[string]any{}
		}
		payload["status"] = "not_ready"
		payload["service"] = "go-mirofish-gateway"
		payload["error"] = err.Error()
		WriteJSON(w, http.StatusServiceUnavailable, payload)
		return
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["status"] = "ready"
	payload["service"] = "go-mirofish-gateway"
	WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) HandleMetrics(w http.ResponseWriter, _ *http.Request) {
	requestMetrics.mu.Lock()
	defer requestMetrics.mu.Unlock()

	routes := make(map[string]routeMetric, len(requestMetrics.RouteData))
	for key, value := range requestMetrics.RouteData {
		routes[key] = value
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"requests":      requestMetrics.Requests,
		"errors":        requestMetrics.Errors,
		"routes":        routes,
		"control_plane": telemetry.SnapshotMetrics(),
	})
}

// HandleAPIProxy returns 410 Gone for any /api/* path not matched by a
// registered Go-native handler. The Python backend is removed.
func (h *Handler) HandleAPIProxy(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusGone, map[string]any{
		"success": false,
		"error":   "route not found — use Go-native /api/* routes",
		"path":    r.URL.Path,
	})
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
		requestID := incomingRequestID(r)
		w.Header().Set("X-Request-ID", requestID)
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			if recovered := recover(); recovered != nil {
				rec.status = http.StatusInternalServerError
				http.Error(rec, "internal server error", http.StatusInternalServerError)
				logStructured(map[string]any{
					"level":      "error",
					"event":      "panic",
					"request_id": requestID,
					"method":     r.Method,
					"path":       r.URL.Path,
					"panic":      recovered,
				})
			}

			duration := time.Since(start)
			recordMetrics(r.Method+" "+r.URL.Path, rec.status, duration)
			logStructured(map[string]any{
				"level":          "info",
				"event":          "http_request",
				"request_id":     requestID,
				"method":         r.Method,
				"path":           r.URL.Path,
				"query":          r.URL.RawQuery,
				"status":         rec.status,
				"duration_ms":    duration.Milliseconds(),
				"response_bytes": rec.bytes,
				"remote_addr":    r.RemoteAddr,
			})
		}()

		next.ServeHTTP(rec, r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID)))
	})
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func isFileInside(candidate, root string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func incomingRequestID(r *http.Request) string {
	if got := strings.TrimSpace(r.Header.Get("X-Request-ID")); got != "" {
		return got
	}
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	return "request-id-unavailable"
}

func recordMetrics(route string, status int, duration time.Duration) {
	requestMetrics.mu.Lock()
	defer requestMetrics.mu.Unlock()

	requestMetrics.Requests++
	metric := requestMetrics.RouteData[route]
	metric.Count++
	metric.TotalDuration += duration
	if duration > metric.MaxDuration {
		metric.MaxDuration = duration
	}
	if status >= http.StatusBadRequest {
		requestMetrics.Errors++
		metric.Errors++
	}
	requestMetrics.RouteData[route] = metric
}

func logStructured(payload map[string]any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		log.Printf(`{"level":"error","event":"log_encode_failed","error":%q}`, err.Error())
		return
	}
	log.Print(string(raw))
}
