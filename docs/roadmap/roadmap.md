# go-mirofish Roadmap

This document describes **where the product is today** (full **Go** ownership on the public path—**not** a Python/Flask hybrid) and **what to do next** (hardening, proof, and docs), grounded in the actual repository.

**Related:** stack rules in [`MIGRATION.md`](../MIGRATION.md) at the repo root; migration history and archival phases in [`go-migration-plan.md`](go-migration-plan.md). **Long-horizon R&D** (non-binding): [`future-consideration.md`](future-consideration.md).

---

## Product shape (current)

- **Control plane and APIs:** Go gateway — [`gateway/cmd/mirofish-gateway`](../gateway/cmd/mirofish-gateway), routes under `gateway/internal/http/*`.
- **Simulation:** Go-native worker; no Python process on the default product loop (see `MIGRATION.md`).
- **UI:** Vue + Vite — [`frontend/`](../frontend); dev server proxies `/api` to the gateway.
- **In-app docs:** Roadmap page is wired in [`frontend/src/docs/manifest.js`](../frontend/src/docs/manifest.js); this file is the `sourcePath` for that page.
- **Observability / tooling (naming):** The binary `mirofish-hybrid` under [`gateway/cmd/mirofish-hybrid`](../gateway/cmd/mirofish-hybrid) is a **Go** helper for live benchmark, API smoke, stress probe, and bundled merge — the name is legacy; it does **not** mean a hybrid Python+Go runtime.

---

## Runtime and developer workflow

- **Canonical dev:** `make up` (Docker gateway, typically `:3000`) + `npm run dev` (Vite, typically `:5173`) — see [`README.md`](../README.md) and [`Makefile`](../Makefile).
- **Optional all-in-one:** `make up-release` for static UI in-container (see `README` / installation docs).
- **Scripts:** [`scripts/dev/`](../scripts/dev) and [`scripts/release/`](../scripts/release) for release and local helpers.

---

## Core application areas (Go)

| Layer | Location (under `gateway/`) |
| --- | --- |
| HTTP | `internal/http/app`, `graph`, `ontology`, `prepare`, `report`, `simulation` |
| Domain / services | `internal/graph`, `internal/ontology`, `internal/provider`, `internal/report`, `internal/simulation`, `internal/worker`, `internal/telemetry` |
| Storage | `internal/store/graph`, `localfs`, `report`, `simulation` |
| Examples & bench helpers | `cmd/go-mirofish-examples`, `cmd/benchmark`, `cmd/mirofish-hybrid`, `cmd/benchmark-report` |

---

## Data and artifacts

- Runtime data: [`data/`](../data) (in Docker, typically mounted as `mirofish-data`).
- Benchmark output: [`benchmark/`](../benchmark).
- Bundled in-app proof JSON: [`docs/bundled-benchmarks/`](../docs/bundled-benchmarks).
- Report and migration docs: [`docs/report/`](../docs/report).

---

## Current strengths

- The **entire** public product path is **Go-owned**; hybrid-era “reduce Python” milestones are **done** for shipping.
- Benchmarks, stress, and live-stack proof are first-class (see `README` — `make benchmark-live`, `make benchmark`, `make api-wiring-report`).
- ARM64 and Raspberry Pi claims are **disciplined** in [`docs/report/raspberry-pi-validation.md`](report/raspberry-pi-validation.md) (ready vs verified language).

---

## Gaps and follow-ups (not “finish hybrid”)

- **`/ready` vs `/health`:** Some hardening work may add an explicit readiness probe; today rely on what [`gateway` exposes](../gateway/cmd/mirofish-gateway/main.go) and [`internal/http/app`](../gateway/internal/http/app/handler.go) document.
- **Observability:** Request logging and production triage may need structure (request IDs, dependency classification) beyond basic handlers.
- **Docs / scripts drift:** A few old references may still mention Python hot paths or removed `scripts/hybrid/*` helpers; canonical automation is **Makefile** + **Go** under `gateway/cmd/mirofish-hybrid` (replaces old shell smoke scripts — see `README`).
- **Release criteria:** Centralize numeric gates where possible (see `benchmark/config/release-criteria.json` and benchmark docs).
- **Raspberry Pi:** Still “ARM64-ready, pending on-device proof” until a captured device run exists.

---

## Roadmap priorities

### 1) Production-harden the Go control plane (near term)

- Explicit readiness and health semantics if/when `ready` is added
- Structured logging, request correlation, clearer failure classes
- Store safety: partial writes, missing artifacts, corrupt files
- Startup validation for required dirs and config

**Touchpoints:** `gateway/cmd/mirofish-gateway`, `internal/http/app`, `internal/store/*`

### 2) Make proof a reproducible release gate

- Keep **`make benchmark-live`**, **`make benchmark`**, and **`make api-wiring-report`** as the primary proof entrypoints (they invoke Go under `gateway/`).
- Align [`docs/report/benchmark-report.md`](report/benchmark-report.md) and bundled JSON with whatever the Makefile targets produce now.
- Document stress/soak thresholds (see `benchmark/config/release-criteria.json`).

### 3) Align documentation with the Go-only stack

- **Done for this file and** [`go-migration-plan.md`](go-migration-plan.md) **:** state full Go ownership; reframe old hybrid language as **historical**.
- Continue sweeping: [`README.md`](../README.md), [`go-parity-matrix.md`](report/go-parity-matrix.md), [`contract-matrix.md`](report/contract-matrix.md) (legacy Flask column = replaced contract, not a second runtime).

### 4) ARM64 / Raspberry Pi validation

- Keep build proof green; capture real hardware evidence before “verified on Pi” language.

### 5) Deeper runtime evolution (later)

- Richer simulation behavior, admin/telemetry, larger orchestration — **after** the control plane and proof path are stable.

### 6) Future consideration (long-horizon R&D, non-binding)

- Research and architecture ideas that are **not** committed work until written up as an RFC or issue. Covered in the companion doc: [`future-consideration.md`](future-consideration.md) (also available in-app under **Contribute** → *Future consideration* on `/docs`).
- Themes include: echo chambers and priors, entropy in interactions, dialectical / friction-first reporting, judgment–action gaps and stakes, audit / oracle layers and knowledge decay, symbolic math in Go, memory compression, Governor-style orchestration, and edge / Raspberry Pi deployment notes.

---

## What is explicitly not the roadmap

- Reintroducing a **Python-first** or **Flask** control plane on the hot path.
- Marketing **hybrid** as the current architecture (the product is **Go-native**; see `MIGRATION.md`).
- Claiming full Raspberry Pi verification without device-captured runs.

---

## Success criteria (short)

- The Go stack is **easy to boot, observe, and trust**.
- Release and proof commands are **documented, reproducible, and current**.
- Docs describe **the code that actually runs** — full **Go** ownership, hybrid migration **historical** where mentioned.
