# Headless SDK

Import path:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/headless
```

The Headless SDK exposes the current go-mirofish gateway stack as an embeddable Go package.

## Use cases

- mount the go-mirofish API into an existing Go service
- run the stack without shelling out to the `mirofish-gateway` binary
- keep the same routes and internal wiring as the gateway app, but under your own process control

## Minimal usage

```go
package main

import (
	"context"
	"log"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/headless"
)

func main() {
	if err := headless.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

## Embedded usage

```go
package main

import (
	"log"
	"net/http"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/headless"
)

func main() {
	cfg, err := headless.LoadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	app, err := headless.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/mirofish/", http.StripPrefix("/mirofish", app.Handler()))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Public API

- `LoadConfigFromEnv() (Config, error)`
- `New(Config) (*App, error)`
- `(*App).Handler() http.Handler`
- `(*App).NewServer() *http.Server`
- `(*App).ListenAndServe(ctx context.Context) error`
- `Run(ctx context.Context) error`
- `NewWasmRuntime(ctx, cfg)`
- `LoadTrustPolicyFile(path)`
- `LoadWasmPluginFromBytes(ctx, runtime, manifestRaw, wasmBytes)`
- `LoadWasmPluginFromFiles(ctx, runtime, manifestPath, wasmPath)`
- `LoadWasmPluginFromDir(ctx, runtime, dir)`
- `LoadWasmPluginFromFilesTrusted(ctx, runtime, manifestPath, wasmPath, policy)`
- `LoadWasmPluginFromDirTrusted(ctx, runtime, dir, policy)`
- `LoadStarlarkPluginFromBytes(runtime, manifestRaw, source)`
- `LoadStarlarkPluginFromFiles(runtime, manifestPath, sourcePath)`
- `LoadStarlarkPluginFromDir(runtime, dir)`
- `LoadStarlarkPluginFromFilesTrusted(runtime, manifestPath, sourcePath, policy)`
- `LoadStarlarkPluginFromDirTrusted(runtime, dir, policy)`
- `(*WasmPlugin).Invoke(ctx, input)`
- `(*WasmPlugin).PluginManifestJSON()`
- `(*StarlarkPlugin).Invoke(ctx, input)`
- `(*StarlarkPlugin).PluginManifestJSON()`
- `NewWasmManager(runtime)`
- `NewTrustedWasmManager(runtime, policy)`
- `NewTrustedWasmManagerFromFile(runtime, policyPath)`
- `NewPluginManager(wasmRuntime, starlarkRuntime)`
- `NewTrustedPluginManager(wasmRuntime, starlarkRuntime, policy)`
- `NewTrustedPluginManagerFromFile(wasmRuntime, starlarkRuntime, policyPath)`
- `(*WasmManager).RegisterDir(dir)`
- `(*WasmManager).RegisterDirs(dirs...)`
- `(*WasmManager).List()`
- `(*WasmManager).LoadByName(ctx, name)`
- `(*WasmManager).InvokeByName(ctx, name, input)`
- `(*PluginManager).RegisterDir(dir)`
- `(*PluginManager).RegisterDirs(dirs...)`
- `(*PluginManager).List()`
- `(*PluginManager).LoadByName(ctx, name)`
- `(*PluginManager).InvokeByName(ctx, name, input)`

## Wasm plugins

The headless SDK now includes the first Wasm-plugin loading path through the same process:

```go
import (
  "context"
  "os"

  "github.com/go-mirofish/go-mirofish/gateway/sdk/headless"
  pluginwasm "github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/wasm"
)

func loadPlugin() error {
  ctx := context.Background()
  runtime, err := headless.NewWasmRuntime(ctx, pluginwasm.DefaultConfig())
  if err != nil {
    return err
  }

  plugin, err := headless.LoadWasmPluginFromDir(
    ctx,
    runtime,
    "examples/wasm-greeter",
  )
  if err != nil {
    return err
  }

  _, err = plugin.Invoke(ctx, []byte("hello"))
  return err
}
```

You can still use `LoadWasmPluginFromBytes(...)` or `LoadWasmPluginFromFiles(...)` when you manage manifests and binaries manually, but `LoadWasmPluginFromDir(...)` is the simplest first-party path.

Repo-owned example:

- `examples/wasm-greeter/manifest.json`
- `examples/wasm-greeter/greet-rust.wasm`
- `examples/wasm-greeter/rust/greet.rs`
- `examples/wasm-event-greeter/manifest.json`
- `examples/wasm-event-greeter/event-greeter.wasm`
- `examples/wasm-event-greeter/src/lib.rs`

### Multi-plugin directory flow

```go
manager, err := headless.NewWasmManager(runtime)
if err != nil {
  return err
}

if err := manager.RegisterDirs("examples/wasm-greeter"); err != nil {
  return err
}

items := manager.List()
result, err := manager.InvokeByName(ctx, "example-greeter", []byte("SDK"))
if err != nil {
  return err
}

fmt.Println(len(items), string(result.Output))
```

### File-backed trust policy

```go
policy, err := headless.LoadTrustPolicyFile("plugins/trust.json")
if err != nil {
  return err
}

manager, err := headless.NewTrustedWasmManager(runtime, policy)
if err != nil {
  return err
}
```

The first-party JSON trust policy format accepts:

- `require_digest`
- `require_signed`
- `allow_unsigned`
- `trusted_signers` keyed by signer ID with base64 or hex Ed25519 public keys

## Second runtime path

The SDK now also has a **Starlark** runtime scaffold under:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/starlark
```

Use that path for deterministic, Python-like plugin logic when you do not need Wasm-level language portability.

## Runtime-neutral plugin manager

The headless SDK now also exposes one registry/manager surface across Wasm and Starlark:

```go
wasmRuntime, err := headless.NewWasmRuntime(ctx, pluginwasm.DefaultConfig())
if err != nil {
  return err
}

starlarkRuntime := pluginstarlark.NewRuntime()

manager, err := headless.NewPluginManager(wasmRuntime, starlarkRuntime)
if err != nil {
  return err
}

if err := manager.RegisterDirs(
  "examples/wasm-greeter",
  "examples/starlark-greeter",
); err != nil {
  return err
}

result, err := manager.InvokeByName(ctx, "starlark-greeter", []byte("SDK"))
if err != nil {
  return err
}
```

## Signing helpers

The trust package now includes first-party signing helpers as well as verification:

- `LoadPrivateKeyFile(path)`
- `ParsePrivateKey(raw)`
- `SignManifestAndModule(manifest, module, signerID, privateKey)`

This keeps digest + signature generation aligned with the same trust payload that verification uses.

## Current release note

This package is the minimal public surface needed for a real `Headless SDK v0.1.0` release. It is intentionally small and reuses the same route wiring as the gateway runtime.
