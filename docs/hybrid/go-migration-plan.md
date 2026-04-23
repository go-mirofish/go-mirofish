# Go Migration Plan

This document defines the practical migration target for `go-mirofish`.

The goal is **not** an immediate all-at-once rewrite. The goal is:

- a materially lighter runtime
- the same public workflow
- the same API shapes and artifact layout
- the same benchmark phases and report semantics
- a migration path that keeps parity testable

## Migration target

### Recommended target

Reach **70-80% Go ownership first**.

That means:

- Go owns the public HTTP surface
- Go owns startup, routing, orchestration, and deployment
- Go owns benchmark tooling, health checks, and release/runtime operations
- Python remains only as a worker for the simulation core until replaced

This gets most of the lightweight benefit without rebuilding the hardest engine parts too early.

### What “100% same output” should mean

Do not define parity as byte-for-byte identical LLM prose.

Use **behavioral parity** instead:

- same routes and request/response contracts
- same status transitions
- same file/artifact names and locations
- same graph-build and simulation phases
- same report structure
- same benchmark pass/fail stage under the same upstream provider conditions

## Why not a full Go rewrite first

The current Python core still owns behavior tightly coupled to Python-first libraries and services:

- OASIS social simulation runtime
- CAMEL-based agent runtime
- Zep ontology + graph ingestion workflow
- report-agent tool loop

External constraints confirmed during migration work:

- **OASIS** is Python-first and built around Python agent abstractions:
  - <https://docs.oasis.camel-ai.org/>
  - <https://docs.oasis.camel-ai.org/key_modules/social_agent>
  - <https://docs.oasis.camel-ai.org/key_modules/platform>
- **Zep** graph ontology has concrete schema limits and API semantics:
  - <https://help.getzep.com/sdk-reference/graph/set-ontology>
- **Gemini** support in this repo depends on Google’s OpenAI-compatible endpoint shape rather than a native Gemini SDK path:
  - <https://ai.google.dev/gemini-api/docs/openai>

So a full rewrite means re-implementing the engine, not just translating syntax.

## Current ownership map

### Already Go-owned

- gateway / reverse proxy
- static frontend serving
- route alias compatibility
- health endpoint
- part of startup/runtime entrypoint support
- part of deployment packaging

### Still Python-owned

- ontology generation
- graph build orchestration
- Zep schema application
- persona/profile generation
- simulation config generation
- simulation lifecycle execution
- report generation
- report-agent tool use
- most benchmark-critical business logic

## Best split for a lightweight product

### Static

- docs
- landing
- showcase
- fixture-driven playground

### Go primary app

- public API surface
- upload handling
- task management
- orchestration
- runtime supervision
- artifact serving
- benchmark harness
- provider configuration validation
- report/status polling

### Python worker

- simulation prepare
- simulation run
- temporary report generation if still needed

### Optional advanced path

- BYOK / local model backends
- local OpenAI-compatible runtime support

## Phase plan

### Phase 0. Freeze parity and evidence

Before more rewrites:

- keep the benchmark fixture stable
- keep the contract matrix stable
- keep the benchmark report flow stable
- add more benchmark fixtures as a golden corpus later

Outputs for this phase:

- benchmark fixture set
- parity route matrix
- benchmark report
- migration parity matrix

### Phase 1. Go control plane

Make Go the primary app surface.

Move or own in Go:

- upload API
- project/task state API
- health/status APIs
- benchmark orchestration
- route normalization
- local runtime wiring
- static file/artifact serving

Python becomes an internal worker target, not the main app.

Success criteria:

- all public HTTP entrypoints terminate in Go first
- Go can dispatch work to Python internally
- frontend never needs to know whether the worker is Python or Go

### Phase 2. Go provider layer

Move all provider-facing LLM work into Go:

- OpenAI-compatible client
- retry/backoff
- JSON repair
- structured upstream error mapping
- provider-specific validation

Target modules to absorb:

- `backend/app/utils/llm_client.py`
- `backend/app/llm_provider.py`
- `backend/app/utils/retry.py`

Success criteria:

- ontology/config/profile/report calls use one Go-side provider client
- provider failures surface the same error contract

### Phase 3. Go ontology + config + profile generation

Move deterministic orchestration and schema-shaping logic first:

- ontology generation request builder
- ontology validation/normalization
- simulation config generation
- persona/profile generation

Target Python modules:

