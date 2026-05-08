package headless

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	"github.com/go-mirofish/go-mirofish/gateway/internal/telemetry"
	intworker "github.com/go-mirofish/go-mirofish/gateway/internal/worker"
	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
	pluginstarlark "github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/starlark"
	plugintrust "github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/trust"
	pluginwasm "github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/wasm"
)

// Config controls the embedded headless gateway wiring.
type Config struct {
	BindHost        string
	Port            string
	FrontendDevURL  *url.URL
	FrontendDistDir string
	ProjectsDir     string
	ReportsDir      string
	TasksDir        string
	SimulationsDir  string
	ScriptsDir      string
}

// App is an embeddable headless gateway instance.
type App struct {
	Config   Config
	handler  http.Handler
	registry *intprovider.Registry
}

// WasmPlugin wraps a compiled Wasm plugin and its manifest.
type WasmPlugin struct {
	Manifest plugins.Manifest
	compiled *pluginwasm.Compiled
	runtime  *pluginwasm.Runtime
}

// StarlarkPlugin wraps a loaded Starlark plugin and its manifest.
type StarlarkPlugin struct {
	Manifest plugins.Manifest
	program  *pluginstarlark.Program
	runtime  *pluginstarlark.Runtime
}

// Plugin is the runtime-neutral loaded plugin surface exposed by the headless SDK.
type Plugin interface {
	PluginManifest() plugins.Manifest
	Invoke(context.Context, []byte) (plugins.Result, error)
}

// WasmManager coordinates Wasm plugin discovery, loading, and invocation.
type WasmManager struct {
	mu       sync.RWMutex
	runtime  *pluginwasm.Runtime
	registry *plugins.Registry
	loaded   map[string]*WasmPlugin
	trust    *plugintrust.Policy
}

// PluginManager coordinates runtime-neutral plugin discovery, loading, and invocation.
type PluginManager struct {
	mu              sync.RWMutex
	registry        *plugins.Registry
	wasmRuntime     *pluginwasm.Runtime
	starlarkRuntime *pluginstarlark.Runtime
	loaded          map[string]Plugin
	trust           *plugintrust.Policy
}

type reportLookup struct {
	store *localfs.Store
}

