package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"strconv"

	intgraph "github.com/go-mirofish/go-mirofish/gateway/internal/graph"
	apphttp "github.com/go-mirofish/go-mirofish/gateway/internal/http/app"
	graphhttp "github.com/go-mirofish/go-mirofish/gateway/internal/http/graph"
	ontologyhttp "github.com/go-mirofish/go-mirofish/gateway/internal/http/ontology"
	preparehttp "github.com/go-mirofish/go-mirofish/gateway/internal/http/prepare"
	reporthttp "github.com/go-mirofish/go-mirofish/gateway/internal/http/report"
	simulationhttp "github.com/go-mirofish/go-mirofish/gateway/internal/http/simulation"
	intmemory "github.com/go-mirofish/go-mirofish/gateway/internal/memory"
	intprovider "github.com/go-mirofish/go-mirofish/gateway/internal/provider"
	intreport "github.com/go-mirofish/go-mirofish/gateway/internal/report"
	intsimulation "github.com/go-mirofish/go-mirofish/gateway/internal/simulation"
	graphstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/graph"
	localfs "github.com/go-mirofish/go-mirofish/gateway/internal/store/localfs"
	reportstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/report"
	simulationstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/simulation"
	"github.com/go-mirofish/go-mirofish/gateway/internal/telemetry"
	intworker "github.com/go-mirofish/go-mirofish/gateway/internal/worker"
)

type config struct {
	bindHost        string
	port            string
	frontendDevURL  *url.URL
	frontendDistDir string
	projectsDir     string
	reportsDir      string
	tasksDir        string
	simulationsDir  string
	scriptsDir      string
}

type gateway struct {
	cfg              config
	frontendDevProxy *httputil.ReverseProxy
}

type reportLookup struct {
	store *localfs.Store
}

type routeLimiter struct {
	key        string
	concurrent chan struct{}
	limiter    *tokenBucket
}

func (r reportLookup) ReadSimulation(simulationID string) (map[string]any, error) {
	return r.store.ReadSimulation(simulationID)
}

func (r reportLookup) ReadProject(projectID string) (map[string]any, error) {
	return r.store.ReadProject(projectID)
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("load gateway config: %v", err)
	}
	if err := validateStartup(cfg); err != nil {
		log.Fatalf("validate gateway startup: %v", err)
	}

	registry := buildProviderRegistry(llmTimeout())

	gw := newGateway(cfg)
	appHandler := apphttp.New(gw.frontendDevProxy, cfg.frontendDistDir)
	appHandler.Ready = readinessChecker(cfg)
	reportHandler := buildReportHandler(cfg, registry)
	graphHandler := buildGraphHandler(cfg)
	ontologyHandler := buildOntologyHandler(cfg, registry)
	prepareHandler := buildPrepareHandler(cfg, registry)
	simulationHandler := buildSimulationHandler(cfg)
	mux := http.NewServeMux()
	graphBuildLimiter := newRouteLimiter("graph_build", 2, 12, time.Minute)
	prepareLimiter := newRouteLimiter("simulation_prepare", 2, 24, time.Minute)
	reportLimiter := newRouteLimiter("report_generate", 2, 12, time.Minute)
	simulationWriteLimiter := newRouteLimiter("simulation_mutation", 4, 60, time.Minute)
	mux.HandleFunc("/health", appHandler.HandleHealth)
	mux.HandleFunc("/ready", appHandler.HandleReady)
	mux.HandleFunc("/metrics", appHandler.HandleMetrics)
	mux.HandleFunc("/api/providers", providerPoolHandler(registry))
	mux.HandleFunc("/api/graph/data/", graphHandler.HandleGraphData)
	mux.Handle("/api/graph/build", graphBuildLimiter.wrap(http.HandlerFunc(graphHandler.HandleBuild)))
	mux.HandleFunc("/api/graph/delete/", graphHandler.HandleDeleteGraph)
	mux.HandleFunc("/api/graph/ontology/generate", ontologyHandler.HandleGenerate)
	mux.HandleFunc("/api/graph/tasks", graphHandler.HandleTaskList)
	mux.HandleFunc("/api/graph/task/", graphHandler.HandleTaskGet)
	mux.HandleFunc("/api/graph/project/list", graphHandler.HandleProjectList)
	mux.HandleFunc("/api/graph/project/", graphHandler.HandleProjectRoute)
	mux.Handle("/api/report/", reportLimiter.wrap(http.HandlerFunc(reportHandler.HandleRoute)))
	mux.Handle("/api/simulation/generate-profiles", prepareLimiter.wrap(http.HandlerFunc(prepareHandler.HandleGenerateProfiles)))
	mux.Handle("/api/simulation/prepare/status", prepareLimiter.wrap(http.HandlerFunc(prepareHandler.HandlePrepareStatus)))
	mux.Handle("/api/simulation/prepare", prepareLimiter.wrap(http.HandlerFunc(prepareHandler.HandlePrepare)))
	mux.Handle("/api/simulation/", simulationWriteLimiter.wrap(http.HandlerFunc(simulationHandler.HandleRoute)))
	// /api/ catch-all: all product routes are Go-native above; unknown paths return 410.
	mux.HandleFunc("/api/", appHandler.HandleAPIProxy)
	mux.HandleFunc("/", appHandler.HandleFrontend)

	addr := net.JoinHostPort(cfg.bindHost, cfg.port)
	server := &http.Server{
		Addr:              addr,
		Handler:           apphttp.RequestLogger(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("mirofish gateway listening on http://%s", addr)
	if cfg.frontendDevURL != nil {
		log.Printf("frontend dev target: %s", cfg.frontendDevURL.String())
	} else {
		log.Printf("frontend dist dir: %s", cfg.frontendDistDir)
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-shutdownCtx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("gateway shutdown failed: %v", err)
		}
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("gateway server failed: %v", err)
		}
	}
}

