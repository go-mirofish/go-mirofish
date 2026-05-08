package wasm

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	DefaultABIVersion       = 1
	DefaultABIExportName    = "mirofish_abi_version"
	DefaultAllocateExport   = "allocate"
	DefaultDeallocateExport = "deallocate"
	DefaultInvokeExport     = "run"
	DefaultHostModuleName   = "mirofish"
	DefaultLogImportName    = "log"
	DefaultEmitEventImport  = "emit_event"
	DefaultTimeNowImport    = "time_now_unix_ms"
	DefaultRandomFillImport = "fill_random"
	defaultLogLevel         = "info"
)

type contextKey string

const executionStateKey contextKey = "mirofish_wasm_execution_state"

// Config defines the host runtime behavior.
type Config struct {
	HostModuleName      string
	LogImportName       string
	EmitEventImportName string
	TimeNowImportName   string
	RandomFillImport    string
	EnableWASI          bool
	RuntimeConfig       wazero.RuntimeConfig
}

// Contract defines the guest ABI expected by the host.
type Contract struct {
	ExpectedABIVersion   uint64
	RequireABIVersion    bool
	ABIExportName        string
	AllocateExportName   string
	DeallocateExportName string
	InvokeExportName     string
	StartFunctions       []string
	ExpectResultBuffer   bool
}

// ContractFromManifest maps a portable plugin manifest onto the Wasm runtime contract.
func ContractFromManifest(manifest plugins.Manifest) Contract {
	contract := DefaultContract()
	contract.ExpectedABIVersion = manifest.APIVersion
	contract.InvokeExportName = manifest.Entry
	if manifest.Allocate != "" {
		contract.AllocateExportName = manifest.Allocate
	}
	if manifest.Deallocate != "" {
		contract.DeallocateExportName = manifest.Deallocate
	}
	if manifest.ABIFunction != "" {
		contract.ABIExportName = manifest.ABIFunction
	}
	contract.StartFunctions = append([]string(nil), manifest.Start...)
	return contract
}

// Runtime hosts Wasm guest modules.
type Runtime struct {
	cfg Config
	rt  wazero.Runtime
}

// Compiled wraps a compiled guest module.
type Compiled struct {
	runtime *Runtime
	module  wazero.CompiledModule
}

type executionState struct {
	mu           sync.Mutex
	events       []plugins.Event
	logs         []plugins.LogEntry
	capabilities plugins.CapabilitySet
}

// DefaultConfig returns a conservative portable host config.
func DefaultConfig() Config {
	return Config{
		HostModuleName:      DefaultHostModuleName,
		LogImportName:       DefaultLogImportName,
		EmitEventImportName: DefaultEmitEventImport,
		TimeNowImportName:   DefaultTimeNowImport,
		RandomFillImport:    DefaultRandomFillImport,
		EnableWASI:          true,
		RuntimeConfig:       wazero.NewRuntimeConfigInterpreter(),
	}
}

// DefaultContract returns the default plugin ABI contract.
func DefaultContract() Contract {
	return Contract{
		ExpectedABIVersion:   DefaultABIVersion,
		RequireABIVersion:    true,
		ABIExportName:        DefaultABIExportName,
		AllocateExportName:   DefaultAllocateExport,
		DeallocateExportName: DefaultDeallocateExport,
		InvokeExportName:     DefaultInvokeExport,
		ExpectResultBuffer:   true,
	}
}

// NewRuntime creates a new Wasm host runtime.
func NewRuntime(ctx context.Context, cfg Config) (*Runtime, error) {
	cfg = normalizeConfig(cfg)
	rt := wazero.NewRuntimeWithConfig(ctx, cfg.RuntimeConfig)

	if cfg.EnableWASI {
		wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	}

	_, err := rt.NewHostModuleBuilder(cfg.HostModuleName).
		NewFunctionBuilder().WithFunc(hostLog).Export(cfg.LogImportName).
		NewFunctionBuilder().WithFunc(hostEmitEvent).Export(cfg.EmitEventImportName).
		NewFunctionBuilder().WithFunc(hostTimeNowUnixMs).Export(cfg.TimeNowImportName).
		NewFunctionBuilder().WithFunc(hostFillRandom).Export(cfg.RandomFillImport).
		Instantiate(ctx)
	if err != nil {
		_ = rt.Close(ctx)
		return nil, err
	}

	return &Runtime{cfg: cfg, rt: rt}, nil
}

