package wasm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
)

func TestDefaultConfigAndContract(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.HostModuleName != DefaultHostModuleName {
		t.Fatalf("expected host module %q, got %q", DefaultHostModuleName, cfg.HostModuleName)
	}
	contract := DefaultContract()
	if contract.InvokeExportName != DefaultInvokeExport {
		t.Fatalf("expected invoke export %q, got %q", DefaultInvokeExport, contract.InvokeExportName)
	}
}

func TestValidateFailsWhenABIExportMissing(t *testing.T) {
	ctx := context.Background()
	rt, err := NewRuntime(ctx, Config{
		HostModuleName: DefaultHostModuleName,
		LogImportName:  DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Close(ctx)

	wasmBytes := mustReadRepoGreeterWasm(t)
	compiled, err := rt.Compile(ctx, wasmBytes)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	err = compiled.Validate(ctx, Contract{
		RequireABIVersion:    true,
		ExpectedABIVersion:   1,
		ABIExportName:        "missing_abi_export",
		AllocateExportName:   "allocate",
		DeallocateExportName: "deallocate",
		InvokeExportName:     "greeting",
		ExpectResultBuffer:   true,
	})
	if err == nil {
		t.Fatalf("expected ABI validation error")
	}
	if !strings.Contains(err.Error(), "missing_abi_export") {
		t.Fatalf("expected missing ABI export error, got %v", err)
	}
}

func TestInvokeAgainstRustGreetingFixture(t *testing.T) {
	ctx := context.Background()
	rt, err := NewRuntime(ctx, Config{
		HostModuleName: DefaultHostModuleName,
		LogImportName:  DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Close(ctx)

	wasmBytes := mustReadRepoGreeterWasm(t)
	compiled, err := rt.Compile(ctx, wasmBytes)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	result, err := compiled.InvokeWithCapabilities(ctx, []byte("SDK"), Contract{
		RequireABIVersion:    false,
		AllocateExportName:   "allocate",
		DeallocateExportName: "deallocate",
		InvokeExportName:     "greeting",
		ExpectResultBuffer:   true,
	}, []string{plugins.CapabilityLog})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK!"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestContractFromManifest(t *testing.T) {
	manifest := plugins.Manifest{
		Name:        "example-greeter",
		Version:     "0.1.0",
		Runtime:     "wasm",
		APIVersion:  7,
		Module:      "guest.wasm",
		Entry:       "invoke",
		Allocate:    "malloc",
		Deallocate:  "free",
		ABIFunction: "abi_version",
		Start:       []string{"_initialize"},
	}
	contract := ContractFromManifest(manifest)
	if contract.ExpectedABIVersion != 7 {
		t.Fatalf("expected api version 7, got %d", contract.ExpectedABIVersion)
	}
	if contract.InvokeExportName != "invoke" {
		t.Fatalf("expected invoke export, got %q", contract.InvokeExportName)
	}
	if contract.ABIExportName != "abi_version" {
		t.Fatalf("expected abi export, got %q", contract.ABIExportName)
	}
}

func TestInvokeWithoutCapabilitiesDoesNotError(t *testing.T) {
	ctx := context.Background()
	rt, err := NewRuntime(ctx, Config{
		HostModuleName: DefaultHostModuleName,
		LogImportName:  DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Close(ctx)

	wasmBytes := mustReadRepoGreeterWasm(t)
	compiled, err := rt.Compile(ctx, wasmBytes)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	result, err := compiled.InvokeWithCapabilities(ctx, []byte("SDK"), Contract{
		RequireABIVersion:    false,
		AllocateExportName:   "allocate",
		DeallocateExportName: "deallocate",
		InvokeExportName:     "greeting",
		ExpectResultBuffer:   true,
	}, nil)
	if err != nil {
		t.Fatalf("InvokeWithCapabilities: %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK!"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if len(result.Events) != 0 {
		t.Fatalf("expected no events, got %#v", result.Events)
	}
}

func TestCompileFile(t *testing.T) {
	ctx := context.Background()
	rt, err := NewRuntime(ctx, Config{
		HostModuleName: DefaultHostModuleName,
		LogImportName:  DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Close(ctx)

	path := repoGreeterWasmPath(t)
	if _, err := rt.CompileFile(ctx, path); err != nil {
		t.Fatalf("CompileFile: %v", err)
	}
}

func TestLoadFromDir(t *testing.T) {
	ctx := context.Background()
	rt, err := NewRuntime(ctx, Config{
		HostModuleName: DefaultHostModuleName,
		LogImportName:  DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Close(ctx)

	dir := repoExamplesRoot(t)
	compiled, manifest, err := rt.LoadFromDir(ctx, dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	if compiled == nil {
		t.Fatalf("expected compiled module")
	}
	if manifest.Name != "example-greeter" {
		t.Fatalf("unexpected manifest name: %q", manifest.Name)
	}
}

func TestInvokeEventGreeterFixture(t *testing.T) {
	ctx := context.Background()
	rt, err := NewRuntime(ctx, Config{
		HostModuleName: DefaultHostModuleName,
		LogImportName:  DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Close(ctx)

	dir := repoEventExamplesRoot(t)
	compiled, manifest, err := rt.LoadFromDir(ctx, dir)
	if err != nil {
		t.Fatalf("LoadFromDir(event): %v", err)
	}
	result, err := compiled.InvokeWithCapabilities(ctx, []byte("SDK"), ContractFromManifest(manifest), manifest.Capabilities)
	if err != nil {
		t.Fatalf("InvokeWithCapabilities(event): %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK!"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(result.Events))
	}
	if result.Events[0].Type != "greeting.created" {
		t.Fatalf("unexpected event type: %q", result.Events[0].Type)
	}
	if len(result.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(result.Logs))
	}
}

func mustReadRepoGreeterWasm(t *testing.T) []byte {
	t.Helper()
	path := repoGreeterWasmPath(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read repo greeter fixture: %v", err)
	}
	return raw
}

func repoGreeterWasmPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "examples", "wasm-greeter", "greet-rust.wasm")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate examples/wasm-greeter/greet-rust.wasm from %s", wd)
	return ""
}

func repoExamplesRoot(t *testing.T) string {
	t.Helper()
	return filepath.Dir(repoGreeterWasmPath(t))
}

func repoEventExamplesRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "examples", "wasm-event-greeter")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate examples/wasm-event-greeter from %s", wd)
	return ""
}
