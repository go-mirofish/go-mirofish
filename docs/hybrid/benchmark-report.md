# go-mirofish v0.1.0 benchmark report

- Captured at: `2026-04-23T19:54:45Z`
- Release: `v0.1.0`
- Backend health service: `go-mirofish-backend`
- Gateway health service: `go-mirofish-gateway`

## Host and process footprint

- Host: `Linux 6.6.87.2-microsoft-standard-WSL2`
- Logical CPUs: `4`
- Backend RSS / HWM: `87088 kB / 87088 kB`
- Gateway RSS / HWM: `10124 kB / 11392 kB`

## Full-flow benchmark

- Result: `PASS`
- Duration: `737.96s`
- Return code: `0`
- Project ID: `proj_3aa5acfe59fd`
- Graph ID: `go_mirofish_d421de1585144117`
- Simulation ID: `sim_9fa6c935d1d5`
- Report ID: `report_3176935f8403`
- Simulation status: `completed`
- Report status: `completed`
- Report non-empty: `False`

## Bounded stress pass

- Requests: `80`
- Successes: `80`
- Failures: `0`
- Latency min: `0.46ms`
- Latency p50: `5.58ms`
- Latency p95: `24.74ms`
- Latency max: `33.41ms`