// Close releases the underlying wazero runtime.
func (r *Runtime) Close(ctx context.Context) error {
	return r.rt.Close(ctx)
}

// Compile compiles a guest Wasm module.
func (r *Runtime) Compile(ctx context.Context, wasm []byte) (*Compiled, error) {
	compiled, err := r.rt.CompileModule(ctx, wasm)
	if err != nil {
		return nil, err
	}
	return &Compiled{runtime: r, module: compiled}, nil
}

// CompileFile reads and compiles a guest Wasm module from disk.
func (r *Runtime) CompileFile(ctx context.Context, path string) (*Compiled, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return r.Compile(ctx, raw)
}

// Validate ensures the compiled module satisfies the supplied contract.
func (c *Compiled) Validate(ctx context.Context, contract Contract) error {
	contract = normalizeContract(contract)

	mod, err := c.runtime.rt.InstantiateModule(ctx, c.module, wazero.NewModuleConfig())
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	if contract.RequireABIVersion {
		fn := mod.ExportedFunction(contract.ABIExportName)
		if fn == nil {
			return fmt.Errorf("missing ABI export %q", contract.ABIExportName)
		}
		def := fn.Definition()
		if len(def.ParamTypes()) != 0 || len(def.ResultTypes()) != 1 {
			return fmt.Errorf("ABI export %q has invalid signature", contract.ABIExportName)
		}
		results, err := fn.Call(ctx)
		if err != nil {
			return fmt.Errorf("call ABI export %q: %w", contract.ABIExportName, err)
		}
		if len(results) == 0 {
			return fmt.Errorf("ABI export %q returned no result", contract.ABIExportName)
		}
		if results[0] != contract.ExpectedABIVersion {
			return fmt.Errorf("ABI version mismatch: got %d want %d", results[0], contract.ExpectedABIVersion)
		}
	}

	for _, name := range []string{contract.AllocateExportName, contract.DeallocateExportName, contract.InvokeExportName} {
		if mod.ExportedFunction(name) == nil {
			return fmt.Errorf("missing required export %q", name)
		}
	}
	if err := validateFunctionShape(mod.ExportedFunction(contract.AllocateExportName), contract.AllocateExportName, 1, 1); err != nil {
		return err
	}
	if err := validateFunctionShape(mod.ExportedFunction(contract.DeallocateExportName), contract.DeallocateExportName, 2, 0); err != nil {
		return err
	}
	if err := validateFunctionShape(mod.ExportedFunction(contract.InvokeExportName), contract.InvokeExportName, 2, 1); err != nil {
		return err
	}
	return nil
}

// Invoke runs the guest contract with the provided input bytes.
func (c *Compiled) Invoke(ctx context.Context, input []byte, contract Contract) (plugins.Result, error) {
	return c.InvokeWithCapabilities(ctx, input, contract, nil)
}

