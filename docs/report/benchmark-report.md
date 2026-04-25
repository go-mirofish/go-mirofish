# go-mirofish benchmark (Docker stack)

Generated: 2026-04-25T08:58:28Z

**Captured:** 2026-04-25T08:58:12Z

**Release:** local

**Base URL:** http://127.0.0.1:3000

## Release Criteria

**Pass:** true

- Load p95 < 500ms
- Load error rate < 0.00%
- Stress p95 < 2000ms

## Benchmark Runs

| Profile | Concurrency | Requests | Throughput (rps) | Error Rate | p50 (ms) | p95 (ms) | p99 (ms) | Alloc MB |
|---------|-------------|----------|-----------------|------------|----------|----------|----------|----------|
| load | 8 | 496 | 49.21 | 0.0000 | 1.50 | 13.89 | 21.96 | 1.37 |
| stress | 16 | 1984 | 198.39 | 0.0000 | 1.91 | 2.74 | 4.41 | 0.00 |
| soak | 4 | 596 | 19.87 | 0.0000 | 1.21 | 1.60 | 2.80 | 0.81 |

## Baseline

baseline: single-process Go gateway; no Python backend dependency; stack: go+vue+docker