type routeLimiter struct {
	key        string
	concurrent chan struct{}
	limiter    *tokenBucket
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

// LoadConfigFromEnv loads the default headless configuration from the same
// environment variables used by the gateway binary.
func LoadConfigFromEnv() (Config, error) {
	repoRoot, haveRepo := findRepoRoot()

	var frontendDevURL *url.URL
	if raw := strings.TrimSpace(os.Getenv("FRONTEND_DEV_URL")); raw != "" {
		parsed, err := url.Parse(raw)
		if err != nil {
			return Config{}, err
		}
		frontendDevURL = parsed
	}

	return Config{
		BindHost:        envOrDefault("GATEWAY_BIND_HOST", "127.0.0.1"),
		Port:            envOrDefault("GATEWAY_PORT", "3000"),
		FrontendDevURL:  frontendDevURL,
		FrontendDistDir: envOrDefaultAnchored("FRONTEND_DIST_DIR", "frontend/dist", repoRoot, haveRepo),
		ProjectsDir:     envOrDefaultAnchored("PROJECTS_DIR", "data/projects", repoRoot, haveRepo),
		ReportsDir:      envOrDefaultAnchored("REPORTS_DIR", "data/reports", repoRoot, haveRepo),
		TasksDir:        envOrDefaultAnchored("TASKS_DIR", "data/tasks", repoRoot, haveRepo),
		SimulationsDir:  envOrDefaultAnchored("SIMULATIONS_DIR", "data/simulations", repoRoot, haveRepo),
		ScriptsDir:      envOrDefaultAnchored("SCRIPTS_DIR", "scripts", repoRoot, haveRepo),
	}, nil
}

// New creates a new embeddable headless gateway application.
func New(cfg Config) (*App, error) {
	if err := validateStartup(cfg); err != nil {
		return nil, err
	}

	registry := buildProviderRegistry(llmTimeout())
	handler := buildHandler(cfg, registry)

	return &App{
		Config:   cfg,
		handler:  handler,
		registry: registry,
	}, nil
}

// Handler returns the fully wired HTTP handler for embedding into an existing
// server or mux.
func (a *App) Handler() http.Handler {
	return a.handler
}

// NewServer creates a standard http.Server using the configured bind address.
func (a *App) NewServer() *http.Server {
	return &http.Server{
		Addr:              net.JoinHostPort(a.Config.BindHost, a.Config.Port),
		Handler:           a.handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

// ListenAndServe serves the headless app until the provided context is
// cancelled or the server exits.
func (a *App) ListenAndServe(ctx context.Context) error {
	server := a.NewServer()
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-serverErr
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

// Run loads config from env, builds the app, and serves it until context
// cancellation.
func Run(ctx context.Context) error {
	cfg, err := LoadConfigFromEnv()
	if err != nil {
		return err
	}
	app, err := New(cfg)
	if err != nil {
		return err
	}
	return app.ListenAndServe(ctx)
}

// NewWasmRuntime creates a plugin runtime using the same Go-native process.
func NewWasmRuntime(ctx context.Context, cfg pluginwasm.Config) (*pluginwasm.Runtime, error) {
	return pluginwasm.NewRuntime(ctx, cfg)
}

// LoadTrustPolicyFile reads a JSON trust policy used by trusted plugin loaders.
func LoadTrustPolicyFile(path string) (plugintrust.Policy, error) {
	return plugintrust.LoadPolicyFile(path)
}

// LoadWasmPluginFromBytes parses a manifest, compiles guest Wasm, and validates the ABI.
func LoadWasmPluginFromBytes(ctx context.Context, runtime *pluginwasm.Runtime, manifestRaw []byte, wasmBytes []byte) (*WasmPlugin, error) {
	if runtime == nil {
		return nil, errors.New("wasm runtime is required")
	}
	manifest, err := plugins.ParseManifest(manifestRaw)
	if err != nil {
		return nil, err
	}
	if plugins.NormalizeRuntime(manifest.Runtime) != plugins.RuntimeWasm {
		return nil, fmt.Errorf("plugin runtime mismatch: got %q", manifest.Runtime)
	}
	manifest = normalizePluginManifest(manifest)
	compiled, err := runtime.Compile(ctx, wasmBytes)
	if err != nil {
		return nil, err
	}
	contract := pluginwasm.ContractFromManifest(manifest)
	if err := compiled.Validate(ctx, contract); err != nil {
		return nil, err
	}
	return &WasmPlugin{
		Manifest: manifest,
		compiled: compiled,
		runtime:  runtime,
	}, nil
}

// LoadWasmPluginFromFiles loads a plugin manifest and Wasm guest from disk.
func LoadWasmPluginFromFiles(ctx context.Context, runtime *pluginwasm.Runtime, manifestPath, wasmPath string) (*WasmPlugin, error) {
	if runtime == nil {
		return nil, errors.New("wasm runtime is required")
	}
	manifest, err := plugins.LoadManifestFile(manifestPath)
	if err != nil {
		return nil, err
	}
	if plugins.NormalizeRuntime(manifest.Runtime) != plugins.RuntimeWasm {
		return nil, fmt.Errorf("plugin runtime mismatch: got %q", manifest.Runtime)
	}
	manifest = normalizePluginManifest(manifest)
	wasmRaw, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, err
	}
	compiled, err := runtime.Compile(ctx, wasmRaw)
	if err != nil {
		return nil, err
	}
	contract := pluginwasm.ContractFromManifest(manifest)
	if err := compiled.Validate(ctx, contract); err != nil {
		return nil, err
	}
	return &WasmPlugin{
		Manifest: manifest,
		compiled: compiled,
		runtime:  runtime,
	}, nil
}

// LoadWasmPluginFromFilesTrusted verifies a plugin against a trust policy before loading it.
func LoadWasmPluginFromFilesTrusted(ctx context.Context, runtime *pluginwasm.Runtime, manifestPath, wasmPath string, policy plugintrust.Policy) (*WasmPlugin, error) {
	if runtime == nil {
		return nil, errors.New("wasm runtime is required")
	}
	manifest, err := plugins.LoadManifestFile(manifestPath)
	if err != nil {
		return nil, err
	}
	wasmRaw, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, err
	}
	if err := plugintrust.VerifyManifestAndModule(policy, manifest, wasmRaw); err != nil {
		return nil, err
	}
	return LoadWasmPluginFromBytes(ctx, runtime, mustMarshalManifest(manifest), wasmRaw)
}

// LoadWasmPluginFromDir loads a plugin from a directory containing manifest.json and the guest module.
func LoadWasmPluginFromDir(ctx context.Context, runtime *pluginwasm.Runtime, dir string) (*WasmPlugin, error) {
	if runtime == nil {
		return nil, errors.New("wasm runtime is required")
	}
	compiled, manifest, err := runtime.LoadFromDir(ctx, dir)
	if err != nil {
		return nil, err
	}
	return &WasmPlugin{
		Manifest: manifest,
		compiled: compiled,
		runtime:  runtime,
	}, nil
}

// LoadWasmPluginFromDirTrusted discovers, verifies, and loads a plugin directory.
func LoadWasmPluginFromDirTrusted(ctx context.Context, runtime *pluginwasm.Runtime, dir string, policy plugintrust.Policy) (*WasmPlugin, error) {
	reg, err := plugins.DiscoverDirectory(dir)
	if err != nil {
		return nil, err
	}
	return LoadWasmPluginFromFilesTrusted(ctx, runtime, reg.ManifestPath, reg.ModulePath, policy)
}

// LoadStarlarkPluginFromBytes parses a manifest and loads Starlark guest source.
func LoadStarlarkPluginFromBytes(runtime *pluginstarlark.Runtime, manifestRaw []byte, source []byte) (*StarlarkPlugin, error) {
	if runtime == nil {
		return nil, errors.New("starlark runtime is required")
	}
	manifest, err := plugins.ParseManifest(manifestRaw)
	if err != nil {
		return nil, err
	}
	if plugins.NormalizeRuntime(manifest.Runtime) != plugins.RuntimeStarlark {
		return nil, fmt.Errorf("plugin runtime mismatch: got %q", manifest.Runtime)
	}
	manifest = normalizePluginManifest(manifest)
	program, err := runtime.LoadFromBytes(manifest, source)
	if err != nil {
		return nil, err
	}
	return &StarlarkPlugin{
		Manifest: manifest,
		program:  program,
		runtime:  runtime,
	}, nil
}

// LoadStarlarkPluginFromFiles loads a plugin manifest and Starlark source from disk.
func LoadStarlarkPluginFromFiles(runtime *pluginstarlark.Runtime, manifestPath, sourcePath string) (*StarlarkPlugin, error) {
	if runtime == nil {
		return nil, errors.New("starlark runtime is required")
	}
	manifest, err := plugins.LoadManifestFile(manifestPath)
	if err != nil {
		return nil, err
	}
	if plugins.NormalizeRuntime(manifest.Runtime) != plugins.RuntimeStarlark {
		return nil, fmt.Errorf("plugin runtime mismatch: got %q", manifest.Runtime)
	}
	manifest = normalizePluginManifest(manifest)
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	program, err := runtime.LoadFromBytes(manifest, source)
	if err != nil {
		return nil, err
	}
	return &StarlarkPlugin{
		Manifest: manifest,
		program:  program,
		runtime:  runtime,
	}, nil
}

// LoadStarlarkPluginFromFilesTrusted verifies a plugin against a trust policy before loading it.
func LoadStarlarkPluginFromFilesTrusted(runtime *pluginstarlark.Runtime, manifestPath, sourcePath string, policy plugintrust.Policy) (*StarlarkPlugin, error) {
	if runtime == nil {
		return nil, errors.New("starlark runtime is required")
	}
	manifest, err := plugins.LoadManifestFile(manifestPath)
	if err != nil {
		return nil, err
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	if err := plugintrust.VerifyManifestAndModule(policy, manifest, source); err != nil {
		return nil, err
	}
	return LoadStarlarkPluginFromBytes(runtime, mustMarshalManifest(manifest), source)
}

// LoadStarlarkPluginFromDir loads a plugin from a directory containing manifest.json and Starlark source.
func LoadStarlarkPluginFromDir(runtime *pluginstarlark.Runtime, dir string) (*StarlarkPlugin, error) {
	if runtime == nil {
		return nil, errors.New("starlark runtime is required")
	}
	program, err := runtime.LoadFromDir(dir)
	if err != nil {
		return nil, err
	}
	return &StarlarkPlugin{
		Manifest: program.Manifest,
		program:  program,
		runtime:  runtime,
	}, nil
}

// LoadStarlarkPluginFromDirTrusted discovers, verifies, and loads a plugin directory.
func LoadStarlarkPluginFromDirTrusted(runtime *pluginstarlark.Runtime, dir string, policy plugintrust.Policy) (*StarlarkPlugin, error) {
	reg, err := plugins.DiscoverDirectory(dir)
	if err != nil {
		return nil, err
	}
	return LoadStarlarkPluginFromFilesTrusted(runtime, reg.ManifestPath, reg.ModulePath, policy)
}

// NewWasmManager creates a Wasm plugin manager over a shared runtime.
func NewWasmManager(runtime *pluginwasm.Runtime) (*WasmManager, error) {
	if runtime == nil {
		return nil, errors.New("wasm runtime is required")
	}
	return &WasmManager{
		runtime:  runtime,
		registry: plugins.NewRegistry(),
		loaded:   map[string]*WasmPlugin{},
	}, nil
}

// NewTrustedWasmManager creates a Wasm manager with a trust policy.
func NewTrustedWasmManager(runtime *pluginwasm.Runtime, policy plugintrust.Policy) (*WasmManager, error) {
	manager, err := NewWasmManager(runtime)
	if err != nil {
		return nil, err
	}
	manager.trust = &policy
	return manager, nil
}

// NewTrustedWasmManagerFromFile loads a trust policy from disk before creating a manager.
func NewTrustedWasmManagerFromFile(runtime *pluginwasm.Runtime, policyPath string) (*WasmManager, error) {
	policy, err := LoadTrustPolicyFile(policyPath)
	if err != nil {
		return nil, err
	}
	return NewTrustedWasmManager(runtime, policy)
}

// NewPluginManager creates a runtime-neutral plugin manager over shared runtimes.
func NewPluginManager(wasmRuntime *pluginwasm.Runtime, starlarkRuntime *pluginstarlark.Runtime) (*PluginManager, error) {
	if wasmRuntime == nil && starlarkRuntime == nil {
		return nil, errors.New("at least one plugin runtime is required")
	}
	return &PluginManager{
		registry:        plugins.NewRegistry(),
		wasmRuntime:     wasmRuntime,
		starlarkRuntime: starlarkRuntime,
		loaded:          map[string]Plugin{},
	}, nil
}

// NewTrustedPluginManager creates a runtime-neutral plugin manager with a trust policy.
func NewTrustedPluginManager(wasmRuntime *pluginwasm.Runtime, starlarkRuntime *pluginstarlark.Runtime, policy plugintrust.Policy) (*PluginManager, error) {
	manager, err := NewPluginManager(wasmRuntime, starlarkRuntime)
	if err != nil {
		return nil, err
	}
	manager.trust = &policy
	return manager, nil
}

// NewTrustedPluginManagerFromFile loads a trust policy from disk before creating a manager.
func NewTrustedPluginManagerFromFile(wasmRuntime *pluginwasm.Runtime, starlarkRuntime *pluginstarlark.Runtime, policyPath string) (*PluginManager, error) {
	policy, err := LoadTrustPolicyFile(policyPath)
	if err != nil {
		return nil, err
	}
	return NewTrustedPluginManager(wasmRuntime, starlarkRuntime, policy)
}

// RegisterDir discovers and registers a plugin directory.
func (m *WasmManager) RegisterDir(dir string) (plugins.Registration, error) {
	if m == nil || m.registry == nil {
		return plugins.Registration{}, errors.New("wasm manager is not initialized")
	}
	return m.registry.RegisterDirectory(dir)
}

// RegisterDirs registers multiple plugin directories.
func (m *WasmManager) RegisterDirs(dirs ...string) error {
	for _, dir := range dirs {
		if _, err := m.RegisterDir(dir); err != nil {
			return err
		}
	}
	return nil
}

// List returns registered plugin metadata.
func (m *WasmManager) List() []plugins.Registration {
	if m == nil || m.registry == nil {
		return nil
	}
	return m.registry.List()
}

// LoadByName compiles and validates a registered plugin by name.
func (m *WasmManager) LoadByName(ctx context.Context, name string) (*WasmPlugin, error) {
	if m == nil || m.registry == nil {
		return nil, errors.New("wasm manager is not initialized")
	}
	m.mu.RLock()
	loaded := m.loaded[name]
	m.mu.RUnlock()
	if loaded != nil {
		return loaded, nil
	}
	reg, ok := m.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("plugin %q is not registered", name)
	}
	var (
		plugin *WasmPlugin
		err    error
	)
	if m.trust != nil {
		plugin, err = LoadWasmPluginFromFilesTrusted(ctx, m.runtime, reg.ManifestPath, reg.ModulePath, *m.trust)
	} else {
		plugin, err = LoadWasmPluginFromFiles(ctx, m.runtime, reg.ManifestPath, reg.ModulePath)
	}
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.loaded[name] = plugin
	m.mu.Unlock()
	return plugin, nil
}

// InvokeByName loads a registered plugin by name and invokes it.
func (m *WasmManager) InvokeByName(ctx context.Context, name string, input []byte) (plugins.Result, error) {
	plugin, err := m.LoadByName(ctx, name)
	if err != nil {
		return plugins.Result{}, err
	}
	return plugin.Invoke(ctx, input)
}

// RegisterDir discovers and registers a plugin directory.
func (m *PluginManager) RegisterDir(dir string) (plugins.Registration, error) {
	if m == nil || m.registry == nil {
		return plugins.Registration{}, errors.New("plugin manager is not initialized")
	}
	return m.registry.RegisterDirectory(dir)
}

// RegisterDirs registers multiple plugin directories.
func (m *PluginManager) RegisterDirs(dirs ...string) error {
	for _, dir := range dirs {
		if _, err := m.RegisterDir(dir); err != nil {
			return err
		}
	}
	return nil
}

// List returns registered plugin metadata.
func (m *PluginManager) List() []plugins.Registration {
	if m == nil || m.registry == nil {
		return nil
	}
	return m.registry.List()
}

// LoadByName loads a registered plugin by name using the runtime declared in its manifest.
func (m *PluginManager) LoadByName(ctx context.Context, name string) (Plugin, error) {
	if m == nil || m.registry == nil {
		return nil, errors.New("plugin manager is not initialized")
	}
	m.mu.RLock()
	loaded := m.loaded[name]
	m.mu.RUnlock()
	if loaded != nil {
		return loaded, nil
	}
	reg, ok := m.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("plugin %q is not registered", name)
	}
	plugin, err := m.loadRegistration(ctx, reg)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.loaded[name] = plugin
	m.mu.Unlock()
	return plugin, nil
}

// InvokeByName loads a registered plugin by name and invokes it.
func (m *PluginManager) InvokeByName(ctx context.Context, name string, input []byte) (plugins.Result, error) {
	plugin, err := m.LoadByName(ctx, name)
	if err != nil {
		return plugins.Result{}, err
	}
	return plugin.Invoke(ctx, input)
}

// PluginManifestJSON returns the normalized manifest as JSON.
func (p *WasmPlugin) PluginManifestJSON() ([]byte, error) {
	if p == nil {
		return nil, errors.New("plugin is not loaded")
	}
	return json.MarshalIndent(p.Manifest, "", "  ")
}

// PluginManifest returns the normalized manifest for the loaded plugin.
func (p *WasmPlugin) PluginManifest() plugins.Manifest {
	if p == nil {
		return plugins.Manifest{}
	}
	return p.Manifest
}

// Invoke runs a loaded Wasm plugin with the manifest-declared contract and capabilities.
func (p *WasmPlugin) Invoke(ctx context.Context, input []byte) (plugins.Result, error) {
	if p == nil || p.compiled == nil {
		return plugins.Result{}, errors.New("plugin is not loaded")
	}
	contract := pluginwasm.ContractFromManifest(p.Manifest)
	return p.compiled.InvokeWithCapabilities(ctx, input, contract, p.Manifest.Capabilities)
}

// PluginManifestJSON returns the normalized manifest as JSON.
func (p *StarlarkPlugin) PluginManifestJSON() ([]byte, error) {
	if p == nil {
		return nil, errors.New("plugin is not loaded")
	}
	return json.MarshalIndent(p.Manifest, "", "  ")
}

// PluginManifest returns the normalized manifest for the loaded plugin.
func (p *StarlarkPlugin) PluginManifest() plugins.Manifest {
	if p == nil {
		return plugins.Manifest{}
	}
	return p.Manifest
}

// Invoke runs a loaded Starlark plugin with the manifest-declared capabilities.
func (p *StarlarkPlugin) Invoke(ctx context.Context, input []byte) (plugins.Result, error) {
	if p == nil || p.program == nil {
		return plugins.Result{}, errors.New("plugin is not loaded")
	}
	return p.program.InvokeWithCapabilities(ctx, input, p.Manifest.Capabilities)
}

func (m *PluginManager) loadRegistration(ctx context.Context, reg plugins.Registration) (Plugin, error) {
	switch plugins.NormalizeRuntime(reg.Manifest.Runtime) {
	case plugins.RuntimeWasm:
		if m.wasmRuntime == nil {
			return nil, errors.New("wasm runtime is not configured")
		}
		if m.trust != nil {
			return LoadWasmPluginFromFilesTrusted(ctx, m.wasmRuntime, reg.ManifestPath, reg.ModulePath, *m.trust)
		}
		return LoadWasmPluginFromFiles(ctx, m.wasmRuntime, reg.ManifestPath, reg.ModulePath)
	case plugins.RuntimeStarlark:
		if m.starlarkRuntime == nil {
			return nil, errors.New("starlark runtime is not configured")
		}
		if m.trust != nil {
			return LoadStarlarkPluginFromFilesTrusted(m.starlarkRuntime, reg.ManifestPath, reg.ModulePath, *m.trust)
		}
		return LoadStarlarkPluginFromFiles(m.starlarkRuntime, reg.ManifestPath, reg.ModulePath)
	default:
		return nil, fmt.Errorf("unsupported plugin runtime %q", reg.Manifest.Runtime)
	}
}

func normalizePluginManifest(manifest plugins.Manifest) plugins.Manifest {
	manifest.Runtime = plugins.NormalizeRuntime(manifest.Runtime)
	if strings.TrimSpace(manifest.Module) == "" {
		if module, err := plugins.DefaultModuleForRuntime(manifest.Runtime); err == nil {
			manifest.Module = module
		}
	}
	return manifest
}

func mustMarshalManifest(manifest plugins.Manifest) []byte {
	raw, err := json.Marshal(manifest)
	if err != nil {
		panic(err)
	}
	return raw
}

func (r reportLookup) ReadSimulation(simulationID string) (map[string]any, error) {
	return r.store.ReadSimulation(simulationID)
}

func (r reportLookup) ReadProject(projectID string) (map[string]any, error) {
	return r.store.ReadProject(projectID)
}

func buildHandler(cfg Config, registry *intprovider.Registry) http.Handler {
	var frontendDevProxy *httputil.ReverseProxy
	if cfg.FrontendDevURL != nil {
		frontendDevProxy = httputil.NewSingleHostReverseProxy(cfg.FrontendDevURL)
		frontendDevProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("frontend proxy error for %s: %v", r.URL.Path, err)
			http.Error(w, "frontend unavailable", http.StatusBadGateway)
		}
	}

	appHandler := apphttp.New(frontendDevProxy, cfg.FrontendDistDir)
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
	mux.HandleFunc("/api/", appHandler.HandleAPIProxy)
	mux.HandleFunc("/", appHandler.HandleFrontend)

	return apphttp.RequestLogger(mux)
}

