# Headless SDK Release Candidate

This file is the exact release-prep bundle for the new headless SDK surface.

It is intentionally isolated from the repo-wide dirty worktree so you can stage only the SDK-facing changes.

## SDK Identity

- SDK surface name: `Headless SDK v0.1.0`
- Go import path: `github.com/go-mirofish/go-mirofish/gateway/sdk/headless`
- npm package name: `go-mirofish-sdk`
- Repo/module tag: use the **next valid repository release version**, not `v0.1.0` again

Recommended public wording:

> Headless SDK v0.1.0 is now available as part of go-mirofish `vX.Y.Z`.

## SDK-only staging set

Stage only these files for the SDK release candidate:

```bash
git -C go-mirofish add -- \
  README.md \
  package.json \
  pnpm-workspace.yaml \
  gateway/sdk/headless/doc.go \
  gateway/sdk/headless/headless.go \
  gateway/sdk/headless/headless_test.go \
  gateway/sdk/headless/example_test.go \
  gateway/sdk/headless/README.md \
  packages/headless-sdk/package.json \
  packages/headless-sdk/index.js \
  packages/headless-sdk/README.md \
  docs/report/headless-sdk-v0.1.0.md \
  docs/report/headless-sdk-release-checklist.md \
  docs/report/headless-sdk-release-candidate.md \
  docs/report/headless-sdk-v0.1.6-release-notes.md \
  scripts/release/README.md
```

## SDK verification

Minimum SDK verification:

```bash
cd go-mirofish/gateway
GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache go test ./sdk/headless
```

```bash
cd go-mirofish/packages/headless-sdk
npm pack --dry-run
```

Recommended supporting verification:

```bash
cd go-mirofish/gateway
GOCACHE=/tmp/go-build-cache GOMODCACHE=/tmp/go-mod-cache go test ./internal/http/app ./internal/http/report ./internal/http/prepare ./internal/http/simulation ./internal/provider ./internal/report ./internal/graph
```

## Changelog block to paste into the next repo release

Use this inside the next repo release section in `CHANGELOG.md`:

```md
### Added

- add Headless SDK v0.1.0 at `github.com/go-mirofish/go-mirofish/gateway/sdk/headless`
- add npm package `go-mirofish-sdk`

### Documentation

- add Headless SDK README with import-and-run and embedded usage examples
- add Headless SDK release note and release checklist

### Notes

- Headless SDK v0.1.0 is introduced as part of this repo release; the SDK version label does not imply the repository tag was reset to `v0.1.0`
```

## Release note body

Use this short release-note summary if needed:

> This release introduces the first public Headless SDK surface for go-mirofish. Integrators can now import `github.com/go-mirofish/go-mirofish/gateway/sdk/headless` and run or mount the current Go-native gateway stack directly from their own Go applications without forking the gateway binary.

## Commit suggestion

If you want one isolated SDK commit:

```bash
git -C go-mirofish commit -m "feat(sdk): add headless sdk surface"
```

## Important caveat

Do not tag the repository as `v0.1.0`.

The repository already has existing tags:

- `v0.1.0`
- `v0.1.1`
- `v0.1.2`

So the SDK should be announced as a feature introduced in the **next repo release**, not as a repo-wide version reset.

Also:

- do not publish the root `go-mirofish` package again
- publish only `go-mirofish-sdk`
