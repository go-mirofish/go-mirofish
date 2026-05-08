package headless

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
	pluginstarlark "github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/starlark"
	plugintrust "github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/trust"
	pluginwasm "github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/wasm"
)

func newTestConfig(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	dist := filepath.Join(root, "frontend", "dist")
	if err := tSetFile(dist, "index.html", "<html>sdk</html>"); err != nil {
		t.Fatalf("write index: %v", err)
	}
	return Config{
		BindHost:        "127.0.0.1",
		Port:            "3000",
		FrontendDistDir: dist,
		ProjectsDir:     filepath.Join(root, "data", "projects"),
		ReportsDir:      filepath.Join(root, "data", "reports"),
		TasksDir:        filepath.Join(root, "data", "tasks"),
		SimulationsDir:  filepath.Join(root, "data", "simulations"),
		ScriptsDir:      filepath.Join(root, "scripts"),
	}
}

func TestNewAndHealthRoutes(t *testing.T) {
	cfg := newTestConfig(t)
	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t.Run("health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("ready", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("providers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["success"] != true {
			t.Fatalf("expected success=true, got %#v", payload["success"])
		}
	})
}

func TestRunStartsAndStopsFromContext(t *testing.T) {
	cfg := newTestConfig(t)
	app, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- app.ListenAndServe(ctx)
	}()
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("ListenAndServe: %v", err)
	}
}

