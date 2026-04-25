# Bundled benchmark JSON (docs UI + fixtures)

The Vue **Benchmarks** page reads committed JSON from this directory so the docs area can show timelines and metrics without a live server.

- **`live-stack__hybrid__latest.json`:** aggregate “stack proof” (gateway health, bounded stress, summary IDs) used as the default combobox entry when present.
- **Per-example `*__small__latest.json` files:** outputs from the Go example runner at **`small`** profile (deterministic local runs; CI-friendly). Regenerate with `go run ./gateway/cmd/go-mirofish-examples --all --bench-only --profile small` and copy artifacts here, or use your own capture workflow.

**Refresh stack fields in all `*__*__latest.json`:** after you capture a live benchmark JSON (e.g. `benchmark/results/benchmarks/live-benchmark.json` from `make benchmark-live`), run:

```bash
cd gateway && go run ./cmd/mirofish-hybrid merge-bundled
```

or from the repo root: `bash scripts/dev/benchmark.sh merge-bundled`

That command merges the standard stack keys from the live file into each matching bundled JSON (see `stackKeys` in `gateway/cmd/mirofish-hybrid/merge_bundled.go`).
