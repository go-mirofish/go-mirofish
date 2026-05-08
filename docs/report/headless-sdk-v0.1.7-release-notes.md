# go-mirofish v0.1.7

## Summary

`v0.1.7` extends the Go-native Headless SDK plugin surface into a more complete product shape.

This release adds:

- a runtime-neutral plugin manager across Wasm and Starlark
- file-backed trust policy loading for trusted plugin bootstrapping
- first-party signing helpers for manifest + module release workflows
- runtime-aware plugin module defaults
- concurrency-safe registry and manager access

## Highlights

### Runtime-neutral plugin loading

The Headless SDK now exposes one generic plugin manager that can register, load, and invoke both:

- Wasm plugins
- Starlark plugins

This removes the need for embedding applications to maintain separate manager flows for each runtime.

### Trust policy from JSON

Trusted plugin loading no longer requires all signer configuration to be hardcoded in Go.

You can now load policy from JSON and use it with:

- `LoadTrustPolicyFile(...)`
- `NewTrustedWasmManagerFromFile(...)`
- `NewTrustedPluginManagerFromFile(...)`

### Signing helpers

The trust package now includes first-party signing utilities:

- `LoadPrivateKeyFile(...)`
- `ParsePrivateKey(...)`
- `SignManifestAndModule(...)`

This keeps plugin signing aligned with the same payload shape enforced by verification.

## Notes

- The JavaScript package `go-mirofish-sdk` is unchanged in this release.
- The new functionality ships in the Go/module SDK surface under `gateway/sdk/headless` and `gateway/sdk/plugins/*`.

## Verification

- `cd gateway && GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache go test ./sdk/...`
