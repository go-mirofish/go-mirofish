# Bundled benchmark JSON (docs UI)

These files power the **Benchmarks** page in the app. They use **short slugs** so runs are easy to spot in the searchable run picker.

## Filename pattern

```text
<scenario>__<profile>__<variant>.json
```

Segments are separated by **double underscores (`__`)** so scenario slugs can keep hyphens (e.g. `defi-stress`).

| Segment   | Meaning | Examples |
| -------- | ------- | -------- |
| scenario | Short id (maps to a display title in `frontend/src/components/docs/lib/benchmarkRunLabel.js`) | `defi-stress`, `urban-planning`, `literary-sim`, `product-launch`, `incident-drill` |
| profile  | Example profile | `small`, `medium` |
| variant  | `latest` or a timestamp id from the capture | `latest`, `20260424T170506Z` |

## Refreshing from local runs

Example and benchmark flows still write under `benchmark/results/<long-example-name>/<profile>/` (gitignored). To update what the **docs build** shows, copy a result into this folder and rename it to match the pattern above, for example:

```bash
cp benchmark/results/defi-sentiment-stress-test/small/latest.json \
  docs/bundled-benchmarks/defi-stress__small__latest.json
```

`scripts/hybrid/merge_hybrid_stack_into_benchmarks.py` merges stack fields from `benchmark/results/v0.1.0-live-benchmark.json` into each `*__*__latest.json` here (when that live file exists).

## Compare (CLI)

```bash
go run ./gateway/cmd/go-mirofish-examples --compare \
  docs/bundled-benchmarks/product-launch__small__latest.json,\
docs/bundled-benchmarks/literary-sim__small__latest.json
```

Use any two JSON paths—local `benchmark/results/...` paths still work for ad-hoc compares.