// InvokeWithCapabilities runs the guest contract with an explicit capability set.
func (c *Compiled) InvokeWithCapabilities(ctx context.Context, input []byte, contract Contract, capabilities []string) (plugins.Result, error) {
	contract = normalizeContract(contract)
	state := &executionState{capabilities: plugins.NewCapabilitySet(capabilities)}
	ctx = context.WithValue(ctx, executionStateKey, state)

	modCfg := wazero.NewModuleConfig()
	if len(contract.StartFunctions) > 0 {
		modCfg = modCfg.WithStartFunctions(contract.StartFunctions...)
	}

	mod, err := c.runtime.rt.InstantiateModule(ctx, c.module, modCfg)
	if err != nil {
		return plugins.Result{}, err
	}
	defer mod.Close(ctx)

	allocator := mod.ExportedFunction(contract.AllocateExportName)
	deallocator := mod.ExportedFunction(contract.DeallocateExportName)
	invoke := mod.ExportedFunction(contract.InvokeExportName)
	mem := mod.Memory()
	if mem == nil {
		return plugins.Result{}, errors.New("guest module has no exported memory")
	}

	var inputPtr uint64
	if len(input) > 0 {
		results, err := allocator.Call(ctx, uint64(len(input)))
		if err != nil {
			return plugins.Result{}, fmt.Errorf("allocate input: %w", err)
		}
		inputPtr = results[0]
		defer deallocator.Call(ctx, inputPtr, uint64(len(input)))
		if !mem.Write(uint32(inputPtr), input) {
			return plugins.Result{}, fmt.Errorf("guest memory write failed at %d", inputPtr)
		}
	}

	results, err := invoke.Call(ctx, inputPtr, uint64(len(input)))
	if err != nil {
		return plugins.Result{}, fmt.Errorf("invoke %q: %w", contract.InvokeExportName, err)
	}

	result := plugins.Result{
		Events: state.snapshotEvents(),
		Logs:   state.snapshotLogs(),
	}
	if !contract.ExpectResultBuffer {
		return result, nil
	}
	if len(results) == 0 {
		return plugins.Result{}, fmt.Errorf("invoke %q returned no result", contract.InvokeExportName)
	}

	outputPtr, outputLen := unpackPtrSize(results[0])
	if outputLen == 0 {
		return result, nil
	}
	output, ok := mem.Read(outputPtr, outputLen)
	if !ok {
		return plugins.Result{}, fmt.Errorf("guest memory read failed at %d len %d", outputPtr, outputLen)
	}
	result.Output = append([]byte(nil), output...)
	_, _ = deallocator.Call(ctx, uint64(outputPtr), uint64(outputLen))
	return result, nil
}

func hostLog(ctx context.Context, mod api.Module, msgPtr, msgLen uint32) {
	msg, err := readGuestString(mod, msgPtr, msgLen)
	if err != nil {
		return
	}
	if state := executionStateFromContext(ctx); state != nil {
		if !state.capabilities.Has(plugins.CapabilityLog) {
			return
		}
		state.appendLog(plugins.LogEntry{Level: defaultLogLevel, Message: msg})
	}
}

func hostEmitEvent(ctx context.Context, mod api.Module, typPtr, typLen, payloadPtr, payloadLen uint32) {
	typ, err := readGuestString(mod, typPtr, typLen)
	if err != nil {
		return
	}
	payload, err := readGuestString(mod, payloadPtr, payloadLen)
	if err != nil {
		return
	}
	if state := executionStateFromContext(ctx); state != nil {
		if !state.capabilities.Has(plugins.CapabilityEmitEvent) {
			return
		}
		state.appendEvent(plugins.Event{Type: typ, Payload: payload})
	}
}

func hostTimeNowUnixMs(ctx context.Context) uint64 {
	if state := executionStateFromContext(ctx); state != nil {
		if !state.capabilities.Has(plugins.CapabilityTimeNow) {
			return 0
		}
	}
	return uint64(time.Now().UnixMilli())
}

func hostFillRandom(ctx context.Context, mod api.Module, ptr, length uint32) uint32 {
	if state := executionStateFromContext(ctx); state != nil {
		if !state.capabilities.Has(plugins.CapabilityRandom) {
			return 0
		}
	}
	mem := mod.Memory()
	if mem == nil {
		return 0
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return 0
	}
	if !mem.Write(ptr, buf) {
		return 0
	}
	return length
}

func readGuestString(mod api.Module, ptr, length uint32) (string, error) {
	mem := mod.Memory()
	if mem == nil {
		return "", errors.New("guest module has no exported memory")
	}
	buf, ok := mem.Read(ptr, length)
	if !ok {
		return "", fmt.Errorf("memory read out of range ptr=%d len=%d", ptr, length)
	}
	return string(buf), nil
}