func TestLoadWasmPluginFromBytes(t *testing.T) {
	ctx := context.Background()
	rt, err := NewWasmRuntime(ctx, pluginwasm.Config{
		HostModuleName: pluginwasm.DefaultHostModuleName,
		LogImportName:  pluginwasm.DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewWasmRuntime: %v", err)
	}
	defer rt.Close(ctx)

	manifestRaw := []byte(`{
		"name":"example-greeter",
		"version":"0.1.0",
		"runtime":"wasm",
		"api_version":1,
		"entry":"greeting",
		"allocate":"allocate",
		"deallocate":"deallocate",
		"abi_function":"mirofish_abi_version",
		"capabilities":["log"]
	}`)
	wasmRaw := mustReadRustGreetFixture(t)

	plugin, err := LoadWasmPluginFromBytes(ctx, rt, manifestRaw, wasmRaw)
	if err != nil {
		t.Fatalf("LoadWasmPluginFromBytes: %v", err)
	}
	result, err := plugin.Invoke(ctx, []byte("SDK"))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK!"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestLoadWasmPluginFromFiles(t *testing.T) {
	ctx := context.Background()
	rt, err := NewWasmRuntime(ctx, pluginwasm.Config{
		HostModuleName: pluginwasm.DefaultHostModuleName,
		LogImportName:  pluginwasm.DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewWasmRuntime: %v", err)
	}
	defer rt.Close(ctx)

	root := repoExamplesRoot(t)
	workdir := copyExampleDir(t, root)
	manifestPath := filepath.Join(workdir, "manifest.json")
	wasmPath := filepath.Join(workdir, "greet-rust.wasm")

	plugin, err := LoadWasmPluginFromFiles(ctx, rt, manifestPath, wasmPath)
	if err != nil {
		t.Fatalf("LoadWasmPluginFromFiles: %v", err)
	}
	if plugin.Manifest.Name != "example-greeter" {
		t.Fatalf("unexpected plugin name: %q", plugin.Manifest.Name)
	}
	result, err := plugin.Invoke(ctx, []byte("SDK"))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK!"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestLoadWasmPluginFromDir(t *testing.T) {
	ctx := context.Background()
	rt, err := NewWasmRuntime(ctx, pluginwasm.Config{
		HostModuleName: pluginwasm.DefaultHostModuleName,
		LogImportName:  pluginwasm.DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewWasmRuntime: %v", err)
	}
	defer rt.Close(ctx)

	root := repoExamplesRoot(t)
	plugin, err := LoadWasmPluginFromDir(ctx, rt, root)
	if err != nil {
		t.Fatalf("LoadWasmPluginFromDir: %v", err)
	}
	if plugin.Manifest.Name != "example-greeter" {
		t.Fatalf("unexpected plugin name: %q", plugin.Manifest.Name)
	}
}

func TestLoadStarlarkPluginFromDir(t *testing.T) {
	rt := pluginstarlark.NewRuntime()
	root := repoExampleDir(t, "starlark-greeter")
	plugin, err := LoadStarlarkPluginFromDir(rt, root)
	if err != nil {
		t.Fatalf("LoadStarlarkPluginFromDir: %v", err)
	}
	if plugin.Manifest.Name != "starlark-greeter" {
		t.Fatalf("unexpected plugin name: %q", plugin.Manifest.Name)
	}
	result, err := plugin.Invoke(context.Background(), []byte("SDK"))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestLoadWasmPluginFromFilesTrusted(t *testing.T) {
	ctx := context.Background()
	rt, err := NewWasmRuntime(ctx, pluginwasm.Config{
		HostModuleName: pluginwasm.DefaultHostModuleName,
		LogImportName:  pluginwasm.DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewWasmRuntime: %v", err)
	}
	defer rt.Close(ctx)

	root := repoExamplesRoot(t)
	workdir := copyExampleDir(t, root)
	manifestPath := filepath.Join(workdir, "manifest.json")
	wasmPath := filepath.Join(workdir, "greet-rust.wasm")
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	wasmRaw, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatalf("read wasm: %v", err)
	}

	manifest := map[string]any{}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	digest := sha256.Sum256(wasmRaw)
	manifest["digest_sha256"] = hex.EncodeToString(digest[:])
	manifest["signer_id"] = "core"

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	unsignedRaw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	unsignedManifest, err := plugins.ParseManifest(unsignedRaw)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	payload, err := plugintrust.SigningPayload(unsignedManifest)
	if err != nil {
		t.Fatalf("SigningPayload: %v", err)
	}
	manifest["signature_ed25519"] = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))
	finalRaw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal signed manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, finalRaw, 0o644); err != nil {
		t.Fatalf("write signed manifest: %v", err)
	}

	plugin, err := LoadWasmPluginFromFilesTrusted(ctx, rt, manifestPath, wasmPath, plugintrust.Policy{
		RequireDigest: true,
		RequireSigned: true,
		TrustedSigners: map[string]ed25519.PublicKey{
			"core": pub,
		},
	})
	if err != nil {
		t.Fatalf("LoadWasmPluginFromFilesTrusted: %v", err)
	}
	result, err := plugin.Invoke(ctx, []byte("SDK"))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK!"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNewTrustedWasmManager(t *testing.T) {
	ctx := context.Background()
	rt, err := NewWasmRuntime(ctx, pluginwasm.Config{
		HostModuleName: pluginwasm.DefaultHostModuleName,
		LogImportName:  pluginwasm.DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewWasmRuntime: %v", err)
	}
	defer rt.Close(ctx)

	root := repoExamplesRoot(t)
	workdir := copyExampleDir(t, root)
	manifestPath := filepath.Join(workdir, "manifest.json")
	wasmPath := filepath.Join(workdir, "greet-rust.wasm")
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	wasmRaw, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatalf("read wasm: %v", err)
	}

	manifest := map[string]any{}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	digest := sha256.Sum256(wasmRaw)
	manifest["digest_sha256"] = hex.EncodeToString(digest[:])
	manifest["signer_id"] = "core"

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	unsignedRaw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	unsignedManifest, err := plugins.ParseManifest(unsignedRaw)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	payload, err := plugintrust.SigningPayload(unsignedManifest)
	if err != nil {
		t.Fatalf("SigningPayload: %v", err)
	}
	manifest["signature_ed25519"] = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))
	finalRaw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal signed manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, finalRaw, 0o644); err != nil {
		t.Fatalf("write signed manifest: %v", err)
	}

	manager, err := NewTrustedWasmManager(rt, plugintrust.Policy{
		RequireDigest: true,
		RequireSigned: true,
		TrustedSigners: map[string]ed25519.PublicKey{
			"core": pub,
		},
	})
	if err != nil {
		t.Fatalf("NewTrustedWasmManager: %v", err)
	}
	if err := manager.RegisterDirs(workdir); err != nil {
		t.Fatalf("RegisterDirs: %v", err)
	}
	result, err := manager.InvokeByName(ctx, "example-greeter", []byte("SDK"))
	if err != nil {
		t.Fatalf("InvokeByName: %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK!"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestLoadTrustPolicyFileAndNewTrustedWasmManagerFromFile(t *testing.T) {
	ctx := context.Background()
	rt, err := NewWasmRuntime(ctx, pluginwasm.Config{
		HostModuleName: pluginwasm.DefaultHostModuleName,
		LogImportName:  pluginwasm.DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewWasmRuntime: %v", err)
	}
	defer rt.Close(ctx)

	root := repoExamplesRoot(t)
	workdir := copyExampleDir(t, root)
	manifestPath := filepath.Join(workdir, "manifest.json")
	wasmPath := filepath.Join(workdir, "greet-rust.wasm")
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	wasmRaw, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatalf("read wasm: %v", err)
	}
	manifest := map[string]any{}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	digest := sha256.Sum256(wasmRaw)
	manifest["digest_sha256"] = hex.EncodeToString(digest[:])
	manifest["signer_id"] = "core"

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	unsignedRaw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	unsignedManifest, err := plugins.ParseManifest(unsignedRaw)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	payload, err := plugintrust.SigningPayload(unsignedManifest)
	if err != nil {
		t.Fatalf("SigningPayload: %v", err)
	}
	manifest["signature_ed25519"] = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))
	finalRaw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal signed manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, finalRaw, 0o644); err != nil {
		t.Fatalf("write signed manifest: %v", err)
	}

	policyPath := filepath.Join(t.TempDir(), "trust.json")
	policyRaw, err := json.Marshal(plugintrust.Config{
		RequireDigest: true,
		RequireSigned: true,
		TrustedSigners: map[string]string{
			"core": base64.StdEncoding.EncodeToString(pub),
		},
	})
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}
	if err := os.WriteFile(policyPath, policyRaw, 0o644); err != nil {
		t.Fatalf("write trust policy: %v", err)
	}

	policy, err := LoadTrustPolicyFile(policyPath)
	if err != nil {
		t.Fatalf("LoadTrustPolicyFile: %v", err)
	}
	if !policy.RequireSigned {
		t.Fatal("expected require_signed=true")
	}

	manager, err := NewTrustedWasmManagerFromFile(rt, policyPath)
	if err != nil {
		t.Fatalf("NewTrustedWasmManagerFromFile: %v", err)
	}
	if err := manager.RegisterDirs(workdir); err != nil {
		t.Fatalf("RegisterDirs: %v", err)
	}
	result, err := manager.InvokeByName(ctx, "example-greeter", []byte("SDK"))
	if err != nil {
		t.Fatalf("InvokeByName: %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK!"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestWasmManagerRegisterListAndInvoke(t *testing.T) {
	ctx := context.Background()
	rt, err := NewWasmRuntime(ctx, pluginwasm.Config{
		HostModuleName: pluginwasm.DefaultHostModuleName,
		LogImportName:  pluginwasm.DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewWasmRuntime: %v", err)
	}
	defer rt.Close(ctx)

	manager, err := NewWasmManager(rt)
	if err != nil {
		t.Fatalf("NewWasmManager: %v", err)
	}
	root := repoExamplesRoot(t)
	if err := manager.RegisterDirs(root); err != nil {
		t.Fatalf("RegisterDirs: %v", err)
	}
	items := manager.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(items))
	}
	if items[0].Manifest.Name != "example-greeter" {
		t.Fatalf("unexpected registration name: %q", items[0].Manifest.Name)
	}
	result, err := manager.InvokeByName(ctx, "example-greeter", []byte("SDK"))
	if err != nil {
		t.Fatalf("InvokeByName: %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK!"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPluginManagerRegisterListAndInvokeMixedRuntimes(t *testing.T) {
	ctx := context.Background()
	wasmRT, err := NewWasmRuntime(ctx, pluginwasm.Config{
		HostModuleName: pluginwasm.DefaultHostModuleName,
		LogImportName:  pluginwasm.DefaultLogImportName,
	})
	if err != nil {
		t.Fatalf("NewWasmRuntime: %v", err)
	}
	defer wasmRT.Close(ctx)
	starlarkRT := pluginstarlark.NewRuntime()

	manager, err := NewPluginManager(wasmRT, starlarkRT)
	if err != nil {
		t.Fatalf("NewPluginManager: %v", err)
	}
	if err := manager.RegisterDirs(
		repoExampleDir(t, "wasm-greeter"),
		repoExampleDir(t, "starlark-greeter"),
	); err != nil {
		t.Fatalf("RegisterDirs: %v", err)
	}
	items := manager.List()
	if len(items) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(items))
	}
	wasmResult, err := manager.InvokeByName(ctx, "example-greeter", []byte("SDK"))
	if err != nil {
		t.Fatalf("InvokeByName(wasm): %v", err)
	}
	if got, want := string(wasmResult.Output), "Hello, SDK!"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	starlarkResult, err := manager.InvokeByName(ctx, "starlark-greeter", []byte("SDK"))
	if err != nil {
		t.Fatalf("InvokeByName(starlark): %v", err)
	}
	if got, want := string(starlarkResult.Output), "Hello, SDK"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func tSetFile(dir, name, content string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}

func mustReadRustGreetFixture(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join(repoExampleDir(t, "wasm-greeter"), "greet-rust.wasm")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rust greet fixture: %v", err)
	}
	return raw
}

func repoExamplesRoot(t *testing.T) string {
	return repoExampleDir(t, "wasm-greeter")
}

func repoExampleDir(t *testing.T, name string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "examples", name)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate examples/%s from %s", name, wd)
	return ""
}

func copyExampleDir(t *testing.T, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if info.IsDir() && info.Name() == "target" {
			return filepath.SkipDir
		}
		dstPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, raw, 0o644)
	}); err != nil {
		t.Fatalf("copy example dir: %v", err)
	}
	return dst
}
