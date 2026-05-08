# Headless SDK Release Checklist

This checklist is for releasing the first public headless SDK surface:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/headless
```

```text
go-mirofish-sdk
```

## Positioning

The SDK claim is:

- integrates seamlessly
- no internal code changes required by the integrator
- import and plug-and-play inside an existing Go service

That claim is only safe when the package docs, example, and tests all agree.

## Required files

- `gateway/sdk/headless/doc.go`
- `gateway/sdk/headless/headless.go`
- `gateway/sdk/headless/headless_test.go`
- `gateway/sdk/headless/example_test.go`
- `gateway/sdk/headless/README.md`
- `packages/headless-sdk/package.json`
- `packages/headless-sdk/index.js`
- `packages/headless-sdk/README.md`
- `docs/report/headless-sdk-v0.1.0.md`

## Verification

Run:

```bash
cd gateway
GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache go test ./sdk/headless
```

Recommended supporting verification:

```bash
cd gateway
GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache go test ./internal/http/app ./internal/http/report ./internal/http/prepare ./internal/http/simulation ./internal/provider ./internal/report ./internal/graph
```

```bash
cd packages/headless-sdk
npm pack --dry-run
```

## Messaging rule

Keep this distinction explicit:

- **SDK surface version:** `Headless SDK v0.1.0`
- **repository/module release tag:** follow the actual repo release line

Recommended wording:

> Headless SDK v0.1.0 is now available as part of go-mirofish `vX.Y.Z`.

## Do not do this

- do not create a Git tag lower than the highest existing repo tag
- do not announce “plug and play” without the import example and SDK test passing
- do not mix the SDK release note with unrelated dirty worktree changes
- do not publish the root `go-mirofish` package again; only publish `go-mirofish-sdk`

## Preferred release sequence

1. isolate SDK-facing files
2. run SDK verification
3. update release note and announcement copy
4. run the repo release gate
5. tag the repo using the next valid repo version
6. mention the SDK surface inside that repo release
