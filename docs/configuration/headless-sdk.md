# Headless SDK

`go-mirofish` now ships two SDK surfaces:

- **Go SDK:** `github.com/go-mirofish/go-mirofish/gateway/sdk/headless`
- **JavaScript SDK:** `go-mirofish-sdk`

Use these when you want to integrate with a running `go-mirofish` gateway without forking the gateway binary or rewriting the route layer.

## JavaScript SDK

Install:

```bash
npm install go-mirofish-sdk
```

Minimal usage:

```js
import createHeadlessSDK from 'go-mirofish-sdk'

const sdk = createHeadlessSDK({
  baseURL: 'http://127.0.0.1:3000',
})

const health = await sdk.system.getHealth()
console.log(health)
```

What it provides:

- graph APIs
- simulation APIs
- report APIs
- `/health`, `/ready`, `/metrics`, and `/api/providers` helpers
- retry handling for non-4xx failures

## Go SDK

Import:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/headless
```

Minimal usage:

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

Embedded usage:

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

### Wasm plugin loading

The Go SDK now includes a first Wasm-plugin loading path:

```go
ctx := context.Background()
rt, err := headless.NewWasmRuntime(ctx, pluginwasm.DefaultConfig())
if err != nil {
	log.Fatal(err)
}

plugin, err := headless.LoadWasmPluginFromDir(
	ctx,
	rt,
	"examples/wasm-greeter",
)
if err != nil {
	log.Fatal(err)
}

result, err := plugin.Invoke(ctx, []byte("SDK"))
if err != nil {
	log.Fatal(err)
}
fmt.Println(string(result.Output))
```

Repo-owned example assets:

- `examples/wasm-greeter/manifest.json`
- `examples/wasm-greeter/greet-rust.wasm`
- `examples/wasm-greeter/rust/greet.rs`
- `examples/wasm-event-greeter/manifest.json`
- `examples/wasm-event-greeter/event-greeter.wasm`
- `examples/wasm-event-greeter/src/lib.rs`

### Multi-plugin registry

The Go SDK now also exposes a small Wasm plugin manager:

```go
manager, err := headless.NewWasmManager(rt)
if err != nil {
	log.Fatal(err)
}

if err := manager.RegisterDirs("examples/wasm-greeter"); err != nil {
	log.Fatal(err)
}

plugins := manager.List()
result, err := manager.InvokeByName(ctx, "example-greeter", []byte("SDK"))
if err != nil {
	log.Fatal(err)
}

fmt.Println(len(plugins), string(result.Output))
```

## Starlark runtime scaffold

The second plugin path is now scaffolded under:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/starlark
```

Repo-owned example assets:

- `examples/starlark-greeter/manifest.json`
- `examples/starlark-greeter/plugin.star`

Use this runtime when you want:

- Python-like syntax
- deterministic rules
- lower complexity than Wasm
- pure Go embedding

## Integration promise

The SDK goal is:

- import and plug in
- no internal code changes required by the integrator
- the same Go-native route wiring as the gateway runtime

## Notes

- The JavaScript SDK does **not** bundle the gateway binary; it is an HTTP client package.
- The Go SDK does **not** create a second runtime; it exposes the existing gateway wiring as an embeddable package.
- The repository root package `go-mirofish` is intentionally private now to prevent accidental npm publishes of the full app instead of the SDK.