func loadConfig() (config, error) {
	repoRoot, haveRepo := findRepoRoot()

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
		frontendDevURL:  frontendDevURL,
		frontendDistDir: envOrDefaultAnchored("FRONTEND_DIST_DIR", "frontend/dist", repoRoot, haveRepo),
		projectsDir:     envOrDefaultAnchored("PROJECTS_DIR", "data/projects", repoRoot, haveRepo),
		reportsDir:      envOrDefaultAnchored("REPORTS_DIR", "data/reports", repoRoot, haveRepo),
		tasksDir:        envOrDefaultAnchored("TASKS_DIR", "data/tasks", repoRoot, haveRepo),
		simulationsDir:  envOrDefaultAnchored("SIMULATIONS_DIR", "data/simulations", repoRoot, haveRepo),
		scriptsDir:      envOrDefaultAnchored("SCRIPTS_DIR", "scripts", repoRoot, haveRepo),
	}, nil
}

func newGateway(cfg config) *gateway {
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
		frontendDevProxy: frontendDevProxy,
	}
}

func (g *gateway) graphService() *intgraph.Service {
	return buildGraphService(g.cfg)
}

// llmTimeout reads LLM_TIMEOUT_SECONDS from the environment.
// Default: 300s — generous for local 7B models on consumer hardware.
// Cloud providers are fast and rarely approach this limit.
func llmTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("LLM_TIMEOUT_SECONDS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 300 * time.Second
}

// buildProviderRegistry creates the provider pool from env vars + auto-discovery.
func buildProviderRegistry(timeout time.Duration) *intprovider.Registry {
	autoDiscover := strings.ToLower(strings.TrimSpace(os.Getenv("LLM_AUTODISCOVER"))) != "false"

	var extraURLs []string
	if raw := strings.TrimSpace(os.Getenv("LLM_DISCOVER_EXTRA_URLS")); raw != "" {
		for _, u := range strings.Split(raw, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				extraURLs = append(extraURLs, u)
			}
		}
	}

	return intprovider.NewRegistry(intprovider.RegistryConfig{
		PrimaryURL:        strings.TrimRight(strings.TrimSpace(os.Getenv("LLM_BASE_URL")), "/"),
		PrimaryKey:        strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		PrimaryModel:      strings.TrimSpace(os.Getenv("LLM_MODEL_NAME")),
		BoostURL:          strings.TrimRight(strings.TrimSpace(os.Getenv("LLM_BOOST_BASE_URL")), "/"),
		BoostKey:          strings.TrimSpace(os.Getenv("LLM_BOOST_API_KEY")),
		BoostModel:        strings.TrimSpace(os.Getenv("LLM_BOOST_MODEL_NAME")),
		AutoDiscover:      autoDiscover,
		ExtraDiscoverURLs: extraURLs,
		Timeout:           timeout,
	})
}

