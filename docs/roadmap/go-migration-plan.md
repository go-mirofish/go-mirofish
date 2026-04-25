# Go migration plan & current ownership

This file serves two purposes:

1. **Authoritative current state** — what the application is today (full **Go** ownership on the product path; **not** a Python/Flask hybrid).
2. **Historical record** — the original phased migration plan used while moving off the upstream Python control plane. Those phases are **complete** for shipping; this section is kept for context and auditability.

For a one-page statement of stack rules, see [`MIGRATION.md`](../../MIGRATION.md) at the repository root.

---

## Current state (authoritative)

**`go-mirofish` is a Go-native application end-to-end on the default product path.**

| Area | Owner | Notes |
| --- | --- | --- |
| Public HTTP / API | **Go** (`gateway/`) | All `/api/*` routes terminate in the gateway. |
| Ontology, graph, prepare, simulation, report | **Go** | Implemented under `gateway/internal/*` (see table below). |
| Simulation engine / worker | **Go** | Native runner in-process with the gateway; no separate Python worker. |
| UI | **Vue** + Vite | Dev: Vite on `:5173` proxying to the gateway. Release: static `dist/` in the release image. |
| Python / Flask | **Not used** | No `backend/.venv`, no Flask on the hot path, no hybrid control plane. |

**Developer workflow (canonical):** `make up` (gateway in Docker, typically `:3000`) + `npm run dev` (Vite, typically `:5173`). Optional: `make up-release` for an all-in-one image.

**Parity and proof:** Benchmarks, contract docs, and the parity matrix still matter for **regression** and **API contract** discipline. They are not evidence that Python remains in the loop—they describe **behavioral** equivalence expectations for the Go implementation.

---

## Behavioral parity (still applies)

Output is **not** required to be byte-for-byte identical LLM prose. **Behavioral parity** means, where documented:

- Same public route set (or documented aliases)
- Same task / simulation / report lifecycle semantics
- Same artifact names and expected locations
- Same benchmark **phase order** and pass/fail meaning for a given run profile
- Same health and status signals at the contract level

Use [`go-parity-matrix.md`](go-parity-matrix.md) and [`contract-matrix.md`](contract-matrix.md) with the understanding that the **Flask** column in historical contract tables describes the **legacy** upstream surface the gateway **replaced**, not a runtime that ships in this tree.

---

## Historical context: why the migration was incremental

The original fork goal was a **lighter runtime** and a **single Go control plane** while preserving a recognizable workflow and testable contracts. Upstream relied on Python-first services (OASIS/CAMEL-style patterns, Zep ingestion, report-agent loops) that could not be translated line-for-line in one step.

The practical approach was: **own the HTTP surface and orchestration in Go first**, then **move** ontology, graph, config, report, and simulation execution into Go behind stable contracts, using benchmarks to prove each lane.

That migration **completed** for the product path: the remaining sections record **what was planned and why**, not work left to do to “exit hybrid” on the default stack.

---

## Historical ownership map (pre–full-Go; superseded)

The following described the **intended** split **during** migration. It is **out of date** as a “current” map.

### Was planned to land in Go first

- Gateway, static serving, routing, health, benchmarks
- Public APIs and orchestration
- Provider (LLM) client consolidation
- Ontology, graph, Zep integration, config/profile generation, report orchestration, simulation lifecycle **as each phase completed**

### Was upstream / Python (MiroFish-era)

- Flask app, `backend/app/*` services, Python simulation scripts

### Current tree

- There is **no** `backend/app` product package in this repository for the hot path. Legacy Python module names in older docs map conceptually to **`gateway/internal/...`** implementations, not to files you run under `uv`/`python` for the product.

---

## Historical phase plan (archive)

The phases below were the **original** sequence. In the **current** repo, the **outcome** is a **fully Go-owned** public stack; individual “Phase N” checkboxes are not tracked as open migration tasks unless a future RFC explicitly reopens a lane (e.g. a second engine for research).

| Phase (historical) | Intent | Outcome in current product |
| --- | --- | --- |
| **0. Freeze parity & evidence** | Stable benchmark fixtures, contract matrix, reports | Ongoing process; artifacts under `docs/`, `benchmark/`, `docs/bundled-benchmarks/`. |
| **1. Go control plane** | All public HTTP, dispatch, static, health | **Done** — gateway only. |
| **2. Go provider layer** | LLM client, retries, error mapping in Go | **Done** — see `gateway/internal` provider code paths. |
| **3. Ontology + config + profiles** | Move generation/orchestration to Go | **Done** for product path. |
| **4. Zep / graph integration** | Graph build and tools in Go | **Done** for product path. |
| **5. Report generation** | Report pipeline in Go | **Done** for product path. |
| **6. Simulation engine** | choose Python worker vs full Go | **Resolved** — **Go-native** simulation is the supported runtime; see `MIGRATION.md`. |

### Historical “Option A vs B” (Python sim vs full Go)

The plan listed **Option A**: keep a Python simulation worker, vs **Option B**: full Go simulation. The **shipped** product on this repo follows the **Go-native** engine path; there is no supported Python worker on the default product loop.

If a future “dual engine” or experimental runner were introduced, it would be a **new** decision with its own proof gates—not a return to the old Flask control plane.

---

## Historical: legacy Python module list (do not use as file paths)

These names appeared in the original plan as **sources to absorb**. They refer to the **upstream** MiroFish / hybrid-era layout, **not** to runnable paths in the current `go-mirofish` tree:

- `backend/app/utils/llm_client.py`, `llm_provider.py`, `retry.py`
- `backend/app/services/ontology_generator.py`, `simulation_config_generator.py`, `oasis_profile_generator.py`
- `backend/app/services/graph_builder.py`, `zep_entity_reader.py`, `zep_tools.py`, `zep_graph_memory_updater.py`
- `backend/app/services/report_agent.py`, `simulation_manager.py`, `simulation_runner.py`, `simulation_ipc.py`
- `backend/scripts/run_parallel_simulation.py`, `run_twitter_simulation.py`, `run_reddit_simulation.py`

**Current work** lands in **Go** under the gateway module; search `gateway/internal/` and `gateway/cmd/` for the real implementation surfaces.

---

## Historical alternatives (decision log)

- **Gateway-only fork** — Go as reverse proxy only; **not** this repo’s direction.
- **Go control plane + Python sim worker** — was a **migration** compromise; **not** the shipped end state here.
- **Dual engine (Go + Python)** — optional future complexity; not the default.
- **Full Go** — this repo’s **default product path** for control plane + simulation + reporting on the hot path.

---

## Decision summary (as shipped)

- **Do** treat the gateway as the only application entrypoint for APIs and the supported simulation/report lifecycle.
- **Do** use benchmarks and parity docs to **prevent contract drift** in Go.
- **Do not** assume a Python process, Flask, or `backend/` venv for normal development (see `MIGRATION.md`).

For forward-looking work (hardening, proof gates, ARM/Pi, docs), see [`../roadmap.md`](../roadmap.md).
