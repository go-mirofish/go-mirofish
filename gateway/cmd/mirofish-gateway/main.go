package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type config struct {
	bindHost        string
	port            string
	backendURL      *url.URL
	frontendDevURL  *url.URL
	frontendDistDir string
	projectsDir     string
}

type gateway struct {
	cfg              config
	backendProxy     *httputil.ReverseProxy
	frontendDevProxy *httputil.ReverseProxy
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("load gateway config: %v", err)
	}

	gw := newGateway(cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", gw.handleHealth)
	mux.HandleFunc("/api/graph/project/list", gw.handleProjectList)
	mux.HandleFunc("/api/graph/project/", gw.handleProjectControlPlane)
	mux.HandleFunc("/api/report/generate/status", gw.handleReportStatusAlias)
	mux.HandleFunc("/api/simulation/run", gw.handleSimulationRunAlias)
	mux.HandleFunc("/api/simulation/", gw.handleSimulationStatusAlias)
	mux.HandleFunc("/api/", gw.handleAPIProxy)
	mux.HandleFunc("/", gw.handleFrontend)

	addr := net.JoinHostPort(cfg.bindHost, cfg.port)
	server := &http.Server{
		Addr:              addr,
		Handler:           requestLogger(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("mirofish gateway listening on http://%s", addr)
	log.Printf("backend target: %s", cfg.backendURL.String())
	if cfg.frontendDevURL != nil {
		log.Printf("frontend dev target: %s", cfg.frontendDevURL.String())
	} else {
		log.Printf("frontend dist dir: %s", cfg.frontendDistDir)
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("gateway server failed: %v", err)
	}
}

func loadConfig() (config, error) {
	backendURL, err := url.Parse(envOrDefault("BACKEND_URL", "http://127.0.0.1:5001"))
	if err != nil {
		return config{}, err
	}

	var frontendDevURL *url.URL
	if raw := strings.TrimSpace(os.Getenv("FRONTEND_DEV_URL")); raw != "" {
		parsed, err := url.Parse(raw)
		if err != nil {
			return config{}, err
		}
		frontendDevURL = parsed
	}

	return config{
		bindHost:        envOrDefault("GATEWAY_BIND_HOST", "127.0.0.1"),
		port:            envOrDefault("GATEWAY_PORT", "3000"),
		backendURL:      backendURL,
		frontendDevURL:  frontendDevURL,
		frontendDistDir: envOrDefault("FRONTEND_DIST_DIR", "frontend/dist"),
		projectsDir:     envOrDefault("PROJECTS_DIR", "backend/uploads/projects"),
	}, nil
}

func newGateway(cfg config) *gateway {
	backendProxy := httputil.NewSingleHostReverseProxy(cfg.backendURL)
	backendProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("backend proxy error for %s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "backend unavailable", http.StatusBadGateway)
	}

	var frontendDevProxy *httputil.ReverseProxy
	if cfg.frontendDevURL != nil {
		frontendDevProxy = httputil.NewSingleHostReverseProxy(cfg.frontendDevURL)
		frontendDevProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("frontend proxy error for %s: %v", r.URL.Path, err)
			http.Error(w, "frontend unavailable", http.StatusBadGateway)
		}
	}

	return &gateway{
		cfg:              cfg,
		backendProxy:     backendProxy,
		frontendDevProxy: frontendDevProxy,
	}
}

func (g *gateway) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "go-mirofish-gateway",
		"backend": g.cfg.backendURL.String(),
	})
}

func (g *gateway) handleAPIProxy(w http.ResponseWriter, r *http.Request) {
	g.backendProxy.ServeHTTP(w, r)
}

func (g *gateway) handleProjectList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	projects, err := g.listProjects(limit)
	if err != nil {
		log.Printf("project list failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    projects,
		"count":   len(projects),
	})
}

func (g *gateway) handleProjectControlPlane(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	const prefix = "api/graph/project/"
	if !strings.HasPrefix(trimmed, prefix) {
		g.handleAPIProxy(w, r)
		return
	}

	rest := strings.TrimPrefix(trimmed, prefix)
	if rest == "" || rest == "." {
		g.handleAPIProxy(w, r)
		return
	}

	if strings.HasSuffix(rest, "/reset") {
		projectID := strings.TrimSuffix(rest, "/reset")
		g.handleProjectReset(w, r, projectID)
		return
	}

	projectID := rest
	switch r.Method {
	case http.MethodGet:
		g.handleProjectGet(w, r, projectID)
	case http.MethodDelete:
		g.handleProjectDelete(w, r, projectID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (g *gateway) handleProjectGet(w http.ResponseWriter, _ *http.Request, projectID string) {
	project, err := g.readProject(projectID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"success": false,
				"error":   "Project not found: " + projectID,
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    project,
	})
}

func (g *gateway) handleProjectDelete(w http.ResponseWriter, _ *http.Request, projectID string) {
	projectDir := g.projectDir(projectID)
	if _, err := os.Stat(projectDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"success": false,
				"error":   "Project delete failed: " + projectID,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if err := os.RemoveAll(projectDir); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Project deleted: " + projectID,
	})
}

func (g *gateway) handleProjectReset(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project, err := g.readProject(projectID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"success": false,
				"error":   "Project not found: " + projectID,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if ontology, ok := project["ontology"]; ok && ontology != nil {
		project["status"] = "ontology_generated"
	} else {
		project["status"] = "created"
	}
	project["graph_id"] = nil
	project["graph_build_task_id"] = nil
	project["error"] = nil
	project["updated_at"] = time.Now().Format(time.RFC3339)

	if err := g.writeProject(projectID, project); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Project reset: " + projectID,
		"data":    project,
	})
}