func buildReportHandler(cfg Config, registry *intprovider.Registry) *reporthttp.Handler {
	llmModel := strings.TrimSpace(os.Getenv("LLM_MODEL_NAME"))
	providerExec := registry.Executor()

	zepBase := strings.TrimSpace(os.Getenv("ZEP_API_URL"))
	if zepBase == "" {
		zepBase = "https://api.getzep.com/api/v2"
	} else {
		zepBase = strings.TrimRight(zepBase, "/") + "/api/v2"
	}

	memoryClient := intmemory.NewZepClient(zepBase, strings.TrimSpace(os.Getenv("ZEP_API_KEY")), nil)
	store := reportstore.NewFileStore(cfg.ReportsDir)
	planner := intreport.NewPlanner(providerExec, llmModel)
	service := intreport.NewService(store, memoryClient, planner)
	lookupStore := localfs.New(cfg.ProjectsDir, cfg.ReportsDir, cfg.TasksDir, cfg.SimulationsDir, cfg.ScriptsDir)

	return reporthttp.NewHandler(service, reportLookup{store: lookupStore}, llmModel, providerExec)
}

func buildGraphHandler(cfg Config) *graphhttp.Handler {
	return graphhttp.NewHandler(buildGraphService(cfg))
}

func buildGraphService(cfg Config) *intgraph.Service {
	zepBase := strings.TrimSpace(os.Getenv("ZEP_API_URL"))
	if zepBase == "" {
		zepBase = "https://api.getzep.com/api/v2"
	} else {
		zepBase = strings.TrimRight(zepBase, "/") + "/api/v2"
	}
	memoryClient := intmemory.NewZepClient(zepBase, strings.TrimSpace(os.Getenv("ZEP_API_KEY")), nil)
	store := graphstore.New(cfg.TasksDir, cfg.ProjectsDir)
	return intgraph.NewService(store, memoryClient)
}

