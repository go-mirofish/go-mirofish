# Headless SDK v0.1.0

This document is the release-note draft for the first public headless SDK surface in `go-mirofish`.

## Summary

The repository now exposes two SDK surfaces for mounting or serving the `go-mirofish` gateway stack:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/headless
```

```text
go-mirofish-sdk
```

The SDK is intended for:

- import-and-run integrations
- embedding `go-mirofish` under an existing `net/http` mux
- reusing the same route wiring as the gateway binary without shelling out to the CLI

## Public API

- `LoadConfigFromEnv() (Config, error)`
- `New(Config) (*App, error)`
- `(*App).Handler() http.Handler`
- `(*App).NewServer() *http.Server`
- `(*App).ListenAndServe(ctx context.Context) error`
- `Run(ctx context.Context) error`

## What it does

- reuses the current Go-native gateway route wiring
- exposes `/health`, `/ready`, `/metrics`, `/api/providers`, and the product API routes through an embeddable handler
- keeps startup validation and readiness behavior aligned with the gateway app

## What it does not do

- it does not introduce a second product runtime
- it does not bypass the current gateway/service logic
- it does not create a separate package manager artifact outside the Go module

## Integration claim

The intended integration story is:

- no fork of the gateway binary
- no custom process wrapper required
- no code changes to `go-mirofish` internals required by the integrator
- just import the package and serve or mount the handler

## Release caveat

`go-mirofish` already has repository tags beyond `v0.1.0`, so any actual Git tag / module publication strategy must be reconciled with the repository’s existing version line. This file describes the SDK surface itself, not the final Git tagging decision.