// providerPoolHandler serves GET /api/providers — returns the current pool.
func providerPoolHandler(reg *intprovider.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apphttp.WriteJSON(w, http.StatusOK, map[string]any{
			"providers": reg.PoolInfo(),
			"success":   true,
		})
	}
}

func buildReportHandler(cfg config, registry *intprovider.Registry) *reporthttp.Handler {
	llmModel := strings.TrimSpace(os.Getenv("LLM_MODEL_NAME"))
	providerExec := registry.Executor()

	zepBase := strings.TrimSpace(os.Getenv("ZEP_API_URL"))
	if zepBase == "" {
		zepBase = "https://api.getzep.com/api/v2"
	} else {
		zepBase = strings.TrimRight(zepBase, "/") + "/api/v2"
	}

	memoryClient := intmemory.NewZepClient(zepBase, strings.TrimSpace(os.Getenv("ZEP_API_KEY")), nil)
	store := reportstore.NewFileStore(cfg.reportsDir)
	planner := intreport.NewPlanner(providerExec, llmModel)
	service := intreport.NewService(store, memoryClient, planner)
	lookupStore := localfs.New(cfg.projectsDir, cfg.reportsDir, cfg.tasksDir, cfg.simulationsDir, cfg.scriptsDir)

	return reporthttp.NewHandler(service, reportLookup{store: lookupStore}, llmModel, providerExec)
}

func buildGraphHandler(cfg config) *graphhttp.Handler {
	return graphhttp.NewHandler(buildGraphService(cfg))
}

func buildGraphService(cfg config) *intgraph.Service {
	zepBase := strings.TrimSpace(os.Getenv("ZEP_API_URL"))
	if zepBase == "" {
		zepBase = "https://api.getzep.com/api/v2"
	} else {
		zepBase = strings.TrimRight(zepBase, "/") + "/api/v2"
	}
	memoryClient := intmemory.NewZepClient(zepBase, strings.TrimSpace(os.Getenv("ZEP_API_KEY")), nil)
	store := graphstore.New(cfg.tasksDir, cfg.projectsDir)
	return intgraph.NewService(store, memoryClient)
}

func buildOntologyHandler(cfg config, registry *intprovider.Registry) *ontologyhttp.Handler {
	store := localfs.New(cfg.projectsDir, cfg.reportsDir, cfg.tasksDir, cfg.simulationsDir, cfg.scriptsDir)
	if exec := registry.Executor(); exec != nil {
		return ontologyhttp.NewHandlerWithExecutor(store, nil, apphttp.WriteJSON, exec)
	}
	return ontologyhttp.NewHandler(store, nil, apphttp.WriteJSON)
}

func buildSimulationHandler(cfg config) *simulationhttp.Handler {
	store := simulationstore.New(cfg.simulationsDir, cfg.scriptsDir, cfg.projectsDir, cfg.reportsDir)
	bridge := intworker.NewNativeBridge(cfg.simulationsDir)
	service := intsimulation.NewService(store, bridge)
	return simulationhttp.NewHandler(service, store, buildGraphService(cfg))
}