func buildOntologyHandler(cfg Config, registry *intprovider.Registry) *ontologyhttp.Handler {
	store := localfs.New(cfg.ProjectsDir, cfg.ReportsDir, cfg.TasksDir, cfg.SimulationsDir, cfg.ScriptsDir)
	if exec := registry.Executor(); exec != nil {
		return ontologyhttp.NewHandlerWithExecutor(store, nil, apphttp.WriteJSON, exec)
	}
	return ontologyhttp.NewHandler(store, nil, apphttp.WriteJSON)
}

func buildSimulationHandler(cfg Config) *simulationhttp.Handler {
	store := simulationstore.New(cfg.SimulationsDir, cfg.ScriptsDir, cfg.ProjectsDir, cfg.ReportsDir)
	bridge := intworker.NewNativeBridge(cfg.SimulationsDir)
	service := intsimulation.NewService(store, bridge)
	return simulationhttp.NewHandler(service, store, buildGraphService(cfg))
}

func buildPrepareHandler(cfg Config, registry *intprovider.Registry) *preparehttp.Handler {
	store := localfs.New(cfg.ProjectsDir, cfg.ReportsDir, cfg.TasksDir, cfg.SimulationsDir, cfg.ScriptsDir)
	if exec := registry.Executor(); exec != nil {
		return preparehttp.NewHandlerWithExecutor(store, buildGraphService(cfg), apphttp.WriteJSON, exec)
	}
	return preparehttp.NewHandler(store, buildGraphService(cfg), apphttp.WriteJSON)
}

