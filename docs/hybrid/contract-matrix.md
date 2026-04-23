# Hybrid Contract Matrix

This document freezes the first gateway compatibility boundary described by the approved hybrid migration plan.

## Scope

- Locked benchmark flow from `.omx/specs/deep-interview-mirofish-hybrid-migration.md`
- Current Flask backend routes
- Current frontend callers
- Gateway aliases required where those surfaces differ

## Matrix

| Capability | Locked benchmark contract | Current backend route | Current frontend caller | Gateway policy |
| --- | --- | --- | --- | --- |
| Graph ontology upload | `POST /api/graph/ontology/generate` | `POST /api/graph/ontology/generate` | `POST /api/graph/ontology/generate` | Direct pass-through |
| Graph task polling | `GET /api/graph/task/:task_id` | `GET /api/graph/task/<task_id>` | `GET /api/graph/task/:task_id` | Direct pass-through |
| Simulation prepare | `POST /api/simulation/prepare` | `POST /api/simulation/prepare` | `POST /api/simulation/prepare` | Direct pass-through |
| Simulation prepare status | `POST /api/simulation/prepare/status` | `POST /api/simulation/prepare/status` | `POST /api/simulation/prepare/status` | Direct pass-through |
| Simulation run | `POST /api/simulation/run` | `POST /api/simulation/start` | `POST /api/simulation/start` | Alias `run -> start` |
| Simulation status | `GET /api/simulation/:id/status` | `GET /api/simulation/<id>/run-status` | `GET /api/simulation/<id>/run-status` | Alias `status -> run-status` |
| Simulation detail status | n/a | `GET /api/simulation/<id>/run-status/detail` | `GET /api/simulation/<id>/run-status/detail` | Direct pass-through |
| Report generate | `POST /api/report/generate` | `POST /api/report/generate` | `POST /api/report/generate` | Direct pass-through |
| Report generate status | `GET /api/report/generate/status` | `POST /api/report/generate/status` | `GET /api/report/generate/status?report_id=...` | Method bridge plus `report_id -> /api/report/:report_id/progress` |
| Report download | `GET /api/report/:report_id/download` | `GET /api/report/<report_id>/download` | `GET /api/report/:report_id/download` | Direct pass-through |
| Realtime profiles | Benchmark parity required | `GET /api/simulation/<id>/profiles/realtime` | `GET /api/simulation/<id>/profiles/realtime` | Direct pass-through |
| Config realtime | Benchmark parity required | `GET /api/simulation/<id>/config/realtime` | `GET /api/simulation/<id>/config/realtime` | Direct pass-through |

## Alias Rules

### Simulation

- `POST /api/simulation/run`
  - Forward to `POST /api/simulation/start`
  - Preserve request body and headers

- `GET /api/simulation/:id/status`
  - Forward to `GET /api/simulation/:id/run-status`
  - Preserve query string

### Report

- `GET /api/report/generate/status?report_id=<id>`
  - Forward to `GET /api/report/<report_id>/progress`
  - Use this for the current frontend polling shape

- `GET /api/report/generate/status?task_id=<id>&simulation_id=<id>`
  - Forward as `POST /api/report/generate/status`
  - Marshal query values into a JSON body because the backend status endpoint is POST-only

## Notes

- The current frontend and backend already diverge on report generation polling semantics.
- The gateway must remain the compatibility boundary instead of pushing that mismatch deeper into the Python core.
- Additional aliases should only be added after documenting the route and verification evidence here first.