- `backend/app/services/ontology_generator.py`
- `backend/app/services/simulation_config_generator.py`
- `backend/app/services/oasis_profile_generator.py`

Success criteria:

- same ontology output shape
- same config JSON shape
- same profile artifact formats:
  - `twitter_profiles.csv`
  - `reddit_profiles.json`

### Phase 4. Go Zep integration

Move graph and retrieval integration into Go:

- graph creation
- ontology submission
- chunk ingestion
- node/edge reads
- retrieval used by config/profile/report flows

Target Python modules:

- `backend/app/services/graph_builder.py`
- `backend/app/services/zep_entity_reader.py`
- `backend/app/services/zep_tools.py`
- `backend/app/services/zep_graph_memory_updater.py`

Success criteria:

- same graph build lifecycle
- same task progress phases
- same graph statistics/report tools semantics

### Phase 5. Go report generation

Move report orchestration before the full simulation engine rewrite.

Target Python modules:

- `backend/app/services/report_agent.py`
- report endpoints currently under `backend/app/api/report.py`

Success criteria:

- same report stages
- same section outputs/artifact naming
- same chat/report-tool semantics where possible

### Phase 6. Simulation engine decision point

At this point choose between two paths:

#### Option A. Keep Python simulation worker

Recommended if the main goal is lightweight deployment.

Benefits:

- most product/runtime surface is already Go
- only simulation execution stays Python
- lowest risk

#### Option B. Full Go simulation rewrite

Do only if you are ready to rebuild:

- dual-platform action model
- agent scheduling
- action logging
- IPC control flow
- OASIS-like environment behavior
- report-interview consistency

Target scripts/modules:

- `backend/scripts/run_parallel_simulation.py`
- `backend/scripts/run_twitter_simulation.py`
- `backend/scripts/run_reddit_simulation.py`
- `backend/app/services/simulation_runner.py`
- `backend/app/services/simulation_manager.py`
- `backend/app/services/simulation_ipc.py`

Success criteria:

- benchmark reaches the same downstream stages
- same state transitions
- same action log semantics
- same interview/report integration contracts

## Recommended end state

### 70-80% Go state

This is the recommended product target.

Go owns:

- API
- orchestration
- provider layer
- graph build layer
- report layer
- startup/deploy/runtime
- benchmark tooling

Python owns only:

- OASIS simulation worker

This gives:

- lower idle memory
- faster startup
- smaller public runtime
- easier deploy story
- much lower rewrite risk

### 100% Go state

Only pursue after the 70-80% state is stable.

Do not attempt before parity coverage is strong enough to tell whether the new engine still behaves the same.

## What to rewrite first inside this repo

Recommended order:

1. `backend/app/utils/llm_client.py`
2. `backend/app/services/ontology_generator.py`
3. `backend/app/services/simulation_config_generator.py`
4. `backend/app/services/oasis_profile_generator.py`
5. `backend/app/services/graph_builder.py`
6. `backend/app/services/zep_entity_reader.py`
7. `backend/app/services/zep_tools.py`
8. `backend/app/services/report_agent.py`
9. `backend/app/services/simulation_manager.py`
10. `backend/app/services/simulation_runner.py`
11. `backend/scripts/run_parallel_simulation.py`
12. `backend/scripts/run_twitter_simulation.py`
13. `backend/scripts/run_reddit_simulation.py`

## Alternatives

### Alternative 1. Gateway-only fork

Keep Go only as a gateway and never migrate deeper.

Result:

- lowest rewrite cost
- smallest improvement
- Python remains the real application core

### Alternative 2. Go control plane + Python simulation worker

Recommended.

Result:

- strong weight reduction
- most user-visible runtime becomes Go
- highest value / risk balance

### Alternative 3. Dual engine

Keep:

- Go lightweight engine for fast/small scenarios
- Python parity engine for full-fidelity scenarios

Result:

- more flexibility
- more maintenance burden

### Alternative 4. Full Go rewrite

Result:

- biggest long-term payoff
- highest delivery risk
- longest parity campaign

## Decision recommendation

If the purpose of the project is to make the product meaningfully lighter:

- **do not** jump to full Go first
- **do** make Go the primary control plane
- **do** migrate ontology/config/profile/graph/report layers next
- **leave OASIS simulation for last**

That is the path most likely to preserve outputs while still delivering the lightweight product you actually want.