func providerPoolHandler(reg *intprovider.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apphttp.WriteJSON(w, http.StatusOK, map[string]any{
			"providers": reg.PoolInfo(),
			"success":   true,
		})
	}
}

func llmTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("LLM_TIMEOUT_SECONDS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 300 * time.Second
}

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

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func validateStartup(cfg Config) error {
	for _, dir := range []string{cfg.ProjectsDir, cfg.ReportsDir, cfg.TasksDir, cfg.SimulationsDir} {
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
	if cfg.FrontendDevURL == nil {
		indexPath := filepath.Join(cfg.FrontendDistDir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			return fmt.Errorf("frontend assets unavailable: %w", err)
		}
	}
	return nil
}

func readinessChecker(cfg Config) apphttp.ReadinessFunc {
	return func(_ context.Context) (map[string]any, error) {
		checks := map[string]any{
			"worker_runtime": "native",
			"stack":          "go",
		}
		for _, dir := range []string{cfg.ProjectsDir, cfg.ReportsDir, cfg.TasksDir, cfg.SimulationsDir} {
			if err := validateWritableDir(dir); err != nil {
				checks[dir] = err.Error()
				return checks, err
			}
			checks[dir] = "writable"
		}
		if cfg.FrontendDevURL != nil {
			checks["frontend"] = cfg.FrontendDevURL.String()
			return checks, nil
		}
		indexPath := filepath.Join(cfg.FrontendDistDir, "index.html")
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
