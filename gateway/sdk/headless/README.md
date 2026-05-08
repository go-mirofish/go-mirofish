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
- `LoadWasmPluginFromBytes(ctx, runtime, manifestRaw, wasmBytes)`
- `LoadWasmPluginFromFiles(ctx, runtime, manifestPath, wasmPath)`
- `LoadWasmPluginFromDir(ctx, runtime, dir)`
- `LoadWasmPluginFromFilesTrusted(ctx, runtime, manifestPath, wasmPath, policy)`
- `LoadWasmPluginFromDirTrusted(ctx, runtime, dir, policy)`
- `(*WasmPlugin).Invoke(ctx, input)`
- `(*WasmPlugin).PluginManifestJSON()`
- `NewWasmManager(runtime)`
- `NewTrustedWasmManager(runtime, policy)`
- `(*WasmManager).RegisterDir(dir)`
- `(*WasmManager).RegisterDirs(dirs...)`
- `(*WasmManager).List()`
- `(*WasmManager).LoadByName(ctx, name)`
- `(*WasmManager).InvokeByName(ctx, name, input)`

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

## Second runtime path

The SDK now also has a **Starlark** runtime scaffold under:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/starlark
```

Use that path for deterministic, Python-like plugin logic when you do not need Wasm-level language portability.

## Current release note

This package is the minimal public surface needed for a real `Headless SDK v0.1.0` release. It is intentionally small and reuses the same route wiring as the gateway runtime.
