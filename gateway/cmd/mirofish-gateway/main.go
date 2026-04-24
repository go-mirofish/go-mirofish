package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

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
	intworker "github.com/go-mirofish/go-mirofish/gateway/internal/worker"
)

type config struct {
	bindHost        string
	port            string
	backendURL      *url.URL
	frontendDevURL  *url.URL
	frontendDistDir string
	projectsDir     string
	reportsDir      string
	tasksDir        string
	simulationsDir  string
	scriptsDir      string
	pythonWorker    string
}

type gateway struct {
	cfg              config
	backendProxy     *httputil.ReverseProxy
	frontendDevProxy *httputil.ReverseProxy
}

type reportLookup struct {
	store *localfs.Store
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

	gw := newGateway(cfg)
	appHandler := apphttp.New(cfg.backendURL.String(), gw.backendProxy, gw.frontendDevProxy, cfg.frontendDistDir)
	reportHandler := buildReportHandler(cfg, gw)
	graphHandler := buildGraphHandler(cfg)
	ontologyHandler := buildOntologyHandler(cfg, gw)
	prepareHandler := buildPrepareHandler(cfg)
	simulationHandler := buildSimulationHandler(cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", appHandler.HandleHealth)
	mux.HandleFunc("/api/graph/data/", graphHandler.HandleGraphData)
	mux.HandleFunc("/api/graph/build", graphHandler.HandleBuild)
	mux.HandleFunc("/api/graph/delete/", graphHandler.HandleDeleteGraph)
	mux.HandleFunc("/api/graph/ontology/generate", ontologyHandler.HandleGenerate)
	mux.HandleFunc("/api/graph/tasks", graphHandler.HandleTaskList)
	mux.HandleFunc("/api/graph/task/", graphHandler.HandleTaskGet)
	mux.HandleFunc("/api/graph/project/list", graphHandler.HandleProjectList)
	mux.HandleFunc("/api/graph/project/", graphHandler.HandleProjectRoute)
	mux.HandleFunc("/api/report/", reportHandler.HandleRoute)
	mux.HandleFunc("/api/simulation/generate-profiles", prepareHandler.HandleGenerateProfiles)
	mux.HandleFunc("/api/simulation/prepare/status", prepareHandler.HandlePrepareStatus)
	mux.HandleFunc("/api/simulation/prepare", prepareHandler.HandlePrepare)
	mux.HandleFunc("/api/simulation/", simulationHandler.HandleRoute)
	mux.HandleFunc("/api/", appHandler.HandleAPIProxy)
	mux.HandleFunc("/", appHandler.HandleFrontend)

	addr := net.JoinHostPort(cfg.bindHost, cfg.port)
	server := &http.Server{
		Addr:              addr,
		Handler:           apphttp.RequestLogger(mux),
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
		reportsDir:      envOrDefault("REPORTS_DIR", "backend/uploads/reports"),
		tasksDir:        envOrDefault("TASKS_DIR", "backend/uploads/tasks"),
		simulationsDir:  envOrDefault("SIMULATIONS_DIR", "backend/uploads/simulations"),
		scriptsDir:      envOrDefault("SCRIPTS_DIR", "backend/scripts"),
		pythonWorker:    envOrDefault("PYTHON_WORKER", "/tmp/go-mirofish-bench-venv/bin/python"),
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

func (g *gateway) graphService() *intgraph.Service {
	return buildGraphService(g.cfg)
}

func buildReportHandler(cfg config, gw *gateway) *reporthttp.Handler {
	llmBase := strings.TrimRight(strings.TrimSpace(os.Getenv("LLM_BASE_URL")), "/")
	llmKey := strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	llmModel := strings.TrimSpace(os.Getenv("LLM_MODEL_NAME"))

	var providerExec intprovider.Executor
	if llmBase != "" && llmKey != "" && llmModel != "" {
		providerExec = intprovider.NewExecutor(intprovider.Config{
			BaseURL:      llmBase,
			APIKey:       llmKey,
			DefaultModel: llmModel,
			ProviderName: "openai-compatible",
		}, nil)
	}

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

func buildOntologyHandler(cfg config, gw *gateway) *ontologyhttp.Handler {
	store := localfs.New(cfg.projectsDir, cfg.reportsDir, cfg.tasksDir, cfg.simulationsDir, cfg.scriptsDir)
	appHandler := apphttp.New(cfg.backendURL.String(), gw.backendProxy, gw.frontendDevProxy, cfg.frontendDistDir)
	return ontologyhttp.NewHandler(store, appHandler.HandleAPIProxy, apphttp.WriteJSON)
}

func buildSimulationHandler(cfg config) *simulationhttp.Handler {
	store := simulationstore.New(cfg.simulationsDir, cfg.scriptsDir, cfg.projectsDir, cfg.reportsDir)
	service := intsimulation.NewService(store, intworker.NewLocalPythonBridge(cfg.simulationsDir, cfg.scriptsDir, cfg.pythonWorker))
	return simulationhttp.NewHandler(service, store, buildGraphService(cfg))
}

func buildPrepareHandler(cfg config) *preparehttp.Handler {
	store := localfs.New(cfg.projectsDir, cfg.reportsDir, cfg.tasksDir, cfg.simulationsDir, cfg.scriptsDir)
	return preparehttp.NewHandler(store, buildGraphService(cfg), apphttp.WriteJSON)
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
