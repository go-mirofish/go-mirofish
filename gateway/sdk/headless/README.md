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

## Current release note

This package is the minimal public surface needed for a real `Headless SDK v0.1.0` release. It is intentionally small and reuses the same route wiring as the gateway runtime.
