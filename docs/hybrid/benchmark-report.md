# go-mirofish benchmark report

## Executive summary

- Live stack captured at: `2026-04-23T21:16:51Z`
- Example suite captured at: `2026-04-24T16:05:47Z`
- Stack boot verdict: `PASS`
- Full benchmark verdict: `PASS`
- Stress verdict: `PASS`
- Example suite verdict: `PASS`
- Current migration evidence: the Go gateway and Go-owned public control plane are verified through both the live stack benchmark and the local example suite benchmark.

## What this proves

- The benchmark was executed against the real go-mirofish hybrid stack, not inherited MiroFish media or mocked API output.
- The Go gateway stayed in the request path for the measured run, while the Python backend still owned the core workflow execution.
- The report below is intended to answer two questions directly:
  1. Which benchmark phases are actually proven by this captured run?
  2. How much of the runtime is already validated as go-mirofish rather than upstream MiroFish lineage?

## Stack evidence

| Surface | Verdict | Evidence |
| --- | --- | --- |
| Backend health | `PASS` | service=`go-mirofish-backend`, status=`ok` |
| Gateway health | `PASS` | service=`go-mirofish-gateway`, status=`ok` |
| Full benchmark | `PASS` | duration=`653.24s`, returncode=`0` |
| Bounded stress pass | `PASS` | requests=`80`, failures=`0` |

## Example suite evidence

| Example | Profile | Verdict | Startup | Runtime | Artifact |
| --- | --- | --- | ---: | ---: | --- |
| Product Launch PR War Room | `medium` | `PASS` | `10.93ms` | `19.81ms` | `risk_report.json` |
| Hyper-Local Urban Planning Rehearsal | `medium` | `PASS` | `17.58ms` | `34.34ms` | `coalition_highway.json`, `coalition_park.json` |
| Zero-Day Cyber Incident Drill | `medium` | `PASS` | `10.70ms` | `20.50ms` | `incident_report.json` |
| De-Fi Market Sentiment Stress-Test | `medium` | `PASS` | `6.92ms` | `15.97ms` | `liquidation_cascade_forecast.json` |
| Lost Ending Literary Simulator | `medium` | `PASS` | `15.04ms` | `26.59ms` | `draft_ending.json`, `draft_ending.txt` |

Result files:

- `benchmark/results/examples-benchmark-suite.json` (local capture)
- `benchmark/results/smoke/latest.json`
- `benchmark/results/<example>/<profile>/latest.json` (paths from the example runner)
- `docs/bundled-benchmarks/<scenario>__<profile>__<variant>.json` — short slugs for the shipped UI (see `docs/bundled-benchmarks/README.md`)

## Benchmark phase evidence

| Phase | Verdict | Evidence |
| --- | --- | --- |
| Ontology generation | `PASS` | project_id=proj_57e3f0da8a33 |
| Graph build | `PASS` | graph_id=go_mirofish_20fe3c512dea4b74 |
| Simulation create | `PASS` | simulation_id=sim_553008e20bc1 |
| Simulation run | `PASS` | simulation_status=completed |
| Report generation | `PASS` | report_id=report_6238622d5fd4, report_status=completed |
| Report content | `PASS` | report_non_empty=True |

## What changed vs MiroFish is proven here

| Claim area | Current evidence from this run | Reading of that evidence |
| --- | --- | --- |
| Go runtime path | gateway health=`go-mirofish-gateway` and benchmark completed through the hybrid entrypoint | Requests were handled through go-mirofish's Go gateway instead of a Python-only surface. |
| Lightweight control plane | gateway RSS/HWM=`10192 kB / 10752 kB` | The Go control plane remained materially smaller than the Python backend during the captured run. |
| Same workflow shape | project_id=`proj_57e3f0da8a33`, graph_id=`go_mirofish_20fe3c512dea4b74`, simulation_id=`sim_553008e20bc1`, report_id=`report_6238622d5fd4` | The benchmark exercised the same major workflow artifacts expected from MiroFish while running under go-mirofish's hybrid stack. |
| Remaining migration gap | simulation_status=`completed`, report_status=`completed`, report_non_empty=`True` | The workflow still depends on Python-owned engine layers even when the Go control plane succeeds. |

## Host and process footprint

- Host: `Linux 6.6.87.2-microsoft-standard-WSL2`
- Logical CPUs: `4`
- Backend RSS / HWM: `81500 kB / 81628 kB`
- Gateway RSS / HWM: `10192 kB / 10752 kB`

## Stress details

- Requests: `80`
- Successes: `80`
- Failures: `0`
- Latency min: `0.4ms`
- Latency p50: `4.55ms`
- Latency p95: `24.59ms`
- Latency max: `29.0ms`

## Migration interpretation

- A green benchmark here does not mean the project is fully rewritten in Go.
- A green benchmark here does mean the Go-owned public control plane can preserve the end-user workflow under the current benchmark contract.
- The example suite additionally proves that the platform is usable as a local-first template library, not only as one benchmarked stack path.
- The remaining migration work should be judged against the parity matrix, not against screenshot lineage or marketing language.

See also:

- `docs/hybrid/go-parity-matrix.md` for the stable parity contract
- `docs/hybrid/go-migration-plan.md` for the staged rewrite plan
