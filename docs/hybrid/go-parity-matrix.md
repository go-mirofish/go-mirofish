# Go Parity Matrix

This matrix defines what must remain stable while `go-mirofish` migrates more of the Python core into Go.

The rule is simple:

- rewrite internals freely
- do not break the parity surface without updating this file and the benchmark contract

## Parity definition

### Required

- same public route set or documented alias behavior
- same task lifecycle states
- same artifact filenames and locations
- same benchmark phase order
- same report/status semantics
- same simulation status semantics
- same health/status signals

### Not required

- byte-for-byte identical LLM prose
- identical internal class/module structure
- identical logs unless user-facing or benchmark-relevant

## How to read benchmark evidence against this matrix

Use the benchmark report as the execution proof for this matrix.

Interpret the captured run in this order:

1. stack proof
2. route and lifecycle proof
3. artifact proof
4. artifact quality proof

That ordering matters because a run can prove parity without proving every output is high quality.

Example:

- a run with `project_id`, `graph_id`, `simulation_id`, and `report_id` proves the major workflow stages executed
- a run with `report_status=completed` proves report lifecycle parity
- a run with `report_non_empty=false` means the workflow completed but the artifact quality still needs follow-up

Do not collapse those into one binary claim.

## Module ownership and migration priority

| Area | Current owner | Target owner | Priority | Parity risk |
| --- | --- | --- | --- | --- |
| Gateway / route aliases | Go | Go | done | low |
| Public API surface | Go | Go | done | low |
| LLM provider client | Python | Go | high | medium |
| Ontology generation | Python | Go | high | high |
| Simulation config generation | Python | Go | high | medium |
| Persona/profile generation | Python | Go | high | medium |
| Zep graph build | Python | Go | high | high |
| Zep retrieval/report tools | Python | Go | medium | high |
| Report generation | Python | Go | medium | high |
| Simulation orchestration | Python | Go | medium | medium |
| OASIS simulation worker | Python | Python or Go later | low for now | very high |

## Public HTTP parity

Must remain stable:

| Capability | Contract |
| --- | --- |
| Ontology generation | `POST /api/graph/ontology/generate` |
| Graph build | `POST /api/graph/build` |
| Graph task polling | `GET /api/graph/task/:task_id` |
| Simulation create | `POST /api/simulation/create` |
| Simulation prepare | `POST /api/simulation/prepare` |
| Simulation prepare status | `POST /api/simulation/prepare/status` |
| Simulation start/run | current alias contract from `docs/hybrid/contract-matrix.md` |
| Simulation status | current alias contract from `docs/hybrid/contract-matrix.md` |
| Report generate | `POST /api/report/generate` |
| Report generate status | current alias contract from `docs/hybrid/contract-matrix.md` |
| Report fetch | `GET /api/report/:report_id` |

## Artifact parity

Must remain stable:

| Artifact | Required shape |
| --- | --- |
| `simulation_config.json` | same top-level keys and route expectations |
| `twitter_profiles.csv` | same downstream OASIS-compatible columns |
| `reddit_profiles.json` | same downstream OASIS-compatible structure |
| report markdown | same section-oriented report output contract |
| benchmark result JSON | same top-level benchmark keys used by report tooling |

## Lifecycle parity

### Graph build

Expected lifecycle:

1. ontology generated
2. graph build task created
3. graph task moves through processing
4. graph id becomes available
5. node/edge statistics become queryable

### Simulation prepare

Expected lifecycle:

1. simulation created
2. prepare task created
3. entities loaded
4. profiles generated
5. config generated
6. status becomes `ready`

### Simulation run

Expected lifecycle:

1. run request accepted
2. runner status becomes `starting` or `running`
3. round/status polling works
4. simulation completes, stops, or fails with a durable error

### Report generation

Expected lifecycle:

1. report task created
2. progress is queryable
3. final report id is durable
4. report fetch and section fetch work

## Benchmark gates

Every migration lane should be judged against these gates.

### Gate 1. Smoke

- backend health passes
- gateway health passes
- route aliases still behave

### Gate 2. Full benchmark

- ontology stage passes
- graph build stage passes
- simulation prepare passes
- simulation run reaches the expected state
- report generation reaches the expected state

### Gate 3. Stress

- bounded health/status probe still passes
- no regression in obvious latency or failure rate

## Failure classification

When a benchmark fails, classify it before blaming the migration.

### External failure

- provider quota
- provider billing
- provider availability
- Zep service-side rejection unrelated to code drift

### Contract failure

- route mismatch
- wrong status surface
- wrong artifact path/name
- missing field required by the next stage

### Engine failure

- simulation runner crash
- graph build logic bug
- report tool loop bug

## Acceptance criteria for calling the project “mostly migrated to Go”

You can claim “mostly migrated to Go” only when:

- Go owns all public HTTP routes
- Go owns provider integration
- Go owns ontology/config/profile/report layers
- Python is only a worker for simulation execution
- the benchmark reaches the same major phases under the same external conditions

That is the 70-80% Go state.

## Current measured state from the latest benchmark report

The latest captured first-party benchmark currently proves:

- Go gateway health is stable during the measured run
- Python backend health is stable during the measured run
- ontology, graph build, simulation, and report generation can complete under the hybrid stack
- the latest run completes with a non-empty report artifact
- the example suite runner completes all five local-first templates and writes deterministic artifacts for each

So the current state is:

- parity proof: strong for the measured workflow stages
- migration proof: strong for the Go public control plane path
- content-quality proof: strong for the currently benchmarked stack and example suite