func (g *gateway) handleSimulationRunAlias(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	g.proxyWithPath(w, r, "/api/simulation/start")
}

func (g *gateway) handleSimulationStatusAlias(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/status") {
		aliasPath := strings.TrimSuffix(r.URL.Path, "/status") + "/run-status"
		g.proxyWithPath(w, r, aliasPath)
		return
	}

	g.handleAPIProxy(w, r)
}

func (g *gateway) handleReportStatusAlias(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		g.proxyWithPath(w, r, "/api/report/generate/status")
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	if reportID := strings.TrimSpace(query.Get("report_id")); reportID != "" && query.Get("task_id") == "" && query.Get("simulation_id") == "" {
		g.proxyWithPath(w, r, "/api/report/"+url.PathEscape(reportID)+"/progress")
		return
	}

	payload := map[string]string{}
	if taskID := strings.TrimSpace(query.Get("task_id")); taskID != "" {
		payload["task_id"] = taskID
	}
	if simulationID := strings.TrimSpace(query.Get("simulation_id")); simulationID != "" {
		payload["simulation_id"] = simulationID
	}

	if len(payload) == 0 {
		http.Error(w, "report_id, task_id, or simulation_id is required", http.StatusBadRequest)
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "failed to encode report status payload", http.StatusInternalServerError)
		return
	}

	g.proxyWithJSONBody(w, r, "/api/report/generate/status", body)
}

func (g *gateway) handleFrontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if g.tryServeStatic(w, r) {
		return
	}

	if g.frontendDevProxy != nil {
		g.frontendDevProxy.ServeHTTP(w, r)
		return
	}

	http.Error(
		w,
		"frontend assets not found; build frontend/dist or set FRONTEND_DEV_URL",
		http.StatusServiceUnavailable,
	)
}

func (g *gateway) tryServeStatic(w http.ResponseWriter, r *http.Request) bool {
	distRoot := filepath.Clean(g.cfg.frontendDistDir)
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

func (g *gateway) proxyWithPath(w http.ResponseWriter, r *http.Request, newPath string) {
	req := r.Clone(r.Context())
	req.URL.Path = newPath
	req.URL.RawPath = ""
	g.backendProxy.ServeHTTP(w, req)
}

func (g *gateway) proxyWithJSONBody(w http.ResponseWriter, r *http.Request, newPath string, body []byte) {
	req := r.Clone(r.Context())
	req.Method = http.MethodPost
	req.URL.Path = newPath
	req.URL.RawPath = ""
	req.URL.RawQuery = ""
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.Header = req.Header.Clone()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", strconvItoa(len(body)))
	g.backendProxy.ServeHTTP(w, req)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func (g *gateway) projectDir(projectID string) string {
	return filepath.Join(g.cfg.projectsDir, projectID)
}

func (g *gateway) projectMetaPath(projectID string) string {
	return filepath.Join(g.projectDir(projectID), "project.json")
}

func (g *gateway) readProject(projectID string) (map[string]any, error) {
	raw, err := os.ReadFile(g.projectMetaPath(projectID))
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (g *gateway) writeProject(projectID string, payload map[string]any) error {
	metaPath := g.projectMetaPath(projectID)
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, raw, 0o644)
}

func (g *gateway) listProjects(limit int) ([]map[string]any, error) {
	if err := os.MkdirAll(g.cfg.projectsDir, 0o755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(g.cfg.projectsDir)
	if err != nil {
		return nil, err
	}

	projects := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		project, err := g.readProject(entry.Name())
		if err != nil {
			continue
		}
		projects = append(projects, project)
	}

	sort.Slice(projects, func(i, j int) bool {
		ic, _ := projects[i]["created_at"].(string)
		jc, _ := projects[j]["created_at"].(string)
		return ic > jc
	})

	if limit > 0 && len(projects) > limit {
		projects = projects[:limit]
	}
	return projects, nil
}

func isFileInside(candidate, root string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func strconvItoa(v int) string {
	return strconv.FormatInt(int64(v), 10)
}
