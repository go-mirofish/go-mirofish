# Headless SDK v0.1.0 introduced in go-mirofish v0.1.6

Use this as the final release-note text for the next repository release.

## Title

Headless SDK v0.1.0 introduced in go-mirofish v0.1.6

## Release note

This release introduces the first public **Headless SDK v0.1.0** surface for `go-mirofish`.

Integrators can now import:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/headless
```

and install:

```bash
npm install go-mirofish-sdk
```

and run or mount the current Go-native gateway stack directly from their own Go applications without forking the gateway binary.

### What is included

- an embeddable Go package for the current gateway HTTP stack
- a dedicated npm package: `go-mirofish-sdk`
- `Run(context.Context)` for direct startup
- `LoadConfigFromEnv()` and `New(Config)` for explicit construction
- `Handler()` for mounting under an existing `net/http` mux
- package documentation and import examples

### Why it matters

- no custom process wrapper is required
- no internal code changes are required by the integrator
- the same route wiring used by the gateway binary is now available as an importable Go package
- JavaScript consumers now have a standalone package name instead of publishing the root app tarball

### Repository versioning note

The SDK surface itself is versioned as **Headless SDK v0.1.0**.

The repository release remains **go-mirofish v0.1.6**.

This is intentional: the SDK is a new public package surface introduced inside the next forward repository release, not a repository-wide version reset.

## Changelog block

```md
## v0.1.6 - 2026-05-09

### Added

- add Headless SDK v0.1.0 at `github.com/go-mirofish/go-mirofish/gateway/sdk/headless`
- add npm package `go-mirofish-sdk`

### Documentation

- add Headless SDK README with import-and-run and embedded usage examples
- add Headless SDK release note, checklist, and release-candidate staging guide

### Notes

- Headless SDK v0.1.0 is introduced as part of go-mirofish `v0.1.6`
- the SDK version label does not imply the repository tag was reset to `v0.1.0`
```
