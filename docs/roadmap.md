# go-mirofish Roadmap

This roadmap is based on the current repository state as indexed on April 25, 2026.

It is informed by the approved sprint artifact at `.omx/plans/sprint-go-mirofish-a1-next-implementation.md`, but it is normalized to the application that exists in this tree today. The biggest consequence of that normalization is simple: the current application is already Go-native on the public product path, so the immediate roadmap is no longer "reduce Python first." The immediate roadmap is "harden, validate, and prove the Go stack."

## Current State Index

## Product shape

- The public application is a Go gateway plus a Vue frontend.
- The gateway entrypoint is [gateway/cmd/mirofish-gateway/main.go](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/cmd/mirofish-gateway/main.go:1).
- The frontend lives under [frontend/src](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/frontend/src).
- The docs UI already exposes a roadmap page and points its source link at this file through [frontend/src/docs/manifest.js](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/frontend/src/docs/manifest.js:79).

## Runtime and developer workflow

- The canonical development path is Docker for the gateway plus local Vite for the frontend, as described in [README.md](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/README.md:55) and [Makefile](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/Makefile:1).
- The repo exposes stable entrypoints for bootstrap, gateway build, frontend build, tests, and benchmarks through [package.json](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/package.json:5).
- The release and developer shell flows are centralized under [scripts/dev](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/scripts/dev) and [scripts/release](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/scripts/release).

## Core application areas

- HTTP surface:
  - `internal/http/app`
  - `internal/http/graph`
  - `internal/http/ontology`
  - `internal/http/prepare`
  - `internal/http/report`
  - `internal/http/simulation`
- Domain/services:
  - `internal/graph`
  - `internal/ontology`
  - `internal/provider`
  - `internal/report`
  - `internal/simulation`
  - `internal/worker`
  - `internal/telemetry`
- Persistence:
  - `internal/store/graph`
  - `internal/store/localfs`
  - `internal/store/report`
  - `internal/store/simulation`
- Example and benchmark tooling:
  - [gateway/cmd/go-mirofish-examples](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/cmd/go-mirofish-examples)
  - [gateway/cmd/benchmark](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/cmd/benchmark)
  - [gateway/cmd/mirofish-hybrid](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/cmd/mirofish-hybrid)

## Data and artifact layout

- Runtime data lives under [data/](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/data).
- Benchmarks and benchmark results live under [benchmark/](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/benchmark).
- Docs-facing benchmark snapshots live under [docs/bundled-benchmarks](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/docs/bundled-benchmarks).
- Report documentation and proof surfaces currently live under [docs/report](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/docs/report).

## Current strengths

- The public API is already Go-owned.
- The repo has a real benchmark and stress surface, not just screenshots.
- The frontend, benchmark docs, and examples are already integrated into one local-first workflow.
- ARM64 and Raspberry Pi are already treated as explicit claim levels instead of hand-wavy promises in [docs/report/raspberry-pi-validation.md](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/docs/report/raspberry-pi-validation.md:1).

## Current gaps

- There is still no `/ready` endpoint in the gateway; only `/health` is exposed today through [main.go](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/cmd/mirofish-gateway/main.go:77) and [internal/http/app/handler.go](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/internal/http/app/handler.go:31).
- Request logging is still basic and not structured enough for production triage in [internal/http/app/handler.go](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/internal/http/app/handler.go:80).
- Some documentation and validation scripts still carry stale hybrid-era assumptions. The clearest example is `scripts/hybrid/verify_contract_matrix.py`, which still points at removed Python API files and currently fails against the repo.
- Release criteria are implied across multiple scripts and docs, but not yet centralized into one enforceable roadmap or release-gate document.
- Raspberry Pi support is still `ARM64-ready` / `pending on-device validation`, not yet real-hardware verified.

## What Is Already Done

The old "reduced Python" goal is effectively complete in the current application tree.

Evidence:

- There is no active `go-mirofish/backend/app` product tree in this repo state.
- The public API surface, examples, benchmark tools, and release path are already Go-native.
- The README now describes the product path as Go-owned and states there is no Python process on the default product path.

That means the roadmap should stop treating "reduce Python" as the first major milestone. The first major milestone is now production hardening and runtime confidence.

## Roadmap Priorities

## Priority 1: Production-Harden the Go Control Plane

This is the immediate next milestone.

Goals:

- add `/ready` separate from `/health`
- add structured request logging, request IDs, and clearer dependency/failure classification
- harden store behavior for malformed files, missing artifacts, and partial writes
- tighten startup validation for required directories, assets, and gateway/runtime config
- make failure modes observable rather than silent

Primary repo surfaces:

- [gateway/cmd/mirofish-gateway/main.go](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/cmd/mirofish-gateway/main.go:1)
- [gateway/internal/http/app/handler.go](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/internal/http/app/handler.go:1)
- [gateway/internal/store/localfs/store.go](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/internal/store/localfs/store.go:1)
- [gateway/internal/store/report/store.go](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/internal/store/report/store.go:1)
- [gateway/internal/store/simulation](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/gateway/internal/store/simulation)