func buildPrepareHandler(cfg config, registry *intprovider.Registry) *preparehttp.Handler {
	store := localfs.New(cfg.projectsDir, cfg.reportsDir, cfg.tasksDir, cfg.simulationsDir, cfg.scriptsDir)
	if exec := registry.Executor(); exec != nil {
		return preparehttp.NewHandlerWithExecutor(store, buildGraphService(cfg), apphttp.WriteJSON, exec)
	}
	return preparehttp.NewHandler(store, buildGraphService(cfg), apphttp.WriteJSON)
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// findRepoRoot walks upward from the working directory to locate the repo root
// by looking for gateway/go.mod.
func findRepoRoot() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	dir := wd
	for i := 0; i < 20; i++ {
		gomod := filepath.Join(dir, "gateway", "go.mod")
		if st, err := os.Stat(gomod); err == nil && !st.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func anchorRepoPath(repoRoot string, haveRepo bool, p string) string {
	p = filepath.Clean(p)
	if !haveRepo || filepath.IsAbs(p) {
		return p
	}
	slash := filepath.ToSlash(p)
	if strings.HasPrefix(slash, "frontend/") || slash == "frontend" {
		return filepath.Join(repoRoot, p)
	}
	if strings.HasPrefix(slash, "data/") || slash == "data" {
		return filepath.Join(repoRoot, p)
	}
	if strings.HasPrefix(slash, "scripts/") || slash == "scripts" {
		return filepath.Join(repoRoot, p)
	}
	return p
}

func envOrDefaultAnchored(key, fallback, repoRoot string, haveRepo bool) string {
	return anchorRepoPath(repoRoot, haveRepo, envOrDefault(key, fallback))
}

func validateStartup(cfg config) error {
	// Enforce canonical data root — reject legacy backend/uploads paths.
	for _, dir := range []string{cfg.projectsDir, cfg.reportsDir, cfg.tasksDir, cfg.simulationsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		if err := validateWritableDir(dir); err != nil {
			return err
		}
		if strings.Contains(filepath.ToSlash(dir), "backend/uploads") {
			return fmt.Errorf("data path %q points to legacy backend/uploads — set *_DIR env vars to data/*", dir)
		}
	}
	if cfg.frontendDevURL == nil {
		indexPath := filepath.Join(cfg.frontendDistDir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			return fmt.Errorf("frontend assets unavailable: %w", err)
		}
	}
	return nil
}

func readinessChecker(cfg config) apphttp.ReadinessFunc {
	return func(_ context.Context) (map[string]any, error) {
		checks := map[string]any{
			"worker_runtime": "native",
			"stack":          "go",
		}
		for _, dir := range []string{cfg.projectsDir, cfg.reportsDir, cfg.tasksDir, cfg.simulationsDir} {
			if err := validateWritableDir(dir); err != nil {
				checks[dir] = err.Error()
				return checks, err
			}
			checks[dir] = "writable"
		}
		if cfg.frontendDevURL != nil {
			checks["frontend"] = cfg.frontendDevURL.String()
			return checks, nil
		}
		indexPath := filepath.Join(cfg.frontendDistDir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			checks["frontend"] = "missing dist"
			return checks, err
		}
		checks["frontend"] = "dist"
		return checks, nil
	}
}

func validateWritableDir(dir string) error {
	f, err := os.CreateTemp(dir, ".gateway-write-check-*")
	if err != nil {
		return fmt.Errorf("dir %s is not writable: %w", dir, err)
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

func fileStatus(path string) string {
	if _, err := os.Stat(path); err != nil {
		return err.Error()
	}
	return "ok"
}

func newRouteLimiter(key string, concurrency int, tokens int, refillEvery time.Duration) routeLimiter {
	return routeLimiter{
		key:        key,
		concurrent: make(chan struct{}, concurrency),
		limiter:    newTokenBucket(tokens, refillEvery),
	}
}

func (r routeLimiter) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet || req.Method == http.MethodHead || req.Method == http.MethodOptions {
			next.ServeHTTP(w, req)
			return
		}
		if !r.limiter.allow(req.RemoteAddr) {
			telemetry.RecordRateLimit(r.key, "requests_per_window_exceeded")
			apphttp.WriteJSON(w, http.StatusTooManyRequests, map[string]any{"success": false, "error": "rate limit exceeded"})
			return
		}
		select {
		case r.concurrent <- struct{}{}:
			defer func() { <-r.concurrent }()
		default:
			telemetry.RecordSaturation(r.key, "concurrency_limit_exceeded")
			apphttp.WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"success": false, "error": "route concurrency limit reached"})
			return
		}
		next.ServeHTTP(w, req)
	})
}

type tokenBucket struct {
	mu          sync.Mutex
	capacity    int
	refillEvery time.Duration
	entries     map[string]bucketEntry
}

type bucketEntry struct {
	count      int
	windowEnds time.Time
}

func newTokenBucket(capacity int, refillEvery time.Duration) *tokenBucket {
	return &tokenBucket{
		capacity:    capacity,
		refillEvery: refillEvery,
		entries:     map[string]bucketEntry{},
	}
}

func (b *tokenBucket) allow(identity string) bool {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		identity = "unknown"
	}
	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()
	entry := b.entries[identity]
	if entry.windowEnds.IsZero() || now.After(entry.windowEnds) {
		entry = bucketEntry{count: 0, windowEnds: now.Add(b.refillEvery)}
	}
	if entry.count >= b.capacity {
		b.entries[identity] = entry
		return false
	}
	entry.count++
	b.entries[identity] = entry
	return true
}