func validateFunctionShape(fn api.Function, name string, wantParams, wantResults int) error {
	if fn == nil {
		return fmt.Errorf("missing required export %q", name)
	}
	def := fn.Definition()
	if got := len(def.ParamTypes()); got != wantParams {
		return fmt.Errorf("export %q has %d params, want %d", name, got, wantParams)
	}
	if got := len(def.ResultTypes()); got != wantResults {
		return fmt.Errorf("export %q has %d results, want %d", name, got, wantResults)
	}
	return nil
}

func executionStateFromContext(ctx context.Context) *executionState {
	state, _ := ctx.Value(executionStateKey).(*executionState)
	return state
}

func (s *executionState) appendEvent(event plugins.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *executionState) appendLog(entry plugins.LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, entry)
}

func (s *executionState) snapshotEvents() []plugins.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]plugins.Event, len(s.events))
	copy(out, s.events)
	return out
}

func (s *executionState) snapshotLogs() []plugins.LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]plugins.LogEntry, len(s.logs))
	copy(out, s.logs)
	return out
}

func normalizeConfig(cfg Config) Config {
	defaults := DefaultConfig()
	if strings.TrimSpace(cfg.HostModuleName) == "" {
		cfg.HostModuleName = defaults.HostModuleName
	}
	if strings.TrimSpace(cfg.LogImportName) == "" {
		cfg.LogImportName = defaults.LogImportName
	}
	if strings.TrimSpace(cfg.EmitEventImportName) == "" {
		cfg.EmitEventImportName = defaults.EmitEventImportName
	}
	if strings.TrimSpace(cfg.TimeNowImportName) == "" {
		cfg.TimeNowImportName = defaults.TimeNowImportName
	}
	if strings.TrimSpace(cfg.RandomFillImport) == "" {
		cfg.RandomFillImport = defaults.RandomFillImport
	}
	if cfg.RuntimeConfig == nil {
		cfg.RuntimeConfig = defaults.RuntimeConfig
	}
	return cfg
}

func normalizeContract(contract Contract) Contract {
	defaults := DefaultContract()
	defaultRequireABI := contract.ABIExportName == "" && contract.ExpectedABIVersion == 0
	defaultExpectResult := contract.InvokeExportName == "" && contract.AllocateExportName == "" && contract.DeallocateExportName == ""
	if strings.TrimSpace(contract.ABIExportName) == "" {
		contract.ABIExportName = defaults.ABIExportName
	}
	if contract.ExpectedABIVersion == 0 {
		contract.ExpectedABIVersion = defaults.ExpectedABIVersion
	}
	if strings.TrimSpace(contract.AllocateExportName) == "" {
		contract.AllocateExportName = defaults.AllocateExportName
	}
	if strings.TrimSpace(contract.DeallocateExportName) == "" {
		contract.DeallocateExportName = defaults.DeallocateExportName
	}
	if strings.TrimSpace(contract.InvokeExportName) == "" {
		contract.InvokeExportName = defaults.InvokeExportName
	}
	if defaultRequireABI {
		contract.RequireABIVersion = defaults.RequireABIVersion
	}
	if defaultExpectResult {
		contract.ExpectResultBuffer = defaults.ExpectResultBuffer
	}
	return contract
}

func unpackPtrSize(value uint64) (uint32, uint32) {
	return uint32(value >> 32), uint32(value)
}

// wazeroModuleDir is test-only plumbing used to locate example fixtures.
func wazeroModuleDir() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/tetratelabs/wazero")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go list wazero module: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	dir := strings.TrimSpace(stdout.String())
	if dir == "" {
		return "", errors.New("empty wazero module dir")
	}
	return filepath.Clean(dir), nil
}

// WazeroModuleDirForTests exposes the wazero module location for fixture-backed tests.
func WazeroModuleDirForTests() (string, error) {
	return wazeroModuleDir()
}