Exit condition:

- cold boot is deterministic
- readiness is explicit
- logs and store errors are diagnosable
- no silent partial-write success paths remain in the hardened surfaces

## Priority 2: Turn Proof Into a Release Gate

The repo already has proof tools. The next step is making them reliable, current, and enforceable.

Goals:

- make benchmark, stress, and live-stack commands the canonical release proof path
- repair stale validation scripts
- add explicit thresholds for stress and bounded soak
- make the release decision reproducible from commands, not memory

Primary repo surfaces:

- [scripts/hybrid/run_benchmark_smoke.py](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/scripts/hybrid/run_benchmark_smoke.py:1)
- [scripts/hybrid/run_stress_probe.py](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/scripts/hybrid/run_stress_probe.py:1)
- [scripts/hybrid/run_live_benchmark.sh](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/scripts/hybrid/run_live_benchmark.sh:1)
- [scripts/hybrid/verify_contract_matrix.py](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/scripts/hybrid/verify_contract_matrix.py:1)
- [docs/report/benchmark-report.md](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/docs/report/benchmark-report.md:1)

Exit condition:

- proof scripts match the current Go-native application
- stress thresholds are numeric and documented
- bounded soak has a defined pass/fail shape
- release criteria can be executed without tribal knowledge

## Priority 3: Align Documentation With the Actual Application

The repo has strong docs coverage, but some pages still reflect older migration phases and hybrid assumptions.

Goals:

- align docs with the actual Go-native public product path
- keep roadmap, benchmark docs, installation, and claim language consistent
- separate "current state" from "future direction" cleanly

Primary repo surfaces:

- [README.md](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/README.md:1)
- [docs/getting-started/installation.md](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/docs/getting-started/installation.md:1)
- [docs/report/go-migration-plan.md](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/docs/report/go-migration-plan.md:1)
- [docs/report/go-parity-matrix.md](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/docs/report/go-parity-matrix.md:1)
- [docs/report/contract-matrix.md](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/docs/report/contract-matrix.md:1)
- [docs/roadmap.md](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/docs/roadmap.md:1)

Exit condition:

- docs no longer rely on removed backend paths
- docs and scripts agree on the current runtime shape
- roadmap and release gates describe the same application

## Priority 4: ARM64 and Raspberry Pi Validation

This is a deployment-confidence milestone, not a marketing milestone.

Goals:

- keep ARM64 build proof healthy
- preserve a thin local-first profile suitable for Pi-class hardware
- capture real Pi proof before upgrading support claims

Primary repo surfaces:

- [docs/report/raspberry-pi-validation.md](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/docs/report/raspberry-pi-validation.md:1)
- [Makefile](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/Makefile:1)
- [scripts/dev/gateway.sh](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/scripts/dev/gateway.sh:1)
- [scripts/release/package_gateway_release.py](/mnt/c/users/justinedevs/downloads/mirofish/go-mirofish/scripts/release/package_gateway_release.py:1)

Exit condition:

- ARM64 build proof remains green
- Pi claim language stays disciplined
- real Raspberry Pi proof package exists before "Pi verified" wording appears

## Priority 5: Deeper Simulation and Runtime Evolution

This is the longer-horizon bucket.

It includes:

- richer simulation engine behavior
- deeper agent/runtime controls
- larger-scale orchestration changes
- broader telemetry, admin, or platform work beyond the immediate hardening path

This is not the next sprint. It comes after the Go control plane is hardened and the runtime proof path is stable.

## Next Sprint

The approved next sprint is still the best immediate execution unit, but it should be read through the current repo state:

- not "reduce Python from the product path"
- instead "finish A1-style control-plane hygiene, hardening, and proof on the Go-native application"

That sprint currently lives at:

- [.omx/plans/sprint-go-mirofish-a1-next-implementation.md](/mnt/c/users/justinedevs/downloads/mirofish/.omx/plans/sprint-go-mirofish-a1-next-implementation.md:1)

Operational reading of that sprint in the current repo:

1. clean up remaining control-plane boundary and state-write contradictions in Go surfaces
2. add readiness and structured observability
3. harden stores and startup checks
4. repair and enforce live-stack proof scripts
5. defer broader runtime reinvention until proof gates are stable

## Not On The Immediate Roadmap

- rewriting the product around a new architecture premise
- reintroducing a Python-first control plane
- broad infrastructure expansion without proof-driven need
- claiming verified Raspberry Pi support before actual device evidence exists

## Success Looks Like This

The short version of the roadmap is:

- the current Go-native stack becomes easier to boot, inspect, and trust
- release gates move from implied to explicit
- docs match the code that is actually running
- ARM64/Pi claims stay disciplined
- future runtime work starts from a hardened base instead of a moving target
